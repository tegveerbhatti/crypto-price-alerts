package grpc

import (
	"context"
	"log"

	pb "crypto-price-alerts/api/gen/crypto-price-alerts/api/gen"
	"crypto-price-alerts/internal/alerts"
	"crypto-price-alerts/pkg/models"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CryptoAlertServiceServer implements the CryptoAlertService gRPC service
type CryptoAlertServiceServer struct {
	pb.UnimplementedCryptoAlertServiceServer
	store      *alerts.Store
	triggerBus *alerts.TriggerBus
}

// NewCryptoAlertServiceServer creates a new CryptoAlertService gRPC server
func NewCryptoAlertServiceServer(store *alerts.Store, triggerBus *alerts.TriggerBus) *CryptoAlertServiceServer {
	return &CryptoAlertServiceServer{
		store:      store,
		triggerBus: triggerBus,
	}
}

// CreateAlert creates a new price alert
func (s *CryptoAlertServiceServer) CreateAlert(ctx context.Context, req *pb.CreateAlertRequest) (*pb.CreateAlertResponse, error) {
	if req.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	if req.Threshold <= 0 {
		return nil, status.Error(codes.InvalidArgument, "threshold must be positive")
	}

	// Convert protobuf comparator to internal type
	comparator := convertComparatorFromProto(req.Comparator)
	if comparator == models.ComparatorUnspecified {
		return nil, status.Error(codes.InvalidArgument, "invalid comparator")
	}

	// Create new alert
	alert := models.NewAlert(req.Symbol, comparator, req.Threshold, req.Note)

	// Store the alert
	if err := s.store.Create(alert); err != nil {
		log.Printf("Error creating alert: %v", err)
		return nil, status.Error(codes.Internal, "failed to create alert")
	}

	log.Printf("Created alert: %s %s %.2f for symbol %s", 
		alert.Symbol, alert.Comparator.String(), alert.Threshold, alert.Symbol)

	return &pb.CreateAlertResponse{
		Alert: convertAlertToProto(alert),
	}, nil
}

// GetAlerts retrieves all alerts
func (s *CryptoAlertServiceServer) GetAlerts(ctx context.Context, req *pb.GetAlertsRequest) (*pb.GetAlertsResponse, error) {
	alerts := s.store.GetAll()

	pbAlerts := make([]*pb.Alert, len(alerts))
	for i, alert := range alerts {
		pbAlerts[i] = convertAlertToProto(alert)
	}

	return &pb.GetAlertsResponse{
		Alerts: pbAlerts,
	}, nil
}

// UpdateAlert updates an existing alert
func (s *CryptoAlertServiceServer) UpdateAlert(ctx context.Context, req *pb.UpdateAlertRequest) (*pb.UpdateAlertResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "alert ID is required")
	}

	// Build updates map
	updates := make(map[string]interface{})

	if req.Symbol != nil {
		if *req.Symbol == "" {
			return nil, status.Error(codes.InvalidArgument, "symbol cannot be empty")
		}
		updates["symbol"] = *req.Symbol
	}

	if req.Comparator != nil {
		comparator := convertComparatorFromProto(*req.Comparator)
		if comparator == models.ComparatorUnspecified {
			return nil, status.Error(codes.InvalidArgument, "invalid comparator")
		}
		updates["comparator"] = comparator
	}

	if req.Threshold != nil {
		if *req.Threshold <= 0 {
			return nil, status.Error(codes.InvalidArgument, "threshold must be positive")
		}
		updates["threshold"] = *req.Threshold
	}

	if req.Note != nil {
		updates["note"] = *req.Note
	}

	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	// Update the alert
	alert, err := s.store.Update(req.Id, updates)
	if err != nil {
		if err == alerts.ErrAlertNotFound {
			return nil, status.Error(codes.NotFound, "alert not found")
		}
		log.Printf("Error updating alert: %v", err)
		return nil, status.Error(codes.Internal, "failed to update alert")
	}

	log.Printf("Updated alert: %s", req.Id)

	return &pb.UpdateAlertResponse{
		Alert: convertAlertToProto(alert),
	}, nil
}

