package datafeed

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"crypto-price-alerts/pkg/models"
)

type MockDataFeed struct {
	symbols    []string
	prices     map[string]float64
	mu         sync.RWMutex
	tickChan   chan *models.Tick
	stopChan   chan struct{}
	running    bool
	tickRate   time.Duration
}

func NewMockDataFeed(symbols []string, tickRate time.Duration) *MockDataFeed {
	initialPrices := map[string]float64{
		"BTC":  110000.00,
		"ETH":  4200.00,
		"ADA":  0.65,
		"SOL":  180.00,
		"DOT":  8.50,
		"MATIC": 1.20,
		"AVAX": 45.00,
		"LINK": 18.50,
	}

	prices := make(map[string]float64)
	for _, symbol := range symbols {
		if price, exists := initialPrices[symbol]; exists {
			prices[symbol] = price
		} else {
			prices[symbol] = 1.00 + rand.Float64()*99.00
		}
	}

	return &MockDataFeed{
		symbols:  symbols,
		prices:   prices,
		tickChan: make(chan *models.Tick, 1000),
		stopChan: make(chan struct{}),
		tickRate: tickRate,
	}
}

func (m *MockDataFeed) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.mu.Unlock()

	go m.generateTicks(ctx)
	return nil
}

func (m *MockDataFeed) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.running {
		return
	}
	
	m.running = false
	close(m.stopChan)
	close(m.tickChan)
}

func (m *MockDataFeed) TickChannel() <-chan *models.Tick {
	return m.tickChan
}

func (m *MockDataFeed) GetCurrentPrice(symbol string) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	price, exists := m.prices[symbol]
	return price, exists
}

func (m *MockDataFeed) generateTicks(ctx context.Context) {
	ticker := time.NewTicker(m.tickRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.generateRandomTick()
		}
	}
}

func (m *MockDataFeed) generateRandomTick() {
	if len(m.symbols) == 0 {
		return
	}

	symbol := m.symbols[rand.Intn(len(m.symbols))]
	
	m.mu.Lock()
	currentPrice := m.prices[symbol]
	
	maxChange := currentPrice * 0.05
	minChange := currentPrice * 0.001
	
	change := minChange + rand.Float64()*(maxChange-minChange)
	if rand.Float64() < 0.5 {
		change = -change
	}
	
	newPrice := currentPrice + change
	
	if newPrice < 0.01 {
		newPrice = 0.01
	}
	
	m.prices[symbol] = newPrice
	m.mu.Unlock()

	tick := models.NewTick(symbol, newPrice)
	
	select {
	case m.tickChan <- tick:
	default:
	}
}

func (m *MockDataFeed) AddSymbol(symbol string, initialPrice float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for _, existing := range m.symbols {
		if existing == symbol {
			return
		}
	}
	
	m.symbols = append(m.symbols, symbol)
	m.prices[symbol] = initialPrice
}

func (m *MockDataFeed) RemoveSymbol(symbol string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for i, existing := range m.symbols {
		if existing == symbol {
			m.symbols = append(m.symbols[:i], m.symbols[i+1:]...)
			break
		}
	}
	
	delete(m.prices, symbol)
}
