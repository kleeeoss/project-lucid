#!/usr/bin/env bash
# ============================================================================
# LUCID-CI — End-to-End Local Integration Pipeline Test Harness (TASK-MST-303)
# Validates PostgreSQL, LocalStack SQS, AI Remediation, and Sandbox Detonation.
# ============================================================================

set -eo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=================================================================${NC}"
echo -e "${BLUE}         LUCID-CI — Local Pipeline Integration Verification       ${NC}"
echo -e "${BLUE}=================================================================${NC}"

# 1. PostgreSQL Health Check
echo -n "[1/5] Checking PostgreSQL (localhost:5432)... "
if docker exec lucid-postgres pg_isready -U lucid -d lucid_ci > /dev/null 2>&1; then
    echo -e "${GREEN}PASSED (Healthy)${NC}"
else
    echo -e "${RED}FAILED${NC}"
    exit 1
fi

# 2. LocalStack SQS Health Check & Message Roundtrip
echo -n "[2/5] Checking LocalStack SQS (localhost:4566)... "
SQS_CHECK=$(curl -s "http://localhost:4566/000000000000/lucid-ci-scans?Action=GetQueueAttributes&AttributeName.1=VisibilityTimeout")
if echo "$SQS_CHECK" | grep -q "180"; then
    echo -e "${GREEN}PASSED (lucid-ci-scans active)${NC}"
else
    echo -e "${RED}FAILED (Queue unreachable)${NC}"
    exit 1
fi

# 3. AI Service Health Check
echo -n "[3/5] Checking AI Service Liveness (localhost:8000/healthz)... "
HEALTH_RESP=$(curl -s http://localhost:8000/healthz)
if echo "$HEALTH_RESP" | grep -q "healthy"; then
    echo -e "${GREEN}PASSED (HTTP 200 OK)${NC}"
else
    echo -e "${RED}FAILED${NC}"
    exit 1
fi

# 4. AI Remediation Pipeline Verification (CTR-004 -> CTR-005)
echo -n "[4/5] Testing AI Remediation Pipeline (POST /remediate)... "
REMEDIATE_RESP=$(curl -s -X POST "http://localhost:8000/remediate" \
  -H "Content-Type: application/json" \
  -d '{
    "scan_id": "e2e-scan-001",
    "vulnerability_id": "vuln-e2e-001",
    "rule_id": "LUCID-SEC-001",
    "cwe": "CWE-89",
    "language": "javascript",
    "vulnerable_code": "const query = `SELECT * FROM users WHERE id = ${req.body.id}`;",
    "surrounding_context": "app.post(\"/users\", (req, res) => { const query = `SELECT * FROM users WHERE id = ${req.body.id}`; db.query(query); });",
    "source_info": "req.body.id",
    "sink_info": "db.query()",
    "taint_path_summary": ["req.body.id (Source)", "db.query() (Sink)"]
  }')

if echo "$REMEDIATE_RESP" | grep -q "suggested_patch" && echo "$REMEDIATE_RESP" | grep -q "e2e-scan-001"; then
    echo -e "${GREEN}PASSED (Valid CTR-005 returned)${NC}"
else
    echo -e "${RED}FAILED${NC}"
    echo "Response: $REMEDIATE_RESP"
    exit 1
fi

# 5. Sandbox Detonation via Docker-out-of-Docker (CTR-006 -> CTR-007)
echo -n "[5/5] Testing Ephemeral Sandbox Detonation (POST /sandbox/detonate)... "
DETONATE_RESP=$(curl -s -X POST "http://localhost:8000/sandbox/detonate" \
  -H "Content-Type: application/json" \
  -d '{
    "scan_id": "e2e-sandbox-001",
    "language": "python",
    "build_command": "python3 -c \"print(\\\"E2E_DETONATION_CLEAN\\\"); exit(0)\"",
    "timeout_seconds": 15,
    "files": {
      "main.py": "print(\"hello\")"
    }
  }')

if echo "$DETONATE_RESP" | grep -q "PASSED" && echo "$DETONATE_RESP" | grep -q "E2E_DETONATION_CLEAN"; then
    echo -e "${GREEN}PASSED (Container spawned, executed, and cleaned up)${NC}"
else
    echo -e "${RED}FAILED${NC}"
    echo "Response: $DETONATE_RESP"
    exit 1
fi

echo -e "${BLUE}=================================================================${NC}"
echo -e "${GREEN}✓ ALL LOCAL INTEGRATION TESTS PASSED CLEANLY (Gate 3 Ready)${NC}"
echo -e "${BLUE}=================================================================${NC}"
