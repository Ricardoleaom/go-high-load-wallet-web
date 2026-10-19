package walletapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/load"
)

// fakeAPI is the wallet API and its token endpoint, in memory, answering
// in the API's shapes: enough for the load lab end to end over HTTP.
type fakeAPI struct {
	t        *testing.T
	server   *httptest.Server
	tokens   atomic.Int64
	expireIn int

	mu       sync.Mutex
	balances map[string]load.Cents
	seen     map[string]bool
	opened   int
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	f := &fakeAPI{t: t, expireIn: 300, balances: map[string]load.Cents{}, seen: map[string]bool{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", f.token)
	mux.HandleFunc("POST /wallets", f.openWallet)
	mux.HandleFunc("GET /wallets/{id}", f.wallet)
	mux.HandleFunc("POST /wallets/{id}/reconciliation", f.reconcile)
	mux.HandleFunc("POST /wagering/transactions", f.submit)
	mux.HandleFunc("GET /providers/{provider}/wagering/transactions/{external}", f.lookup)
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeAPI) client(t *testing.T) *Client {
	t.Helper()
	return New(Config{
		BaseURL:          f.server.URL,
		TokenURL:         f.server.URL + "/token",
		ProviderID:       "bench-provider",
		ProviderClientID: "bench-provider",
		ProviderSecret:   "bench-provider-secret",
		InternalClientID: "bench-internal",
		InternalSecret:   "bench-internal-secret",
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func brl(c load.Cents) map[string]string {
	return map[string]string{"amount": c.String(), "currency": "BRL"}
}

func (f *fakeAPI) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || r.Form.Get("grant_type") != "client_credentials" ||
		r.Form.Get("client_secret") != r.Form.Get("client_id")+"-secret" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
		return
	}
	n := f.tokens.Add(1)
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": fmt.Sprintf("%s-token-%d", r.Form.Get("client_id"), n),
		"expires_in":   f.expireIn,
	})
}

func (f *fakeAPI) requireToken(w http.ResponseWriter, r *http.Request, client string) bool {
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer "+client+"-token-") {
		writeJSON(w, http.StatusForbidden, map[string]string{"code": "FORBIDDEN", "error": "wrong token"})
		return false
	}
	return true
}

func (f *fakeAPI) openWallet(w http.ResponseWriter, r *http.Request) {
	if !f.requireToken(w, r, "bench-internal") {
		return
	}
	var body struct {
		PlayerID       string `json:"playerId"`
		InitialBalance struct {
			Amount string `json:"amount"`
		} `json:"initialBalance"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	initial, err := load.ParseCents(body.InitialBalance.Amount)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "INVALID_MONEY", "error": err.Error()})
		return
	}
	f.mu.Lock()
	f.opened++
	id := fmt.Sprintf("0192f28f-0000-7000-8000-%012d", f.opened)
	f.balances[id] = initial
	f.mu.Unlock()
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "playerId": body.PlayerID, "balance": brl(initial), "version": 1})
}

func (f *fakeAPI) wallet(w http.ResponseWriter, r *http.Request) {
	if !f.requireToken(w, r, "bench-internal") {
		return
	}
	f.mu.Lock()
	balance, ok := f.balances[r.PathValue("id")]
	f.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"code": "WALLET_NOT_FOUND", "error": "wallet not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": r.PathValue("id"), "balance": brl(balance), "version": 1})
}

func (f *fakeAPI) reconcile(w http.ResponseWriter, r *http.Request) {
	if !f.requireToken(w, r, "bench-internal") {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"walletId": r.PathValue("id"), "consistent": true, "checkedEntries": 1})
}

func (f *fakeAPI) submit(w http.ResponseWriter, r *http.Request) {
	if !f.requireToken(w, r, "bench-provider") {
		return
	}
	var body struct {
		ExternalTransactionID string `json:"externalTransactionId"`
		WalletID              string `json:"walletId"`
		Kind                  string `json:"kind"`
		Money                 struct {
			Amount string `json:"amount"`
		} `json:"money"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if r.Header.Get("Idempotency-Key") == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "MISSING_IDEMPOTENCY_KEY", "error": "missing key"})
		return
	}
	amount, _ := load.ParseCents(body.Money.Amount)
	f.mu.Lock()
	replay := f.seen[body.ExternalTransactionID]
	if !replay {
		f.seen[body.ExternalTransactionID] = true
		if body.Kind == "WIN" {
			f.balances[body.WalletID] += amount
		} else {
			f.balances[body.WalletID] -= amount
		}
	}
	balance := f.balances[body.WalletID]
	f.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"transactionId": "0192f28f-0000-7000-8000-000000000001", "status": "PROCESSED",
		"balance": brl(balance), "idempotentReplay": replay,
	})
}

func (f *fakeAPI) lookup(w http.ResponseWriter, r *http.Request) {
	if !f.requireToken(w, r, "bench-provider") {
		return
	}
	f.mu.Lock()
	ok := f.seen[r.PathValue("external")]
	f.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"code": "TRANSACTION_NOT_FOUND", "error": "transaction not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "PROCESSED", "kind": "BET"})
}
