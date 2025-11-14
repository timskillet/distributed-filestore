package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/timskillet/distributed-filestore/internal/config"
	"github.com/timskillet/distributed-filestore/internal/dynamodb"
	"github.com/timskillet/distributed-filestore/internal/node"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Get node ID and private IP from EC2 metadata or environment
	nodeID := os.Getenv("NODE_ID")
	instanceID, privateIP, err := node.GetEC2InstanceMetadata()
	if err != nil {
		log.Printf("Warning: Failed to get EC2 metadata: %v. Using environment variables.", err)
		if nodeID == "" {
			nodeID = "node-local"
		}
		if privateIP == "" {
			privateIP = "127.0.0.1"
		}
	} else {
		if nodeID == "" {
			nodeID = instanceID // Use instance ID as node ID
		}
	}

	port := node.GetNodePort()

	// Create node info
	nodeInfo := &dynamodb.NodeInfo{
		NodeID:     nodeID,
		PrivateIP:  privateIP,
		Port:       port,
		Status:     "active",
		InstanceID: instanceID,
	}

	// Initialize node server
	srv, err := node.NewServer(cfg, nodeID, nodeInfo)
	if err != nil {
		log.Fatalf("Failed to initialize node server: %v", err)
	}
	node.SetServer(srv)

	// Register node in DynamoDB
	ctx := context.Background()
	if err := srv.Register(ctx); err != nil {
		log.Fatalf("Failed to register node: %v", err)
	}
	fmt.Printf("Node %s registered successfully\n", nodeID)

	// Start heartbeat goroutine
	go srv.StartHeartbeat(ctx)
	fmt.Printf("Heartbeat started (interval: %d seconds)\n", cfg.NodeHeartbeatInterval)

	// Start HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/store-chunk", node.HandleStoreChunk())
	mux.HandleFunc("/get-chunk", node.HandleGetChunk())

	fmt.Printf("Starting DFS storage node on port %d\n", port)
	fmt.Printf("Node ID: %s\n", nodeID)
	fmt.Printf("Private IP: %s\n", privateIP)
	fmt.Printf("AWS Region: %s\n", cfg.AWSRegion)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
