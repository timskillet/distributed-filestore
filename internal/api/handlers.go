package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/timskillet/distributed-filestore/internal/dynamodb"
)

type InitUploadRequest struct {
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	ChunkSize int    `json:"chunk_size"`
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

var apiServer *Server

func SetServer(s *Server) {
	apiServer = s
}

func HandleInitUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}

	if apiServer == nil {
		http.Error(w, "API server not initialized", http.StatusInternalServerError)
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

	// Load available storage nodes from DynamoDB
	ctx := context.Background()
	nodes, err := apiServer.dbClient.ListActiveNodes(ctx, apiServer.cfg.NodeRegistryTable, int64(apiServer.cfg.NodeHeartbeatTimeout))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get nodes: %v", err), http.StatusInternalServerError)
		return
	}

	if len(nodes) < apiServer.cfg.ReplicationFactor {
		http.Error(w, fmt.Sprintf("Not enough active nodes. Required: %d, Available: %d", apiServer.cfg.ReplicationFactor, len(nodes)), http.StatusServiceUnavailable)
		return
	}

	// Use provided chunk size or default to 1KB
	chunkSize := req.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 1024 // Default to 1KB
	}
	totalChunks := int((req.Size + int64(chunkSize) - 1) / int64(chunkSize))

	fileID := uuid.New().String()
	uploadTargets := make([]UploadTarget, totalChunks)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Select nodes for each chunk with replication
	for i := 0; i < totalChunks; i++ {
		// Select N nodes (replication factor) for this chunk
		selectedNodes := selectNodesForChunk(nodes, apiServer.cfg.ReplicationFactor, rng)

		// First node is primary, rest are secondary
		primaryNode := selectedNodes[0]
		url := fmt.Sprintf("http://%s:%d/store-chunk?file_id=%s&chunk_index=%d", primaryNode.PrivateIP, primaryNode.Port, fileID, i)
		uploadTargets[i] = UploadTarget{i, primaryNode.NodeID, url}
	}

	resp := InitUploadResponse{fileID, chunkSize, uploadTargets}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// selectNodesForChunk selects N random nodes for replication
func selectNodesForChunk(nodes []*dynamodb.NodeInfo, count int, rng *rand.Rand) []*dynamodb.NodeInfo {
	if len(nodes) <= count {
		return nodes
	}

	// Shuffle and take first N
	shuffled := make([]*dynamodb.NodeInfo, len(nodes))
	copy(shuffled, nodes)
	rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:count]
}

func HandleFinalizeUpload(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "File upload finalized successfully!")
}

type DownloadTarget struct {
	ChunkIndex int    `json:"chunk_index"`
	URL        string `json:"url"`
	Checksum   string `json:"checksum"`
}

func HandleDownloadPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}

	if apiServer == nil {
		http.Error(w, "API server not initialized", http.StatusInternalServerError)
		return
	}

	// Get file ID from query parameters
	fileID := r.URL.Query().Get("file_id")
	if fileID == "" {
		http.Error(w, "missing file_id", http.StatusBadRequest)
		return
	}

	// Query DynamoDB for all chunks of this file
	ctx := context.Background()
	chunks, err := apiServer.dbClient.GetChunksByFileID(ctx, apiServer.cfg.ChunkMetadataTable, fileID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get chunks: %v", err), http.StatusInternalServerError)
		return
	}

	if len(chunks) == 0 {
		http.Error(w, "no chunks found for file", http.StatusNotFound)
		return
	}

	// Get active nodes to map node_id to IP/port
	nodes, err := apiServer.dbClient.ListActiveNodes(ctx, apiServer.cfg.NodeRegistryTable, int64(apiServer.cfg.NodeHeartbeatTimeout))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get nodes: %v", err), http.StatusInternalServerError)
		return
	}

	nodeMap := make(map[string]*dynamodb.NodeInfo)
	for _, node := range nodes {
		nodeMap[node.NodeID] = node
	}

	// Group chunks by chunk_index and select best replica for each
	chunkMap := make(map[int][]*dynamodb.ChunkMetadata)
	for _, chunk := range chunks {
		chunkMap[chunk.ChunkIndex] = append(chunkMap[chunk.ChunkIndex], chunk)
	}

	var targets []DownloadTarget
	for chunkIndex, replicas := range chunkMap {
		// Select best replica (prefer primary, then healthy nodes)
		selectedReplica := selectBestReplica(replicas, nodeMap)
		if selectedReplica == nil {
			continue
		}

		node, ok := nodeMap[selectedReplica.NodeID]
		if !ok {
			continue
		}

		url := fmt.Sprintf("http://%s:%d/get-chunk?file_id=%s&chunk_index=%d", node.PrivateIP, node.Port, fileID, chunkIndex)
		targets = append(targets, DownloadTarget{chunkIndex, url, selectedReplica.Checksum})
	}

	// Sort by chunk index
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].ChunkIndex < targets[j].ChunkIndex
	})

	if len(targets) == 0 {
		http.Error(w, "no valid chunks found for file", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targets)
}

// selectBestReplica selects the best replica from available replicas
func selectBestReplica(replicas []*dynamodb.ChunkMetadata, nodeMap map[string]*dynamodb.NodeInfo) *dynamodb.ChunkMetadata {
	// Prefer primary replicas
	for _, replica := range replicas {
		if replica.ReplicaType == "primary" {
			if node, ok := nodeMap[replica.NodeID]; ok && node.Status == "active" {
				return replica
			}
		}
	}

	// Fallback to any active secondary
	for _, replica := range replicas {
		if node, ok := nodeMap[replica.NodeID]; ok && node.Status == "active" {
			return replica
		}
	}

	// Return first replica if no active nodes found (will fail but at least we try)
	if len(replicas) > 0 {
		return replicas[0]
	}

	return nil
}
