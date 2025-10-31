package grpc

import (
	"log"

	pb "crypto-price-alerts/api/gen/crypto-price-alerts/api/gen"
	"crypto-price-alerts/internal/pubsub"
	"crypto-price-alerts/pkg/models"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type CryptoMarketDataServer struct {
	pb.UnimplementedCryptoMarketDataServer
	broker *pubsub.Broker
}

func NewCryptoMarketDataServer(broker *pubsub.Broker) *CryptoMarketDataServer {
	return &CryptoMarketDataServer{
		broker: broker,
	}
}

func (s *CryptoMarketDataServer) SubscribePrices(req *pb.PriceSubscriptionRequest, stream pb.CryptoMarketData_SubscribePricesServer) error {
	if len(req.Symbols) == 0 {
		return stream.Context().Err()
	}

	subscriberID := generateSubscriberID()
	
	log.Printf("Client subscribing to price updates for symbols: %v (subscriber: %s)", req.Symbols, subscriberID)

	subscriber := s.broker.Subscribe(subscriberID, req.Symbols, 100)
	defer s.broker.Unsubscribe(subscriberID)

	for {
		select {
		case <-stream.Context().Done():
			log.Printf("Client disconnected from price stream (subscriber: %s)", subscriberID)
			return stream.Context().Err()
		case tick, ok := <-subscriber.TickChan:
			if !ok {
				return nil
			}

			pbTick := &pb.PriceTick{
				Symbol:    tick.Symbol,
				Price:     tick.Price,
				Timestamp: timestamppb.New(tick.Timestamp),
			}

			if err := stream.Send(pbTick); err != nil {
				log.Printf("Error sending price tick to client (subscriber: %s): %v", subscriberID, err)
				return err
			}
		}
	}
}

func convertTickToProto(tick *models.Tick) *pb.PriceTick {
	return &pb.PriceTick{
		Symbol:    tick.Symbol,
		Price:     tick.Price,
		Timestamp: timestamppb.New(tick.Timestamp),
	}
}
