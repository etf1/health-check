package handlers

import (
	"github.com/etf1/health-check/checks"
	"net/http"
)

// Handler is an http.Handler with additional methods that register health and
// readiness checks. It handles "/live" and "/ready" HTTP endpoints.
type Handler interface {
	// The Handler is an http.Handler, so it can be exposed directly and handle
	// /live and /ready endpoints.
	http.Handler

	// AddLivenessCheck adds a check that indicates that this instance of the
	// application should be destroyed or restarted. A failed liveness check
	// indicates that this instance is unhealthy, not some upstream dependency.
	// Every liveness check is also included as a readiness check.
	AddLivenessCheck(name string, check healthcheck.Check)

	// AddReadinessCheck adds a check that indicates that this instance of the
	// application is currently unable to serve requests because of an upstream
	// or some transient failure. If a readiness check fails, this instance
	// should no longer receiver requests, but should not be restarted or
	// destroyed.
	AddReadinessCheck(name string, check healthcheck.Check)

	// LiveEndpoint is the HTTP handlers for just the /live endpoint, which is
	// useful if you need to attach it into your own HTTP handlers tree.
	LiveEndpoint(http.ResponseWriter, *http.Request)

	// ReadyEndpoint is the HTTP handlers for just the /ready endpoint, which is
	// useful if you need to attach it into your own HTTP handlers tree.
	ReadyEndpoint(http.ResponseWriter, *http.Request)
}

type Options struct {
	Metadata map[string]string
}

// Response represents the response that will be returned when calling LiveEndpoint and ReadyEndpoint
type response struct {
	Checks map[string]string
	Metadata map[string]string
}
