package store

import (
	"encoding/json"
	"os"
	"sync"
)

type FileStore struct {
	mu       sync.RWMutex
	filePath string
	emails   []string
}

func NewFileStore(filePath string) (*FileStore, error) {
	s := &FileStore{
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

func (s *FileStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.emails)
}

func (s *FileStore) save() error {
	data, err := json.MarshalIndent(s.emails, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *FileStore) Add(email string) error {
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

func (s *FileStore) Remove(email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, e := range s.emails {
		if e == email {
			// Remove element at index i
			s.emails = append(s.emails[:i], s.emails[i+1:]...)
			return s.save()
		}
	}
	return nil // Not found, treat as success
}

func (s *FileStore) GetAll() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Return a copy
	result := make([]string, len(s.emails))
	copy(result, s.emails)
	return result, nil
}
