#!/bin/bash
# ============================================================================
# LUCID-CI — Production EC2 Boot Provisioner (Ubuntu 24.04 LTS)
# 1. Configures 4GB Swap Buffer
# 2. Installs Docker Engine, Compose plugin, and Google gVisor (runsc)
# 3. Creates Persistent Host Directories for Postgres & Caddy
# 4. Bootstraps Repository Code (Git Clone) & Builds Sandbox Runner Image
# 5. Configures lucid-autostop.service (Server-Side Deterministic Timer via IMDSv2)
# 6. Configures lucid.service (Auto-starts Docker Compose stack on boot)
# ============================================================================

set -eo pipefail
export DEBIAN_FRONTEND=noninteractive

echo "=================================================="
echo "Starting Lucid-CI EC2 Host Provisioning..."
echo "=================================================="

# 1. Configure 4 GB Swapfile Buffer
if [ ! -f /swapfile ]; then
    echo "Creating 4 GB swapfile buffer..."
    fallocate -l 4G /swapfile || dd if=/dev/zero of=/swapfile bs=1M count=4096
    chmod 600 /swapfile
    mkswap /swapfile
    swapon /swapfile
    echo '/swapfile none swap sw 0 0' >> /etc/fstab
fi

# 2. Update packages and install base dependencies
apt-get update && apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    git \
    gnupg \
    lsb-release \
    python3

# Install official Docker Engine & Compose plugin
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Google gVisor (runsc)
ARCH=$(uname -m)
curl -fsSL https://gvisor.dev/archive.key | gpg --dearmor -o /usr/share/keyrings/gvisor-archive-keyring.gpg
echo "deb [arch=${ARCH} signed-by=/usr/share/keyrings/gvisor-archive-keyring.gpg] https://storage.googleapis.com/gvisor/releases release main" | \
  tee /etc/apt/sources.list.d/gvisor.list > /dev/null

apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin runsc

# Configure Docker daemon to register runsc runtime
mkdir -p /etc/docker
python3 -c '
import json, os
path = "/etc/docker/daemon.json"
try:
    with open(path, "r") as f:
        data = json.load(f)
except Exception:
    data = {}
runtimes = data.get("runtimes", {})
runtimes["runsc"] = {
    "path": "/usr/bin/runsc",
    "runtimeArgs": [
        "--network=none",
        "--platform=ptrace"
    ]
}
data["runtimes"] = runtimes
with open(path, "w") as f:
    json.dump(data, f, indent=2)
'

systemctl enable docker
systemctl restart docker
usermod -aG docker ubuntu

# 3. Create Persistent EBS Host Directories
echo "Setting up persistent host directories..."
mkdir -p /var/lib/lucid/data/postgres
mkdir -p /var/lib/lucid/data/caddy
mkdir -p /var/lib/lucid/config
chmod 700 /var/lib/lucid/config
mkdir -p /var/lib/lucid/config/caddy
mkdir -p /tmp/lucid-sandboxes
mkdir -p /opt/lucid-ci

# UID 999:999 matches unprivileged postgres in postgres:16-alpine
chown -R 999:999 /var/lib/lucid/data/postgres
chmod 777 /tmp/lucid-sandboxes

# 4. Clone Repository & Pre-build Sandbox Runner
echo "Bootstrapping Lucid-CI codebase..."

# Query IMDSv2 token for metadata discovery
TOKEN=$(curl -s -S -f -X PUT "http://169.254.169.254/latest/api/token" \
  -H "X-aws-ec2-metadata-token-ttl-seconds: 300" 2>/dev/null || true)

# 4a. Resolve Git Revision (Default to main; pinned commit SHA / branch can be specified via tag or env)
RAW_REVISION=""
if [ -n "$TOKEN" ]; then
    RAW_REVISION=$(curl -s -S -f -H "X-aws-ec2-metadata-token: $TOKEN" \
      "http://169.254.169.254/latest/meta-data/tags/instance/GitRevision" 2>/dev/null || true)
fi
GIT_REVISION="${GIT_REVISION:-${RAW_REVISION:-main}}"

if [ ! -d "/opt/lucid-ci/.git" ]; then
    git clone https://github.com/kleeeoss/project-lucid.git /opt/lucid-ci
    git -C /opt/lucid-ci checkout "$GIT_REVISION"
else
    git -C /opt/lucid-ci fetch origin
    git -C /opt/lucid-ci checkout "$GIT_REVISION"
    git -C /opt/lucid-ci pull origin "$GIT_REVISION" 2>/dev/null || true
fi

# 4b. Discover Real AWS SQS Queue URL & AWS Region from IMDSv2
RAW_SQS_TAG=""
if [ -n "$TOKEN" ]; then
    RAW_SQS_TAG=$(curl -s -S -f -H "X-aws-ec2-metadata-token: $TOKEN" \
      "http://169.254.169.254/latest/meta-data/tags/instance/SqsQueueUrl" 2>/dev/null || true)
fi

IDENTITY_DOC=""
if [ -n "$TOKEN" ]; then
    IDENTITY_DOC=$(curl -s -S -f -H "X-aws-ec2-metadata-token: $TOKEN" \
      "http://169.254.169.254/latest/dynamic/instance-identity/document" 2>/dev/null || true)
fi

EC2_REGION=""
EC2_ACCOUNT_ID=""
if [ -n "$IDENTITY_DOC" ]; then
    EC2_REGION=$(echo "$IDENTITY_DOC" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("region", ""))' 2>/dev/null || true)
    EC2_ACCOUNT_ID=$(echo "$IDENTITY_DOC" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("accountId", ""))' 2>/dev/null || true)
