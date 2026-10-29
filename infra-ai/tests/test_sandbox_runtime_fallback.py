"""
Unit tests for Dynamic Sandbox Detonation Orchestrator Runtime Fallback.
Tests OCI runtime start failures, automated runc fallback, container cleanup,
and preservation of non-runtime errors without requiring a live Docker daemon.
"""

from unittest.mock import MagicMock, patch
import pytest
from docker.errors import APIError
from requests.models import Response

from models.schemas import SandboxRequest
from sandbox.orchestrator import SandboxOrchestrator
from service.config import settings


def _make_api_error(status_code: int, message: str) -> APIError:
    """Helper to construct a realistic Docker APIError."""
    response = Response()
    response.status_code = status_code
    return APIError(message, response=response, explanation=message)


@pytest.fixture(autouse=True)
def isolated_sandbox_temp_dir(tmp_path, monkeypatch):
    """Ensure test uses a test-scoped temporary directory owned by the test runner."""
    monkeypatch.setattr(settings, "SANDBOX_BASE_TEMP_DIR", str(tmp_path))
    yield tmp_path


@pytest.fixture
def mock_orchestrator():
    orchestrator = SandboxOrchestrator()
    return orchestrator


@pytest.mark.asyncio
async def test_orchestrator_falls_back_on_start_runtime_error(mock_orchestrator, monkeypatch):
    """
    When settings.SANDBOX_RUNTIME is set (e.g. 'runsc') and container.start() fails
    with an OCI runtime setup error, orchestrator should:
    1. Force-remove the failed container.
    2. Log a fallback warning.
    3. Re-create and start a fallback container without runtime='runsc'.
    4. Complete the detonation with status='PASSED'.
    """
    monkeypatch.setattr(settings, "SANDBOX_RUNTIME", "runsc")
    monkeypatch.setattr(settings, "SANDBOX_ALLOW_RUNTIME_FALLBACK", True)

    req = SandboxRequest(
        scan_id="test-start-fallback",
        language="python",
        build_command="python3 -c 'print(\"FALLBACK_SUCCESS\")'",
        timeout_seconds=10,
        files={},
    )

    # First container fails on .start()
    mock_failed_container = MagicMock()
    mock_failed_container.start.side_effect = _make_api_error(
        500,
        "OCI runtime start failed: starting container: setting up network: "
        "creating interfaces from net namespace \"/proc/123/ns/net\": "
        "cannot run with network enabled in root network namespace: unknown"
    )

    # Second container (fallback) succeeds on .start() and .wait()
    mock_fallback_container = MagicMock()
    mock_fallback_container.start.return_value = None
    mock_fallback_container.wait.return_value = {"StatusCode": 0}
    mock_fallback_container.logs.side_effect = lambda stdout=True, stderr=True: (
        b"FALLBACK_SUCCESS\n" if stdout else b""
    )

    mock_client = MagicMock()
    mock_client.containers.create.side_effect = [
        mock_failed_container,
        mock_fallback_container,
    ]

    with patch.object(mock_orchestrator, "_get_client", return_value=mock_client):
        result = await mock_orchestrator.detonate(req)

    # Verify first container was removed
    mock_failed_container.remove.assert_called_once_with(force=True)

    # Verify containers.create was called twice
    assert mock_client.containers.create.call_count == 2
    first_call_kwargs = mock_client.containers.create.call_args_list[0].kwargs
    second_call_kwargs = mock_client.containers.create.call_args_list[1].kwargs

    # First call had runtime='runsc'
    assert first_call_kwargs.get("runtime") == "runsc"
    # Second call stripped runtime (fallback to standard runc)
    assert "runtime" not in second_call_kwargs or second_call_kwargs.get("runtime") is None

    # Detonation succeeded via fallback with actual runtime reported as runc
    assert result.status == "PASSED"
    assert result.exit_code == 0
    assert result.runtime == "runc"
    assert "FALLBACK_SUCCESS" in result.stdout


