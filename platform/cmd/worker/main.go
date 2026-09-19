package main

import (
	"context"
	"fmt"
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

	// 1. Concurrency configuration
	concurrency := 3
	if cStr := os.Getenv("WORKER_CONCURRENCY"); cStr != "" {
		if c, err := strconv.Atoi(cStr); err == nil && c > 0 {
			concurrency = c
		}
	}

	queueURL := os.Getenv("SQS_QUEUE_URL")
	databaseURL := os.Getenv("DATABASE_URL")
	aiServiceURL := os.Getenv("AI_SERVICE_URL")

	// 2. Database Store Connection
	var store db.Store
	if databaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		s, err := db.NewStore(ctx, databaseURL)
		cancel()
		if err != nil {
			logger.Warn("Database connection failed, worker continuing in memory mode", "error", err)
		} else {
			store = s
			defer store.Close()
			logger.Info("Connected to PostgreSQL persistence store")
		}
	}

	// 3. Initialize AI & Sandbox Clients (Garv's microservice)
	aiClient := worker.NewAIClient(aiServiceURL, logger)
	sandboxClient := worker.NewSandboxClient(aiServiceURL, logger)

	// 4. AWS SQS Client Initialization
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

	// 5. Orchestration Pipeline Handler (TASK-PLT-301, 302, 303)
	pipelineHandler := func(ctx context.Context, task models.ScanTaskMessage) error {
		logger.Info("Starting end-to-end scan pipeline orchestration",
			"task_id", task.TaskID,
			"repo", task.RepositoryName,
			"pr", task.PRNumber,
			"commit", task.CommitSHA,
		)

		// Record Scan Run in PostgreSQL
		if store != nil {
			repo := &models.Repository{
				ID:             task.RepositoryID,
				FullName:       task.RepositoryName,
				InstallationID: task.InstallationID,
				DefaultBranch:  "main",
			}
			if err := store.UpsertRepository(ctx, repo); err != nil {
				logger.Warn("Failed to upsert repository", "error", err)
			}

			scanRun := &models.ScanRun{
				RepositoryID:  task.RepositoryID,
				PRNumber:      task.PRNumber,
				CommitSHA:     task.CommitSHA,
				Status:        models.ScanStatusScanning,
				FindingsCount: 0,
			}
			if err := store.CreateScanRun(ctx, scanRun); err != nil {
				logger.Warn("Failed to create scan run record", "error", err)
			} else {
				logger.Info("Created scan_run in PostgreSQL", "scan_id", scanRun.ID)
			}
		}

		// AI Remediation Hook (Fail-Open BR-002: Failures in AI do NOT block PR)
		remReq := worker.RemediationRequest{
			ScanID:          task.TaskID,
			VulnerabilityID: fmt.Sprintf("vuln-%s", task.TaskID[:8]),
			RuleID:          "LUCID-SEC-001",
			CWE:             "CWE-89",
			Language:        "javascript",
			VulnerableCode:  "db.query(`SELECT * FROM users WHERE id = '${userId}'`)",
		}
		remResp, err := aiClient.RemediateVulnerability(ctx, remReq)
		if err != nil {
			logger.Warn("AI Remediation skipped or timed out (Fail-Open BR-002 active)", "error", err)
		} else {
			logger.Info("AI Remediation generated patch successfully",
				"model", remResp.ModelName,
				"confidence", remResp.Confidence,
			)
		}

		// Dynamic Sandbox Detonation Hook (CTR-006 / CTR-007)
		sandboxReq := worker.SandboxRequest{
			ScanID:         task.TaskID,
			CommitSHA:      task.CommitSHA,
			Language:       "javascript",
			BuildCommand:   "npm test",
			TimeoutSeconds: 60, // Standardized 60s container timeout
		}
		sandboxResult, err := sandboxClient.Detonate(ctx, sandboxReq)
		if err != nil {
			logger.Warn("Sandbox detonation skipped or offline", "error", err)
		} else {
			logger.Info("Sandbox detonation executed",
				"status", sandboxResult.Status,
				"exit_code", sandboxResult.ExitCode,
			)
		}

		return nil
	}

	// 6. Launch Worker Pool
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
		logger.Info("SQS_QUEUE_URL not configured. Worker standing by in idle mode.")
		<-workerCtx.Done()
	}
}
