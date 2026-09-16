# Dead Letter Queue for poison-pill messages
resource "aws_sqs_queue" "scan_dlq" {
  name                      = "lucid-ci-scans-dlq"
  message_retention_seconds = 1209600 # 14 days

  tags = {
    Name        = "lucid-ci-scans-dlq"
    Environment = var.environment
    Project     = "lucid-ci"
  }
}

# Primary Queue for webhook tasks
resource "aws_sqs_queue" "scan_queue" {
  name                       = "lucid-ci-scans"
  visibility_timeout_seconds = 180   # 3 minutes (allows worker pipeline to complete)
  message_retention_seconds  = 345600 # 4 days
  receive_wait_time_seconds  = 20     # Long polling enabled

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.scan_dlq.arn
    maxReceiveCount     = 3
  })

  tags = {
    Name        = "lucid-ci-scans"
    Environment = var.environment
    Project     = "lucid-ci"
  }
}