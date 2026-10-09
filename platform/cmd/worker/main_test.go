package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"lucid-ci/platform/db"
	"lucid-ci/platform/models"
	"lucid-ci/platform/worker"
)

type mockStore struct {
	db.Store
	mu                sync.Mutex
	upsertRepoCalls   []*models.Repository
	createScanCalls   []*models.ScanRun
	updateStatusCalls []statusUpdateCall
	insertVulnsCalls  []vulnInsertCall

	upsertRepoErr     error
	createScanErr     error
	updateStatusErr   error
	insertVulnsErr    error
}

type statusUpdateCall struct {
	scanID        string
	status        models.ScanStatus
	findingsCount int
	durationMs    *int
}

type vulnInsertCall struct {
	scanID string
	vulns  []models.Vulnerability
}

func (m *mockStore) UpsertRepository(ctx context.Context, repo *models.Repository) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upsertRepoCalls = append(m.upsertRepoCalls, repo)
	return m.upsertRepoErr
}

func (m *mockStore) CreateScanRun(ctx context.Context, scan *models.ScanRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createScanCalls = append(m.createScanCalls, scan)
	return m.createScanErr
}

func (m *mockStore) UpdateScanRunStatus(ctx context.Context, scanID string, status models.ScanStatus, findingsCount int, durationMs *int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateStatusCalls = append(m.updateStatusCalls, statusUpdateCall{
		scanID:        scanID,
		status:        status,
		findingsCount: findingsCount,
		durationMs:    durationMs,
	})
	return m.updateStatusErr
}

func (m *mockStore) UpdateCheckRunID(ctx context.Context, scanID string, checkRunID int64) error {
	return nil
}

func (m *mockStore) InsertVulnerabilities(ctx context.Context, scanID string, vulns []models.Vulnerability) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.insertVulnsCalls = append(m.insertVulnsCalls, vulnInsertCall{scanID: scanID, vulns: vulns})
	return m.insertVulnsErr
}

type mockAIClient struct {
	resp *worker.RemediationResponse
	err  error
}

func (m *mockAIClient) RemediateVulnerability(ctx context.Context, req worker.RemediationRequest) (*worker.RemediationResponse, error) {
	return m.resp, m.err
}

type mockSandboxClient struct {
	result *worker.SandboxResult
	err    error
}

func (m *mockSandboxClient) Detonate(ctx context.Context, req worker.SandboxRequest) (*worker.SandboxResult, error) {
	return m.result, m.err
}

func TestPipelineHandler_SandboxFailure_CompletesWithUnverifiedVulns(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &mockStore{}

	patch := "const safe = true;"
	aiClient := &mockAIClient{
		resp: &worker.RemediationResponse{
			SuggestedPatch: patch,
			Explanation:    "Safe patch",
			ModelName:      "test-model",
			Confidence:     0.9,
		},
	}

	// Simulates the exact user scenario: sandbox returned FAILED with exit code 254
	sandboxClient := &mockSandboxClient{
		result: &worker.SandboxResult{
			Status:   "FAILED",
			ExitCode: 254,
			Stderr:   "npm test failed with exit code 254",
		},
	}

	handler := NewPipelineHandler(logger, store, aiClient, sandboxClient, nil, nil)

	task := models.ScanTaskMessage{
		TaskID:         "96b01061d1f61bf06baaf26eab2634f5",
		RepositoryID:  12345,
		RepositoryName: "acme/lucid-ci",
		PRNumber:       42,
		CommitSHA:      "6dcb09b5",
	}

	err := handler(context.Background(), task)
	if err != nil {
		t.Fatalf("Expected pipeline to succeed despite sandbox test failure, got error: %v", err)
	}

	// 1. Verify Scan Run was created with SCANNING
	if len(store.createScanCalls) != 1 {
		t.Fatalf("Expected 1 CreateScanRun call, got %d", len(store.createScanCalls))
	}
	if store.createScanCalls[0].Status != models.ScanStatusScanning {
		t.Errorf("Expected initial status %s, got %s", models.ScanStatusScanning, store.createScanCalls[0].Status)
	}

	// 2. Verify vulnerabilities persisted with SandboxVerified = false
	if len(store.insertVulnsCalls) != 1 {
		t.Fatalf("Expected 1 InsertVulnerabilities call, got %d", len(store.insertVulnsCalls))
	}
	vulns := store.insertVulnsCalls[0].vulns
	if len(vulns) != 4 {
		t.Fatalf("Expected 4 persisted vulnerabilities, got %d", len(vulns))
	}
	ruleSet := make(map[string]bool)
	for i, v := range vulns {
		ruleSet[v.RuleID] = true
		if v.SandboxVerified {
			t.Errorf("Vuln %d (%s): expected SandboxVerified = false when sandbox returns exit code 254", i, v.RuleID)
		}
	}
	for _, expectedRule := range []string{"LUCID-SEC-001", "LUCID-SEC-002", "LUCID-SEC-003", "LUCID-SEC-004"} {
		if !ruleSet[expectedRule] {
			t.Errorf("Expected rule %s in persisted findings, but was missing", expectedRule)
		}
	}

	// 3. Verify final status transition to COMPLETED with correct findings count and duration
	if len(store.updateStatusCalls) == 0 {
		t.Fatalf("Expected UpdateScanRunStatus to be called")
	}
	finalCall := store.updateStatusCalls[len(store.updateStatusCalls)-1]
	if finalCall.status != models.ScanStatusCompleted {
		t.Errorf("Expected final status %s, got %s", models.ScanStatusCompleted, finalCall.status)
	}
	if finalCall.findingsCount != 4 {
		t.Errorf("Expected final findings_count 4, got %d", finalCall.findingsCount)
	}
	if finalCall.durationMs == nil || *finalCall.durationMs < 0 {
		t.Errorf("Expected durationMs to be recorded, got %v", finalCall.durationMs)
	}
}

