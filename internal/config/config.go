package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AWSRegion              string
	ChunkMetadataTable     string
	NodeRegistryTable      string
	ReplicationFactor      int
	ReplicationStrategy    string // "sync" or "async"
	ReplicationTimeout     int    // seconds
	NodeHeartbeatInterval  int    // seconds
	NodeHeartbeatTimeout   int    // seconds (nodes considered dead after this)
}

func Load() (*Config, error) {
	cfg := &Config{
		AWSRegion:              getEnv("AWS_REGION", "us-east-1"),
		ChunkMetadataTable:     getEnv("CHUNK_METADATA_TABLE", "dfs-chunk-metadata"),
		NodeRegistryTable:      getEnv("NODE_REGISTRY_TABLE", "dfs-node-registry"),
		ReplicationFactor:      getEnvInt("REPLICATION_FACTOR", 2),
		ReplicationStrategy:    getEnv("REPLICATION_STRATEGY", "sync"),
		ReplicationTimeout:     getEnvInt("REPLICATION_TIMEOUT", 30),
		NodeHeartbeatInterval:  getEnvInt("NODE_HEARTBEAT_INTERVAL", 30),
		NodeHeartbeatTimeout:   getEnvInt("NODE_HEARTBEAT_TIMEOUT", 60),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.AWSRegion == "" {
		return fmt.Errorf("AWS_REGION is required")
	}
	if c.ChunkMetadataTable == "" {
		return fmt.Errorf("CHUNK_METADATA_TABLE is required")
	}
	if c.NodeRegistryTable == "" {
		return fmt.Errorf("NODE_REGISTRY_TABLE is required")
	}
	if c.ReplicationFactor < 1 {
		return fmt.Errorf("REPLICATION_FACTOR must be at least 1")
	}
	if c.ReplicationStrategy != "sync" && c.ReplicationStrategy != "async" {
		return fmt.Errorf("REPLICATION_STRATEGY must be 'sync' or 'async'")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

