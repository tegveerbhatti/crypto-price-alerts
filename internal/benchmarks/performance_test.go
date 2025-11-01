package benchmarks

import (
	"context"
	"crypto-price-alerts/internal/alerts"
	"crypto-price-alerts/internal/pubsub"
	"crypto-price-alerts/pkg/models"
	"sync"
	"testing"
	"time"
	
	"github.com/google/uuid"
)

func BenchmarkAlertEngine(b *testing.B) {
	store := alerts.NewStore()
	triggerBus := alerts.NewTriggerBus()
	engine := alerts.NewEngine(store, triggerBus, 1*time.Second)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	triggerBus.Start(ctx)
	engine.Start(ctx)
	
	for i := 0; i < 1000; i++ {
		alert := &models.Alert{
			ID:         uuid.New().String(),
			Symbol:     "BTC",
			Comparator: models.ComparatorGT,
			Threshold:  50000.0,
			Enabled:    true,
		}
		store.Create(alert)
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		tick := models.NewTick("BTC", 60000.0)
		engine.ProcessTick(tick)
	}
}

func BenchmarkPubSubBroker(b *testing.B) {
	broker := pubsub.NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	broker.Start(ctx)
	
	numSubscribers := 100
	for i := 0; i < numSubscribers; i++ {
		broker.Subscribe(uuid.New().String(), []string{"BTC", "ETH"}, 1000)
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		tick := models.NewTick("BTC", 60000.0)
		broker.Publish(tick)
	}
}

func BenchmarkConcurrentAlertProcessing(b *testing.B) {
	store := alerts.NewStore()
	triggerBus := alerts.NewTriggerBus()
	engine := alerts.NewEngine(store, triggerBus, 100*time.Millisecond)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	triggerBus.Start(ctx)
	engine.Start(ctx)
	
	symbols := []string{"BTC", "ETH", "ADA", "SOL", "DOT", "MATIC", "AVAX", "LINK"}
	for _, symbol := range symbols {
		for i := 0; i < 100; i++ {
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
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for _, symbol := range symbols {
				tick := models.NewTick(symbol, 2000.0)
				engine.ProcessTick(tick)
			}
		}
	})
}

func BenchmarkHighVolumeTickProcessing(b *testing.B) {
	broker := pubsub.NewBroker()
	store := alerts.NewStore()
	triggerBus := alerts.NewTriggerBus()
	engine := alerts.NewEngine(store, triggerBus, 50*time.Millisecond)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	broker.Start(ctx)
	triggerBus.Start(ctx)
	engine.Start(ctx)
	
	numSubscribers := 50
	numAlertsPerSymbol := 200
	symbols := []string{"BTC", "ETH", "ADA", "SOL", "DOT", "MATIC", "AVAX", "LINK"}
	
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
	
	b.ResetTimer()
	b.ReportAllocs()
	
	var wg sync.WaitGroup
	
	for i := 0; i < b.N; i++ {
		wg.Add(len(symbols))
		for _, symbol := range symbols {
			go func(sym string) {
				defer wg.Done()
				tick := models.NewTick(sym, 2000.0)
				broker.Publish(tick)
				engine.ProcessTick(tick)
			}(symbol)
		}
		wg.Wait()
	}
}

func BenchmarkMemoryEfficiency(b *testing.B) {
	store := alerts.NewStore()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		alert := &models.Alert{
			ID:         uuid.New().String(),
			Symbol:     "BTC",
			Comparator: models.ComparatorGT,
			Threshold:  50000.0,
			Enabled:    true,
		}
		store.Create(alert)
		
		alerts := store.GetEnabledBySymbol("BTC")
		for _, a := range alerts {
			_ = a.ShouldTrigger(60000.0)
		}
	}
}

func BenchmarkLatency(b *testing.B) {
	broker := pubsub.NewBroker()
	store := alerts.NewStore()
	triggerBus := alerts.NewTriggerBus()
	engine := alerts.NewEngine(store, triggerBus, 10*time.Millisecond)
	
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
	
	subscriber := triggerBus.Subscribe(uuid.New().String(), 100)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		start := time.Now()
		
		tick := models.NewTick("BTC", 60000.0)
		broker.Publish(tick)
		engine.ProcessTick(tick)
		
		select {
		case <-subscriber.TriggerChan:
			latency := time.Since(start)
			b.ReportMetric(float64(latency.Nanoseconds()), "ns/op")
		case <-time.After(100 * time.Millisecond):
			b.Fatal("Timeout waiting for alert trigger")
		}
	}
}
