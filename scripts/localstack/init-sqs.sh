#!/usr/bin/env bash
# LocalStack init script executed when LocalStack is ready.
# Mounts into /etc/localstack/init/ready.d/init-sqs.sh

set -eo pipefail

echo "=========================================================="
echo "Initializing LocalStack SQS Queues for Lucid-CI..."
echo "=========================================================="

AWS_REGION="us-east-1"
LOCALSTACK_ENDPOINT="http://localhost:4566"

# 1. Create Dead Letter Queue (DLQ)
echo "Creating Dead Letter Queue: lucid-ci-scans-dlq"
awslocal --region "${AWS_REGION}" sqs create-queue \
  --queue-name "lucid-ci-scans-dlq" \
  --attributes '{"MessageRetentionPeriod":"1209600"}'

# 2. Extract DLQ ARN
DLQ_ARN=$(awslocal --region "${AWS_REGION}" sqs get-queue-attributes \
  --queue-url "${LOCALSTACK_ENDPOINT}/000000000000/lucid-ci-scans-dlq" \
  --attribute-names QueueArn \
  --query 'Attributes.QueueArn' \
  --output text)

echo "DLQ created with ARN: ${DLQ_ARN}"

# 3. Create Primary Queue with JSON attributes file (immune to comma-splitting bugs)
cat <<EOF > /tmp/primary-queue-attrs.json
{
  "VisibilityTimeout": "180",
  "ReceiveMessageWaitTimeSeconds": "20",
  "MessageRetentionPeriod": "345600",
  "RedrivePolicy": "{\"deadLetterTargetArn\":\"${DLQ_ARN}\",\"maxReceiveCount\":\"3\"}"
}
EOF

echo "Creating Primary Queue: lucid-ci-scans (with DLQ redrive policy)"
awslocal --region "${AWS_REGION}" sqs create-queue \
  --queue-name "lucid-ci-scans" \
  --attributes file:///tmp/primary-queue-attrs.json

PRIMARY_URL=$(awslocal --region "${AWS_REGION}" sqs get-queue-url \
  --queue-name "lucid-ci-scans" \
  --query 'QueueUrl' \
  --output text)

echo "Primary Queue created: ${PRIMARY_URL}"
echo "=========================================================="
echo "LocalStack SQS Initialization Complete!"
echo "=========================================================="
