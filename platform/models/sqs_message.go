package models

import "time"

// ScanTaskMessage represents the normalized asynchronous job queued on AWS SQS (Contract CTR-002).
type ScanTaskMessage struct {
	TaskID         string    `json:"task_id"`
	InstallationID int64     `json:"installation_id"`
	RepositoryID   int64     `json:"repository_id"`
	RepositoryName string    `json:"repository_name"`
	PRNumber       int       `json:"pr_number"`
	CommitSHA      string    `json:"commit_sha"`
	BaseSHA        string    `json:"base_sha"`
	DiffURL        string    `json:"diff_url"`
	TriggerAction  string    `json:"trigger_action"`
	EnqueuedAt     time.Time `json:"enqueued_at"`
	AttemptCount   int       `json:"attempt_count"`
}
