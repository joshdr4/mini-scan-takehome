.PHONY: run-consumer start-scanner start-dynamo test test-integration

# Run consumer with optional arguments
# Usage: make run-consumer ARGS="--project test-project --subscription scan-sub --consumers 10"
# Default args included below
ARGS ?= --project test-project --subscription scan-sub --consumers 10 --max-outstanding 1000

start-dynamo:
	docker-compose -f docker-compose.dynamo.yml up

start-scanner:
	docker-compose up

run-consumer:
	PUBSUB_EMULATOR_HOST=localhost:8085 go run main.go consumer $(ARGS)

test:
	go test ./... -v

test-integration:
	go test -tags=integration -v

