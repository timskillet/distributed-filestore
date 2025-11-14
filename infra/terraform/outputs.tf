##########################
# Terraform Outputs
##########################

# API Server outputs
output "dfs_api_public_ip" {
  description = "Public IP address of the DFS API server"
  value       = aws_instance.dfs_api.public_ip
}

output "dfs_api_public_dns" {
  description = "Public DNS of the DFS API server"
  value       = aws_instance.dfs_api.public_dns
}

# Storage Node outputs
output "dfs_node_public_ips" {
  description = "Public IP addresses of the DFS storage nodes"
  value       = aws_instance.dfs_node[*].public_ip
}

output "dfs_node_public_dns" {
  description = "Public DNS of the DFS storage nodes"
  value       = aws_instance.dfs_node[*].public_dns
}

# S3 bucket name
output "dfs_backup_bucket" {
  description = "S3 bucket used for DFS backups"
  value       = aws_s3_bucket.dfs_backup.bucket
}

# DynamoDB table names
output "dfs_chunk_metadata_table" {
  description = "DynamoDB table used for chunk metadata"
  value       = aws_dynamodb_table.dfs_chunk_metadata.name
}

output "dfs_node_registry_table" {
  description = "DynamoDB table used for node registry"
  value       = aws_dynamodb_table.dfs_node_registry.name
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
