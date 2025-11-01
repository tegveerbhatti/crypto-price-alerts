package benchmarks

import (
	"context"
	"crypto-price-alerts/internal/alerts"
	"crypto-price-alerts/internal/datafeed"
	"crypto-price-alerts/internal/pubsub"
	"crypto-price-alerts/pkg/models"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	
	"github.com/google/uuid"
)

func TestThroughputMetrics(t *testing.T) {
	t.Run("TickProcessingThroughput", func(t *testing.T) {
		measureTickProcessingThroughput(t)
	})
	
	t.Run("AlertEvaluationThroughput", func(t *testing.T) {
		measureAlertEvaluationThroughput(t)
	})
	
	t.Run("PubSubThroughput", func(t *testing.T) {
		measurePubSubThroughput(t)
	})
	
	t.Run("EndToEndThroughput", func(t *testing.T) {
		measureEndToEndThroughput(t)
	})
}

func measureTickProcessingThroughput(t *testing.T) {
	symbols := []string{"BTC", "ETH", "ADA", "SOL", "DOT", "MATIC", "AVAX", "LINK"}
	mockFeed := datafeed.NewMockDataFeed(symbols, 1*time.Millisecond)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	mockFeed.Start(ctx)
	defer mockFeed.Stop()
	
	var tickCount int64
	start := time.Now()
	
	go func() {
		for tick := range mockFeed.TickChannel() {
			atomic.AddInt64(&tickCount, 1)
			_ = tick
		}
	}()
	
	time.Sleep(5 * time.Second)
	duration := time.Since(start)
	
	finalCount := atomic.LoadInt64(&tickCount)
	throughput := float64(finalCount) / duration.Seconds()
	
	t.Logf("Tick Processing Throughput: %.0f ticks/second", throughput)
	t.Logf("Total ticks processed: %d in %v", finalCount, duration)
	
	if throughput < 1000 {
		t.Errorf("Expected throughput > 1000 ticks/second, got %.0f", throughput)
	}
}

func measureAlertEvaluationThroughput(t *testing.T) {
	store := alerts.NewStore()
	triggerBus := alerts.NewTriggerBus()
	engine := alerts.NewEngine(store, triggerBus, 10*time.Millisecond)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	triggerBus.Start(ctx)
	engine.Start(ctx)
	
	symbols := []string{"BTC", "ETH", "ADA", "SOL", "DOT", "MATIC", "AVAX", "LINK"}
	alertsPerSymbol := 1250
	
	for _, symbol := range symbols {
		for i := 0; i < alertsPerSymbol; i++ {
		alert := &models.Alert{
			ID:         uuid.New().String(),
			Symbol:     symbol,
			Comparator: models.ComparatorGT,
			Threshold:  1000.0,
			Enabled:    true,
		}
			store.Create(alert)
		}
	}
	
	var evaluationCount int64
	start := time.Now()
	
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(1 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				for _, symbol := range symbols {
					tick := models.NewTick(symbol, 2000.0)
					engine.ProcessTick(tick)
					atomic.AddInt64(&evaluationCount, int64(alertsPerSymbol))
				}
			}
		}
	}()
	
	time.Sleep(5 * time.Second)
	close(done)
	duration := time.Since(start)
	
	finalCount := atomic.LoadInt64(&evaluationCount)
	throughput := float64(finalCount) / duration.Seconds()
	
	t.Logf("Alert Evaluation Throughput: %.0f evaluations/second", throughput)
	t.Logf("Total evaluations: %d in %v", finalCount, duration)
	
	if throughput < 100000 {
		t.Errorf("Expected throughput > 100,000 evaluations/second, got %.0f", throughput)
	}
}

func measurePubSubThroughput(t *testing.T) {
	broker := pubsub.NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	broker.Start(ctx)
	defer broker.Stop()
	
	numSubscribers := 1000
	symbols := []string{"BTC", "ETH", "ADA", "SOL"}
	
	var receivedCount int64
	var wg sync.WaitGroup
	
	for i := 0; i < numSubscribers; i++ {
		subscriber := broker.Subscribe(uuid.New().String(), symbols, 1000)
		wg.Add(1)
		
		go func() {
			defer wg.Done()
			for range subscriber.TickChan {
				atomic.AddInt64(&receivedCount, 1)
			}
		}()
	}
	
	var publishCount int64
	start := time.Now()
	
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(100 * time.Microsecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				for _, symbol := range symbols {
					tick := models.NewTick(symbol, 50000.0)
					broker.Publish(tick)
					atomic.AddInt64(&publishCount, 1)
				}
			}
		}
	}()
	
	time.Sleep(5 * time.Second)
	close(done)
	duration := time.Since(start)
	
	time.Sleep(100 * time.Millisecond)
	broker.Stop()
	wg.Wait()
	
	finalPublished := atomic.LoadInt64(&publishCount)
	finalReceived := atomic.LoadInt64(&receivedCount)
	
	publishThroughput := float64(finalPublished) / duration.Seconds()
	receiveThroughput := float64(finalReceived) / duration.Seconds()
	
	t.Logf("Publish Throughput: %.0f messages/second", publishThroughput)
	t.Logf("Receive Throughput: %.0f messages/second", receiveThroughput)
	t.Logf("Fan-out Ratio: %.1fx", float64(finalReceived)/float64(finalPublished))
	
	if publishThroughput < 10000 {
		t.Errorf("Expected publish throughput > 10,000 messages/second, got %.0f", publishThroughput)
	}
}

