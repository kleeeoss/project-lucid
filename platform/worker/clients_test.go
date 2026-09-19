package worker

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAIClient_RemediateVulnerability(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successful remediation returns patch", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/remediate" || r.Method != http.MethodPost {
				t.Errorf("unexpected route: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(RemediationResponse{
				ScanID:             "scan-123",
				VulnerabilityID:    "vuln-456",
				SuggestedPatch:     "const query = 'SELECT * FROM users WHERE id = $1';",
				Explanation:        "Parameterized query prevents SQL injection",
				Confidence:         0.98,
				ModelName:          "llama-3.1-8b-instant",
				InferenceLatencyMs: 450,
			})
		}))
		defer mockServer.Close()

		client := NewAIClient(mockServer.URL, logger)
		resp, err := client.RemediateVulnerability(context.Background(), RemediationRequest{
			ScanID:          "scan-123",
			VulnerabilityID: "vuln-456",
			CWE:             "CWE-89",
		})

		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if resp.SuggestedPatch == "" || resp.Confidence != 0.98 {
			t.Errorf("unexpected response content: %+v", resp)
		}
	})

	t.Run("HTTP 500 from AI service returns error for Fail-Open handling", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "model out of memory"}`))
		}))
		defer mockServer.Close()

		client := NewAIClient(mockServer.URL, logger)
		_, err := client.RemediateVulnerability(context.Background(), RemediationRequest{
			ScanID: "scan-fail",
		})

		if err == nil {
			t.Fatalf("expected error on HTTP 500, got nil")
		}
	})
}

func TestSandboxClient_Detonate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sandbox/detonate" || r.Method != http.MethodPost {
			t.Errorf("unexpected route: %s %s", r.Method, r.URL.Path)
		}

		var req SandboxRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.TimeoutSeconds != 60 {
			t.Errorf("expected default timeout 60s, got %d", req.TimeoutSeconds)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(SandboxResult{
			ScanID:     req.ScanID,
			Status:     "PASSED",
			ExitCode:   0,
			DurationMs: 1200,
			Stdout:     "tests passed",
		})
	}))
	defer mockServer.Close()

	client := NewSandboxClient(mockServer.URL, logger)
	result, err := client.Detonate(context.Background(), SandboxRequest{
		ScanID:   "scan-sandbox-1",
		Language: "javascript",
	})

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if result.Status != "PASSED" || result.ExitCode != 0 {
		t.Errorf("unexpected sandbox result: %+v", result)
	}
}
