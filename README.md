# Crypto Price Alert Engine

A high-performance, real-time cryptocurrency price alert system built with Go and gRPC. This system demonstrates streaming architecture, pub/sub messaging, and clean modular design.

## 🚀 Features

- **Real-time Price Streaming**: Subscribe to live cryptocurrency price updates via gRPC streams
- **Flexible Alert Rules**: Create alerts with various comparators (>, >=, <, <=, ==)
- **High-Performance Architecture**: Channel-based pub/sub with backpressure handling
- **Mock Data Feed**: Simulated crypto prices using random walk algorithm with higher volatility
- **Binance Integration**: Real-time crypto prices from Binance WebSocket API
- **Interactive CLI**: Easy-to-use command-line interface for testing
- **Thread-Safe Operations**: Concurrent-safe alert storage and processing
- **Cooldown Management**: Prevents alert spam with configurable cooldown periods
- **Docker Support**: Containerized deployment with health checks

## 🏗️ Architecture

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
│ Mock Data Feed  │             │  Alert Store    │
└─────────────────┘             └─────────────────┘
```

## 📋 Prerequisites

- Go 1.22 or higher
- Protocol Buffers compiler (`protoc`)
- Docker (optional, for containerized deployment)

## 🛠️ Quick Start

### 1. Clone and Setup

```bash
git clone <repository-url>
cd stock-price-alerts
make setup  # Downloads deps, installs tools, generates protobuf code
```

### 2. Run the Server

```bash
make run-server
```

The server will start on `localhost:9090` and begin generating mock price data for:
- BTC, ETH, ADA, SOL, DOT, MATIC, AVAX, LINK

### 3. Run the CLI Client

In a new terminal:

```bash
make run-cli
```

## 🎯 Usage Examples

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
# - BTC > $45000.00
# - ETH <= $2500.00
```

### Monitor Alert Triggers

```bash
# In the CLI, enter:
watch-alerts
```

## 🔧 Development

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

## 🐳 Docker Deployment

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

## 🧪 Testing

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

## 📊 Performance Characteristics

- **Latency**: <150ms for local operations
- **Throughput**: Handles thousands of price ticks per second
- **Concurrency**: Thread-safe operations with minimal locking
- **Memory**: Efficient channel-based messaging with bounded buffers
- **Backpressure**: Automatic handling of slow consumers

## 🔧 Configuration

### Server Configuration

The server uses these default settings:

- **Port**: 8080
- **Tick Rate**: 200ms (configurable in `cmd/server/main.go`)
- **Alert Cooldown**: 30 seconds
- **Buffer Sizes**: 1000 for price ticks, 100 for alert triggers

### Supported Crypto Symbols

Default symbols with realistic starting prices:
- BTC: $43,000.00
- ETH: $2,600.00
- ADA: $0.45
- SOL: $95.00
- DOT: $6.50
- MATIC: $0.85
- AVAX: $36.00
- LINK: $14.50

## 🚦 API Reference

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

### Alert Comparators

- `COMPARATOR_GT`: Greater than (>)
- `COMPARATOR_GTE`: Greater than or equal (>=)
- `COMPARATOR_LT`: Less than (<)
- `COMPARATOR_LTE`: Less than or equal (<=)
- `COMPARATOR_EQ`: Equal (==)

## 🔮 Future Enhancements

### Phase 2
- [ ] SQLite persistence
- [ ] REST API gateway
- [ ] Web dashboard
- [ ] Historical price replay

### Phase 3
- [ ] Redis/NATS pub/sub
- [ ] Distributed alert engine
- [ ] Real crypto data integration (Binance, Coinbase)
- [ ] ML-based trading signals

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run `make check` to ensure quality
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙋‍♂️ Support

For questions or issues:
1. Check the existing issues on GitHub
2. Create a new issue with detailed information
3. Include logs and steps to reproduce

---

**Built with ❤️ using Go, gRPC, and Protocol Buffers**
