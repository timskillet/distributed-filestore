#!/bin/bash
sudo apt update -y
sudo apt install -y golang git
cd /home/ubuntu
git clone https://github.com/yourusername/dfs-project.git
cd dfs-project
go build -o dfs-api cmd/dfs-server/main.go
nohup ./dfs-api > /var/log/dfs-api.log 2>&1 &
