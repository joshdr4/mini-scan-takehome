package dynamodb

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func TestNewDynamoDB(t *testing.T) {
	t.Run("should return error if config is nil", func(t *testing.T) {
		_, err := NewDynamoDB(nil)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("should return error if client is nil", func(t *testing.T) {
		_, err := NewDynamoDB(&DynamoDBConfig{})
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("should return a new DynamoDB", func(t *testing.T) {
		_, err := NewDynamoDB(&DynamoDBConfig{
			Client: &dynamodb.Client{},
		})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
