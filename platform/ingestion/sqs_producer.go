package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"net/url"
	"os"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go/aws"

	"lucid-ci/platform/models"
)

// SQSProducer defines the interface for publishing scan tasks to SQS.
type SQSProducer interface {
	PublishScanTask(ctx context.Context, task models.ScanTaskMessage) (string, error)
}

type sqsProducer struct {
	client   *sqs.Client
	queueURL string
	logger   *slog.Logger
}

// NewSQSProducer initializes the SQS client with AWS SDK v2.
// It automatically detects LOCALSTACK_ENDPOINT or AWS_ENDPOINT_URL for local testing.
func NewSQSProducer(ctx context.Context, queueURL string, logger *slog.Logger) (SQSProducer, error) {
	if queueURL == "" {
		return nil, fmt.Errorf("SQS queueURL cannot be empty")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}

	// Check for local development endpoint override (LocalStack)
	endpointURL := os.Getenv("LOCALSTACK_ENDPOINT")
	if endpointURL == "" {
		endpointURL = os.Getenv("AWS_ENDPOINT_URL")
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS default config: %w", err)
	}

	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		if endpointURL != "" {
			o.BaseEndpoint = aws.String(endpointURL)
		}
	})

	return &sqsProducer{
		client:   client,
		queueURL: queueURL,
		logger:   logger,
	}, nil
}

// PublishScanTask publishes a ScanTaskMessage to SQS with jittered exponential backoff.
func (p *sqsProducer) PublishScanTask(ctx context.Context, task models.ScanTaskMessage) (string, error) {
	body, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ScanTaskMessage: %w", err)
	}

	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		input := &sqs.SendMessageInput{
			QueueUrl:    aws.String(p.queueURL),
			MessageBody: aws.String(string(body)),
			MessageAttributes: map[string]types.MessageAttributeValue{
				"TaskID": {
					DataType:    aws.String("String"),
					StringValue: aws.String(task.TaskID),
				},
				"Repository": {
					DataType:    aws.String("String"),
					StringValue: aws.String(task.RepositoryName),
				},
				"PRNumber": {
					DataType:    aws.String("Number"),
					StringValue: aws.String(fmt.Sprintf("%d", task.PRNumber)),
				},
			},
		}

		resp, err := p.client.SendMessage(ctx, input)
		if err == nil {
			msgID := ""
			if resp.MessageId != nil {
				msgID = *resp.MessageId
			}
			p.logger.Info("Successfully enqueued scan task to SQS",
				"task_id", task.TaskID,
				"message_id", msgID,
				"repo", task.RepositoryName,
				"pr", task.PRNumber,
			)
			return msgID, nil
		}

		lastErr = err
		p.logger.Warn("Failed to send message to SQS, retrying...",
			"attempt", attempt+1,
			"error", err,
			"task_id", task.TaskID,
		)

		// Exponential backoff: 100ms, 200ms, 400ms + random jitter
		backoff := time.Duration(math.Pow(2, float64(attempt))*100)*time.Millisecond +
			time.Duration(rand.Intn(50))*time.Millisecond

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	return "", fmt.Errorf("exhausted retries publishing task %s to SQS: %w", task.TaskID, lastErr)
}

// IsValidQueueURL performs basic structural validation on an SQS URL.
func IsValidQueueURL(rawURL string) bool {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
