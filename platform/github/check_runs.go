package github

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

// CheckRunAnnotation represents a line-level finding attached to a Check Run.
type CheckRunAnnotation struct {
	Path            string `json:"path"`
	StartLine       int    `json:"start_line"`
	EndLine         int    `json:"end_line"`
	AnnotationLevel string `json:"annotation_level"` // "notice", "warning", "failure"
	Message         string `json:"message"`
	Title           string `json:"title"`
	RawDetails      string `json:"raw_details,omitempty"`
}

// CheckRunOutput contains the markdown summary and annotations for a Check Run.
type CheckRunOutput struct {
	Title       string               `json:"title"`
	Summary     string               `json:"summary"`
	Text        string               `json:"text,omitempty"`
	Annotations []CheckRunAnnotation `json:"annotations,omitempty"`
}

// CheckRunClient defines methods for interacting with GitHub's Checks and Reviews APIs.
type CheckRunClient interface {
	CreateCheckRun(ctx context.Context, token, owner, repo, headSHA, name string) (int64, error)
	UpdateCheckRun(ctx context.Context, token, owner, repo string, checkRunID int64, conclusion string, output CheckRunOutput) error
	PostPRReviewComment(ctx context.Context, token, owner, repo string, prNumber int, commitSHA, filePath string, line int, commentBody string) error
}

type checkRunClient struct {
	apiBaseURL string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewCheckRunClient(apiBaseURL string, logger *slog.Logger) CheckRunClient {
	if apiBaseURL == "" {
		apiBaseURL = "https://api.github.com"
	}
	return &checkRunClient{
		apiBaseURL: apiBaseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		logger:     logger,
	}
}

type createCheckRunRequest struct {
	Name      string    `json:"name"`
	HeadSHA   string    `json:"head_sha"`
	Status    string    `json:"status"` // "in_progress"
	StartedAt time.Time `json:"started_at"`
}

type checkRunResponse struct {
	ID int64 `json:"id"`
}

func (c *checkRunClient) CreateCheckRun(ctx context.Context, token, owner, repo, headSHA, name string) (int64, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/check-runs", c.apiBaseURL, owner, repo)

	reqBody := createCheckRunRequest{
		Name:      name,
		HeadSHA:   headSHA,
		Status:    "in_progress",
		StartedAt: time.Now().UTC(),
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal create check run body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, fmt.Errorf("failed to create check run request: %w", err)
	}

	c.setHeaders(req, token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("CreateCheckRun network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("CreateCheckRun failed with status %d: %s", resp.StatusCode, string(b))
	}

	var crResp checkRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&crResp); err != nil {
		return 0, fmt.Errorf("failed to decode check run response: %w", err)
	}

	c.logger.Info("Created GitHub Check Run",
		"check_run_id", crResp.ID,
		"repo", owner+"/"+repo,
		"head_sha", headSHA,
	)

	return crResp.ID, nil
}

type updateCheckRunRequest struct {
	Status      string          `json:"status"` // "completed"
	Conclusion  string          `json:"conclusion,omitempty"`
	CompletedAt time.Time       `json:"completed_at"`
	Output      *CheckRunOutput `json:"output,omitempty"`
}

func (c *checkRunClient) UpdateCheckRun(ctx context.Context, token, owner, repo string, checkRunID int64, conclusion string, output CheckRunOutput) error {
	url := fmt.Sprintf("%s/repos/%s/%s/check-runs/%d", c.apiBaseURL, owner, repo, checkRunID)

	allAnnotations := output.Annotations
	batchSize := 50 // GitHub API limit: max 50 annotations per call

	// If 50 or fewer annotations, update in a single call
	if len(allAnnotations) <= batchSize {
		return c.sendUpdateCheckRun(ctx, token, url, conclusion, &output)
	}

	// For > 50 annotations: send initial batch with status completed, then append remaining batches
	initialOutput := output
	initialOutput.Annotations = allAnnotations[:batchSize]

	if err := c.sendUpdateCheckRun(ctx, token, url, conclusion, &initialOutput); err != nil {
		return err
	}

	// Send remaining annotations in subsequent batches
	for i := batchSize; i < len(allAnnotations); i += batchSize {
		end := i + batchSize
		if end > len(allAnnotations) {
			end = len(allAnnotations)
		}

		batchOutput := CheckRunOutput{
			Title:       output.Title,
			Summary:     output.Summary,
			Annotations: allAnnotations[i:end],
		}

		if err := c.sendUpdateCheckRun(ctx, token, url, conclusion, &batchOutput); err != nil {
			c.logger.Warn("Failed to send chunked annotations batch", "start", i, "end", end, "error", err)
			return err
		}
	}

	return nil
}

func (c *checkRunClient) sendUpdateCheckRun(ctx context.Context, token, url, conclusion string, output *CheckRunOutput) error {
	reqBody := updateCheckRunRequest{
		Status:      "completed",
		Conclusion:  conclusion,
		CompletedAt: time.Now().UTC(),
		Output:      output,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal update check run body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create update check run request: %w", err)
	}

	c.setHeaders(req, token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("UpdateCheckRun network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("UpdateCheckRun failed with status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

type prCommentRequest struct {
	Body     string `json:"body"`
	CommitID string `json:"commit_id"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Side     string `json:"side"` // "RIGHT" for added/modified lines in PR diff
}

func (c *checkRunClient) PostPRReviewComment(ctx context.Context, token, owner, repo string, prNumber int, commitSHA, filePath string, line int, commentBody string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/comments", c.apiBaseURL, owner, repo, prNumber)

	reqBody := prCommentRequest{
		Body:     commentBody,
		CommitID: commitSHA,
		Path:     filePath,
		Line:     line,
		Side:     "RIGHT",
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal PR comment body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create PR comment request: %w", err)
	}

	c.setHeaders(req, token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("PostPRReviewComment network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PostPRReviewComment failed with status %d: %s", resp.StatusCode, string(b))
	}

	c.logger.Info("Posted inline PR suggestion comment",
		"repo", owner+"/"+repo,
		"pr", prNumber,
		"file", filePath,
		"line", line,
	)

	return nil
}

func (c *checkRunClient) setHeaders(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
}
