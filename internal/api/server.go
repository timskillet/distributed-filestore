package api

import (
	"context"
	"fmt"

	"github.com/timskillet/distributed-filestore/internal/config"
	"github.com/timskillet/distributed-filestore/internal/dynamodb"
)

type Server struct {
	dbClient *dynamodb.Client
	cfg      *config.Config
}

func NewServer(cfg *config.Config) (*Server, error) {
	ctx := context.Background()
	dbClient, err := dynamodb.NewClient(ctx, cfg.AWSRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to create DynamoDB client: %w", err)
	}

	return &Server{
		dbClient: dbClient,
		cfg:      cfg,
	}, nil
}

func (s *Server) GetDBClient() *dynamodb.Client {
	return s.dbClient
}

func (s *Server) GetConfig() *config.Config {
	return s.cfg
}
