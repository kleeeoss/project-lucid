package models

import "time"

// GitHubWebhookPayload represents the Pull Request webhook event (Contract CTR-001).
type GitHubWebhookPayload struct {
	Action       string             `json:"action"` // e.g., "opened", "synchronize", "reopened"
	Number       int                `json:"number"`
	PullRequest  PullRequestDetails `json:"pull_request"`
	Repository   RepositoryDetails  `json:"repository"`
	Installation InstallationRef    `json:"installation"`
	Sender       SenderRef          `json:"sender"`
}

type PullRequestDetails struct {
	ID        int64     `json:"id"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Head      GitCommit `json:"head"`
	Base      GitCommit `json:"base"`
	DiffURL   string    `json:"diff_url"`
	PatchURL  string    `json:"patch_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GitCommit struct {
	SHA  string  `json:"sha"`
	Ref  string  `json:"ref"`
	Repo RepoRef `json:"repo"`
}

type RepoRef struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
}

type RepositoryDetails struct {
	ID            int64  `json:"id"`
	FullName      string `json:"full_name"` // e.g., "owner/repo"
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
}

type InstallationRef struct {
	ID int64 `json:"id"`
}

type SenderRef struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

func (p *GitHubWebhookPayload) IsValidPRAction() bool {
	switch p.Action {
	case "opened", "synchronize", "reopened":
		return true
	default:
		return false
	}
}

func (p *GitHubWebhookPayload) ToSQSMessage(taskID string) ScanTaskMessage {
	return ScanTaskMessage{
		TaskID:         taskID,
		InstallationID: p.Installation.ID,
		RepositoryID:   p.Repository.ID,
		RepositoryName: p.Repository.FullName,
		PRNumber:       p.Number,
		CommitSHA:      p.PullRequest.Head.SHA,
		BaseSHA:        p.PullRequest.Base.SHA,
		DiffURL:        p.PullRequest.DiffURL,
		TriggerAction:  p.Action,
		EnqueuedAt:     time.Now().UTC(),
		AttemptCount:   0,
	}
}
