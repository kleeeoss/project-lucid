# Look up latest official Ubuntu 24.04 LTS AMI (Noble Numbat)
data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical official AWS account ID

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

resource "aws_instance" "host" {
  ami                         = data.aws_ami.ubuntu.id
  instance_type               = var.instance_type
  subnet_id                   = var.subnet_id
  vpc_security_group_ids      = [var.security_group_id]
  iam_instance_profile        = var.instance_profile_name
  key_name                    = var.key_name != "" ? var.key_name : null
  user_data                   = file("${path.module}/userdata.sh")

  # CRITICAL: Ensures an OS shutdown transitions instance to STOPPED (not TERMINATED)
  instance_initiated_shutdown_behavior = "stop"

  # CRITICAL: Enables IMDSv2 and exposes instance tags to guest OS.
  # http_put_response_hop_limit is set to 2 so containers on Docker bridge networks can reach IMDS for IAM credentials.
  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required" # IMDSv2 mandatory
    instance_metadata_tags      = "enabled"  # Exposes DemoRuntimeMinutes tag to guest OS
    http_put_response_hop_limit = 2          # Allows Docker bridge container hops to retrieve IAM credentials
  }

  root_block_device {
    volume_size           = var.volume_size
    volume_type           = "gp3"
    delete_on_termination = true
    encrypted             = true
  }

  tags = {
    Name               = "lucid-ci-host"
    Environment        = var.environment
    Project            = "lucid-ci"
    DemoRuntimeMinutes = "120" # Initial default tag; updated by demo.sh prior to each start
  }
}

# Elastic IP for persistent public IPv4 address across stop/start cycles
resource "aws_eip" "host_ip" {
  instance = aws_instance.host.id
  domain   = "vpc"

  tags = {
    Name        = "lucid-ci-host-eip"
    Environment = var.environment
    Project     = "lucid-ci"
  }
}
