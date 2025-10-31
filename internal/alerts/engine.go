package alerts

import (
	"context"
	"log"
	"sync"
	"time"

	"crypto-price-alerts/pkg/models"
)

type Engine struct {
	store       *Store
	triggerBus  *TriggerBus
	tickChan    chan *models.Tick
	stopChan    chan struct{}
	running     bool
	mu          sync.RWMutex
	cooldownMap map[string]time.Time
	cooldown    time.Duration
}

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

func (e *Engine) ProcessTick(tick *models.Tick) {
	select {
	case e.tickChan <- tick:
	default:
		log.Printf("Warning: Dropping tick for %s due to full channel", tick.Symbol)
	}
}

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

func (e *Engine) evaluateTick(tick *models.Tick) {
	alerts := e.store.GetEnabledBySymbol(tick.Symbol)

	for _, alert := range alerts {
		if e.shouldTriggerAlert(alert, tick.Price) {
			e.triggerAlert(alert, tick.Price)
		}
	}
}

func (e *Engine) shouldTriggerAlert(alert *models.Alert, price float64) bool {
	if !alert.ShouldTrigger(price) {
		return false
	}


	e.mu.RLock()
	lastTrigger, exists := e.cooldownMap[alert.ID]
	e.mu.RUnlock()

	if exists && time.Since(lastTrigger) < e.cooldown {
		return false
	}

	return true
}

func (e *Engine) triggerAlert(alert *models.Alert, triggeredPrice float64) {
	e.mu.Lock()
	e.cooldownMap[alert.ID] = time.Now()
	e.mu.Unlock()


	if err := e.store.MarkTriggered(alert.ID); err != nil {
		log.Printf("Error marking alert %s as triggered: %v", alert.ID, err)
	}

	trigger := models.NewAlertTrigger(alert, triggeredPrice)

	e.triggerBus.Publish(trigger)

	log.Printf("Alert triggered: %s %s %.2f (triggered at %.2f)",
		alert.Symbol, alert.Comparator.String(), alert.Threshold, triggeredPrice)
}

func (e *Engine) GetStats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return EngineStats{
		Running:         e.running,
		CooldownEntries: len(e.cooldownMap),
		QueuedTicks:     len(e.tickChan),
	}
}

func (e *Engine) CleanupCooldowns() {
	e.mu.Lock()
	defer e.mu.Unlock()

	cutoff := time.Now().Add(-e.cooldown * 2)

	for alertID, lastTrigger := range e.cooldownMap {
		if lastTrigger.Before(cutoff) {
			delete(e.cooldownMap, alertID)
		}
	}
}

type EngineStats struct {
	Running         bool `json:"running"`
	CooldownEntries int  `json:"cooldown_entries"`
	QueuedTicks     int  `json:"queued_ticks"`
}
