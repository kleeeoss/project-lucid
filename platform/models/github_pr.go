package models

// WebhookPayload represents the root JSON sent by GitHub.
type WebhookPayload struct {
	Action      string      `json:"action"`
	PullRequest PullRequest `json:"pull_request"`
	Repository  Repository  `json:"repository"`
}

// PullRequest drills down into the PR details.
type PullRequest struct {
	DiffURL string `json:"diff_url"`
	Head    Head   `json:"head"`
}

// Head captures the specific commit hash.
type Head struct {
	SHA string `json:"sha"`
}

// Repository captures the repo name.
type Repository struct {
	FullName string `json:"full_name"`
}
