package main

import (
	"encoding/json"
	"strings"
	"testing"

	entities "github.com/bolt-observer/go_common/entities"
)

func TestDeserialization(t *testing.T) {
	inputs := []string{
		`{
		"pubkey": "0327f763c849bfd218910e41eef74f5a737989358ab3565f185e1a61bb7df445b8",
		"macaroon_hex": "abcd",
		"certificate_base64": "",
		"endpoint": "127.0.0.1:10009"
	  }`,
		`{
		"pubkey": "0327f763c849bfd218910e41eef74f5a737989358ab3565f185e1a61bb7df445b8",
		"macaroon_hex": "abcd",
		"certificate_base64": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNOVENDQWRxZ0F3SUJBZ0lSQU1PV2pQUWYyKzlobDVrWElNaFdIMUF3Q2dZSUtvWkl6ajBFQXdJd0xqRWYKTUIwR0ExVUVDaE1XYkc1a0lHRjFkRzluWlc1bGNtRjBaV1FnWTJWeWRERUxNQWtHQTFVRUF4TUNiRzR3SGhjTgpNakV3T1RJME1UY3pPVEF4V2hjTk1qSXhNVEU1TVRjek9UQXhXakF1TVI4d0hRWURWUVFLRXhac2JtUWdZWFYwCmIyZGxibVZ5WVhSbFpDQmpaWEowTVFzd0NRWURWUVFERXdKc2JqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDkKQXdFSEEwSUFCSEs1VStQYXpDU2JjTW1mMUJucTFDY2xteU1nOGMvUkhhcXNabnpJKzErNjJnR2wyNk4vM2ZrZwpGdzV2dUtUMS9zWEVLblhUSE9LYUNSL1lkaklVNUQ2amdkZ3dnZFV3RGdZRFZSMFBBUUgvQkFRREFnS2tNQk1HCkExVWRKUVFNTUFvR0NDc0dBUVVGQndNQk1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0hRWURWUjBPQkJZRUZIanEKQ3RnVFRZTmdDTU85OUZBS2dJSEF4NUlnTUg0R0ExVWRFUVIzTUhXQ0FteHVnZ2xzYjJOaGJHaHZjM1NDRFd4dQpMbVY0WldOMlpTNXZjbWVDQkhWdWFYaUNDblZ1YVhod1lXTnJaWFNDQjJKMVptTnZibTZIQkg4QUFBR0hFQUFBCkFBQUFBQUFBQUFBQUFBQUFBQUdIQk1Db1FCZUhCS3dSQUFHSEVQNkFBQUFBQUFBQTNxWXkvLzQ4SmFxSEJGblUKL2VZd0NnWUlLb1pJemowRUF3SURTUUF3UmdJaEFOVTlDWHYxajZQVk9oQzZMMFp1Y3Z1WnVtb0tjb1NnTTFLTgpYR1E3eUNSQUFpRUF3N0ZiZ05qSHUvQVFveDJONTl2alFRTjI0NzlwSTRQT0c1MUFDbWI2SlhjPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
		"endpoint": "1.2.3.4:10009",
		"tags": "dummy,66"
	  }`,
		`{
		"pubkey": "0327f763c849bfd218910e41eef74f5a737989358ab3565f185e1a61bb7df445b8",
		"macaroon_hex": "abcd",
		"certificate_base64": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNOVENDQWRxZ0F3SUJBZ0lSQU1PV2pQUWYyKzlobDVrWElNaFdIMUF3Q2dZSUtvWkl6ajBFQXdJd0xqRWYKTUIwR0ExVUVDaE1XYkc1a0lHRjFkRzluWlc1bGNtRjBaV1FnWTJWeWRERUxNQWtHQTFVRUF4TUNiRzR3SGhjTgpNakV3T1RJME1UY3pPVEF4V2hjTk1qSXhNVEU1TVRjek9UQXhXakF1TVI4d0hRWURWUVFLRXhac2JtUWdZWFYwCmIyZGxibVZ5WVhSbFpDQmpaWEowTVFzd0NRWURWUVFERXdKc2JqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDkKQXdFSEEwSUFCSEs1VStQYXpDU2JjTW1mMUJucTFDY2xteU1nOGMvUkhhcXNabnpJKzErNjJnR2wyNk4vM2ZrZwpGdzV2dUtUMS9zWEVLblhUSE9LYUNSL1lkaklVNUQ2amdkZ3dnZFV3RGdZRFZSMFBBUUgvQkFRREFnS2tNQk1HCkExVWRKUVFNTUFvR0NDc0dBUVVGQndNQk1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0hRWURWUjBPQkJZRUZIanEKQ3RnVFRZTmdDTU85OUZBS2dJSEF4NUlnTUg0R0ExVWRFUVIzTUhXQ0FteHVnZ2xzYjJOaGJHaHZjM1NDRFd4dQpMbVY0WldOMlpTNXZjbWVDQkhWdWFYaUNDblZ1YVhod1lXTnJaWFNDQjJKMVptTnZibTZIQkg4QUFBR0hFQUFBCkFBQUFBQUFBQUFBQUFBQUFBQUdIQk1Db1FCZUhCS3dSQUFHSEVQNkFBQUFBQUFBQTNxWXkvLzQ4SmFxSEJGblUKL2VZd0NnWUlLb1pJemowRUF3SURTUUF3UmdJaEFOVTlDWHYxajZQVk9oQzZMMFp1Y3Z1WnVtb0tjb1NnTTFLTgpYR1E3eUNSQUFpRUF3N0ZiZ05qSHUvQVFveDJONTl2alFRTjI0NzlwSTRQT0c1MUFDbWI2SlhjPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
		"endpoint": "1.2.3.4:10009",
		"tags": "dummy,66",
		"api_type": 1
	  }`,
	}

	var data entities.Data

	for _, one := range inputs {
		decoder := json.NewDecoder(strings.NewReader(one))
		err := decoder.Decode(&data)
		if err != nil {
			t.Fatalf("deserialization failed: %v", err)
		}
	}
}
