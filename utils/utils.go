package utils

import (
	"time"

	entities "github.com/bolt-observer/go_common/entities"
)

const (
	// Delimiter between entries
	Delimiter = ","
	// UserPassSeparator separates username from password (cannot use :)
	UserPassSeparator = "|"
	// IAMAuthFlag defines that IAM authentication should be used
	IAMAuthFlag = "$iam" // starts with $ so it's an invalid crypted password
)

// GetConstrained returns a constrained version of d (macaroon will be time constrained)
func GetConstrained(d *entities.Data, duration time.Duration) entities.Data {
	data := new(entities.Data)
	data.PubKey = d.PubKey
	data.CertificateBase64 = d.CertificateBase64
	data.Endpoint = d.Endpoint
	data.Tags = d.Tags
	data.ApiType = d.ApiType
	data.CertVerificationType = d.CertVerificationType

	mac, err := ConstraintMacaroon(d.MacaroonHex, duration)
	if err != nil {
		data.MacaroonHex = ""
	} else {
		data.MacaroonHex = mac
	}
	return *data
}
