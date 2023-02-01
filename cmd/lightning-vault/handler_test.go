package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	api "github.com/bolt-observer/agent/lightning"
	entities "github.com/bolt-observer/go_common/entities"
	local_utils "github.com/bolt-observer/lightning-vault/utils"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestMainHandler(t *testing.T) {

	r := httptest.NewRequest(http.MethodGet, "https://localhost", nil)
	w := httptest.NewRecorder()

	h := MakeNewHandlers()
	h.MainHandler(w, r)

	if want, got := http.StatusOK, w.Result().StatusCode; want != got {
		t.Fatalf("expected a %d, instead got %d", want, got)
	}

	data, err := io.ReadAll(w.Body)
	if err != nil {
		t.Fatalf("got error %v", err)
	}

	if string(data) != "Welcome to Lightning Vault!\n" {
		t.Fatalf("invalid response")
	}
}

func TestPutHandler(t *testing.T) {

	r := httptest.NewRequest(http.MethodPost, "https://localhost/put/", nil)
	w := httptest.NewRecorder()

	h := MakeNewHandlers()

	h.VerifyCall = func(w http.ResponseWriter, r *http.Request, data *entities.Data, pubkey, uniqueID string) bool {
		return true
	}

	prometheusInit()
	h.PutHandler(w, r)

	if want, got := http.StatusBadRequest, w.Result().StatusCode; want != got {
		t.Fatalf("expected a %d, instead got %d", want, got)
	}

	// Don't worry macaroon is fake
	valid := `{
		"pubkey": "0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7",
		"macaroon_hex": "0201036c6e640224030a10b493608461fb6e64810053fa31ef27991201301a0c0a04696e666f120472656164000216697061646472203139322e3136382e3139322e3136380000062072ea006233da839ce6e9f4721331a12041b228d36c0fdad552680f615766d2f4",
		"certificate_base64": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNKakNDQWN5Z0F3SUJBZ0lRUmU4QzhCcURubEF3b0VxRjdMRTVGREFLQmdncWhrak9QUVFEQWpBeE1SOHcKSFFZRFZRUUtFeFpzYm1RZ1lYVjBiMmRsYm1WeVlYUmxaQ0JqWlhKME1RNHdEQVlEVlFRREV3VmhiR2xqWlRBZQpGdzB5TXpBeE1ESXhOVE0xTXpsYUZ3MHlOREF5TWpjeE5UTTFNemxhTURFeEh6QWRCZ05WQkFvVEZteHVaQ0JoCmRYUnZaMlZ1WlhKaGRHVmtJR05sY25ReERqQU1CZ05WQkFNVEJXRnNhV05sTUZrd0V3WUhLb1pJemowQ0FRWUkKS29aSXpqMERBUWNEUWdBRXlKaHRYWk1NT0NQYzYxWmlISmVyKzdHUm9HalFzcWtNcjdvQVVjNnZsZC9JNDl2SwpHR01mRjhMcDhTSm1jNlJVOHQxN3FEZFhyUmZMbTdLSjB0eDBkcU9CeFRDQndqQU9CZ05WSFE4QkFmOEVCQU1DCkFxUXdFd1lEVlIwbEJBd3dDZ1lJS3dZQkJRVUhBd0V3RHdZRFZSMFRBUUgvQkFVd0F3RUIvekFkQmdOVkhRNEUKRmdRVU5BUW5BYVBNOStrZEpxMXdud2FtbldpY1d1SXdhd1lEVlIwUkJHUXdZb0lGWVd4cFkyV0NDV3h2WTJGcwphRzl6ZElJRllXeHBZMldDRG5CdmJHRnlMVzQyTFdGc2FXTmxnZ1IxYm1sNGdncDFibWw0Y0dGamEyVjBnZ2RpCmRXWmpiMjV1aHdSL0FBQUJoeEFBQUFBQUFBQUFBQUFBQUFBQUFBQUJod1NzR0FBQ01Bb0dDQ3FHU000OUJBTUMKQTBnQU1FVUNJUUQ2dElDMVdTWFRWNkpuSzVlN3FkdDRBVHp2Q0ZHUldPTmp2T29tUUdScXB3SWdiR1ZJWFVPbgpHamlUdTZ5MXVMT1pRS0VPTnB1MXZkYUNKejVpanNRdlVndz0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=",
		"endpoint": "192.168.192.168:10009"
	  }
	`
	r = httptest.NewRequest(http.MethodPost, "https://localhost/put", strings.NewReader(valid))
	w = httptest.NewRecorder()

	wasCalled := false
	h.AddCall = func(ctx context.Context, name, value string) (string, local_utils.Change, error) {
		wasCalled = true
		return "", local_utils.Inserted, nil
	}

	h.PutHandler(w, r)

	if !wasCalled {
		t.Fatalf("save was not called")
	}

	if want, got := http.StatusBadRequest, w.Result().StatusCode; want == got {
		t.Fatalf("expected not %d, instead got %d", want, got)
	}
}

