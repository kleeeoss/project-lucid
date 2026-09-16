variable "aws_region" {
  type        = string
  default     = "us-east-1"
  description = "AWS target region"
}

variable "environment" {
  type        = string
  default     = "development"
  description = "Environment identifier tag (development | staging | production)"
}

variable "instance_type" {
  type        = string
  default     = "t3.xlarge"
  description = "Primary V1 instance size (4 vCPU, 16GB RAM for Docker + gVisor + PostgreSQL)"
}

variable "admin_cidr" {
  type        = string
  default     = "0.0.0.0/0"
  description = "CIDR allowed SSH access (change to your specific public IP in production)"
}

variable "key_name" {
  type        = string
  default     = ""
  description = "Optional AWS EC2 key pair name for SSH access"
}