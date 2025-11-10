######################
# Networking
######################
resource "aws_vps" "dfs_vpc" {
    cidr_block = "10.0.0/16"
    tags = {
        Name = "dfs-vps"
    }
}

resource "aws_subnet" "dfs_subnet" {
    vpc_id = aws_vpc.dfs_vpc.id
    cidr_block = "10.0.1.0/24"
    map_public_ip_on_launch = true
    tags = {
        Name = "dfs-subnet"
    }
}

resource "aws_security group" "dfs_sg" {
    vpc_id = aws_vpc.dfs_vpc.id
    name = "dfs-sg"

    ingress {
        from_port = 8080
        to_port = 8080
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    ingress {
        from_port = 22
        to_port = 22
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }

    tags = {
        Name = "dfs-sg"
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
# DynamoDB Table
######################
resource "aws_dynamodb_table" "dfs_metadata" {
  name         = "dfs-metadata"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "file_id"

  attribute {
    name = "file_id"
    type = "S"
  }

  tags = { Name = "dfs-metadata" }
}

######################
# S3 Bucket
######################
resource "random_id" "bucket_suffix" {
  byte_length = 4
}

resource "aws_s3_bucket" "dfs_backup" {
  bucket = "dfs-backup-${random_id.bucket_suffix.hex}"

  tags = { Name = "dfs-backup" }
}

######################
# EC2 + EBS Volume
######################
resource "aws_instance" "dfs_node" {
  ami                    = "ami-0c55b159cbfafe1f0" # Amazon Linux 2 (update per region)
  instance_type          = var.instance_type
  subnet_id              = aws_subnet.dfs_subnet.id
  vpc_security_group_ids = [aws_security_group.dfs_sg.id]
  key_name               = var.key_pair_name
  iam_instance_profile   = aws_iam_instance_profile.dfs_instance_profile.name
  user_data              = file("scripts/bootstrap.sh")

  tags = { Name = "dfs-node" }
}

resource "aws_ebs_volume" "dfs_data_volume" {
  availability_zone = aws_instance.dfs_node.availability_zone
  size              = var.volume_size
  tags = { Name = "dfs-data" }
}

resource "aws_volume_attachment" "dfs_data_attach" {
  device_name = "/dev/xvdf"
  volume_id   = aws_ebs_volume.dfs_data_volume.id
  instance_id = aws_instance.dfs_node.id
}