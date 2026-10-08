resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "lucid-ci-vpc"
    Environment = var.environment
    Project     = "lucid-ci"
  }
}

resource "aws_internet_gateway" "gw" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name        = "lucid-ci-igw"
    Environment = var.environment
    Project     = "lucid-ci"
  }
}

resource "aws_subnet" "public" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = var.public_subnet_cidr
  availability_zone       = var.availability_zone
  map_public_ip_on_launch = true

  tags = {
    Name        = "lucid-ci-public-subnet"
    Environment = var.environment
    Project     = "lucid-ci"
  }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.gw.id
  }

  tags = {
    Name        = "lucid-ci-public-rt"
    Environment = var.environment
    Project     = "lucid-ci"
  }
}

resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public.id
  route_table_id = aws_route_table.public.id
}

resource "aws_security_group" "lucid_host" {
  name        = "lucid-ci-host-sg"
  description = "Security group for Lucid-CI unified host (Caddy, Platform, AI)"
  vpc_id      = aws_vpc.main.id

  # Inbound SSH (port 22) restricted to administrator CIDR
  ingress {
    description = "SSH for administrator shell access"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.admin_cidr]
  }

  # Inbound HTTP (port 80) for Caddy ACME challenge and bare IP access
  ingress {
    description = "HTTP for ACME TLS challenge and bare IP traffic"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Inbound HTTPS (port 443) for GitHub Webhooks and Dashboard
  ingress {
    description = "HTTPS for GitHub Webhooks and Dashboard"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Outbound: full egress permitted (needed for Docker builds, Groq/Gemini APIs, GitHub API)
  egress {
    description = "Full outbound egress"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "lucid-ci-host-sg"
    Environment = var.environment
    Project     = "lucid-ci"
  }
}
