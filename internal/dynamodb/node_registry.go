package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type NodeInfo struct {
	NodeID         string `dynamodbav:"node_id"`
	PrivateIP      string `dynamodbav:"private_ip"`
	Port           int    `dynamodbav:"port"`
	HeartbeatTS    int64  `dynamodbav:"heartbeat_ts"`
	Status         string `dynamodbav:"status"`
	InstanceID     string `dynamodbav:"instance_id,omitempty"`
	AvailableSpace int64  `dynamodbav:"available_space,omitempty"`
}

func (c *Client) RegisterNode(ctx context.Context, tableName string, node *NodeInfo) error {
	node.HeartbeatTS = time.Now().Unix()
	node.Status = "active"

	item, err := attributevalue.MarshalMap(node)
	if err != nil {
		return fmt.Errorf("failed to marshal node info: %w", err)
	}

	_, err = c.svc.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	return nil
}

func (c *Client) UpdateHeartbeat(ctx context.Context, tableName string, nodeID string) error {
	heartbeatTS := time.Now().Unix()

	_, err := c.svc.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"node_id": &types.AttributeValueMemberS{Value: nodeID},
		},
		UpdateExpression: aws.String("SET heartbeat_ts = :ts, #status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":ts":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", heartbeatTS)},
			":status": &types.AttributeValueMemberS{Value: "active"},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	return nil
}

func (c *Client) ListActiveNodes(ctx context.Context, tableName string, heartbeatTimeout int64) ([]*NodeInfo, error) {
	// Scan table and filter active nodes
	// Note: For production, consider using GSI or query patterns
	result, err := c.svc.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan nodes: %w", err)
	}

	var nodes []*NodeInfo
	now := time.Now().Unix()

	for _, item := range result.Items {
		var node NodeInfo
		if err := attributevalue.UnmarshalMap(item, &node); err != nil {
			continue
		}

		// Filter by heartbeat timeout
		if now-node.HeartbeatTS <= heartbeatTimeout && node.Status == "active" {
			nodes = append(nodes, &node)
		}
	}

	return nodes, nil
}

func (c *Client) GetNode(ctx context.Context, tableName string, nodeID string) (*NodeInfo, error) {
	result, err := c.svc.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"node_id": &types.AttributeValueMemberS{Value: nodeID},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	var node NodeInfo
	if err := attributevalue.UnmarshalMap(result.Item, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node: %w", err)
	}

	return &node, nil
}
