package utils

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"unicode"

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
func DetectAuthenticatorType(str string, whenMultipleMatch *api.APIType) AuthenticatorType {
	matches := 0
	result := Unknown

	if isRune(str) {
		result = Rune
		matches++
	}

	if isMacaroon(str) {
		result = Macaroon
		matches++
	}

	if matches > 1 && whenMultipleMatch != nil {
		defaultType := ToAuthenticatorType(*whenMultipleMatch)
		if defaultType != Unknown {
			result = defaultType
		}
	}

	return result
}

// ToAuthenticatorType returns what kind of authenticator a given API uses
func ToAuthenticatorType(t api.APIType) AuthenticatorType {
	if t == api.LndGrpc || t == api.LndRest {
		return Macaroon
	}

	if t == api.ClnCommando {
		return Rune
	}

	return Unknown
}

func isMacaroon(str string) bool {
	// hex.DecodeString could just parse part of it
	re := regexp.MustCompile(`[A-Fa-f0-9]+`)
	dummy := re.ReplaceAllString(str, "")
	if len(dummy) > 0 {
		return false
	}

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

	// Lib is quite forgiving but we are more strict, if value is not printable it's not a rune
	for _, rest := range r.Restrictions {
		for _, alt := range rest.Alternatives {
			s := fmt.Sprintf("%v", alt.Value)
			for _, rune := range s {
				if !unicode.IsPrint(rune) {
					return false
				}
			}
		}
	}

	return true
}
