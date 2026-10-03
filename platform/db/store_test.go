package db

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"lucid-ci/platform/models"
)

func getTestStore(t *testing.T) Store {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://lucid:lucid_dev@localhost:5432/lucid_ci?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	store, err := NewStore(ctx, dbURL)
	if err != nil {
		t.Skipf("PostgreSQL store not available at %s, skipping store integration test: %v", dbURL, err)
		return nil
	}
	return store
}

func TestStore_ScanRunLifecycle(t *testing.T) {
	store := getTestStore(t)
	if store == nil {
		return
	}
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Upsert Repository
	repo := &models.Repository{
		ID:             99001,
		FullName:       "acme/lifecycle-test",
		InstallationID: 101,
		DefaultBranch:  "main",
	}
	if err := store.UpsertRepository(ctx, repo); err != nil {
		t.Fatalf("UpsertRepository failed: %v", err)
	}

	// 2. Create Scan Run with unhyphenated 32-character hex ID (Contract CTR-002 TaskID)
	taskID := fmt.Sprintf("deadbeef%024x", time.Now().UnixNano()%1000000000)
	scanRun := &models.ScanRun{
		ID:            taskID,
		RepositoryID:  repo.ID,
		PRNumber:      77,
		CommitSHA:     "a1b2c3d4",
		Status:        models.ScanStatusScanning,
		FindingsCount: 0,
	}
	if err := store.CreateScanRun(ctx, scanRun); err != nil {
		t.Fatalf("CreateScanRun failed: %v", err)
	}

	// Verify initial state
	scan, err := store.GetScanRunByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetScanRunByID failed: %v", err)
	}
	if scan.Status != models.ScanStatusScanning {
		t.Errorf("Expected status %s, got %s", models.ScanStatusScanning, scan.Status)
	}
	if scan.FindingsCount != 0 {
		t.Errorf("Expected findings_count 0, got %d", scan.FindingsCount)
	}
	if scan.CompletedAt != nil {
		t.Errorf("Expected completed_at to be nil, got %v", scan.CompletedAt)
	}
	if scan.ScanDurationMs != nil {
		t.Errorf("Expected scan_duration_ms to be nil, got %v", scan.ScanDurationMs)
	}

	// 3. Transition to ANALYZING_AI
	if err := store.UpdateScanRunStatus(ctx, taskID, models.ScanStatusAnalyzingAI, 3, nil); err != nil {
		t.Fatalf("UpdateScanRunStatus to ANALYZING_AI failed: %v", err)
	}

	scan, err = store.GetScanRunByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetScanRunByID failed: %v", err)
	}
	if scan.Status != models.ScanStatusAnalyzingAI {
		t.Errorf("Expected status %s, got %s", models.ScanStatusAnalyzingAI, scan.Status)
	}
	if scan.FindingsCount != 3 {
		t.Errorf("Expected findings_count 3, got %d", scan.FindingsCount)
	}
	if scan.CompletedAt != nil {
		t.Errorf("Expected completed_at to remain nil, got %v", scan.CompletedAt)
	}

	// 4. Transition to SANDBOXING
	if err := store.UpdateScanRunStatus(ctx, taskID, models.ScanStatusSandboxing, 3, nil); err != nil {
		t.Fatalf("UpdateScanRunStatus to SANDBOXING failed: %v", err)
	}

	scan, err = store.GetScanRunByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetScanRunByID failed: %v", err)
	}
	if scan.Status != models.ScanStatusSandboxing {
		t.Errorf("Expected status %s, got %s", models.ScanStatusSandboxing, scan.Status)
	}
	if scan.CompletedAt != nil {
		t.Errorf("Expected completed_at to remain nil, got %v", scan.CompletedAt)
	}

	// 5. Transition to COMPLETED
	durationMs := 1450
	if err := store.UpdateScanRunStatus(ctx, taskID, models.ScanStatusCompleted, 3, &durationMs); err != nil {
		t.Fatalf("UpdateScanRunStatus to COMPLETED failed: %v", err)
	}

	scan, err = store.GetScanRunByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetScanRunByID failed: %v", err)
	}
	if scan.Status != models.ScanStatusCompleted {
		t.Errorf("Expected status %s, got %s", models.ScanStatusCompleted, scan.Status)
	}
	if scan.FindingsCount != 3 {
		t.Errorf("Expected findings_count 3, got %d", scan.FindingsCount)
	}
	if scan.CompletedAt == nil {
		t.Errorf("Expected completed_at to be non-nil after COMPLETED")
	}
	if scan.ScanDurationMs == nil || *scan.ScanDurationMs != 1450 {
		t.Errorf("Expected scan_duration_ms 1450, got %v", scan.ScanDurationMs)
	}

	// 6. Transition back to SCANNING: completed_at and scan_duration_ms MUST be reset to nil
	if err := store.UpdateScanRunStatus(ctx, taskID, models.ScanStatusScanning, 0, nil); err != nil {
		t.Fatalf("UpdateScanRunStatus to SCANNING failed: %v", err)
	}

	scan, err = store.GetScanRunByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetScanRunByID failed: %v", err)
	}
	if scan.Status != models.ScanStatusScanning {
		t.Errorf("Expected status %s, got %s", models.ScanStatusScanning, scan.Status)
	}
	if scan.CompletedAt != nil {
		t.Errorf("Expected completed_at to be reset to nil on transition to SCANNING, got %v", scan.CompletedAt)
	}
	if scan.ScanDurationMs != nil {
		t.Errorf("Expected scan_duration_ms to be reset to nil on transition to SCANNING, got %v", scan.ScanDurationMs)
	}
}

