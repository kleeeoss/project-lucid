package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"lucid-ci/platform/db"
	"lucid-ci/platform/models"
)

// TaskHandler is the function signature executed per scan task.
type TaskHandler func(ctx context.Context, task models.ScanTaskMessage) error

// Pool manages a bounded goroutine worker pool consuming from AWS SQS.
type Pool struct {
	client      *sqs.Client
	queueURL    string
	store       db.Store
	logger      *slog.Logger
	concurrency int
	handler     TaskHandler
	wg          sync.WaitGroup
}

// NewPool initializes a Worker Pool with configured concurrency and dependencies.
func NewPool(client *sqs.Client, queueURL string, store db.Store, concurrency int, logger *slog.Logger, handler TaskHandler) *Pool {
	if concurrency <= 0 {
		concurrency = 3
	}
	return &Pool{
		client:      client,
		queueURL:    queueURL,
		store:       store,
		logger:      logger,
		concurrency: concurrency,
		handler:     handler,
	}
}

// Start launches the consumer dispatcher and worker goroutines until context is cancelled.
func (p *Pool) Start(ctx context.Context) {
	p.logger.Info("Starting Platform Worker Pool",
		"concurrency", p.concurrency,
		"queue_url", p.queueURL,
	)

	taskChan := make(chan *models.ScanTaskMessage, p.concurrency*2)

	// Launch worker goroutines
	for i := 1; i <= p.concurrency; i++ {
		p.wg.Add(1)
		go p.workerLoop(ctx, i, taskChan)
	}

	// Dispatcher loop: polls SQS and pushes to taskChan
	p.dispatcherLoop(ctx, taskChan)

	// Context cancelled, close task channel and wait for workers to finish active jobs
	close(taskChan)
	p.wg.Wait()
	p.logger.Info("Platform Worker Pool stopped cleanly")
}

func (p *Pool) dispatcherLoop(ctx context.Context, taskChan chan<- *models.ScanTaskMessage) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Long poll SQS for up to 20 seconds
			output, err := p.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(p.queueURL),
				MaxNumberOfMessages: 5,
				WaitTimeSeconds:     20,
				VisibilityTimeout:   300, // 5 minutes processing window
			})
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				p.logger.Error("SQS ReceiveMessage failed", "error", err)
				time.Sleep(2 * time.Second) // backoff before retrying poll
				continue
			}

			for _, msg := range output.Messages {
				var task models.ScanTaskMessage
				if err := json.Unmarshal([]byte(*msg.Body), &task); err != nil {
					p.logger.Error("Failed to unmarshal SQS scan message body", "error", err)
					// Delete poisoned message to prevent infinite queue loops
					p.deleteSQSMessage(ctx, *msg.ReceiptHandle)
					continue
				}

				// Dispatch task to worker channel
				select {
				case taskChan <- &task:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func (p *Pool) workerLoop(ctx context.Context, workerID int, taskChan <-chan *models.ScanTaskMessage) {
	defer p.wg.Done()
	p.logger.Debug("Worker goroutine started", "worker_id", workerID)

	for task := range taskChan {
		p.processTaskSafely(ctx, workerID, *task)
	}
}

func (p *Pool) processTaskSafely(ctx context.Context, workerID int, task models.ScanTaskMessage) {
	// Panic recovery guard per scan execution
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error("Worker goroutine panic recovered",
				"worker_id", workerID,
				"task_id", task.TaskID,
				"panic", r,
			)
			if p.store != nil {
				_ = p.store.UpdateScanRunStatus(ctx, task.TaskID, models.ScanStatusFailed, 0, nil)
			}
		}
	}()

	start := time.Now()
	p.logger.Info("Worker picked up scan task",
		"worker_id", workerID,
		"task_id", task.TaskID,
		"repo", task.RepositoryName,
		"pr", task.PRNumber,
		"commit", task.CommitSHA,
	)

	// Transition status to SCANNING in database
	if p.store != nil {
		_ = p.store.UpdateScanRunStatus(ctx, task.TaskID, models.ScanStatusScanning, 0, nil)
	}

	var err error
	if p.handler != nil {
		err = p.handler(ctx, task)
	}

	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		p.logger.Error("Scan task failed",
			"task_id", task.TaskID,
			"duration_ms", durationMs,
			"error", err,
		)
		if p.store != nil {
			_ = p.store.UpdateScanRunStatus(ctx, task.TaskID, models.ScanStatusFailed, 0, &durationMs)
		}
		return
	}

	p.logger.Info("Scan task completed successfully",
		"task_id", task.TaskID,
		"duration_ms", durationMs,
	)
	if p.store != nil {
		_ = p.store.UpdateScanRunStatus(ctx, task.TaskID, models.ScanStatusCompleted, 0, &durationMs)
	}
}

func (p *Pool) deleteSQSMessage(ctx context.Context, receiptHandle string) {
	_, err := p.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(p.queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		p.logger.Warn("Failed to delete SQS message", "error", err)
	}
}
