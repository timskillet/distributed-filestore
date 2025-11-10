variable "aws_region" {
    description = "AWS region to deploy resources"
    type = string
    default = "us-east-1"
}

variable "aws_profile" {
    description = "AWS CLI profile name"
    type = string
    default = "dfs-project"
}

variable "instance_type {
    description = "EC2 instance type"
    type = string
    default = "t2.micro"
}

variable "key_pair_name" {
    description = "Existing key pair for SSH"
    type = string
}

variable "volume_size" {
    description = "EBS volume size"
    type = number
    default = 5
}