#!/bin/bash
yum update -y
yum install -y golang git awscli
mkdir -p /opt/dfs
echo "Storage node initialized" > /opt/dfs/health.txt
