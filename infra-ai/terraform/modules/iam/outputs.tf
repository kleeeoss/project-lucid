output "instance_profile_name" {
  value = aws_iam_instance_profile.instance_profile.name
}

output "role_arn" {
  value = aws_iam_role.ec2_role.arn
}