func TestPutHandlerWithNoApiType(t *testing.T) {
	pubkey := "0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7"

	h := MakeNewHandlers()

	h.VerifyCall = func(w http.ResponseWriter, r *http.Request, data *entities.Data, pubkey, uniqueID string) bool {
		return true
	}

	prometheusInit()

	// Don't worry macaroon is fake
	valid := fmt.Sprintf(`{
		"pubkey": "%s",
		"macaroon_hex": "0201036c6e640224030a10b493608461fb6e64810053fa31ef27991201301a0c0a04696e666f120472656164000216697061646472203139322e3136382e3139322e3136380000062072ea006233da839ce6e9f4721331a12041b228d36c0fdad552680f615766d2f4",
		"certificate_base64": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNKakNDQWN5Z0F3SUJBZ0lRUmU4QzhCcURubEF3b0VxRjdMRTVGREFLQmdncWhrak9QUVFEQWpBeE1SOHcKSFFZRFZRUUtFeFpzYm1RZ1lYVjBiMmRsYm1WeVlYUmxaQ0JqWlhKME1RNHdEQVlEVlFRREV3VmhiR2xqWlRBZQpGdzB5TXpBeE1ESXhOVE0xTXpsYUZ3MHlOREF5TWpjeE5UTTFNemxhTURFeEh6QWRCZ05WQkFvVEZteHVaQ0JoCmRYUnZaMlZ1WlhKaGRHVmtJR05sY25ReERqQU1CZ05WQkFNVEJXRnNhV05sTUZrd0V3WUhLb1pJemowQ0FRWUkKS29aSXpqMERBUWNEUWdBRXlKaHRYWk1NT0NQYzYxWmlISmVyKzdHUm9HalFzcWtNcjdvQVVjNnZsZC9JNDl2SwpHR01mRjhMcDhTSm1jNlJVOHQxN3FEZFhyUmZMbTdLSjB0eDBkcU9CeFRDQndqQU9CZ05WSFE4QkFmOEVCQU1DCkFxUXdFd1lEVlIwbEJBd3dDZ1lJS3dZQkJRVUhBd0V3RHdZRFZSMFRBUUgvQkFVd0F3RUIvekFkQmdOVkhRNEUKRmdRVU5BUW5BYVBNOStrZEpxMXdud2FtbldpY1d1SXdhd1lEVlIwUkJHUXdZb0lGWVd4cFkyV0NDV3h2WTJGcwphRzl6ZElJRllXeHBZMldDRG5CdmJHRnlMVzQyTFdGc2FXTmxnZ1IxYm1sNGdncDFibWw0Y0dGamEyVjBnZ2RpCmRXWmpiMjV1aHdSL0FBQUJoeEFBQUFBQUFBQUFBQUFBQUFBQUFBQUJod1NzR0FBQ01Bb0dDQ3FHU000OUJBTUMKQTBnQU1FVUNJUUQ2dElDMVdTWFRWNkpuSzVlN3FkdDRBVHp2Q0ZHUldPTmp2T29tUUdScXB3SWdiR1ZJWFVPbgpHamlUdTZ5MXVMT1pRS0VPTnB1MXZkYUNKejVpanNRdlVndz0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=",
		"endpoint": "192.168.192.168:10009",
		"api_type": null
	  }
	`, pubkey)

	r := httptest.NewRequest(http.MethodPost, "https://localhost/put", strings.NewReader(valid))
	w := httptest.NewRecorder()

	h.AddCall = func(ctx context.Context, name, value string) (string, local_utils.Change, error) {
		return "", local_utils.Inserted, nil
	}

	h.PutHandler(w, r)

	if w.Result().StatusCode != http.StatusOK && w.Result().StatusCode != http.StatusCreated {
		t.Fatalf("expected success got %d", w.Result().StatusCode)
	}

	// Update cache to simulate older type of entity
	temp := h.Lookup[pubkey]
	temp.ApiType = nil
	h.Lookup[pubkey] = temp

	r = httptest.NewRequest(http.MethodPost, "https://localhost/put", strings.NewReader(valid))
	w = httptest.NewRecorder()
	h.PutHandler(w, r)

	if w.Result().StatusCode != http.StatusOK && w.Result().StatusCode != http.StatusCreated {
		t.Fatalf("expected success got %d", w.Result().StatusCode)
	}
}

