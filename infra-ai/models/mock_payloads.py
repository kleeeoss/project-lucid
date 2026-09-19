"""
Mock payload factories for testing and cross-domain contract verification.
"""

import uuid
from typing import Optional, Dict
from models.schemas import (
    RemediationRequest,
    RemediationResponse,
    SandboxRequest,
    SandboxResult,
)


def create_mock_remediation_request(
    scan_id: Optional[str] = None,
    vulnerability_id: Optional[str] = None,
    cwe: str = "CWE-89",
    rule_id: str = "LUCID-SEC-001",
    nonce: Optional[str] = "a1b2c3d4e5f67890",
) -> RemediationRequest:
    return RemediationRequest(
        scan_id=scan_id or str(uuid.uuid4()),
        vulnerability_id=vulnerability_id or f"vuln_{uuid.uuid4().hex[:8]}",
        rule_id=rule_id,
        cwe=cwe,
        language="javascript",
        vulnerable_code="const query = `SELECT * FROM users WHERE id = '${userId}'`;",
        surrounding_context=(
            "async function getUser(req, res) {\n"
            "  const userId = req.params.id;\n"
            "  const query = `SELECT * FROM users WHERE id = '${userId}'`;\n"
            "  const user = await db.query(query);\n"
            "  return res.json(user);\n"
            "}"
        ),
        source_info="req.params.id (HTTP route parameter)",
        sink_info="db.query (Raw database query execution)",
        taint_path_summary=[
            "req.params.id (Source: User input)",
            "userId (Assignment)",
            "query (String template concatenation)",
            "db.query(query) (Sink: SQL injection)"
        ],
        nonce=nonce,
    )


def create_mock_remediation_response(
    scan_id: Optional[str] = None,
    vulnerability_id: Optional[str] = None,
) -> RemediationResponse:
    return RemediationResponse(
        scan_id=scan_id or str(uuid.uuid4()),
        vulnerability_id=vulnerability_id or f"vuln_{uuid.uuid4().hex[:8]}",
        suggested_patch=(
            "const query = 'SELECT * FROM users WHERE id = $1';\n"
            "const user = await db.query(query, [userId]);"
        ),
        explanation=(
            "User-controlled parameter `userId` is interpolated directly into the SQL query string. "
            "An attacker can pass malicious SQL payloads to bypass authentication or extract sensitive records."
        ),
        security_rationale=(
            "The query was rewritten using parameterized query markers ($1) and parameters array. "
            "The PostgreSQL driver separates SQL code from user data, preventing command structure manipulation."
        ),
        confidence=0.95,
        model_name="llama-3.1-8b-instant",
        tokens_used=245,
        inference_latency_ms=850,
    )


def create_mock_sandbox_request(
    scan_id: Optional[str] = None,
    files: Optional[Dict[str, str]] = None,
) -> SandboxRequest:
    return SandboxRequest(
        scan_id=scan_id or str(uuid.uuid4()),
        files=files or {
            "package.json": '{"name": "test-app", "scripts": {"test": "node index.test.js"}}',
            "index.test.js": 'console.log("Tests passed successfully!"); process.exit(0);',
        },
        language="javascript",
        build_command="npm test",
        timeout_seconds=60,
    )


def create_mock_sandbox_result(
    scan_id: Optional[str] = None,
    status: str = "PASSED",
) -> SandboxResult:
    return SandboxResult(
        scan_id=scan_id or str(uuid.uuid4()),
        status=status,
        exit_code=0,
        duration_ms=1200,
        stdout="Tests passed successfully!\n",
        stderr="",
        network_egress_attempts=0,
    )
