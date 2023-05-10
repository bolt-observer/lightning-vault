package utils

import "testing"

func TestDetermineProvider(t *testing.T) {
	provider := DetermineProvider()
	t.Logf("Provider %v", provider)
}
