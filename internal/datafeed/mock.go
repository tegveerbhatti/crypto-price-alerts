package datafeed

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"crypto-price-alerts/pkg/models"
)

// MockDataFeed simulates real-time cryptocurrency price data using random walk
type MockDataFeed struct {
	symbols    []string
	prices     map[string]float64
	mu         sync.RWMutex
	tickChan   chan *models.Tick
	stopChan   chan struct{}
	running    bool
	tickRate   time.Duration
}

// NewMockDataFeed creates a new mock data feed with initial crypto prices
func NewMockDataFeed(symbols []string, tickRate time.Duration) *MockDataFeed {
	// Initialize with realistic starting crypto prices (in USD) - October 2025
	initialPrices := map[string]float64{
		"BTC":  110000.00, // Bitcoin - current market price
		"ETH":  4200.00,   // Ethereum
		"ADA":  0.65,      // Cardano
		"SOL":  180.00,    // Solana
		"DOT":  8.50,      // Polkadot
		"MATIC": 1.20,     // Polygon
		"AVAX": 45.00,     // Avalanche
		"LINK": 18.50,     // Chainlink
	}

	prices := make(map[string]float64)
	for _, symbol := range symbols {
		if price, exists := initialPrices[symbol]; exists {
			prices[symbol] = price
		} else {
			// Default price for unknown crypto symbols
			prices[symbol] = 1.00 + rand.Float64()*99.00 // Random price between $1-100 for unknown cryptos
		}
	}

	return &MockDataFeed{
		symbols:  symbols,
		prices:   prices,
		tickChan: make(chan *models.Tick, 1000), // Buffered channel
		stopChan: make(chan struct{}),
		tickRate: tickRate,
	}
}

// Start begins generating mock price ticks
func (m *MockDataFeed) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil // Already running
	}
	m.running = true
	m.mu.Unlock()

	go m.generateTicks(ctx)
	return nil
}

// Stop stops the mock data feed
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

// TickChannel returns the channel for receiving price ticks
func (m *MockDataFeed) TickChannel() <-chan *models.Tick {
	return m.tickChan
}

// GetCurrentPrice returns the current price for a symbol
func (m *MockDataFeed) GetCurrentPrice(symbol string) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	price, exists := m.prices[symbol]
	return price, exists
}

// generateTicks runs the main tick generation loop
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

// generateRandomTick creates a new price tick for a random symbol
func (m *MockDataFeed) generateRandomTick() {
	if len(m.symbols) == 0 {
		return
	}

	// Pick a random symbol
	symbol := m.symbols[rand.Intn(len(m.symbols))]
	
	m.mu.Lock()
	currentPrice := m.prices[symbol]
	
	// Generate price movement using random walk
	// Crypto is more volatile - price can move up or down by 0.1% to 5% of current price
	maxChange := currentPrice * 0.05 // 5% max change for crypto volatility
	minChange := currentPrice * 0.001 // 0.1% min change
	
	change := minChange + rand.Float64()*(maxChange-minChange)
	if rand.Float64() < 0.5 {
		change = -change // 50% chance of negative movement
	}
	
	newPrice := currentPrice + change
	
	// Ensure price doesn't go below $0.01 (some cryptos can be very cheap)
	if newPrice < 0.01 {
		newPrice = 0.01
	}
	
	m.prices[symbol] = newPrice
	m.mu.Unlock()

	// Create and send the tick
	tick := models.NewTick(symbol, newPrice)
	
	select {
	case m.tickChan <- tick:
		// Tick sent successfully
	default:
		// Channel is full, drop the tick to prevent blocking
		// In a production system, you might want to log this
	}
}

// AddSymbol adds a new symbol to track
func (m *MockDataFeed) AddSymbol(symbol string, initialPrice float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if symbol already exists
	for _, existing := range m.symbols {
		if existing == symbol {
			return
		}
	}
	
	m.symbols = append(m.symbols, symbol)
	m.prices[symbol] = initialPrice
}

// RemoveSymbol removes a symbol from tracking
func (m *MockDataFeed) RemoveSymbol(symbol string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Remove from symbols slice
	for i, existing := range m.symbols {
		if existing == symbol {
			m.symbols = append(m.symbols[:i], m.symbols[i+1:]...)
			break
		}
	}
	
	// Remove from prices map
	delete(m.prices, symbol)
}