@pytest.mark.asyncio
async def test_orchestrator_raises_in_strict_mode_when_runtime_fails(mock_orchestrator, monkeypatch):
    """
    When settings.SANDBOX_ALLOW_RUNTIME_FALLBACK is False, an OCI runtime start error
    must fail immediately without falling back to runc, returning status='FAILED'
    and reporting the requested runtime.
    """
    monkeypatch.setattr(settings, "SANDBOX_RUNTIME", "runsc")
    monkeypatch.setattr(settings, "SANDBOX_ALLOW_RUNTIME_FALLBACK", False)

    req = SandboxRequest(
        scan_id="test-strict-runtime-failure",
        language="python",
        build_command="python3 -c 'print(\"STRICT_FAIL\")'",
        timeout_seconds=10,
        files={},
    )

    mock_failed_container = MagicMock()
    mock_failed_container.start.side_effect = _make_api_error(
        500,
        "OCI runtime start failed: starting container: setting up network: "
        "root network namespace: unknown runtime"
    )

    mock_client = MagicMock()
    mock_client.containers.create.return_value = mock_failed_container

    with patch.object(mock_orchestrator, "_get_client", return_value=mock_client):
        result = await mock_orchestrator.detonate(req)

    # Failed container must be removed
    mock_failed_container.remove.assert_called_once_with(force=True)

    # Must only call create once (zero fallback attempts)
    assert mock_client.containers.create.call_count == 1
    assert mock_client.containers.create.call_args.kwargs.get("runtime") == "runsc"

    # Outcome must be FAILED, not PASSED
    assert result.status == "FAILED"
    assert result.exit_code == 1
    assert result.runtime == "runsc"
    assert "Strict runtime requirement failed" in result.stderr
    assert "runsc" in result.stderr


@pytest.mark.asyncio
async def test_orchestrator_strict_mode_create_runtime_error(mock_orchestrator, monkeypatch):
    """
    When settings.SANDBOX_ALLOW_RUNTIME_FALLBACK is False and container.create()
    fails with unknown runtime error, it must fail immediately without fallback.
    """
    monkeypatch.setattr(settings, "SANDBOX_RUNTIME", "runsc")
    monkeypatch.setattr(settings, "SANDBOX_ALLOW_RUNTIME_FALLBACK", False)

    req = SandboxRequest(
        scan_id="test-strict-create-failure",
        language="python",
        build_command="python3 -c 'print(\"STRICT_CREATE_FAIL\")'",
        timeout_seconds=10,
        files={},
    )

    mock_client = MagicMock()
    mock_client.containers.create.side_effect = _make_api_error(
        500, "Unknown runtime specified: runsc"
    )

    with patch.object(mock_orchestrator, "_get_client", return_value=mock_client):
        result = await mock_orchestrator.detonate(req)

    assert mock_client.containers.create.call_count == 1
    assert result.status == "FAILED"
    assert result.exit_code == 1
    assert result.runtime == "runsc"
    assert "Strict runtime requirement failed" in result.stderr


@pytest.mark.asyncio
async def test_orchestrator_does_not_fall_back_on_non_runtime_start_error(mock_orchestrator, monkeypatch):
    """
    When container.start() fails with a non-runtime error (e.g. disk space),
    orchestrator must NOT fall back, preserving genuine failure.
    """
    monkeypatch.setattr(settings, "SANDBOX_RUNTIME", "runsc")

    req = SandboxRequest(
        scan_id="test-no-fallback",
        language="python",
        build_command="python3 -c 'print(\"FAIL\")'",
        timeout_seconds=10,
        files={},
    )

    mock_container = MagicMock()
    mock_container.start.side_effect = _make_api_error(500, "Internal Server Error: no space left on device")

    mock_client = MagicMock()
    mock_client.containers.create.return_value = mock_container

    with patch.object(mock_orchestrator, "_get_client", return_value=mock_client):
        result = await mock_orchestrator.detonate(req)

    # Only one container create attempt
    assert mock_client.containers.create.call_count == 1
    assert result.status == "FAILED"
    assert "no space left on device" in result.stderr