// DeleteAlert deletes an alert
func (s *CryptoAlertServiceServer) DeleteAlert(ctx context.Context, req *pb.DeleteAlertRequest) (*pb.DeleteAlertResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "alert ID is required")
	}

	err := s.store.Delete(req.Id)
	if err != nil {
		if err == alerts.ErrAlertNotFound {
			return nil, status.Error(codes.NotFound, "alert not found")
		}
		log.Printf("Error deleting alert: %v", err)
		return nil, status.Error(codes.Internal, "failed to delete alert")
	}

	log.Printf("Deleted alert: %s", req.Id)

	return &pb.DeleteAlertResponse{
		Success: true,
	}, nil
}

// SubscribeAlerts streams alert triggers to the client
func (s *CryptoAlertServiceServer) SubscribeAlerts(req *pb.AlertSubscriptionRequest, stream pb.CryptoAlertService_SubscribeAlertsServer) error {
	// Create a unique subscriber ID for this stream
	subscriberID := generateSubscriberID()
	
	log.Printf("Client subscribing to alert triggers (subscriber: %s)", subscriberID)

	// Subscribe to the trigger bus with a reasonable buffer size
	subscriber := s.triggerBus.Subscribe(subscriberID, 100)
	defer s.triggerBus.Unsubscribe(subscriberID)

	// Stream alert triggers to the client
	for {
		select {
		case <-stream.Context().Done():
			log.Printf("Client disconnected from alert stream (subscriber: %s)", subscriberID)
			return stream.Context().Err()
		case trigger, ok := <-subscriber.TriggerChan:
			if !ok {
				// Channel closed
				return nil
			}

			// Convert internal trigger to protobuf message
			pbTrigger := convertAlertTriggerToProto(trigger)

			// Send trigger to client
			if err := stream.Send(pbTrigger); err != nil {
				log.Printf("Error sending alert trigger to client (subscriber: %s): %v", subscriberID, err)
				return err
			}
		}
	}
}

// convertComparatorFromProto converts a protobuf Comparator to internal type
func convertComparatorFromProto(pbComparator pb.Comparator) models.Comparator {
	switch pbComparator {
	case pb.Comparator_COMPARATOR_GT:
		return models.ComparatorGT
	case pb.Comparator_COMPARATOR_GTE:
		return models.ComparatorGTE
	case pb.Comparator_COMPARATOR_LT:
		return models.ComparatorLT
	case pb.Comparator_COMPARATOR_LTE:
		return models.ComparatorLTE
	case pb.Comparator_COMPARATOR_EQ:
		return models.ComparatorEQ
	default:
		return models.ComparatorUnspecified
	}
}

// convertComparatorToProto converts an internal Comparator to protobuf type
func convertComparatorToProto(comparator models.Comparator) pb.Comparator {
	switch comparator {
	case models.ComparatorGT:
		return pb.Comparator_COMPARATOR_GT
	case models.ComparatorGTE:
		return pb.Comparator_COMPARATOR_GTE
	case models.ComparatorLT:
		return pb.Comparator_COMPARATOR_LT
	case models.ComparatorLTE:
		return pb.Comparator_COMPARATOR_LTE
	case models.ComparatorEQ:
		return pb.Comparator_COMPARATOR_EQ
	default:
		return pb.Comparator_COMPARATOR_UNSPECIFIED
	}
}

// convertAlertToProto converts an internal Alert to protobuf Alert
func convertAlertToProto(alert *models.Alert) *pb.Alert {
	pbAlert := &pb.Alert{
		Id:         alert.ID,
		Symbol:     alert.Symbol,
		Comparator: convertComparatorToProto(alert.Comparator),
		Threshold:  alert.Threshold,
		Note:       alert.Note,
		Enabled:    alert.Enabled,
	}

	if alert.LastTrigger != nil {
		pbAlert.LastTrigger = timestamppb.New(*alert.LastTrigger)
	}

	return pbAlert
}

// convertAlertTriggerToProto converts an internal AlertTrigger to protobuf AlertTrigger
func convertAlertTriggerToProto(trigger *models.AlertTrigger) *pb.AlertTrigger {
	return &pb.AlertTrigger{
		Alert:          convertAlertToProto(trigger.Alert),
		TriggeredPrice: trigger.TriggeredPrice,
		Timestamp:      timestamppb.New(trigger.Timestamp),
	}
}
