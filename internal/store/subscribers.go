package store

import (
	"encoding/json"
	"os"
	"sync"
)

type SubscriberStore struct {
	mu       sync.RWMutex
	filePath string
	emails   []string
}

func NewSubscriberStore(filePath string) (*SubscriberStore, error) {
	s := &SubscriberStore{
		filePath: filePath,
		emails:   []string{},
	}
	if err := s.load(); err != nil {
		if os.IsNotExist(err) {
			return s, nil // New file will be created on save
		}
		return nil, err
	}
	return s, nil
}

func (s *SubscriberStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.emails)
}

func (s *SubscriberStore) save() error {
	data, err := json.MarshalIndent(s.emails, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *SubscriberStore) Add(email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicates
	for _, e := range s.emails {
		if e == email {
			return nil // Already exists
		}
	}

	s.emails = append(s.emails, email)
	return s.save()
}

func (s *SubscriberStore) GetAll() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Return a copy
	result := make([]string, len(s.emails))
	copy(result, s.emails)
	return result
}