@pytest.mark.asyncio
async def test_orchestrator_normal_runsc_succeeds_without_fallback(mock_orchestrator, monkeypatch):
    """
    When settings.SANDBOX_RUNTIME is 'runsc' and start() succeeds (normal path),
    it runs with runtime='runsc' and does not invoke fallback.
    """
    monkeypatch.setattr(settings, "SANDBOX_RUNTIME", "runsc")

    req = SandboxRequest(
        scan_id="test-normal-runsc",
        language="python",
        build_command="python3 -c 'print(\"RUNSC_OK\")'",
        timeout_seconds=10,
        files={},
    )

    mock_container = MagicMock()
    mock_container.start.return_value = None
    mock_container.wait.return_value = {"StatusCode": 0}
    mock_container.logs.side_effect = lambda stdout=True, stderr=True: (
        b"RUNSC_OK\n" if stdout else b""
    )

    mock_client = MagicMock()
    mock_client.containers.create.return_value = mock_container

    with patch.object(mock_orchestrator, "_get_client", return_value=mock_client):
        result = await mock_orchestrator.detonate(req)

    assert mock_client.containers.create.call_count == 1
    call_kwargs = mock_client.containers.create.call_args.kwargs
    assert call_kwargs.get("runtime") == "runsc"
    assert result.status == "PASSED"
    assert result.exit_code == 0
    assert result.runtime == "runsc"
    assert "RUNSC_OK" in result.stdout


@pytest.mark.asyncio
async def test_orchestrator_handles_early_preparation_failure_without_unbound_local_error(mock_orchestrator, monkeypatch):
    """
    If workspace preparation or source prep raises an exception before container creation,
    the outer error handler must return a valid SandboxResult without crashing on UnboundLocalError.
    """
    monkeypatch.setattr(settings, "SANDBOX_RUNTIME", "runsc")

    req = SandboxRequest(
        scan_id="test-early-prep-failure",
        language="python",
        build_command="python3 -c 'print(1)'",
        timeout_seconds=10,
        files={},
    )

    mock_client = MagicMock()
    with patch.object(mock_orchestrator, "_get_client", return_value=mock_client), \
         patch.object(mock_orchestrator, "_prepare_source_code", side_effect=OSError("Disk write error during source prep")):
        result = await mock_orchestrator.detonate(req)

    assert result.status == "FAILED"
    assert result.exit_code == 1
    assert result.runtime == "runsc"
    assert "Disk write error during source prep" in result.stderr


@pytest.mark.asyncio
async def test_orchestrator_detects_containerd_not_found_runtime_error(mock_orchestrator, monkeypatch):
    """
    When containerd reports that a runtime was not found / not installed,
    orchestrator must detect it as a runtime error and respect strict mode.
    """
    monkeypatch.setattr(settings, "SANDBOX_RUNTIME", "runsc")
    monkeypatch.setattr(settings, "SANDBOX_ALLOW_RUNTIME_FALLBACK", False)

    req = SandboxRequest(
        scan_id="test-containerd-not-found",
        language="python",
        build_command="python3 -c 'print(1)'",
        timeout_seconds=10,
        files={},
    )

    mock_client = MagicMock()
    mock_client.containers.create.side_effect = _make_api_error(
        500, "failed to create containerd task: runtime 'runsc' not found: unknown"
    )

    with patch.object(mock_orchestrator, "_get_client", return_value=mock_client):
        result = await mock_orchestrator.detonate(req)

    assert mock_client.containers.create.call_count == 1
    assert result.status == "FAILED"
    assert result.exit_code == 1
    assert result.runtime == "runsc"
    assert "Strict runtime requirement failed" in result.stderr