func TestPutHandlerInvalidAuthenticator(t *testing.T) {

	r := httptest.NewRequest(http.MethodPost, "https://localhost/put/", nil)
	w := httptest.NewRecorder()

	h := MakeNewHandlers()

	h.VerifyCall = func(w http.ResponseWriter, r *http.Request, data *entities.Data, pubkey, uniqueID string) bool {
		return true
	}

	prometheusInit()
	h.PutHandler(w, r)

	if want, got := http.StatusBadRequest, w.Result().StatusCode; want != got {
		t.Fatalf("expected a %d, instead got %d", want, got)
	}

	// Don't worry macaroon is fake
	valid := `{
		"pubkey": "0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7",
		"macaroon_hex": "tU-RLjMiDpY2U0o3W1oFowar36RFGpWloPbW9-RuZdo9MyZpZD0wMjRiOWExZmE4ZTAwNmYxZTM5MzdmNjVmNjZjNDA4ZTZkYThlMWNhNzI4ZWE0MzIyMmE3MzgxZGYxY2M0NDk2MDUmbWV0aG9kPWxpc3RwZWVycyZwbnVtPTEmcG5hbWVpZF4wMjRiOWExZmE4ZTAwNmYxZTM5M3xwYXJyMF4wMjRiOWExZmE4ZTAwNmYxZTM5MyZ0aW1lPDE2NTY5MjA1MzgmcmF0ZT0y",
		"certificate_base64": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNKakNDQWN5Z0F3SUJBZ0lRUmU4QzhCcURubEF3b0VxRjdMRTVGREFLQmdncWhrak9QUVFEQWpBeE1SOHcKSFFZRFZRUUtFeFpzYm1RZ1lYVjBiMmRsYm1WeVlYUmxaQ0JqWlhKME1RNHdEQVlEVlFRREV3VmhiR2xqWlRBZQpGdzB5TXpBeE1ESXhOVE0xTXpsYUZ3MHlOREF5TWpjeE5UTTFNemxhTURFeEh6QWRCZ05WQkFvVEZteHVaQ0JoCmRYUnZaMlZ1WlhKaGRHVmtJR05sY25ReERqQU1CZ05WQkFNVEJXRnNhV05sTUZrd0V3WUhLb1pJemowQ0FRWUkKS29aSXpqMERBUWNEUWdBRXlKaHRYWk1NT0NQYzYxWmlISmVyKzdHUm9HalFzcWtNcjdvQVVjNnZsZC9JNDl2SwpHR01mRjhMcDhTSm1jNlJVOHQxN3FEZFhyUmZMbTdLSjB0eDBkcU9CeFRDQndqQU9CZ05WSFE4QkFmOEVCQU1DCkFxUXdFd1lEVlIwbEJBd3dDZ1lJS3dZQkJRVUhBd0V3RHdZRFZSMFRBUUgvQkFVd0F3RUIvekFkQmdOVkhRNEUKRmdRVU5BUW5BYVBNOStrZEpxMXdud2FtbldpY1d1SXdhd1lEVlIwUkJHUXdZb0lGWVd4cFkyV0NDV3h2WTJGcwphRzl6ZElJRllXeHBZMldDRG5CdmJHRnlMVzQyTFdGc2FXTmxnZ1IxYm1sNGdncDFibWw0Y0dGamEyVjBnZ2RpCmRXWmpiMjV1aHdSL0FBQUJoeEFBQUFBQUFBQUFBQUFBQUFBQUFBQUJod1NzR0FBQ01Bb0dDQ3FHU000OUJBTUMKQTBnQU1FVUNJUUQ2dElDMVdTWFRWNkpuSzVlN3FkdDRBVHp2Q0ZHUldPTmp2T29tUUdScXB3SWdiR1ZJWFVPbgpHamlUdTZ5MXVMT1pRS0VPTnB1MXZkYUNKejVpanNRdlVndz0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=",
		"endpoint": "192.168.192.168:10009",
		"api_type": 0
	  }
	`
	r = httptest.NewRequest(http.MethodPost, "https://localhost/put", strings.NewReader(valid))
	w = httptest.NewRecorder()

	h.AddCall = func(ctx context.Context, name, value string) (string, local_utils.Change, error) {
		return "", local_utils.Inserted, nil
	}

	h.PutHandler(w, r)

	if want, got := http.StatusBadRequest, w.Result().StatusCode; want != got {
		t.Fatalf("expected not %d, instead got %d", want, got)
	}
}

