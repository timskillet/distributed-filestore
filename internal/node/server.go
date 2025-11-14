package node

import (
	"context"
	"fmt"
	"time"

	"github.com/timskillet/distributed-filestore/internal/config"
	"github.com/timskillet/distributed-filestore/internal/dynamodb"
)

type Server struct {
	dbClient *dynamodb.Client
	cfg      *config.Config
	nodeID   string
	nodeInfo *dynamodb.NodeInfo
	stopChan chan struct{}
}

func NewServer(cfg *config.Config, nodeID string, nodeInfo *dynamodb.NodeInfo) (*Server, error) {
	ctx := context.Background()
	dbClient, err := dynamodb.NewClient(ctx, cfg.AWSRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to create DynamoDB client: %w", err)
	}

	return &Server{
		dbClient: dbClient,
		cfg:      cfg,
		nodeID:   nodeID,
		nodeInfo: nodeInfo,
		stopChan: make(chan struct{}),
	}, nil
}

func (s *Server) Register(ctx context.Context) error {
	return s.dbClient.RegisterNode(ctx, s.cfg.NodeRegistryTable, s.nodeInfo)
}

func (s *Server) StartHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(s.cfg.NodeHeartbeatInterval) * time.Second)
	defer ticker.Stop()

	// Send initial heartbeat
	if err := s.dbClient.UpdateHeartbeat(ctx, s.cfg.NodeRegistryTable, s.nodeID); err != nil {
		fmt.Printf("Warning: Failed to send initial heartbeat: %v\n", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := s.dbClient.UpdateHeartbeat(ctx, s.cfg.NodeRegistryTable, s.nodeID); err != nil {
				fmt.Printf("Warning: Failed to update heartbeat: %v\n", err)
			}
		case <-s.stopChan:
			return
		}
	}
}

func (s *Server) Stop() {
	close(s.stopChan)
}

func (s *Server) GetDBClient() *dynamodb.Client {
	return s.dbClient
}

func (s *Server) GetConfig() *config.Config {
	return s.cfg
}

func (s *Server) GetNodeID() string {
	return s.nodeID
}
