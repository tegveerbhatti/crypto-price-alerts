package models

import (
	"time"

	"github.com/google/uuid"
)

// Comparator represents the comparison operation for an alert
type Comparator int

const (
	ComparatorUnspecified Comparator = iota
	ComparatorGT                     // Greater than
	ComparatorGTE                    // Greater than or equal
	ComparatorLT                     // Less than
	ComparatorLTE                    // Less than or equal
	ComparatorEQ                     // Equal
)

// String returns the string representation of the comparator
func (c Comparator) String() string {
	switch c {
	case ComparatorGT:
		return ">"
	case ComparatorGTE:
		return ">="
	case ComparatorLT:
		return "<"
	case ComparatorLTE:
		return "<="
	case ComparatorEQ:
		return "=="
	default:
		return "unknown"
	}
}

// Alert represents a price alert rule
type Alert struct {
	ID          string     `json:"id"`
	Symbol      string     `json:"symbol"`
	Comparator  Comparator `json:"comparator"`
	Threshold   float64    `json:"threshold"`
	Note        string     `json:"note"`
	Enabled     bool       `json:"enabled"`
	LastTrigger *time.Time `json:"last_trigger,omitempty"`
}

// NewAlert creates a new alert with a generated UUID
func NewAlert(symbol string, comparator Comparator, threshold float64, note string) *Alert {
	return &Alert{
		ID:         uuid.New().String(),
		Symbol:     symbol,
		Comparator: comparator,
		Threshold:  threshold,
		Note:       note,
		Enabled:    true,
	}
}

// ShouldTrigger evaluates if the alert should trigger for the given price
func (a *Alert) ShouldTrigger(price float64) bool {
	if !a.Enabled {
		return false
	}

	switch a.Comparator {
	case ComparatorGT:
		return price > a.Threshold
	case ComparatorGTE:
		return price >= a.Threshold
	case ComparatorLT:
		return price < a.Threshold
	case ComparatorLTE:
		return price <= a.Threshold
	case ComparatorEQ:
		// Use a small epsilon for floating point comparison
		const epsilon = 0.001
		return abs(price-a.Threshold) < epsilon
	default:
		return false
	}
}

// MarkTriggered updates the last trigger time
func (a *Alert) MarkTriggered() {
	now := time.Now()
	a.LastTrigger = &now
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// AlertTrigger represents a triggered alert event
type AlertTrigger struct {
	Alert          *Alert    `json:"alert"`
	TriggeredPrice float64   `json:"triggered_price"`
	Timestamp      time.Time `json:"timestamp"`
}

// NewAlertTrigger creates a new alert trigger event
func NewAlertTrigger(alert *Alert, triggeredPrice float64) *AlertTrigger {
	return &AlertTrigger{
		Alert:          alert,
		TriggeredPrice: triggeredPrice,
		Timestamp:      time.Now(),
	}
}