func TestStore_ScanRunFailureLifecycle(t *testing.T) {
	store := getTestStore(t)
	if store == nil {
		return
	}
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repo := &models.Repository{
		ID:             99002,
		FullName:       "acme/failure-test",
		InstallationID: 102,
		DefaultBranch:  "main",
	}
	if err := store.UpsertRepository(ctx, repo); err != nil {
		t.Fatalf("UpsertRepository failed: %v", err)
	}

	taskID := fmt.Sprintf("cafe%028x", (time.Now().UnixNano()/1000)%1000000000000)
	scanRun := &models.ScanRun{
		ID:            taskID,
		RepositoryID:  repo.ID,
		PRNumber:      44,
		CommitSHA:     "b2c3d4e5",
		Status:        models.ScanStatusScanning,
		FindingsCount: 2,
	}
	if err := store.CreateScanRun(ctx, scanRun); err != nil {
		t.Fatalf("CreateScanRun failed: %v", err)
	}

	durationMs := 820
	// Transition to FAILED: should preserve existing findings_count if 0 is passed
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
	if scan.FindingsCount != 2 {
		t.Errorf("Expected findings_count 2 to be preserved, got %d", scan.FindingsCount)
	}
	if scan.CompletedAt == nil {
		t.Errorf("Expected completed_at to be non-nil after FAILED")
	}
	if scan.ScanDurationMs == nil || *scan.ScanDurationMs != 820 {
		t.Errorf("Expected scan_duration_ms 820, got %v", scan.ScanDurationMs)
	}
}

func TestStore_UpdateNonExistentScanRun(t *testing.T) {
	store := getTestStore(t)
	if store == nil {
		return
	}
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nonExistentID := "00000000-0000-0000-0000-000000000000"
	err := store.UpdateScanRunStatus(ctx, nonExistentID, models.ScanStatusCompleted, 0, nil)
	if err == nil {
		t.Errorf("Expected error when updating non-existent scan run, got nil")
	}

	err = store.UpdateCheckRunID(ctx, nonExistentID, 123456)
	if err == nil {
		t.Errorf("Expected error when updating check run ID on non-existent scan run, got nil")
	}
}

