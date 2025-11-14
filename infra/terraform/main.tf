######################
# Networking
######################

# Create a VPC
resource "aws_vpc" "dfs_vpc" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "dfs-vpc"
  }
}

# Create a public subnet
resource "aws_subnet" "dfs_subnet" {
  vpc_id                  = aws_vpc.dfs_vpc.id
  cidr_block              = "10.0.1.0/24"
  availability_zone       = "us-east-1a"
  map_public_ip_on_launch = true

  tags = {
    Name = "dfs-subnet"
  }
}

# Internet Gateway
resource "aws_internet_gateway" "dfs_igw" {
  vpc_id = aws_vpc.dfs_vpc.id

  tags = {
    Name = "dfs-igw"
  }
}

# Route Table
resource "aws_route_table" "dfs_route_table" {
  vpc_id = aws_vpc.dfs_vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.dfs_igw.id
  }

  tags = {
    Name = "dfs-route-table"
  }
}

# Associate Route Table with Subnet
resource "aws_route_table_association" "dfs_route_assoc" {
  subnet_id      = aws_subnet.dfs_subnet.id
  route_table_id = aws_route_table.dfs_route_table.id
}

# Security Group for API Server
resource "aws_security_group" "dfs_api_sg" {
  vpc_id = aws_vpc.dfs_vpc.id
  name   = "dfs-api-sg"

  ingress {
    description = "Allow API traffic"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "Allow SSH access"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "dfs-api-sg"
  }
}

# Security Group for Storage Nodes
resource "aws_security_group" "dfs_node_sg" {
  vpc_id = aws_vpc.dfs_vpc.id
  name   = "dfs-node-sg"

  # Allow node-to-node communication for replication
  ingress {
    description     = "Allow node-to-node communication"
    from_port       = 8080
    to_port         = 8090
    protocol        = "tcp"
    self            = true
  }

  # Allow API server to access nodes
  ingress {
    description     = "Allow API server to access nodes"
    from_port       = 8080
    to_port         = 8090
    protocol        = "tcp"
    security_groups = [aws_security_group.dfs_api_sg.id]
  }

  ingress {
    description = "Allow SSH access"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "dfs-node-sg"
  }
}

######################
# IAM Role
######################
resource "aws_iam_role" "dfs_ec2_role" {
  name = "dfs-ec2-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "s3_policy" {
  role       = aws_iam_role.dfs_ec2_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonS3FullAccess"
}

resource "aws_iam_role_policy_attachment" "dynamo_policy" {
  role       = aws_iam_role.dfs_ec2_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess"
}

resource "aws_iam_instance_profile" "dfs_instance_profile" {
  name = "dfs-instance-profile"
  role = aws_iam_role.dfs_ec2_role.name
}

######################
# DynamoDB Tables
######################
# Chunk Metadata Table - stores chunk locations with support for multiple replicas
# Uses composite range key: chunk_index#node_id to allow multiple replicas per chunk
resource "aws_dynamodb_table" "dfs_chunk_metadata" {
  name         = "dfs-chunk-metadata"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "file_id"
  range_key    = "chunk_replica_key"  # Format: "chunk_index#node_id"

  attribute {
    name = "file_id"
    type = "S"
  }

  attribute {
    name = "chunk_replica_key"
    type = "S"
  }

  # Global Secondary Index to query by node_id
  global_secondary_index {
    name            = "node-id-index"
    hash_key        = "node_id"
    projection_type = "ALL"
  }

  attribute {
    name = "node_id"
    type = "S"
  }

  tags = {
    Name = "dfs-chunk-metadata"
  }
}

# Node Registry Table - service discovery for storage nodes
resource "aws_dynamodb_table" "dfs_node_registry" {
  name         = "dfs-node-registry"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "node_id"

  attribute {
    name = "node_id"
    type = "S"
  }

  # TTL for automatic cleanup of stale nodes
  ttl {
    attribute_name = "heartbeat_ts"
    enabled        = true
  }

  tags = {
    Name = "dfs-node-registry"
  }
}

######################
# S3 Bucket
######################
resource "random_id" "bucket_suffix" {
  byte_length = 4
}

resource "aws_s3_bucket" "dfs_backup" {
  bucket = "dfs-backup-${random_id.bucket_suffix.hex}"

  tags = {
    Name = "dfs-backup"
  }
}

######################
# EC2 Instance + EBS Volume
######################
# API Server Instance
resource "aws_instance" "dfs_api" {
  ami                    = var.ami_id != "" ? var.ami_id : data.aws_ami.ubuntu.id
  instance_type          = "t3.micro"
  subnet_id              = aws_subnet.dfs_subnet.id
  vpc_security_group_ids = [aws_security_group.dfs_api_sg.id]
  iam_instance_profile   = aws_iam_instance_profile.dfs_instance_profile.name
  user_data              = file("../../scripts/bootstrap_api.sh")
  key_name               = var.key_pair_name

  tags = {
    Name = "DFS-API"
  }
}

# Storage Node Instances (multiple nodes for replication)
resource "aws_instance" "dfs_node" {
  count                  = var.node_count
  ami                    = var.node_ami_id != "" ? var.node_ami_id : data.aws_ami.amazon_linux_2.id
  instance_type          = var.instance_type
  subnet_id              = aws_subnet.dfs_subnet.id
  vpc_security_group_ids = [aws_security_group.dfs_node_sg.id]
  key_name               = var.key_pair_name
  iam_instance_profile   = aws_iam_instance_profile.dfs_instance_profile.name
  user_data              = templatefile("../../scripts/bootstrap_node.sh", {
    node_id = "node-${count.index}"
  })

  tags = {
    Name = "dfs-node-${count.index}"
  }
}

# EBS Volumes for each storage node
resource "aws_ebs_volume" "dfs_data_volume" {
  count             = var.node_count
  availability_zone = aws_instance.dfs_node[count.index].availability_zone
  size              = var.volume_size

  tags = {
    Name = "dfs-data-volume-${count.index}"
  }
}

resource "aws_volume_attachment" "dfs_data_attach" {
  count       = var.node_count
  device_name = "/dev/xvdf"
  volume_id   = aws_ebs_volume.dfs_data_volume[count.index].id
  instance_id = aws_instance.dfs_node[count.index].id
}