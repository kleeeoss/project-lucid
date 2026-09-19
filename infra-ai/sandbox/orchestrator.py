"""
LUCID-CI — Dynamic Sandbox Detonation Orchestrator.
Manages ephemeral workspaces, execution containment, timeouts, and guaranteed cleanup.
"""

import asyncio
import os
import re
import shutil
import subprocess
import time
import uuid
from pathlib import Path
from typing import Optional, Tuple
import docker
from docker.errors import DockerException, APIError
import structlog

from models.schemas import SandboxRequest, SandboxResult
from service.config import settings

logger = structlog.get_logger()


class SandboxExecutionError(Exception):
    """Raised when container orchestration fails critically."""
    pass


class SandboxOrchestrator:
    """Orchestrates ephemeral Docker sandbox containers under strict isolation."""

    def __init__(self, docker_client: Optional[docker.DockerClient] = None):
        self._client = docker_client

    def _get_client(self) -> docker.DockerClient:
        if not self._client:
            try:
                self._client = docker.from_env()
            except DockerException as err:
                logger.error("docker_daemon_unavailable", error=str(err))
                raise SandboxExecutionError(f"Docker daemon unavailable: {err}") from err
        return self._client

    async def detonate(self, request: SandboxRequest) -> SandboxResult:
        """Non-blocking entry point executing detonation in a worker thread."""
        return await asyncio.to_thread(self._detonate_sync, request)

    def _detonate_sync(self, request: SandboxRequest) -> SandboxResult:
        """Synchronous container detonation lifecycle."""
        client = self._get_client()
        session_id = uuid.uuid4().hex[:8]
        host_root = Path(settings.SANDBOX_BASE_TEMP_DIR) / f"scan-{request.scan_id}-{session_id}"
        workspace_dir = host_root / "workspace"

        container = None
        start_time = time.perf_counter()

        try:
            # 1. Prepare host workspace
            workspace_dir.mkdir(parents=True, exist_ok=True)
            # Make workspace writable by sandboxuser (UID 10001)
            os.chmod(host_root, 0o777)
            os.chmod(workspace_dir, 0o777)

            self._prepare_source_code(request, workspace_dir)

            # 2. Configure container security parameters
            container_name = f"lucid-sb-{request.scan_id[:8]}-{session_id}"

            # Convert CPU cap to nano-cpus for Docker SDK
            nano_cpus = int(settings.SANDBOX_CPU_LIMIT * 1_000_000_000)

            logger.info(
                "launching_sandbox_container",
                scan_id=request.scan_id,
                container_name=container_name,
                command=request.build_command,
                timeout=request.timeout_seconds,
            )

            # Command executes via entrypoint.sh in /workspace
            cmd_args = ["/bin/bash", "-c", request.build_command]

            container = client.containers.create(
                image=settings.SANDBOX_IMAGE_TAG,
                name=container_name,
                command=cmd_args,
                volumes={str(workspace_dir): {"bind": "/workspace", "mode": "rw"}},
                network_mode="none",
                mem_limit=settings.SANDBOX_MEMORY_LIMIT,
                nano_cpus=nano_cpus,
                pids_limit=settings.SANDBOX_PID_LIMIT,
                read_only=True,
                tmpfs={"/tmp": "rw,noexec,nosuid,size=64m"},
                user="10001:10001",
            )

            container.start()

            # 3. Wait for process completion under watchdog timer
            status = "PASSED"
            exit_code = 0
            timed_out = False

            try:
                result_payload = container.wait(timeout=request.timeout_seconds)
                exit_code = result_payload.get("StatusCode", 0)
                status = "PASSED" if exit_code == 0 else "FAILED"
            except (APIError, Exception) as wait_err:
                # Watchdog timer fired or container hanging
                timed_out = True
                status = "TIMED_OUT"
                exit_code = 137
                logger.warning(
                    "sandbox_watchdog_killed_container",
                    scan_id=request.scan_id,
                    timeout=request.timeout_seconds,
                    error=str(wait_err),
                )
                try:
                    container.kill()
                except Exception:
                    pass

            # 4. Collect logs (bounded to max 10KB)
            stdout, stderr = self._collect_logs(container)
            duration_ms = int((time.perf_counter() - start_time) * 1000)

            # 5. Detect network egress attempts from log traces
            egress_attempts = self._detect_egress_attempts(stdout, stderr)

            return SandboxResult(
                scan_id=request.scan_id,
                status=status,
                exit_code=exit_code,
                duration_ms=duration_ms,
                stdout=stdout,
                stderr=stderr,
                network_egress_attempts=egress_attempts,
            )

        except Exception as err:
            logger.error("sandbox_execution_failed", scan_id=request.scan_id, error=str(err))
            duration_ms = int((time.perf_counter() - start_time) * 1000)
            return SandboxResult(
                scan_id=request.scan_id,
                status="FAILED",
                exit_code=1,
                duration_ms=duration_ms,
                stdout="",
                stderr=f"Sandbox Orchestrator Failure: {err}",
                network_egress_attempts=0,
            )

        finally:
            # 6. Guaranteed Cleanup
            if container:
                try:
                    container.remove(force=True)
                except Exception as clean_err:
                    logger.warning("container_remove_failed", error=str(clean_err))

            if host_root.exists():
                try:
                    shutil.rmtree(host_root, ignore_errors=True)
                except Exception as rmtree_err:
                    logger.warning("workspace_cleanup_failed", error=str(rmtree_err))

    def _prepare_source_code(self, request: SandboxRequest, workspace_dir: Path):
        """Prepares code files inside workspace on the host."""
        # Ingress Mode A: In-memory file map (synthetic tests / fast runs)
        if request.files:
            for file_path, content in request.files.items():
                dest = workspace_dir / file_path
                dest.parent.mkdir(parents=True, exist_ok=True)
                dest.write_text(content, encoding="utf-8")

        # Ingress Mode B: Shallow Git clone on the host
        elif request.repository_url:
            clone_cmd = ["git", "clone", "--depth=1", request.repository_url, str(workspace_dir)]
            subprocess.run(clone_cmd, check=True, capture_output=True, timeout=30)

            if request.commit_sha:
                checkout_cmd = ["git", "-C", str(workspace_dir), "checkout", request.commit_sha]
                subprocess.run(checkout_cmd, check=True, capture_output=True, timeout=10)

            # Scrub any git credentials from .git/config
            git_config = workspace_dir / ".git" / "config"
            if git_config.exists():
                git_config.unlink()

            # Apply patch diff if provided
            if request.patch_content:
                patch_file = workspace_dir / ".lucid_remediation.patch"
                patch_file.write_text(request.patch_content, encoding="utf-8")
                patch_cmd = ["git", "-C", str(workspace_dir), "apply", str(patch_file)]
                subprocess.run(patch_cmd, check=False, capture_output=True, timeout=10)
                patch_file.unlink(missing_ok=True)

        # Ensure all created files are accessible to UID 10001
        for root, dirs, files in os.walk(workspace_dir):
            for d in dirs:
                os.chmod(os.path.join(root, d), 0o777)
            for f in files:
                os.chmod(os.path.join(root, f), 0o666)

    def _collect_logs(self, container) -> Tuple[str, str]:
        """Collects and truncates stdout and stderr from container."""
        try:
            raw_stdout = container.logs(stdout=True, stderr=False).decode("utf-8", errors="replace")
            raw_stderr = container.logs(stdout=False, stderr=True).decode("utf-8", errors="replace")
            # Truncate to max 10KB to avoid excessive payload size
            return raw_stdout[:10240], raw_stderr[:10240]
        except Exception:
            return "", ""

    def _detect_egress_attempts(self, stdout: str, stderr: str) -> int:
        """Scans process logs for network socket error signatures."""
        combined = f"{stdout}\n{stderr}"
        egress_patterns = [
            r"Network is unreachable",
            r"Connection refused",
            r"getaddrinfo ENOTFOUND",
            r"Failed to establish a new connection",
            r"Name or service not known",
        ]
        matches = 0
        for pattern in egress_patterns:
            matches += len(re.findall(pattern, combined, re.IGNORECASE))
        return matches