fi
EC2_REGION="${EC2_REGION:-ap-southeast-2}"

REAL_SQS_URL="${SQS_QUEUE_URL:-${RAW_SQS_TAG:-}}"
if [ -z "$REAL_SQS_URL" ] && command -v aws >/dev/null 2>&1; then
    REAL_SQS_URL=$(aws sqs get-queue-url --queue-name lucid-ci-scans --region "$EC2_REGION" --query 'QueueUrl' --output text 2>/dev/null || true)
fi

if [ -z "$REAL_SQS_URL" ] || [ "$REAL_SQS_URL" = "None" ]; then
    if [ -n "$EC2_ACCOUNT_ID" ]; then
        REAL_SQS_URL="https://sqs.${EC2_REGION}.amazonaws.com/${EC2_ACCOUNT_ID}/lucid-ci-scans"
    else
        REAL_SQS_URL="https://sqs.${EC2_REGION}.amazonaws.com/lucid-ci-scans"
    fi
fi

# 4c. Configure Environment (.env) with Real AWS SQS URL and Directory Mounts
if [ -f "/opt/lucid-ci/.env.example" ] && [ ! -f "/opt/lucid-ci/.env" ]; then
    cp /opt/lucid-ci/.env.example /opt/lucid-ci/.env
fi

if [ -f "/opt/lucid-ci/.env" ]; then
    # Ensure real AWS SQS queue URL is set instead of LocalStack localhost:4566
    if grep -q "^SQS_QUEUE_URL=" /opt/lucid-ci/.env; then
        sed -i "s|^SQS_QUEUE_URL=.*|SQS_QUEUE_URL=${REAL_SQS_URL}|" /opt/lucid-ci/.env
    else
        echo "SQS_QUEUE_URL=${REAL_SQS_URL}" >> /opt/lucid-ci/.env
    fi
    # Clear localstack endpoint override so AWS SDK targets standard regional endpoints
    sed -i "s|^SQS_ENDPOINT_URL=.*|SQS_ENDPOINT_URL=|" /opt/lucid-ci/.env
    sed -i "s|^AWS_REGION=.*|AWS_REGION=${EC2_REGION}|" /opt/lucid-ci/.env
    sed -i "s|^AWS_DEFAULT_REGION=.*|AWS_DEFAULT_REGION=${EC2_REGION}|" /opt/lucid-ci/.env
fi

# Pre-build Sandbox Runner image so dynamic execution is immediate
if [ -f "/opt/lucid-ci/infra-ai/sandbox/Dockerfile.runner" ]; then
    docker build -t lucid-sandbox-runner:latest \
      -f /opt/lucid-ci/infra-ai/sandbox/Dockerfile.runner \
      /opt/lucid-ci/infra-ai/sandbox
fi

chown -R ubuntu:ubuntu /opt/lucid-ci

# 5. Install Server-Side Auto-Stop Script & Systemd Unit
cat <<'EOF' > /usr/local/bin/lucid-autostop.sh
#!/usr/bin/env bash
# Reads DemoRuntimeMinutes tag via IMDSv2 and schedules a clean OS shutdown

set -eo pipefail

TOKEN=$(curl -s -S -f -X PUT "http://169.254.169.254/latest/api/token" \
  -H "X-aws-ec2-metadata-token-ttl-seconds: 60" 2>/dev/null || true)

RAW_MINUTES=""
if [ -n "$TOKEN" ]; then
    RAW_MINUTES=$(curl -s -S -f -H "X-aws-ec2-metadata-token: $TOKEN" \
      "http://169.254.169.254/latest/meta-data/tags/instance/DemoRuntimeMinutes" 2>/dev/null || true)
fi

# Validate integer between 5 and 480 (8 hours max); default to 120
if [[ "$RAW_MINUTES" =~ ^[0-9]+$ ]] && [ "$RAW_MINUTES" -ge 5 ] && [ "$RAW_MINUTES" -le 480 ]; then
    RUNTIME_MINUTES="$RAW_MINUTES"
else
    RUNTIME_MINUTES="120"
fi

echo "Lucid-CI: Scheduling server-side shutdown in ${RUNTIME_MINUTES} minutes..."
shutdown -h "+${RUNTIME_MINUTES}" "Lucid-CI Auto-Stop: Showcase runtime expired. Halting instance to cease compute charges."
EOF

chmod +x /usr/local/bin/lucid-autostop.sh

cat <<'EOF' > /etc/systemd/system/lucid-autostop.service
[Unit]
Description=Lucid-CI Server-Side Auto-Stop Timer
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/local/bin/lucid-autostop.sh

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable lucid-autostop.service
systemctl start lucid-autostop.service

# 6. Configure Systemd Auto-Start for Docker Compose Stack
cat <<'EOF' > /etc/systemd/system/lucid.service
[Unit]
Description=Lucid-CI Production Stack
After=docker.service network-online.target
Requires=docker.service
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/lucid-ci
# ConditionCheck: verify docker-compose.prod.yml exists before starting
ExecStartPre=/bin/bash -c "test -f /opt/lucid-ci/docker-compose.prod.yml || { echo '/opt/lucid-ci/docker-compose.prod.yml not found.'; exit 0; }"
ExecStart=/usr/bin/docker compose -f /opt/lucid-ci/docker-compose.prod.yml up -d
ExecStop=/usr/bin/docker compose -f /opt/lucid-ci/docker-compose.prod.yml down

[Install]
WantedBy=multi-user.target
EOF

systemctl enable lucid.service
systemctl start --no-block lucid.service

echo "=================================================="
echo "Lucid-CI Host Provisioning Complete!"
echo "=================================================="
