package utils

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/lightningnetwork/lnd/macaroons"

	macaroon "gopkg.in/macaroon.v2"
)

// ConstraintMacaroon - adds a time constraint for duration since now on the macaroon
func ConstraintMacaroon(macHex string, duration time.Duration) (string, error) {
	if duration > time.Hour*24 {
		return "", fmt.Errorf("duration too long")
	}
	return constraintMacaroon(macHex, duration)
}

func constraintMacaroon(macHex string, duration time.Duration) (string, error) {
	macBytes, err := hex.DecodeString(macHex)
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