func TestStore_UpdateCheckRunID(t *testing.T) {
	store := getTestStore(t)
	if store == nil {
		return
	}
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repo := &models.Repository{
		ID:             99003,
		FullName:       "acme/checkrun-test",
		InstallationID: 103,
		DefaultBranch:  "main",
	}
	_ = store.UpsertRepository(ctx, repo)

	taskID := fmt.Sprintf("c001%028x", (time.Now().UnixNano()/2000)%1000000000000)
	scanRun := &models.ScanRun{
		ID:            taskID,
		RepositoryID:  repo.ID,
		PRNumber:      55,
		CommitSHA:     "c3d4e5f6",
		Status:        models.ScanStatusScanning,
		FindingsCount: 0,
	}
	if err := store.CreateScanRun(ctx, scanRun); err != nil {
		t.Fatalf("CreateScanRun failed: %v", err)
	}

	checkRunID := int64(987654321)
	if err := store.UpdateCheckRunID(ctx, taskID, checkRunID); err != nil {
		t.Fatalf("UpdateCheckRunID failed: %v", err)
	}

	scan, err := store.GetScanRunByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetScanRunByID failed: %v", err)
	}
	if scan.CheckRunID == nil || *scan.CheckRunID != checkRunID {
		t.Errorf("Expected check_run_id %d, got %v", checkRunID, scan.CheckRunID)
	}
}

func TestStore_InsertAndGetVulnerabilities(t *testing.T) {
	store := getTestStore(t)
	if store == nil {
		return
	}
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repo := &models.Repository{
		ID:             99004,
		FullName:       "acme/vuln-test",
		InstallationID: 104,
		DefaultBranch:  "main",
	}
	_ = store.UpsertRepository(ctx, repo)

	taskID := fmt.Sprintf("beef%028x", (time.Now().UnixNano()/3000)%1000000000000)
	scanRun := &models.ScanRun{
		ID:            taskID,
		RepositoryID:  repo.ID,
		PRNumber:      66,
		CommitSHA:     "d4e5f6a1",
		Status:        models.ScanStatusScanning,
		FindingsCount: 0,
	}
	if err := store.CreateScanRun(ctx, scanRun); err != nil {
		t.Fatalf("CreateScanRun failed: %v", err)
	}

	patch := "const q = db.paramQuery(id);"
	explanation := "Use parameterized queries."
	vulns := []models.Vulnerability{
		{
			ID:                 "vuln-test-001",
			ScanRunID:          taskID,
			RuleID:             "LUCID-SEC-001",
			CWE:                "CWE-89",
			Severity:           models.SeverityHigh,
			ConfidenceScore:    0.95,
			FilePath:           "app.js",
			LineStart:          10,
			LineEnd:            12,
			VulnerableCode:     "db.query(q);",
			AIRemediationPatch: &patch,
			AIExplanation:      &explanation,
			SandboxVerified:    false,
		},
		{
			ID:                 "vuln-test-002",
			ScanRunID:          taskID,
			RuleID:             "LUCID-SEC-002",
			CWE:                "CWE-78",
			Severity:           models.SeverityCritical,
			ConfidenceScore:    0.98,
			FilePath:           "app.js",
			LineStart:          20,
			LineEnd:            22,
			VulnerableCode:     "child_process.exec(cmd);",
			SandboxVerified:    true,
		},
	}

	if err := store.InsertVulnerabilities(ctx, taskID, vulns); err != nil {
		t.Fatalf("InsertVulnerabilities failed: %v", err)
	}

	fetched, err := store.GetVulnerabilitiesByScanRunID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetVulnerabilitiesByScanRunID failed: %v", err)
	}

	if len(fetched) != 2 {
		t.Fatalf("Expected 2 vulnerabilities, got %d", len(fetched))
	}

	if fetched[0].RuleID != "LUCID-SEC-001" || fetched[0].SandboxVerified != false {
		t.Errorf("Unexpected vuln 0: %+v", fetched[0])
	}
	if fetched[1].RuleID != "LUCID-SEC-002" || fetched[1].SandboxVerified != true {
		t.Errorf("Unexpected vuln 1: %+v", fetched[1])
	}
}
