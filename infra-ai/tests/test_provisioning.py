import os
import subprocess
from pathlib import Path


def test_userdata_bash_syntax():
    """Verify userdata.sh contains valid bash syntax with no parsing errors."""
    repo_root = Path(__file__).resolve().parent.parent.parent
    userdata_path = repo_root / "infra-ai" / "terraform" / "modules" / "ec2" / "userdata.sh"
    assert userdata_path.exists(), f"userdata.sh not found at {userdata_path}"

    result = subprocess.run(
        ["bash", "-n", str(userdata_path)],
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0, f"Bash syntax check failed: {result.stderr}"


def test_userdata_autostop_service_enabled_and_started():
    """Verify that lucid-autostop.service is both enabled and actively started."""
    repo_root = Path(__file__).resolve().parent.parent.parent
    userdata_path = repo_root / "infra-ai" / "terraform" / "modules" / "ec2" / "userdata.sh"
    content = userdata_path.read_text(encoding="utf-8")

    assert "systemctl enable lucid-autostop.service" in content, (
        "userdata.sh must enable lucid-autostop.service for reboot persistence"
    )
    assert "systemctl start lucid-autostop.service" in content, (
        "userdata.sh must explicitly start lucid-autostop.service on initial provisioning"
    )


def test_userdata_lucid_service_enabled_and_started():
    """Verify that lucid.service is both enabled and actively started."""
    repo_root = Path(__file__).resolve().parent.parent.parent
    userdata_path = repo_root / "infra-ai" / "terraform" / "modules" / "ec2" / "userdata.sh"
    content = userdata_path.read_text(encoding="utf-8")

    assert "systemctl enable lucid.service" in content, (
        "userdata.sh must enable lucid.service for reboot persistence"
    )
    assert "systemctl start --no-block lucid.service" in content, (
        "userdata.sh must start lucid.service on initial provisioning"
    )


def test_userdata_autostop_imdsv2_and_bounds():
    """Verify that lucid-autostop.sh uses IMDSv2 token and bounds runtime to [5, 480]."""
    repo_root = Path(__file__).resolve().parent.parent.parent
    userdata_path = repo_root / "infra-ai" / "terraform" / "modules" / "ec2" / "userdata.sh"
    content = userdata_path.read_text(encoding="utf-8")

    # IMDSv2 token call
    assert "X-aws-ec2-metadata-token-ttl-seconds" in content
    assert "latest/meta-data/tags/instance/DemoRuntimeMinutes" in content
    # Bounds check
    assert "-ge 5" in content
    assert "-le 480" in content
    # Shutdown call
    assert 'shutdown -h "+${RUNTIME_MINUTES}"' in content


def test_userdata_runsc_network_none_runtime_args():
    """Verify that userdata.sh registers runsc with runtimeArgs: ['--network=none', '--platform=ptrace']."""
    repo_root = Path(__file__).resolve().parent.parent.parent
    userdata_path = repo_root / "infra-ai" / "terraform" / "modules" / "ec2" / "userdata.sh"
    content = userdata_path.read_text(encoding="utf-8")

    assert '"runtimeArgs"' in content and '"--network=none"' in content and '"--platform=ptrace"' in content, (
        "userdata.sh must configure runsc with runtimeArgs: ['--network=none', '--platform=ptrace'] to prevent root netns errors and kernel hangs"
    )

