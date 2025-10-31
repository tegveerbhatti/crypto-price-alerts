package pubsub

import (
	"context"
	"sync"

	"crypto-price-alerts/pkg/models"
)

type Subscriber struct {
	ID       string
	Symbols  map[string]bool
	TickChan chan *models.Tick
	ctx      context.Context
	cancel   context.CancelFunc
}

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

func (s *Subscriber) Close() {
	s.cancel()
	close(s.TickChan)
}

func (s *Subscriber) IsInterestedIn(symbol string) bool {
	return s.Symbols[symbol]
}

type Broker struct {
	subscribers map[string]*Subscriber
	mu          sync.RWMutex
	tickChan    chan *models.Tick
	stopChan    chan struct{}
	running     bool
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string]*Subscriber),
		tickChan:    make(chan *models.Tick, 10000),
		stopChan:    make(chan struct{}),
	}
}

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

func (b *Broker) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if !b.running {
		return
	}
	
	b.running = false
	close(b.stopChan)
	
	for _, subscriber := range b.subscribers {
		subscriber.Close()
	}
	
	close(b.tickChan)
}

func (b *Broker) Subscribe(subscriberID string, symbols []string, bufferSize int) *Subscriber {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if existing, exists := b.subscribers[subscriberID]; exists {
		existing.Close()
	}
	
	subscriber := NewSubscriber(subscriberID, symbols, bufferSize)
	b.subscribers[subscriberID] = subscriber
	
	return subscriber
}

func (b *Broker) Unsubscribe(subscriberID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if subscriber, exists := b.subscribers[subscriberID]; exists {
		subscriber.Close()
		delete(b.subscribers, subscriberID)
	}
}

func (b *Broker) Publish(tick *models.Tick) {
	select {
	case b.tickChan <- tick:
	default:
	}
}

func (b *Broker) GetSubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

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

func (b *Broker) fanOutTick(tick *models.Tick) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	for _, subscriber := range b.subscribers {
		if subscriber.IsInterestedIn(tick.Symbol) {
			select {
			case subscriber.TickChan <- tick:
			case <-subscriber.ctx.Done():
			default:
			}
		}
	}
}

func (b *Broker) UpdateSubscription(subscriberID string, symbols []string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	subscriber, exists := b.subscribers[subscriberID]
	if !exists {
		return false
	}
	
	symbolSet := make(map[string]bool)
	for _, symbol := range symbols {
		symbolSet[symbol] = true
	}
	subscriber.Symbols = symbolSet
	
	return true
}

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
