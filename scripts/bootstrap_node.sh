#!/bin/bash
set -e

# Get node ID from Terraform template variable
NODE_ID=${node_id:-node-0}

echo "Starting storage node bootstrap for $NODE_ID..."

# Update system
sudo yum update -y
sudo yum install -y golang git awscli

# Install Go 1.23+ if not available
if ! command -v go &> /dev/null || [ "$(go version | awk '{print $3}' | cut -d. -f2)" -lt 23 ]; then
    echo "Installing Go 1.23..."
    wget -q https://go.dev/dl/go1.23.2.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
fi

# Create application directory
APP_DIR="/opt/dfs-node"
sudo mkdir -p $APP_DIR
cd $APP_DIR

# Clone repository (update with your actual repo URL)
# For now, we'll assume the code is already there or will be copied
# git clone https://github.com/yourusername/distributed-filestore.git .

# Get EC2 instance metadata
INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id)
PRIVATE_IP=$(curl -s http://169.254.169.254/latest/meta-data/local-ipv4)

# Set environment variables
export AWS_REGION=${AWS_REGION:-us-east-1}
export CHUNK_METADATA_TABLE=${CHUNK_METADATA_TABLE:-dfs-chunk-metadata}
export NODE_REGISTRY_TABLE=${NODE_REGISTRY_TABLE:-dfs-node-registry}
export NODE_ID=$NODE_ID
export NODE_PORT=${NODE_PORT:-8080}
export REPLICATION_FACTOR=${REPLICATION_FACTOR:-2}
export NODE_HEARTBEAT_INTERVAL=${NODE_HEARTBEAT_INTERVAL:-30}
export NODE_HEARTBEAT_TIMEOUT=${NODE_HEARTBEAT_TIMEOUT:-60}

# Build the node server
echo "Building node server..."
go mod download
go build -o dfs-node ./cmd/dfs-node

# Create chunks directory
sudo mkdir -p /opt/dfs/chunks
sudo chown ec2-user:ec2-user /opt/dfs/chunks

# Create systemd service
sudo tee /etc/systemd/system/dfs-node.service > /dev/null <<EOF
[Unit]
Description=DFS Storage Node
After=network.target

[Service]
Type=simple
User=ec2-user
WorkingDirectory=$APP_DIR
Environment="AWS_REGION=$AWS_REGION"
Environment="CHUNK_METADATA_TABLE=$CHUNK_METADATA_TABLE"
Environment="NODE_REGISTRY_TABLE=$NODE_REGISTRY_TABLE"
Environment="NODE_ID=$NODE_ID"
Environment="NODE_PORT=$NODE_PORT"
Environment="REPLICATION_FACTOR=$REPLICATION_FACTOR"
Environment="NODE_HEARTBEAT_INTERVAL=$NODE_HEARTBEAT_INTERVAL"
Environment="NODE_HEARTBEAT_TIMEOUT=$NODE_HEARTBEAT_TIMEOUT"
ExecStart=$APP_DIR/dfs-node
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable dfs-node
sudo systemctl start dfs-node

echo "Storage node bootstrap complete!"
echo "Node ID: $NODE_ID"
echo "Instance ID: $INSTANCE_ID"
echo "Private IP: $PRIVATE_IP"
echo "Check status with: sudo systemctl status dfs-node"
echo "View logs with: sudo journalctl -u dfs-node -f"

