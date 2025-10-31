.PHONY: help build run-server run-cli proto clean test lint docker-build docker-run

# Default target
help:
	@echo "Crypto Price Alert Engine - Available Commands:"
	@echo ""
	@echo "Development:"
	@echo "  make build        - Build server and CLI binaries"
	@echo "  make run-server   - Run the gRPC server"
	@echo "  make run-cli      - Run the CLI client"
	@echo "  make proto        - Generate protobuf code"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  make test         - Run all tests"
	@echo "  make test-race    - Run tests with race detector"
	@echo "  make lint         - Run linter (requires golangci-lint)"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run server in Docker container"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make deps         - Download dependencies"

# Build binaries
build:
	@echo "Building binaries..."
	go build -o bin/server ./cmd/server
	go build -o bin/cli ./cmd/cli
	@echo "✅ Build complete! Binaries available in ./bin/"

# Run the server
run-server:
	@echo "Starting gRPC server..."
	go run ./cmd/server

# Run the CLI client
run-cli:
	@echo "Starting CLI client..."
	go run ./cmd/cli

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@mkdir -p api/gen
	protoc --go_out=api/gen --go-grpc_out=api/gen api/cryptoalert.proto
	@echo "✅ Protobuf code generated!"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test -race -v ./...

# Run linter (requires golangci-lint to be installed)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not found. Install it with:"; \
		echo "    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf api/gen/
	go clean -cache
	@echo "✅ Clean complete!"

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t crypto-price-alerts .
	@echo "✅ Docker image built!"

# Docker run
docker-run:
	@echo "Running server in Docker container..."
	docker run -p 8080:8080 crypto-price-alerts

# Install protobuf tools
install-proto-tools:
	@echo "Installing protobuf tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "✅ Protobuf tools installed!"

# Development setup
setup: deps install-proto-tools proto
	@echo "✅ Development environment setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Run 'make run-server' to start the server"
	@echo "  2. In another terminal, run 'make run-cli' to start the client"

# Quick start (build and run server)
start: build
	@echo "Starting server..."
	./bin/server

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted!"

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...
	@echo "✅ Code vetted!"

# Run all quality checks
check: fmt vet lint test
	@echo "✅ All quality checks passed!"
