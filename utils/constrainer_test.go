package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMacaroonConstrainer(t *testing.T) {
	mac := "0201036c6e640224030a10b493608461fb6e64810053fa31ef27991201301a0c0a04696e666f120472656164000216697061646472203139322e3136382e3139322e3136380000062072ea006233da839ce6e9f4721331a12041b228d36c0fdad552680f615766d2f4"

	_, err := Constrain(mac, 2*24*time.Hour, nil)
	assert.Error(t, err)

	constrainedMac, err := Constrain(mac, 2*time.Hour, nil)
	assert.NoError(t, err)
	assert.NotEqual(t, constrainedMac, mac, "contained macaroon should be different than original")

	constrainedMac2, err := Constrain(mac, 2*time.Hour, nil)
	assert.NoError(t, err)

	assert.NotEqual(t, constrainedMac, constrainedMac2, "contained macaroon should be different all the time you constraint it")
}

func TestRuneConstrainer(t *testing.T) {
	rune := "y3niiNN_cNeIP_SPeoxzXSQMZnqkieqvtABj37rH_UQ9MA=="

	_, err := Constrain(rune, 2*24*time.Hour, nil)
	assert.Error(t, err)

	constrainedRune, err := Constrain(rune, 2*time.Hour, nil)
	assert.NoError(t, err)
	assert.NotEqual(t, constrainedRune, rune, "contained macaroon should be different than original")
}
