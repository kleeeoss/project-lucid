package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"

	"lucid-ci/platform/db"
	"lucid-ci/platform/models"
)

func TestPool_ProcessTaskSafely_PanicRecovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	panickingHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		panic("simulated fatal failure in engine analysis")
	}

	pool := NewPool(nil, "http://mock-queue", nil, 2, logger, panickingHandler)

	task := models.ScanTaskMessage{
		TaskID:         "panic-test-task",
		RepositoryName: "acme/lucid-ci",
		PRNumber:       42,
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("processTaskSafely failed to recover from panic: %v", r)
		}
	}()

	pool.processTaskSafely(context.Background(), 1, task, "")
}

func TestPool_ProcessTaskSafely_SuccessAndFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	var executed atomic.Bool

	successHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		executed.Store(true)
		return nil
	}

	pool := NewPool(nil, "http://mock-queue", nil, 1, logger, successHandler)
	task := models.ScanTaskMessage{TaskID: "success-task"}

	pool.processTaskSafely(context.Background(), 1, task, "")

	if !executed.Load() {
		t.Errorf("Expected handler to be executed")
	}

	errorHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		return errors.New("simulated network error")
	}
	pool.handler = errorHandler
	pool.processTaskSafely(context.Background(), 1, task, "")
}

type mockPoolStore struct {
	db.Store
	updateStatusCalls []statusCall
	updateStatusErr   error
}

type statusCall struct {
	scanID        string
	status        models.ScanStatus
	findingsCount int
	durationMs    *int
}

func (m *mockPoolStore) UpdateScanRunStatus(ctx context.Context, scanID string, status models.ScanStatus, findingsCount int, durationMs *int) error {
	m.updateStatusCalls = append(m.updateStatusCalls, statusCall{scanID, status, findingsCount, durationMs})
	return m.updateStatusErr
}

func TestPool_ProcessTaskSafely_StoreStatusOnFailureAndPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// 1. Failure case
	mockStore := &mockPoolStore{}
	failHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		return errors.New("pipeline fatal failure")
	}

	pool := NewPool(nil, "http://mock-queue", mockStore, 1, logger, failHandler)
	task := models.ScanTaskMessage{TaskID: "task-fail-001"}
	pool.processTaskSafely(context.Background(), 1, task, "")

	if len(mockStore.updateStatusCalls) < 2 {
		t.Fatalf("Expected at least 2 status update calls (SCANNING and FAILED), got %d", len(mockStore.updateStatusCalls))
	}
	lastCall := mockStore.updateStatusCalls[len(mockStore.updateStatusCalls)-1]
	if lastCall.status != models.ScanStatusFailed {
		t.Errorf("Expected final status to be %s, got %s", models.ScanStatusFailed, lastCall.status)
	}
	if lastCall.scanID != "task-fail-001" {
		t.Errorf("Expected scan ID 'task-fail-001', got %s", lastCall.scanID)
	}

	// 2. Panic case
	mockStorePanic := &mockPoolStore{}
	panicHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		panic("catastrophic engine crash")
	}
	poolPanic := NewPool(nil, "http://mock-queue", mockStorePanic, 1, logger, panicHandler)
	taskPanic := models.ScanTaskMessage{TaskID: "task-panic-001"}
	poolPanic.processTaskSafely(context.Background(), 1, taskPanic, "")

	if len(mockStorePanic.updateStatusCalls) < 2 {
		t.Fatalf("Expected at least 2 status update calls on panic, got %d", len(mockStorePanic.updateStatusCalls))
	}
	lastPanicCall := mockStorePanic.updateStatusCalls[len(mockStorePanic.updateStatusCalls)-1]
	if lastPanicCall.status != models.ScanStatusFailed {
		t.Errorf("Expected final status on panic to be %s, got %s", models.ScanStatusFailed, lastPanicCall.status)
	}

	// 3. Store error resilience: when store returns error on update, worker does not panic
	mockStoreErr := &mockPoolStore{updateStatusErr: errors.New("db connection lost")}
	poolErr := NewPool(nil, "http://mock-queue", mockStoreErr, 1, logger, failHandler)
	poolErr.processTaskSafely(context.Background(), 1, task, "")
}
