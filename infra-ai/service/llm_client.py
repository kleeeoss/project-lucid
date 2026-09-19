"""
Provider-Agnostic LLM Client Abstraction with Automated Failover.
Supports Groq (Llama-3.1-8B), Google Gemini (1.5 Flash), and local Ollama.
"""

import time
from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Optional
import structlog
import httpx
from groq import AsyncGroq
import google.generativeai as genai

from service.config import settings

logger = structlog.get_logger()


class LLMInferenceError(Exception):
    """Raised when an LLM provider fails inference or exhausts retries."""
    pass


@dataclass
class LLMResponse:
    """Standardized raw response envelope across any AI vendor."""
    content: str
    model_name: str
    tokens_used: int
    inference_latency_ms: int
    provider: str


class BaseLLMClient(ABC):
    """Abstract interface defining required LLM vendor adapter methods."""

    @abstractmethod
    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        """Execute async inference and return normalized LLMResponse."""
        pass


# ============================================================================
# Groq Provider Adapter (Default Primary)
# ============================================================================
class GroqClient(BaseLLMClient):
    def __init__(self, api_key: Optional[str] = None, model: Optional[str] = None):
        self.api_key = api_key if api_key is not None else settings.GROQ_API_KEY
        self.model = model if model is not None else settings.GROQ_MODEL
        self._client: Optional[AsyncGroq] = None

    def _get_client(self) -> AsyncGroq:
        if not self._client:
            self._client = AsyncGroq(api_key=self.api_key)
        return self._client

    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        if not self.api_key:
            raise LLMInferenceError("Groq API key not configured")

        client = self._get_client()
        start_time = time.perf_counter()

        try:
            response = await client.chat.completions.create(
                model=self.model,
                messages=[
                    {"role": "system", "content": system_prompt},
                    {"role": "user", "content": user_prompt},
                ],
                temperature=settings.LLM_TEMPERATURE,
                max_tokens=settings.LLM_MAX_TOKENS,
                timeout=settings.LLM_TIMEOUT_SECONDS,
            )

            latency_ms = int((time.perf_counter() - start_time) * 1000)
            choice = response.choices[0]
            tokens = response.usage.total_tokens if response.usage else 0

            return LLMResponse(
                content=choice.message.content or "",
                model_name=self.model,
                tokens_used=tokens,
                inference_latency_ms=latency_ms,
                provider="groq",
            )
        except Exception as e:
            logger.warning("groq_inference_failed", error=str(e), model=self.model)
            raise LLMInferenceError(f"Groq error: {e}") from e


# ============================================================================
# Google Gemini Provider Adapter (Default Fallback)
# ============================================================================
class GeminiClient(BaseLLMClient):
    def __init__(self, api_key: Optional[str] = None, model: Optional[str] = None):
        self.api_key = api_key if api_key is not None else settings.GEMINI_API_KEY
        self.model = model if model is not None else settings.GEMINI_MODEL
        self._configured = False

    def _setup(self):
        if not self._configured:
            if not self.api_key:
                raise LLMInferenceError("Gemini API key not configured")
            genai.configure(api_key=self.api_key)
            self._configured = True

    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        self._setup()
        start_time = time.perf_counter()

        try:
            # Combine system prompt with user prompt for Gemini standard API
            combined_prompt = f"System Instructions:\n{system_prompt}\n\nTask:\n{user_prompt}"
            model_instance = genai.GenerativeModel(self.model)

            # genai.generate_content_async provides native non-blocking async inference
            response = await model_instance.generate_content_async(
                combined_prompt,
                generation_config=genai.types.GenerationConfig(
                    temperature=settings.LLM_TEMPERATURE,
                    max_output_tokens=settings.LLM_MAX_TOKENS,
                ),
            )

            latency_ms = int((time.perf_counter() - start_time) * 1000)
            text_content = response.text if response.text else ""

            # Approximate token estimation if Gemini metadata is omitted
            estimated_tokens = len(combined_prompt.split()) + len(text_content.split())

            return LLMResponse(
                content=text_content,
                model_name=self.model,
                tokens_used=estimated_tokens,
                inference_latency_ms=latency_ms,
                provider="gemini",
            )
        except Exception as e:
            logger.warning("gemini_inference_failed", error=str(e), model=self.model)
            raise LLMInferenceError(f"Gemini error: {e}") from e


