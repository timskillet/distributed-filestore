package types

import (
	"time"
)

type FileMetadata struct {
	FileID     string    `json:"file_id"`
	FileName   string    `json:"file_name"`
	OwnerID    string    `json:"owner_id"`
	Size       int64     `json:"size"`
	ChunkIDs   []string  `json:"chunk_ids"`
	SharedWith []string  `json:"shared_with"`
	CreatedAt  time.Time `json:"created_at"`
}

type ChunkMetadata struct {
	ChunkID  string `json:"chunk_id"`
	NodeID   string `json:"node_id"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
}
