variable "aws_region" {
  type        = string
  default     = "ap-southeast-2"
  description = "AWS target region for this deployment account (ap-southeast-2 / Sydney)"
}

variable "environment" {
  type        = string
  default     = "showcase"
  description = "Environment identifier tag (showcase | staging | production)"
}

variable "instance_type" {
  type        = string
  default     = "m7i-flex.large"
  description = "EC2 instance size (2 vCPU, 8.0 GiB RAM). Fallbacks: c7i-flex.large, t3.medium"
}

variable "admin_cidr" {
  type        = string
  default     = "0.0.0.0/0"
  description = "CIDR allowed SSH access (port 22)"
}

variable "key_name" {
  type        = string
  default     = ""
  description = "Optional AWS EC2 key pair name for SSH access"
}

variable "domain" {
  type        = string
  default     = ""
  description = "Optional FQDN for Showcase (e.g. lucid-demo.duckdns.org). If empty, serves plain HTTP on Elastic IP"
}
