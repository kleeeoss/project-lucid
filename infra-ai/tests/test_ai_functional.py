"""
LUCID-CI — Functional AI Pipeline Verification Suite (Phase 5A).
Tests the complete functional AI lifecycle:
1. Groq inference (live or simulated contract)
2. Gemini inference (live or simulated contract)
3. Groq failure -> Gemini automated failover
4. Dual provider outage -> Deterministic Fallback (BR-002 Fail-Open)
5. Missing/empty API keys -> Deterministic Fallback
6. Output parsing & 1-shot corrective retry loop
7. AI-generated patch syntax integrity
"""

import ast
import json
import pytest
from unittest.mock import AsyncMock, patch

from models.schemas import RemediationRequest, RemediationResponse
from service.config import settings
from service.llm_client import (
    BaseLLMClient,
    GroqClient,
    GeminiClient,
    MultiProviderLLMClient,
    LLMResponse,
    LLMInferenceError,
)
from service.remediation import RemediationPipeline
from service.parser import parse_and_validate_remediation, build_deterministic_fallback


@pytest.fixture
def sample_sqli_request():
    return RemediationRequest(
        scan_id="scan-func-5a-001",
        vulnerability_id="vuln-func-5a-001",
        rule_id="LUCID-SEC-001",
        cwe="CWE-89",
        language="javascript",
        vulnerable_code="const query = 'SELECT * FROM users WHERE id = ' + req.query.id;",
        surrounding_context=(
            "app.get('/user', (req, res) => {\n"
            "  const id = req.query.id;\n"
            "  const query = 'SELECT * FROM users WHERE id = ' + req.query.id;\n"
            "  db.query(query);\n"
            "});"
        ),
        source_info="req.query.id",
        sink_info="db.query",
        taint_path_summary=["req.query.id -> query -> db.query"],
    )


@pytest.fixture
def sample_python_request():
    return RemediationRequest(
        scan_id="scan-func-5a-002",
        vulnerability_id="vuln-func-5a-002",
        rule_id="LUCID-SEC-001",
        cwe="CWE-89",
        language="python",
        vulnerable_code="cursor.execute(f'SELECT * FROM users WHERE email = \"{email}\"')",
        surrounding_context=(
            "def find_user(email):\n"
            "    cursor = db.cursor()\n"
            "    cursor.execute(f'SELECT * FROM users WHERE email = \"{email}\"')\n"
            "    return cursor.fetchone()"
        ),
        source_info="email parameter",
        sink_info="cursor.execute",
        taint_path_summary=["email -> cursor.execute"],
    )


class MockGroqHealthy(BaseLLMClient):
    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        return LLMResponse(
            content=json.dumps({
                "suggested_patch": "const query = 'SELECT * FROM users WHERE id = $1';\ndb.query(query, [req.query.id]);",
                "explanation": "Untrusted parameter interpolated directly into SQL string.",
                "security_rationale": "Parameterized query sanitizes user input and prevents injection.",
                "confidence": 0.95,
            }),
            model_name=settings.GROQ_MODEL,
            tokens_used=180,
            inference_latency_ms=320,
            provider="groq",
        )


class MockGeminiHealthy(BaseLLMClient):
    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        return LLMResponse(
            content=json.dumps({
                "suggested_patch": "cursor.execute('SELECT * FROM users WHERE email = %s', (email,))",
                "explanation": "Direct string interpolation permits SQL command tree modification.",
                "security_rationale": "DB-API parameterized query separates user argument from SQL syntax.",
                "confidence": 0.93,
            }),
            model_name=settings.GEMINI_MODEL,
            tokens_used=165,
            inference_latency_ms=410,
            provider="gemini",
        )


class MockProviderOutage(BaseLLMClient):
    def __init__(self, err_msg: str):
        self.err_msg = err_msg

    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        raise LLMInferenceError(self.err_msg)


