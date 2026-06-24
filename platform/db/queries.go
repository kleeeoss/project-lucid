package db

import (
	"log"
	"lucid-platform/models"
	"time"
)

// InsertScan saves a new incoming PR scan to the database
func InsertScan(payload models.WebhookPayload) {
	query := `
	INSERT INTO scan_results (repo_name, commit_sha, status, created_at)
	VALUES ($1, $2, $3, $4)
	`

	// Default status is "pending"
	_, err := DB.Exec(query, payload.Repository.FullName, payload.PullRequest.Head.SHA, "pending", time.Now())
	if err != nil {
		log.Printf("❌ Failed to insert scan result: %v\n", err)
	} else {
		log.Println("📥 Successfully saved PR to database as 'pending'!")
	}
}
func UpdateScanStatus(commitSHA string, status string) {
	query := `UPDATE scan_results SET status = $1 WHERE commit_sha = $2`

	_, err := DB.Exec(query, status, commitSHA)
	if err != nil {
		log.Printf("❌ Failed to update scan status: %v\n", err)
	} else {
		log.Printf("🔄 PR %s updated to '%s'!\n", commitSHA[:7], status) // Shortened SHA for clean logs
	}
}
