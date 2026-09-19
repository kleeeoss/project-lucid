"""
Unit tests for Provider-Agnostic LLM Client & Failover Mechanism.
Uses mocks to verify resilience without burning live API quotas.
"""

import pytest
from unittest.mock import AsyncMock, patch
from service.llm_client import (
    BaseLLMClient,
    GroqClient,
    GeminiClient,
    OllamaClient,
    MultiProviderLLMClient,
    LLMResponse,
    LLMInferenceError,
)


class MockHealthyProvider(BaseLLMClient):
    def __init__(self, name: str = "mock-primary"):
        self.name = name

    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        return LLMResponse(
            content='{"status": "remediated"}',
            model_name="mock-model-v1",
            tokens_used=120,
            inference_latency_ms=350,
            provider=self.name,
        )


class MockFailingProvider(BaseLLMClient):
    def __init__(self, error_message: str = "HTTP 429 Rate Limit Exceeded"):
        self.error_message = error_message

    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        raise LLMInferenceError(self.error_message)


@pytest.mark.asyncio
async def test_primary_provider_success():
    """Verify primary provider returns cleanly when healthy."""
    primary = MockHealthyProvider("groq-mock")
    fallback = MockFailingProvider("Fallback should not be called")

    orchestrator = MultiProviderLLMClient(primary=primary, fallback=fallback)
    res = await orchestrator.generate_with_failover("system", "user")

    assert res.provider == "groq-mock"
    assert res.tokens_used == 120
    assert '{"status": "remediated"}' in res.content


@pytest.mark.asyncio
async def test_failover_on_primary_rate_limit():
    """Verify primary 429 error automatically triggers fallback provider."""
    primary = MockFailingProvider("HTTP 429 Too Many Requests")
    fallback = MockHealthyProvider("gemini-fallback")

    orchestrator = MultiProviderLLMClient(primary=primary, fallback=fallback)
    res = await orchestrator.generate_with_failover("system", "user")

    assert res.provider == "gemini-fallback"
    assert res.inference_latency_ms == 350


@pytest.mark.asyncio
async def test_all_providers_exhausted_raises_exception():
    """Verify LLMInferenceError is raised if both primary and fallback fail."""
    primary = MockFailingProvider("Groq 503 Outage")
    fallback = MockFailingProvider("Gemini 429 Quota Depleted")

    orchestrator = MultiProviderLLMClient(primary=primary, fallback=fallback)
    with pytest.raises(LLMInferenceError) as exc_info:
        await orchestrator.generate_with_failover("system", "user")

    assert "All AI inference providers failed" in str(exc_info.value)
    assert "Groq 503" in str(exc_info.value)
    assert "Gemini 429" in str(exc_info.value)


@pytest.mark.asyncio
async def test_groq_missing_api_key_raises():
    client = GroqClient(api_key="")
    with pytest.raises(LLMInferenceError) as exc:
        await client.generate("sys", "user")
    assert "Groq API key not configured" in str(exc.value)


@pytest.mark.asyncio
async def test_gemini_missing_api_key_raises():
    client = GeminiClient(api_key="")
    with pytest.raises(LLMInferenceError) as exc:
        await client.generate("sys", "user")
    assert "Gemini API key not configured" in str(exc.value)
