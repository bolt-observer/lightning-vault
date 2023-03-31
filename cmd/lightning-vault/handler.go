package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	api "github.com/bolt-observer/agent/lightning"
	entities "github.com/bolt-observer/go_common/entities"
	utils "github.com/bolt-observer/go_common/utils"
	local_utils "github.com/bolt-observer/lightning-vault/utils"
	sentry "github.com/getsentry/sentry-go"
	"github.com/gorilla/mux"
)

// Handlers struct (all method used by HTTP handlers)
type Handlers struct {
	AddCall    local_utils.InsertOrUpdateSecretSignature
	DeleteCall local_utils.DeleteSecretSignature
	VerifyCall func(w http.ResponseWriter, r *http.Request, data *entities.Data, pubkey, uniqueID string) bool
	Lookup     map[string]entities.Data
}

// MakeNewHandlers - creates new Handlers
func MakeNewHandlers() *Handlers {
	r := &Handlers{
		AddCall:    local_utils.InsertOrUpdateSecretWithRetries,
		DeleteCall: local_utils.InvalidateSecretWithRetries,
		Lookup:     make(map[string]entities.Data),
	}

	r.VerifyCall = r.verify
	return r
}

// MakeNewDummyHandlers - create new Handlers that have external calls mocked
func MakeNewDummyHandlers() *Handlers {
	r := &Handlers{
		AddCall:    local_utils.InsertSecretDummy,
		DeleteCall: local_utils.InvalidateSecretDummy,
		Lookup:     make(map[string]entities.Data),
	}

	r.VerifyCall = r.verify
	return r
}

// MainHandler - / route response
func (h *Handlers) MainHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to Lightning Vault!\n")
}

func (h *Handlers) obtainUniqueID(w http.ResponseWriter, r *http.Request) (string, error) {
	params := mux.Vars(r)

	uniqueID, ok := params["uniqueId"]
	if !ok {
		uniqueID = ""
	} else {
		if !utils.AlphaNumeric.MatchString(uniqueID) {
			h.badRequest(w, r, "uniqueId parameter is invalid", fmt.Sprintf("uniqueId parameter is invalid - %v", uniqueID))
			return "", fmt.Errorf("invalid parameter")
		}
	}

	return uniqueID, nil
}

// QueryHandler - /query route can check whether macaroon exists or not
func (h *Handlers) QueryHandler(w http.ResponseWriter, r *http.Request) {
	// This must not leak any data about the macaroon

	params := mux.Vars(r)
	pubkey := params["pubkey"]

	uniqueID, err := h.obtainUniqueID(w, r)
	if err != nil {
		return
	}

	auditLog(r.Header.Get("Authorization"), r.RemoteAddr, fmt.Sprintf("Query %s (%s)", pubkey, uniqueID), r.Method)

	_, ok := h.Lookup[pubkey+uniqueID]
	if !ok {
		failureLog(r.Header.Get("Authorization"), r.RemoteAddr, fmt.Sprintf("[Query] Secret %s not found", pubkey), r.Method)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Not found\n")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Macaroon exists\n")
}

// DeleteHandler - delete macaroon
func (h *Handlers) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	params := mux.Vars(r)
	pubkey := params["pubkey"]

	uniqueID, err := h.obtainUniqueID(w, r)
	if err != nil {
		return
	}

	auditLog(r.Header.Get("Authorization"), r.RemoteAddr, fmt.Sprintf("Delete %s (%s)", pubkey, uniqueID), r.Method)

	e, ok := h.Lookup[pubkey+uniqueID]
	if !ok {
		failureLog(r.Header.Get("Authorization"), r.RemoteAddr, fmt.Sprintf("[Delete] Secret %s not found", pubkey), r.Method)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Not found\n")
		return
	}

	_, err = h.DeleteCall(ctx, fmt.Sprintf("%s_%s%s_", prefix, e.PubKey, uniqueID))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		failureLog(r.Header.Get("Authorization"), r.RemoteAddr, fmt.Sprintf("AWS delete secret failed with error %v", err), r.Method)
		fmt.Fprintf(w, "Internal error\n")
		return
	}

	h.deleteLookup(e, uniqueID)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Macaroon deleted\n")
}

