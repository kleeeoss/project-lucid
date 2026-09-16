"""
LUCID-CI — Master Contracts Definition for /infra-ai
Governs CTR-004, CTR-005, CTR-006, CTR-007.
"""

from typing import Dict, List, Literal, Optional
from pydantic import BaseModel, ConfigDict, Field


class ASTNodeLocation(BaseModel):
    """Line and byte offsets for accurate patch placement."""
    model_config = ConfigDict(extra="forbid")

    line: int = Field(..., ge=1, description="1-based line number")
    column: int = Field(..., ge=0, description="0-based column offset")
    byte_start: Optional[int] = Field(None, ge=0)
    byte_end: Optional[int] = Field(None, ge=0)


# ============================================================================
# CTR-004: AI Remediation Request (Input to AI Microservice)
# ============================================================================
class RemediationRequest(BaseModel):
    """
    Contract CTR-004: Received by POST /remediate from Platform Worker.
    Represents a single static vulnerability finding requiring AI remediation.
    """
    model_config = ConfigDict(extra="forbid")

    scan_id: str = Field(..., min_length=1, description="Unique UUID of the scan run")
    vulnerability_id: str = Field(..., min_length=1, description="Unique UUID of detected vulnerability")
    rule_id: str = Field(..., min_length=1, description="Rule identifier, e.g. LUCID-SEC-001")
    cwe: str = Field(..., min_length=1, description="Common Weakness Enumeration, e.g. CWE-89")
    language: Literal["javascript", "typescript", "python", "go"] = Field(
        ..., description="Programming language of the target file"
    )
    vulnerable_code: str = Field(..., min_length=1, description="Exact vulnerable code slice detected")
    surrounding_context: str = Field(
        ..., min_length=1, description="Enclosing function or ±20 lines of surrounding code context"
    )
    source_info: str = Field(..., min_length=1, description="Identified untrusted input source (e.g. req.body.userId)")
    sink_info: str = Field(..., min_length=1, description="Identified dangerous execution sink (e.g. db.query)")
    taint_path_summary: List[str] = Field(
        default_factory=list, description="Ordered variable flow transitions from source to sink"
    )
    nonce: Optional[str] = Field(
        None, description="Optional caller-supplied cryptographic nonce for prompt fencing"
    )


# ============================================================================
# CTR-005: AI Remediation Response (Output from AI Microservice)
# ============================================================================
# Replace the RemediationResponse definition in infra-ai/models/schemas.py with this:

class RemediationResponse(BaseModel):
    """
    Contract CTR-005: Returned by POST /remediate to Platform Worker.
    Contains the structured, validated remediation patch and security rationale.
    """
    model_config = ConfigDict(extra="forbid", protected_namespaces=())

    scan_id: str = Field(..., min_length=1, description="Correlated scan run UUID")
    vulnerability_id: str = Field(..., min_length=1, description="Correlated vulnerability UUID")
    suggested_patch: str = Field(
        ..., min_length=1, description="Exact replacement code snippet or unified diff"
    )
    explanation: str = Field(
        ..., min_length=1, description="Concise, 2-3 sentence technical explanation of why the code is vulnerable"
    )
    security_rationale: str = Field(
        ..., min_length=1, description="Technical rationale of how the fix neutralizes the identified CWE"
    )
    confidence: float = Field(
        ..., ge=0.0, le=1.0, description="Model self-reported confidence score (0.0 to 1.0)"
    )
    model_name: str = Field(..., min_length=1, description="Name of the model used (e.g. llama-3.1-8b-instant)")
    tokens_used: int = Field(..., ge=0, description="Total tokens consumed by prompt and completion")
    inference_latency_ms: int = Field(..., ge=0, description="Inference execution duration in milliseconds")


# ============================================================================
# CTR-006: Sandbox Detonation Request (Input to Dynamic Sandbox)
# ============================================================================
class SandboxRequest(BaseModel):
    """
    Contract CTR-006: Received by POST /sandbox/detonate from Platform Worker.
    Requests an isolated, network-less execution run inside a gVisor container.
    """
    model_config = ConfigDict(extra="forbid")

    scan_id: str = Field(..., min_length=1, description="Unique UUID of the scan run")
    repository_url: Optional[str] = Field(None, description="GitHub clone URL for live PR builds")
    commit_sha: Optional[str] = Field(None, description="Target commit SHA to checkout on the host")
    files: Optional[Dict[str, str]] = Field(
        None, description="In-memory file map {path: content} for fast synthetic testing"
    )
    patch_content: Optional[str] = Field(None, description="Optional patch diff to apply before running tests")
    language: Literal["javascript", "typescript", "python", "go"] = Field(
        ..., description="Primary runtime environment for the container"
    )
    build_command: str = Field(
        default="npm test", min_length=1, description="Command to execute inside the sandbox workspace"
    )
    timeout_seconds: int = Field(
        default=60, ge=5, le=120, description="Hard timeout in seconds (default: 60s, max: 120s)"
    )


# ============================================================================
# CTR-007: Sandbox Detonation Result (Output from Dynamic Sandbox)
# ============================================================================
class SandboxResult(BaseModel):
    """
    Contract CTR-007: Returned by POST /sandbox/detonate to Platform Worker.
    Captures process exit code and output after execution under --network none.
    """
    model_config = ConfigDict(extra="forbid")

    scan_id: str = Field(..., min_length=1, description="Correlated scan run UUID")
    status: Literal["PASSED", "FAILED", "TIMED_OUT", "ESCAPE_DETECTED"] = Field(
        ..., description="Execution outcome classification"
    )
    exit_code: int = Field(..., description="Process exit code (0 for success, 137 for watchdog kill)")
    duration_ms: int = Field(..., ge=0, description="Total execution duration in milliseconds")
    stdout: str = Field(default="", description="Captured standard output (truncated to max 10KB)")
    stderr: str = Field(default="", description="Captured standard error (truncated to max 10KB)")
    network_egress_attempts: int = Field(
        default=0, ge=0, description="Count of blocked outbound network socket attempts"
    )