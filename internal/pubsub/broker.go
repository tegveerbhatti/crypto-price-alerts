package pubsub

import (
	"context"
	"sync"

	"crypto-price-alerts/pkg/models"
)

// Subscriber represents a subscriber to price updates
type Subscriber struct {
	ID       string
	Symbols  map[string]bool // Set of symbols this subscriber is interested in
	TickChan chan *models.Tick
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewSubscriber creates a new subscriber
func NewSubscriber(id string, symbols []string, bufferSize int) *Subscriber {
	ctx, cancel := context.WithCancel(context.Background())
	
	symbolSet := make(map[string]bool)
	for _, symbol := range symbols {
		symbolSet[symbol] = true
	}
	
	return &Subscriber{
		ID:       id,
		Symbols:  symbolSet,
		TickChan: make(chan *models.Tick, bufferSize),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Close closes the subscriber and cleans up resources
func (s *Subscriber) Close() {
	s.cancel()
	close(s.TickChan)
}

// IsInterestedIn checks if the subscriber is interested in a symbol
func (s *Subscriber) IsInterestedIn(symbol string) bool {
	return s.Symbols[symbol]
}

// Broker manages pub/sub for price ticks
type Broker struct {
	subscribers map[string]*Subscriber
	mu          sync.RWMutex
	tickChan    chan *models.Tick
	stopChan    chan struct{}
	running     bool
}

// NewBroker creates a new pub/sub broker
func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string]*Subscriber),
		tickChan:    make(chan *models.Tick, 10000), // Large buffer for high throughput
		stopChan:    make(chan struct{}),
	}
}

// Start begins the broker's message distribution loop
func (b *Broker) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return nil
	}
	b.running = true
	b.mu.Unlock()

	go b.distributeTicks(ctx)
	return nil
}

// Stop stops the broker
func (b *Broker) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if !b.running {
		return
	}
	
	b.running = false
	close(b.stopChan)
	
	// Close all subscribers
	for _, subscriber := range b.subscribers {
		subscriber.Close()
	}
	
	close(b.tickChan)
}

// Subscribe adds a new subscriber for the given symbols
func (b *Broker) Subscribe(subscriberID string, symbols []string, bufferSize int) *Subscriber {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Remove existing subscriber if it exists
	if existing, exists := b.subscribers[subscriberID]; exists {
		existing.Close()
	}
	
	subscriber := NewSubscriber(subscriberID, symbols, bufferSize)
	b.subscribers[subscriberID] = subscriber
	
	return subscriber
}

// Unsubscribe removes a subscriber
func (b *Broker) Unsubscribe(subscriberID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if subscriber, exists := b.subscribers[subscriberID]; exists {
		subscriber.Close()
		delete(b.subscribers, subscriberID)
	}
}

// Publish publishes a tick to all interested subscribers
func (b *Broker) Publish(tick *models.Tick) {
	select {
	case b.tickChan <- tick:
		// Tick queued successfully
	default:
		// Channel is full, drop the tick
		// In production, you might want to log this or implement backpressure
	}
}

// GetSubscriberCount returns the current number of subscribers
func (b *Broker) GetSubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// GetSubscriberCountForSymbol returns the number of subscribers interested in a symbol
func (b *Broker) GetSubscriberCountForSymbol(symbol string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	count := 0
	for _, subscriber := range b.subscribers {
		if subscriber.IsInterestedIn(symbol) {
			count++
		}
	}
	return count
}

// distributeTicks runs the main distribution loop
func (b *Broker) distributeTicks(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.stopChan:
			return
		case tick := <-b.tickChan:
			b.fanOutTick(tick)
		}
	}
}

// fanOutTick distributes a tick to all interested subscribers
func (b *Broker) fanOutTick(tick *models.Tick) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	for _, subscriber := range b.subscribers {
		if subscriber.IsInterestedIn(tick.Symbol) {
			select {
			case subscriber.TickChan <- tick:
				// Tick sent successfully
			case <-subscriber.ctx.Done():
				// Subscriber is closed, skip
			default:
				// Subscriber's channel is full, drop the tick for this subscriber
				// This implements backpressure by dropping messages for slow consumers
			}
		}
	}
}

// UpdateSubscription updates a subscriber's symbol list
func (b *Broker) UpdateSubscription(subscriberID string, symbols []string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	subscriber, exists := b.subscribers[subscriberID]
	if !exists {
		return false
	}
	
	// Update symbol set
	symbolSet := make(map[string]bool)
	for _, symbol := range symbols {
		symbolSet[symbol] = true
	}
	subscriber.Symbols = symbolSet
	
	return true
}

// GetActiveSymbols returns all symbols that have at least one subscriber
func (b *Broker) GetActiveSymbols() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	symbolSet := make(map[string]bool)
	for _, subscriber := range b.subscribers {
		for symbol := range subscriber.Symbols {
			symbolSet[symbol] = true
		}
	}
	
	symbols := make([]string, 0, len(symbolSet))
	for symbol := range symbolSet {
		symbols = append(symbols, symbol)
	}
	
	return symbols
}
