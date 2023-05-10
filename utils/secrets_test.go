package utils

import (
	"context"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func TestPlatformSecretsManager(t *testing.T) {
	const Prefix = "unittest"
	provider := DetermineProvider()
	if provider == UnknownProvider {
		t.Log("Unknown cloud provider")
		return
	}

	ctx := context.Background()
	s := GetPlatformSecretsManager()

	all := s.LoadSecrets(ctx, Prefix)
	require.NotNil(t, all)
	if len(all) != 0 {
		for k := range all {
			t.Logf("Deleting old secret leftover %s", k)
			_, err := s.DeleteSecret(ctx, k)
			require.NoError(t, err)
		}
	}

	t.Log("Trying first secret")
	name := Prefix + RandSeq(10)
	_, ch, err := s.InsertOrUpdateSecret(ctx, name, "secret1")
	require.NoError(t, err)
	require.Equal(t, Inserted, ch)

	all = s.LoadSecrets(ctx, Prefix)
	require.NotNil(t, all)
	val, ok := all[name]
	require.Equal(t, true, ok)
	require.Equal(t, "secret1", val)
	_, ok = all[Prefix+"fake"]
	require.Equal(t, false, ok)

	t.Log("Trying second secret")
	_, ch, err = s.InsertOrUpdateSecret(ctx, name, "secret2")
	require.NoError(t, err)
	require.Equal(t, Updated, ch)

	all = s.LoadSecrets(ctx, Prefix)
	require.NotNil(t, all)
	val, ok = all[name]
	require.Equal(t, true, ok)
	require.Equal(t, "secret2", val)
	_, ok = all[Prefix+"fake"]
	require.Equal(t, false, ok)

	t.Log("Cleaning up")
	_, err = s.DeleteSecret(ctx, Prefix+"fake")
	require.Error(t, err)
	_, err = s.DeleteSecret(ctx, name)
	require.NoError(t, err)

	all = s.LoadSecrets(ctx, Prefix)
	require.NotNil(t, all)
	_, ok = all[name]
	require.Equal(t, false, ok)
}
