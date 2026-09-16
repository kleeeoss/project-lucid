variable "vpc_cidr" {
  type        = string
  default     = "10.0.0.0/16"
  description = "CIDR block for the VPC"
}

variable "public_subnet_cidr" {
  type        = string
  default     = "10.0.1.0/24"
  description = "CIDR block for the public subnet"
}

variable "availability_zone" {
  type        = string
  description = "Availability zone for the public subnet"
}

variable "admin_cidr" {
  type        = string
  default     = "0.0.0.0/0"
  description = "CIDR block permitted to SSH into EC2 (set to your IP in production)"
}

variable "environment" {
  type        = string
  default     = "development"
  description = "Environment identifier tag"
}