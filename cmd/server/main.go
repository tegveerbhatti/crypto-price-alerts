package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "crypto-price-alerts/api/gen/crypto-price-alerts/api/gen"
	"crypto-price-alerts/internal/alerts"
	"crypto-price-alerts/internal/datafeed"
	grpchandlers "crypto-price-alerts/internal/grpc"
	"crypto-price-alerts/internal/pubsub"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":9090"
	tickRate = 200 * time.Millisecond
	alertCooldown = 30 * time.Second
)

func main() {
	log.Println("Starting Crypto Price Alert Engine...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()


	broker := pubsub.NewBroker()
	alertStore := alerts.NewStore()
	triggerBus := alerts.NewTriggerBus()
	alertEngine := alerts.NewEngine(alertStore, triggerBus, alertCooldown)

	symbols := []string{"BTC", "ETH", "ADA", "SOL", "DOT", "MATIC", "AVAX", "LINK"}
	binanceFeed := datafeed.NewBinanceDataFeed(symbols)


	log.Println("Starting services...")
	
	if err := broker.Start(ctx); err != nil {
		log.Fatalf("Failed to start broker: %v", err)
	}

	if err := triggerBus.Start(ctx); err != nil {
		log.Fatalf("Failed to start trigger bus: %v", err)
	}

	if err := alertEngine.Start(ctx); err != nil {
		log.Fatalf("Failed to start alert engine: %v", err)
	}

	if err := binanceFeed.Start(ctx); err != nil {
		log.Fatalf("Failed to start Binance data feed: %v", err)
	}

	go func() {
		for tick := range binanceFeed.TickChannel() {
			broker.Publish(tick)
			alertEngine.ProcessTick(tick)
		}
	}()

	log.Printf("Attempting to bind to port %s...", port)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", port, err)
	}
	log.Printf("Successfully bound to %s", lis.Addr().String())

	grpcServer := grpc.NewServer()

	cryptoMarketDataServer := grpchandlers.NewCryptoMarketDataServer(broker)
	cryptoAlertServiceServer := grpchandlers.NewCryptoAlertServiceServer(alertStore, triggerBus)

	pb.RegisterCryptoMarketDataServer(grpcServer, cryptoMarketDataServer)
	pb.RegisterCryptoAlertServiceServer(grpcServer, cryptoAlertServiceServer)

	reflection.Register(grpcServer)


	go func() {
		log.Printf("🚀 gRPC server starting on %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				alertEngine.CleanupCooldowns()
			}
		}
	}()

	log.Printf("Server started successfully!")
	log.Printf("Real-time Binance WebSocket connected!")
	log.Printf("Available crypto symbols: %v", symbols)
	log.Printf("Alert cooldown: %v", alertCooldown)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")

	grpcServer.GracefulStop()
	
	binanceFeed.Stop()
	alertEngine.Stop()
	triggerBus.Stop()
	broker.Stop()

	log.Println("Server stopped")
}
