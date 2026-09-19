package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// RemediationRequest defines input to Garv's AI remediation service (Contract CTR-004).
type RemediationRequest struct {
	ScanID             string   `json:"scan_id"`
	VulnerabilityID    string   `json:"vulnerability_id"`
	RuleID             string   `json:"rule_id"`
	CWE                string   `json:"cwe"`
	Language           string   `json:"language"`
	VulnerableCode     string   `json:"vulnerable_code"`
	SurroundingContext string   `json:"surrounding_context"`
	SourceInfo         string   `json:"source_info"`
	SinkInfo           string   `json:"sink_info"`
	TaintPathSummary   []string `json:"taint_path_summary"`
	Nonce              string   `json:"nonce,omitempty"`
}

// RemediationResponse defines output from Garv's AI remediation service (Contract CTR-005).
type RemediationResponse struct {
	ScanID             string  `json:"scan_id"`
	VulnerabilityID    string  `json:"vulnerability_id"`
	SuggestedPatch     string  `json:"suggested_patch"`
	Explanation        string  `json:"explanation"`
	SecurityRationale  string  `json:"security_rationale"`
	Confidence         float64 `json:"confidence"`
	ModelName          string  `json:"model_name"`
	TokensUsed         int     `json:"tokens_used"`
	InferenceLatencyMs int     `json:"inference_latency_ms"`
}

// AIClient defines the interface for communicating with the AI microservice.
type AIClient interface {
	RemediateVulnerability(ctx context.Context, req RemediationRequest) (*RemediationResponse, error)
}

type aiClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewAIClient initializes the HTTP client with a 15-second timeout.
func NewAIClient(baseURL string, logger *slog.Logger) AIClient {
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	return &aiClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		logger:     logger,
	}
}

// RemediateVulnerability calls POST /remediate on Garv's service.
// On failure, it returns an error so caller can apply Fail-Open rule (BR-002).
func (c *aiClient) RemediateVulnerability(ctx context.Context, req RemediationRequest) (*RemediationResponse, error) {
	url := fmt.Sprintf("%s/remediate", c.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RemediationRequest: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create remediation request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Warn("AI remediation service unreachable (Fail-Open active)",
			"scan_id", req.ScanID,
			"vuln_id", req.VulnerabilityID,
			"error", err,
		)
		return nil, fmt.Errorf("AI service network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		c.logger.Warn("AI remediation returned non-200 status (Fail-Open active)",
			"status_code", resp.StatusCode,
			"body", string(b),
		)
		return nil, fmt.Errorf("AI service returned status %d: %s", resp.StatusCode, string(b))
	}

	var remResp RemediationResponse
	if err := json.NewDecoder(resp.Body).Decode(&remResp); err != nil {
		return nil, fmt.Errorf("failed to decode RemediationResponse: %w", err)
	}

	c.logger.Info("Received AI remediation patch",
		"scan_id", remResp.ScanID,
		"model", remResp.ModelName,
		"confidence", remResp.Confidence,
		"latency_ms", remResp.InferenceLatencyMs,
	)

	return &remResp, nil
}
