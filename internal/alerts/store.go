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

// Store provides thread-safe storage for alerts
type Store struct {
	// alerts maps alert ID to alert
	alerts map[string]*models.Alert
	// symbolIndex maps symbol to slice of alert IDs for efficient lookup
	symbolIndex map[string][]string
	mu          sync.RWMutex
}

// NewStore creates a new alert store
func NewStore() *Store {
	return &Store{
		alerts:      make(map[string]*models.Alert),
		symbolIndex: make(map[string][]string),
	}
}

// Create adds a new alert to the store
func (s *Store) Create(alert *models.Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if alert already exists
	if _, exists := s.alerts[alert.ID]; exists {
		return ErrAlertExists
	}

	// Add alert to main storage
	s.alerts[alert.ID] = alert

	// Add to symbol index
	s.addToSymbolIndex(alert.Symbol, alert.ID)

	return nil
}

// Get retrieves an alert by ID
func (s *Store) Get(id string) (*models.Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alert, exists := s.alerts[id]
	if !exists {
		return nil, ErrAlertNotFound
	}

	// Return a copy to prevent external modification
	alertCopy := *alert
	return &alertCopy, nil
}

// Update modifies an existing alert
func (s *Store) Update(id string, updates map[string]interface{}) (*models.Alert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert, exists := s.alerts[id]
	if !exists {
		return nil, ErrAlertNotFound
	}

	oldSymbol := alert.Symbol

	// Apply updates
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

	// Update symbol index if symbol changed
	if oldSymbol != alert.Symbol {
		s.removeFromSymbolIndex(oldSymbol, id)
		s.addToSymbolIndex(alert.Symbol, id)
	}

	// Return a copy
	alertCopy := *alert
	return &alertCopy, nil
}

// Delete removes an alert from the store
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert, exists := s.alerts[id]
	if !exists {
		return ErrAlertNotFound
	}

	// Remove from symbol index
	s.removeFromSymbolIndex(alert.Symbol, id)

	// Remove from main storage
	delete(s.alerts, id)

	return nil
}

// GetAll returns all alerts
func (s *Store) GetAll() []*models.Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alerts := make([]*models.Alert, 0, len(s.alerts))
	for _, alert := range s.alerts {
		// Return copies to prevent external modification
		alertCopy := *alert
		alerts = append(alerts, &alertCopy)
	}

	return alerts
}

// GetBySymbol returns all alerts for a specific symbol
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
			// Return copies to prevent external modification
			alertCopy := *alert
			alerts = append(alerts, &alertCopy)
		}
	}

	return alerts
}

// GetEnabledBySymbol returns all enabled alerts for a specific symbol
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
			// Return copies to prevent external modification
			alertCopy := *alert
			alerts = append(alerts, &alertCopy)
		}
	}

	return alerts
}

// Count returns the total number of alerts
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.alerts)
}

// CountBySymbol returns the number of alerts for a specific symbol
func (s *Store) CountBySymbol(symbol string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alertIDs, exists := s.symbolIndex[symbol]
	if !exists {
		return 0
	}

	return len(alertIDs)
}

// GetActiveSymbols returns all symbols that have at least one alert
func (s *Store) GetActiveSymbols() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	symbols := make([]string, 0, len(s.symbolIndex))
	for symbol := range s.symbolIndex {
		symbols = append(symbols, symbol)
	}

	return symbols
}

// MarkTriggered updates the last trigger time for an alert
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

// addToSymbolIndex adds an alert ID to the symbol index
func (s *Store) addToSymbolIndex(symbol, alertID string) {
	if alertIDs, exists := s.symbolIndex[symbol]; exists {
		// Check if alert ID already exists to avoid duplicates
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

// removeFromSymbolIndex removes an alert ID from the symbol index
func (s *Store) removeFromSymbolIndex(symbol, alertID string) {
	alertIDs, exists := s.symbolIndex[symbol]
	if !exists {
		return
	}

	// Find and remove the alert ID
	for i, id := range alertIDs {
		if id == alertID {
			s.symbolIndex[symbol] = append(alertIDs[:i], alertIDs[i+1:]...)
			break
		}
	}

	// Remove the symbol entry if no alerts remain
	if len(s.symbolIndex[symbol]) == 0 {
		delete(s.symbolIndex, symbol)
	}
}
