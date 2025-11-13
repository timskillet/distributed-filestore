package node

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type ChunkMetadata struct {
	FileID     string `json:"file_id"`
	ChunkIndex int    `json:"chunk_index"`
	NodeID     string `json:"node_id"`
	Path       string `json:"path"`
}

const metadataFile = "metadata.json"

var metadataMutex sync.Mutex

func HandleStoreChunk(nodeID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		dir := filepath.Join("./", nodeID, "chunks")
		os.MkdirAll(dir, 0755)
		chunkPath := filepath.Join(dir, fmt.Sprintf("%s_%d.bin", fileID, chunkIndex))
		outFile, err := os.Create(chunkPath)
		if err != nil {
			http.Error(w, "failed to create chunk file", http.StatusInternalServerError)
			return
		}
		defer outFile.Close()
		_, err = io.Copy(outFile, r.Body)
		if err != nil {
			http.Error(w, "failed to copy chunk data", http.StatusInternalServerError)
			return
		}

		// Synchronize access to metadata file
		metadataMutex.Lock()
		defer metadataMutex.Unlock()

		// Read existing metadata
		var metadata []ChunkMetadata
		if _, err := os.Stat(metadataFile); err == nil {
			f, err := os.Open(metadataFile)
			if err == nil {
				json.NewDecoder(f).Decode(&metadata)
				f.Close()
			}
		}

		// Write file metadata to file
		metadata = append(metadata, ChunkMetadata{FileID: fileID, ChunkIndex: chunkIndex, NodeID: nodeID, Path: chunkPath})
		f, err := os.Create(metadataFile)
		if err != nil {
			http.Error(w, "failed to write metadata", http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(f).Encode(metadata)
		f.Close()
		if err != nil {
			http.Error(w, "failed to encode metadata", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "Chunk %d stored on node %s", chunkIndex, nodeID)
	}
}

func HandleGetChunk(nodeID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		chunkPath := filepath.Join("./", nodeID, "chunks", fmt.Sprintf("%s_%d.bin", fileID, chunkIndex))
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
		w.WriteHeader(http.StatusOK)
	}
}
