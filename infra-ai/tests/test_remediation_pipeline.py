"""
LUCID-CI — Comprehensive Remediation Pipeline Tests against Synthetic CWE Injections.
Tests prompt assembly, schema compliance, and patch characteristics across 10 fixtures.
Fulfills Gate 2 Exit Criteria for the /infra-ai domain.
"""

import json
from pathlib import Path
import pytest

from models.schemas import RemediationRequest, RemediationResponse
from prompts.remediation_prompt import build_remediation_prompt
from service.config import settings
from service.llm_client import BaseLLMClient, LLMResponse, GroqClient, GeminiClient
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


# Optional Live Integration Test (only executes if user adds a live API key to .env)
@pytest.mark.skipif(
    not settings.GROQ_API_KEY and not settings.GEMINI_API_KEY,
    reason="Live LLM API keys not configured in .env; skipping live inference test",
)
@pytest.mark.asyncio
async def test_live_llm_remediation_sample():
    """Executes a single live inference query against Groq or Gemini if API keys are set."""
    pipeline = RemediationPipeline()
    sample_req = synthetic_requests[0]  # JS SQLi

    response = await pipeline.remediate(sample_req)

    assert response.scan_id == sample_req.scan_id
    assert response.confidence > 0.5
    assert response.tokens_used > 50
    assert "deterministic-rule-fallback" not in response.model_name
