#!/bin/bash
echo "Initializing LocalStack SQS queues..."
awslocal sqs create-queue --queue-name lucid-ci-scan-queue --region us-east-1
awslocal sqs create-queue --queue-name lucid-ci-scan-dlq --region us-east-1
echo "Queues created successfully:"
awslocal sqs list-queues --region us-east-1