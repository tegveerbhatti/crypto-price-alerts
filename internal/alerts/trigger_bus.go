package alerts

import (
	"context"
	"sync"

	"crypto-price-alerts/pkg/models"
)

type TriggerSubscriber struct {
	ID          string
	TriggerChan chan *models.AlertTrigger
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewTriggerSubscriber(id string, bufferSize int) *TriggerSubscriber {
	ctx, cancel := context.WithCancel(context.Background())

	return &TriggerSubscriber{
		ID:          id,
		TriggerChan: make(chan *models.AlertTrigger, bufferSize),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (ts *TriggerSubscriber) Close() {
	ts.cancel()
	close(ts.TriggerChan)
}

type TriggerBus struct {
	subscribers  map[string]*TriggerSubscriber
	mu           sync.RWMutex
	triggerChan  chan *models.AlertTrigger
	stopChan     chan struct{}
	running      bool
}

func NewTriggerBus() *TriggerBus {
	return &TriggerBus{
		subscribers: make(map[string]*TriggerSubscriber),
		triggerChan: make(chan *models.AlertTrigger, 1000),
		stopChan:    make(chan struct{}),
	}
}

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

func (tb *TriggerBus) Stop() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if !tb.running {
		return
	}

	tb.running = false
	close(tb.stopChan)

	for _, subscriber := range tb.subscribers {
		subscriber.Close()
	}

	close(tb.triggerChan)
}

func (tb *TriggerBus) Subscribe(subscriberID string, bufferSize int) *TriggerSubscriber {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if existing, exists := tb.subscribers[subscriberID]; exists {
		existing.Close()
	}

	subscriber := NewTriggerSubscriber(subscriberID, bufferSize)
	tb.subscribers[subscriberID] = subscriber

	return subscriber
}

func (tb *TriggerBus) Unsubscribe(subscriberID string) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if subscriber, exists := tb.subscribers[subscriberID]; exists {
		subscriber.Close()
		delete(tb.subscribers, subscriberID)
	}
}

func (tb *TriggerBus) Publish(trigger *models.AlertTrigger) {
	select {
	case tb.triggerChan <- trigger:
	default:
	}
}

func (tb *TriggerBus) GetSubscriberCount() int {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return len(tb.subscribers)
}

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

func (tb *TriggerBus) fanOutTrigger(trigger *models.AlertTrigger) {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	for _, subscriber := range tb.subscribers {
		select {
		case subscriber.TriggerChan <- trigger:
		case <-subscriber.ctx.Done():
		default:
		}
	}
}

func (tb *TriggerBus) GetStats() TriggerBusStats {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	return TriggerBusStats{
		Running:           tb.running,
		SubscriberCount:   len(tb.subscribers),
		QueuedTriggers:    len(tb.triggerChan),
	}
}

type TriggerBusStats struct {
	Running         bool `json:"running"`
	SubscriberCount int  `json:"subscriber_count"`
	QueuedTriggers  int  `json:"queued_triggers"`
}
