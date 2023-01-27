package utils

import (
	"encoding/hex"
	"fmt"
	"time"

	runes "github.com/bolt-observer/go-runes/runes"
	"github.com/golang/glog"
	"github.com/lightningnetwork/lnd/macaroons"
	macaroon "gopkg.in/macaroon.v2"
)

// ConstrainFunc is the method signature
type ConstrainFunc func(string, time.Duration) (string, error)

var (
	mapping = map[AuthenticatorType]ConstrainFunc{
		Macaroon: ConstrainFunc(macaroonConstrainer),
		Rune:     ConstrainFunc(runeConstrainer),
	}
)

// Constrain constrains a given authenticator
func Constrain(original string, duration time.Duration) (string, error) {
	if duration > time.Hour*24 {
		return "", fmt.Errorf("duration too long")
	}

	classification := DetectAuthenticatorType(original)
	if val, ok := mapping[classification]; ok {
		return val(original, duration)
	}
	return unknownConstrainer(original, duration)
}

func macaroonConstrainer(original string, duration time.Duration) (string, error) {
	macBytes, err := hex.DecodeString(original)
	if err != nil {
		glog.Errorf("Could not decode macaroon: %v", err)
		return "", err
	}

	mac := &macaroon.Macaroon{}
	if err = mac.UnmarshalBinary(macBytes); err != nil {
		glog.Errorf("Could not decode macaroon: %v", err)
		return "", err
	}

	macConstraints := []macaroons.Constraint{
		macaroons.TimeoutConstraint(int64(duration.Seconds())),
	}

	constrainedMac, err := macaroons.AddConstraints(mac, macConstraints...)
	if err != nil {
		glog.Errorf("Could not decode macaroon: %v", err)
		return "", err
	}
	result, err := constrainedMac.MarshalBinary()
	if err != nil {
		glog.Errorf("Could not decode macaroon: %v", err)
		return "", err
	}

	return hex.EncodeToString(result), nil
}

func runeConstrainer(original string, duration time.Duration) (string, error) {
	r, err := runes.FromBase64(original)
	if err != nil {
		return "", err
	}

	limit := time.Now().Add(duration).Unix()
	rest, _, err := runes.MakeRestrictionFromString(fmt.Sprintf("time<%d", limit), false)
	if err != nil {
		return "", err
	}

	result, err := r.GetRestricted(*rest)
	if err != nil {
		return "", err
	}

	return result.ToBase64(), nil
}

func unknownConstrainer(original string, duration time.Duration) (string, error) {
	glog.Warningf("Trying to constraint unknown authenticator for %v", duration) // do not log original on purpose since it might be sensitive
	return "", nil
}
