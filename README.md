# Crypto Price Alert Engine

A high-performance, real-time cryptocurrency price alert system built with Go and gRPC. Features live Binance WebSocket integration, flexible alert rules, and clean terminal interface. This system demonstrates streaming architecture, pub/sub messaging, and production-ready modular design.

## Features

- **Live Binance Integration**: Real-time crypto prices from Binance WebSocket API (~$109K BTC, ~$3.8K ETH)
- **Real-time Price Streaming**: Subscribe to live cryptocurrency price updates via gRPC streams
- **Smart Alert System**: Create alerts with various comparators (>, >=, <, <=, ==) that trigger on real price movements
- **High-Performance Architecture**: Channel-based pub/sub with backpressure handling
- **Mock Data Feed**: Simulated crypto prices for testing (when Binance is unavailable)
- **Interactive CLI**: Easy-to-use command-line interface for managing alerts and watching prices
- **Thread-Safe Operations**: Concurrent-safe alert storage and processing
- **Cooldown Management**: Prevents alert spam with configurable cooldown periods
- **Docker Support**: Containerized deployment with health checks

## Architecture

```
┌─────────────────┐    gRPC     ┌─────────────────┐
│   CLI Client    │◄────────────┤  gRPC Server    │
└─────────────────┘             └─────────────────┘
                                          │
                                          ▼
┌─────────────────┐             ┌─────────────────┐
│  Pub/Sub Broker │◄────────────┤  Alert Engine   │
└─────────────────┘             └─────────────────┘
          ▲                               ▲
          │                               │
          ▼                               ▼
┌─────────────────┐             ┌─────────────────┐
│ Binance WebSocket│             │  Alert Store    │
│  (Live Prices)  │             │                 │
└─────────────────┘             └─────────────────┘
```

## Prerequisites

- Go 1.22 or higher
- Protocol Buffers compiler (`protoc`)
- Docker (optional, for containerized deployment)

## 🛠️ Quick Start

### 1. Clone and Setup

```bash
git clone https://github.com/tegveerbhatti/crypto-price-alerts.git
cd crypto-price-alerts
make setup  # Downloads deps, installs tools, generates protobuf code
```

### 2. Run the Server

```bash
make run-server
```

The server will start on `localhost:9090` and connect to Binance WebSocket for live prices:
- BTC (~$109,000), ETH (~$3,800), ADA (~$0.60), SOL (~$185), DOT (~$2.85), MATIC, AVAX (~$18), LINK (~$17)

You'll see clean logs like:
```
LIVE: BTC = $109025.14
LIVE: ETH = $3812.29
Alert triggered: BTC > 109000.00 (triggered at 109025.14)
```

### 3. Run the CLI Client

In a new terminal:

```bash
make run-cli
```

## Usage Examples

### Watch Real-time Prices

```bash
# In the CLI, enter:
watch BTC,ETH,ADA
```

### Create Price Alerts

```bash
# In the CLI, enter:
create-alert
# Follow the prompts to set up alerts like:
# - BTC > $109000.00 (will trigger immediately with current prices!)
# - ETH <= $3800.00
# - SOL >= $200.00
```

Example alert creation:
```
Creating a new alert
Enter symbol (e.g., BTC): BTC
Select comparator:
1. > (greater than)
2. >= (greater than or equal)
3. < (less than)
4. <= (less than or equal)
5. == (equal)
Enter choice (1-5): 1
Enter threshold price: $109000
Enter note (optional): BTC above 109k
Alert created successfully!
ID: c5709cbd-7582-4158-8044-75ceecd3401c
Rule: BTC > $109000.00
Note: BTC above 109k
```

### Monitor Alert Triggers

```bash
# In the CLI, enter:
watch-alerts
```

### List All Alerts

```bash
# In the CLI, enter:
list-alerts
```

Example output:
```
Listing all alerts
Found 2 alert(s):

1. Enabled
   ID: c5709cbd-7582-4158-8044-75ceecd3401c
   Rule: BTC > $109000.00
   Note: BTC above 109k
   Last triggered: 2025-10-31 18:34:27

2. Enabled
   ID: 67c5cec3-0350-48fd-9f07-89229da75bd1
   Rule: BTC > $120000.00
   Note: Test alert - won't trigger yet
```