func TestUniqueId(t *testing.T) {
	// Don't worry macaroon is fake
	valid := `{
		"pubkey": "0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7",
		"macaroon_hex": "0201036c6e640224030a10b493608461fb6e64810053fa31ef27991201301a0c0a04696e666f120472656164000216697061646472203139322e3136382e3139322e3136380000062072ea006233da839ce6e9f4721331a12041b228d36c0fdad552680f615766d2f4",
		"certificate_base64": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNKakNDQWN5Z0F3SUJBZ0lRUmU4QzhCcURubEF3b0VxRjdMRTVGREFLQmdncWhrak9QUVFEQWpBeE1SOHcKSFFZRFZRUUtFeFpzYm1RZ1lYVjBiMmRsYm1WeVlYUmxaQ0JqWlhKME1RNHdEQVlEVlFRREV3VmhiR2xqWlRBZQpGdzB5TXpBeE1ESXhOVE0xTXpsYUZ3MHlOREF5TWpjeE5UTTFNemxhTURFeEh6QWRCZ05WQkFvVEZteHVaQ0JoCmRYUnZaMlZ1WlhKaGRHVmtJR05sY25ReERqQU1CZ05WQkFNVEJXRnNhV05sTUZrd0V3WUhLb1pJemowQ0FRWUkKS29aSXpqMERBUWNEUWdBRXlKaHRYWk1NT0NQYzYxWmlISmVyKzdHUm9HalFzcWtNcjdvQVVjNnZsZC9JNDl2SwpHR01mRjhMcDhTSm1jNlJVOHQxN3FEZFhyUmZMbTdLSjB0eDBkcU9CeFRDQndqQU9CZ05WSFE4QkFmOEVCQU1DCkFxUXdFd1lEVlIwbEJBd3dDZ1lJS3dZQkJRVUhBd0V3RHdZRFZSMFRBUUgvQkFVd0F3RUIvekFkQmdOVkhRNEUKRmdRVU5BUW5BYVBNOStrZEpxMXdud2FtbldpY1d1SXdhd1lEVlIwUkJHUXdZb0lGWVd4cFkyV0NDV3h2WTJGcwphRzl6ZElJRllXeHBZMldDRG5CdmJHRnlMVzQyTFdGc2FXTmxnZ1IxYm1sNGdncDFibWw0Y0dGamEyVjBnZ2RpCmRXWmpiMjV1aHdSL0FBQUJoeEFBQUFBQUFBQUFBQUFBQUFBQUFBQUJod1NzR0FBQ01Bb0dDQ3FHU000OUJBTUMKQTBnQU1FVUNJUUQ2dElDMVdTWFRWNkpuSzVlN3FkdDRBVHp2Q0ZHUldPTmp2T29tUUdScXB3SWdiR1ZJWFVPbgpHamlUdTZ5MXVMT1pRS0VPTnB1MXZkYUNKejVpanNRdlVndz0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=",
		"endpoint": "1.2.3.4:10009",
		"tags": "test"
	  }
	`
	h := MakeNewHandlers()

	h.AddCall = func(ctx context.Context, name, value string) (string, local_utils.Change, error) {
		return "", local_utils.Inserted, nil
	}
	h.VerifyCall = func(w http.ResponseWriter, r *http.Request, data *entities.Data, pubkey, uniqueId string) bool {
		return true
	}

	prometheusInit()

	r := httptest.NewRequest(http.MethodPost, "https://localhost/put/id1", strings.NewReader(valid))
	r = mux.SetURLVars(r, map[string]string{"uniqueId": "id1"})
	w := httptest.NewRecorder()

	h.PutHandler(w, r)

	if want, got := http.StatusBadRequest, w.Result().StatusCode; want == got {
		t.Fatalf("expected not %d, instead got %d", want, got)
	}

	expectNotRead("0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7", "", h, t)
	expectNotRead("test", "", h, t)
	expectNotRead("0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7", "id2", h, t)
	expectNotRead("test", "id2", h, t)

	expectRead("0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7", "0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7", "id1", h, t)
	expectRead("test", "0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7", "id1", h, t)

	valid2 := `{
		"pubkey": "0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7",
		"macaroon_hex": "0201036c6e640224030a10b493608461fb6e64810053fa31ef27991201301a0c0a04696e666f120472656164000216697061646472203139322e3136382e3139322e3136380000062072ea006233da839ce6e9f4721331a12041b228d36c0fdad552680f615766d2f4",
		"certificate_base64": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNOVENDQWRxZ0F3SUJBZ0lSQU1PV2pQUWYyKzlobDVrWElNaFdIMUF3Q2dZSUtvWkl6ajBFQXdJd0xqRWYKTUIwR0ExVUVDaE1XYkc1a0lHRjFkRzluWlc1bGNtRjBaV1FnWTJWeWRERUxNQWtHQTFVRUF4TUNiRzR3SGhjTgpNakV3T1RJME1UY3pPVEF4V2hjTk1qSXhNVEU1TVRjek9UQXhXakF1TVI4d0hRWURWUVFLRXhac2JtUWdZWFYwCmIyZGxibVZ5WVhSbFpDQmpaWEowTVFzd0NRWURWUVFERXdKc2JqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDkKQXdFSEEwSUFCSEs1VStQYXpDU2JjTW1mMUJucTFDY2xteU1nOGMvUkhhcXNabnpJKzErNjJnR2wyNk4vM2ZrZwpGdzV2dUtUMS9zWEVLblhUSE9LYUNSL1lkaklVNUQ2amdkZ3dnZFV3RGdZRFZSMFBBUUgvQkFRREFnS2tNQk1HCkExVWRKUVFNTUFvR0NDc0dBUVVGQndNQk1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0hRWURWUjBPQkJZRUZIanEKQ3RnVFRZTmdDTU85OUZBS2dJSEF4NUlnTUg0R0ExVWRFUVIzTUhXQ0FteHVnZ2xzYjJOaGJHaHZjM1NDRFd4dQpMbVY0WldOMlpTNXZjbWVDQkhWdWFYaUNDblZ1YVhod1lXTnJaWFNDQjJKMVptTnZibTZIQkg4QUFBR0hFQUFBCkFBQUFBQUFBQUFBQUFBQUFBQUdIQk1Db1FCZUhCS3dSQUFHSEVQNkFBQUFBQUFBQTNxWXkvLzQ4SmFxSEJGblUKL2VZd0NnWUlLb1pJemowRUF3SURTUUF3UmdJaEFOVTlDWHYxajZQVk9oQzZMMFp1Y3Z1WnVtb0tjb1NnTTFLTgpYR1E3eUNSQUFpRUF3N0ZiZ05qSHUvQVFveDJONTl2alFRTjI0NzlwSTRQT0c1MUFDbWI2SlhjPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
		"endpoint": "4.5.6.7:10009",
		"tags": "test"
	  }
	`

	r = httptest.NewRequest(http.MethodPost, "https://localhost/put/id2", strings.NewReader(valid2))
	r = mux.SetURLVars(r, map[string]string{"uniqueId": "id2"})
	w = httptest.NewRecorder()

	h.PutHandler(w, r)

	if want, got := http.StatusBadRequest, w.Result().StatusCode; want == got {
		t.Fatalf("expected not %d, instead got %d", want, got)
	}

	d1 := read("0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7", "id1", h, t)
	d2 := read("0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7", "id2", h, t)

	if d1.Endpoint == d2.Endpoint {
		t.Fatalf("wrong result was returned")
	}

	if d1.PubKey != d2.PubKey {
		t.Fatalf("wrong result was returned")
	}
}

