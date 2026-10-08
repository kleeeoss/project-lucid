#!/usr/bin/env bash
# ============================================================================
# LUCID-CI — On-Demand Showcase Lifecycle Manager (POSIX / Bash)
# Region: ap-southeast-2 (Asia Pacific - Sydney)
# Commands:
#   bash scripts/demo.sh up [runtime_minutes]
#   bash scripts/demo.sh down
#   bash scripts/demo.sh status
# ============================================================================

set -eo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TF_DIR="${REPO_ROOT}/infra-ai/terraform"
ENV_OVERRIDE_FILE="${REPO_ROOT}/.demo.env"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default deployment region & optional profile for this project
AWS_REGION="${AWS_REGION:-ap-southeast-2}"
# Default to empty profile to allow standard default credentials unless explicitly set
AWS_PROFILE="${AWS_PROFILE:-}"

# Wrapper for AWS CLI enforcing region and profile
run_aws() {
    if [ -n "${AWS_PROFILE}" ]; then
        aws --region "${AWS_REGION}" --profile "${AWS_PROFILE}" "$@"
    else
        aws --region "${AWS_REGION}" "$@"
    fi
}

# 1. Dependency Verification
check_dependencies() {
    if ! command -v aws > /dev/null 2>&1; then
        echo -e "${RED}ERROR: AWS CLI ('aws') is not installed or not in PATH.${NC}" >&2
        exit 1
    fi
    if ! command -v curl > /dev/null 2>&1; then
        echo -e "${RED}ERROR: 'curl' is not installed or not in PATH.${NC}" >&2
        exit 1
    fi
}

# 2. Resolve Infrastructure Identifiers
load_infra_config() {
    # Check for optional override file first
    if [ -f "${ENV_OVERRIDE_FILE}" ]; then
        # shellcheck disable=SC1090
        source "${ENV_OVERRIDE_FILE}"
    fi

    if [ -z "${INSTANCE_ID:-}" ] || [ -z "${PUBLIC_IP:-}" ]; then
        if [ ! -d "${TF_DIR}/.terraform" ]; then
            echo -e "${RED}ERROR: Terraform state not found at ${TF_DIR}.${NC}" >&2
            echo -e "Run 'terraform apply' first, or set INSTANCE_ID and PUBLIC_IP in .demo.env." >&2
            exit 1
        fi

        INSTANCE_ID=$(terraform -chdir="${TF_DIR}" output -raw instance_id 2>/dev/null || true)
        PUBLIC_IP=$(terraform -chdir="${TF_DIR}" output -raw ec2_public_ip 2>/dev/null || true)
        DOMAIN=$(terraform -chdir="${TF_DIR}" output -raw domain 2>/dev/null || true)
        TF_REGION=$(terraform -chdir="${TF_DIR}" output -raw aws_region 2>/dev/null || true)

        if [ -n "${TF_REGION}" ] && [ "${TF_REGION}" != "null" ]; then
            AWS_REGION="${TF_REGION}"
        fi
    fi

    if [ -z "${INSTANCE_ID:-}" ] || [ "${INSTANCE_ID}" = "null" ]; then
        echo -e "${RED}ERROR: Unable to resolve EC2 Instance ID.${NC}" >&2
        exit 1
    fi
}

# 3. Get Current Instance State
get_instance_state() {
    run_aws ec2 describe-instances \
      --instance-ids "${INSTANCE_ID}" \
      --query 'Reservations[0].Instances[0].State.Name' \
      --output text 2>/dev/null || echo "unknown"
}

