package models

import (
	"testing"
)

func TestAlert_ShouldTrigger(t *testing.T) {
	tests := []struct {
		name       string
		alert      *Alert
		price      float64
		expected   bool
	}{
		{
			name: "GT trigger when price is greater",
			alert: &Alert{
				Comparator: ComparatorGT,
				Threshold:  100.0,
				Enabled:    true,
			},
			price:    101.0,
			expected: true,
		},
		{
			name: "GT no trigger when price is equal",
			alert: &Alert{
				Comparator: ComparatorGT,
				Threshold:  100.0,
				Enabled:    true,
			},
			price:    100.0,
			expected: false,
		},
		{
			name: "GTE trigger when price is equal",
			alert: &Alert{
				Comparator: ComparatorGTE,
				Threshold:  100.0,
				Enabled:    true,
			},
			price:    100.0,
			expected: true,
		},
		{
			name: "LT trigger when price is less",
			alert: &Alert{
				Comparator: ComparatorLT,
				Threshold:  100.0,
				Enabled:    true,
			},
			price:    99.0,
			expected: true,
		},
		{
			name: "LTE trigger when price is equal",
			alert: &Alert{
				Comparator: ComparatorLTE,
				Threshold:  100.0,
				Enabled:    true,
			},
			price:    100.0,
			expected: true,
		},
		{
			name: "EQ trigger when price is equal",
			alert: &Alert{
				Comparator: ComparatorEQ,
				Threshold:  100.0,
				Enabled:    true,
			},
			price:    100.0,
			expected: true,
		},
		{
			name: "No trigger when disabled",
			alert: &Alert{
				Comparator: ComparatorGT,
				Threshold:  100.0,
				Enabled:    false,
			},
			price:    101.0,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.alert.ShouldTrigger(tt.price)
			if result != tt.expected {
				t.Errorf("ShouldTrigger() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestNewAlert(t *testing.T) {
	alert := NewAlert("AAPL", ComparatorGT, 150.0, "Test alert")

	if alert.ID == "" {
		t.Error("Expected non-empty ID")
	}

	if alert.Symbol != "AAPL" {
		t.Errorf("Expected symbol AAPL, got %s", alert.Symbol)
	}

	if alert.Comparator != ComparatorGT {
		t.Errorf("Expected ComparatorGT, got %v", alert.Comparator)
	}

	if alert.Threshold != 150.0 {
		t.Errorf("Expected threshold 150.0, got %f", alert.Threshold)
	}

	if alert.Note != "Test alert" {
		t.Errorf("Expected note 'Test alert', got %s", alert.Note)
	}

	if !alert.Enabled {
		t.Error("Expected alert to be enabled by default")
	}

	if alert.LastTrigger != nil {
		t.Error("Expected LastTrigger to be nil initially")
	}
}

func TestComparator_String(t *testing.T) {
	tests := []struct {
		comparator Comparator
		expected   string
	}{
		{ComparatorGT, ">"},
		{ComparatorGTE, ">="},
		{ComparatorLT, "<"},
		{ComparatorLTE, "<="},
		{ComparatorEQ, "=="},
		{ComparatorUnspecified, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.comparator.String()
			if result != tt.expected {
				t.Errorf("String() = %s, expected %s", result, tt.expected)
			}
		})
	}
}
