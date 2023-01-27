package utils

import (
	"encoding/hex"

	api "github.com/bolt-observer/agent/lightning"
	runes "github.com/bolt-observer/go-runes/runes"
	macaroon "gopkg.in/macaroon.v2"
)

// AuthenticatorType enum
type AuthenticatorType int

// AuthenticatorType values
const (
	Unknown AuthenticatorType = iota
	Macaroon
	Rune
)

// DetectAuthenticatorType detects what kind of authenticator is used
func DetectAuthenticatorType(str string) AuthenticatorType {
	// Eventually this could be more complicated
	if isMacaroon(str) {
		return Macaroon
	} else if isRune(str) {
		return Rune
	}

	return Unknown
}

// APITypeToAuthenticatorType returns what kind of authenticator a given API uses
func APITypeToAuthenticatorType(t *api.APIType) AuthenticatorType {
	if t == nil {
		return Unknown
	}

	if *t == api.LndGrpc || *t == api.LndRest {
		return Macaroon
	}

	if *t == api.ClnSocket {
		return Rune
	}

	return Unknown
}

func isMacaroon(str string) bool {
	macBytes, err := hex.DecodeString(str)
	if err != nil {
		return false
	}

	mac := &macaroon.Macaroon{}
	if err = mac.UnmarshalBinary(macBytes); err != nil {
		return false
	}

	return mac.Location() == "lnd" && mac.Version() == macaroon.V2
}

func isRune(str string) bool {
	r, err := runes.FromBase64(str)
	if err != nil {
		return false
	}

	if r.GetVersion() != 0 {
		return false
	}

	return true
}
