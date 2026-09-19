"""
Integration tests for Dynamic Sandbox Detonation Orchestrator.
Verifies passing execution, failures, 60s watchdog timeouts, and guaranteed cleanup.
"""

import os
from pathlib import Path
import pytest
from models.schemas import SandboxRequest
from sandbox.orchestrator import SandboxOrchestrator
from service.config import settings

orchestrator = SandboxOrchestrator()


@pytest.mark.asyncio
async def test_orchestrator_executes_passing_code():
    req = SandboxRequest(
        scan_id="test-pass-scan",
        language="python",
        build_command="python3 test_app.py",
        timeout_seconds=15,
        files={
            "test_app.py": "print('All unit tests passed successfully!'); exit(0)",
        },
    )

    result = await orchestrator.detonate(req)

    assert result.scan_id == "test-pass-scan"
    assert result.status == "PASSED"
    assert result.exit_code == 0
    assert "All unit tests passed successfully!" in result.stdout
    assert result.duration_ms > 0


@pytest.mark.asyncio
async def test_orchestrator_captures_test_failure():
    req = SandboxRequest(
        scan_id="test-fail-scan",
        language="python",
        build_command="python3 test_fail.py",
        timeout_seconds=15,
        files={
            "test_fail.py": "import sys; sys.stderr.write('AssertionError: expected True got False\\n'); exit(1)",
        },
    )

    result = await orchestrator.detonate(req)

    assert result.scan_id == "test-fail-scan"
    assert result.status == "FAILED"
    assert result.exit_code == 1
    assert "AssertionError" in result.stderr


@pytest.mark.asyncio
async def test_orchestrator_watchdog_kills_infinite_loop():
    """Verify that a hanging process is killed and returns TIMED_OUT (exit 137)."""
    req = SandboxRequest(
        scan_id="test-timeout-scan",
        language="python",
        build_command="python3 -c 'import time; time.sleep(15)'",
        timeout_seconds=5,  # Short timeout for test
        files={"dummy.py": "print('start')"},
    )

    result = await orchestrator.detonate(req)

    assert result.scan_id == "test-timeout-scan"
    assert result.status == "TIMED_OUT"
    assert result.exit_code == 137
    assert result.duration_ms >= 4500


@pytest.mark.asyncio
async def test_orchestrator_detects_network_egress_attempts():
    """Verify code attempting to reach external network triggers egress count."""
    req = SandboxRequest(
        scan_id="test-egress-scan",
        language="python",
        build_command="python3 exfil.py",
        timeout_seconds=10,
        files={
            "exfil.py": (
                "import urllib.request\n"
                "try:\n"
                "    urllib.request.urlopen('http://169.254.169.254', timeout=2)\n"
                "except Exception as e:\n"
                "    print(f'Network error: {e}')\n"
            ),
        },
    )

    result = await orchestrator.detonate(req)

    assert result.status == "PASSED"  # Script caught exception and exited 0
    assert result.network_egress_attempts >= 1
    assert "Network is unreachable" in result.stdout or "Network error" in result.stdout


@pytest.mark.asyncio
async def test_orchestrator_guarantees_cleanup():
    """Verify that temporary host directories are deleted post-detonation."""
    scan_id = "test-cleanup-scan"
    req = SandboxRequest(
        scan_id=scan_id,
        language="python",
        build_command="python3 -c 'print(1)'",
        timeout_seconds=10,
        files={"test.py": "print(1)"},
    )

    await orchestrator.detonate(req)

    # Check base temp dir: no directory starting with scan-test-cleanup-scan should remain
    base_dir = Path(settings.SANDBOX_BASE_TEMP_DIR)
    leaked = list(base_dir.glob(f"scan-{scan_id}-*"))
    assert len(leaked) == 0
