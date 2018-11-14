// Copyright 2017 by the contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"github.com/etf1/health-check/checks"
)

const livenessPatternEnvVar = "HEALTH_LIVENESS_PATTERN"
const readynessPatternEnvVar = "HEALTH_READYNESS_PATTERN"

const defaultLivenessPattern = "/live"
const defaultReadynessPattern = "/ready"

type basicHandler struct {
	http.ServeMux
	checksMutex     sync.RWMutex
	livenessChecks  map[string]healthcheck.Check
	readinessChecks map[string]healthcheck.Check
	metadata 		map[string]string
}

// creates a handlers that listens on /live & /ready
// and returns liveness and readiness statuses for each provided livenessChecks & readynessChecks
// it also returns metadata in each response
func NewHandler(options Options) Handler {
	h := &basicHandler{
		livenessChecks:  make(map[string]healthcheck.Check),
		readinessChecks: make(map[string]healthcheck.Check),
		metadata: 		 make(map[string]string),
	}
	// Set the metadata that will be return on each http response
	if options.Metadata != nil { h.metadata = options.Metadata }

	// Get live & ready endpoint patterns from env vars
	livenessPattern, ok := os.LookupEnv(livenessPatternEnvVar)
	readynessPattern, ok := os.LookupEnv(readynessPatternEnvVar)
	if !ok { livenessPattern = defaultLivenessPattern }
	if !ok { readynessPattern = defaultReadynessPattern }

	h.Handle(livenessPattern, http.HandlerFunc(h.LiveEndpoint))
	h.Handle(readynessPattern, http.HandlerFunc(h.ReadyEndpoint))
	return h
}

func (s *basicHandler) LiveEndpoint(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, s.livenessChecks)
}

func (s *basicHandler) ReadyEndpoint(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, s.readinessChecks, s.livenessChecks)
}

func (s *basicHandler) AddLivenessCheck(name string, check healthcheck.Check) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.livenessChecks[name] = check
}

func (s *basicHandler) AddReadinessCheck(name string, check healthcheck.Check) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.readinessChecks[name] = check
}

func (s *basicHandler) collectChecks(checks map[string]healthcheck.Check, resultsOut map[string]string, statusOut *int) {
	s.checksMutex.RLock()
	defer s.checksMutex.RUnlock()
	for name, check := range checks {
		if err := check(); err != nil {
			*statusOut = http.StatusServiceUnavailable
			resultsOut[name] = err.Error()
		} else {
			resultsOut[name] = "OK"
		}
	}
}

func (s *basicHandler) handle(w http.ResponseWriter, r *http.Request, checks ...map[string]healthcheck.Check) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	checkResults := &response{
		Checks: make(map[string]string),
		Metadata: s.metadata,
	}

	status := http.StatusOK
	for _, checks := range checks {
		s.collectChecks(checks, checkResults.Checks, &status)
	}

	// write out the response code and content type header
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	// unless ?full=1, return an empty body. Kubernetes only cares about the
	// HTTP status code, so we won't waste bytes on the full body.
	if r.URL.Query().Get("full") != "1" {
		w.Write([]byte("{}\n"))
		return
	}

	// otherwise, write the JSON body ignoring any encoding errors (which
	// shouldn't really be possible since we're encoding a map[string]string).
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "    ")
	encoder.Encode(checkResults)
}
