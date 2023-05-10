package utils

import (
	"context"
	"sync"
)

// DummySecretsManager struct.
type DummySecretsManager struct {
	// Mutex is used for mutual exclusion
	Mutex sync.Mutex
	// Names contains all the secrets
	Names map[string]struct{}
}

// NewDummySecretsManager creates a new DummySecretsManager
func NewDummySecretsManager() *DummySecretsManager {
	return &DummySecretsManager{
		Names: make(map[string]struct{}),
	}
}

// InsertOrUpdateSecret - inserts or updates a secret
func (s *DummySecretsManager) InsertOrUpdateSecret(ctx context.Context, name, value string) (string, Change, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	change := Updated
	_, ok := s.Names[name]
	if !ok {
		s.Names[name] = struct{}{}
		change = Inserted
	}

	return name, change, nil
}

// DeleteSecret - deletes a secret
func (s *DummySecretsManager) DeleteSecret(ctx context.Context, name string) (string, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	delete(s.Names, name)
	return name, nil
}

// LoadSecrets - loads all secrets (used at startup)
func (s *DummySecretsManager) LoadSecrets(ctx context.Context, prefix string) map[string]string {
	return make(map[string]string)
}