func TestReadMyWriteMacaroon(t *testing.T) {
	pubKey := "0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7"
	origMac := "0201036c6e640224030a10b493608461fb6e64810053fa31ef27991201301a0c0a04696e666f120472656164000216697061646472203139322e3136382e3139322e3136380000062072ea006233da839ce6e9f4721331a12041b228d36c0fdad552680f615766d2f4"

	readMyWrite(t, pubKey, origMac)
}

func TestReadMyWriteRune(t *testing.T) {
	pubKey := "0367fa307a6e0ce29efadc4f7c4d1109ee689aa1e7bd442afd7270919f9e28c3b7"
	origMac := "tU-RLjMiDpY2U0o3W1oFowar36RFGpWloPbW9-RuZdo9MyZpZD0wMjRiOWExZmE4ZTAwNmYxZTM5MzdmNjVmNjZjNDA4ZTZkYThlMWNhNzI4ZWE0MzIyMmE3MzgxZGYxY2M0NDk2MDUmbWV0aG9kPWxpc3RwZWVycyZwbnVtPTEmcG5hbWVpZF4wMjRiOWExZmE4ZTAwNmYxZTM5M3xwYXJyMF4wMjRiOWExZmE4ZTAwNmYxZTM5MyZ0aW1lPDE2NTY5MjA1MzgmcmF0ZT0y"

	readMyWrite(t, pubKey, origMac)
}