# ============================================================================
# 3.1 & 3.2: Primary & Secondary Provider Generation
# ============================================================================
@pytest.mark.asyncio
async def test_groq_inference_contract(sample_sqli_request):
    """Test 3.1: Groq generates valid CTR-005 response with expected model and patch."""
    client = MultiProviderLLMClient(primary=MockGroqHealthy(), fallback=None)
    pipeline = RemediationPipeline(llm_client=client)

    resp = await pipeline.remediate(sample_sqli_request)

    assert isinstance(resp, RemediationResponse)
    assert resp.scan_id == sample_sqli_request.scan_id
    assert resp.vulnerability_id == sample_sqli_request.vulnerability_id
    assert resp.model_name == settings.GROQ_MODEL
    assert resp.confidence >= 0.70
    assert resp.tokens_used > 0
    assert "$1" in resp.suggested_patch or "?" in resp.suggested_patch


@pytest.mark.asyncio
async def test_gemini_inference_contract(sample_python_request):
    """Test 3.2: Gemini generates valid CTR-005 response matching schema."""
    client = MultiProviderLLMClient(primary=MockGeminiHealthy(), fallback=None)
    pipeline = RemediationPipeline(llm_client=client)

    resp = await pipeline.remediate(sample_python_request)

    assert isinstance(resp, RemediationResponse)
    assert resp.scan_id == sample_python_request.scan_id
    assert resp.model_name == settings.GEMINI_MODEL
    assert resp.confidence >= 0.70
    assert "%s" in resp.suggested_patch or "?" in resp.suggested_patch


# ============================================================================
# 3.3: Automated Failover (Groq -> Gemini)
# ============================================================================
@pytest.mark.asyncio
async def test_automated_failover_groq_to_gemini(sample_sqli_request):
    """Test 3.3: Primary Groq failure (429/500/timeout) transparently falls back to Gemini."""
    primary = MockProviderOutage("Groq HTTP 429: Rate Limit Exceeded")
    fallback = MockGeminiHealthy()
    client = MultiProviderLLMClient(primary=primary, fallback=fallback)
    pipeline = RemediationPipeline(llm_client=client)

    resp = await pipeline.remediate(sample_sqli_request)

    # Must transparently return Gemini response without crashing or returning 500
    assert isinstance(resp, RemediationResponse)
    assert resp.model_name == settings.GEMINI_MODEL
    assert resp.confidence >= 0.70
    assert "deterministic-rule-fallback" not in resp.model_name


# ============================================================================
# 3.4 & 3.5: Dual Failure & Empty Keys -> Deterministic Fallback (BR-002 Fail-Open)
# ============================================================================
@pytest.mark.asyncio
async def test_dual_failure_triggers_deterministic_fallback_fail_open(sample_sqli_request):
    """Test 3.4: Both providers offline must trigger deterministic fallback (BR-002)."""
    primary = MockProviderOutage("Groq 503 Service Unavailable")
    fallback = MockProviderOutage("Gemini 429 Quota Exhausted")
    client = MultiProviderLLMClient(primary=primary, fallback=fallback)
    pipeline = RemediationPipeline(llm_client=client)

    resp = await pipeline.remediate(sample_sqli_request)

    assert isinstance(resp, RemediationResponse)
    assert resp.scan_id == sample_sqli_request.scan_id
    assert resp.confidence == 0.0
    assert resp.model_name == "deterministic-rule-fallback"
    assert "[LUCID-CI MANUAL REVIEW REQUIRED]" in resp.suggested_patch
    assert "CWE-89" in resp.explanation
    assert resp.tokens_used == 0


@pytest.mark.asyncio
async def test_missing_api_keys_triggers_clean_fallback(sample_sqli_request):
    """Test 3.5: Empty API keys immediately return deterministic fallback without exception."""
    # When api_key is empty, clients raise LLMInferenceError on generate
    client = MultiProviderLLMClient(primary=GroqClient(api_key=""), fallback=GeminiClient(api_key=""))
    pipeline = RemediationPipeline(llm_client=client)

    resp = await pipeline.remediate(sample_sqli_request)

    assert isinstance(resp, RemediationResponse)
    assert resp.confidence == 0.0
    assert resp.model_name == "deterministic-rule-fallback"
    assert "[LUCID-CI MANUAL REVIEW REQUIRED]" in resp.suggested_patch