## Development

### Available Make Commands

```bash
make help           # Show all available commands
make build          # Build server and CLI binaries
make test           # Run all tests
make test-race      # Run tests with race detector
make lint           # Run linter (requires golangci-lint)
make proto          # Regenerate protobuf code
make clean          # Clean build artifacts
```

### Project Structure

```
crypto-price-alerts/
├── api/
│   ├── gen/                    # Generated protobuf code
│   └── cryptoalert.proto        # gRPC service definitions
├── cmd/
│   ├── cli/main.go            # CLI client application
│   └── server/main.go         # gRPC server application
├── internal/
│   ├── alerts/
│   │   ├── engine.go          # Rule evaluation engine
│   │   ├── store.go           # Thread-safe alert storage
│   │   └── trigger_bus.go     # Alert trigger pub/sub
│   ├── datafeed/
│   │   ├── binance.go         # Live Binance WebSocket integration
│   │   └── mock.go            # Mock price data generator
│   ├── grpc/
│   │   ├── alertservice.go    # Alert gRPC service
│   │   ├── marketdata.go      # Market data gRPC service
│   │   └── utils.go           # Common utilities
│   └── pubsub/
│       └── broker.go          # Price data pub/sub broker
├── pkg/
│   └── models/
│       ├── alert.go           # Alert data model
│       └── tick.go            # Price tick data model
├── deploy/
│   └── docker-compose.yml     # Docker Compose configuration
├── Dockerfile                 # Container build instructions
├── Makefile                   # Development automation
└── README.md                  # This file
```

## Docker Deployment

### Build and Run with Docker

```bash
make docker-build
make docker-run
```

### Using Docker Compose

```bash
cd deploy
docker-compose up -d
```

## Testing

### Run All Tests

```bash
make test
```

### Run with Race Detection

```bash
make test-race
```

### Manual Testing with CLI

1. Start the server: `make run-server`
2. Start the CLI: `make run-cli`
3. Create some alerts and watch price streams

## Performance Characteristics

- **Latency**: <150ms for local operations
- **Throughput**: Handles thousands of price ticks per second
- **Concurrency**: Thread-safe operations with minimal locking
- **Memory**: Efficient channel-based messaging with bounded buffers
- **Backpressure**: Automatic handling of slow consumers

## Configuration

### Server Configuration

The server uses these default settings:

- **Port**: 9090
- **Data Source**: Live Binance WebSocket (real-time)
- **Alert Cooldown**: 30 seconds
- **Buffer Sizes**: 1000 for price ticks, 100 for alert triggers

### Supported Crypto Symbols

Live prices from Binance (as of October 2025):
- **BTC**: ~$109,000 (Bitcoin)
- **ETH**: ~$3,800 (Ethereum)
- **ADA**: ~$0.60 (Cardano)
- **SOL**: ~$185 (Solana)
- **DOT**: ~$2.85 (Polkadot)
- **MATIC**: ~$0.40 (Polygon)
- **AVAX**: ~$18 (Avalanche)
- **LINK**: ~$17 (Chainlink)

*Note: Prices update in real-time via Binance WebSocket. Mock data is available as fallback.*

## API Reference

### gRPC Services

#### CryptoMarketData Service

```protobuf
service CryptoMarketData {
  rpc SubscribePrices(PriceSubscriptionRequest) returns (stream PriceTick);
}
```

#### CryptoAlertService

```protobuf
service CryptoAlertService {
  rpc CreateAlert(CreateAlertRequest) returns (CreateAlertResponse);
  rpc GetAlerts(GetAlertsRequest) returns (GetAlertsResponse);
  rpc UpdateAlert(UpdateAlertRequest) returns (UpdateAlertResponse);
  rpc DeleteAlert(DeleteAlertRequest) returns (DeleteAlertResponse);
  rpc SubscribeAlerts(AlertSubscriptionRequest) returns (stream AlertTrigger);
}
```  
