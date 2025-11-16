package consumer

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"cloud.google.com/go/pubsub"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/censys/scan-takehome/internal/managers/scan_manager"
	dynamodbstore "github.com/censys/scan-takehome/internal/repositories/dynamodb"
	"github.com/censys/scan-takehome/internal/serializer"
	"github.com/spf13/cobra"
)

var (
	projectID      string
	subscriptionID string
	numConsumers   int
	maxOutstanding int
)

func NewConsumerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consumer",
		Short: "Run the consumer to process scan results",
		Long:  "Consumer for processing scan results from the scanner",
		Run:   runConsumer,
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "test-project", "GCP Project ID")
	cmd.Flags().StringVarP(&subscriptionID, "subscription", "s", "scan-sub", "GCP PubSub Subscription ID")
	cmd.Flags().IntVarP(&numConsumers, "consumers", "c", 10, "Number of concurrent consumers")
	cmd.Flags().IntVarP(&maxOutstanding, "max-outstanding", "m", 1000, "Max outstanding messages")

	return cmd
}

func runConsumer(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fmt.Printf("Starting consumer for project: %s, subscription: %s\n", projectID, subscriptionID)
	fmt.Printf("Concurrent consumers: %d, Max outstanding messages: %d\n", numConsumers, maxOutstanding)

	// Create DynamoDB client for local development
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"dummy", "dummy", "",
		)),
	)

	if err != nil {
		fmt.Printf("Error loading AWS config: %v\n", err)
		return
	}

	dynamoClient := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String("http://localhost:8000")
	})

	// Initialize the dynamoDB repository for storing scan results
	store, err := dynamodbstore.NewDynamoDB(&dynamodbstore.DynamoDBConfig{
		Client: dynamoClient,
	})

	if err != nil {
		fmt.Printf("Error initializing scanner store: %v\n", err)
		return
	}

	// Initialize scan manager with store as repository
	manager, err := scan_manager.NewScanManager(&scan_manager.ScanManagerConfig{
		Repo: store,
	})

	if err != nil {
		fmt.Printf("Error initializing scan manager: %v\n", err)
		return
	}

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		fmt.Printf("Error creating PubSub client: %v\n", err)
		return
	}

	defer client.Close()

	sub := client.Subscription(subscriptionID)

	sub.ReceiveSettings.NumGoroutines = numConsumers
	sub.ReceiveSettings.MaxOutstandingMessages = maxOutstanding

	var processed, failed atomic.Int64

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal, stopping consumer...")
		cancel()
	}()

	fmt.Println("Consumer started, waiting for messages...")
	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		result, err := serializer.ParseScanMessage(msg.Data)
		if err != nil {
			fmt.Printf("Error parsing scan message: %v\n", err)
			failed.Add(1)
			msg.Nack()
			return
		}

		if err := manager.PutScan(ctx, result); err != nil {
			fmt.Printf("Error storing scan: %v\n", err)
			failed.Add(1)
			msg.Nack()
			return
		}

		processed.Add(1)
		msg.Ack()
	})

	if err != nil && err != context.Canceled {
		fmt.Printf("Error receiving messages: %v\n", err)
		return
	}

	fmt.Printf("\nConsumer stopped. Final stats - Processed: %d, Failed: %d\n", processed.Load(), failed.Load())
}