# ============================================================================
# Local Ollama Provider Adapter (Offline / Air-gapped)
# ============================================================================
class OllamaClient(BaseLLMClient):
    def __init__(self, base_url: Optional[str] = None, model: Optional[str] = None):
        raw_url = base_url if base_url is not None else settings.OLLAMA_BASE_URL
        self.base_url = raw_url.rstrip("/")
        self.model = model if model is not None else settings.OLLAMA_MODEL

    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        start_time = time.perf_counter()
        payload = {
            "model": self.model,
            "system": system_prompt,
            "prompt": user_prompt,
            "stream": False,
            "options": {
                "temperature": settings.LLM_TEMPERATURE,
                "num_predict": settings.LLM_MAX_TOKENS,
            },
        }

        try:
            async with httpx.AsyncClient(timeout=settings.LLM_TIMEOUT_SECONDS) as client:
                res = await client.post(f"{self.base_url}/api/generate", json=payload)
                res.raise_for_status()
                data = res.json()

            latency_ms = int((time.perf_counter() - start_time) * 1000)
            tokens = data.get("eval_count", 0) + data.get("prompt_eval_count", 0)

            return LLMResponse(
                content=data.get("response", ""),
                model_name=self.model,
                tokens_used=tokens,
                inference_latency_ms=latency_ms,
                provider="ollama",
            )
        except Exception as e:
            logger.warning("ollama_inference_failed", error=str(e), model=self.model)
            raise LLMInferenceError(f"Ollama error: {e}") from e


# ============================================================================
# Multi-Provider Resilient Orchestrator (Failover Manager)
# ============================================================================
class MultiProviderLLMClient:
    """
    Coordinates primary and fallback AI providers.
    Automatically catches transient rate-limits (429), outages (503), or timeouts,
    triggering instant failover to secondary provider.
    """

    def __init__(
        self,
        primary: Optional[BaseLLMClient] = None,
        fallback: Optional[BaseLLMClient] = None,
    ):
        self.primary = primary or self._resolve_client(settings.LLM_PRIMARY_PROVIDER)
        self.fallback = fallback or self._resolve_client(settings.LLM_FALLBACK_PROVIDER)

    def _resolve_client(self, name: str) -> Optional[BaseLLMClient]:
        match name.lower():
            case "groq":
                return GroqClient()
            case "gemini":
                return GeminiClient()
            case "ollama":
                return OllamaClient()
            case "none" | "":
                return None
            case _:
                logger.error("unknown_llm_provider", provider=name)
                return None

    async def generate_with_failover(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        errors = []

        # 1. Attempt Primary Provider
        if self.primary:
            try:
                return await self.primary.generate(system_prompt, user_prompt)
            except LLMInferenceError as e:
                errors.append(f"Primary failed: {e}")
                logger.warning("llm_failover_triggered", reason=str(e))

        # 2. Attempt Fallback Provider
        if self.fallback:
            try:
                logger.info("attempting_fallback_provider", fallback=type(self.fallback).__name__)
                return await self.fallback.generate(system_prompt, user_prompt)
            except LLMInferenceError as e:
                errors.append(f"Fallback failed: {e}")

        # 3. Exhausted all configured providers
        error_msg = " | ".join(errors) if errors else "No LLM providers configured"
        logger.error("all_llm_providers_exhausted", details=error_msg)
        raise LLMInferenceError(f"All AI inference providers failed: {error_msg}")
