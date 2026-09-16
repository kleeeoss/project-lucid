"""
Unit tests for Contracts CTR-004, CTR-005, CTR-006, and CTR-007.
Verifies Pydantic V2 schema enforcement, boundaries, and serialization.
"""

import pytest
from pydantic import ValidationError
from models.schemas import (
    RemediationRequest,
    RemediationResponse,
    SandboxRequest,
    SandboxResult,
)
from models.mock_payloads import (
    create_mock_remediation_request,
    create_mock_remediation_response,
    create_mock_sandbox_request,
    create_mock_sandbox_result,
)


class TestRemediationRequestCTR004:
    def test_valid_remediation_request_serialization(self):
        req = create_mock_remediation_request()
        json_data = req.model_dump_json()
        assert "scan_id" in json_data
        assert "CWE-89" in json_data

        # Test deserialization parity
        reconstructed = RemediationRequest.model_validate_json(json_data)
        assert reconstructed.scan_id == req.scan_id
        assert reconstructed.language == "javascript"

    def test_invalid_language_rejected(self):
        with pytest.raises(ValidationError) as exc:
            req = create_mock_remediation_request()
            data = req.model_dump()
            data["language"] = "ruby"  # Unsupported in V1
            RemediationRequest.model_validate(data)
        assert "Input should be 'javascript', 'typescript', 'python' or 'go'" in str(exc.value)

    def test_missing_required_fields_rejected(self):
        with pytest.raises(ValidationError):
            RemediationRequest(scan_id="test-id")  # Missing code, cwe, etc.


class TestRemediationResponseCTR005:
    def test_valid_remediation_response(self):
        res = create_mock_remediation_response()
        assert 0.0 <= res.confidence <= 1.0
        assert res.tokens_used > 0

    @pytest.mark.parametrize("invalid_confidence", [-0.1, 1.01, 2.5])
    def test_confidence_boundary_enforced(self, invalid_confidence):
        res = create_mock_remediation_response()
        data = res.model_dump()
        data["confidence"] = invalid_confidence
        with pytest.raises(ValidationError):
            RemediationResponse.model_validate(data)

    def test_extra_fields_forbidden(self):
        res = create_mock_remediation_response()
        data = res.model_dump()
        data["unauthorized_injected_field"] = "malicious_payload"
        with pytest.raises(ValidationError):
            RemediationResponse.model_validate(data)


class TestSandboxRequestCTR006:
    def test_valid_sandbox_request(self):
        req = create_mock_sandbox_request()
        assert req.timeout_seconds == 60
        assert "package.json" in req.files

    @pytest.mark.parametrize("invalid_timeout", [1, 4, 121, 300])
    def test_timeout_bounds_enforced(self, invalid_timeout):
        req = create_mock_sandbox_request()
        data = req.model_dump()
        data["timeout_seconds"] = invalid_timeout
        with pytest.raises(ValidationError):
            SandboxRequest.model_validate(data)


class TestSandboxResultCTR007:
    def test_valid_sandbox_result(self):
        res = create_mock_sandbox_result()
        assert res.status == "PASSED"
        assert res.exit_code == 0

    def test_invalid_status_rejected(self):
        res = create_mock_sandbox_result()
        data = res.model_dump()
        data["status"] = "UNKNOWN_STATUS"
        with pytest.raises(ValidationError):
            SandboxResult.model_validate(data)