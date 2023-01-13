package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gobwas/glob"
	"github.com/golang/glog"

	entities "github.com/bolt-observer/go_common/entities"
	utils "github.com/bolt-observer/go_common/utils"
	local_utils "github.com/bolt-observer/lightning-vault/utils"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	sentry "github.com/getsentry/sentry-go"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: lightning-vault -stderrthreshold=[INFO|WARNING|FATAL] -log_dir=[string]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func initSentry(env string) {
	if os.Getenv("SENTRY_DSN") == "" {
		glog.Warning("SENTRY_DSN not set, sentry not configured")
		return
	}

	err := sentry.Init(sentry.ClientOptions{
		Environment: env,
		Release:     fmt.Sprintf("lightning-vault@%s", GitRevision),
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			ToCensor := []string{"SecretString", "MacaroonHex"}

			for _, exception := range event.Exception {
				for _, frame := range exception.Stacktrace.Frames {
					for _, censor := range ToCensor {
						delete(frame.Vars, censor)
					}
				}
			}

			for _, thread := range event.Threads {
				for _, frame := range thread.Stacktrace.Frames {
					for _, censor := range ToCensor {
						delete(frame.Vars, censor)
					}
				}
			}

			return event
		},
	})

	if err != nil {
		glog.Warningf("Sentry initialization failed %v", err)
	}
}

func initalize() {
	flag.Usage = usage
	flag.Parse()

	prometheusInit()
}

var (
	// GitRevision baked in during build
	GitRevision = "unknownVersion"
)

func (h *Handlers) toLookup(data entities.Data, uniqueID string) {
	if data.Tags != "" {
		for _, v := range strings.Split(data.Tags, local_utils.Delimiter) {
			if !utils.ValidatePubkey(v) && !utils.ValidatePubkey(v+uniqueID) && utils.AlphaNumeric.MatchString(v) {
				old, exists := h.Lookup[v+uniqueID]
				if exists && old.PubKey != data.PubKey {
					glog.Warningf("Key already exists %s", v+uniqueID)
					continue
				}

				h.Lookup[v+uniqueID] = data
			}
		}
	}

	h.Lookup[data.PubKey+uniqueID] = data
}

func (h *Handlers) deleteLookup(data entities.Data, uniqueID string) {
	if data.Tags != "" {
		for _, v := range strings.Split(data.Tags, local_utils.Delimiter) {
			if !utils.ValidatePubkey(v) && !utils.ValidatePubkey(v+uniqueID) && utils.AlphaNumeric.MatchString(v) {
				delete(h.Lookup, v+uniqueID)
			}
		}
	}

	delete(h.Lookup, data.PubKey+uniqueID)
}

func (h *Handlers) initialLoad() {
	glog.Info("Initial load of keys from secrets manager...")
	ctx := context.Background()
	secrets := local_utils.LoadSecrets(ctx, prefix)

	for k, v := range secrets {
		if v == "{}" {
			glog.Infof("Ignoring empty secret: %v\n", k)
			continue
		}
		var data entities.Data
		err := json.Unmarshal([]byte(v), &data)
		if err != nil || !utils.ValidatePubkey(data.PubKey) {
			sentry.CaptureMessage(fmt.Sprintf("Error unmarshaling secrets: %v", k))
			glog.Warningf("Error unmarshalling secret %v: %v\n", k, err)
			continue
		}

		keys := strings.Split(k, "_")
		if len(keys) != 3 && len(keys) != 2 {
			sentry.CaptureMessage(fmt.Sprintf("Invalid key: %v", k))
			glog.Warningf("Invalid key %v\n", k)
			continue
		}

		if len(keys[1]) < utils.PUBKEY_LEN || keys[1][0:utils.PUBKEY_LEN] != data.PubKey {
			sentry.CaptureMessage(fmt.Sprintf("Invalid key: %v vs %v", k, data.PubKey))
			glog.Warningf("Invalid key %v vs. %v\n", k, data.PubKey)
			continue
		}

		h.toLookup(data, keys[1][66:])
	}

	glog.Info("Initial load of keys from secrets manager... done")
}

var (
	readDurations map[string]time.Duration
	prefix        string
)

func main() {
	initalize()

	env := utils.GetEnvWithDefault("ENV", "")
	initSentry(env)

	fmt.Printf("Macaroon service %s (env: %s) started\n", GitRevision, env)
	godotenv.Load()
	prefix = fmt.Sprintf("%s%s", env, "macaroon")

	if strings.ToLower(env) == "local" {
		h := MakeNewDummyHandlers()
		h.httpListen(false)
	} else {
		h := MakeNewHandlers()
		h.httpListen(true)
	}

}

