output "instance_public_ip" {
  value = aws_instance.dfs_node.public_ip
}

output "s3_bucket_name" {
  value = aws_s3_bucket.dfs_backup.bucket
}

output "dynamodb_table_name" {
  value = aws_dynamodb_table.dfs_metadata.name
}
