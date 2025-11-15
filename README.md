# Distributed File Store

A distributed file storage system built with Go that splits files into chunks and distributes them across multiple storage nodes with replication support. The system uses AWS DynamoDB for service discovery and metadata management, and can be deployed on AWS EC2 instances using Terraform.

## Overview

This project implements a distributed file storage system with the following features:

- **File Chunking**: Files are split into configurable-sized chunks (default 1KB)
- **Distributed Storage**: Chunks are distributed across multiple storage nodes
- **Replication**: Configurable replication factor for data redundancy
- **Service Discovery**: Automatic node registration and discovery using DynamoDB
- **RESTful API**: HTTP-based API for file upload and download operations
- **AWS Integration**: Designed for AWS deployment with Terraform provisioning

## Technologies Used

- **Go 1.23+**: Core application language
- **AWS DynamoDB**: Service discovery and chunk metadata storage
- **AWS EC2**: Compute instances for API server and storage nodes
- **AWS EBS**: Persistent storage volumes for storage nodes
- **Terraform**: Infrastructure as Code for AWS resource provisioning
- **HTTP/REST**: API communication protocol
- **AWS SDK for Go v2**: AWS service integration

## Architecture

The system consists of three main components:

1. **API Server** (`cmd/dfs-api`): Handles file upload/download requests, manages chunk distribution, and coordinates with storage nodes
2. **Storage Nodes** (`cmd/dfs-node`): Store file chunks locally and register themselves in DynamoDB for service discovery
3. **Client** (`cmd/dfs-client`): Command-line tool for uploading and downloading files

### Data Flow

1. **Upload**: Client → API Server → Chunks distributed to storage nodes → Metadata stored in DynamoDB
2. **Download**: Client → API Server → Queries DynamoDB for chunk locations → Retrieves chunks from nodes → Reassembles file

### Service Discovery

Storage nodes automatically register themselves in DynamoDB when they start up. They send periodic heartbeats to indicate they're active. The API server queries DynamoDB to discover available nodes for chunk distribution.

## Prerequisites

### For Local Development

- **Go 1.23.2** or later
- **AWS Account** with appropriate permissions
- **AWS CLI** configured with credentials

### For AWS Deployment

- **Terraform** 1.0+ installed
- **AWS Account** with billing enabled
- **EC2 Key Pair** created in your target AWS region
- **AWS CLI** configured

### AWS Permissions Required

- DynamoDB: Create tables, read/write operations
- EC2: Launch instances, manage security groups, create EBS volumes
- IAM: Create roles and policies
- VPC: Create networking resources

## Local Development Setup

### 1. Clone the Repository

```bash
git clone https://github.com/timskillet/distributed-filestore.git
cd distributed-filestore
```

### 2. Set Up AWS Credentials

Configure AWS credentials using one of these methods:

```bash
# Option 1: AWS CLI
aws configure

# Option 2: Environment variables
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_REGION=us-east-1
```

### 3. Create DynamoDB Tables

The system requires two DynamoDB tables. You can create them manually via AWS Console or use Terraform:

**Option A: Using Terraform (Recommended)**

```bash
cd infra/terraform
terraform init
terraform apply -target=aws_dynamodb_table.dfs_chunk_metadata
terraform apply -target=aws_dynamodb_table.dfs_node_registry
```

**Option B: Manual Creation via AWS Console**

1. **Chunk Metadata Table**:

   - Table name: `dfs-chunk-metadata`
   - Partition key: `file_id` (String)
   - Sort key: `chunk_replica_key` (String)
   - Billing mode: On-demand
   - Add Global Secondary Index: `node-id-index` with partition key `node_id` (String)

2. **Node Registry Table**:
   - Table name: `dfs-node-registry`
   - Partition key: `node_id` (String)
   - Billing mode: On-demand
   - Enable TTL with attribute name: `ttl`

### 4. Set Environment Variables

```bash
export AWS_REGION=us-east-1
export CHUNK_METADATA_TABLE=dfs-chunk-metadata
export NODE_REGISTRY_TABLE=dfs-node-registry
export REPLICATION_FACTOR=2
export REPLICATION_STRATEGY=sync
export REPLICATION_TIMEOUT=30
export NODE_HEARTBEAT_INTERVAL=30
export NODE_HEARTBEAT_TIMEOUT=60
```

### 5. Build the Components

```bash
go mod download
go build -o dfs-api ./cmd/dfs-api
go build -o dfs-node ./cmd/dfs-node
go build -o dfs-client ./cmd/dfs-client
```

### 6. Start Storage Nodes

Start at least 2 nodes (to meet the default replication factor of 2). Open separate terminals:

**Terminal 1 - Node A:**

```bash
export NODE_ID=nodeA
export NODE_PORT=8081
./dfs-node
```

**Terminal 2 - Node B:**

```bash
export NODE_ID=nodeB
export NODE_PORT=8082
./dfs-node
```

**Terminal 3 - Node C (optional):**

```bash
export NODE_ID=nodeC
export NODE_PORT=8083
./dfs-node
```

