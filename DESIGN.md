# Crypto Price Alert Engine — DESIGN.md

## Overview

This system provides real-time cryptocurrency price alerts. Users subscribe to crypto symbols and define alert rules (e.g., BTC ≥ 45000). The backend streams crypto prices and notifies clients when rules trigger.

This project demonstrates:
- High-frequency streaming architecture
- gRPC bidirectional + server-stream RPC patterns
- In-memory + optional persistence
- Clean modular Go architecture
- Pub/sub messaging
- Basic rule engine

Initial scope targets mock crypto feed simulation, then extends to real crypto data from Binance WebSocket API.

## High-Level Architecture

```text
+-----------------------------+
|         Client(s)           |
|   CLI / TUI / Dashboard     |
|  - Subscribes to alerts     |
|  - Creates rules            |
+-------------+---------------+
              | gRPC
              v
+-------------+---------------+
|        gRPC API Layer       |
|  - MarketData service       |
|  - AlertService             |
+-------------+---------------+
              |
              v
+--------------------------------------+
|           Core Services              |
|   Pub/Sub Broker      Alert Engine   |
|  - Symbol channels   - Evaluate rules|
|  - Fan-out ticks     - Emit triggers |
+--------------------------------------+
              |
              v
+-------------------------------+
|   Data Feed (Mock or Live)    |
|  - Simulated ticks            |
|  - Later: Alpaca, IEX, Finnhub|
+-------------------------------+
```

## Tech Stack

### Languages
- Go 1.22+

### Communication
- gRPC over HTTP/2
- Protobuf definitions for contracts

### Modules

| Piece       | Technology                               |
|-------------|------------------------------------------|
| Streaming   | gRPC server-stream                       |
| Data feed   | Mock generator → optional real API       |
| Pub/Sub     | Channel-based broker → optional Redis/NATS |
| Storage     | In-memory → optional SQLite/Postgres     |
| CLI         | Go gRPC client                           |

### Dev Tools
- `protoc` + `buf` (optional)
- `go test`, race detector
- Docker (optional production)

## Functional Requirements

### User Capabilities
- Subscribe to crypto symbols
- Create price rules:
  - GT, GTE, LT, LTE, EQ
- Receive push alerts in real-time
- List, update, disable rules

### Non-Functional
- Low latency (<150ms local)
- Handle bursty events
- Do not drop alerts, buffer slow clients
- Horizontally scalable pub/sub design

## Data Model

### Alert

| Field        | Type                      | Description             |
|--------------|---------------------------|-------------------------|
| id           | `string` (UUID)           | Unique alert identifier |
| symbol       | `string` (e.g., "BTC")    | Crypto ticker symbol    |
| comparator   | `enum`                    | GT, GTE, LT, LTE, EQ    |
| threshold    | `float64`                 | Price threshold to trigger |
| note         | `string`                  | Optional user note      |
| enabled      | `bool`                    | Is the alert active     |
| last_trigger | `time`                    | For cooldown logic      |

### Tick

| Field  | Type      | Description         |
|--------|-----------|---------------------|
| symbol | `string`  | Crypto ticker symbol |
| price  | `float64` | Current price       |
| ts     | `unix time` | Timestamp of tick   |

## API Contract

**Location**: `api/cryptoalert.proto`

(Provided previously; reference that file here.)

### Services:
- `CryptoMarketData.SubscribePrices()`
- `CryptoAlertService.CreateAlert()`
- `CryptoAlertService.SubscribeAlerts()`
- CRUD for rules

### Streaming:
- Server pushes `PriceTick`
- Server pushes `AlertTrigger`

## Directory Structure

```
crypto-alert/
├── api/
│   ├── gen/                # protoc output
│   └── cryptoalert.proto
├── cmd/
│   ├── cli/
│   │   └── main.go
│   └── server/
│       └── main.go
├── deploy/
│   └── docker-compose.yml
├── internal/
│   ├── alerts/
│   │   ├── engine.go        # rule engine
│   │   ├── store.go         # in-memory alert store
│   │   └── trigger_bus.go   # trigger pub/sub
│   ├── datafeed/
│   │   ├── iex.go           # real provider (later)
│   │   └── mock.go          # simulated ticks
│   ├── grpc/
│   │   ├── alertservice.go
│   │   └── marketdata.go    # implements proto service
│   └── pubsub/
│       └── broker.go        # channel-based fan-out
├── pkg/
│   └── models/
│       ├── alert.go
│       └── tick.go
├── DESIGN.md
├── Makefile
├── README.md
└── go.mod
```

## Component Design

### 1. Data Feed
Responsible for producing real-time ticks.
- **Initial**: mock generator random-walk
- **Later**: live API (Alpaca web socket or IEX SSE)

**Key responsibilities:**
- Connect to provider
- Normalize price events
- Publish ticks to broker

### 2. Pub/Sub Broker
Fans price events to consumers.
- Thread-safe
- Bounded buffers
- Per-symbol subscriber list

### 3. Alert Store
Simple in-memory map: `map[symbol][]Alert`
- **Optional persistence backend**:
  - SQLite
  - Redis JSON
  - Postgres

### 4. Rule Engine
Executes on each tick.
- **Algorithm complexity**: O(N_symbol_alerts)
- **Functions**:
  - Load alerts for symbol
  - Evaluate comparator
  - Cooldown enforcement
  - Emit triggers

### 5. Trigger Bus
Pub/sub for triggered alerts. Used by gRPC server streams.

### 6. gRPC Server
- **`MarketData.SubscribePrices`**:
  - Client requests symbols
  - Stream singleton feed subset to client
- **`AlertService.SubscribeAlerts`**:
  - Stream user alert events
- **CRUD**:
  - Validate inputs
  - Index by symbol

## Implementation Notes

### Comparators
`func compare(price float64, cmp Comparator, threshold float64) bool`

### Concurrency Strategy
- Use `RWMutex` around alert map
- Channels for events
- Context-driven shutdown
- Avoid global state; inject dependencies

### Backpressure
- Bounded channels for streams
- Drop oldest or disconnect slow client

### Ticker Granularity
- Mock tick every ~200ms.

### Metrics to Expose (optional)
- Alerts created
- Alerts triggered
- Tick latency
- Stream backpressure count

## Testing Plan

| Layer       | Tests                                     |
|-------------|-------------------------------------------|
| `compare` fn| Unit test each comparator branch          |
| Rule engine | Given tick → emits correct triggers       |
| Broker      | Subscribe/unsubscribe, concurrent fan-out |
| gRPC        | Integration test via `bufconn`            |
| CLI         | Manual + scripted demonstration           |

## Local Development

### Start server
```bash
go run ./cmd/server
```

### Run client
```bash
go run ./cmd/cli
```

### Generate protos
```bash
protoc --go_out=api/gen --go-grpc_out=api/gen api/stockalert.proto
```

## Extensions Roadmap

### Phase 1
- Mock feed
- Basic CRUD + stream
- Bin-exec CLI client

### Phase 2
- SQLite storage
- REST gateway for web UI
- Backtesting simulated price replay
- Optional WebSockets UI

### Phase 3
- NATS or Redis pub/sub
- Distributed alert engine (sharding symbols)
- ML-based trading signals (future project synergy)

## Security Considerations
- mTLS (optional local dev)
- Do not expose real brokerage credentials
- Rate limit client connections
- Validate symbol format

## Deployment
- Docker image with compiled Go binary
- Run as single service initially
- Scale broker workers later

## Success Criteria
- Works in real time
- Alert rules are consistent & correct
- Handles concurrent streams
- Clean code structure + clear interfaces
- Extensible to real stock data source