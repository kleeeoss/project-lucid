package db

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"lucid-ci/platform/models"
)

func getTestStore(t *testing.T) Store {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://lucid_user:lucid_password@localhost:5433/lucid_ci?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store, err := NewStore(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to connect to test database at %s: %v", databaseURL, err)
	}
	t.Cleanup(func() {
		store.Close()
	})
	return store
}

func newRandomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant RFC4122
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func TestScanRun_IdempotencyAndStatus(t *testing.T) {
	ctx := context.Background()
	store := getTestStore(t)

	repoID := time.Now().UnixNano()
	repo := &models.Repository{
		ID:             repoID,
		FullName:       fmt.Sprintf("lucid-org/test-repo-%d", repoID),
		InstallationID: 1001,
		DefaultBranch:  "main",
	}
	if err := store.UpsertRepository(ctx, repo); err != nil {
		t.Fatalf("UpsertRepository failed: %v", err)
	}

	taskID := newRandomUUID()

	// 1. Creating a scan run with a specific TaskID succeeds and sets scan_runs.id == TaskID
	scanRun1 := &models.ScanRun{
		ID:            taskID,
		RepositoryID:  repo.ID,
		PRNumber:      42,
		CommitSHA:     "a1b2c3d4e5f678901234567890abcdef12345678",
		Status:        models.ScanStatusScanning,
		FindingsCount: 0,
	}

	err := store.CreateScanRun(ctx, scanRun1)
	if err != nil {
		t.Fatalf("CreateScanRun failed for initial insertion: %v", err)
	}
	if scanRun1.ID != taskID {
		t.Errorf("expected scan_runs.id == TaskID (%s), got: %s", taskID, scanRun1.ID)
	}

	// Verify in database
	fetchedScan, err := store.GetScanRunByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetScanRunByID failed: %v", err)
	}
	if fetchedScan == nil || fetchedScan.ID != taskID {
		t.Fatalf("fetched scan ID mismatch: expected %s, got %+v", taskID, fetchedScan)
	}
	if fetchedScan.Status != models.ScanStatusScanning {
		t.Errorf("expected status SCANNING, got %s", fetchedScan.Status)
	}

	// 2. Creating a second scan run with the duplicate TaskID does NOT throw an error (idempotency verified)
	scanRun2 := &models.ScanRun{
		ID:            taskID,
		RepositoryID:  repo.ID,
		PRNumber:      42,
		CommitSHA:     "a1b2c3d4e5f678901234567890abcdef12345678",
		Status:        models.ScanStatusScanning,
		FindingsCount: 0,
	}

	err = store.CreateScanRun(ctx, scanRun2)
	if err != nil {
		t.Fatalf("CreateScanRun failed on duplicate TaskID insertion (idempotency violation): %v", err)
	}
	if scanRun2.ID != taskID {
		t.Errorf("expected duplicate scan_runs.id == TaskID (%s), got: %s", taskID, scanRun2.ID)
	}

	// 3. Calling UpdateScanRunStatus updates the status to COMPLETED and returns no error
	durationMs := 1250
	findingsCount := 3
	err = store.UpdateScanRunStatus(ctx, taskID, models.ScanStatusCompleted, findingsCount, &durationMs)
	if err != nil {
		t.Fatalf("UpdateScanRunStatus failed to update status to COMPLETED: %v", err)
	}

	// Verify updated status in database
	updatedScan, err := store.GetScanRunByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetScanRunByID after update failed: %v", err)
	}
	if updatedScan.Status != models.ScanStatusCompleted {
		t.Errorf("expected status %s, got %s", models.ScanStatusCompleted, updatedScan.Status)
	}
	if updatedScan.FindingsCount != findingsCount {
		t.Errorf("expected findings_count %d, got %d", findingsCount, updatedScan.FindingsCount)
	}
	if updatedScan.ScanDurationMs == nil || *updatedScan.ScanDurationMs != durationMs {
		t.Errorf("expected scan_duration_ms %d, got %v", durationMs, updatedScan.ScanDurationMs)
	}
	if updatedScan.CompletedAt == nil {
		t.Errorf("expected completed_at to be populated on COMPLETED status, got nil")
	}

	// 4. Calling UpdateScanRunStatus with a non-existent UUID returns an error ("scan_run not found")
	nonExistentUUID := newRandomUUID()
	err = store.UpdateScanRunStatus(ctx, nonExistentUUID, models.ScanStatusCompleted, 0, nil)
	if err == nil {
		t.Fatalf("expected error for non-existent UUID %s, got nil", nonExistentUUID)
	}
	if !strings.Contains(err.Error(), "scan_run not found") {
		t.Errorf("expected error to contain 'scan_run not found', got: %v", err)
	}
}

func TestScanRun_Validation(t *testing.T) {
	ctx := context.Background()
	store := getTestStore(t)

	repoID := time.Now().UnixNano()
	repo := &models.Repository{
		ID:             repoID,
		FullName:       fmt.Sprintf("lucid-org/test-validation-repo-%d", repoID),
		InstallationID: 1002,
		DefaultBranch:  "main",
	}
	if err := store.UpsertRepository(ctx, repo); err != nil {
		t.Fatalf("UpsertRepository failed: %v", err)
	}

	t.Run("CreateScanRun with malformed UUID fails cleanly", func(t *testing.T) {
		scan := &models.ScanRun{
			ID:           "not-a-valid-uuid",
			RepositoryID: repo.ID,
			PRNumber:     1,
			CommitSHA:    "1234567890123456789012345678901234567890",
		}
		err := store.CreateScanRun(ctx, scan)
		if err == nil {
			t.Fatalf("expected error for malformed UUID, got nil")
		}
		if !strings.Contains(err.Error(), "invalid scan ID") {
			t.Errorf("expected error containing 'invalid scan ID', got: %v", err)
		}
	})

	t.Run("CreateScanRun with empty ID auto-generates valid UUID", func(t *testing.T) {
		scan := &models.ScanRun{
			ID:           "",
			RepositoryID: repo.ID,
			PRNumber:     2,
			CommitSHA:    "1234567890123456789012345678901234567890",
		}
		err := store.CreateScanRun(ctx, scan)
		if err != nil {
			t.Fatalf("CreateScanRun with empty ID failed: %v", err)
		}
		if !IsValidUUID(scan.ID) {
			t.Errorf("expected auto-generated scan.ID to be valid UUID, got: %s", scan.ID)
		}
	})

	t.Run("UpdateScanRunStatus with malformed UUID fails cleanly", func(t *testing.T) {
		err := store.UpdateScanRunStatus(ctx, "invalid-scan-id", models.ScanStatusCompleted, 0, nil)
		if err == nil {
			t.Fatalf("expected error for malformed UUID in UpdateScanRunStatus, got nil")
		}
		if !strings.Contains(err.Error(), "invalid scan ID") {
			t.Errorf("expected error containing 'invalid scan ID', got: %v", err)
		}
	})
}