func measureEndToEndThroughput(t *testing.T) {
	broker := pubsub.NewBroker()
	store := alerts.NewStore()
	triggerBus := alerts.NewTriggerBus()
	engine := alerts.NewEngine(store, triggerBus, 10*time.Millisecond)
	
	symbols := []string{"BTC", "ETH", "ADA", "SOL", "DOT", "MATIC", "AVAX", "LINK"}
	mockFeed := datafeed.NewMockDataFeed(symbols, 500*time.Microsecond)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	broker.Start(ctx)
	triggerBus.Start(ctx)
	engine.Start(ctx)
	mockFeed.Start(ctx)
	
	numSubscribers := 100
	numAlertsPerSymbol := 100
	
	for i := 0; i < numSubscribers; i++ {
		broker.Subscribe(uuid.New().String(), symbols, 1000)
	}
	
	for _, symbol := range symbols {
		for i := 0; i < numAlertsPerSymbol; i++ {
		alert := &models.Alert{
			ID:         uuid.New().String(),
			Symbol:     symbol,
			Comparator: models.ComparatorGT,
			Threshold:  1000.0,
			Enabled:    true,
		}
			store.Create(alert)
		}
	}
	
	triggerSubscriber := triggerBus.Subscribe(uuid.New().String(), 1000)
	
	var ticksProcessed int64
	var alertsTriggered int64
	
	go func() {
		for tick := range mockFeed.TickChannel() {
			broker.Publish(tick)
			engine.ProcessTick(tick)
			atomic.AddInt64(&ticksProcessed, 1)
		}
	}()
	
	go func() {
		for range triggerSubscriber.TriggerChan {
			atomic.AddInt64(&alertsTriggered, 1)
		}
	}()
	
	start := time.Now()
	time.Sleep(10 * time.Second)
	duration := time.Since(start)
	
	mockFeed.Stop()
	broker.Stop()
	engine.Stop()
	triggerBus.Stop()
	
	finalTicks := atomic.LoadInt64(&ticksProcessed)
	finalTriggers := atomic.LoadInt64(&alertsTriggered)
	
	tickThroughput := float64(finalTicks) / duration.Seconds()
	triggerThroughput := float64(finalTriggers) / duration.Seconds()
	
	t.Logf("End-to-End Metrics:")
	t.Logf("  Tick Throughput: %.0f ticks/second", tickThroughput)
	t.Logf("  Alert Trigger Throughput: %.0f triggers/second", triggerThroughput)
	t.Logf("  Total Ticks Processed: %d", finalTicks)
	t.Logf("  Total Alerts Triggered: %d", finalTriggers)
	t.Logf("  Trigger Rate: %.2f%% of ticks triggered alerts", 
		float64(finalTriggers)/float64(finalTicks)*100)
	
	if tickThroughput < 1000 {
		t.Errorf("Expected end-to-end tick throughput > 1,000/second, got %.0f", tickThroughput)
	}
}

func TestLatencyMetrics(t *testing.T) {
	broker := pubsub.NewBroker()
	store := alerts.NewStore()
	triggerBus := alerts.NewTriggerBus()
	engine := alerts.NewEngine(store, triggerBus, 1*time.Millisecond)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	broker.Start(ctx)
	triggerBus.Start(ctx)
	engine.Start(ctx)
	
	alert := &models.Alert{
		ID:         uuid.New().String(),
		Symbol:     "BTC",
		Comparator: models.ComparatorGT,
		Threshold:  50000.0,
		Enabled:    true,
	}
	store.Create(alert)
	
	triggerSubscriber := triggerBus.Subscribe(uuid.New().String(), 100)
	
	numTests := 1000
	latencies := make([]time.Duration, numTests)
	
	for i := 0; i < numTests; i++ {
		start := time.Now()
		
		tick := models.NewTick("BTC", 60000.0)
		broker.Publish(tick)
		engine.ProcessTick(tick)
		
		select {
		case <-triggerSubscriber.TriggerChan:
			latencies[i] = time.Since(start)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for alert trigger")
		}
		
		time.Sleep(10 * time.Millisecond)
	}
	
	var total time.Duration
	min := latencies[0]
	max := latencies[0]
	
	for _, latency := range latencies {
		total += latency
		if latency < min {
			min = latency
		}
		if latency > max {
			max = latency
		}
	}
	
	avg := total / time.Duration(numTests)
	
	t.Logf("Latency Metrics (n=%d):", numTests)
	t.Logf("  Average: %v", avg)
	t.Logf("  Minimum: %v", min)
	t.Logf("  Maximum: %v", max)
	
	if avg > 50*time.Millisecond {
		t.Errorf("Expected average latency < 50ms, got %v", avg)
	}
}
