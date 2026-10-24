"""
LUCID-CI — Comprehensive Remediation Pipeline Tests against Synthetic CWE Injections.
Tests prompt assembly, schema compliance, and patch characteristics across 10 fixtures.
Fulfills Gate 2 Exit Criteria for the /infra-ai domain.
"""

import json
from pathlib import Path
import pytest

from unittest.mock import AsyncMock, MagicMock
from models.schemas import RemediationRequest, RemediationResponse
from prompts.remediation_prompt import build_remediation_prompt
from service.config import settings
from service.llm_client import (
    BaseLLMClient,
    LLMResponse,
    GroqClient,
    GeminiClient,
    VertexAIClient,
    MultiProviderLLMClient,
    LLMInferenceError,
    _resolve_gcp_project,
)
from service.parser import parse_and_validate_remediation
from service.remediation import RemediationPipeline

FIXTURES_PATH = Path(__file__).parent / "fixtures" / "synthetic_vulns.json"


def load_synthetic_fixtures():
    with open(FIXTURES_PATH, "r", encoding="utf-8") as f:
        data = json.load(f)
    return [RemediationRequest.model_validate(item) for item in data]


synthetic_requests = load_synthetic_fixtures()


class MockSmartRemediationClient(BaseLLMClient):
    """
    Simulates high-quality, CWE-aware LLM completions matching target security patterns.
    """
    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        return await self.generate_with_failover(system_prompt, user_prompt)

    async def generate_with_failover(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        # Generate targeted CWE patches based on user prompt metadata
        if "CWE-89" in user_prompt:
            content = json.dumps({
                "suggested_patch": "const query = 'SELECT * FROM users WHERE email = $1';\nconst user = await db.query(query, [email]);",
                "explanation": "Untrusted user input was interpolated directly into the SQL string.",
                "security_rationale": "Parameterized queries separate untrusted parameters from the SQL command tree.",
                "confidence": 0.96,
            })
        elif "CWE-78" in user_prompt:
            content = json.dumps({
                "suggested_patch": "execFile('ping', ['-c', '1', host], (err, stdout) => { ... });",
                "explanation": "Direct shell execution allows command chaining with shell metacharacters.",
                "security_rationale": "execFile avoids passing strings to the system shell interpreter.",
                "confidence": 0.94,
            })
        elif "CWE-798" in user_prompt:
            content = json.dumps({
                "suggested_patch": "const STRIPE_SECRET_KEY = process.env.STRIPE_SECRET_KEY;",
                "explanation": "Secret access key was hardcoded into source code.",
                "security_rationale": "Environment variables decouple sensitive credentials from revision control.",
                "confidence": 0.99,
            })
        elif "CWE-22" in user_prompt:
            content = json.dumps({
                "suggested_patch": "const safeFile = path.basename(req.query.file);\nconst data = fs.readFileSync(path.join(UPLOADS_DIR, safeFile));",
                "explanation": "User-controlled path allows directory traversal via '../' sequences.",
                "security_rationale": "path.basename strips directory traversal sequences.",
                "confidence": 0.91,
            })
        else:
            content = json.dumps({
                "suggested_patch": "// Replaced dynamic evaluation with safe parser",
                "explanation": "Dynamic code evaluation permits arbitrary code execution.",
                "security_rationale": "Avoid dynamic code execution APIs.",
                "confidence": 0.90,
            })

        return LLMResponse(
            content=content,
            model_name="mock-smart-llm",
            tokens_used=180,
            inference_latency_ms=250,
            provider="mock",
        )


@pytest.mark.parametrize("req", synthetic_requests, ids=lambda r: r.vulnerability_id)
def test_prompt_assembly_for_all_fixtures(req: RemediationRequest):
    """Verify that every synthetic fixture compiles into a valid, nonce-fenced PromptBundle."""
    bundle = build_remediation_prompt(req)

    assert len(bundle.nonce) == 32
    assert bundle.nonce in bundle.system_prompt
    assert f'<untrusted_user_code nonce="{bundle.nonce}">' in bundle.user_prompt
    assert req.cwe in bundle.user_prompt
    assert req.rule_id in bundle.user_prompt
    assert req.source_info in bundle.user_prompt


@pytest.mark.parametrize("req", synthetic_requests, ids=lambda r: r.vulnerability_id)
@pytest.mark.asyncio
async def test_remediation_pipeline_e2e_all_fixtures(req: RemediationRequest):
    """
    End-to-End Pipeline Verification across all 10 synthetic CWE fixtures:
    Prompt Assembly -> LLM Inference -> Pydantic Validation -> RemediationResponse.
    """
    client = MockSmartRemediationClient()
    pipeline = RemediationPipeline(llm_client=client)

    response = await pipeline.remediate(req)

    # 1. Contract Structure Assertions (CTR-005)
    assert isinstance(response, RemediationResponse)
    assert response.scan_id == req.scan_id
    assert response.vulnerability_id == req.vulnerability_id
    assert 0.8 <= response.confidence <= 1.0
    assert response.model_name == "mock-smart-llm"
    assert response.tokens_used > 0
    assert len(response.suggested_patch) > 0
    assert len(response.explanation) > 0
    assert len(response.security_rationale) > 0

    # 2. CWE-Specific Security Fix Assertions
    if req.cwe == "CWE-89":
        assert "$1" in response.suggested_patch or "?" in response.suggested_patch
    elif req.cwe == "CWE-798":
        assert "process.env" in response.suggested_patch or "os.environ" in response.suggested_patch


# =========================================================================
# Remediation Pipeline Tests for Vertex AI, 1-Shot Retry, and BR-002 Fallback
# =========================================================================

def test_parse_and_validate_remediation_vertex():
    """Verify parse_and_validate_remediation correctly parses structured JSON produced by Vertex AI (Gemini 3.5 Flash-Lite)."""
    sample_req = synthetic_requests[0]
    vertex_raw_content = """```json
{
  "suggested_patch": "const query = 'SELECT * FROM users WHERE email = $1';\\nconst user = await db.query(query, [email]);",
  "explanation": "Untrusted user input was interpolated directly into the SQL string.",
  "security_rationale": "Parameterized queries separate untrusted parameters from the SQL command tree.",
  "confidence": 0.97
}
```"""
    validated, err = parse_and_validate_remediation(
        raw_content=vertex_raw_content,
        request=sample_req,
        model_name="gemini-3.5-flash-lite",
        tokens_used=185,
        latency_ms=320,
    )

    assert err is None
    assert validated is not None
    assert validated.scan_id == sample_req.scan_id
    assert validated.vulnerability_id == sample_req.vulnerability_id
    assert validated.model_name == "gemini-3.5-flash-lite"
    assert validated.tokens_used == 185
    assert validated.confidence == 0.97
    assert "$1" in validated.suggested_patch


@pytest.mark.asyncio
async def test_remediation_pipeline_retry_loop():
    """Verify 1-shot corrective retry loop: Attempt 1 returns malformed JSON, Attempt 2 corrects it and succeeds."""
    sample_req = synthetic_requests[0]

    mock_llm = MagicMock(spec=BaseLLMClient)
    # Attempt 1: malformed / non-JSON output
    malformed_resp = LLMResponse(
        content="I think you should fix the SQL injection by parameterizing it.",
        model_name="gemini-3.5-flash-lite",
        tokens_used=50,
        inference_latency_ms=200,
        provider="vertexai",
    )
    # Attempt 2: valid JSON after corrective prompt
    valid_resp = LLMResponse(
        content=json.dumps({
            "suggested_patch": "const user = await db.query('SELECT * FROM users WHERE email = $1', [email]);",
            "explanation": "Parameterized query applied.",
            "security_rationale": "Separates data from execution.",
            "confidence": 0.95,
        }),
        model_name="gemini-3.5-flash-lite",
        tokens_used=120,
        inference_latency_ms=250,
        provider="vertexai",
    )

    mock_llm.generate_with_failover = AsyncMock(side_effect=[malformed_resp, valid_resp])
    pipeline = RemediationPipeline(llm_client=mock_llm)

    response = await pipeline.remediate(sample_req)

    assert mock_llm.generate_with_failover.call_count == 2
    # Verify retry prompt included error context
    second_call_prompt = mock_llm.generate_with_failover.call_args_list[1].kwargs["user_prompt"]
    assert "CRITICAL ERROR IN PREVIOUS ATTEMPT" in second_call_prompt
    # Assert successful validation of retry response
    assert response.confidence == 0.95
    assert "$1" in response.suggested_patch
    assert response.tokens_used == 170  # 50 + 120 combined


@pytest.mark.asyncio
async def test_all_providers_exhausted_preserves_br002():
    """Verify that when all AI providers fail (Groq + Vertex AI), the pipeline yields a deterministic fallback patch (BR-002 fail-open)."""
    sample_req = synthetic_requests[0]

    mock_llm = MagicMock(spec=BaseLLMClient)
    mock_llm.generate_with_failover = AsyncMock(
        side_effect=LLMInferenceError("All AI inference providers failed. Root causes: Primary (Groq): 429; Fallback (VertexAI): 503")
    )
    pipeline = RemediationPipeline(llm_client=mock_llm)

    response = await pipeline.remediate(sample_req)

    assert response.confidence == 0.0
    assert response.model_name == "deterministic-rule-fallback"
    assert "AI Provider Outage" in response.explanation
    assert "// [LUCID-CI MANUAL REVIEW REQUIRED]" in response.suggested_patch
    assert sample_req.cwe in response.suggested_patch


@pytest.mark.asyncio
async def test_remediation_pipeline_with_vertex_ai_client():
    """Verify RemediationPipeline integrates end-to-end with VertexAIClient mock."""
    sample_req = synthetic_requests[0]

    mock_response = MagicMock()
    mock_response.text = json.dumps({
        "suggested_patch": "const user = await db.query('SELECT * FROM users WHERE email = $1', [email]);",
        "explanation": "SQL injection patched via parameterization.",
        "security_rationale": "Parameters are escaped safely.",
        "confidence": 0.98,
    })
    mock_response.usage_metadata.total_token_count = 160

    mock_client = MagicMock()
    mock_client.aio.models.generate_content = AsyncMock(return_value=mock_response)

    vertex_client = VertexAIClient(
        project="lucid-ci-test",
        location="global",
        model="gemini-3.5-flash-lite",
        client=mock_client,
    )
    pipeline = RemediationPipeline(llm_client=vertex_client)

    response = await pipeline.remediate(sample_req)

    assert response.scan_id == sample_req.scan_id
    assert response.confidence == 0.98
    assert response.model_name == "gemini-3.5-flash-lite"
    assert response.tokens_used == 160
    assert "$1" in response.suggested_patch


# Optional Live Integration Test (executes if Groq, Gemini API key, or GCP ADC is present)
@pytest.mark.skipif(
    not settings.GROQ_API_KEY and not settings.GEMINI_API_KEY and not _resolve_gcp_project(),
    reason="Live LLM credentials (Groq, Gemini API key, or GCP ADC/project) not configured; skipping live inference test",
)
@pytest.mark.asyncio
async def test_live_llm_remediation_sample():
    """Executes a single live inference query against Groq, Gemini, or Vertex AI if credentials are set."""
    pipeline = RemediationPipeline()
    sample_req = synthetic_requests[0]  # JS SQLi

    response = await pipeline.remediate(sample_req)

    assert response.scan_id == sample_req.scan_id
    assert response.confidence > 0.5
    assert response.tokens_used > 50
    assert "deterministic-rule-fallback" not in response.model_name
