package datafeed

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"crypto-price-alerts/pkg/models"

	"github.com/gorilla/websocket"
)

// BinanceDataFeed connects to Binance WebSocket for real-time crypto prices
type BinanceDataFeed struct {
	symbols  []string
	tickChan chan *models.Tick
	stopChan chan struct{}
	running  bool
	mu       sync.RWMutex
	conn     *websocket.Conn
}

// BinanceTickerMessage represents Binance WebSocket 24hr ticker message
// Being very explicit about field mapping to avoid Go's JSON ambiguity
type BinanceTickerMessage struct {
	Symbol        string          `json:"s"`  // Symbol (e.g., "BTCUSDT")
	PriceRaw      json.RawMessage `json:"c"`  // Current close price (lowercase c!)
	CloseTime     int64           `json:"C"`  // Close time (uppercase C) - explicitly mapped to avoid confusion
}

// NewBinanceDataFeed creates a new Binance WebSocket data feed
func NewBinanceDataFeed(symbols []string) *BinanceDataFeed {
	return &BinanceDataFeed{
		symbols:  symbols,
		tickChan: make(chan *models.Tick, 1000),
		stopChan: make(chan struct{}),
	}
}

// Start begins receiving real-time crypto prices from Binance
func (b *BinanceDataFeed) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return nil
	}
	b.running = true
	b.mu.Unlock()

	// Convert symbols to Binance format (e.g., BTC -> btcusdt)
	streams := make([]string, len(b.symbols))
	for i, symbol := range b.symbols {
		streams[i] = strings.ToLower(symbol) + "usdt@ticker"
	}

	// Build WebSocket URL
	streamParam := strings.Join(streams, "/")
	wsURL := fmt.Sprintf("wss://stream.binance.com:9443/ws/%s", streamParam)

	log.Printf("Connecting to Binance WebSocket: %s", wsURL)

	// Connect to Binance WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Binance WebSocket: %v", err)
	}

	b.conn = conn
	go b.readMessages(ctx)

	return nil
}

// Stop stops the Binance data feed
func (b *BinanceDataFeed) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return
	}

	b.running = false
	close(b.stopChan)

	if b.conn != nil {
		b.conn.Close()
	}

	close(b.tickChan)
}

// TickChannel returns the channel for receiving price ticks
func (b *BinanceDataFeed) TickChannel() <-chan *models.Tick {
	return b.tickChan
}

// readMessages reads and processes WebSocket messages
func (b *BinanceDataFeed) readMessages(ctx context.Context) {
	defer b.conn.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.stopChan:
			return
		default:
			var msg BinanceTickerMessage
			err := b.conn.ReadJSON(&msg)
			if err != nil {
				log.Printf("Error reading WebSocket message: %v", err)
				return
			}

			b.processTicker(msg)
		}
	}
}

// processTicker converts Binance ticker to internal tick format
func (b *BinanceDataFeed) processTicker(msg BinanceTickerMessage) {
	// Handle Binance's inconsistent price format (string OR number)
	var price float64
	var err error
	
	// Try parsing as number first
	if err = json.Unmarshal(msg.PriceRaw, &price); err != nil {
		// If that fails, try parsing as string
		var priceStr string
		if err = json.Unmarshal(msg.PriceRaw, &priceStr); err != nil {
			log.Printf("❌ Error parsing price from JSON: %v", err)
			return
		}
		price, err = strconv.ParseFloat(priceStr, 64)
		if err != nil {
			log.Printf("❌ Error converting price string %s to float: %v", priceStr, err)
			return
		}
	}
	
	// Convert symbol (BTCUSDT -> BTC)
	symbol := strings.TrimSuffix(msg.Symbol, "USDT")
	
	// Log real-time price updates
	log.Printf("LIVE: %s = $%.2f", symbol, price)

	// Create tick
	tick := models.NewTick(symbol, price)

	// Send to channel
	select {
	case b.tickChan <- tick:
		// Tick sent successfully
	default:
		// Channel full, drop tick
	}
}

// GetCurrentPrice returns the current price for a symbol (not implemented for WebSocket feed)
func (b *BinanceDataFeed) GetCurrentPrice(symbol string) (float64, bool) {
	// For WebSocket feeds, we don't maintain current prices
	// This would require additional state management
	return 0, false
}
