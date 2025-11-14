package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/timskillet/distributed-filestore/internal/api"
	"github.com/timskillet/distributed-filestore/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize API server with DynamoDB
	srv, err := api.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize API server: %v", err)
	}
	api.SetServer(srv)

	mux := http.NewServeMux()
	mux.HandleFunc("/init-upload", api.HandleInitUpload)
	mux.HandleFunc("/finalize-upload", api.HandleFinalizeUpload)
	mux.HandleFunc("/download-plan", api.HandleDownloadPlan)

	fmt.Printf("Starting DFS API server on port 8080\n")
	fmt.Printf("AWS Region: %s\n", cfg.AWSRegion)
	fmt.Printf("Chunk Metadata Table: %s\n", cfg.ChunkMetadataTable)
	fmt.Printf("Node Registry Table: %s\n", cfg.NodeRegistryTable)
	fmt.Printf("Replication Factor: %d\n", cfg.ReplicationFactor)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
