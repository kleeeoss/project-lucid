"""
LUCID-CI — Remediation Pipeline Orchestrator.
Coordinates prompt generation, LLM client, 1-shot retry loop, and fallback.
"""

import time
from typing import Optional
import structlog

from models.schemas import RemediationRequest, RemediationResponse
from prompts.remediation_prompt import build_remediation_prompt
from service.llm_client import BaseLLMClient, MultiProviderLLMClient, LLMInferenceError
from service.parser import parse_and_validate_remediation, build_deterministic_fallback

logger = structlog.get_logger()


class RemediationPipeline:
    """End-to-end remediation pipeline with automated retry and fallback."""

    def __init__(self, llm_client: Optional[BaseLLMClient] = None):
        self.llm_client = llm_client or MultiProviderLLMClient()

    async def remediate(self, request: RemediationRequest) -> RemediationResponse:
        start_time = time.perf_counter()

        # 1. Assemble nonce-hardened prompts
        prompt_bundle = build_remediation_prompt(request)

        # 2. Attempt 1: Standard inference
        raw_response = None
        try:
            raw_response = await self.llm_client.generate_with_failover(
                system_prompt=prompt_bundle.system_prompt,
                user_prompt=prompt_bundle.user_prompt,
            )
        except LLMInferenceError as err:
            latency_ms = int((time.perf_counter() - start_time) * 1000)
            return build_deterministic_fallback(request, f"AI Provider Outage: {err}", latency_ms)

        # 3. Parse and validate Attempt 1
        validated, err_msg = parse_and_validate_remediation(
            raw_content=raw_response.content,
            request=request,
            model_name=raw_response.model_name,
            tokens_used=raw_response.tokens_used,
            latency_ms=raw_response.inference_latency_ms,
        )

        if validated:
            return validated

        # 4. Attempt 2: 1-Shot Corrective Retry Loop
        logger.info(
            "remediation_retry_triggered",
            scan_id=request.scan_id,
            validation_error=err_msg,
        )
        retry_user_prompt = (
            f"{prompt_bundle.user_prompt}\n\n"
            f"### CRITICAL ERROR IN PREVIOUS ATTEMPT:\n"
            f"Your previous output failed validation: {err_msg}\n"
            f"Fix the error and output ONLY a valid JSON object matching the required schema."
        )

        try:
            retry_raw_response = await self.llm_client.generate_with_failover(
                system_prompt=prompt_bundle.system_prompt,
                user_prompt=retry_user_prompt,
            )
            total_latency_ms = int((time.perf_counter() - start_time) * 1000)
            total_tokens = raw_response.tokens_used + retry_raw_response.tokens_used

            retry_validated, retry_err = parse_and_validate_remediation(
                raw_content=retry_raw_response.content,
                request=request,
                model_name=retry_raw_response.model_name,
                tokens_used=total_tokens,
                latency_ms=total_latency_ms,
            )

            if retry_validated:
                return retry_validated
            err_msg = f"Retry also failed: {retry_err}"
        except LLMInferenceError as retry_err:
            err_msg = f"Retry provider error: {retry_err}"

        # 5. Exhausted retries -> Deterministic Fallback (BR-002)
        total_latency_ms = int((time.perf_counter() - start_time) * 1000)
        return build_deterministic_fallback(request, err_msg, total_latency_ms)