// GetHandler - gets nacaroon
func (h *Handlers) GetHandler(w http.ResponseWriter, r *http.Request) {
	var (
		data entities.Data
		ok   bool
	)

	params := mux.Vars(r)
	pubkey := params["pubkey"]

	uniqueID, err := h.obtainUniqueID(w, r)
	if err != nil {
		return
	}

	data, ok = h.Lookup[pubkey+uniqueID]
	if !ok {
		failureLog(r.Header.Get("Authorization"), r.RemoteAddr, fmt.Sprintf("[Get] Secret %s not found", pubkey), r.Method)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Not found\n")
		return
	}

	duration, ok := readDurations[r.Header.Get("Authorization")]
	if !ok {
		duration = time.Minute * 10
	}

	data = local_utils.GetConstrained(&data, duration)

	auditLog(r.Header.Get("Authorization"), r.RemoteAddr, fmt.Sprintf("Get %s (%s) valid for %v", pubkey, uniqueID, duration), r.Method)
	encoder := json.NewEncoder(w)
	err = encoder.Encode(&data)
	if err != nil {
		h.badRequest(w, r, "json encoding failed", fmt.Sprintf("[Get] json encoding failed: %v", err))
		sentry.CaptureException(err)
		return
	}
}

func extractHostnameAndPort(endpoint string) (string, int) {
	defaultPort := -1
	if strings.HasPrefix(strings.ToLower(endpoint), "https") {
		defaultPort = 443
	} else if strings.HasPrefix(strings.ToLower(endpoint), "http") {
		defaultPort = 80
	}

	proto := regexp.MustCompile("^[:a-zA-Z0-9_-]+//.*")
	uri := endpoint
	if !proto.MatchString(endpoint) {
		uri = fmt.Sprintf("http://%s", endpoint)
	}
	u, err := url.Parse(uri)

	if err != nil {
		// Handle the [::1]:1337 IPv6 corner case
		re := regexp.MustCompile(`\[(.+)\]`)
		sanitized := re.ReplaceAllString(endpoint, "")

		s := strings.Split(sanitized, ":")
		port := -1
		if len(s) > 1 {
			port, err := strconv.Atoi(s[1])
			if err != nil || port < 0 || port > 65535 {
				port = defaultPort
			}
		}

		if endpoint != sanitized {
			endpoint = re.ReplaceAllString(endpoint, "$1")
		}

		return endpoint, port
	}

	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 0 || port > 65535 {
		port = defaultPort
	}

	return u.Hostname(), port
}

func autoDetectAPIType(data *entities.Data) {
	if strings.HasPrefix(data.Endpoint, "http") {
		data.ApiType = intPtr(int(api.LndRest))

		u, err := url.Parse(data.Endpoint)
		if err != nil {
			data.ApiType = intPtr(int(api.LndGrpc))
		} else if u.Port() == "10009" {
			data.ApiType = intPtr(int(api.LndGrpc))
			data.Endpoint = u.Host
		}
	} else {
		data.ApiType = intPtr(int(api.LndGrpc))
	}

	t := api.ClnCommando
	if local_utils.DetectAuthenticatorType(data.MacaroonHex, &t) == local_utils.Rune {
		data.ApiType = intPtr(int(t))
	}
}

func intPtr(i int) *int {
	return &i
}

func complainAboutInvalidAuthenticator(data entities.Data) bool {
	if data.ApiType == nil {
		return false
	}

	apiType, err := api.GetAPIType(data.ApiType)
	if err != nil || apiType == nil {
		return false
	}

	a := local_utils.ToAuthenticatorType(*apiType)
	b := local_utils.DetectAuthenticatorType(data.MacaroonHex, apiType)

	if a == local_utils.Unknown {
		return false
	}

	// If type is known the used authenticator needs to match

	return a != b
}

