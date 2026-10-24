package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"lucid-ci/engine/rules"
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
			Runtime:  "runsc",
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

func TestPipelineHandler_StrictSandboxFailure_Unverified(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &mockStore{}

	aiClient := &mockAIClient{
		resp: &worker.RemediationResponse{
			SuggestedPatch: "const safe = true;",
			Explanation:    "Candidate patch",
		},
	}

	sandboxClient := &mockSandboxClient{
		result: &worker.SandboxResult{
			Status:   "FAILED",
			ExitCode: 1,
			Stderr:   "Strict runtime requirement failed: runtime 'runsc' could not start",
			Runtime:  "runsc",
		},
	}

	handler := NewPipelineHandler(logger, store, aiClient, sandboxClient, nil, nil)
	task := models.ScanTaskMessage{
		TaskID:         "strict-fail-task-001",
		RepositoryID:  12345,
		RepositoryName: "acme/lucid-ci",
	}

	err := handler(context.Background(), task)
	if err != nil {
		t.Fatalf("Expected pipeline to succeed despite strict sandbox failure, got: %v", err)
	}

	vulns := store.insertVulnsCalls[0].vulns
	for i, v := range vulns {
		if v.SandboxVerified {
			t.Errorf("Vuln %d (%s): expected SandboxVerified = false when strict runtime fails", i, v.RuleID)
		}
	}
}

func TestPipelineHandler_SandboxPassedNonZeroExitCode_Unverified(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &mockStore{}

	aiClient := &mockAIClient{
		resp: &worker.RemediationResponse{
			SuggestedPatch: "const safe = true;",
			Explanation:    "Candidate patch",
		},
	}

	// Status claims PASSED but exit code is non-zero (invalid execution evidence)
	sandboxClient := &mockSandboxClient{
		result: &worker.SandboxResult{
			Status:   "PASSED",
			ExitCode: 1,
			Runtime:  "runsc",
		},
	}

	handler := NewPipelineHandler(logger, store, aiClient, sandboxClient, nil, nil)
	task := models.ScanTaskMessage{
		TaskID:         "anomaly-sandbox-task-002",
		RepositoryID:  12345,
		RepositoryName: "acme/lucid-ci",
	}

	err := handler(context.Background(), task)
	if err != nil {
		t.Fatalf("Expected pipeline to succeed, got: %v", err)
	}

	vulns := store.insertVulnsCalls[0].vulns
	for i, v := range vulns {
		if v.SandboxVerified {
			t.Errorf("Vuln %d (%s): expected SandboxVerified = false when exit code is non-zero despite PASSED status", i, v.RuleID)
		}
	}
}

func TestPipelineHandler_SandboxEgressAttempt_Unverified(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &mockStore{}

	aiClient := &mockAIClient{
		resp: &worker.RemediationResponse{
			SuggestedPatch: "const safe = true;",
			Explanation:    "Candidate patch",
		},
	}

	// Status claims PASSED and ExitCode is 0, but network egress attempts were detected
	sandboxClient := &mockSandboxClient{
		result: &worker.SandboxResult{
			Status:                "PASSED",
			ExitCode:              0,
			NetworkEgressAttempts: 2,
			Runtime:               "runsc",
		},
	}

	handler := NewPipelineHandler(logger, store, aiClient, sandboxClient, nil, nil)
	task := models.ScanTaskMessage{
		TaskID:         "egress-sandbox-task-003",
		RepositoryID:  12345,
		RepositoryName: "acme/lucid-ci",
	}

	err := handler(context.Background(), task)
	if err != nil {
		t.Fatalf("Expected pipeline to succeed, got: %v", err)
	}

	vulns := store.insertVulnsCalls[0].vulns
	for i, v := range vulns {
		if v.SandboxVerified {
			t.Errorf("Vuln %d (%s): expected SandboxVerified = false when network egress attempts > 0", i, v.RuleID)
		}
	}
}

