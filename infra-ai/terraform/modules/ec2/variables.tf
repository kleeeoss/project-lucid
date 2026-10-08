variable "instance_type" {
  type        = string
  default     = "m7i-flex.large"
  description = "EC2 instance size"
}

variable "volume_size" {
  type        = number
  default     = 30
  description = "Root gp3 EBS volume size in GB"
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
  default     = "showcase"
  description = "Environment identifier tag"
}