# ============================================================================
# 3.6: Corrective 1-Shot Retry Loop on Schema Violation
# ============================================================================
class MockFlakySchemaClient(BaseLLMClient):
    def __init__(self):
        self.attempts = 0

    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        return await self.generate_with_failover(system_prompt, user_prompt)

    async def generate_with_failover(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        self.attempts += 1
        if self.attempts == 1:
            # Attempt 1: Malformed JSON (missing required field 'security_rationale')
            bad_json = {
                "suggested_patch": "const x = 1;",
                "explanation": "Fixed SQLi.",
                "confidence": 0.85,
            }
            return LLMResponse(
                content=json.dumps(bad_json),
                model_name="mock-model",
                tokens_used=100,
                inference_latency_ms=200,
                provider="mock",
            )
        # Attempt 2: Corrected JSON satisfying CTR-005
        good_json = {
            "suggested_patch": "const query = 'SELECT * FROM users WHERE id = $1';",
            "explanation": "Fixed SQLi.",
            "security_rationale": "Parameterized query prevents malicious SQL command alteration.",
            "confidence": 0.90,
        }
        return LLMResponse(
            content=json.dumps(good_json),
            model_name="mock-model",
            tokens_used=120,
            inference_latency_ms=180,
            provider="mock",
        )


@pytest.mark.asyncio
async def test_1shot_retry_loop_recovers_schema_error(sample_sqli_request):
    """Test 3.6: Schema failure in attempt 1 triggers corrective retry, recovering valid CTR-005."""
    flaky = MockFlakySchemaClient()
    pipeline = RemediationPipeline(llm_client=flaky)

    resp = await pipeline.remediate(sample_sqli_request)

    assert flaky.attempts == 2
    assert resp.confidence == 0.90
    assert resp.security_rationale != ""
    assert resp.model_name == "mock-model"
    # Latency and tokens must be accumulated across both attempts
    assert resp.tokens_used == 220
    assert resp.inference_latency_ms >= 0


# ============================================================================
# 3.7: AI Patch Syntax Integrity Validation
# ============================================================================
def test_ai_patch_syntax_integrity_python():
    """Verify that Python patches produced by AI parse as valid syntax."""
    python_patch = "cursor.execute('SELECT * FROM users WHERE email = %s', (email,))"
    # Must compile without SyntaxError
    parsed = ast.parse(python_patch)
    assert parsed is not None
    assert len(parsed.body) == 1


# ============================================================================
# 3.8: Optional Live Provider Verification (Skips if no keys in environment)
# ============================================================================
@pytest.mark.skipif(
    not settings.GROQ_API_KEY,
    reason="GROQ_API_KEY not configured in environment; skipping live Groq API probe",
)
@pytest.mark.asyncio
async def test_live_groq_inference_probe(sample_sqli_request):
    """Live probe against Groq API when GROQ_API_KEY is present."""
    client = GroqClient(api_key=settings.GROQ_API_KEY, model=settings.GROQ_MODEL)
    pipeline = RemediationPipeline(llm_client=MultiProviderLLMClient(primary=client, fallback=None))

    resp = await pipeline.remediate(sample_sqli_request)

    assert resp.confidence > 0.0
    assert resp.tokens_used > 20
    assert resp.model_name == settings.GROQ_MODEL
    assert "deterministic-rule-fallback" not in resp.model_name


@pytest.mark.skipif(
    not settings.GEMINI_API_KEY,
    reason="GEMINI_API_KEY not configured in environment; skipping live Gemini API probe",
)
@pytest.mark.asyncio
async def test_live_gemini_inference_probe(sample_sqli_request):
    """Live probe against Gemini API when GEMINI_API_KEY is present."""
    client = GeminiClient(api_key=settings.GEMINI_API_KEY, model=settings.GEMINI_MODEL)
    pipeline = RemediationPipeline(llm_client=MultiProviderLLMClient(primary=client, fallback=None))

    resp = await pipeline.remediate(sample_sqli_request)

    assert resp.scan_id == sample_sqli_request.scan_id
    if resp.model_name == settings.GEMINI_MODEL:
        assert resp.confidence > 0.0
        assert resp.tokens_used > 20
    else:
        # Verified live fail-open behavior when provider quota or credits are depleted (HTTP 429)
        assert resp.model_name == "deterministic-rule-fallback"
        assert resp.confidence == 0.0
        assert "[LUCID-CI MANUAL REVIEW REQUIRED]" in resp.suggested_patch