func readMyWrite(t *testing.T, pubKey, origMac string) {
	h := MakeNewHandlers()
	prometheusInit()

	// Don't worry macaroon is fake
	valid := fmt.Sprintf(`{
		"pubkey": "%s",
		"macaroon_hex": "%s",
		"certificate_base64": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNKakNDQWN5Z0F3SUJBZ0lRUmU4QzhCcURubEF3b0VxRjdMRTVGREFLQmdncWhrak9QUVFEQWpBeE1SOHcKSFFZRFZRUUtFeFpzYm1RZ1lYVjBiMmRsYm1WeVlYUmxaQ0JqWlhKME1RNHdEQVlEVlFRREV3VmhiR2xqWlRBZQpGdzB5TXpBeE1ESXhOVE0xTXpsYUZ3MHlOREF5TWpjeE5UTTFNemxhTURFeEh6QWRCZ05WQkFvVEZteHVaQ0JoCmRYUnZaMlZ1WlhKaGRHVmtJR05sY25ReERqQU1CZ05WQkFNVEJXRnNhV05sTUZrd0V3WUhLb1pJemowQ0FRWUkKS29aSXpqMERBUWNEUWdBRXlKaHRYWk1NT0NQYzYxWmlISmVyKzdHUm9HalFzcWtNcjdvQVVjNnZsZC9JNDl2SwpHR01mRjhMcDhTSm1jNlJVOHQxN3FEZFhyUmZMbTdLSjB0eDBkcU9CeFRDQndqQU9CZ05WSFE4QkFmOEVCQU1DCkFxUXdFd1lEVlIwbEJBd3dDZ1lJS3dZQkJRVUhBd0V3RHdZRFZSMFRBUUgvQkFVd0F3RUIvekFkQmdOVkhRNEUKRmdRVU5BUW5BYVBNOStrZEpxMXdud2FtbldpY1d1SXdhd1lEVlIwUkJHUXdZb0lGWVd4cFkyV0NDV3h2WTJGcwphRzl6ZElJRllXeHBZMldDRG5CdmJHRnlMVzQyTFdGc2FXTmxnZ1IxYm1sNGdncDFibWw0Y0dGamEyVjBnZ2RpCmRXWmpiMjV1aHdSL0FBQUJoeEFBQUFBQUFBQUFBQUFBQUFBQUFBQUJod1NzR0FBQ01Bb0dDQ3FHU000OUJBTUMKQTBnQU1FVUNJUUQ2dElDMVdTWFRWNkpuSzVlN3FkdDRBVHp2Q0ZHUldPTmp2T29tUUdScXB3SWdiR1ZJWFVPbgpHamlUdTZ5MXVMT1pRS0VPTnB1MXZkYUNKejVpanNRdlVndz0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=",
		"endpoint": "[::1]:10009",
		"tags": "some,test"
	  }
	`, pubKey, origMac)

	r := httptest.NewRequest(http.MethodPost, "https://localhost/put", strings.NewReader(valid))
	w := httptest.NewRecorder()

	saveCalled := false
	h.AddCall = func(ctx context.Context, name, value string) (string, local_utils.Change, error) {
		saveCalled = true
		return "", local_utils.Inserted, nil
	}
	h.VerifyCall = func(w http.ResponseWriter, r *http.Request, data *entities.Data, pubkey, uniqueID string) bool {
		return true
	}

	deleteCalled := false

	h.DeleteCall = func(ctx context.Context, name string) (string, error) {
		deleteCalled = true
		return "", nil
	}

	h.PutHandler(w, r)

	if !saveCalled {
		t.Fatalf("save was not called")
	}

	if want, got := http.StatusBadRequest, w.Result().StatusCode; want == got {
		t.Fatalf("expected not %d, instead got %d", want, got)
	}

	mac1 := expectRead(pubKey, pubKey, "", h, t)
	expectQuery(pubKey, "", h, t, http.StatusOK)
	time.Sleep(1 * time.Second)
	mac2 := expectRead("test", pubKey, "", h, t)
	expectQuery("test", "", h, t, http.StatusOK)

	if mac1 == origMac {
		t.Fatalf("expected different macaroons %s and %s", mac1, origMac)
	}

	if mac1 == mac2 {
		t.Fatalf("expected different macaroons %s and %s", mac1, mac2)
	}

	expectNotRead("nonexisting", "", h, t)
	expectQuery("nonexisting", "", h, t, http.StatusNotFound)

	// Delete

	r = httptest.NewRequest(http.MethodDelete, "https://localhost/put/id1", nil)
	w = httptest.NewRecorder()
	r = mux.SetURLVars(r, map[string]string{"pubkey": pubKey})

	h.DeleteHandler(w, r)

	if want, got := http.StatusOK, w.Result().StatusCode; want != got {
		t.Fatalf("expected not %d, instead got %d", want, got)
	}

	if !deleteCalled {
		t.Fatalf("delete was not called")
	}

	expectNotRead(pubKey, "", h, t)
}