# 4. Hostname-Aware Health Polling
poll_health() {
    local max_attempts=36 # 36 * 5s = 180s total timeout
    local attempt=1

    echo -e "${BLUE}Polling service health check (/healthz)...${NC}"

    while [ "${attempt}" -le "${max_attempts}" ]; do
        local http_code="000"

        if [ -n "${DOMAIN:-}" ] && [ "${DOMAIN}" != "null" ]; then
            # SNI / TLS validation via --resolve to avoid DNS race conditions
            http_code=$(curl -s -k -m 4 -o /dev/null -w "%{http_code}" \
              --resolve "${DOMAIN}:443:${PUBLIC_IP}" \
              "https://${DOMAIN}/healthz" 2>/dev/null || echo "000")
        else
            # Plain HTTP on bare Elastic IP
            http_code=$(curl -s -m 4 -o /dev/null -w "%{http_code}" \
              "http://${PUBLIC_IP}/healthz" 2>/dev/null || echo "000")
        fi

        if [ "${http_code}" = "200" ]; then
            echo -e "\n${GREEN}✓ Lucid-CI services are healthy and responsive (HTTP 200).${NC}"
            return 0
        fi

        echo -n "."
        sleep 5
        attempt=$((attempt + 1))
    done

    echo -e "\n${YELLOW}WARNING: Health check timed out after 180s. Containers may still be starting.${NC}"
    return 1
}

# ============================================================================
# COMMAND: demo up [runtime_minutes]
# ============================================================================
cmd_up() {
    local raw_runtime="${1:-120}"
    local runtime=120

    if [[ "${raw_runtime}" =~ ^[0-9]+$ ]] && [ "${raw_runtime}" -ge 5 ] && [ "${raw_runtime}" -le 480 ]; then
        runtime="${raw_runtime}"
    else
        echo -e "${YELLOW}Notice: Invalid runtime '${raw_runtime}'. Using default of 120 minutes (5-480 allowed).${NC}"
    fi

    local state
    state=$(get_instance_state)

    if [ "${state}" = "running" ]; then
        echo -e "${YELLOW}Notice: Instance ${INSTANCE_ID} is ALREADY RUNNING in ${AWS_REGION}.${NC}"
        echo -e "The server-side auto-stop timer was armed at boot and cannot be rescheduled from this command."
        echo -e "Use 'bash scripts/demo.sh status' to inspect current health."
        exit 0
    fi

    if [ "${state}" = "stopping" ]; then
        echo -e "${BLUE}Instance is currently stopping. Waiting for 'stopped' state before restarting...${NC}"
        run_aws ec2 wait instance-stopped --instance-ids "${INSTANCE_ID}"
    fi

    echo -e "${BLUE}Setting server-side auto-stop timer: ${runtime} minutes (${AWS_REGION})...${NC}"
    run_aws ec2 create-tags \
      --resources "${INSTANCE_ID}" \
      --tags "Key=DemoRuntimeMinutes,Value=${runtime}"

    echo -e "${BLUE}Starting EC2 Instance (${INSTANCE_ID})...${NC}"
    run_aws ec2 start-instances --instance-ids "${INSTANCE_ID}" > /dev/null

    echo -e "${BLUE}Waiting for instance to reach 'running' state...${NC}"
    run_aws ec2 wait instance-running --instance-ids "${INSTANCE_ID}"
    echo -e "${GREEN}✓ EC2 instance is running.${NC}"

    # Poll health check
    poll_health || true

    # Display endpoints
    echo -e "\n${BLUE}=================================================================${NC}"
    echo -e "${GREEN}             LUCID-CI SHOWCASE DEPLOYMENT ONLINE                 ${NC}"
    echo -e "${BLUE}=================================================================${NC}"
    echo -e "AWS Region:         ${AWS_REGION}"
    echo -e "Instance ID:        ${INSTANCE_ID}"
    echo -e "Public Elastic IP:  ${PUBLIC_IP}"

    if [ -n "${DOMAIN:-}" ] && [ "${DOMAIN}" != "null" ]; then
        echo -e "Live Domain:        https://${DOMAIN}"
        echo -e "Webhook Target:     https://${DOMAIN}/webhook"
        echo -e "Health Endpoint:    https://${DOMAIN}/healthz"
    else
        echo -e "Live URL (HTTP):    http://${PUBLIC_IP}"
        echo -e "Webhook Target:     http://${PUBLIC_IP}/webhook"
        echo -e "Health Endpoint:    http://${PUBLIC_IP}/healthz"
    fi

    echo -e "${YELLOW}Auto-Stop Notice:   Armed for ${runtime} minutes. Compute charges will automatically cease.${NC}"
    echo -e "${BLUE}=================================================================${NC}"
}