func TestPipelineHandler_SandboxRuntimeRunc_Unverified(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &mockStore{}

	aiClient := &mockAIClient{
		resp: &worker.RemediationResponse{
			SuggestedPatch: "const safe = true;",
			Explanation:    "Candidate patch",
		},
	}

	// Invariant: Status PASSED and ExitCode 0, but runtime is runc (local dev fallback)
	// Must NOT mark sandbox_verified = true because gVisor isolation was not used.
	sandboxClient := &mockSandboxClient{
		result: &worker.SandboxResult{
			Status:                "PASSED",
			ExitCode:              0,
			NetworkEgressAttempts: 0,
			Runtime:               "runc",
		},
	}

	handler := NewPipelineHandler(logger, store, aiClient, sandboxClient, nil, nil)
	task := models.ScanTaskMessage{
		TaskID:         "runc-fallback-task-005",
		RepositoryID:  12345,
		RepositoryName: "acme/lucid-ci",
	}

	err := handler(context.Background(), task)
	if err != nil {
		t.Fatalf("Expected pipeline to succeed, got: %v", err)
	}

	vulns := store.insertVulnsCalls[0].vulns
	for i, v := range vulns {
		if v.SandboxVerified {
			t.Errorf("Vuln %d (%s): expected SandboxVerified = false when runtime is runc (must require runsc)", i, v.RuleID)
		}
	}
}

func TestPipelineHandler_SandboxRuntimeEmpty_Unverified(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &mockStore{}

	aiClient := &mockAIClient{
		resp: &worker.RemediationResponse{
			SuggestedPatch: "const safe = true;",
			Explanation:    "Candidate patch",
		},
	}

	// Invariant: Status PASSED and ExitCode 0, but runtime is empty string
	// Must NOT mark sandbox_verified = true without explicit runsc confirmation.
	sandboxClient := &mockSandboxClient{
		result: &worker.SandboxResult{
			Status:                "PASSED",
			ExitCode:              0,
			NetworkEgressAttempts: 0,
			Runtime:               "",
		},
	}

	handler := NewPipelineHandler(logger, store, aiClient, sandboxClient, nil, nil)
	task := models.ScanTaskMessage{
		TaskID:         "empty-runtime-task-006",
		RepositoryID:  12345,
		RepositoryName: "acme/lucid-ci",
	}

	err := handler(context.Background(), task)
	if err != nil {
		t.Fatalf("Expected pipeline to succeed, got: %v", err)
	}

	vulns := store.insertVulnsCalls[0].vulns
	for i, v := range vulns {
		if v.SandboxVerified {
			t.Errorf("Vuln %d (%s): expected SandboxVerified = false when runtime is empty", i, v.RuleID)
		}
	}
}

func TestPipelineHandler_SandboxRuntimeUnexpected_Unverified(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &mockStore{}

	aiClient := &mockAIClient{
		resp: &worker.RemediationResponse{
			SuggestedPatch: "const safe = true;",
			Explanation:    "Candidate patch",
		},
	}

	// Invariant: Status PASSED and ExitCode 0, but runtime is unexpected/unknown
	sandboxClient := &mockSandboxClient{
		result: &worker.SandboxResult{
			Status:                "PASSED",
			ExitCode:              0,
			NetworkEgressAttempts: 0,
			Runtime:               "crun",
		},
	}

	handler := NewPipelineHandler(logger, store, aiClient, sandboxClient, nil, nil)
	task := models.ScanTaskMessage{
		TaskID:         "custom-runtime-task-007",
		RepositoryID:  12345,
		RepositoryName: "acme/lucid-ci",
	}

	err := handler(context.Background(), task)
	if err != nil {
		t.Fatalf("Expected pipeline to succeed, got: %v", err)
	}

	vulns := store.insertVulnsCalls[0].vulns
	for i, v := range vulns {
		if v.SandboxVerified {
			t.Errorf("Vuln %d (%s): expected SandboxVerified = false when runtime is unexpected (%s)", i, v.RuleID, sandboxClient.result.Runtime)
		}
	}
}

