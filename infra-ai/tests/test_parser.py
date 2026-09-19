"""
Unit tests for Parser, Extraction, 1-Shot Retry Loop, and Fallback.
"""

import pytest
from models.mock_payloads import create_mock_remediation_request
from service.llm_client import BaseLLMClient, LLMResponse, LLMInferenceError
from service.parser import (
    extract_json_payload,
    parse_and_validate_remediation,
    build_deterministic_fallback,
)
from service.remediation import RemediationPipeline


def test_extract_json_payload_raw_json():
    raw = '{"suggested_patch": "const x = 1;", "confidence": 0.9}'
    assert extract_json_payload(raw) == raw


def test_extract_json_payload_markdown_code_block():
    raw = """Here is the fix:
```json
{
  "suggested_patch": "const x = 1;",
  "confidence": 0.95
}
```
Hope this helps!"""
    extracted = extract_json_payload(raw)
    assert extracted.startswith("{") and extracted.endswith("}")
    assert '"confidence": 0.95' in extracted


def test_extract_json_payload_embedded_braces():
    raw = 'Sure! { "suggested_patch": "test" } done.'
    assert extract_json_payload(raw) == '{ "suggested_patch": "test" }'


def test_parse_and_validate_valid_response():
    req = create_mock_remediation_request()
    raw = """{
  "suggested_patch": "const query = 'SELECT * FROM users WHERE id = $1';",
  "explanation": "SQL injection vulnerability via direct concatenation.",
  "security_rationale": "Parameterized queries sanitize input.",
  "confidence": 0.92
}"""
    validated, err = parse_and_validate_remediation(
        raw_content=raw,
        request=req,
        model_name="mock-model",
        tokens_used=100,
        latency_ms=200,
    )

    assert err is None
    assert validated is not None
    assert validated.scan_id == req.scan_id
    assert validated.confidence == 0.92
    assert validated.model_name == "mock-model"


def test_deterministic_fallback_generation():
    req = create_mock_remediation_request(cwe="CWE-89", rule_id="LUCID-SEC-001")
    fallback = build_deterministic_fallback(req, "Service timeout", latency_ms=500)

    assert fallback.scan_id == req.scan_id
    assert fallback.confidence == 0.0
    assert fallback.model_name == "deterministic-rule-fallback"
    assert "CWE-89" in fallback.explanation
    assert "[LUCID-CI MANUAL REVIEW REQUIRED]" in fallback.suggested_patch


class MockFlakyLLMClient(BaseLLMClient):
    """Fails on attempt 1 with malformed JSON; succeeds on attempt 2."""

    def __init__(self):
        self.call_count = 0

    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        return await self.generate_with_failover(system_prompt, user_prompt)

    async def generate_with_failover(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        self.call_count += 1
        if self.call_count == 1:
            # Attempt 1: Malformed output missing valid JSON structure
            return LLMResponse(
                content="Here is some broken conversational text without valid JSON",
                model_name="test-model",
                tokens_used=50,
                inference_latency_ms=100,
                provider="mock",
            )
        # Attempt 2 (Retry): Valid JSON
        return LLMResponse(
            content="""```json
{
  "suggested_patch": "const x = 1;",
  "explanation": "Fixed SQLi.",
  "security_rationale": "Parameterized query.",
  "confidence": 0.88
}
```""",
            model_name="test-model",
            tokens_used=80,
            inference_latency_ms=150,
            provider="mock",
        )


@pytest.mark.asyncio
async def test_remediation_pipeline_recovers_on_retry():
    req = create_mock_remediation_request()
    flaky_client = MockFlakyLLMClient()
    pipeline = RemediationPipeline(llm_client=flaky_client)

    result = await pipeline.remediate(req)

    assert flaky_client.call_count == 2
    assert result.confidence == 0.88
    assert result.explanation == "Fixed SQLi."
    assert result.model_name == "test-model"


class MockPermanentlyFailingLLMClient(BaseLLMClient):
    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        return await self.generate_with_failover(system_prompt, user_prompt)

    async def generate_with_failover(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        raise LLMInferenceError("Groq and Gemini both offline")


@pytest.mark.asyncio
async def test_remediation_pipeline_fails_open_on_provider_outage():
    req = create_mock_remediation_request()
    failing_client = MockPermanentlyFailingLLMClient()
    pipeline = RemediationPipeline(llm_client=failing_client)

    result = await pipeline.remediate(req)

    # Must NOT raise exception; must return deterministic fallback (BR-002)
    assert result.scan_id == req.scan_id
    assert result.confidence == 0.0
    assert result.model_name == "deterministic-rule-fallback"
    assert "Groq and Gemini both offline" in result.explanation
