##########################
# Terraform Outputs
##########################

# EC2 public IP
output "dfs_node_public_ip" {
  description = "Public IP address of the DFS EC2 node"
  value       = aws_instance.dfs_node.public_ip
}

# EC2 public DNS
output "dfs_node_public_dns" {
  description = "Public DNS of the DFS EC2 node"
  value       = aws_instance.dfs_node.public_dns
}

# S3 bucket name
output "dfs_backup_bucket" {
  description = "S3 bucket used for DFS backups"
  value       = aws_s3_bucket.dfs_backup.bucket
}

# DynamoDB table name
output "dfs_dynamodb_table" {
  description = "DynamoDB table used for file metadata"
  value       = aws_dynamodb_table.dfs_metadata.name
}

# IAM instance profile
output "dfs_instance_profile" {
  description = "IAM instance profile attached to EC2"
  value       = aws_iam_instance_profile.dfs_instance_profile.name
}

# VPC and subnet info
output "dfs_vpc_id" {
  description = "VPC ID for DFS network"
  value       = aws_vpc.dfs_vpc.id
}

output "dfs_subnet_id" {
  description = "Subnet ID for DFS public subnet"
  value       = aws_subnet.dfs_subnet.id
}
