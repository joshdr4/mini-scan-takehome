//go:build integration

package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/censys/scan-takehome/internal/managers/scan_manager"
	dynamodbstore "github.com/censys/scan-takehome/internal/repositories/dynamodb"
	"github.com/censys/scan-takehome/internal/serializer"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupDynamoDB creates an ephemeral DynamoDB container and returns a client
func setupDynamoDB(t *testing.T) (*dynamodb.Client, func()) {
	ctx := context.Background()

	// Start DynamoDB Local container
	req := testcontainers.ContainerRequest{
		Image:        "amazon/dynamodb-local:latest",
		ExposedPorts: []string{"8000/tcp"},
		Cmd:          []string{"-jar", "DynamoDBLocal.jar", "-sharedDb", "-inMemory"},
		WaitingFor:   wait.ForListeningPort("8000/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		t.Fatalf("Failed to start DynamoDB container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "8000")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	endpoint := fmt.Sprintf("http://%s:%s", host, port.Port())

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"dummy", "dummy", "",
		)),
	)

	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	// Create the scan-results table
	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String("scan-results"),
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("pk"),
				KeyType:       types.KeyTypeHash,
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("pk"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	cleanup := func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	return client, cleanup
}

// getItemFromDynamoDB retrieves an item from DynamoDB
func getItemFromDynamoDB(t *testing.T, client *dynamodb.Client, ip string, port uint32, service string) map[string]types.AttributeValue {
	pk := fmt.Sprintf("%s#%d#%s", ip, port, service)

	result, err := client.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String("scan-results"),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: pk},
		},
	})

	if err != nil {
		t.Fatalf("Failed to get item from DynamoDB: %v", err)
	}

	return result.Item
}

func TestIntegration_OutOfOrderMessages(t *testing.T) {
	client, cleanup := setupDynamoDB(t)
	defer cleanup()

	store, err := dynamodbstore.NewDynamoDB(&dynamodbstore.DynamoDBConfig{Client: client})
	if err != nil {
		t.Fatalf("Failed to create DynamoDB store: %v", err)
	}

	manager, err := scan_manager.NewScanManager(&scan_manager.ScanManagerConfig{Repo: store})
	if err != nil {
		t.Fatalf("Failed to create scan manager: %v", err)
	}

	// First message with newer timestamp
	jsonData1 := `{
		"ip": "172.16.0.1",
		"port": 443,
		"service": "https",
		"timestamp": 5000000000,
		"data_version": 2,
		"data": {
			"response_str": "newer response"
		}
	}`

	result1, _ := serializer.ParseScanMessage([]byte(jsonData1))
	err = manager.PutScan(context.Background(), result1)
	if err != nil {
		t.Fatalf("Failed to put first scan: %v", err)
	}

	// Second message with older timestamp (should be rejected)
	jsonData2 := `{
		"ip": "172.16.0.1",
		"port": 443,
		"service": "https",
		"timestamp": 3000000000,
		"data_version": 2,
		"data": {
			"response_str": "older response"
		}
	}`

	result2, _ := serializer.ParseScanMessage([]byte(jsonData2))
	err = manager.PutScan(context.Background(), result2)
	// Should succeed (conditional check handles it silently)
	if err != nil {
		t.Fatalf("Failed to put second scan: %v", err)
	}

	// Verify only the newer response is stored
	item := getItemFromDynamoDB(t, client, "172.16.0.1", 443, "https")
	if resp, ok := item["response"].(*types.AttributeValueMemberS); ok {
		if resp.Value != "newer response" {
			t.Errorf("Expected 'newer response', got '%s' - older message overwrote newer", resp.Value)
		}
	}
}

func TestIntegration_MultipleServices(t *testing.T) {
	client, cleanup := setupDynamoDB(t)
	defer cleanup()

	store, err := dynamodbstore.NewDynamoDB(&dynamodbstore.DynamoDBConfig{Client: client})
	if err != nil {
		t.Fatalf("Failed to create DynamoDB store: %v", err)
	}

	manager, err := scan_manager.NewScanManager(&scan_manager.ScanManagerConfig{Repo: store})
	if err != nil {
		t.Fatalf("Failed to create scan manager: %v", err)
	}

	// Same IP, different ports and services
	services := []struct {
		port    uint32
		service string
		resp    string
	}{
		{80, "http", "HTTP response"},
		{443, "https", "HTTPS response"},
		{22, "ssh", "SSH response"},
	}

	for _, svc := range services {
		jsonData := fmt.Sprintf(`{
			"ip": "192.168.10.1",
			"port": %d,
			"service": "%s",
			"timestamp": 9000000000,
			"data_version": 2,
			"data": {
				"response_str": "%s"
			}
		}`, svc.port, svc.service, svc.resp)

		result, _ := serializer.ParseScanMessage([]byte(jsonData))
		err = manager.PutScan(context.Background(), result)
		if err != nil {
			t.Fatalf("Failed to put scan for %s: %v", svc.service, err)
		}
	}

	// Verify all services are stored separately
	for _, svc := range services {
		item := getItemFromDynamoDB(t, client, "192.168.10.1", svc.port, svc.service)
		if item == nil {
			t.Errorf("Item not found for service %s", svc.service)
			continue
		}

		if resp, ok := item["response"].(*types.AttributeValueMemberS); ok {
			if resp.Value != svc.resp {
				t.Errorf("Expected '%s', got '%s' for service %s", svc.resp, resp.Value, svc.service)
			}
		}
	}
}
