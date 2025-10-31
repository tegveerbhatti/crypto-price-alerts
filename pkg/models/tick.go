package models

import "time"

type Tick struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

func NewTick(symbol string, price float64) *Tick {
	return &Tick{
		Symbol:    symbol,
		Price:     price,
		Timestamp: time.Now(),
	}
}
