package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"

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

	pool.processTaskSafely(context.Background(), 1, task)
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

	pool.processTaskSafely(context.Background(), 1, task)

	if !executed.Load() {
		t.Errorf("Expected handler to be executed")
	}

	errorHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		return errors.New("simulated network error")
	}
	pool.handler = errorHandler
	pool.processTaskSafely(context.Background(), 1, task)
}
