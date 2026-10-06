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

    # Detonation succeeded via fallback
    assert result.status == "PASSED"
    assert result.exit_code == 0
    assert "FALLBACK_SUCCESS" in result.stdout


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
    assert "RUNSC_OK" in result.stdout
