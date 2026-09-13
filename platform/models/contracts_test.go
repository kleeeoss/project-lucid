package models

import (
	"encoding/json"
	"testing"
)

func TestGitHubWebhookPayload_IsValidPRAction(t *testing.T) {
	tests := []struct {
		action   string
		expected bool
	}{
		{"opened", true},
		{"synchronize", true},
		{"reopened", true},
		{"closed", false},
		{"labeled", false},
		{"edited", false},
		{"assigned", false},
	}

	for _, tt := range tests {
		p := GitHubWebhookPayload{Action: tt.action}
		if got := p.IsValidPRAction(); got != tt.expected {
			t.Errorf("Action %q: expected %v, got %v", tt.action, tt.expected, got)
		}
	}
}

func TestGitHubWebhookPayload_ToSQSMessage(t *testing.T) {
	rawJSON := `{
		"action": "opened",
		"number": 10,
		"pull_request": {
			"head": {"sha": "abc1234"},
			"base": {"sha": "def5678"},
			"diff_url": "https://github.com/org/repo/pull/10.diff"
		},
		"repository": {
			"id": 999,
			"full_name": "org/repo"
		},
		"installation": {
			"id": 1234
		}
	}`

	var payload GitHubWebhookPayload
	if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	taskID := "test-task-123"
	msg := payload.ToSQSMessage(taskID)

	if msg.TaskID != taskID {
		t.Errorf("Expected TaskID %q, got %q", taskID, msg.TaskID)
	}
	if msg.RepositoryID != 999 {
		t.Errorf("Expected RepositoryID 999, got %d", msg.RepositoryID)
	}
	if msg.RepositoryName != "org/repo" {
		t.Errorf("Expected RepositoryName 'org/repo', got %q", msg.RepositoryName)
	}
	if msg.PRNumber != 10 {
		t.Errorf("Expected PRNumber 10, got %d", msg.PRNumber)
	}
	if msg.CommitSHA != "abc1234" {
		t.Errorf("Expected CommitSHA 'abc1234', got %q", msg.CommitSHA)
	}
	if msg.TriggerAction != "opened" {
		t.Errorf("Expected TriggerAction 'opened', got %q", msg.TriggerAction)
	}
}