func TestAuth(t *testing.T) {
	router := mux.NewRouter()
	prometheusInit()

	router.Use(authMiddleware(map[string]string{"user1": "pass1", "user2": "$2a$10$m.Wdkic9j5eOO0L9w49Zo.1HrSDglSc6M1QcaZO5egLs2teohd9Wi"}))
	router.Path("/").Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	auth("user1", "pass1", http.StatusOK, router, t)
	auth("user2", "pass2", http.StatusOK, router, t)
	auth("user3", "pass3", http.StatusUnauthorized, router, t)
}

func auth(user, pass string, status int, router *mux.Router, t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "https://localhost/", nil)
	w := httptest.NewRecorder()
	r.SetBasicAuth(user, pass)

	router.ServeHTTP(w, r)

	if want, got := status, w.Result().StatusCode; want != got {
		t.Fatalf("expected %d, instead got %d", want, got)
	}
}

func read(term, uniqueID string, h *Handlers, t *testing.T) entities.Data {
	var data entities.Data

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost/get/%s", term), nil)
	if uniqueID != "" {
		r = mux.SetURLVars(r, map[string]string{"pubkey": term, "uniqueId": uniqueID})
	} else {
		r = mux.SetURLVars(r, map[string]string{"pubkey": term})
	}
	w := httptest.NewRecorder()

	h.GetHandler(w, r)

	if want, got := http.StatusOK, w.Result().StatusCode; want != got {
		t.Fatalf("expected %d, instead got %d", want, got)
	}

	decoder := json.NewDecoder(w.Body)
	err := decoder.Decode(&data)
	if err != nil {
		t.Fatalf("deserialization failed: %v", err)
	}

	return data
}

func expectRead(term, pubKey, uniqueID string, h *Handlers, t *testing.T) string {
	var data entities.Data

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost/get/%s", term), nil)
	if uniqueID != "" {
		r = mux.SetURLVars(r, map[string]string{"pubkey": term, "uniqueId": uniqueID})
	} else {
		r = mux.SetURLVars(r, map[string]string{"pubkey": term})
	}
	w := httptest.NewRecorder()

	h.GetHandler(w, r)

	if want, got := http.StatusOK, w.Result().StatusCode; want != got {
		t.Fatalf("expected %d, instead got %d", want, got)
	}

	decoder := json.NewDecoder(w.Body)
	err := decoder.Decode(&data)
	if err != nil {
		t.Fatalf("deserialization failed: %v", err)
	}

	if data.PubKey != pubKey {
		t.Fatalf("wrong entity data: %+v", data)
	}

	return data.MacaroonHex
}

func expectNotRead(term, uniqueID string, h *Handlers, t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost/get/%s", term), nil)

	if uniqueID != "" {
		r = mux.SetURLVars(r, map[string]string{"pubkey": term, "uniqueId": uniqueID})
	} else {
		r = mux.SetURLVars(r, map[string]string{"pubkey": term})
	}

	w := httptest.NewRecorder()

	h.GetHandler(w, r)

	if want, got := http.StatusNotFound, w.Result().StatusCode; want != got {
		t.Fatalf("expected %d, instead got %d", want, got)
	}
}

