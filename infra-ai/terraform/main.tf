provider "aws" {
  region = var.aws_region
}

# Pick first available AZ in the selected region
data "aws_availability_zones" "available" {
  state = "available"
}

module "vpc" {
  source            = "./modules/vpc"
  availability_zone = data.aws_availability_zones.available.names[0]
  admin_cidr        = var.admin_cidr
  environment       = var.environment
}

module "sqs" {
  source      = "./modules/sqs"
  environment = var.environment
}

module "iam" {
  source        = "./modules/iam"
  sqs_queue_arn = module.sqs.queue_arn
  sqs_dlq_arn   = module.sqs.dlq_arn
  environment   = var.environment
}

module "ec2" {
  source                = "./modules/ec2"
  instance_type         = var.instance_type
  subnet_id             = module.vpc.public_subnet_id
  security_group_id     = module.vpc.security_group_id
  instance_profile_name = module.iam.instance_profile_name
  key_name              = var.key_name
  environment           = var.environment
}