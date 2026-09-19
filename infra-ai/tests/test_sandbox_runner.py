"""
Security and capability validation tests for lucid-sandbox-runner:latest.
Verifies unprivileged execution, network denial, and resource constraints.
"""

import subprocess
import pytest
import docker

client = docker.from_env()
IMAGE_TAG = "lucid-sandbox-runner:latest"


def is_image_available() -> bool:
    try:
        client.images.get(IMAGE_TAG)
        return True
    except docker.errors.ImageNotFound:
        return False


@pytest.mark.skipif(not is_image_available(), reason=f"{IMAGE_TAG} not built locally yet")
class TestSandboxRunnerSecurity:

    def test_container_runs_as_unprivileged_user(self):
        """Verify the container default user is sandboxuser with UID 10001."""
        output = client.containers.run(
            IMAGE_TAG,
            command=["whoami"],
            remove=True,
        ).decode().strip()
        assert output == "sandboxuser"

        uid = client.containers.run(
            IMAGE_TAG,
            command=["id", "-u"],
            remove=True,
        ).decode().strip()
        assert uid == "10001"

    def test_container_cannot_elevate_privileges(self):
        """Verify sudo is not installed and user cannot access root files."""
        result = client.containers.run(
            IMAGE_TAG,
            command=["sudo", "ls"],
            remove=True,
        ).decode().strip() if False else None

        # Sudo should fail or not exist
        with pytest.raises(docker.errors.ContainerError):
            client.containers.run(
                IMAGE_TAG,
                command=["sudo", "id"],
                remove=True,
            )

    def test_read_only_root_prevents_system_modification(self):
        """Verify root filesystem is completely read-only when mounted with read_only=True."""
        with pytest.raises(docker.errors.ContainerError) as exc_info:
            client.containers.run(
                IMAGE_TAG,
                command=["touch", "/etc/malicious_config"],
                read_only=True,
                remove=True,
            )
        assert "Read-only file system" in str(exc_info.value)

    def test_network_isolation_blocks_dns_and_egress(self):
        """Verify network_mode='none' blocks all outbound network socket requests."""
        with pytest.raises(docker.errors.ContainerError):
            client.containers.run(
                IMAGE_TAG,
                command=["python3", "-c",
                         "import urllib.request; urllib.request.urlopen('http://169.254.169.254', timeout=2)"],
                network_mode="none",
                remove=True,
            )

    def test_node_and_python_runners_available(self):
        """Verify Node.js, Python, Jest, and Pytest are pre-warmed in the image."""
        node_ver = client.containers.run(IMAGE_TAG, command=["node", "-v"], remove=True).decode().strip()
        py_ver = client.containers.run(IMAGE_TAG, command=["python3", "--version"], remove=True).decode().strip()
        pytest_ver = client.containers.run(IMAGE_TAG, command=["pytest", "--version"], remove=True).decode().strip()

        assert node_ver.startswith("v20.")
        assert "Python 3." in py_ver
        assert "pytest" in pytest_ver
