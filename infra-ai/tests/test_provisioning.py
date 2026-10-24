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


def test_userdata_gcp_config_dir_created_and_secured():
    """Verify that userdata.sh creates /var/lib/lucid/config with chmod 700 before Docker starts."""
    repo_root = Path(__file__).resolve().parent.parent.parent
    userdata_path = repo_root / "infra-ai" / "terraform" / "modules" / "ec2" / "userdata.sh"
    content = userdata_path.read_text(encoding="utf-8")

    assert "mkdir -p /var/lib/lucid/config" in content, (
        "userdata.sh must create /var/lib/lucid/config directory on host"
    )
    assert "chmod 700 /var/lib/lucid/config" in content, (
        "userdata.sh must restrict permissions on /var/lib/lucid/config to 700"
    )


def test_docker_compose_prod_gcp_mount_uses_directory():
    """Verify that docker-compose.prod.yml mounts the GCP config directory rather than a missing file."""
    repo_root = Path(__file__).resolve().parent.parent.parent
    compose_path = repo_root / "docker-compose.prod.yml"
    assert compose_path.exists(), f"docker-compose.prod.yml not found at {compose_path}"
    content = compose_path.read_text(encoding="utf-8")

    # Verify directory mount is used
    assert "${GCP_CONFIG_DIR:-/var/lib/lucid/config}:/etc/lucid:ro" in content, (
        "docker-compose.prod.yml must mount host config directory to /etc/lucid:ro to prevent Docker creating missing files as directories"
    )
    # Ensure legacy file bind-mount is absent
    assert ":/etc/lucid/gcp-credentials.json:ro" not in content, (
        "docker-compose.prod.yml must NOT directly bind-mount gcp-credentials.json file"
    )
    # Verify GOOGLE_APPLICATION_CREDENTIALS environment variable
    assert "GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS:-/etc/lucid/gcp-credentials.json}" in content, (
        "docker-compose.prod.yml must set GOOGLE_APPLICATION_CREDENTIALS inside container"
    )


def test_userdata_git_revision_selection_defaults_to_main():
    """Verify that userdata.sh defaults to main branch and supports pinned commit SHA checkout."""
    repo_root = Path(__file__).resolve().parent.parent.parent
    userdata_path = repo_root / "infra-ai" / "terraform" / "modules" / "ec2" / "userdata.sh"
    content = userdata_path.read_text(encoding="utf-8")

    # Invariant: Must default to main
    assert 'GIT_REVISION="${GIT_REVISION:-${RAW_REVISION:-main}}"' in content or 'GIT_REVISION="${GIT_REVISION:-main}"' in content, (
        "userdata.sh must default GIT_REVISION to main"
    )
    assert 'git -C /opt/lucid-ci checkout "$GIT_REVISION"' in content, (
        "userdata.sh must explicitly check out $GIT_REVISION"
    )
    # Never default or hardcode garv in userdata.sh
    assert 'GIT_REVISION="garv"' not in content, (
        "userdata.sh must not hardcode garv branch as deployment target"
    )


def test_userdata_sqs_queue_url_configuration():
    """Verify that userdata.sh discovers the real AWS SQS queue URL and clears LocalStack endpoint."""
    repo_root = Path(__file__).resolve().parent.parent.parent
    userdata_path = repo_root / "infra-ai" / "terraform" / "modules" / "ec2" / "userdata.sh"
    content = userdata_path.read_text(encoding="utf-8")

    # Metadata discovery
    assert "tags/instance/SqsQueueUrl" in content, (
        "userdata.sh must query IMDSv2 for SqsQueueUrl tag"
    )
    assert "dynamic/instance-identity/document" in content, (
        "userdata.sh must inspect instance identity document for region and account ID"
    )

    # .env substitution logic
    assert 'sed -i "s|^SQS_QUEUE_URL=.*|SQS_QUEUE_URL=${REAL_SQS_URL}|" /opt/lucid-ci/.env' in content, (
        "userdata.sh must replace SQS_QUEUE_URL in .env with real SQS URL"
    )
    assert 'sed -i "s|^SQS_ENDPOINT_URL=.*|SQS_ENDPOINT_URL=|" /opt/lucid-ci/.env' in content, (
        "userdata.sh must clear SQS_ENDPOINT_URL so AWS SDK routes to real SQS instead of LocalStack"
    )


def test_terraform_ec2_wiring_and_tags():
    """Verify that Terraform passes sqs_queue_url and git_revision to EC2 instance tags."""
    repo_root = Path(__file__).resolve().parent.parent.parent
    ec2_main_path = repo_root / "infra-ai" / "terraform" / "modules" / "ec2" / "main.tf"
    content = ec2_main_path.read_text(encoding="utf-8")

    assert "SqsQueueUrl        = var.sqs_queue_url" in content, (
        "EC2 instance tags must include SqsQueueUrl"
    )
    assert "GitRevision        = var.git_revision" in content, (
        "EC2 instance tags must include GitRevision"
    )


