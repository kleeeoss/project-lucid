package ingestion

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"lucid-ci/platform/models"
)

// WebhookHandler processes incoming GitHub App webhooks and enqueues scan jobs.
type WebhookHandler struct {
	logger      *slog.Logger
	secret      string
	sqsProducer SQSProducer
}

func NewWebhookHandler(logger *slog.Logger, producer SQSProducer) *WebhookHandler {
	secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if secret == "" {
		logger.Warn("GITHUB_WEBHOOK_SECRET is not set; webhook signature verification will be skipped for development")
	}
	return &WebhookHandler{
		logger:      logger,
		secret:      secret,
		sqsProducer: producer,
	}
}

func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit payload to 10MB to prevent memory exhaustion DDoS
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read webhook body", "error", err)
		http.Error(w, "Payload too large or read error", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// SEC-001 & SEC-002: Verify HMAC-SHA256 signature if secret is configured
	if h.secret != "" {
		sigHeader := r.Header.Get("X-Hub-Signature-256")
		if err := VerifySignature(body, sigHeader, h.secret); err != nil {
			h.logger.Warn("Unauthorized webhook delivery: signature verification failed",
				"error", err,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "Unauthorized: signature verification failed", http.StatusUnauthorized)
			return
		}
	}

	eventType := r.Header.Get("X-GitHub-Event")
	deliveryID := r.Header.Get("X-GitHub-Delivery")

	h.logger.Info("Received verified GitHub webhook",
		"event_type", eventType,
		"delivery_id", deliveryID,
	)

	// Process only pull_request events
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

	// Check if this action triggers a security scan
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

	var messageID string
	if h.sqsProducer != nil {
		mid, err := h.sqsProducer.PublishScanTask(r.Context(), scanMessage)
		if err != nil {
			h.logger.Error("Failed to enqueue scan task to SQS",
				"task_id", taskID,
				"error", err,
			)
			http.Error(w, "Failed to enqueue scan job", http.StatusInternalServerError)
			return
		}
		messageID = mid
	} else {
		h.logger.Info("SQS producer not configured, running in mock queue mode",
			"task_id", taskID,
		)
	}

	// Return 202 Accepted immediately to GitHub (< 300ms)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     "queued",
		"task_id":    taskID,
		"message_id": messageID,
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
