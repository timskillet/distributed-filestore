package dynamodb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type ChunkMetadata struct {
	FileID          string `dynamodbav:"file_id"`
	ChunkReplicaKey string `dynamodbav:"chunk_replica_key"` // Format: "chunk_index#node_id"
	ChunkIndex      int    `dynamodbav:"chunk_index"`
	NodeID          string `dynamodbav:"node_id"`
	Path            string `dynamodbav:"path"`
	Checksum        string `dynamodbav:"checksum"`
	ReplicaType     string `dynamodbav:"replica_type"` // "primary" or "secondary"
	CreatedAt       int64  `dynamodbav:"created_at"`
}

// PutChunkMetadata stores a chunk replica in DynamoDB
func (c *Client) PutChunkMetadata(ctx context.Context, tableName string, metadata *ChunkMetadata) error {
	// Generate chunk_replica_key
	metadata.ChunkReplicaKey = fmt.Sprintf("%d#%s", metadata.ChunkIndex, metadata.NodeID)

	item, err := attributevalue.MarshalMap(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk metadata: %w", err)
	}

	_, err = c.svc.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to put chunk metadata: %w", err)
	}

	return nil
}

// GetChunkReplicas returns all replicas for a specific chunk
func (c *Client) GetChunkReplicas(ctx context.Context, tableName string, fileID string, chunkIndex int) ([]*ChunkMetadata, error) {
	// Query by file_id (hash key) - all chunks for this file
	// Filter by chunk_index in application code since range key is composite
	result, err := c.svc.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		KeyConditionExpression: aws.String("file_id = :file_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":file_id": &types.AttributeValueMemberS{Value: fileID},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query chunk replicas: %w", err)
	}

	var replicas []*ChunkMetadata
	for _, item := range result.Items {
		var metadata ChunkMetadata
		if err := attributevalue.UnmarshalMap(item, &metadata); err != nil {
			continue
		}
		// Filter by chunk_index
		if metadata.ChunkIndex == chunkIndex {
			replicas = append(replicas, &metadata)
		}
	}

	return replicas, nil
}

// GetChunksByFileID returns all chunks (with all replicas) for a file
func (c *Client) GetChunksByFileID(ctx context.Context, tableName string, fileID string) ([]*ChunkMetadata, error) {
	result, err := c.svc.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		KeyConditionExpression: aws.String("file_id = :file_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":file_id": &types.AttributeValueMemberS{Value: fileID},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}

	var chunks []*ChunkMetadata
	for _, item := range result.Items {
		var metadata ChunkMetadata
		if err := attributevalue.UnmarshalMap(item, &metadata); err != nil {
			continue
		}
		chunks = append(chunks, &metadata)
	}

	return chunks, nil
}

// GetChunksByNodeID returns all chunks stored on a specific node (using GSI)
func (c *Client) GetChunksByNodeID(ctx context.Context, tableName string, nodeID string) ([]*ChunkMetadata, error) {
	result, err := c.svc.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		IndexName:              aws.String("node-id-index"),
		KeyConditionExpression: aws.String("node_id = :node_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":node_id": &types.AttributeValueMemberS{Value: nodeID},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query chunks by node: %w", err)
	}

	var chunks []*ChunkMetadata
	for _, item := range result.Items {
		var metadata ChunkMetadata
		if err := attributevalue.UnmarshalMap(item, &metadata); err != nil {
			continue
		}
		chunks = append(chunks, &metadata)
	}

	return chunks, nil
}

// DeleteChunkMetadata deletes a specific chunk replica
func (c *Client) DeleteChunkMetadata(ctx context.Context, tableName string, fileID string, chunkIndex int, nodeID string) error {
	chunkReplicaKey := fmt.Sprintf("%d#%s", chunkIndex, nodeID)

	_, err := c.svc.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"file_id":           &types.AttributeValueMemberS{Value: fileID},
			"chunk_replica_key": &types.AttributeValueMemberS{Value: chunkReplicaKey},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete chunk metadata: %w", err)
	}

	return nil
}
