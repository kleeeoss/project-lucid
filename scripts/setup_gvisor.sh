#!/usr/bin/env bash
# ============================================================================
# LUCID-CI — gVisor (runsc) Host Installation Script for Ubuntu 24.04 LTS
# Installs and registers gVisor as a secure container runtime in Docker daemon.
# ============================================================================

set -eo pipefail

echo "=========================================================="
echo "Installing gVisor (runsc) on Ubuntu 24.04 LTS..."
echo "=========================================================="

ARCH=$(uname -m)

# 1. Add Google gVisor official apt repository
sudo apt-get update && sudo apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg

curl -fsSL https://gvisor.dev/archive.key | sudo gpg --dearmor -o /usr/share/keyrings/gvisor-archive-keyring.gpg
echo "deb [arch=${ARCH} signed-by=/usr/share/keyrings/gvisor-archive-keyring.gpg] https://storage.googleapis.com/gvisor/releases release main" | sudo tee /etc/apt/sources.list.d/gvisor.list > /dev/null

# 2. Install runsc
sudo apt-get update && sudo apt-get install -y runsc

# 3. Configure Docker Daemon to register runsc runtime
echo "Configuring Docker daemon with runsc runtime..."
sudo mkdir -p /etc/docker

if [ ! -f /etc/docker/daemon.json ]; then
    echo '{}' | sudo tee /etc/docker/daemon.json > /dev/null
fi

# Use Python to merge runsc configuration into /etc/docker/daemon.json cleanly
sudo python3 -c '
import json, os
path = "/etc/docker/daemon.json"
try:
    with open(path, "r") as f:
        data = json.load(f)
except Exception:
    data = {}
runtimes = data.get("runtimes", {})
runtimes["runsc"] = {"path": "/usr/bin/runsc"}
data["runtimes"] = runtimes
with open(path, "w") as f:
    json.dump(data, f, indent=2)
'

# 4. Restart Docker daemon
echo "Restarting Docker daemon..."
sudo systemctl restart docker

# 5. Verify gVisor installation
echo "Verifying gVisor installation..."
if sudo runsc --version > /dev/null 2>&1; then
    echo "gVisor binary verified: $(sudo runsc --version | head -1)"
fi

if sudo docker info | grep -q "runsc"; then
    echo "✓ Docker successfully configured with runsc runtime!"
else
    echo "WARNING: runsc not detected in docker info. Check /etc/docker/daemon.json"
    exit 1
fi

echo "=========================================================="
echo "gVisor Installation & Docker Configuration Complete!"
echo "=========================================================="
