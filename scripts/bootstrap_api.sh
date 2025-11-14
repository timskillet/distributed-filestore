#!/bin/bash
set -e

echo "Starting API server bootstrap..."

# Update system
sudo apt update -y
sudo apt install -y golang-go git

# Install Go 1.23+ if not available
if ! command -v go &> /dev/null || [ "$(go version | awk '{print $3}' | cut -d. -f2)" -lt 23 ]; then
    echo "Installing Go 1.23..."
    wget -q https://go.dev/dl/go1.23.2.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
fi

# Create application directory
APP_DIR="/opt/dfs-api"
sudo mkdir -p $APP_DIR
cd $APP_DIR

# Clone repository (update with your actual repo URL)
# For now, we'll assume the code is already there or will be copied
git clone https://github.com/timskillet/distributed-filestore.git .

# Set environment variables
export AWS_REGION=${AWS_REGION:-us-east-1}
export CHUNK_METADATA_TABLE=${CHUNK_METADATA_TABLE:-dfs-chunk-metadata}
export NODE_REGISTRY_TABLE=${NODE_REGISTRY_TABLE:-dfs-node-registry}
export REPLICATION_FACTOR=${REPLICATION_FACTOR:-2}

# Build the API server
echo "Building API server..."
go mod download
go build -o dfs-api ./cmd/dfs-api

# Create systemd service
sudo tee /etc/systemd/system/dfs-api.service > /dev/null <<EOF
[Unit]
Description=DFS API Server
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=$APP_DIR
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
sudo systemctl daemon-reload
sudo systemctl enable dfs-api
sudo systemctl start dfs-api

echo "API server bootstrap complete!"
echo "Check status with: sudo systemctl status dfs-api"
echo "View logs with: sudo journalctl -u dfs-api -f"
