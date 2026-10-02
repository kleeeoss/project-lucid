output "aws_region" {
  description = "Active AWS deployment region"
  value       = var.aws_region
}

output "instance_id" {
  description = "EC2 Instance ID (required by scripts/demo.sh)"
  value       = module.ec2.instance_id
}

output "ec2_public_ip" {
  description = "Persistent Public Elastic IP of the Lucid-CI EC2 host"
  value       = module.ec2.public_ip
}

output "domain" {
  description = "Configured Showcase Domain (empty if using bare IP)"
  value       = var.domain
}

output "sqs_queue_url" {
  description = "URL of the primary SQS scan queue"
  value       = module.sqs.queue_url
}

output "sqs_dlq_url" {
  description = "URL of the Dead Letter Queue"
  value       = module.sqs.dlq_url
}