func TestPipelineHandler_SandboxPassed_CompletesWithVerifiedVulns(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &mockStore{}

	aiClient := &mockAIClient{
		resp: &worker.RemediationResponse{
			SuggestedPatch: "const safe = true;",
			Explanation:    "Verified safe patch",
		},
	}

	sandboxClient := &mockSandboxClient{
		result: &worker.SandboxResult{
			Status:   "PASSED",
			ExitCode: 0,
		},
	}

	handler := NewPipelineHandler(logger, store, aiClient, sandboxClient, nil, nil)
	task := models.ScanTaskMessage{
		TaskID:         "passed-task-1111222233334444",
		RepositoryID:  12345,
		RepositoryName: "acme/lucid-ci",
	}

	err := handler(context.Background(), task)
	if err != nil {
		t.Fatalf("Expected pipeline to succeed, got error: %v", err)
	}

	vulns := store.insertVulnsCalls[0].vulns
	if len(vulns) != 4 {
		t.Fatalf("Expected 4 persisted vulnerabilities, got %d", len(vulns))
	}
	for i, v := range vulns {
		if !v.SandboxVerified {
			t.Errorf("Vuln %d: expected SandboxVerified = true when sandbox passes", i)
		}
	}

	finalCall := store.updateStatusCalls[len(store.updateStatusCalls)-1]
	if finalCall.status != models.ScanStatusCompleted || finalCall.findingsCount != 4 {
		t.Errorf("Expected COMPLETED with 4 findings, got status=%s findings=%d", finalCall.status, finalCall.findingsCount)
	}
}

func TestPipelineHandler_AIFailure_FailOpen(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &mockStore{}

	// Simulates missing API key / timeout (Fail-Open BR-002)
	aiClient := &mockAIClient{
		err: errors.New("Groq/Gemini API key not configured"),
	}

	handler := NewPipelineHandler(logger, store, aiClient, nil, nil, nil)
	task := models.ScanTaskMessage{
		TaskID:         "failopen-task-0001",
		RepositoryID:  12345,
		RepositoryName: "acme/lucid-ci",
	}

	err := handler(context.Background(), task)
	if err != nil {
		t.Fatalf("Expected pipeline to continue fail-open when AI fails, got error: %v", err)
	}

	finalCall := store.updateStatusCalls[len(store.updateStatusCalls)-1]
	if finalCall.status != models.ScanStatusCompleted || finalCall.findingsCount != 4 {
		t.Errorf("Expected COMPLETED with 4 findings, got status=%s findings=%d", finalCall.status, finalCall.findingsCount)
	}
}

func TestPipelineHandler_StoreErrors_Propagated(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	task := models.ScanTaskMessage{
		TaskID:         "err-task-0001",
		RepositoryID:  12345,
		RepositoryName: "acme/lucid-ci",
	}

	// 1. CreateScanRun error
	store1 := &mockStore{createScanErr: errors.New("db create error")}
	h1 := NewPipelineHandler(logger, store1, nil, nil, nil, nil)
	if err := h1(context.Background(), task); err == nil {
		t.Errorf("Expected error when CreateScanRun fails, got nil")
	}

	// 2. InsertVulnerabilities error
	store2 := &mockStore{insertVulnsErr: errors.New("db insert vulns error")}
	h2 := NewPipelineHandler(logger, store2, nil, nil, nil, nil)
	if err := h2(context.Background(), task); err == nil {
		t.Errorf("Expected error when InsertVulnerabilities fails, got nil")
	}

	// 3. Final UpdateScanRunStatus error
	store3 := &mockStore{updateStatusErr: errors.New("db update status error")}
	h3 := NewPipelineHandler(logger, store3, nil, nil, nil, nil)
	if err := h3(context.Background(), task); err == nil {
		t.Errorf("Expected error when UpdateScanRunStatus fails, got nil")
	}
}

