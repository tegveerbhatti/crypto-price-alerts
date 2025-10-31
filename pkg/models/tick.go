package models

import "time"

// Tick represents a real-time price update for a stock symbol
type Tick struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

// NewTick creates a new price tick with the current timestamp
func NewTick(symbol string, price float64) *Tick {
	return &Tick{
		Symbol:    symbol,
		Price:     price,
		Timestamp: time.Now(),
	}
}