func (h *Handlers) httpListen(load bool) {

	readDurations = make(map[string]time.Duration)

	for _, key := range strings.Split(utils.GetEnv("READ_API_KEY_10M"), local_utils.Delimiter) {
		readDurations[key] = time.Minute * 10
	}
	for _, key := range strings.Split(utils.GetEnv("READ_API_KEY_1H"), local_utils.Delimiter) {
		readDurations[key] = time.Hour
	}
	for _, key := range strings.Split(utils.GetEnv("READ_API_KEY_1D"), local_utils.Delimiter) {
		readDurations[key] = time.Hour * 24
	}

	writeAPIKeys := strings.Split(utils.GetEnv("WRITE_API_KEY"), local_utils.Delimiter)
	port := utils.GetEnvWithDefault("PORT", "1339")

	if load {
		h.initialLoad()
	}

	router := mux.NewRouter().StrictSlash(false)
	router.Use(handlers.ProxyHeaders)
	router.Use(recoveryMiddleware())

	registerPrometheusHandler(router)

	if !utils.AreElementsUnique(utils.GetKeys(readDurations)) {
		fatalError("Keys are not unique", nil)
		return
	}
	if !utils.AreElementsUnique(writeAPIKeys) {
		fatalError("Keys are not unique", nil)
		return
	}

	router.Path("/").HandlerFunc(h.MainHandler).Methods(http.MethodGet)

	readRoutes := router.PathPrefix("/get/").Subrouter()
	readRoutes.Use(authMiddleware(toDict(utils.GetKeys(readDurations))))
	writeRoutes := router.PathPrefix("/put/").Subrouter()
	writeRoutes.Use(authMiddleware(toDict(writeAPIKeys)))
	verifyRoutes := router.PathPrefix("/verify/").Subrouter()
	verifyRoutes.Use(authMiddleware(toDict(writeAPIKeys)))

	keys := make([]string, 0)
	keys = append(keys, utils.GetKeys(readDurations)...)
	keys = append(keys, writeAPIKeys...)
	queryRoutes := router.PathPrefix("/query/").Subrouter()
	queryRoutes.Use(authMiddleware(toDict(keys)))

	writeRoutes.Path("/").HandlerFunc(h.PutHandler).Methods(http.MethodPost)
	writeRoutes.Path("/{uniqueId}").HandlerFunc(h.PutHandler).Methods(http.MethodPost)

	writeRoutes.Path("/{pubkey}").HandlerFunc(h.DeleteHandler).Methods(http.MethodDelete)
	writeRoutes.Path("/{uniqueId}/{pubkey}").HandlerFunc(h.DeleteHandler).Methods(http.MethodDelete)

	readRoutes.Path("/{pubkey}").HandlerFunc(h.GetHandler).Methods(http.MethodPost, http.MethodGet)
	readRoutes.Path("/{uniqueId}/{pubkey}").HandlerFunc(h.GetHandler).Methods(http.MethodPost, http.MethodGet)

	queryRoutes.Path("/{pubkey}").HandlerFunc(h.QueryHandler).Methods(http.MethodPost, http.MethodGet)
	queryRoutes.Path("/{uniqueId}/{pubkey}").HandlerFunc(h.QueryHandler).Methods(http.MethodPost, http.MethodGet)

	verifyRoutes.Path("/{pubkey}").HandlerFunc(h.VerifyHandler).Methods(http.MethodPost, http.MethodGet)
	verifyRoutes.Path("/{uniqueId}/{pubkey}").HandlerFunc(h.VerifyHandler).Methods(http.MethodPost, http.MethodGet)

	timeout := utils.GetEnvWithDefault("TIMEOUT", "10")
	timeoutInt, err := strconv.Atoi(timeout)
	if err != nil {
		fatalError("timeout could not be parsed", err)
	}

	fmt.Printf("Listening on port %s\n", port)
	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%s", port),
		WriteTimeout: time.Duration(timeoutInt) * time.Second,
		ReadTimeout:  time.Duration(timeoutInt) * time.Second,
	}

	err = srv.ListenAndServe()
	if err != nil {
		sentry.CaptureException(err)
	}

	sentry.CaptureMessage("Server stopped")
	sentry.Flush(time.Second * 1)
}

func fatalError(msg string, err error) {
	if err != nil {
		sentry.CaptureException(err)
	} else {
		sentry.CaptureMessage(msg)
	}
	sentry.Flush(time.Second * 1)
	glog.Fatalf("%s %v", msg, err)
}

func toDict(tokens []string) map[string]string {
	result := make(map[string]string)
	for _, token := range tokens {
		split := strings.Split(token, local_utils.UserPassSeparator)

		if len(split) != 2 {
			sentry.CaptureMessage(fmt.Sprintf("Entry is invalid: %v", token))
			glog.Warningf("Entry is invalid: %s", token)
			continue
		}

		result[split[0]] = split[1]
	}

	return result
}

func unauthorized(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, "You are not authorized to do that\n")
	// auth token here is invalid - do not use it for audit logging
	failureLog("invalid", r.RemoteAddr, "Unauthorized", r.Method)
}

func verifyPresign(w http.ResponseWriter, r *http.Request, credentials map[string]string) bool {
	presign := r.Header.Get(local_utils.PresignHeader)
	if presign == "" {
		return false
	}

	arn, err := local_utils.VerifyGetCallerIdentity(presign, 5*time.Second)
	if err != nil {
		glog.Warningf("Presign check failed: %v", err)
		return false
	}

	for k, v := range credentials {
		if v == local_utils.IAMAuthFlag {
			// k is a glob
			g, err := glob.Compile(k)
			if err != nil {
				continue
			}

			if g.Match(arn) {
				return true
			}
		}
	}

	return false
}

func authMiddleware(credentials map[string]string) mux.MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !verifyPresign(w, r, credentials) {
				u, p, ok := r.BasicAuth()
				if !ok {
					unauthorized(w, r)
					return
				}

				pass, ok := credentials[u]
				if !ok {
					unauthorized(w, r)
					return
				}
				if strings.HasPrefix(pass, "$") {
					// Password hash
					err := bcrypt.CompareHashAndPassword([]byte(pass), []byte(p))
					if err != nil {
						unauthorized(w, r)
						return
					}
				} else {
					// Plaintext password
					if p != pass {
						unauthorized(w, r)
						return
					}
				}
			}

			h.ServeHTTP(w, r)
		})
	}
}

func recoveryMiddleware() mux.MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				err := recover()
				if err != nil {
					glog.Errorf("Uncaught panic happened %v\n", err)
					sentry.CurrentHub().Recover(err)
					sentry.Flush(time.Second * 5)

					w.WriteHeader(http.StatusUnauthorized)
					fmt.Fprintf(w, "Internal server error\n")
				}

			}()
			h.ServeHTTP(w, r)
		})
	}
}