func expectQuery(term, uniqueID string, h *Handlers, t *testing.T, expected int) {
	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost/query/%s", term), nil)
	r = mux.SetURLVars(r, map[string]string{"pubkey": term})
	if uniqueID != "" {
		r = mux.SetURLVars(r, map[string]string{"pubkey": term, "uniqueId": uniqueID})
	} else {
		r = mux.SetURLVars(r, map[string]string{"pubkey": term})
	}
	w := httptest.NewRecorder()

	h.QueryHandler(w, r)

	if want, got := expected, w.Result().StatusCode; want != got {
		t.Fatalf("expected %d, instead got %d", want, got)
	}
}

func TestAutoDetectAPIType(t *testing.T) {
	data := &entities.Data{
		Endpoint:    "http://bolt.observer",
		MacaroonHex: "",
	}

	autoDetectAPIType(data)
	assert.NotNil(t, data.ApiType)
	assert.Equal(t, int(api.LndRest), *data.ApiType)
	assert.Equal(t, false, strings.HasSuffix("http", data.Endpoint))

	data.ApiType = nil
	data.Endpoint = "https://bolt.observer:1234"
	autoDetectAPIType(data)
	assert.NotNil(t, data.ApiType)
	assert.Equal(t, int(api.LndRest), *data.ApiType)
	assert.Equal(t, false, strings.HasSuffix("http", data.Endpoint))

	data.ApiType = nil
	data.Endpoint = "bolt.observer:10009"
	autoDetectAPIType(data)
	assert.NotNil(t, data.ApiType)
	assert.Equal(t, int(api.LndGrpc), *data.ApiType)

	data.ApiType = nil
	data.MacaroonHex = "tU-RLjMiDpY2U0o3W1oFowar36RFGpWloPbW9-RuZdo9MyZpZD0wMjRiOWExZmE4ZTAwNmYxZTM5MzdmNjVmNjZjNDA4ZTZkYThlMWNhNzI4ZWE0MzIyMmE3MzgxZGYxY2M0NDk2MDUmbWV0aG9kPWxpc3RwZWVycyZwbnVtPTEmcG5hbWVpZF4wMjRiOWExZmE4ZTAwNmYxZTM5M3xwYXJyMF4wMjRiOWExZmE4ZTAwNmYxZTM5MyZ0aW1lPDE2NTY5MjA1MzgmcmF0ZT0y"
	autoDetectAPIType(data)
	assert.NotNil(t, data.ApiType)
	assert.Equal(t, int(api.ClnSocket), *data.ApiType) // TODO
}

func TestComplainAboutInvalidAuthenticator(t *testing.T) {
	data := entities.Data{
		Endpoint:    "http://bolt.observer",
		MacaroonHex: "",
	}

	assert.Equal(t, false, complainAboutInvalidAuthenticator(data))

	data.MacaroonHex = "0201036c6e640224030a10b493608461fb6e64810053fa31ef27991201301a0c0a04696e666f120472656164000216697061646472203139322e3136382e3139322e3136380000062072ea006233da839ce6e9f4721331a12041b228d36c0fdad552680f615766d2f4"
	assert.Equal(t, false, complainAboutInvalidAuthenticator(data))

	data.ApiType = intPtr(int(api.ClnSocket)) // TODO
	assert.Equal(t, true, complainAboutInvalidAuthenticator(data))

	data.ApiType = intPtr(int(api.LndGrpc))
	assert.Equal(t, false, complainAboutInvalidAuthenticator(data))

	data.ApiType = intPtr(int(api.LndRest))
	assert.Equal(t, false, complainAboutInvalidAuthenticator(data))

	data.ApiType = intPtr(int(api.LndRest))
	data.MacaroonHex = "tU-RLjMiDpY2U0o3W1oFowar36RFGpWloPbW9-RuZdo9MyZpZD0wMjRiOWExZmE4ZTAwNmYxZTM5MzdmNjVmNjZjNDA4ZTZkYThlMWNhNzI4ZWE0MzIyMmE3MzgxZGYxY2M0NDk2MDUmbWV0aG9kPWxpc3RwZWVycyZwbnVtPTEmcG5hbWVpZF4wMjRiOWExZmE4ZTAwNmYxZTM5M3xwYXJyMF4wMjRiOWExZmE4ZTAwNmYxZTM5MyZ0aW1lPDE2NTY5MjA1MzgmcmF0ZT0y"
	assert.Equal(t, true, complainAboutInvalidAuthenticator(data))

	data.ApiType = intPtr(int(api.LndGrpc))
	assert.Equal(t, true, complainAboutInvalidAuthenticator(data))

	data.ApiType = intPtr(int(api.ClnSocket)) // TODO
	assert.Equal(t, false, complainAboutInvalidAuthenticator(data))
}
