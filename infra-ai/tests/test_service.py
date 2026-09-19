"""
Integration tests for FastAPI endpoints using TestClient.
Verifies /healthz, /remediate (CTR-004 -> CTR-005), and /sandbox/detonate (CTR-006 -> CTR-007).
"""

from fastapi.testclient import TestClient
from service.main import app
from models.mock_payloads import (
    create_mock_remediation_request,
    create_mock_sandbox_request,
)

client = TestClient(app)


def test_healthz_endpoint():
    response = client.get("/healthz")
    assert response.status_code == 200
    assert response.json() == {"status": "healthy", "service": "lucid-ai"}


def test_remediate_endpoint_success():
    req = create_mock_remediation_request(
        scan_id="scan-uuid-12345",
        vulnerability_id="vuln-uuid-67890",
        cwe="CWE-89",
    )
    response = client.post("/remediate", json=req.model_dump())
    assert response.status_code == 200

    data = response.json()
    assert data["scan_id"] == "scan-uuid-12345"
    assert data["vulnerability_id"] == "vuln-uuid-67890"
    # Even without live API keys set in test environment, it falls back cleanly to CTR-005 (BR-002)
    assert "CWE-89" in data["explanation"]
    assert "suggested_patch" in data
    assert 0.0 <= data["confidence"] <= 1.0


def test_remediate_endpoint_validation_error():
    bad_payload = {
        "scan_id": "scan-123",
        "language": "ruby",  # invalid language
    }
    response = client.post("/remediate", json=bad_payload)
    assert response.status_code == 422
    data = response.json()
    assert data["error"] == "SCHEMA_VALIDATION_ERROR"
    assert len(data["details"]) > 0


def test_sandbox_detonate_endpoint_success():
    req = create_mock_sandbox_request(scan_id="scan-uuid-99999")
    response = client.post("/sandbox/detonate", json=req.model_dump())
    assert response.status_code == 200

    data = response.json()
    assert data["scan_id"] == "scan-uuid-99999"
    assert data["status"] == "PASSED"
    assert data["exit_code"] == 0
    assert data["network_egress_attempts"] == 0
