package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"lucid-ci/platform/db"
	"lucid-ci/platform/models"
)

type mockSQSClient struct {
	deleteMessageFunc  func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
	receiveMessageFunc func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	deletedHandles     []string
	mu                 sync.Mutex
}

func (m *mockSQSClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if m.receiveMessageFunc != nil {
		return m.receiveMessageFunc(ctx, params, optFns...)
	}
	return &sqs.ReceiveMessageOutput{}, nil
}

func (m *mockSQSClient) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if params != nil && params.ReceiptHandle != nil {
		m.deletedHandles = append(m.deletedHandles, *params.ReceiptHandle)
	}
	if m.deleteMessageFunc != nil {
		return m.deleteMessageFunc(ctx, params, optFns...)
	}
	return &sqs.DeleteMessageOutput{}, nil
}

func (m *mockSQSClient) GetDeletedHandles() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.deletedHandles))
	copy(result, m.deletedHandles)
	return result
}

type mockStore struct {
	db.Store
	updateStatusFunc func(ctx context.Context, scanID string, status models.ScanStatus, findingsCount int, durationMs *int) error
}

func (m *mockStore) UpdateScanRunStatus(ctx context.Context, scanID string, status models.ScanStatus, findingsCount int, durationMs *int) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, scanID, status, findingsCount, durationMs)
	}
	return nil
}

func TestPool_ProcessTaskSafely_PanicRecovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	panickingHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		panic("simulated fatal failure in engine analysis")
	}

	pool := NewPool(nil, "http://mock-queue", nil, 2, logger, panickingHandler)

	job := queuedJob{
		task: models.ScanTaskMessage{
			TaskID:         "panic-test-task",
			RepositoryName: "acme/lucid-ci",
			PRNumber:       42,
		},
		receiptHandle: "mock-receipt-handle",
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("processTaskSafely failed to recover from panic: %v", r)
		}
	}()

	pool.processTaskSafely(context.Background(), 1, job)
}

func TestPool_ProcessTaskSafely_SuccessAndFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	var executed atomic.Bool

	successHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		executed.Store(true)
		return nil
	}

	pool := NewPool(nil, "http://mock-queue", nil, 1, logger, successHandler)
	job := queuedJob{
		task:          models.ScanTaskMessage{TaskID: "success-task"},
		receiptHandle: "mock-receipt-handle",
	}

	pool.processTaskSafely(context.Background(), 1, job)

	if !executed.Load() {
		t.Errorf("Expected handler to be executed")
	}

	errorHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		return errors.New("simulated network error")
	}
	pool.handler = errorHandler
	pool.processTaskSafely(context.Background(), 1, job)
}

func TestPool_ProcessTaskSafely_SuccessDeletesMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSQS := &mockSQSClient{}
	mockSt := &mockStore{}

	successHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		return nil
	}

	pool := NewPool(mockSQS, "http://mock-queue/tasks", mockSt, 1, logger, successHandler)
	job := queuedJob{
		task: models.ScanTaskMessage{
			TaskID:         "task-success-123",
			RepositoryName: "lucid-org/test-repo",
			PRNumber:       10,
		},
		receiptHandle: "receipt-handle-success-xyz",
	}

	pool.processTaskSafely(context.Background(), 1, job)

	deleted := mockSQS.GetDeletedHandles()
	if len(deleted) != 1 {
		t.Fatalf("expected exactly 1 deleted message, got %d", len(deleted))
	}
	if deleted[0] != "receipt-handle-success-xyz" {
		t.Errorf("expected deleted handle 'receipt-handle-success-xyz', got %q", deleted[0])
	}
}

func TestPool_ProcessTaskSafely_FailureDoesNotDeleteMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSQS := &mockSQSClient{}
	mockSt := &mockStore{}

	failureHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		return errors.New("transient engine failure")
	}

	pool := NewPool(mockSQS, "http://mock-queue/tasks", mockSt, 1, logger, failureHandler)
	job := queuedJob{
		task: models.ScanTaskMessage{
			TaskID:         "task-fail-456",
			RepositoryName: "lucid-org/test-repo",
			PRNumber:       11,
		},
		receiptHandle: "receipt-handle-fail-abc",
	}

	pool.processTaskSafely(context.Background(), 1, job)

	deleted := mockSQS.GetDeletedHandles()
	if len(deleted) != 0 {
		t.Fatalf("expected message to NOT be deleted on task failure, but got: %v", deleted)
	}
}

func TestPool_ProcessTaskSafely_StatusUpdateFailureDoesNotDeleteMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSQS := &mockSQSClient{}
	mockSt := &mockStore{
		updateStatusFunc: func(ctx context.Context, scanID string, status models.ScanStatus, findingsCount int, durationMs *int) error {
			if status == models.ScanStatusCompleted {
				return errors.New("database connection lost")
			}
			return nil
		},
	}

	successHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		return nil
	}

	pool := NewPool(mockSQS, "http://mock-queue/tasks", mockSt, 1, logger, successHandler)
	job := queuedJob{
		task: models.ScanTaskMessage{
			TaskID: "task-db-error-789",
		},
		receiptHandle: "receipt-handle-db-error",
	}

	pool.processTaskSafely(context.Background(), 1, job)

	deleted := mockSQS.GetDeletedHandles()
	if len(deleted) != 0 {
		t.Fatalf("expected message to NOT be deleted when UpdateScanRunStatus fails, got: %v", deleted)
	}
}

func TestPool_DispatcherLoop_PoisonMessageDeletedImmediately(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSQS := &mockSQSClient{}

	callCount := 0
	mockSQS.receiveMessageFunc = func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
		callCount++
		if callCount == 1 {
			return &sqs.ReceiveMessageOutput{
				Messages: []sqsTypes.Message{
					{
						Body:          aws.String("poison-malformed-json{{"),
						ReceiptHandle: aws.String("receipt-poison-handle"),
					},
				},
			}, nil
		}
		// Second call returns empty, wait for context cancellation
		<-ctx.Done()
		return nil, ctx.Err()
	}

	pool := NewPool(mockSQS, "http://mock-queue/tasks", nil, 1, logger, nil)
	taskChan := make(chan queuedJob, 5)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	pool.dispatcherLoop(ctx, taskChan)

	deleted := mockSQS.GetDeletedHandles()
	if len(deleted) != 1 {
		t.Fatalf("expected 1 deleted handle for poisoned message, got %d", len(deleted))
	}
	if deleted[0] != "receipt-poison-handle" {
		t.Errorf("expected deleted handle 'receipt-poison-handle', got %q", deleted[0])
	}
	if len(taskChan) != 0 {
		t.Errorf("expected taskChan to be empty (poison message should not be dispatched), got %d items", len(taskChan))
	}
}
