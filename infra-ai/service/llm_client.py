"""
Provider-Agnostic LLM Client Abstraction with Automated Failover.
Supports Groq (Llama-3.1-8B), Google Vertex AI Gemini (3.5 Flash-Lite),
Google Gemini Developer API, and local Ollama.
"""

import json
import os
import time
from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Optional
import structlog
import httpx
from groq import AsyncGroq
from google import genai
from google.genai import types

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

    async def generate_with_failover(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        """Default failover implementation delegates to direct generate for single-provider clients."""
        return await self.generate(system_prompt, user_prompt)


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


def _resolve_gcp_project(explicit_project: Optional[str] = None) -> Optional[str]:
    """
    Resolves GCP project ID from explicit argument, configuration settings,
    or Google Application Default Credentials (ADC) JSON file.
    """
    if explicit_project:
        return explicit_project
    if settings.GOOGLE_CLOUD_PROJECT:
        return settings.GOOGLE_CLOUD_PROJECT
    creds_path = os.environ.get("GOOGLE_APPLICATION_CREDENTIALS") or os.path.expanduser(
        "~/.config/gcloud/application_default_credentials.json"
    )
    if os.path.exists(creds_path):
        try:
            with open(creds_path, "r", encoding="utf-8") as f:
                data = json.load(f)
                return data.get("project_id") or data.get("quota_project_id")
        except Exception:
            pass
    return None


def _extract_response_text(response: any) -> str:
    """Extracts text content safely across standard and reasoning/thinking models."""
    if not response:
        return ""
    if hasattr(response, "candidates") and response.candidates:
        first_candidate = response.candidates[0]
        if hasattr(first_candidate, "content") and first_candidate.content:
            parts = getattr(first_candidate.content, "parts", None)
            if parts:
                text_parts = [part.text for part in parts if getattr(part, "text", None)]
                if text_parts:
                    return "".join(text_parts)
    if hasattr(response, "text") and response.text:
        return response.text
    return ""


# ============================================================================
# Google Cloud Vertex AI Provider Adapter (Gemini 3.5 Flash-Lite)
# ============================================================================
class VertexAIClient(BaseLLMClient):
    """
    Production Vertex AI Client utilizing the official google-genai SDK.
    Supports ADC (Application Default Credentials), Service Account keys via
    GOOGLE_APPLICATION_CREDENTIALS, or direct API keys.
    """

    def __init__(
        self,
        project: Optional[str] = None,
        location: Optional[str] = None,
        model: Optional[str] = None,
        client: Optional[genai.Client] = None,
    ):
        self.project = project if project is not None else settings.GOOGLE_CLOUD_PROJECT
        self.location = location if location is not None else settings.GOOGLE_CLOUD_LOCATION
        self.model = model if model is not None else (settings.VERTEX_AI_MODEL or settings.GEMINI_MODEL)
        self._client = client

    def _get_client(self) -> genai.Client:
        if not self._client:
            kwargs = {"vertexai": True}
            project = _resolve_gcp_project(self.project)
            if project:
                kwargs["project"] = project
            if self.location:
                kwargs["location"] = self.location
            try:
                self._client = genai.Client(**kwargs)
            except Exception as e:
                logger.warning("vertex_ai_client_init_failed", error=str(e))
                raise LLMInferenceError(f"Vertex AI initialization failed: {e}") from e
        return self._client

    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        client = self._get_client()
        start_time = time.perf_counter()

        try:
            config = types.GenerateContentConfig(
                system_instruction=system_prompt,
                temperature=settings.LLM_TEMPERATURE,
                max_output_tokens=settings.LLM_MAX_TOKENS,
            )

            response = await client.aio.models.generate_content(
                model=self.model,
                contents=user_prompt,
                config=config,
            )

            latency_ms = int((time.perf_counter() - start_time) * 1000)
            text_content = _extract_response_text(response)

            tokens = 0
            if hasattr(response, "usage_metadata") and response.usage_metadata:
                tokens = getattr(response.usage_metadata, "total_token_count", 0) or 0
            if tokens == 0 and text_content:
                tokens = len(system_prompt.split()) + len(user_prompt.split()) + len(text_content.split())

            return LLMResponse(
                content=text_content,
                model_name=self.model,
                tokens_used=tokens,
                inference_latency_ms=latency_ms,
                provider="vertexai",
            )
        except LLMInferenceError:
            raise
        except Exception as e:
            logger.warning("vertex_ai_inference_failed", error=str(e), model=self.model)
            raise LLMInferenceError(f"Vertex AI error: {e}") from e


# ============================================================================
# Google Gemini Provider Adapter (Backwards-Compatibility Alias)
# ============================================================================
class GeminiClient(BaseLLMClient):
    """
    Backwards-compatible Gemini client adapter.
    Uses Google GenAI SDK with api_key if provided, or Vertex AI if configured.
    """

    def __init__(
        self,
        api_key: Optional[str] = None,
        model: Optional[str] = None,
        client: Optional[genai.Client] = None,
    ):
        self.api_key = api_key if api_key is not None else settings.GEMINI_API_KEY
        self.model = model if model is not None else (settings.GEMINI_MODEL or settings.VERTEX_AI_MODEL)
        self._client = client

    def _get_client(self) -> genai.Client:
        if not self._client:
            if not self.api_key:
                raise LLMInferenceError("Gemini API key not configured")
            try:
                self._client = genai.Client(api_key=self.api_key)
            except Exception as e:
                raise LLMInferenceError(f"Gemini API key initialization failed: {e}") from e
        return self._client

    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        client = self._get_client()
        start_time = time.perf_counter()

        try:
            config = types.GenerateContentConfig(
                system_instruction=system_prompt,
                temperature=settings.LLM_TEMPERATURE,
                max_output_tokens=settings.LLM_MAX_TOKENS,
            )

            response = await client.aio.models.generate_content(
                model=self.model,
                contents=user_prompt,
                config=config,
            )

            latency_ms = int((time.perf_counter() - start_time) * 1000)
            text_content = _extract_response_text(response)

            tokens = 0
            if hasattr(response, "usage_metadata") and response.usage_metadata:
                tokens = getattr(response.usage_metadata, "total_token_count", 0) or 0
            if tokens == 0 and text_content:
                tokens = len(system_prompt.split()) + len(user_prompt.split()) + len(text_content.split())

            return LLMResponse(
                content=text_content,
                model_name=self.model,
                tokens_used=tokens,
                inference_latency_ms=latency_ms,
                provider="gemini",
            )
        except LLMInferenceError:
            raise
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
class MultiProviderLLMClient(BaseLLMClient):
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

    async def generate(self, system_prompt: str, user_prompt: str) -> LLMResponse:
        """Implements BaseLLMClient interface by executing inference with automated failover."""
        return await self.generate_with_failover(system_prompt, user_prompt)

    def _resolve_client(self, name: str) -> Optional[BaseLLMClient]:
        match name.lower():
            case "groq":
                return GroqClient()
            case "vertex" | "vertexai":
                return VertexAIClient()
            case "gemini":
                if settings.GEMINI_API_KEY:
                    return GeminiClient()
                if _resolve_gcp_project():
                    return VertexAIClient()
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
