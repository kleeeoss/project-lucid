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
from service.remediation import RemediationPipeline

router = APIRouter()
pipeline = RemediationPipeline()


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
    Production Remediation Pipeline (TASK-INF-203):
    Coordinates nonce prompt generation, LLM client with failover,
    1-shot Pydantic validation retry, and deterministic fallback (BR-002).
    """
    return await pipeline.remediate(request)


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
    (TASK-INF-205 connects this to real Docker/gVisor runner execution).
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
