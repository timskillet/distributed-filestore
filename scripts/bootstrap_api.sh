#!/bin/bash

set -e

echo "Starting API server bootstrap..."

export HOME=${HOME:-/root}

# Update system
apt update -y
apt install -y git wget build-essential

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
export GOPATH=/home/ubuntu/go
export GOCACHE=/tmp/go-build-cache
export GOMODCACHE=$GOPATH/pkg/mod

mkdir -p $GOPATH $GOCACHE
chown -R ubuntu:ubuntu /home/ubuntu/go || true

echo "Go version: $(go version)"

# Create application directory
APP_DIR="/opt/dfs-api"
mkdir -p $APP_DIR
chown -R ubuntu:ubuntu $APP_DIR

# Remove existing directory if it exists (from previous failed runs)
if [ -d "$APP_DIR/.git" ]; then
    echo "Removing existing repository..."
    rm -rf $APP_DIR/*
fi

cd $APP_DIR

# Clone repo
echo "Cloning repository..."
git clone https://github.com/timskillet/distributed-filestore.git . || {
    echo "ERROR: Failed to clone repository"
    exit 1
}

export AWS_REGION=$${AWS_REGION:-us-east-1}
export CHUNK_METADATA_TABLE=$${CHUNK_METADATA_TABLE:-dfs-chunk-metadata}
export NODE_REGISTRY_TABLE=$${NODE_REGISTRY_TABLE:-dfs-node-registry}
export REPLICATION_FACTOR=$${REPLICATION_FACTOR:-2}

echo "Building API server..."
/usr/local/go/bin/go mod download || {
    echo "ERROR: Failed to download Go modules"
    exit 1
}

/usr/local/go/bin/go build -o dfs-api ./cmd/dfs-api || {
    echo "ERROR: Failed to build API server"
    exit 1
}

chmod +x dfs-api
chown ubuntu:ubuntu dfs-api

echo "Binary created: $APP_DIR/dfs-api"

# Create systemd service
echo "Creating systemd service..."
tee /etc/systemd/system/dfs-api.service > /dev/null <<EOF
[Unit]
Description=DFS API Server
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=$APP_DIR
Environment="PATH=/usr/local/go/bin:/usr/bin:/bin"
Environment="HOME=/home/ubuntu"
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

systemctl daemon-reload
systemctl enable dfs-api
systemctl start dfs-api

sleep 2
if systemctl is-active --quiet dfs-api; then
    echo "✅ API server bootstrap complete! Service is running."
else
    echo "⚠️ Service may not be active. Check logs:"
    systemctl status dfs-api --no-pager -l || true
fi
