package main

import (
	"net/http"
	"strings"

	"github.com/cabify/gotoprom"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// LoggingResponseWriter struct
type LoggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

type labels struct {
	Method string `label:"method"`
	Path   string `label:"path"`
}

type labelsCode struct {
	Method string `label:"method"`
	Path   string `label:"path"`
	Code   int    `label:"code"`
}

type authLabels struct {
	Identifier string `label:"identifier"`
	Method     string `label:"method"`
	Success    bool   `label:"success"`
}

var (
	promInitialized = false
	metrics         struct {
		HTTPDuration func(labels) prometheus.Histogram   `name:"http_duration" help:"Duration of HTTP requests" buckets:""`
		Reqs         func(labelsCode) prometheus.Counter `name:"requests_total" help:"How many HTTP requests processed"`
		AuthReqs     func(authLabels) prometheus.Counter `name:"auth_requests_total" help:"How many HTTP requests processed per user"`
	}
)

// NewLoggingResponseWriter - constructs a new LoggingResponseWriter
func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{w, http.StatusOK}
}

// WriteHeader - writes the HTTP response header
func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		lrw := NewLoggingResponseWriter(w)
		timer := prometheus.NewTimer(metrics.HTTPDuration(labels{Path: path, Method: r.Method}))
		next.ServeHTTP(lrw, r)

		timer.ObserveDuration()
		metrics.Reqs(labelsCode{Path: path, Method: r.Method, Code: lrw.statusCode}).Inc()
	})
}

func prometheusInit() {
	if !promInitialized {
		gotoprom.MustInit(&metrics, "macaroon")
		promInitialized = true
	}
}

func registerPrometheusHandler(router *mux.Router) {
	// Prometheus
	router.Use(prometheusMiddleware)
	router.Path("/metrics").Handler(promhttp.Handler())
}

func auditLog(token, addr, message, method string) {
	tokens := strings.Split(token, ":")
	if len(tokens) != 2 {
		glog.Infof("[AUDIT LOG] [%v] %s", addr, message)
		split := strings.Split(addr, ":")
		identifier := addr
		if len(split) == 2 {
			identifier = split[0]
		}
		metrics.AuthReqs(authLabels{Identifier: identifier, Method: method, Success: true}).Inc()
	} else {
		glog.Infof("[AUDIT LOG] [%v] token(%s) %s", tokens[0], addr, message)
		metrics.AuthReqs(authLabels{Identifier: tokens[0], Method: method, Success: true}).Inc()
	}
}

func failureLog(token, addr, message, method string) {
	tokens := strings.Split(token, ":")
	if len(tokens) != 2 {
		glog.Infof("[FAILURE LOG] [%v] %s", addr, message)
		split := strings.Split(addr, ":")
		identifier := addr
		if len(split) == 2 {
			identifier = split[0]
		}

		metrics.AuthReqs(authLabels{Identifier: identifier, Method: method, Success: false}).Inc()
	} else {
		glog.Infof("[FAILURE LOG] [%v] token(%s) %s", tokens[0], addr, message)
		metrics.AuthReqs(authLabels{Identifier: tokens[0], Method: method, Success: false}).Inc()
	}
}
