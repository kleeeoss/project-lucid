package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go/aws"

	"lucid-ci/platform/db"
	"lucid-ci/platform/models"
	"lucid-ci/platform/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Initializing Lucid-CI Platform Worker Daemon")

	concurrency := 3
	if cStr := os.Getenv("WORKER_CONCURRENCY"); cStr != "" {
		if c, err := strconv.Atoi(cStr); err == nil && c > 0 {
			concurrency = c
		}
	}

	queueURL := os.Getenv("SQS_QUEUE_URL")
	databaseURL := os.Getenv("DATABASE_URL")

	var store db.Store
	if databaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		s, err := db.NewStore(ctx, databaseURL)
		cancel()
		if err != nil {
			logger.Warn("Database connection failed, running worker in memory mode", "error", err)
		} else {
			store = s
			defer store.Close()
			logger.Info("Connected to PostgreSQL persistence store")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion("us-east-1"))
	cancel()
	if err != nil {
		logger.Error("Failed to load AWS configuration", "error", err)
		os.Exit(1)
	}

	endpointURL := os.Getenv("LOCALSTACK_ENDPOINT")
	if endpointURL == "" {
		endpointURL = os.Getenv("AWS_ENDPOINT_URL")
	}

	sqsClient := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		if endpointURL != "" {
			o.BaseEndpoint = aws.String(endpointURL)
		}
	})

	pipelineHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		logger.Info("Executing analysis pipeline for task",
			"task_id", task.TaskID,
			"repo", task.RepositoryName,
			"pr", task.PRNumber,
		)
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	pool := worker.NewPool(sqsClient, queueURL, store, concurrency, logger, pipelineHandler)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownChan
		logger.Info("Received shutdown signal, draining worker pool...")
		workerCancel()
	}()

	if queueURL != "" {
		pool.Start(workerCtx)
	} else {
		logger.Info("SQS_QUEUE_URL not configured. Worker daemon standing by in idle mode.")
		<-workerCtx.Done()
	}
}
