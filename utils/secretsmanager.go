package utils

import (
	"context"
)

// Change enum
type Change int

// Change enum values
const (
	Undefined Change = iota
	Inserted
	Updated
)

// SecretsManager interface
type SecretsManager interface {
	InsertOrUpdateSecret(ctx context.Context, name, value string) (string, Change, error)
	DeleteSecret(ctx context.Context, name string) (string, error)
	LoadSecrets(ctx context.Context, prefix string) map[string]string
}

// GetPlatformSecretsManager - gets the implementation for the current platform
func GetPlatformSecretsManager() SecretsManager {
	switch DetermineProvider() {
	case AWS:
		return SecretsManager(NewAwsSecretsManager())
	case GCP:
		return SecretsManager(NewGcpSecretsManager())
	default:
		return SecretsManager(NewTestSecretsManager())
	}
}
