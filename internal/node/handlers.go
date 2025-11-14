package node

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/timskillet/distributed-filestore/internal/dynamodb"
)

var nodeServer *Server

func SetServer(s *Server) {
	nodeServer = s
}

func HandleStoreChunk() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if nodeServer == nil {
			http.Error(w, "Node server not initialized", http.StatusInternalServerError)
			return
		}

		fileID := r.URL.Query().Get("file_id")
		chunkIndexStr := r.URL.Query().Get("chunk_index")
		if fileID == "" || chunkIndexStr == "" {
			http.Error(w, "missing file_id or chunk_index", http.StatusBadRequest)
			return
		}

		chunkIndex, err := strconv.Atoi(chunkIndexStr)
		if err != nil {
			http.Error(w, "invalid chunk_index", http.StatusBadRequest)
			return
		}

		// Determine replica type (primary if first upload, secondary if replication)
		replicaType := r.URL.Query().Get("replica_type")
		if replicaType == "" {
			replicaType = "primary" // Default to primary
		}

		dir := filepath.Join("./", nodeServer.nodeID, "chunks")
		os.MkdirAll(dir, 0755)
		chunkPath := filepath.Join(dir, fmt.Sprintf("%s_%d.bin", fileID, chunkIndex))

		// Read chunk data and calculate checksum
		chunkData, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read chunk data", http.StatusInternalServerError)
			return
		}

		// Calculate SHA256 checksum
		hash := sha256.Sum256(chunkData)
		checksum := hex.EncodeToString(hash[:])

		// Validate checksum if provided in header
		expectedChecksum := r.Header.Get("X-Chunk-Checksum")
		if expectedChecksum != "" && expectedChecksum != checksum {
			http.Error(w, fmt.Sprintf("checksum mismatch: expected %s, got %s", expectedChecksum, checksum), http.StatusBadRequest)
			return
		}

		// Write chunk to file
		outFile, err := os.Create(chunkPath)
		if err != nil {
			http.Error(w, "failed to create chunk file", http.StatusInternalServerError)
			return
		}
		defer outFile.Close()
		_, err = outFile.Write(chunkData)
		if err != nil {
			http.Error(w, "failed to write chunk data", http.StatusInternalServerError)
			return
		}

		// Store metadata in DynamoDB
		ctx := context.Background()
		metadata := &dynamodb.ChunkMetadata{
			FileID:      fileID,
			ChunkIndex:  chunkIndex,
			NodeID:      nodeServer.nodeID,
			Path:        chunkPath,
			Checksum:    checksum,
			ReplicaType: replicaType,
			CreatedAt:   time.Now().Unix(),
		}

		if err := nodeServer.dbClient.PutChunkMetadata(ctx, nodeServer.cfg.ChunkMetadataTable, metadata); err != nil {
			http.Error(w, fmt.Sprintf("failed to store metadata: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "Chunk %d stored on node %s", chunkIndex, nodeServer.nodeID)
	}
}

func HandleGetChunk() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if nodeServer == nil {
			http.Error(w, "Node server not initialized", http.StatusInternalServerError)
			return
		}

		fileID := r.URL.Query().Get("file_id")
		chunkIndexStr := r.URL.Query().Get("chunk_index")
		if fileID == "" || chunkIndexStr == "" {
			http.Error(w, "missing file_id or chunk_index", http.StatusBadRequest)
			return
		}

		chunkIndex, err := strconv.Atoi(chunkIndexStr)
		if err != nil {
			http.Error(w, "invalid chunk_index", http.StatusBadRequest)
			return
		}

		chunkPath := filepath.Join("./", nodeServer.nodeID, "chunks", fmt.Sprintf("%s_%d.bin", fileID, chunkIndex))
		inFile, err := os.Open(chunkPath)
		if err != nil {
			http.Error(w, "failed to open chunk file", http.StatusNotFound)
			return
		}
		defer inFile.Close()
		_, err = io.Copy(w, inFile)
		if err != nil {
			http.Error(w, "failed to copy chunk data", http.StatusInternalServerError)
			return
		}
	}
}
