"""
API endpoints for /infra-ai service.
Exposes /healthz, /remediate (CTR-004 -> CTR-005), and /sandbox/detonate (CTR-006 -> CTR-007).
"""

from fastapi import APIRouter, status
from models.schemas import (
    RemediationRequest,
    RemediationResponse,
    SandboxRequest,
    SandboxResult,
)

router = APIRouter()


@router.get("/healthz", status_code=status.HTTP_200_OK)
async def health_check():
    """Liveness probe for Docker, Caddy, and ALB."""
    return {"status": "healthy", "service": "lucid-ai"}


@router.post(
    "/remediate",
    response_model=RemediationResponse,
    status_code=status.HTTP_200_OK,
    summary="Generate AI remediation patch for detected vulnerability",
)
async def remediate(request: RemediationRequest) -> RemediationResponse:
    """
    Phase 1 Mock Implementation:
    Returns a deterministic, correlated remediation response matching CTR-005.
    (Phase 2 replaces this mock with live Groq / Gemini SLM inference).
    """
    # Dynamic synthetic fix adapted to requested CWE
    suggested_patch = (
        "const query = 'SELECT * FROM users WHERE id = $1';\n"
        "const user = await db.query(query, [userId]);"
    )
    explanation = (
        f"Detected {request.cwe} in file using rule {request.rule_id}. "
        f"Untrusted source '{request.source_info}' propagates directly into dangerous execution sink '{request.sink_info}'."
    )
    security_rationale = (
        "Replaced dynamic string interpolation with parameterized query placeholders ($1). "
        "Untrusted input is treated strictly as literal data by the database driver, neutralizing code injection."
    )

    return RemediationResponse(
        scan_id=request.scan_id,
        vulnerability_id=request.vulnerability_id,
        suggested_patch=suggested_patch,
        explanation=explanation,
        security_rationale=security_rationale,
        confidence=0.95,
        model_name="mock-slm-phase1",
        tokens_used=180,
        inference_latency_ms=10,
    )


@router.post(
    "/sandbox/detonate",
    response_model=SandboxResult,
    status_code=status.HTTP_200_OK,
    summary="Execute untrusted PR code in isolated gVisor container",
)
async def detonate(request: SandboxRequest) -> SandboxResult:
    """
    Phase 1 Mock Implementation:
    Returns clean mock execution results matching CTR-007.
    (Phase 2 replaces this mock with real Docker/gVisor runner execution).
    """
    return SandboxResult(
        scan_id=request.scan_id,
        status="PASSED",
        exit_code=0,
        duration_ms=850,
        stdout=f"Mock detonation completed cleanly for scan {request.scan_id}.\nTests passed: 1/1.\n",
        stderr="",
        network_egress_attempts=0,
    )