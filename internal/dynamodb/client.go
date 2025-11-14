package dynamodb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Client struct {
	svc    *dynamodb.Client
	region string
}

func NewClient(ctx context.Context, region string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	svc := dynamodb.NewFromConfig(cfg)

	return &Client{
		svc:    svc,
		region: region,
	}, nil
}

func (c *Client) GetDynamoDBClient() *dynamodb.Client {
	return c.svc
}
