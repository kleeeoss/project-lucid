"""unit tests for Provider-Agnostic LLM Client & Failover Mechanism.
Includes Groq, Google Vertex AI Gemini (3.5 Flash-Lite), and Ollama adapters.
Uses mocks to verify resilience without burning live API quotas.
"""

import pytest
from unittest.mock import AsyncMock, MagicMock, patch
from service.llm_client import (
    BaseLLMClient,
    GroqClient,
    VertexAIClient,
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
    fallback = MockHealthyProvider("vertex-fallback")

    orchestrator = MultiProviderLLMClient(primary=primary, fallback=fallback)
    res = await orchestrator.generate_with_failover("system", "user")

    assert res.provider == "vertex-fallback"
    assert res.inference_latency_ms == 350


@pytest.mark.asyncio
async def test_all_providers_exhausted_raises_exception():
    """Verify LLMInferenceError is raised if both primary and fallback fail."""
    primary = MockFailingProvider("Groq 503 Outage")
    fallback = MockFailingProvider("Vertex AI 429 Quota Depleted")

    orchestrator = MultiProviderLLMClient(primary=primary, fallback=fallback)
    with pytest.raises(LLMInferenceError) as exc_info:
        await orchestrator.generate_with_failover("system", "user")

    assert "All AI inference providers failed" in str(exc_info.value)
    assert "Groq 503" in str(exc_info.value)
    assert "Vertex AI 429" in str(exc_info.value)


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


# =========================================================================
# Google Cloud Vertex AI Client Tests
# =========================================================================

@pytest.mark.asyncio
async def test_vertex_ai_generate_success():
    """Verify VertexAIClient successfully calls google-genai async API and returns structured LLMResponse."""
    mock_response = MagicMock()
    mock_response.text = '{"suggested_patch": "const id = req.params.id;", "explanation": "fixed", "confidence": 0.95}'
    mock_response.usage_metadata.total_token_count = 142

    mock_client = MagicMock()
    mock_client.aio.models.generate_content = AsyncMock(return_value=mock_response)

    vertex_client = VertexAIClient(
        project="lucid-ci-prod",
        location="us-central1",
        model="gemini-3.5-flash-lite",
        client=mock_client,
    )

    res = await vertex_client.generate("system instructions", "user prompt")

    assert res.provider == "vertexai"
    assert res.model_name == "gemini-3.5-flash-lite"
    assert res.tokens_used == 142
    assert "const id = req.params.id" in res.content
    assert res.inference_latency_ms >= 0

    # Verify call parameters
    mock_client.aio.models.generate_content.assert_called_once()
    call_kwargs = mock_client.aio.models.generate_content.call_args.kwargs
    assert call_kwargs["model"] == "gemini-3.5-flash-lite"
    assert call_kwargs["contents"] == "user prompt"
    assert call_kwargs["config"].system_instruction == "system instructions"


@pytest.mark.asyncio
async def test_vertex_ai_error_wraps_in_llm_inference_error():
    """Verify exceptions from google-genai are wrapped in LLMInferenceError."""
    mock_client = MagicMock()
    mock_client.aio.models.generate_content = AsyncMock(side_effect=RuntimeError("Google Vertex AI 503 Service Unavailable"))

    vertex_client = VertexAIClient(project="lucid-ci-prod", client=mock_client)
    with pytest.raises(LLMInferenceError) as exc_info:
        await vertex_client.generate("sys", "user")

    assert "Vertex AI error" in str(exc_info.value)
    assert "503" in str(exc_info.value)


@pytest.mark.asyncio
async def test_failover_groq_to_vertex_ai():
    """Verify automatic failover from Groq to Vertex AI."""
    primary = MockFailingProvider("HTTP 429 Groq Rate Limit")
    fallback = MockHealthyProvider("vertexai")

    orchestrator = MultiProviderLLMClient(primary=primary, fallback=fallback)
    res = await orchestrator.generate_with_failover("system", "user")

    assert res.provider == "vertexai"


@pytest.mark.asyncio
async def test_failover_vertex_ai_to_groq():
    """Verify automatic failover when Vertex AI is primary and Groq is fallback."""
    primary = MockFailingProvider("Vertex AI Quota Depleted")
    fallback = MockHealthyProvider("groq")

    orchestrator = MultiProviderLLMClient(primary=primary, fallback=fallback)
    res = await orchestrator.generate_with_failover("system", "user")

    assert res.provider == "groq"


def test_multi_provider_resolve_vertexai():
    """Verify that MultiProviderLLMClient resolves 'vertex' and 'vertexai' to VertexAIClient."""
    orchestrator = MultiProviderLLMClient(primary=None, fallback=None)
    c1 = orchestrator._resolve_client("vertexai")
    c2 = orchestrator._resolve_client("vertex")

    assert isinstance(c1, VertexAIClient)
    assert isinstance(c2, VertexAIClient)


def test_multi_provider_resolve_gemini_with_api_key():
    """Verify that MultiProviderLLMClient resolves 'gemini' to GeminiClient when GEMINI_API_KEY is configured."""
    orchestrator = MultiProviderLLMClient(primary=None, fallback=None)
    with patch("service.llm_client.settings.GEMINI_API_KEY", "test-api-key"):
        client = orchestrator._resolve_client("gemini")
        assert isinstance(client, GeminiClient)


def test_multi_provider_resolve_gemini_falls_back_to_vertex():
    """Verify that MultiProviderLLMClient upgrades 'gemini' to VertexAIClient when API key is empty but GCP project exists."""
    orchestrator = MultiProviderLLMClient(primary=None, fallback=None)
    with patch("service.llm_client.settings.GEMINI_API_KEY", ""):
        with patch("service.llm_client._resolve_gcp_project", return_value="lucid-ci-project"):
            client = orchestrator._resolve_client("gemini")
            assert isinstance(client, VertexAIClient)


def test_multi_provider_resolve_gemini_without_key_or_gcp():
    """Verify that MultiProviderLLMClient returns GeminiClient when neither API key nor GCP project is available."""
    orchestrator = MultiProviderLLMClient(primary=None, fallback=None)
    with patch("service.llm_client.settings.GEMINI_API_KEY", ""):
        with patch("service.llm_client._resolve_gcp_project", return_value=None):
            client = orchestrator._resolve_client("gemini")
            assert isinstance(client, GeminiClient)

