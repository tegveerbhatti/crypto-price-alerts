package alerts

import (
	"errors"
	"sync"

	"crypto-price-alerts/pkg/models"
)

var (
	ErrAlertNotFound = errors.New("alert not found")
	ErrAlertExists   = errors.New("alert already exists")
)

type Store struct {
	alerts map[string]*models.Alert
	symbolIndex map[string][]string
	mu          sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		alerts:      make(map[string]*models.Alert),
		symbolIndex: make(map[string][]string),
	}
}

func (s *Store) Create(alert *models.Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.alerts[alert.ID]; exists {
		return ErrAlertExists
	}

	s.alerts[alert.ID] = alert

	s.addToSymbolIndex(alert.Symbol, alert.ID)

	return nil
}

func (s *Store) Get(id string) (*models.Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alert, exists := s.alerts[id]
	if !exists {
		return nil, ErrAlertNotFound
	}

	alertCopy := *alert
	return &alertCopy, nil
}

func (s *Store) Update(id string, updates map[string]interface{}) (*models.Alert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert, exists := s.alerts[id]
	if !exists {
		return nil, ErrAlertNotFound
	}

	oldSymbol := alert.Symbol

	for field, value := range updates {
		switch field {
		case "symbol":
			if symbol, ok := value.(string); ok {
				alert.Symbol = symbol
			}
		case "comparator":
			if comparator, ok := value.(models.Comparator); ok {
				alert.Comparator = comparator
			}
		case "threshold":
			if threshold, ok := value.(float64); ok {
				alert.Threshold = threshold
			}
		case "note":
			if note, ok := value.(string); ok {
				alert.Note = note
			}
		case "enabled":
			if enabled, ok := value.(bool); ok {
				alert.Enabled = enabled
			}
		}
	}

	if oldSymbol != alert.Symbol {
		s.removeFromSymbolIndex(oldSymbol, id)
		s.addToSymbolIndex(alert.Symbol, id)
	}

	alertCopy := *alert
	return &alertCopy, nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert, exists := s.alerts[id]
	if !exists {
		return ErrAlertNotFound
	}

	s.removeFromSymbolIndex(alert.Symbol, id)

	delete(s.alerts, id)

	return nil
}

func (s *Store) GetAll() []*models.Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alerts := make([]*models.Alert, 0, len(s.alerts))
	for _, alert := range s.alerts {
		alertCopy := *alert
		alerts = append(alerts, &alertCopy)
	}

	return alerts
}

func (s *Store) GetBySymbol(symbol string) []*models.Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alertIDs, exists := s.symbolIndex[symbol]
	if !exists {
		return []*models.Alert{}
	}

	alerts := make([]*models.Alert, 0, len(alertIDs))
	for _, id := range alertIDs {
		if alert, exists := s.alerts[id]; exists {
			alertCopy := *alert
			alerts = append(alerts, &alertCopy)
		}
	}

	return alerts
}

func (s *Store) GetEnabledBySymbol(symbol string) []*models.Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alertIDs, exists := s.symbolIndex[symbol]
	if !exists {
		return []*models.Alert{}
	}

	alerts := make([]*models.Alert, 0, len(alertIDs))
	for _, id := range alertIDs {
		if alert, exists := s.alerts[id]; exists && alert.Enabled {
			alertCopy := *alert
			alerts = append(alerts, &alertCopy)
		}
	}

	return alerts
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.alerts)
}

func (s *Store) CountBySymbol(symbol string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alertIDs, exists := s.symbolIndex[symbol]
	if !exists {
		return 0
	}

	return len(alertIDs)
}

func (s *Store) GetActiveSymbols() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	symbols := make([]string, 0, len(s.symbolIndex))
	for symbol := range s.symbolIndex {
		symbols = append(symbols, symbol)
	}

	return symbols
}

func (s *Store) MarkTriggered(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert, exists := s.alerts[id]
	if !exists {
		return ErrAlertNotFound
	}

	alert.MarkTriggered()
	return nil
}

func (s *Store) addToSymbolIndex(symbol, alertID string) {
	if alertIDs, exists := s.symbolIndex[symbol]; exists {
		for _, id := range alertIDs {
			if id == alertID {
				return
			}
		}
		s.symbolIndex[symbol] = append(alertIDs, alertID)
	} else {
		s.symbolIndex[symbol] = []string{alertID}
	}
}

func (s *Store) removeFromSymbolIndex(symbol, alertID string) {
	alertIDs, exists := s.symbolIndex[symbol]
	if !exists {
		return
	}

	for i, id := range alertIDs {
		if id == alertID {
			s.symbolIndex[symbol] = append(alertIDs[:i], alertIDs[i+1:]...)
			break
		}
	}

	if len(s.symbolIndex[symbol]) == 0 {
		delete(s.symbolIndex, symbol)
	}
}
