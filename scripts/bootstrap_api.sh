#!/bin/bash
set -e

echo "Starting API server bootstrap..."

# Update system
sudo apt update -y
sudo apt install -y git wget

# Install Go 1.23+
echo "Installing Go 1.23..."
cd /tmp
wget -q https://go.dev/dl/go1.23.2.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz
rm -f go1.23.2.linux-amd64.tar.gz

# Ensure Go is in PATH for this script
export PATH=/usr/local/go/bin:$PATH

# Set GOPATH for Go module cache (required for go mod commands)
export GOPATH=/home/ubuntu/go
mkdir -p $GOPATH

# Verify Go version
echo "Go version: $(go version)"

# Create application directory
APP_DIR="/opt/dfs-api"
sudo mkdir -p $APP_DIR

# Remove existing directory if it exists (from previous failed runs)
if [ -d "$APP_DIR/.git" ]; then
    echo "Removing existing repository..."
    sudo rm -rf $APP_DIR/*
fi

cd $APP_DIR

# Clone repository
echo "Cloning repository..."
git clone https://github.com/timskillet/distributed-filestore.git . || {
    echo "ERROR: Failed to clone repository"
    exit 1
}

# Set environment variables
export AWS_REGION=${AWS_REGION:-us-east-1}
export CHUNK_METADATA_TABLE=${CHUNK_METADATA_TABLE:-dfs-chunk-metadata}
export NODE_REGISTRY_TABLE=${NODE_REGISTRY_TABLE:-dfs-node-registry}
export REPLICATION_FACTOR=${REPLICATION_FACTOR:-2}

# Build the API server
echo "Building API server..."
# Explicitly use /usr/local/go/bin/go to ensure we use the right version
/usr/local/go/bin/go mod download || {
    echo "ERROR: Failed to download Go modules"
    exit 1
}

/usr/local/go/bin/go build -o dfs-api ./cmd/dfs-api || {
    echo "ERROR: Failed to build API server"
    exit 1
}

# Make binary executable
chmod +x $APP_DIR/dfs-api

# Verify binary exists
if [ ! -f "$APP_DIR/dfs-api" ]; then
    echo "ERROR: Binary was not created"
    exit 1
fi

echo "Binary created successfully: $APP_DIR/dfs-api"

# Create systemd service
echo "Creating systemd service..."
sudo tee /etc/systemd/system/dfs-api.service > /dev/null <<EOF
[Unit]
Description=DFS API Server
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=$APP_DIR
Environment="PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
Environment="GOPATH=/home/ubuntu/go"
Environment="AWS_REGION=$AWS_REGION"
Environment="CHUNK_METADATA_TABLE=$CHUNK_METADATA_TABLE"
Environment="NODE_REGISTRY_TABLE=$NODE_REGISTRY_TABLE"
Environment="REPLICATION_FACTOR=$REPLICATION_FACTOR"
ExecStart=$APP_DIR/dfs-api
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
echo "Starting service..."
sudo systemctl daemon-reload
sudo systemctl enable dfs-api
sudo systemctl start dfs-api

# Wait a moment and check status
sleep 2
if sudo systemctl is-active --quiet dfs-api; then
    echo "✅ API server bootstrap complete! Service is running."
else
    echo "⚠️  Service started but may not be active. Check status with: sudo systemctl status dfs-api"
    sudo systemctl status dfs-api --no-pager -l || true
fi

echo "Check status with: sudo systemctl status dfs-api"
echo "View logs with: sudo journalctl -u dfs-api -f"
