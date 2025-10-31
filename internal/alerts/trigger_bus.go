package alerts

import (
	"context"
	"sync"

	"crypto-price-alerts/pkg/models"
)

// TriggerSubscriber represents a subscriber to alert triggers
type TriggerSubscriber struct {
	ID          string
	TriggerChan chan *models.AlertTrigger
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewTriggerSubscriber creates a new trigger subscriber
func NewTriggerSubscriber(id string, bufferSize int) *TriggerSubscriber {
	ctx, cancel := context.WithCancel(context.Background())

	return &TriggerSubscriber{
		ID:          id,
		TriggerChan: make(chan *models.AlertTrigger, bufferSize),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Close closes the trigger subscriber and cleans up resources
func (ts *TriggerSubscriber) Close() {
	ts.cancel()
	close(ts.TriggerChan)
}

// TriggerBus manages pub/sub for alert triggers
type TriggerBus struct {
	subscribers  map[string]*TriggerSubscriber
	mu           sync.RWMutex
	triggerChan  chan *models.AlertTrigger
	stopChan     chan struct{}
	running      bool
}

// NewTriggerBus creates a new trigger bus
func NewTriggerBus() *TriggerBus {
	return &TriggerBus{
		subscribers: make(map[string]*TriggerSubscriber),
		triggerChan: make(chan *models.AlertTrigger, 1000),
		stopChan:    make(chan struct{}),
	}
}

// Start begins the trigger distribution loop
func (tb *TriggerBus) Start(ctx context.Context) error {
	tb.mu.Lock()
	if tb.running {
		tb.mu.Unlock()
		return nil
	}
	tb.running = true
	tb.mu.Unlock()

	go tb.distributeTriggers(ctx)
	return nil
}

// Stop stops the trigger bus
func (tb *TriggerBus) Stop() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if !tb.running {
		return
	}

	tb.running = false
	close(tb.stopChan)

	// Close all subscribers
	for _, subscriber := range tb.subscribers {
		subscriber.Close()
	}

	close(tb.triggerChan)
}

// Subscribe adds a new subscriber for alert triggers
func (tb *TriggerBus) Subscribe(subscriberID string, bufferSize int) *TriggerSubscriber {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Remove existing subscriber if it exists
	if existing, exists := tb.subscribers[subscriberID]; exists {
		existing.Close()
	}

	subscriber := NewTriggerSubscriber(subscriberID, bufferSize)
	tb.subscribers[subscriberID] = subscriber

	return subscriber
}

// Unsubscribe removes a subscriber
func (tb *TriggerBus) Unsubscribe(subscriberID string) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if subscriber, exists := tb.subscribers[subscriberID]; exists {
		subscriber.Close()
		delete(tb.subscribers, subscriberID)
	}
}

// Publish publishes an alert trigger to all subscribers
func (tb *TriggerBus) Publish(trigger *models.AlertTrigger) {
	select {
	case tb.triggerChan <- trigger:
		// Trigger queued successfully
	default:
		// Channel is full, drop the trigger
		// In production, you might want to log this or implement backpressure
	}
}

// GetSubscriberCount returns the current number of subscribers
func (tb *TriggerBus) GetSubscriberCount() int {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return len(tb.subscribers)
}

// distributeTriggers runs the main distribution loop
func (tb *TriggerBus) distributeTriggers(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-tb.stopChan:
			return
		case trigger := <-tb.triggerChan:
			tb.fanOutTrigger(trigger)
		}
	}
}

// fanOutTrigger distributes a trigger to all subscribers
func (tb *TriggerBus) fanOutTrigger(trigger *models.AlertTrigger) {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	for _, subscriber := range tb.subscribers {
		select {
		case subscriber.TriggerChan <- trigger:
			// Trigger sent successfully
		case <-subscriber.ctx.Done():
			// Subscriber is closed, skip
		default:
			// Subscriber's channel is full, drop the trigger for this subscriber
			// This implements backpressure by dropping messages for slow consumers
		}
	}
}

// GetStats returns trigger bus statistics
func (tb *TriggerBus) GetStats() TriggerBusStats {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	return TriggerBusStats{
		Running:           tb.running,
		SubscriberCount:   len(tb.subscribers),
		QueuedTriggers:    len(tb.triggerChan),
	}
}

// TriggerBusStats contains runtime statistics for the trigger bus
type TriggerBusStats struct {
	Running         bool `json:"running"`
	SubscriberCount int  `json:"subscriber_count"`
	QueuedTriggers  int  `json:"queued_triggers"`
}
