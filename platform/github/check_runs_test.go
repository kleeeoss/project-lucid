package github

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestCheckRunClient_CreateCheckRun(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test_token_123" {
			t.Errorf("missing or invalid authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(checkRunResponse{ID: 777})
	}))
	defer mockServer.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewCheckRunClient(mockServer.URL, logger)

	id, err := client.CreateCheckRun(context.Background(), "test_token_123", "owner", "repo", "sha123", "Lucid-CI")
	if err != nil {
		t.Fatalf("CreateCheckRun failed: %v", err)
	}
	if id != 777 {
		t.Errorf("expected CheckRun ID 777, got %d", id)
	}
}

func TestCheckRunClient_UpdateCheckRun_Batching(t *testing.T) {
	var patchCount atomic.Int32
	var totalAnnotationsReceived atomic.Int32

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		var body updateCheckRunRequest
		_ = json.NewDecoder(r.Body).Decode(&body)

		if body.Output != nil {
			totalAnnotationsReceived.Add(int32(len(body.Output.Annotations)))
		}

		patchCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewCheckRunClient(mockServer.URL, logger)

	// Create 75 test annotations (exceeds GitHub's 50-item limit per call)
	annotations := make([]CheckRunAnnotation, 75)
	for i := 0; i < 75; i++ {
		annotations[i] = CheckRunAnnotation{
			Path:            "main.go",
			StartLine:       i + 1,
			EndLine:         i + 1,
			AnnotationLevel: "failure",
			Message:         "test vulnerability",
			Title:           "CWE-89",
		}
	}

	output := CheckRunOutput{
		Title:       "Scan Complete",
		Summary:     "Found 75 issues",
		Annotations: annotations,
	}

	err := client.UpdateCheckRun(context.Background(), "test_token_123", "owner", "repo", 777, "failure", output)
	if err != nil {
		t.Fatalf("UpdateCheckRun failed: %v", err)
	}

	// 75 annotations should be split into 2 PATCH calls: 50 + 25
	if count := patchCount.Load(); count != 2 {
		t.Errorf("expected 2 PATCH requests due to chunking, got %d", count)
	}
	if total := totalAnnotationsReceived.Load(); total != 75 {
		t.Errorf("expected 75 total annotations across batches, got %d", total)
	}
}

func TestCheckRunClient_PostPRReviewComment(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var req prCommentRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Line != 42 || req.Path != "src/auth.js" {
			t.Errorf("unexpected comment target: %s:%d", req.Path, req.Line)
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer mockServer.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewCheckRunClient(mockServer.URL, logger)

	suggestion := "```suggestion\nconst safe = true;\n```"
	err := client.PostPRReviewComment(context.Background(), "token", "owner", "repo", 10, "sha123", "src/auth.js", 42, suggestion)
	if err != nil {
		t.Fatalf("PostPRReviewComment failed: %v", err)
	}
}
