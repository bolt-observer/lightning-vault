package utils

import (
	"testing"
	"time"
)

func TestConstraintMacaroon(t *testing.T) {

	_, err := ConstraintMacaroon("fff", time.Minute)
	if err == nil {
		t.Fatalf("wrong macaroon")
	}

	mac := "0201036c6e640224030a10b493608461fb6e64810053fa31ef27991201301a0c0a04696e666f120472656164000216697061646472203139322e3136382e3139322e3136380000062072ea006233da839ce6e9f4721331a12041b228d36c0fdad552680f615766d2f4"

	_, err = ConstraintMacaroon(mac, 2*24*time.Hour)
	if err == nil {
		t.Fatalf("macaroon valid for too long")
	}

	constrainedMac, err := ConstraintMacaroon(mac, 2*time.Hour)
	if err != nil {
		t.Fatalf("macaroon could not be constrained")
	}

	if constrainedMac == mac {
		t.Fatalf("contained macaroon should be different than original")
	}

	constrainedMac2, err := ConstraintMacaroon(mac, 2*time.Hour)
	if err != nil {
		t.Fatalf("macaroon could not be constrained")
	}

	if constrainedMac == constrainedMac2 {
		t.Fatalf("contained macaroon should be different all the time you constraint it")
	}
}
