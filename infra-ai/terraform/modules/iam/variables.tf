variable "sqs_queue_arn" {
  type        = string
  description = "ARN of the primary SQS queue"
}

variable "sqs_dlq_arn" {
  type        = string
  description = "ARN of the SQS Dead Letter Queue"
}

variable "environment" {
  type        = string
  default     = "development"
  description = "Environment identifier tag"
}