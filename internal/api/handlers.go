package api

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

type InitUploadRequest struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

type UploadTarget struct {
	ChunkIndex int    `json:"chunk_index"`
	Node       string `json:"node"`
	URL        string `json:"url"`
}

type InitUploadResponse struct {
	FileID        string         `json:"file_id"`
	ChunkSize     int            `json:"chunk_size"`
	UploadTargets []UploadTarget `json:"upload_targets"`
}

func HandleInitUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}

	var req InitUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Filename == "" || req.Size <= 0 {
		http.Error(w, "Invalid file info", http.StatusBadRequest)
		return
	}

	// TODO: Load available storage nodes from DynamoDB
	nodes := []struct {
		ID      string
		Address string
		Port    string
	}{
		{"nodeA", "http://localhost", "8081"},
		{"nodeB", "http://localhost", "8082"},
	}

	// Parse request body
	chunkSize := 1024 // 1KB
	totalChunks := int((req.Size + int64(chunkSize) - 1) / int64(chunkSize))

	fileID := uuid.New().String()
	uploadTargets := make([]UploadTarget, totalChunks)
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < totalChunks; i++ {
		node := nodes[rand.Intn(len(nodes))]
		url := fmt.Sprintf("%s:%s/store-chunk?file_id=%s&chunk_index=%d", node.Address, node.Port, fileID, i)
		uploadTargets[i] = UploadTarget{i, node.ID, url}
	}

	resp := InitUploadResponse{fileID, chunkSize, uploadTargets}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func HandleFinalizeUpload(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "File upload finalized successfully!")
}

type DownloadTarget struct {
	ChunkIndex int    `json:"chunk_index"`
	URL        string `json:"url"`
}

// Temporary metadata file for retrieving chunk locations until we implement DynamoDB integration
const metadataFile = "metadata.json"

func HandleDownloadPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}

	// Get file ID from query parameters
	fileID := r.URL.Query().Get("file_id")
	if fileID == "" {
		http.Error(w, "missing file_id", http.StatusBadRequest)
		return
	}

	// Read simulated DynamoDB metadata
	f, err := os.Open(metadataFile)
	if err != nil {
		http.Error(w, "metadata file not found", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	var metadata []struct {
		FileID     string `json:"file_id"`
		ChunkIndex int    `json:"chunk_index"`
		NodeID     string `json:"node_id"`
		Path       string `json:"path"`
	}
	if err := json.NewDecoder(f).Decode(&metadata); err != nil {
		http.Error(w, "failed to parse metadata", http.StatusInternalServerError)
		return
	}

	// Generate download targets for requested file
	var targets []DownloadTarget
	for _, entry := range metadata {
		if entry.FileID == fileID {
			port := mapNodeToPort(entry.NodeID)
			url := fmt.Sprintf("http://localhost:%d/get-chunk?file_id=%s&chunk_index=%d", port, fileID, entry.ChunkIndex)
			targets = append(targets, DownloadTarget{entry.ChunkIndex, url})
		}
	}

	if len(targets) == 0 {
		http.Error(w, "no chunks found for file", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targets)
}

// Map node IDs to local ports (for testing)
func mapNodeToPort(nodeID string) int {
	switch nodeID {
	case "nodeA":
		return 8081
	case "nodeB":
		return 8082
	default:
		return 8081
	}
}
