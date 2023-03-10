package utils

import (
	"testing"

	api "github.com/bolt-observer/agent/lightning"
	"github.com/stretchr/testify/assert"
)

func TestDetectAuthenticatorType(t *testing.T) {

	mac := "0201036c6e640224030a10b493608461fb6e64810053fa31ef27991201301a0c0a04696e666f120472656164000216697061646472203139322e3136382e3139322e3136380000062072ea006233da839ce6e9f4721331a12041b228d36c0fdad552680f615766d2f4"
	rune := "tU-RLjMiDpY2U0o3W1oFowar36RFGpWloPbW9-RuZdo9MyZpZD0wMjRiOWExZmE4ZTAwNmYxZTM5MzdmNjVmNjZjNDA4ZTZkYThlMWNhNzI4ZWE0MzIyMmE3MzgxZGYxY2M0NDk2MDUmbWV0aG9kPWxpc3RwZWVycyZwbnVtPTEmcG5hbWVpZF4wMjRiOWExZmE4ZTAwNmYxZTM5M3xwYXJyMF4wMjRiOWExZmE4ZTAwNmYxZTM5MyZ0aW1lPDE2NTY5MjA1MzgmcmF0ZT0y"

	assert.Equal(t, Macaroon, DetectAuthenticatorType(mac, nil))
	assert.Equal(t, Rune, DetectAuthenticatorType(rune, nil))
	assert.Equal(t, Unknown, DetectAuthenticatorType("", nil))
	assert.Equal(t, Unknown, DetectAuthenticatorType("burek", nil))
	assert.Equal(t, Unknown, DetectAuthenticatorType("0201036c6e640224030a10b4936084", nil))
	assert.Equal(t, Unknown, DetectAuthenticatorType(mac+"X", nil))

	assert.Equal(t, Unknown, DetectAuthenticatorType(rune+mac, nil))
	assert.Equal(t, Unknown, DetectAuthenticatorType(mac+rune, nil))
}

func TestToAuthenticatorType(t *testing.T) {
	assert.Equal(t, Macaroon, ToAuthenticatorType(api.LndGrpc))
	assert.Equal(t, Macaroon, ToAuthenticatorType(api.LndGrpc))
	assert.Equal(t, Rune, ToAuthenticatorType(api.ClnCommando))
}
