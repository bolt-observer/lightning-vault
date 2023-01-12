package utils

import (
	"math/rand"
	"reflect"
	"testing"
	"time"

	entities "github.com/bolt-observer/go_common/entities"
	"github.com/mitchellh/hashstructure/v2"
)

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func TestGetConstrained(t *testing.T) {
	data := &entities.Data{}

	rand.Seed(time.Now().UnixNano())

	// fill with junk using reflection
	n := reflect.ValueOf(data).Elem().NumField()
	for i := 0; i < n; i++ {
		field := reflect.ValueOf(data).Elem().Field(i)

		switch field.Kind() {
		case reflect.String:
			field.SetString(randomString(16))
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
			field.SetInt(int64(rand.Int()))
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			field.SetUint(uint64(rand.Int()))
		case reflect.Bool:
			val := rand.Int()
			b := val&0x01 == 1
			field.SetBool(b)
		case reflect.Ptr:
			pointee := reflect.TypeOf(data).Elem().Field(i).Type.Elem()
			x := reflect.New(pointee)

			switch pointee.Kind() {
			case reflect.String:
				x.Elem().SetString(randomString(16))
			case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
				x.Elem().SetInt(int64(rand.Int()))
			case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				x.Elem().SetUint(uint64(rand.Int()))
			case reflect.Bool:
				val := rand.Int()
				b := val&0x01 == 1
				x.Elem().SetBool(b)
			default:
				t.Fatalf("Unknown field type - nothing critical this is just to remind you to update TestGetConstrained")
			}

		default:
			t.Fatalf("Unknown field type - nothing critical this is just to remind you to update TestGetConstrained")
		}
	}

	// fill with meaningful values
	data.MacaroonHex = "0201036c6e640224030a10b493608461fb6e64810053fa31ef27991201301a0c0a04696e666f120472656164000216697061646472203139322e3136382e3139322e3136380000062072ea006233da839ce6e9f4721331a12041b228d36c0fdad552680f615766d2f4"
	data.PubKey = "0327f763c849bfd218910e41eef74f5a737989358ab3565f185e1a61bb7df445b8"
	data.CertificateBase64 = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNOVENDQWRxZ0F3SUJBZ0lSQU1PV2pQUWYyKzlobDVrWElNaFdIMUF3Q2dZSUtvWkl6ajBFQXdJd0xqRWYKTUIwR0ExVUVDaE1XYkc1a0lHRjFkRzluWlc1bGNtRjBaV1FnWTJWeWRERUxNQWtHQTFVRUF4TUNiRzR3SGhjTgpNakV3T1RJME1UY3pPVEF4V2hjTk1qSXhNVEU1TVRjek9UQXhXakF1TVI4d0hRWURWUVFLRXhac2JtUWdZWFYwCmIyZGxibVZ5WVhSbFpDQmpaWEowTVFzd0NRWURWUVFERXdKc2JqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDkKQXdFSEEwSUFCSEs1VStQYXpDU2JjTW1mMUJucTFDY2xteU1nOGMvUkhhcXNabnpJKzErNjJnR2wyNk4vM2ZrZwpGdzV2dUtUMS9zWEVLblhUSE9LYUNSL1lkaklVNUQ2amdkZ3dnZFV3RGdZRFZSMFBBUUgvQkFRREFnS2tNQk1HCkExVWRKUVFNTUFvR0NDc0dBUVVGQndNQk1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0hRWURWUjBPQkJZRUZIanEKQ3RnVFRZTmdDTU85OUZBS2dJSEF4NUlnTUg0R0ExVWRFUVIzTUhXQ0FteHVnZ2xzYjJOaGJHaHZjM1NDRFd4dQpMbVY0WldOMlpTNXZjbWVDQkhWdWFYaUNDblZ1YVhod1lXTnJaWFNDQjJKMVptTnZibTZIQkg4QUFBR0hFQUFBCkFBQUFBQUFBQUFBQUFBQUFBQUdIQk1Db1FCZUhCS3dSQUFHSEVQNkFBQUFBQUFBQTNxWXkvLzQ4SmFxSEJGblUKL2VZd0NnWUlLb1pJemowRUF3SURTUUF3UmdJaEFOVTlDWHYxajZQVk9oQzZMMFp1Y3Z1WnVtb0tjb1NnTTFLTgpYR1E3eUNSQUFpRUF3N0ZiZ05qSHUvQVFveDJONTl2alFRTjI0NzlwSTRQT0c1MUFDbWI2SlhjPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
	data.Endpoint = "1.3.3.7:1337"
	data.Tags = "a,b,c"

	one := 1
	data.ApiType = &one
	data.CertVerificationType = &one

	d2 := GetConstrained(data, time.Hour)
	if d2.MacaroonHex == data.MacaroonHex {
		t.Fatalf("Macaroon should not be the same")
		return
	}

	data.MacaroonHex = ""
	d2.MacaroonHex = ""

	hash1, err := hashstructure.Hash(data, hashstructure.FormatV2, nil)
	if err != nil {
		t.Fatalf("hash failed: %v", err)
		return
	}

	hash2, err := hashstructure.Hash(d2, hashstructure.FormatV2, nil)
	if err != nil {
		t.Fatalf("hash failed: %v", err)
		return
	}

	if hash1 != hash2 {
		t.Fatalf("Structures are not the same")
		return
	}
}