### 7. Start the API Server

```bash
./dfs-api
```

The API server will start on port 8080 and display configuration information.

### 8. Use the Client

**Upload a file:**

```bash
./dfs-client upload http://localhost:8080 ./test-files/test.txt
```

**Upload with custom chunk size:**

```bash
./dfs-client upload -chunk-size=2048 http://localhost:8080 ./test-files/test.txt
```

**Download a file:**

```bash
# Use the file_id returned from the upload command
./dfs-client download http://localhost:8080 <file_id> ./downloaded.txt
```

## AWS Deployment

### 1. Prerequisites

- Terraform installed (`terraform --version`)
- AWS account with appropriate permissions
- EC2 key pair created in your target region

### 2. Create EC2 Key Pair

If you don't have a key pair, create one in the AWS Console:

- Go to EC2 → Key Pairs → Create Key Pair
- Note the key pair name (e.g., `dfs-key`)

### 3. Configure Terraform Variables

Edit `infra/terraform/variables.tf` or set environment variables:

```bash
export TF_VAR_key_pair_name=your-key-pair-name
export TF_VAR_aws_region=us-east-1
export TF_VAR_instance_type=t3.micro
export TF_VAR_node_count=3
export TF_VAR_replication_factor=2
export TF_VAR_volume_size=5
```

Or create a `terraform.tfvars` file:

```hcl
key_pair_name      = "your-key-pair-name"
aws_region         = "us-east-1"
instance_type      = "t3.micro"
node_count         = 3
replication_factor = 2
volume_size        = 5
```

### 4. Initialize and Apply Terraform

```bash
cd infra/terraform
terraform init
terraform plan
terraform apply
```

This will create:

- VPC, subnets, internet gateway, and route tables
- Security groups for API server and storage nodes
- DynamoDB tables (chunk metadata and node registry)
- IAM roles with DynamoDB and EC2 permissions
- EC2 instance for API server (Ubuntu)
- Multiple EC2 instances for storage nodes (Amazon Linux 2)
- EBS volumes attached to storage nodes
- S3 bucket for backups

**Note**: The first deployment may take 10-15 minutes as instances bootstrap and build the application.

### 5. Get Deployment Information

After Terraform completes, get the API server endpoint:

```bash
terraform output dfs_api_public_ip
terraform output dfs_api_public_dns
```

View all outputs:

```bash
terraform output
```

### 6. Verify Services Are Running

SSH into the instances to verify services:

**Check API Server:**

```bash
ssh -i ~/.ssh/your-key.pem ubuntu@<api-server-ip>
sudo systemctl status dfs-api
sudo journalctl -u dfs-api -f
```

**Check Storage Nodes:**

```bash
ssh -i ~/.ssh/your-key.pem ec2-user@<node-ip>
sudo systemctl status dfs-node
sudo journalctl -u dfs-node -f
```

### 7. Test the Deployment

From your local machine:

```bash
# Upload a file
./dfs-client upload http://<api-server-ip>:8080 ./test-files/test.txt

# Download the file (use file_id from upload output)
./dfs-client download http://<api-server-ip>:8080 <file_id> ./downloaded.txt
```

### 8. Clean Up Resources

To destroy all AWS resources:

```bash
cd infra/terraform
terraform destroy
```

**Warning**: This will delete all data stored in the system!

## Configuration

All configuration is done via environment variables with the following defaults:

| Variable                  | Default              | Description                                       |
| ------------------------- | -------------------- | ------------------------------------------------- |
| `AWS_REGION`              | `us-east-1`          | AWS region for resources                          |
| `CHUNK_METADATA_TABLE`    | `dfs-chunk-metadata` | DynamoDB table for chunk metadata                 |
| `NODE_REGISTRY_TABLE`     | `dfs-node-registry`  | DynamoDB table for node registry                  |
| `REPLICATION_FACTOR`      | `2`                  | Number of replicas per chunk                      |
| `REPLICATION_STRATEGY`    | `sync`               | Replication strategy: `sync` or `async`           |
| `REPLICATION_TIMEOUT`     | `30`                 | Replication timeout in seconds                    |
| `NODE_HEARTBEAT_INTERVAL` | `30`                 | Node heartbeat interval in seconds                |
| `NODE_HEARTBEAT_TIMEOUT`  | `60`                 | Node timeout threshold in seconds                 |
| `NODE_ID`                 | Auto-detected        | Node identifier (uses EC2 instance ID if not set) |
| `NODE_PORT`               | Auto-assigned        | Port for storage node (8080-8090 range)           |

### Terraform Variables

| Variable             | Default       | Description                      |
| -------------------- | ------------- | -------------------------------- |
| `key_pair_name`      | **Required**  | EC2 key pair name for SSH access |
| `aws_region`         | `us-east-1`   | AWS region for deployment        |
| `instance_type`      | `t2.micro`    | EC2 instance type                |
| `node_count`         | `3`           | Number of storage node instances |
| `replication_factor` | `2`           | Number of replicas per chunk     |
| `volume_size`        | `5`           | EBS volume size in GB            |
| `ami_id`             | Auto-detected | Ubuntu AMI for API server        |
| `node_ami_id`        | Auto-detected | Amazon Linux 2 AMI for nodes     |

