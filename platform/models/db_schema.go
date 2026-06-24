package models

import "time"

// ScanResult represents how we store the PR data in PostgreSQL
type ScanResult struct {
	ID        int       `json:"id"`
	RepoName  string    `json:"repo_name"`
	CommitSHA string    `json:"commit_sha"`
	Status    string    `json:"status"` // e.g., "pending", "passed", "failed"
	CreatedAt time.Time `json:"created_at"`
}
