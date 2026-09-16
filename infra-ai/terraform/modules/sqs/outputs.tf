output "queue_url" {
  value = aws_sqs_queue.scan_queue.url
}

output "queue_arn" {
  value = aws_sqs_queue.scan_queue.arn
}

output "dlq_url" {
  value = aws_sqs_queue.scan_dlq.url
}

output "dlq_arn" {
  value = aws_sqs_queue.scan_dlq.arn
}