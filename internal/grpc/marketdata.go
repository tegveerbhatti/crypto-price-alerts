package grpc

import (
	"log"

	pb "crypto-price-alerts/api/gen/crypto-price-alerts/api/gen"
	"crypto-price-alerts/internal/pubsub"
	"crypto-price-alerts/pkg/models"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// CryptoMarketDataServer implements the CryptoMarketData gRPC service
type CryptoMarketDataServer struct {
	pb.UnimplementedCryptoMarketDataServer
	broker *pubsub.Broker
}

// NewCryptoMarketDataServer creates a new CryptoMarketData gRPC server
func NewCryptoMarketDataServer(broker *pubsub.Broker) *CryptoMarketDataServer {
	return &CryptoMarketDataServer{
		broker: broker,
	}
}

// SubscribePrices streams price updates for the requested symbols
func (s *CryptoMarketDataServer) SubscribePrices(req *pb.PriceSubscriptionRequest, stream pb.CryptoMarketData_SubscribePricesServer) error {
	if len(req.Symbols) == 0 {
		return stream.Context().Err()
	}

	// Create a unique subscriber ID for this stream
	subscriberID := generateSubscriberID()
	
	log.Printf("Client subscribing to price updates for symbols: %v (subscriber: %s)", req.Symbols, subscriberID)

	// Subscribe to the broker with a reasonable buffer size
	subscriber := s.broker.Subscribe(subscriberID, req.Symbols, 100)
	defer s.broker.Unsubscribe(subscriberID)

	// Stream price ticks to the client
	for {
		select {
		case <-stream.Context().Done():
			log.Printf("Client disconnected from price stream (subscriber: %s)", subscriberID)
			return stream.Context().Err()
		case tick, ok := <-subscriber.TickChan:
			if !ok {
				// Channel closed
				return nil
			}

			// Convert internal tick to protobuf message
			pbTick := &pb.PriceTick{
				Symbol:    tick.Symbol,
				Price:     tick.Price,
				Timestamp: timestamppb.New(tick.Timestamp),
			}

			// Send tick to client
			if err := stream.Send(pbTick); err != nil {
				log.Printf("Error sending price tick to client (subscriber: %s): %v", subscriberID, err)
				return err
			}
		}
	}
}

// convertTickToProto converts an internal Tick to a protobuf PriceTick
func convertTickToProto(tick *models.Tick) *pb.PriceTick {
	return &pb.PriceTick{
		Symbol:    tick.Symbol,
		Price:     tick.Price,
		Timestamp: timestamppb.New(tick.Timestamp),
	}
}
