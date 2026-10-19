// Package server is the load lab's HTTP API: operators start, follow and
// stop load runs; one runs at a time.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/auth"
	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/load"
)

// Verifier admits operators.
type Verifier interface {
	Operator(token string) (name string, err error)
}

// keepRuns is how many finished runs stay readable.
const keepRuns = 20

// Lab holds the current run and the last finished ones.
type Lab struct {
	ctx    context.Context
	target load.Target
	limits load.Limits

	mu      sync.Mutex
	current *load.Run
	runs    []*load.Run // newest last
}

// NewLab runs load against target within limits; ctx bounds every run.
func NewLab(ctx context.Context, target load.Target, limits load.Limits) *Lab {
	return &Lab{ctx: ctx, target: target, limits: limits}
}

var errBusy = errors.New("a run is in progress")

func (l *Lab) start(spec load.Spec) (*load.Run, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.current != nil {
		select {
		case <-l.current.Done():
		default:
			return nil, errBusy
		}
	}
	run, err := load.Start(l.ctx, l.target, spec, l.limits)
	if err != nil {
		return nil, err
	}
	l.current = run
	l.runs = append(l.runs, run)
	if len(l.runs) > keepRuns {
		l.runs = l.runs[len(l.runs)-keepRuns:]
	}
	return run, nil
}

func (l *Lab) find(id string) *load.Run {
	l.mu.Lock()
	defer l.mu.Unlock()
	i := slices.IndexFunc(l.runs, func(r *load.Run) bool { return r.ID == id })
	if i < 0 {
		return nil
	}
	return l.runs[i]
}

func (l *Lab) list() []load.Snapshot {
	l.mu.Lock()
	runs := slices.Clone(l.runs)
	l.mu.Unlock()
	snapshots := make([]load.Snapshot, 0, len(runs))
	for _, r := range slices.Backward(runs) {
		s := r.Snapshot()
		s.Series = nil
		snapshots = append(snapshots, s)
	}
	return snapshots
}

// NewHandler serves the lab. Every route but /health needs an operator's
// token; origins may call it from a browser.
func NewHandler(lab *Lab, verifier Verifier, origins []string) http.Handler {
	h := &handler{lab: lab}
	protected := http.NewServeMux()
	protected.HandleFunc("POST /runs", h.startRun)
	protected.HandleFunc("GET /runs", h.listRuns)
	protected.HandleFunc("GET /runs/{id}", h.getRun)
	protected.HandleFunc("POST /runs/{id}/stop", h.stopRun)
	protected.HandleFunc("GET /runs/{id}/events", h.events)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("/", requireOperator(verifier, protected))
	return withCORS(origins, mux)
}

type handler struct{ lab *Lab }

type startRequest struct {
	Scenario        load.Scenario `json:"scenario"`
	Rate            int           `json:"rate"`
	DurationSeconds int           `json:"durationSeconds"`
	Wallets         int           `json:"wallets"`
}

func (h *handler) startRun(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "malformed JSON body")
		return
	}
	run, err := h.lab.start(load.Spec{
		Scenario: req.Scenario,
		Rate:     req.Rate,
		Duration: time.Duration(req.DurationSeconds) * time.Second,
		Wallets:  req.Wallets,
	})
	switch {
	case errors.Is(err, errBusy):
		writeError(w, http.StatusConflict, "RUN_IN_PROGRESS", "a run is in progress; stop it or wait for it")
		return
	case err != nil:
		writeError(w, http.StatusBadRequest, "INVALID_RUN", err.Error())
		return
	}
	slog.Info("run started", "run", run.ID, "operator", operatorFrom(r.Context()),
		"scenario", req.Scenario, "rate", req.Rate, "seconds", req.DurationSeconds, "wallets", req.Wallets)
	writeJSON(w, http.StatusCreated, run.Snapshot())
}

func (h *handler) listRuns(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"runs": h.lab.list()})
}

func (h *handler) run(w http.ResponseWriter, r *http.Request) *load.Run {
	run := h.lab.find(r.PathValue("id"))
	if run == nil {
		writeError(w, http.StatusNotFound, "RUN_NOT_FOUND", "run not found")
	}
	return run
}

func (h *handler) getRun(w http.ResponseWriter, r *http.Request) {
	if run := h.run(w, r); run != nil {
		writeJSON(w, http.StatusOK, run.Snapshot())
	}
}

func (h *handler) stopRun(w http.ResponseWriter, r *http.Request) {
	if run := h.run(w, r); run != nil {
		run.Stop()
		slog.Info("run stopped", "run", run.ID, "operator", operatorFrom(r.Context()))
		writeJSON(w, http.StatusAccepted, run.Snapshot())
	}
}

// progress is what the event stream sends every second: the run without
// its series, which grows with the run, plus the second just closed. A
// client that reconnects reads the series once from GET /runs/{id}.
type progress struct {
	load.Snapshot
	Point *load.Point `json:"point,omitempty"`
}

func progressOf(run *load.Run) progress {
	s := run.Snapshot()
	p := progress{Snapshot: s}
	if n := len(s.Series); n > 0 {
		p.Point = &s.Series[n-1]
	}
	p.Series = nil
	return p
}

// events streams the run as Server-Sent Events: "progress" every second
// and "result", with the final run, when it ends.
func (h *handler) events(w http.ResponseWriter, r *http.Request) {
	run := h.run(w, r)
	if run == nil {
		return
	}
	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	send := func(event string, v any) bool {
		data, err := json.Marshal(v)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
			return false
		}
		return rc.Flush() == nil
	}

	if !send("progress", progressOf(run)) {
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-run.Done():
			final := run.Snapshot()
			final.Series = nil
			send("result", final)
			return
		case <-ticker.C:
			if !send("progress", progressOf(run)) {
				return
			}
		}
	}
}

type contextKey struct{}

func operatorFrom(ctx context.Context) string {
	name, _ := ctx.Value(contextKey{}).(string)
	return name
}

func requireOperator(verifier Verifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "missing bearer token")
			return
		}
		name, err := verifier.Operator(token)
		if err != nil {
			status, code := http.StatusUnauthorized, "UNAUTHENTICATED"
			if errors.Is(err, auth.ErrNotOperator) {
				status, code = http.StatusForbidden, "FORBIDDEN"
			}
			writeError(w, status, code, "the load lab is for operators")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, name)))
	})
}

// withCORS admits the listed browser origins as the wallet API does:
// preflights answered before authentication, other origins refused, no
// credentials.
func withCORS(origins []string, next http.Handler) http.Handler {
	if len(origins) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && slices.Contains(origins, origin)
		w.Header().Add("Vary", "Origin")
		if r.Method == http.MethodOptions && origin != "" && r.Header.Get("Access-Control-Request-Method") != "" {
			if !allowed {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "origin not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "error": message})
}