## Usage Examples

### Upload Files

```bash
# Basic upload
./dfs-client upload http://localhost:8080 ./document.pdf

# Upload with custom chunk size (2KB)
./dfs-client upload -chunk-size=2048 http://localhost:8080 ./large-file.zip

# Upload to AWS deployment
./dfs-client upload http://54.123.45.67:8080 ./test-files/test.txt
```

### Download Files

```bash
# Download using file ID
./dfs-client download http://localhost:8080 bd08c6c6-3b5d-4f28-a32a-2bab04d1aba5 ./restored-file.txt

# Download from AWS deployment
./dfs-client download http://54.123.45.67:8080 <file_id> ./downloaded.txt
```

### API Endpoints

The API server exposes the following endpoints:

- `POST /init-upload` - Initialize file upload and get chunk targets
- `POST /finalize-upload` - Finalize upload after all chunks are uploaded
- `GET /download-plan?file_id=<id>` - Get download plan with chunk locations
- `POST /proxy-chunk-upload` - Proxy chunk upload to storage nodes
- `GET /proxy-chunk-download` - Proxy chunk download from storage nodes

## Project Structure

```
distributed-filestore/
├── cmd/
│   ├── dfs-api/          # API server entry point
│   ├── dfs-node/         # Storage node entry point
│   └── dfs-client/       # Client CLI tool
├── internal/
│   ├── api/              # API server handlers and logic
│   ├── node/             # Storage node handlers and logic
│   ├── client/           # Client upload/download logic
│   ├── dynamodb/         # DynamoDB client and operations
│   └── config/           # Configuration management
├── infra/
│   └── terraform/        # Terraform infrastructure code
│       ├── main.tf       # Main infrastructure definitions
│       ├── variables.tf  # Input variables
│       ├── outputs.tf    # Output values
│       └── data.tf       # Data sources (AMI lookups)
├── scripts/              # Bootstrap scripts for EC2
│   ├── bootstrap_api.sh  # API server bootstrap
│   └── bootstrap_node.sh # Storage node bootstrap
├── test-files/           # Test files for upload/download
└── go.mod                # Go module dependencies
```

## Troubleshooting

### Nodes Not Registering

- **Check AWS credentials**: Verify credentials are configured correctly
  ```bash
  aws sts get-caller-identity
  ```
- **Verify DynamoDB tables exist**: Check AWS Console or use Terraform outputs
- **Check node logs**:
  ```bash
  sudo journalctl -u dfs-node -f
  ```
- **Verify IAM permissions**: Nodes need DynamoDB write permissions

### Upload/Download Failures

- **Check API server connectivity**: Ensure API server can reach storage nodes
- **Verify security groups**: Check that ports 8080-8090 are open between API server and nodes
- **Check replication factor**: Ensure `REPLICATION_FACTOR` doesn't exceed number of active nodes
- **View API server logs**:
  ```bash
  sudo journalctl -u dfs-api -f
  ```

### DynamoDB Errors

- **Verify table names**: Check that environment variables match actual table names
- **Check IAM roles**: Ensure IAM roles have DynamoDB permissions
- **Verify region**: Ensure tables are in the correct AWS region
- **Check table structure**: Verify tables have correct keys and indexes

### Terraform Issues

- **Key pair not found**: Ensure key pair exists in the target region
- **AMI not found**: Terraform will auto-detect AMIs, but you can specify custom AMI IDs
- **Insufficient permissions**: Ensure your AWS credentials have necessary permissions
- **Resource limits**: Check AWS account limits for EC2 instances, EBS volumes, etc.

### Local Development Issues

- **Port conflicts**: Ensure ports 8080-8090 are available
- **Go version**: Ensure Go 1.23.2 or later is installed
- **Module dependencies**: Run `go mod download` if build fails
- **DynamoDB access**: Ensure local AWS credentials have DynamoDB access

## Cost Considerations

### AWS Free Tier

- **EC2**: 750 hours/month of t2.micro instances (first 12 months)
- **DynamoDB**: 25 GB storage, 25 WCU, 25 RCU (first 12 months)
- **EBS**: 30 GB free storage (first 12 months)

### Estimated Monthly Costs (Outside Free Tier)

- **EC2**: ~$7-15/month per t3.micro instance (depending on usage)
- **DynamoDB**: Pay-per-request pricing (very low for small workloads)
- **EBS**: ~$0.50/month per 5GB volume
- **Data Transfer**: First 100GB/month free, then $0.09/GB

**Note**: Always monitor your AWS usage and set up billing alerts!

## Security Considerations

- **IAM Roles**: Use IAM roles instead of access keys when possible
- **Security Groups**: Restrict access to necessary ports only
- **Key Pairs**: Keep EC2 key pairs secure and don't commit them to version control
- **DynamoDB**: Consider enabling encryption at rest
- **VPC**: Use private subnets for storage nodes in production