func TestPipelineHandler_SandboxNilResult_Unverified(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &mockStore{}

	aiClient := &mockAIClient{
		resp: &worker.RemediationResponse{
			SuggestedPatch: "const safe = true;",
			Explanation:    "Candidate patch",
		},
	}

	// Sandbox client returns nil result with nil error
	sandboxClient := &mockSandboxClient{
		result: nil,
		err:    nil,
	}

	handler := NewPipelineHandler(logger, store, aiClient, sandboxClient, nil, nil)
	task := models.ScanTaskMessage{
		TaskID:         "nil-sandbox-task-004",
		RepositoryID:  12345,
		RepositoryName: "acme/lucid-ci",
	}

	// Must not panic on nil pointer dereference
	err := handler(context.Background(), task)
	if err != nil {
		t.Fatalf("Expected pipeline to succeed without panic, got: %v", err)
	}

	vulns := store.insertVulnsCalls[0].vulns
	for i, v := range vulns {
		if v.SandboxVerified {
			t.Errorf("Vuln %d (%s): expected SandboxVerified = false when sandbox result is nil", i, v.RuleID)
		}
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
	if scan.FindingsCount != 4 {
		t.Errorf("Expected findings_count 4, got %d", scan.FindingsCount)
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
	if len(vulns) != 4 {
		t.Fatalf("Expected 4 vulnerabilities, got %d", len(vulns))
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

func TestApplySuggestedPatch_DeclaratorConst(t *testing.T) {
	content := "const AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n"
	vulnCode := "AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\""
	patch := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"

	result := applySuggestedPatch(content, vulnCode, patch, 1, 1)
	expected := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n"

	if result != expected {
		t.Fatalf("expected:\n%q\ngot:\n%q", expected, result)
	}
	if strings.Contains(result, "const const") {
		t.Errorf("result contains duplicate declaration keyword 'const const'")
	}
	if strings.Contains(result, ";;") {
		t.Errorf("result contains duplicate semicolons ';;'")
	}
}

func TestApplySuggestedPatch_DeclaratorLetAndVar(t *testing.T) {
	vulnCode := "AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\""
	patchConst := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"
	patchLet := "let AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"
	patchVar := "var AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"

	tests := []struct {
		name     string
		input    string
		patch    string
		expected string
	}{
		{
			name:     "let replaced with const patch",
			input:    "let AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n",
			patch:    patchConst,
			expected: "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n",
		},
		{
			name:     "let replaced with let patch",
			input:    "let AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n",
			patch:    patchLet,
			expected: "let AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n",
		},
		{
			name:     "var replaced with const patch",
			input:    "var AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n",
			patch:    patchConst,
			expected: "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n",
		},
		{
			name:     "var replaced with var patch",
			input:    "var AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n",
			patch:    patchVar,
			expected: "var AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := applySuggestedPatch(tt.input, vulnCode, tt.patch, 1, 1)
			if res != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, res)
			}
			if strings.Contains(res, "let const") || strings.Contains(res, "var const") ||
				strings.Contains(res, "let let") || strings.Contains(res, "var var") ||
				strings.Contains(res, "const const") {
				t.Errorf("unexpected duplicate keyword in: %q", res)
			}
			if strings.Contains(res, ";;") {
				t.Errorf("duplicate semicolon in: %q", res)
			}
		})
	}
}

func TestApplySuggestedPatch_KeywordsAndSemicolonsNoDuplication(t *testing.T) {
	vulnCode := "AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\""

	t.Run("existing semicolon with patch having semicolon", func(t *testing.T) {
		content := "  const AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n"
		patch := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"
		res := applySuggestedPatch(content, vulnCode, patch, 1, 1)
		expected := "  const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n"
		if res != expected {
			t.Errorf("expected %q, got %q", expected, res)
		}
	})

	t.Run("existing semicolon with patch lacking semicolon", func(t *testing.T) {
		content := "const AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n"
		patch := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY"
		res := applySuggestedPatch(content, vulnCode, patch, 1, 1)
		expected := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n"
		if res != expected {
			t.Errorf("expected %q, got %q", expected, res)
		}
	})

	t.Run("no existing semicolon with patch having semicolon", func(t *testing.T) {
		content := "const AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\"\n"
		patch := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"
		res := applySuggestedPatch(content, vulnCode, patch, 1, 1)
		expected := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n"
		if res != expected {
			t.Errorf("expected %q, got %q", expected, res)
		}
	})

	t.Run("declarator patch without keyword preserves existing keyword", func(t *testing.T) {
		content := "const AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n"
		patch := "AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"
		res := applySuggestedPatch(content, vulnCode, patch, 1, 1)
		expected := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n"
		if res != expected {
			t.Errorf("expected %q, got %q", expected, res)
		}
	})
}

func TestApplySuggestedPatch_OriginalFailingSecretFixture(t *testing.T) {
	fixtureContent := `const express = require('express');
const app = express();
app.get('/user', (req, res) => {
  const id = req.query.id;
  const q = "SELECT * FROM users WHERE id = " + id;
  db.query(q);
});
app.get('/ping', (req, res) => {
  const host = req.query.host;
  const cmd = "ping -c 1 " + host;
  child_process.exec(cmd);
});
app.get('/download', (req, res) => {
  const file = req.query.file;
  fs.readFileSync("/var/www/" + file);
});
const AWS_SECRET_KEY = "AKIA1111111111EXAMPLE";
`
	// Run actual scanner to get real engine finding
	findings, err := rules.DetectInsecureSecrets("app.js", []byte(fixtureContent))
	if err != nil {
		t.Fatalf("DetectInsecureSecrets: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 secret finding, got %d", len(findings))
	}

	secretFinding := findings[0]
	if secretFinding.VulnerableCode != "AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\"" {
		t.Fatalf("unexpected VulnerableCode: %s", secretFinding.VulnerableCode)
	}

	llmPatch := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"
	patched := applySuggestedPatch(fixtureContent, secretFinding.VulnerableCode, llmPatch, secretFinding.LineStart, secretFinding.LineEnd)

	if strings.Contains(patched, "const const") {
		t.Errorf("patched content contains 'const const':\n%s", patched)
	}
	if strings.Contains(patched, ";;") {
		t.Errorf("patched content contains ';;':\n%s", patched)
	}
	if !strings.Contains(patched, "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;") {
		t.Errorf("patched content missing expected patch line:\n%s", patched)
	}

	// Verify all other parts of app.js remain intact
	if !strings.Contains(patched, "const express = require('express');") {
		t.Errorf("express require statement was damaged")
	}
	if !strings.Contains(patched, "db.query(q);") {
		t.Errorf("db.query statement was damaged")
	}
	if !strings.Contains(patched, "child_process.exec(cmd);") {
		t.Errorf("child_process.exec statement was damaged")
	}
	if !strings.Contains(patched, "fs.readFileSync(\"/var/www/\" + file);") {
		t.Errorf("fs.readFileSync statement was damaged")
	}
}

func TestApplySuggestedPatch_NegativeUnrelatedDeclarationsUnchanged(t *testing.T) {
	fixture := `const express = require('express');
const app = express();
const otherSecret = "AKIA2222222222EXAMPLE";
let counter = 10;
const AWS_SECRET_KEY = "AKIA1111111111EXAMPLE";
var isReady = true;
`
	vulnCode := "AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\""
	patch := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"

	patched := applySuggestedPatch(fixture, vulnCode, patch, 5, 5)

	expected := `const express = require('express');
const app = express();
const otherSecret = "AKIA2222222222EXAMPLE";
let counter = 10;
const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;
var isReady = true;
`
	if patched != expected {
		t.Fatalf("unrelated declarations were modified!\nExpected:\n%s\nGot:\n%s", expected, patched)
	}
}

func TestApplySuggestedPatch_MultilineDeclarationKeyword(t *testing.T) {
	content := "const\n  AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n"
	vulnCode := "AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\""
	patch := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"

	res := applySuggestedPatch(content, vulnCode, patch, 2, 2)
	expected := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n"

	if res != expected {
		t.Fatalf("expected:\n%q\ngot:\n%q", expected, res)
	}
	if strings.Contains(res, "const const") || strings.Contains(res, "const\n  const") {
		t.Errorf("duplicate declaration keyword found across lines: %q", res)
	}
}

func TestApplySuggestedPatch_CompoundDeclarations(t *testing.T) {
	vulnCode := "AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\""
	patch := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"

	t.Run("secret is second in list", func(t *testing.T) {
		content := "const a = 1, AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n"
		res := applySuggestedPatch(content, vulnCode, patch, 1, 1)
		expected := "const a = 1;\nconst AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n"
		if res != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, res)
		}
		if strings.Contains(res, ", const") || strings.Contains(res, ",const") {
			t.Errorf("invalid compound syntax in: %q", res)
		}
	})

	t.Run("secret is first in list", func(t *testing.T) {
		content := "const AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\", b = 2;\n"
		res := applySuggestedPatch(content, vulnCode, patch, 1, 1)
		expected := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\nconst b = 2;\n"
		if res != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, res)
		}
		if strings.Contains(res, ";,") || strings.Contains(res, "; ,") {
			t.Errorf("invalid trailing comma in: %q", res)
		}
	})

	t.Run("secret is in middle of list", func(t *testing.T) {
		content := "let a = 1, AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\", b = 2;\n"
		res := applySuggestedPatch(content, vulnCode, patch, 1, 1)
		expected := "let a = 1;\nconst AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\nlet b = 2;\n"
		if res != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, res)
		}
	})
}