// PutHandler - put a macaroon
func (h *Handlers) PutHandler(w http.ResponseWriter, r *http.Request) {
	var data entities.Data

	ctx := context.Background()
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&data)
	if err != nil {
		h.badRequest(w, r, "json decoding failed", fmt.Sprintf("[Put] json decoding failed: %v", err))
		sentry.CaptureException(err)
		return
	}

	uniqueID, err := h.obtainUniqueID(w, r)
	if err != nil {
		return
	}

	// Some basic validation
	if !utils.ValidatePubkey(data.PubKey) {
		h.badRequest(w, r, "pubkey validation failed", fmt.Sprintf("[Put] pubkey validation failed: %v", data.PubKey))
		return
	}

	orig, ok := h.Lookup[data.PubKey+uniqueID]

	if data.Endpoint == "" {
		if !ok {
			h.badRequest(w, r, "empty endpoint", "[Put] empty endpoint")
			return
		}

		auditLog(r.Header.Get("Authorization"), r.RemoteAddr, "[Put] using old endpoint (no new one supplied)", r.Method)
		data.Endpoint = orig.Endpoint
	}

	if data.ApiType != nil {
		t, err := api.GetAPIType(data.ApiType)

		if err != nil || *t == api.ClnSocket {
			h.badRequest(w, r, "invalid api type", fmt.Sprintf("[Put] invalid api type - %v", data.ApiType))
		}
	} else {
		if ok {
			if orig.ApiType != nil {
				data.ApiType = orig.ApiType
			}
		} else {
			autoDetectAPIType(&data)
		}
	}

	hostname := ""
	port := -1
	needCert := false

	if data.ApiType != nil {
		if *data.ApiType == int(api.LndGrpc) {
			hostname, port = extractHostnameAndPort(data.Endpoint)
			needCert = true
			if port < 0 {
				h.badRequest(w, r, "invalid endpoint", fmt.Sprintf("[Put] invalid endpoint - %s", data.Endpoint))
				return
			}
		} else if *data.ApiType == int(api.LndRest) {
			hostname, port = extractHostnameAndPort(data.Endpoint)
			needCert = true
		} else if *data.ApiType == int(api.ClnCommando) {
			needCert = false
		} else {
			h.badRequest(w, r, "unsupported api type", fmt.Sprintf("[Put] unsupported api type - %v", *data.ApiType))
		}
	}

	if port > 0 {
		hostname = fmt.Sprintf("%s:%d", hostname, port)
	}

	if data.CertificateBase64 == "" {
		if ok && orig.CertificateBase64 != "" {
			auditLog(r.Header.Get("Authorization"), r.RemoteAddr, "[Put] using old certificate (no new one supplied)", r.Method)
			data.CertificateBase64 = orig.CertificateBase64
		} else {
			// TODO: deprecate this
			if needCert {
				data.CertificateBase64 = utils.ObtainCert(hostname)
			}
		}
	}

	if data.CertificateBase64 == "" && needCert {
		h.badRequest(w, r, "empty certificate", "[Put] empty certificate")
		return
	}

	_, err = utils.SafeBase64Decode(data.CertificateBase64)
	if err != nil {
		h.badRequest(w, r, "invalid certificate", fmt.Sprintf("[Put] invalid certificate - %s", data.CertificateBase64))
		return
	}

	if data.MacaroonHex == "" {
		if !ok {
			h.badRequest(w, r, "empty macaroon/rune value", "[Put] empty macaroon/rune value")
			return
		}

		auditLog(r.Header.Get("Authorization"), r.RemoteAddr, "[Put] using old macaroon/rune (no new one supplied)", r.Method)
		data.MacaroonHex = orig.MacaroonHex
	}

	if data.CertVerificationType == nil && ok {
		if orig.CertVerificationType != nil {
			auditLog(r.Header.Get("Authorization"), r.RemoteAddr, "[Put] using old certificate verification type (no new one supplied)", r.Method)
			data.CertVerificationType = orig.CertVerificationType
		}
	}

	if complainAboutInvalidAuthenticator(data) {
		h.badRequest(w, r, "invalid macaroon/rune", "[Put] invalid macaroon/rune - not compatible with API type")
		return
	}

	apiType, err := api.GetAPIType(data.ApiType)
	if err != nil {
		apiType = nil
	}

	_, err = local_utils.Constrain(data.MacaroonHex, 1*time.Minute, apiType)
	if err != nil {
		h.badRequest(w, r, "invalid macaroon/rune", "[Put] invalid macaroon/rune - could not constrain")
		return
	}

	result := new(bytes.Buffer)
	encoder := json.NewEncoder(result)
	err = encoder.Encode(&data)
	if err != nil {
		h.badRequest(w, r, "json encoding failed", fmt.Sprintf("[Put] json encoding failed: %v", err))
		sentry.CaptureException(err)
		return
	}

	verify, err := strconv.ParseBool(utils.GetEnvWithDefault("VERIFY", "true"))
	if err == nil && verify && !h.VerifyCall(w, r, &data, data.PubKey, uniqueID) {
		return
	}

	h.toLookup(data, uniqueID)

	_, status, err := h.AddCall(ctx, fmt.Sprintf("%s_%s%s_", prefix, data.PubKey, uniqueID), result.String())

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		failureLog(r.Header.Get("Authorization"), r.RemoteAddr, fmt.Sprintf("AWS add secret failed with error %v", err), r.Method)
		fmt.Fprintf(w, "Internal error\n")
		return
	}

	if status == local_utils.Updated {
		w.WriteHeader(http.StatusOK)
		auditLog(r.Header.Get("Authorization"), r.RemoteAddr, fmt.Sprintf("Put (update) %v", data.PubKey), r.Method)
		fmt.Fprintf(w, "Updated secret %v", data.PubKey)
	} else {
		w.WriteHeader(http.StatusCreated)
		auditLog(r.Header.Get("Authorization"), r.RemoteAddr, fmt.Sprintf("Put (new) %v", data.PubKey), r.Method)
		fmt.Fprintf(w, "Inserted secret %v", data.PubKey)
	}
}

