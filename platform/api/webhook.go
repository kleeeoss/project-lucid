package api

import (
	"encoding/json"
	"fmt"
	"lucid-platform/db"
	"lucid-platform/engine"
	"lucid-platform/models"
	"net/http"
)

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload models.WebhookPayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	fmt.Println("\n--- New GitHub Webhook Received ---")
	fmt.Printf("Repository: %s\n", payload.Repository.FullName)
	fmt.Printf("Action:     %s\n", payload.Action)
	fmt.Printf("Commit SHA: %s\n", payload.PullRequest.Head.SHA)
	fmt.Printf("Diff URL:   %s\n", payload.PullRequest.DiffURL)
	fmt.Println("-----------------------------------")

	db.InsertScan(payload)

	go engine.RunScan(payload)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received, parsed, and saved to database!\n"))
}