func TestApplySuggestedPatch_ExportPreservation(t *testing.T) {
	vulnCode := "AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\""

	t.Run("export preserved when patch lacks export", func(t *testing.T) {
		content := "export const AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n"
		patch := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"
		res := applySuggestedPatch(content, vulnCode, patch, 1, 1)
		expected := "export const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n"
		if res != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, res)
		}
	})

	t.Run("no duplicate export when patch includes export", func(t *testing.T) {
		content := "export const AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n"
		patch := "export const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"
		res := applySuggestedPatch(content, vulnCode, patch, 1, 1)
		expected := "export const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n"
		if res != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, res)
		}
		if strings.Contains(res, "export export") {
			t.Errorf("duplicate export in: %q", res)
		}
	})
}

func TestApplySuggestedPatch_IndentationPreservation(t *testing.T) {
	vulnCode := "AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\""

	t.Run("source has 4 spaces, patch has 0 spaces", func(t *testing.T) {
		content := "    const AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n"
		patch := "const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"
		res := applySuggestedPatch(content, vulnCode, patch, 1, 1)
		expected := "    const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n"
		if res != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, res)
		}
	})

	t.Run("source has 4 spaces, patch has 2 spaces", func(t *testing.T) {
		content := "    const AWS_SECRET_KEY = \"AKIA1111111111EXAMPLE\";\n"
		patch := "  const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;"
		res := applySuggestedPatch(content, vulnCode, patch, 1, 1)
		expected := "    const AWS_SECRET_KEY = process.env.AWS_SECRET_KEY;\n"
		if res != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, res)
		}
	})
}

func TestApplySuggestedPatch_PrecedingDeclarationReplaced(t *testing.T) {
	content := `app.get('/user', (req, res) => {
  const id = req.query.id;
  const q = "SELECT * FROM users WHERE id = " + id;
  db.query(q);
});`
	vulnCode := "db.query(q)"
	patch := `  const q = "SELECT * FROM users WHERE id = ?";
  db.query(q, [id]);`

	res := applySuggestedPatch(content, vulnCode, patch, 4, 4)

	if strings.Contains(res, "const q = \"SELECT * FROM users WHERE id = \" + id;") {
		t.Errorf("old vulnerable query declaration was not replaced: %s", res)
	}
	if strings.Count(res, "const q =") > 1 {
		t.Errorf("duplicate declaration of const q: %s", res)
	}
	if !strings.Contains(res, "db.query(q, [id]);") {
		t.Errorf("remediated query call missing: %s", res)
	}
}