# ============================================================================
# COMMAND: demo down
# ============================================================================
cmd_down() {
    local state
    state=$(get_instance_state)

    if [ "${state}" = "stopped" ]; then
        echo -e "${GREEN}Instance ${INSTANCE_ID} is already stopped. Compute cost is $0.00/hr.${NC}"
        exit 0
    fi

    echo -e "${BLUE}Initiating graceful shutdown on ${INSTANCE_ID} (${AWS_REGION})...${NC}"
    run_aws ec2 stop-instances --instance-ids "${INSTANCE_ID}" > /dev/null

    echo -e "${BLUE}Waiting for instance to enter 'stopped' state...${NC}"
    run_aws ec2 wait instance-stopped --instance-ids "${INSTANCE_ID}"

    echo -e "${GREEN}✓ Instance stopped successfully.${NC}"
    echo -e "Compute billing halted. Persistent state on EBS and Elastic IP remain preserved."
}

# ============================================================================
# COMMAND: demo status
# ============================================================================
cmd_status() {
    echo -e "${BLUE}Querying Lucid-CI Showcase status (${AWS_REGION})...${NC}"
    local json_desc
    json_desc=$(run_aws ec2 describe-instances --instance-ids "${INSTANCE_ID}" --query 'Reservations[0].Instances[0]' 2>/dev/null)

    local state
    state=$(echo "${json_desc}" | jq -r '.State.Name // "unknown"')
    local launch_time
    launch_time=$(echo "${json_desc}" | jq -r '.LaunchTime // "N/A"')
    local runtime_tag
    runtime_tag=$(echo "${json_desc}" | jq -r '(.Tags[] | select(.Key=="DemoRuntimeMinutes") | .Value) // "120"')

    echo -e "\n${BLUE}=================================================================${NC}"
    echo -e "AWS Region:         ${AWS_REGION}"
    echo -e "Instance ID:        ${INSTANCE_ID}"
    echo -e "Public Elastic IP:  ${PUBLIC_IP}"
    echo -e "Configured Domain:  ${DOMAIN:-None (Plain HTTP)}"

    if [ "${state}" = "running" ]; then
        echo -e "State:              ${GREEN}RUNNING${NC}"
        echo -e "Boot Timestamp:     ${launch_time}"
        echo -e "Auto-Stop Tag:      ${runtime_tag} minutes"

        echo -n "Live Health Probe:  "
        local http_code="000"
        if [ -n "${DOMAIN:-}" ] && [ "${DOMAIN}" != "null" ]; then
            http_code=$(curl -s -k -m 3 -o /dev/null -w "%{http_code}" \
              --resolve "${DOMAIN}:443:${PUBLIC_IP}" "https://${DOMAIN}/healthz" 2>/dev/null || echo "000")
        else
            http_code=$(curl -s -m 3 -o /dev/null -w "%{http_code}" "http://${PUBLIC_IP}/healthz" 2>/dev/null || echo "000")
        fi

        if [ "${http_code}" = "200" ]; then
            echo -e "${GREEN}HEALTHY (HTTP 200)${NC}"
        else
            echo -e "${YELLOW}UNRESPONSIVE (HTTP ${http_code})${NC}"
        fi
    else
        echo -e "State:              ${YELLOW}STOPPED${NC} (Compute billing halted: $0.00/hr)"
    fi
    echo -e "${BLUE}=================================================================${NC}"
}

# ============================================================================
# Main Entry Point
# ============================================================================
case "${1:-}" in
    up)
        check_dependencies
        load_infra_config
        cmd_up "${2:-120}"
        ;;
    down)
        check_dependencies
        load_infra_config
        cmd_down
        ;;
    status)
        check_dependencies
        load_infra_config
        cmd_status
        ;;
    *)
        echo "Usage: bash scripts/demo.sh {up [runtime_minutes]|down|status}" >&2
        echo "Environment variables: AWS_REGION (default: ap-southeast-2), AWS_PROFILE (optional)" >&2
        exit 1
        ;;
esac
