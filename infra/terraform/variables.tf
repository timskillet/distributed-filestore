##########################
# Terraform Variables
##########################

# AWS region
variable "aws_region" {
  description = "AWS region for resource deployment"
  type        = string
  default     = "us-east-1"
}

# Instance type
variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t2.micro"
}

# EC2 key pair name
variable "key_pair_name" {
  description = "Existing EC2 key pair for SSH access"
  type        = string
}

# Size of attached EBS volume
variable "volume_size" {
  description = "EBS data volume size in GB"
  type        = number
  default     = 5
}

# Project tags
variable "project_name" {
  description = "Tag name for project identification"
  type        = string
  default     = "distributed-file-storage"
}

# Number of storage nodes
variable "node_count" {
  description = "Number of storage node instances to create"
  type        = number
  default     = 3
}

# Replication factor
variable "replication_factor" {
  description = "Number of replicas per chunk"
  type        = number
  default     = 2
}

# AMI ID for API server
variable "ami_id" {
  description = "AMI ID for API server (Ubuntu)"
  type        = string
  default     = "ami-0c55b159cbfafe1f0" # Amazon Linux 2, update for Ubuntu if needed
}

# AMI ID for storage nodes
variable "node_ami_id" {
  description = "AMI ID for storage nodes (Amazon Linux 2)"
  type        = string
  default     = ""
}
