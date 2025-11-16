package dynamodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/censys/scan-takehome/internal/managers/scan_manager"
)

type DynamoDBConfig struct {
	Client *dynamodb.Client
}

type dynamoDB struct {
	client *dynamodb.Client
}

func NewDynamoDB(cfg *DynamoDBConfig) (*dynamoDB, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	if cfg.Client == nil {
		return nil, errors.New("DynamoDB client is nil")
	}

	db := &dynamoDB{
		client: cfg.Client,
	}

	return db, nil
}

func (d *dynamoDB) Put(ctx context.Context, result *scan_manager.ScanResult) error {
	pk := fmt.Sprintf("%s#%d#%s", result.IP, result.Port, result.Service)

	item := map[string]types.AttributeValue{
		"pk":           &types.AttributeValueMemberS{Value: pk},
		"ip":           &types.AttributeValueMemberS{Value: result.IP},
		"port":         &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", result.Port)},
		"service":      &types.AttributeValueMemberS{Value: result.Service},
		"timestamp":    &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", result.Timestamp)},
		"response":     &types.AttributeValueMemberS{Value: result.Response},
		"data_version": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", result.DataVersion)},
	}

	// Conditional write: only accept if item doesn't exist OR new timestamp > existing timestamp
	// This handles out-of-order messages and ensures we keep the latest scan
	input := &dynamodb.PutItemInput{
		TableName: aws.String("scan-results"),
		Item:      item,
		ConditionExpression: aws.String(
			"attribute_not_exists(pk) OR #ts < :new_ts",
		),
		ExpressionAttributeNames: map[string]string{
			"#ts": "timestamp",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":new_ts": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", result.Timestamp)},
		},
	}

	_, err := d.client.PutItem(ctx, input)
	if err != nil {
		// Check if it's a conditional check failure (item exists with newer timestamp)
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			// This is expected for out-of-order messages with older timestamps
			return nil
		}
		return fmt.Errorf("failed to put item to DynamoDB: %w", err)
	}

	return nil
}