func TestPipelineHandler_RealStore_Integration(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://lucid:lucid_dev@localhost:5432/lucid_ci?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	store, err := db.NewStore(ctx, dbURL)
	if err != nil {
		t.Skipf("PostgreSQL store not available at %s, skipping store integration test: %v", dbURL, err)
		return
	}
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Replicate the exact user scenario:
	// - AI fallback generates deterministic patch
	// - Sandbox returns FAILED with exit code 254
	aiClient := &mockAIClient{
		resp: &worker.RemediationResponse{
			SuggestedPatch: "const safe = true;",
			Explanation:    "Deterministic fallback patch",
			ModelName:      "deterministic-fallback",
			Confidence:     0.75,
		},
	}
	sandboxClient := &mockSandboxClient{
		result: &worker.SandboxResult{
			Status:   "FAILED",
			ExitCode: 254,
			Stderr:   "npm test failed with exit code 254",
		},
	}

	handler := NewPipelineHandler(logger, store, aiClient, sandboxClient, nil, nil)

	// Generate 32-char hex task ID (Contract CTR-002)
	taskID := fmt.Sprintf("e2e0%028x", time.Now().UnixNano()%1000000000000)
	task := models.ScanTaskMessage{
		TaskID:         taskID,
		RepositoryID:  12345,
		RepositoryName: "acme/lucid-ci",
		PRNumber:       42,
		CommitSHA:      "6dcb09b5",
	}

	err = handler(context.Background(), task)
	if err != nil {
		t.Fatalf("Pipeline handler failed: %v", err)
	}

	// 1. Verify scan_runs record in real PostgreSQL
	scan, err := store.GetScanRunByID(context.Background(), taskID)
	if err != nil {
		t.Fatalf("GetScanRunByID failed: %v", err)
	}
	if scan == nil {
		t.Fatalf("Scan run record %s not found in PostgreSQL", taskID)
	}
	if scan.Status != models.ScanStatusCompleted {
		t.Errorf("Expected final status %s, got %s", models.ScanStatusCompleted, scan.Status)
	}
	if scan.FindingsCount != 3 {
		t.Errorf("Expected findings_count 3, got %d", scan.FindingsCount)
	}
	if scan.CompletedAt == nil {
		t.Errorf("Expected completed_at to be populated")
	}
	if scan.ScanDurationMs == nil || *scan.ScanDurationMs < 0 {
		t.Errorf("Expected scan_duration_ms to be populated, got %v", scan.ScanDurationMs)
	}

	// 2. Verify vulnerabilities record in real PostgreSQL
	vulns, err := store.GetVulnerabilitiesByScanRunID(context.Background(), taskID)
	if err != nil {
		t.Fatalf("GetVulnerabilitiesByScanRunID failed: %v", err)
	}
	if len(vulns) != 3 {
		t.Fatalf("Expected 3 vulnerabilities, got %d", len(vulns))
	}
	for i, v := range vulns {
		if v.SandboxVerified {
			t.Errorf("Vuln %d: expected SandboxVerified = false when sandbox returns exit code 254", i)
		}
		if v.ScanRunID != scan.ID {
			t.Errorf("Vuln %d: expected scan_run_id %s, got %s", i, scan.ID, v.ScanRunID)
		}
	}
}

func TestPipelineHandler_RealStore_FailureLifecycle(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://lucid:lucid_dev@localhost:5432/lucid_ci?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	store, err := db.NewStore(ctx, dbURL)
	if err != nil {
		t.Skipf("PostgreSQL store not available at %s, skipping store integration test: %v", dbURL, err)
		return
	}
	defer store.Close()

	// Initial scan run created with findings_count = 3
	taskID := fmt.Sprintf("fa110%027x", time.Now().UnixNano()%1000000000000)
	scanRun := &models.ScanRun{
		ID:            taskID,
		RepositoryID:  12345,
		PRNumber:      42,
		CommitSHA:     "6dcb09b5",
		Status:        models.ScanStatusScanning,
		FindingsCount: 3,
	}
	if err := store.CreateScanRun(ctx, scanRun); err != nil {
		t.Fatalf("CreateScanRun failed: %v", err)
	}

	durationMs := 450
	// Transition to FAILED with findingsCount = 0 (as pool.go does on pipeline error)
	if err := store.UpdateScanRunStatus(ctx, taskID, models.ScanStatusFailed, 0, &durationMs); err != nil {
		t.Fatalf("UpdateScanRunStatus to FAILED failed: %v", err)
	}

	scan, err := store.GetScanRunByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetScanRunByID failed: %v", err)
	}
	if scan.Status != models.ScanStatusFailed {
		t.Errorf("Expected status %s, got %s", models.ScanStatusFailed, scan.Status)
	}
	if scan.FindingsCount != 3 {
		t.Errorf("Expected findings_count 3 to be preserved, got %d", scan.FindingsCount)
	}
	if scan.CompletedAt == nil {
		t.Errorf("Expected completed_at to be populated on failure")
	}
	if scan.ScanDurationMs == nil || *scan.ScanDurationMs != 450 {
		t.Errorf("Expected scan_duration_ms 450, got %v", scan.ScanDurationMs)
	}
}


