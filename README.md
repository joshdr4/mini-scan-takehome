# Josh Rosen Mini Scan Takehome

A scalable consumer application that processes network scan results from Google Cloud Pub/Sub and stores them in DynamoDB with conditional timestamp-based writes.

## Architecture Overview

```
Scanner (Censys) → Pub/Sub → Consumer (Go) → DynamoDB
                      ↓
                  Multiple concurrent
                  goroutine workers
```

### Components

1. **Serializer** (`internal/serializer`)
   - Parses and normalizes scan messages from Pub/Sub
   - Supports V1 (base64 encoded) and V2 (plain string) data formats

2. **Scan Manager** (`internal/managers/scan_manager`)
   - Business logic layer for processing scan results
   - Delegates storage to Repository interface for clean separation

3. **DynamoDB Repository** (`internal/repositories/dynamodb`)
   - Implements `Repository` interface for DynamoDB storage
   - Composite primary key: `ip#port#service`
   - Conditional writes: only accepts scans with timestamps > existing

4. **Consumer** (`cmd/consumer`)
   - Receives messages from Pub/Sub subscription
   - Orchestrates serializer → manager → repository pipeline
   - Configurable concurrency and message backlog

## Scaling Architecture

### Consumer Scaling

The consumer scales horizontally and vertically through multiple mechanisms:

#### 1. **Concurrent Message Processing**
```go
sub.ReceiveSettings.NumGoroutines = 10        // Concurrent workers
sub.ReceiveSettings.MaxOutstandingMessages = 1000  // Message buffer
```

- **NumGoroutines**: Number of concurrent goroutines processing messages
  - Default: 10 workers
  - Each worker processes one message at a time
  - Increase for higher throughput (limited by CPU cores)

- **MaxOutstandingMessages**: Buffered messages awaiting processing
  - Default: 1000 messages
  - Acts as a queue when workers are busy
  - Prevents message loss during traffic spikes

#### 2. **Horizontal Scaling (Multiple Consumer Instances)**

Pub/Sub automatically load-balances across multiple consumer instances:


#### 3. **Database Choice**

DynamoDB was chosen for its ability to scale horizontally through partition-based consistent hashing, where each partition operates independently to handle massive write throughput without coordination overhead. The composite primary key (`ip#port#service`) ensures even data distribution across partitions, preventing hot spots while DynamoDB's native conditional write expressions satisfy the requirement for timestamp-based updates, rejecting stale data atomically without application-level locking.

Notes:

DynamoDB does have two main drawbacks.
First, DynamoDB is AWS specific which means this project uses both GPC and AWS.
Second, DynamoDB can be fairly expensive and would likely need to be used with provisioned capacity mode for cost savings.

Good scalable alternatives would be:

Cassandra, ScyllaDB, and Bigtable.

## Quick Start

**Start DynamoDB**
```bash
make start-dynamo
# Runs: docker-compose -f docker-compose.dynamo.yml up
```

**Start Scanner**
```bash
make start-scanner
# Runs: docker-compose up
```

**Run Consumer**
```bash
make run-consumer
# Runs: PUBSUB_EMULATOR_HOST=localhost:8085 go run main.go consumer \
#   --project test-project --subscription scan-sub --consumers 10 --max-outstanding 1000
# Override args: make run-consumer ARGS="--project myproject --consumers 20"
```

Example output from the consumer:

```bash
Concurrent consumers: 10, Max outstanding messages: 1000
Consumer started, waiting for messages...
scan result stored: &{IP:1.1.1.116 Port:31982 Service:SSH Timestamp:1763253174 Response:service response: 31 DataVersion:1}
scan result stored: &{IP:1.1.1.34 Port:21346 Service:HTTP Timestamp:1763253175 Response:service response: 73 DataVersion:2}
scan result stored: &{IP:1.1.1.80 Port:37431 Service:SSH Timestamp:1763253176 Response:service response: 42 DataVersion:2}
scan result stored: &{IP:1.1.1.99 Port:62469 Service:SSH Timestamp:1763253177 Response:service response: 44 DataVersion:1}
scan result stored: &{IP:1.1.1.134 Port:37925 Service:HTTP Timestamp:1763253178 Response:service response: 59 DataVersion:1}
scan result stored: &{IP:1.1.1.53 Port:12585 Service:SSH Timestamp:1763253179 Response:service response: 58 DataVersion:2}
```

This project includes basic unit testing and integration testing.

The integration tests test handling out of order messaging and testing each scan result is stored separately.

**Testing**
```bash
make test                # Runs: go test ./... -v
make test-integration    # Runs: go test -tags=integration -v
```

