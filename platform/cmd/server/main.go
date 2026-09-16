package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lucid-ci/platform/ingestion"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	queueURL := os.Getenv("SQS_QUEUE_URL")
	var sqsProducer ingestion.SQSProducer

	if queueURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		producer, err := ingestion.NewSQSProducer(ctx, queueURL, logger)
		cancel()
		if err != nil {
			logger.Warn("Failed to initialize AWS SQS producer, continuing in mock queue mode", "error", err)
		} else {
			sqsProducer = producer
			logger.Info("AWS SQS producer initialized successfully", "queue_url", queueURL)
		}
	} else {
		logger.Info("SQS_QUEUE_URL not set; running with mock queue handler")
	}

	router := ingestion.NewRouter(logger, sqsProducer)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Starting Lucid-CI Platform Server", "port", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server listen failed", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownChan
	logger.Info("Shutting down Platform Server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Platform Server stopped cleanly")
}
