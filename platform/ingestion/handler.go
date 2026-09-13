package ingestion

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"lucid-ci/platform/models"
)

// WebhookHandler processes incoming GitHub App webhooks.
type WebhookHandler struct {
	logger *slog.Logger
}

func NewWebhookHandler(logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		logger: logger,
	}
}

func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit payload to 10MB to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read webhook body", "error", err)
		http.Error(w, "Payload too large or read error", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	eventType := r.Header.Get("X-GitHub-Event")
	deliveryID := r.Header.Get("X-GitHub-Delivery")

	h.logger.Info("Received GitHub webhook",
		"event_type", eventType,
		"delivery_id", deliveryID,
	)

	// In Phase 1 mock: We only process pull_request events
	if eventType != "pull_request" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ignored",
			"message": fmt.Sprintf("Event type '%s' is not processed", eventType),
		})
		return
	}

	var payload models.GitHubWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Warn("Failed to unmarshal GitHub PR payload", "error", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Check if this action triggers a scan
	if !payload.IsValidPRAction() {
		h.logger.Info("PR action ignored",
			"action", payload.Action,
			"pr", payload.Number,
			"repo", payload.Repository.FullName,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ignored",
			"message": fmt.Sprintf("Action '%s' does not trigger analysis", payload.Action),
		})
		return
	}

	// Generate task UUID for tracking
	taskID := generateUUID()
	scanMessage := payload.ToSQSMessage(taskID)

	h.logger.Info("Scan task created (Phase 1 mock queued)",
		"task_id", taskID,
		"repo", scanMessage.RepositoryName,
		"pr", scanMessage.PRNumber,
		"commit_sha", scanMessage.CommitSHA,
	)

	// Phase 1: Return 202 Accepted with the generated task_id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     "queued",
		"task_id":    taskID,
		"repository": scanMessage.RepositoryName,
		"pr_number":  scanMessage.PRNumber,
		"commit_sha": scanMessage.CommitSHA,
	})
}

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
