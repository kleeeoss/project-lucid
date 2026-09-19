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

// SandboxRequest defines input to Garv's sandbox detonation service (Contract CTR-006).
type SandboxRequest struct {
	ScanID         string            `json:"scan_id"`
	RepositoryURL  string            `json:"repository_url,omitempty"`
	CommitSHA      string            `json:"commit_sha,omitempty"`
	Files          map[string]string `json:"files,omitempty"`
	PatchContent   string            `json:"patch_content,omitempty"`
	Language       string            `json:"language"`
	BuildCommand   string            `json:"build_command"`
	TimeoutSeconds int               `json:"timeout_seconds"` // Standardized: 60s
}

// SandboxResult defines output from Garv's sandbox detonation service (Contract CTR-007).
type SandboxResult struct {
	ScanID                string `json:"scan_id"`
	Status                string `json:"status"` // "PASSED", "FAILED", "TIMED_OUT", "ESCAPE_DETECTED"
	ExitCode              int    `json:"exit_code"`
	DurationMs            int    `json:"duration_ms"`
	Stdout                string `json:"stdout"`
	Stderr                string `json:"stderr"`
	NetworkEgressAttempts int    `json:"network_egress_attempts"`
}

// SandboxClient defines the interface for executing dynamic detonation tests.
type SandboxClient interface {
	Detonate(ctx context.Context, req SandboxRequest) (*SandboxResult, error)
}

type sandboxClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewSandboxClient initializes the sandbox client with a 120-second caller context timeout.
func NewSandboxClient(baseURL string, logger *slog.Logger) SandboxClient {
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	return &sandboxClient{
		baseURL: baseURL,
		// 120s caller timeout allows for the 60s container timeout + initialization overhead
		httpClient: &http.Client{Timeout: 120 * time.Second},
		logger:     logger,
	}
}

// Detonate calls POST /sandbox/detonate on Garv's microservice.
func (c *sandboxClient) Detonate(ctx context.Context, req SandboxRequest) (*SandboxResult, error) {
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 60 // Standardized 60-second watchdog
	}

	url := fmt.Sprintf("%s/sandbox/detonate", c.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SandboxRequest: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox detonation request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	c.logger.Info("Requesting dynamic sandbox detonation",
		"scan_id", req.ScanID,
		"language", req.Language,
		"timeout_seconds", req.TimeoutSeconds,
	)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sandbox service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sandbox service returned status %d: %s", resp.StatusCode, string(b))
	}

	var result SandboxResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode SandboxResult: %w", err)
	}

	c.logger.Info("Sandbox detonation completed",
		"scan_id", result.ScanID,
		"status", result.Status,
		"exit_code", result.ExitCode,
		"duration_ms", result.DurationMs,
	)

	return &result, nil
}
