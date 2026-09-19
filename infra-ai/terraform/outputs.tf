output "ec2_public_ip" {
  description = "Public Elastic IP of the Lucid-CI EC2 host"
  value       = module.ec2.public_ip
}

output "sqs_queue_url" {
  description = "URL of the primary SQS scan queue (for platform worker)"
  value       = module.sqs.queue_url
}

output "sqs_dlq_url" {
  description = "URL of the Dead Letter Queue"
  value       = module.sqs.dlq_url
}