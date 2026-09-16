variable "instance_type" {
  type        = string
  default     = "t3.xlarge"
  description = "EC2 instance size (Primary V1 baseline: t3.xlarge, Fallback: c6i.large)"
}

variable "subnet_id" {
  type        = string
  description = "VPC public subnet ID"
}

variable "security_group_id" {
  type        = string
  description = "Host security group ID"
}

variable "instance_profile_name" {
  type        = string
  description = "IAM instance profile name"
}

variable "key_name" {
  type        = string
  default     = ""
  description = "Optional AWS EC2 key pair name for SSH access"
}

variable "environment" {
  type        = string
  default     = "development"
  description = "Environment identifier tag"
}