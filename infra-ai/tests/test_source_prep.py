"""
Unit tests for SandboxOrchestrator._prepare_source_code.
Verifies in-memory file preparation, patch diff application, and permission setting without Docker daemon.
"""

import os
from pathlib import Path
import pytest
from models.schemas import SandboxRequest
from sandbox.orchestrator import SandboxOrchestrator


def test_prepare_source_code_in_memory_files_with_patch(tmp_path):
    orchestrator = SandboxOrchestrator(docker_client=None)
    workspace_dir = tmp_path / "workspace"
    workspace_dir.mkdir(parents=True, exist_ok=True)

    initial_content = "function test() {\n  const x = 1;\n  return x;\n}\n"
    diff_patch = (
        "--- a/app.js\n"
        "+++ b/app.js\n"
        "@@ -1,4 +1,4 @@\n"
        " function test() {\n"
        "-  const x = 1;\n"
        "+  const x = 2;\n"
        "   return x;\n"
        " }\n"
    )

    req = SandboxRequest(
        scan_id="test-prep-scan",
        language="javascript",
        files={"app.js": initial_content},
        patch_content=diff_patch,
        build_command="node app.js",
    )

    orchestrator._prepare_source_code(req, workspace_dir)

    target_file = workspace_dir / "app.js"
    assert target_file.exists()
    content = target_file.read_text(encoding="utf-8")
    # Must contain the patched line "const x = 2;"
    assert "const x = 2;" in content
    assert "const x = 1;" not in content


def test_prepare_source_code_preserves_pre_spliced_files(tmp_path):
    orchestrator = SandboxOrchestrator(docker_client=None)
    workspace_dir = tmp_path / "workspace"
    workspace_dir.mkdir(parents=True, exist_ok=True)

    spliced_content = "const safe = true;\nconsole.log(safe);\n"
    req = SandboxRequest(
        scan_id="test-prespliced-scan",
        language="javascript",
        files={"index.js": spliced_content},
        patch_content="const safe = true;",
        build_command="node index.js",
    )

    orchestrator._prepare_source_code(req, workspace_dir)

    target_file = workspace_dir / "index.js"
    assert target_file.exists()
    assert target_file.read_text(encoding="utf-8") == spliced_content
