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
from sandbox.orchestrator import SandboxOrchestrator

router = APIRouter()
pipeline = RemediationPipeline()
orchestrator = SandboxOrchestrator()


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
    Production Dynamic Detonation Pipeline (TASK-INF-205):
    Launches ephemeral container with --network none, memory/cpu limits,
    enforces 60-second watchdog timeout, and guarantees complete cleanup.
    """
    return await orchestrator.detonate(request)
