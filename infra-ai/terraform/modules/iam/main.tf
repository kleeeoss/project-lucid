resource "aws_iam_role" "ec2_role" {
  name = "lucid-ci-ec2-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "lucid-ci-ec2-role"
    Environment = var.environment
    Project     = "lucid-ci"
  }
}

resource "aws_iam_policy" "sqs_access" {
  name        = "lucid-ci-sqs-access-policy"
  description = "Least-privilege SQS access for Lucid-CI worker"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage",
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl",
          "sqs:ChangeMessageVisibility"
        ]
        Resource = [
          var.sqs_queue_arn,
          var.sqs_dlq_arn
        ]
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "sqs_attach" {
  role       = aws_iam_role.ec2_role.name
  policy_arn = aws_iam_policy.sqs_access.arn
}

resource "aws_iam_instance_profile" "instance_profile" {
  name = "lucid-ci-instance-profile"
  role = aws_iam_role.ec2_role.name
}