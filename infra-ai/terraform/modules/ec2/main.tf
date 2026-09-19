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
  ami                  = data.aws_ami.ubuntu.id
  instance_type        = var.instance_type
  subnet_id            = var.subnet_id
  vpc_security_group_ids = [var.security_group_id]
  iam_instance_profile = var.instance_profile_name
  key_name             = var.key_name != "" ? var.key_name : null

  root_block_device {
    volume_size           = 50 # GB
    volume_type           = "gp3"
    delete_on_termination = true
  }

  tags = {
    Name        = "lucid-ci-host"
    Environment = var.environment
    Project     = "lucid-ci"
  }
}

# Elastic IP for persistent public DNS and webhook targeting
resource "aws_eip" "host_ip" {
  instance = aws_instance.host.id
  domain   = "vpc"

  tags = {
    Name        = "lucid-ci-host-eip"
    Environment = var.environment
    Project     = "lucid-ci"
  }
}