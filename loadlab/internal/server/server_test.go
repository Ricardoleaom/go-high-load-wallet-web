package server_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/auth"
	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/auth/authtest"
	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/load"
	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/server"
)

const consoleOrigin = "http://localhost:5173"

// memoryTarget is an always-consistent wallet API in memory.
type memoryTarget struct {
	mu       sync.Mutex
	n        int
	balances map[string]load.Cents
	seen     map[string]bool
}

func (m *memoryTarget) OpenWallet(_ context.Context, _ string, initial load.Cents) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.n++
	id := fmt.Sprintf("w%d", m.n)
	m.balances[id] = initial
	return id, nil
}

func (m *memoryTarget) Submit(_ context.Context, op load.Operation) (load.Outcome, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.seen[op.ExternalID] {
		return load.Outcome{HTTPStatus: 200, Status: "PROCESSED", Replay: true}, nil
	}
	m.seen[op.ExternalID] = true
	m.balances[op.WalletID] += op.Delta()
	return load.Outcome{HTTPStatus: 200, Status: "PROCESSED"}, nil
}

func (m *memoryTarget) Lookup(context.Context, string) (string, bool, error) { return "", false, nil }

func (m *memoryTarget) Balance(_ context.Context, id string) (load.Cents, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.balances[id], nil
}

func (m *memoryTarget) Reconcile(context.Context, string) (bool, error) { return true, nil }

type harness struct {
	server *httptest.Server
	issuer *authtest.Issuer
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	issuer := authtest.NewIssuer(t)
	verifier, err := auth.NewVerifier(context.Background(), auth.Config{
		JWKSURL: issuer.JWKSURL(), Issuer: issuer.URL(), Audience: authtest.Audience, RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	lab := server.NewLab(context.Background(),
		&memoryTarget{balances: map[string]load.Cents{}, seen: map[string]bool{}},
		load.Limits{MaxRate: 5000, MaxDuration: time.Minute, MaxWallets: 100, MaxInFlight: 1000})
	srv := httptest.NewServer(server.NewHandler(lab, verifier, []string{consoleOrigin}))
	t.Cleanup(srv.Close)
	return &harness{server: srv, issuer: issuer}
}

func (h *harness) request(t *testing.T, method, path, token string, body any) *http.Response {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, h.server.URL+path, &payload)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := h.server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return v
}

type snapshot struct {
	ID     string `json:"id"`
	Phase  string `json:"phase"`
	Sent   int64  `json:"sent"`
	Result *struct {
		Phase        string `json:"phase"`
		Verification struct {
			Passed bool `json:"passed"`
		} `json:"verification"`
	} `json:"result"`
}

var shortRun = map[string]any{"scenario": "uniform", "rate": 50, "durationSeconds": 1, "wallets": 2}

func TestOnlyOperatorsStartRuns(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	if got := h.request(t, http.MethodPost, "/runs", "", shortRun).StatusCode; got != http.StatusUnauthorized {
		t.Fatalf("no token: %d, want 401", got)
	}
	if got := h.request(t, http.MethodPost, "/runs", h.issuer.ProviderToken(t), shortRun).StatusCode; got != http.StatusForbidden {
		t.Fatalf("provider token: %d, want 403", got)
	}
	resp := h.request(t, http.MethodPost, "/runs", h.issuer.OperatorToken(t), shortRun)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("operator token: %d, want 201", resp.StatusCode)
	}
	if s := decode[snapshot](t, resp); s.ID == "" {
		t.Fatal("created run has no ID")
	}
}

func TestAnInvalidRunIsRefused(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	resp := h.request(t, http.MethodPost, "/runs", h.issuer.OperatorToken(t),
		map[string]any{"scenario": "uniform", "rate": 999999, "durationSeconds": 1, "wallets": 1})

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", resp.StatusCode)
	}
	if body := decode[map[string]string](t, resp); body["code"] != "INVALID_RUN" || !strings.Contains(body["error"], "rate") {
		t.Fatalf("body %v", body)
	}
}

func TestOneRunAtATime(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	operator := h.issuer.OperatorToken(t)
	long := map[string]any{"scenario": "uniform", "rate": 10, "durationSeconds": 30, "wallets": 1}

	first := decode[snapshot](t, h.request(t, http.MethodPost, "/runs", operator, long))
	second := h.request(t, http.MethodPost, "/runs", operator, shortRun)
	if second.StatusCode != http.StatusConflict {
		t.Fatalf("second run: %d, want 409", second.StatusCode)
	}
	if body := decode[map[string]string](t, second); body["code"] != "RUN_IN_PROGRESS" {
		t.Fatalf("body %v", body)
	}

	stopped := h.request(t, http.MethodPost, "/runs/"+first.ID+"/stop", operator, nil)
	if stopped.StatusCode != http.StatusAccepted {
		t.Fatalf("stop: %d, want 202", stopped.StatusCode)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		s := decode[snapshot](t, h.request(t, http.MethodGet, "/runs/"+first.ID, operator, nil))
		if s.Result != nil {
			if s.Result.Phase != "stopped" {
				t.Fatalf("stopped run ended %q", s.Result.Phase)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("stopped run never finished")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got := h.request(t, http.MethodPost, "/runs", operator, shortRun).StatusCode; got != http.StatusCreated {
		t.Fatalf("run after the first ended: %d, want 201", got)
	}
	if got := h.request(t, http.MethodGet, "/runs/nope", operator, nil).StatusCode; got != http.StatusNotFound {
		t.Fatalf("unknown run: %d, want 404", got)
	}
}

func TestEventsStreamProgressAndTheResult(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	operator := h.issuer.OperatorToken(t)
	run := decode[snapshot](t, h.request(t, http.MethodPost, "/runs", operator,
		map[string]any{"scenario": "uniform", "rate": 100, "durationSeconds": 2, "wallets": 2}))

	resp := h.request(t, http.MethodGet, "/runs/"+run.ID+"/events", operator, nil)
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type %q", ct)
	}
	var events []string
	var result snapshot
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	event := ""
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimPrefix(line, "event: ")
			events = append(events, event)
		case strings.HasPrefix(line, "data: ") && event == "result":
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &result); err != nil {
				t.Fatalf("result data: %v", err)
			}
		}
	}

	progress := 0
	for _, e := range events {
		if e == "progress" {
			progress++
		}
	}
	if progress < 2 || events[len(events)-1] != "result" {
		t.Fatalf("events %v, want progress every second and the result last", events)
	}
	if result.Result == nil || !result.Result.Verification.Passed || result.Sent != 200 {
		t.Fatalf("result %+v, want 200 sent and verification passed", result)
	}
}

func TestCORSForTheConsole(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	req, _ := http.NewRequest(http.MethodOptions, h.server.URL+"/runs", nil)
	req.Header.Set("Origin", consoleOrigin)
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp, err := h.server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent || resp.Header.Get("Access-Control-Allow-Origin") != consoleOrigin {
		t.Fatalf("preflight: %d, origin %q", resp.StatusCode, resp.Header.Get("Access-Control-Allow-Origin"))
	}

	req, _ = http.NewRequest(http.MethodOptions, h.server.URL+"/runs", nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp, err = h.server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden || resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("foreign preflight: %d, origin %q", resp.StatusCode, resp.Header.Get("Access-Control-Allow-Origin"))
	}
}
