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

type BinanceDataFeed struct {
	symbols  []string
	tickChan chan *models.Tick
	stopChan chan struct{}
	running  bool
	mu       sync.RWMutex
	conn     *websocket.Conn
}

type BinanceTickerMessage struct {
	Symbol        string          `json:"s"`
	PriceRaw      json.RawMessage `json:"c"`
	CloseTime     int64           `json:"C"`
}

func NewBinanceDataFeed(symbols []string) *BinanceDataFeed {
	return &BinanceDataFeed{
		symbols:  symbols,
		tickChan: make(chan *models.Tick, 1000),
		stopChan: make(chan struct{}),
	}
}

func (b *BinanceDataFeed) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return nil
	}
	b.running = true
	b.mu.Unlock()

	streams := make([]string, len(b.symbols))
	for i, symbol := range b.symbols {
		streams[i] = strings.ToLower(symbol) + "usdt@ticker"
	}

	streamParam := strings.Join(streams, "/")
	wsURL := fmt.Sprintf("wss://stream.binance.com:9443/ws/%s", streamParam)

	log.Printf("Connecting to Binance WebSocket: %s", wsURL)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Binance WebSocket: %v", err)
	}

	b.conn = conn
	go b.readMessages(ctx)

	return nil
}

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

func (b *BinanceDataFeed) TickChannel() <-chan *models.Tick {
	return b.tickChan
}

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

func (b *BinanceDataFeed) processTicker(msg BinanceTickerMessage) {
	var price float64
	var err error
	
	if err = json.Unmarshal(msg.PriceRaw, &price); err != nil {
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
	
	symbol := strings.TrimSuffix(msg.Symbol, "USDT")
	
	log.Printf("LIVE: %s = $%.2f", symbol, price)

	tick := models.NewTick(symbol, price)

	select {
	case b.tickChan <- tick:
	default:
	}
}

func (b *BinanceDataFeed) GetCurrentPrice(symbol string) (float64, bool) {
	return 0, false
}
