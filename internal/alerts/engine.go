package alerts

import (
	"context"
	"log"
	"sync"
	"time"

	"crypto-price-alerts/pkg/models"
)

// Engine evaluates price ticks against alert rules and emits triggers
type Engine struct {
	store       *Store
	triggerBus  *TriggerBus
	tickChan    chan *models.Tick
	stopChan    chan struct{}
	running     bool
	mu          sync.RWMutex
	cooldownMap map[string]time.Time // Alert ID -> last trigger time for cooldown
	cooldown    time.Duration        // Minimum time between triggers for the same alert
}

// NewEngine creates a new alert rule engine
func NewEngine(store *Store, triggerBus *TriggerBus, cooldown time.Duration) *Engine {
	return &Engine{
		store:       store,
		triggerBus:  triggerBus,
		tickChan:    make(chan *models.Tick, 1000),
		stopChan:    make(chan struct{}),
		cooldownMap: make(map[string]time.Time),
		cooldown:    cooldown,
	}
}

// Start begins processing price ticks
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = true
	e.mu.Unlock()

	go e.processTicks(ctx)
	return nil
}

// Stop stops the rule engine
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	e.running = false
	close(e.stopChan)
	close(e.tickChan)
}

// ProcessTick processes a single price tick
func (e *Engine) ProcessTick(tick *models.Tick) {
	select {
	case e.tickChan <- tick:
		// Tick queued successfully
	default:
		// Channel is full, drop the tick
		log.Printf("Warning: Dropping tick for %s due to full channel", tick.Symbol)
	}
}

// processTicks runs the main tick processing loop
func (e *Engine) processTicks(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case tick := <-e.tickChan:
			e.evaluateTick(tick)
		}
	}
}

// evaluateTick evaluates a price tick against all alerts for the symbol
func (e *Engine) evaluateTick(tick *models.Tick) {
	// Get all enabled alerts for this symbol
	alerts := e.store.GetEnabledBySymbol(tick.Symbol)

	for _, alert := range alerts {
		if e.shouldTriggerAlert(alert, tick.Price) {
			e.triggerAlert(alert, tick.Price)
		}
	}
}

// shouldTriggerAlert checks if an alert should trigger based on price and cooldown
func (e *Engine) shouldTriggerAlert(alert *models.Alert, price float64) bool {
	// Check if alert condition is met
	if !alert.ShouldTrigger(price) {
		return false
	}

	// Check cooldown
	e.mu.RLock()
	lastTrigger, exists := e.cooldownMap[alert.ID]
	e.mu.RUnlock()

	if exists && time.Since(lastTrigger) < e.cooldown {
		return false // Still in cooldown period
	}

	return true
}

// triggerAlert creates and publishes an alert trigger
func (e *Engine) triggerAlert(alert *models.Alert, triggeredPrice float64) {
	// Update cooldown map
	e.mu.Lock()
	e.cooldownMap[alert.ID] = time.Now()
	e.mu.Unlock()

	// Mark alert as triggered in store
	if err := e.store.MarkTriggered(alert.ID); err != nil {
		log.Printf("Error marking alert %s as triggered: %v", alert.ID, err)
	}

	// Create trigger event
	trigger := models.NewAlertTrigger(alert, triggeredPrice)

	// Publish trigger
	e.triggerBus.Publish(trigger)

	log.Printf("Alert triggered: %s %s %.2f (triggered at %.2f)",
		alert.Symbol, alert.Comparator.String(), alert.Threshold, triggeredPrice)
}

// GetStats returns engine statistics
func (e *Engine) GetStats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return EngineStats{
		Running:         e.running,
		CooldownEntries: len(e.cooldownMap),
		QueuedTicks:     len(e.tickChan),
	}
}

// CleanupCooldowns removes old cooldown entries to prevent memory leaks
func (e *Engine) CleanupCooldowns() {
	e.mu.Lock()
	defer e.mu.Unlock()

	cutoff := time.Now().Add(-e.cooldown * 2) // Keep entries for 2x cooldown period

	for alertID, lastTrigger := range e.cooldownMap {
		if lastTrigger.Before(cutoff) {
			delete(e.cooldownMap, alertID)
		}
	}
}

// EngineStats contains runtime statistics for the engine
type EngineStats struct {
	Running         bool `json:"running"`
	CooldownEntries int  `json:"cooldown_entries"`
	QueuedTicks     int  `json:"queued_ticks"`
}
