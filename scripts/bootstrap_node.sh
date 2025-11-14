#!/bin/bash
set -e

# Get node ID from Terraform template variable
NODE_ID=${node_id}

echo "Starting storage node bootstrap for $NODE_ID..."

export HOME=$${HOME:-/root}

# Update system
yum update -y
yum install -y git wget awscli gcc make

# Install Go 1.23.x
echo "Installing Go 1.23..."
cd /tmp
wget -q https://go.dev/dl/go1.23.2.linux-amd64.tar.gz

# Remove old Go if present
rm -rf /usr/local/go

tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz
rm -f go1.23.2.linux-amd64.tar.gz

export PATH="/usr/local/go/bin:$PATH"

# Set Go environment
export GOPATH=/home/ec2-user/go
export GOCACHE=/tmp/go-build-cache
export GOMODCACHE=$GOPATH/pkg/mod

mkdir -p $GOPATH $GOCACHE
chown -R ec2-user:ec2-user /home/ec2-user/go || true

echo "Go version: $(go version)"

# Create application directory
APP_DIR="/opt/dfs-node"
mkdir -p $APP_DIR
chown -R ec2-user:ec2-user $APP_DIR

# Remove existing directory if it exists (from previous failed runs)
if [ -d "$APP_DIR/.git" ]; then
    echo "Removing existing repository..."
    rm -rf $APP_DIR/*
fi

cd $APP_DIR

# Clone repository
echo "Cloning repository..."
git clone https://github.com/timskillet/distributed-filestore.git . || {
    echo "ERROR: Failed to clone repository"
    exit 1
}

# Get EC2 instance metadata
INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id)
PRIVATE_IP=$(curl -s http://169.254.169.254/latest/meta-data/local-ipv4)

# Set environment variables (escape $ for templatefile)
export AWS_REGION=$${AWS_REGION:-us-east-1}
export CHUNK_METADATA_TABLE=$${CHUNK_METADATA_TABLE:-dfs-chunk-metadata}
export NODE_REGISTRY_TABLE=$${NODE_REGISTRY_TABLE:-dfs-node-registry}
export NODE_ID=$NODE_ID
export NODE_PORT=$${NODE_PORT:-8080}
export REPLICATION_FACTOR=$${REPLICATION_FACTOR:-2}
export NODE_HEARTBEAT_INTERVAL=$${NODE_HEARTBEAT_INTERVAL:-30}
export NODE_HEARTBEAT_TIMEOUT=$${NODE_HEARTBEAT_TIMEOUT:-60}

# Build the node server
echo "Building node server..."
/usr/local/go/bin/go mod download || {
    echo "ERROR: Failed to download Go modules"
    exit 1
}

/usr/local/go/bin/go build -buildvcs=false -o dfs-node ./cmd/dfs-node || {
    echo "ERROR: Failed to build node server"
    exit 1
}

chmod +x dfs-node
chown ec2-user:ec2-user dfs-node

echo "Binary created: $APP_DIR/dfs-node"

# Create chunks directory
mkdir -p /opt/dfs/chunks
chown ec2-user:ec2-user /opt/dfs/chunks

# Create systemd service
echo "Creating systemd service..."
tee /etc/systemd/system/dfs-node.service > /dev/null <<EOF
[Unit]
Description=DFS Storage Node
After=network.target

[Service]
Type=simple
User=ec2-user
WorkingDirectory=$APP_DIR
Environment="PATH=/usr/local/go/bin:/usr/bin:/bin"
Environment="HOME=/home/ec2-user"
Environment="GOPATH=/home/ec2-user/go"
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

systemctl daemon-reload
systemctl enable dfs-node
systemctl start dfs-node

sleep 2
if systemctl is-active --quiet dfs-node; then
    echo "✅ Storage node bootstrap complete! Service is running."
else
    echo "⚠️ Service may not be active. Check logs:"
    systemctl status dfs-node --no-pager -l || true
fi

echo "Node ID: $NODE_ID"
echo "Instance ID: $INSTANCE_ID"
echo "Private IP: $PRIVATE_IP"
echo "Check status with: sudo systemctl status dfs-node"
echo "View logs with: sudo journalctl -u dfs-node -f"