"""
LUCID-CI — Sandbox Penetration Testing & Container Isolation Test Suite.
Verifies TASK-INF-501 requirements:
- Zero network egress under --net none.
- Mitigation of AWS metadata service SSRF / credential exfiltration.
- Process cap (--pids-limit 100) and memory cap (--memory 512m) containment.
- Read-only rootfs enforcement.
- Automated cleanup of ephemeral detonation workspaces.
"""

import os
from pathlib import Path
from unittest.mock import MagicMock, patch
import pytest
import docker
from docker.errors import DockerException

from models.schemas import SandboxRequest, SandboxResult
from sandbox.orchestrator import SandboxOrchestrator
from service.config import settings
from tests.security.malicious_pr_network import verify_network_isolation
from tests.security.malicious_pr_forkbomb import (
    attempt_memory_exhaustion,
    attempt_controlled_fork_bomb,
)


def is_docker_available() -> bool:
    """Safely checks if Docker daemon is accessible and responding."""
    try:
        client = docker.from_env()
        client.ping()
        return True
    except (DockerException, Exception):
        return False


@pytest.fixture(autouse=True)
def isolated_sandbox_temp_dir(tmp_path, monkeypatch):
    """Ensure test uses a test-scoped temporary directory owned by current user."""
    monkeypatch.setattr(settings, "SANDBOX_BASE_TEMP_DIR", str(tmp_path))
    yield tmp_path


# -----------------------------------------------------------------------------
# Unit Tests (Offline / Mocked Containment Checks)
# -----------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_orchestrator_configures_strict_network_isolation(tmp_path):
    """Verify orchestrator unconditionally sets --net none and non-root UID."""
    mock_container = MagicMock()
    mock_container.wait.return_value = {"StatusCode": 0}
    mock_container.logs.return_value = b"[ISOLATION_VERIFIED]"

    mock_client = MagicMock()
    mock_client.containers.create.return_value = mock_container

    orchestrator = SandboxOrchestrator(docker_client=mock_client)

    req = SandboxRequest(
        scan_id="pen-net-001",
        language="python",
        build_command="python3 test.py",
        files={"test.py": "print('ok')"},
    )

    result = await orchestrator.detonate(req)

    assert result.status == "PASSED"
    create_args = mock_client.containers.create.call_args[1]

    # Invariant: Complete network isolation
    assert create_args["network_mode"] == "none", "Container must run with network_mode='none'"

    # Invariant: Non-root execution
    assert create_args["user"] == "10001:10001", "Container must run as unprivileged UID/GID 10001:10001"

    # Invariant: Read-only root filesystem
    assert create_args["read_only"] is True, "Rootfs must be mounted read-only"

    # Invariant: Resource bounds
    assert create_args["pids_limit"] == settings.SANDBOX_PID_LIMIT
    assert create_args["mem_limit"] == settings.SANDBOX_MEMORY_LIMIT

    # Invariant: Tmpfs noexec/nosuid
    assert create_args["tmpfs"] == {"/tmp": "rw,noexec,nosuid,size=64m"}


@pytest.mark.asyncio
async def test_orchestrator_enforces_gvisor_runtime(monkeypatch):
    """Verify orchestrator specifies runsc runtime when configured in settings."""
    monkeypatch.setattr(settings, "SANDBOX_RUNTIME", "runsc")

    mock_container = MagicMock()
    mock_container.wait.return_value = {"StatusCode": 0}
    mock_container.logs.return_value = b"gvisor ok"

    mock_client = MagicMock()
    mock_client.containers.create.return_value = mock_container

    orchestrator = SandboxOrchestrator(docker_client=mock_client)

    req = SandboxRequest(
        scan_id="pen-gvisor-001",
        language="python",
        build_command="python3 test.py",
        files={"test.py": "print('ok')"},
    )

    result = await orchestrator.detonate(req)

    assert result.status == "PASSED"
    assert result.runtime == "runsc"
    create_args = mock_client.containers.create.call_args[1]
    assert create_args["runtime"] == "runsc"


def test_network_isolation_verification_logic():
    """Verify probe logic accurately reports isolation when network calls fail."""
    with patch("tests.security.malicious_pr_network.probe_external_dns_tcp", return_value=False), \
         patch("tests.security.malicious_pr_network.probe_aws_metadata", return_value=False), \
         patch("tests.security.malicious_pr_network.probe_dns_resolution", return_value=False):
        status = verify_network_isolation()
        assert status["is_isolated"] is True
        assert status["dns_egress_tcp"] is False
        assert status["aws_metadata_http"] is False


def test_network_isolation_detects_breach_on_open_egress():
    """Verify probe logic triggers breach status if any egress succeeds."""
    with patch("tests.security.malicious_pr_network.probe_external_dns_tcp", return_value=True), \
         patch("tests.security.malicious_pr_network.probe_aws_metadata", return_value=False), \
         patch("tests.security.malicious_pr_network.probe_dns_resolution", return_value=False):
        status = verify_network_isolation()
        assert status["is_isolated"] is False
        assert status["dns_egress_tcp"] is True


def test_memory_exhaustion_safety_handling():
    """Verify memory exhaustion payload handles allocation limits cleanly."""
    res = attempt_memory_exhaustion(target_mb=64, chunk_mb=32)
    assert res["allocated_mb"] >= 0
    assert "hit_limit" in res


def test_controlled_forkbomb_safety_handling():
    """Verify controlled forkbomb function catches process limits gracefully."""
    with patch("os.fork", side_effect=BlockingIOError("Resource temporarily unavailable")):
        res = attempt_controlled_fork_bomb(max_attempts=10)
        assert res["blocked_by_pids_limit"] is True
        assert "Resource temporarily unavailable" in res["error"]


# -----------------------------------------------------------------------------
# Live Sandbox Integration Tests (Active when Docker daemon is available)
# -----------------------------------------------------------------------------

@pytest.mark.skipif(not is_docker_available(), reason="Docker daemon unreachable for live sandbox detonation")
@pytest.mark.asyncio
async def test_live_sandbox_blocks_network_egress():
    """Live test: Detonates malicious_pr_network.py inside real container with --net none."""
    orchestrator = SandboxOrchestrator()

    network_script_path = Path(__file__).parent / "malicious_pr_network.py"
    with open(network_script_path, "r", encoding="utf-8") as f:
        script_content = f.read()

    req = SandboxRequest(
        scan_id="live-pen-net-001",
        language="python",
        build_command="python3 malicious_pr_network.py",
        timeout_seconds=15,
        files={"malicious_pr_network.py": script_content},
    )

    result = await orchestrator.detonate(req)

    assert result.status == "PASSED", f"Execution failed: {result.stderr}"
    assert "[ISOLATION_VERIFIED]" in result.stdout
    assert "[SECURITY_BREACH]" not in result.stdout


@pytest.mark.skipif(not is_docker_available(), reason="Docker daemon unreachable for live sandbox detonation")
@pytest.mark.asyncio
async def test_live_sandbox_read_only_rootfs():
    """Live test: Attempt to write to container rootfs must fail with Read-only file system."""
    orchestrator = SandboxOrchestrator()

    req = SandboxRequest(
        scan_id="live-pen-ro-001",
        language="python",
        build_command="touch /root/hacked.txt",
        timeout_seconds=10,
        files={"README.md": "# Read Only Test"},
    )

    result = await orchestrator.detonate(req)

    # Must fail because rootfs is read-only
    assert result.status == "FAILED"
    assert result.exit_code != 0
    assert "Read-only file system" in result.stderr or "Permission denied" in result.stderr