func (h *Handlers) badRequest(w http.ResponseWriter, r *http.Request, reason, logReason string) {
	failureLog(r.Header.Get("Authorization"), r.RemoteAddr, fmt.Sprintf("Bad request - %s", logReason), r.Method)
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, "Bad request - %s\n", reason)
}

// VerifyHandler - check whether macaroon is usable
func (h *Handlers) VerifyHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	pubkey := params["pubkey"]

	uniqueID, err := h.obtainUniqueID(w, r)
	if err != nil {
		return
	}

	auditLog(r.Header.Get("Authorization"), r.RemoteAddr, fmt.Sprintf("Verify %s (%s)", pubkey, uniqueID), r.Method)

	if !utils.ValidatePubkey(pubkey) {
		h.badRequest(w, r, "pubkey validation failed", fmt.Sprintf("[Verify] pubkey validation failed: %v", pubkey))
		return
	}

	data, ok := h.Lookup[pubkey+uniqueID]
	if !ok {
		failureLog(r.Header.Get("Authorization"), r.RemoteAddr, fmt.Sprintf("[Verify] Secret %s not found", pubkey), r.Method)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Not found\n")
		return
	}

	if !h.VerifyCall(w, r, &data, pubkey, uniqueID) {
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Everything is ok\n")
}

func (h *Handlers) verify(w http.ResponseWriter, r *http.Request, data *entities.Data, pubkey, uniqueID string) bool {
	api, err := api.NewAPI(api.LndGrpc, func() (*entities.Data, error) { return data, nil })
	if err != nil {
		h.badRequest(w, r, "invalid credentials - check failed", fmt.Sprintf("failed to get lightning client, error %v", err))
		return false
	}
	if api == nil {
		h.badRequest(w, r, "invalid credentials - check failed", "failed to get lightning client")
		return false
	}
	defer api.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := api.GetInfo(ctx)
	if err != nil {
		h.badRequest(w, r, "invalid credentials - check failed", fmt.Sprintf("[Verify] failed to get info %v", err))
		return false
	}

	if !strings.EqualFold(info.IdentityPubkey, pubkey) {
		h.badRequest(w, r, "invalid credentials - check failed", fmt.Sprintf("[Verify] endpoint is %s not %s", info.IdentityPubkey, pubkey))
		return false
	}

	_, err = api.GetChannels(ctx)
	if err != nil {
		h.badRequest(w, r, "invalid credentials - check failed", fmt.Sprintf("[Verify] failed to get channels %v", err))
		return false
	}

	return true
}
