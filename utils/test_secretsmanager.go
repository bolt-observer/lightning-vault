package utils

import (
	"context"
)

// InsertOrUpdateSecretFn method
type InsertOrUpdateSecretFn func(ctx context.Context, name, value string) (string, Change, error)

// DeleteSecretFn method
type DeleteSecretFn func(ctx context.Context, name string) (string, error)

// LoadSecretsFn method
type LoadSecretsFn func(ctx context.Context, prefix string) map[string]string

// TestSecretsManager struct.
type TestSecretsManager struct {
	InsertOrUpdateSecretFn InsertOrUpdateSecretFn
	DeleteSecretFn         DeleteSecretFn
	LoadSecretsFn          LoadSecretsFn
	Dummy                  DummySecretsManager
}

// NewTestSecretsManager - create a new test secrets manager
func NewTestSecretsManager() *TestSecretsManager {
	return &TestSecretsManager{
		InsertOrUpdateSecretFn: nil,
		DeleteSecretFn:         nil,
		LoadSecretsFn:          nil,
		Dummy:                  *NewDummySecretsManager(),
	}
}

// InsertOrUpdateSecret - inserts or updates secret
func (s *TestSecretsManager) InsertOrUpdateSecret(ctx context.Context, name, value string) (string, Change, error) {
	if s.InsertOrUpdateSecretFn != nil {
		return s.InsertOrUpdateSecretFn(ctx, name, value)
	}

	return s.Dummy.InsertOrUpdateSecret(ctx, name, value)
}

// DeleteSecret - deletes a secret
func (s *TestSecretsManager) DeleteSecret(ctx context.Context, name string) (string, error) {
	if s.DeleteSecretFn != nil {
		return s.DeleteSecretFn(ctx, name)
	}

	return s.Dummy.DeleteSecret(ctx, name)
}

// LoadSecrets - loads secrets
func (s *TestSecretsManager) LoadSecrets(ctx context.Context, prefix string) map[string]string {
	if s.LoadSecretsFn != nil {
		return s.LoadSecretsFn(ctx, prefix)
	}

	return s.Dummy.LoadSecrets(ctx, prefix)
}
