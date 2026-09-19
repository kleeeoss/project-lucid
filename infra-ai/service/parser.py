"""
LUCID-CI — Response Parsing, JSON Extraction, and Deterministic Fallback.
Guarantees 100% schema compliance for CTR-005.
"""

import json
import re
from typing import Any, Dict, Optional, Tuple
from pydantic import ValidationError
import structlog

from models.schemas import RemediationRequest, RemediationResponse

logger = structlog.get_logger()


def extract_json_payload(raw_text: str) -> str:
    """
    Extracts raw JSON substring from LLM completion text.
    Handles ```json ... ``` markdown blocks, leading/trailing conversational text,
    and whitespace.
    """
    cleaned = raw_text.strip()

    # 1. Match standard markdown code blocks
    markdown_match = re.search(r"```(?:json)?\s*(\{.*?\})\s*```", cleaned, re.DOTALL)
    if markdown_match:
        return markdown_match.group(1).strip()

    # 2. Find outermost curly braces { ... }
    first_brace = cleaned.find("{")
    last_brace = cleaned.rfind("}")
    if first_brace != -1 and last_brace != -1 and last_brace > first_brace:
        return cleaned[first_brace : last_brace + 1].strip()

    return cleaned


def parse_and_validate_remediation(
    raw_content: str,
    request: RemediationRequest,
    model_name: str,
    tokens_used: int,
    latency_ms: int,
) -> Tuple[Optional[RemediationResponse], Optional[str]]:
    """
    Attempts to parse and validate LLM output against RemediationResponse.
    Returns:
        (RemediationResponse, None) on success.
        (None, error_message) on failure.
    """
    json_str = extract_json_payload(raw_content)

    try:
        parsed_data = json.loads(json_str)
        if not isinstance(parsed_data, dict):
            return None, "Model output parsed as valid JSON, but was not a JSON object/dictionary."
    except json.JSONDecodeError as err:
        return None, f"Malformed JSON: {err}"

    # Deterministically bind correlated IDs from request and telemetry
    candidate_data: Dict[str, Any] = {
        "scan_id": request.scan_id,
        "vulnerability_id": request.vulnerability_id,
        "suggested_patch": parsed_data.get("suggested_patch"),
        "explanation": parsed_data.get("explanation"),
        "security_rationale": parsed_data.get("security_rationale"),
        "confidence": parsed_data.get("confidence"),
        "model_name": model_name,
        "tokens_used": tokens_used,
        "inference_latency_ms": latency_ms,
    }

    try:
        validated = RemediationResponse.model_validate(candidate_data)
        return validated, None
    except ValidationError as err:
        return None, f"Schema validation failed: {err}"


def build_deterministic_fallback(
    request: RemediationRequest,
    reason: str,
    latency_ms: int = 0,
) -> RemediationResponse:
    """
    Constructs a deterministic, rule-based response when the LLM fails or is unavailable.
    Satisfies BR-002 (Fail-Open): Never blocks CI due to AI service failure.
    """
    logger.warning(
        "generating_deterministic_fallback",
        scan_id=request.scan_id,
        vulnerability_id=request.vulnerability_id,
        reason=reason,
    )

    explanation = (
        f"Automated AI patch generation unavailable ({reason}). "
        f"Detected {request.cwe} via rule {request.rule_id}. "
        f"Untrusted source '{request.source_info}' flows directly into sink '{request.sink_info}'."
    )
    security_rationale = (
        f"Manual security review required. Ensure all untrusted inputs from {request.source_info} "
        f"are sanitized or parameterized before execution in {request.sink_info}."
    )
    fallback_patch = (
        f"// [LUCID-CI MANUAL REVIEW REQUIRED]\n"
        f"// Vulnerability: {request.cwe} ({request.rule_id})\n"
        f"// Remediation patch could not be automatically synthesized.\n"
        f"// Review source: {request.source_info}\n"
        f"// Review sink:   {request.sink_info}\n"
    )

    return RemediationResponse(
        scan_id=request.scan_id,
        vulnerability_id=request.vulnerability_id,
        suggested_patch=fallback_patch,
        explanation=explanation,
        security_rationale=security_rationale,
        confidence=0.0,
        model_name="deterministic-rule-fallback",
        tokens_used=0,
        inference_latency_ms=latency_ms,
    )
