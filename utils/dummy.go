package utils

import (
	"context"
	"sync"
)

var (
	// Mutex is used for mutual exclusion
	Mutex sync.Mutex
	// Names contains all the secrets
	Names = make(map[string]struct{})
)

// InsertSecretDummy - mock version of the InsertSecret method
func InsertSecretDummy(ctx context.Context, name, value string) (string, Change, error) {
	Mutex.Lock()
	defer Mutex.Unlock()

	change := Updated
	_, ok := Names[name]
	if !ok {
		Names[name] = struct{}{}
		change = Inserted
	}

	return name, change, nil
}

// InvalidateSecretDummy - mock version of the InvalidateSecret method
func InvalidateSecretDummy(ctx context.Context, name string) (string, error) {
	Mutex.Lock()
	defer Mutex.Unlock()

	delete(Names, name)
	return name, nil
}
