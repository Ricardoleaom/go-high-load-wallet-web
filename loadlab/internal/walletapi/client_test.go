package walletapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/load"
)

func TestSubmitAndBooksRoundTrip(t *testing.T) {
	t.Parallel()
	api := newFakeAPI(t)
	c := api.client(t)
	ctx := context.Background()

	walletID, err := c.OpenWallet(ctx, "0192f28f-0000-7000-8000-00000000aaaa", 1000)
	if err != nil {
		t.Fatalf("OpenWallet: %v", err)
	}
	op := load.Operation{ExternalID: "e-1", IdempotencyKey: "e-1", WalletID: walletID, PlayerID: "p", Kind: load.KindBet, Amount: 250}

	first, err := c.Submit(ctx, op)
	if err != nil || first.HTTPStatus != 200 || first.Status != "PROCESSED" || first.Replay {
		t.Fatalf("first submit = %+v, %v", first, err)
	}
	again, err := c.Submit(ctx, op)
	if err != nil || !again.Replay {
		t.Fatalf("resend = %+v, %v; want a replay", again, err)
	}
	balance, err := c.Balance(ctx, walletID)
	if err != nil || balance != 750 {
		t.Fatalf("Balance = %v, %v; want 7.50", balance, err)
	}
	status, found, err := c.Lookup(ctx, "e-1")
	if err != nil || !found || status != "PROCESSED" {
		t.Fatalf("Lookup = %q, %v, %v", status, found, err)
	}
	if _, found, err := c.Lookup(ctx, "never-sent"); err != nil || found {
		t.Fatalf("Lookup of an unknown ID = found %v, %v; want not found", found, err)
	}
	consistent, err := c.Reconcile(ctx, walletID)
	if err != nil || !consistent {
		t.Fatalf("Reconcile = %v, %v", consistent, err)
	}
}

func TestTokensAreFetchedOnceAndRenewedBeforeExpiry(t *testing.T) {
	t.Parallel()
	api := newFakeAPI(t)
	c := api.client(t)
	ctx := context.Background()
	walletID, err := c.OpenWallet(ctx, "p", 100)
	if err != nil {
		t.Fatalf("OpenWallet: %v", err)
	}
	for range 20 {
		if _, err := c.Balance(ctx, walletID); err != nil {
			t.Fatal(err)
		}
	}
	if n := api.tokens.Load(); n != 1 {
		t.Fatalf("fetched %d tokens for one client, want 1", n)
	}

	// A token that has less than the safety margin left is renewed.
	c.internal.now = func() time.Time { return time.Now().Add(299 * time.Second) }
	if _, err := c.Balance(ctx, walletID); err != nil {
		t.Fatal(err)
	}
	if n := api.tokens.Load(); n != 2 {
		t.Fatalf("fetched %d tokens, want a renewal", n)
	}
}

func TestAnErrorBodyIsAnAnswerNotAnError(t *testing.T) {
	t.Parallel()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			writeJSON(w, http.StatusOK, map[string]any{"access_token": "t", "expires_in": 300})
			return
		}
		w.Header().Set("Retry-After", "1")
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"code": "TIMEOUT", "error": "too slow"})
	}))
	t.Cleanup(api.Close)
	c := New(Config{BaseURL: api.URL, TokenURL: api.URL + "/token", ProviderID: "p"})

	outcome, err := c.Submit(context.Background(), load.Operation{ExternalID: "e", IdempotencyKey: "e", Kind: load.KindBet, Amount: 1})

	if err != nil {
		t.Fatalf("Submit returned %v for an answered request", err)
	}
	if outcome.HTTPStatus != http.StatusServiceUnavailable || outcome.Status != "" {
		t.Fatalf("outcome = %+v, want the 503 with no status", outcome)
	}
}

func TestSustainsTwoThousandPerSecondAgainstAFastServer(t *testing.T) {
	if testing.Short() {
		t.Skip("load test")
	}
	if raceEnabled {
		t.Skip("throughput assertion unreliable under -race's instrumentation overhead; see race_test.go")
	}
	api := newFakeAPI(t)
	c := api.client(t)

	run, err := load.Start(context.Background(), c,
		load.Spec{Scenario: load.ScenarioUniform, Rate: 2000, Duration: 2 * time.Second, Wallets: 20},
		load.Limits{MaxRate: 5000, MaxDuration: time.Minute, MaxWallets: 100, MaxInFlight: 4096})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	<-run.Done()
	result, _ := run.Result()

	if result.Completed < 3800 || result.Dropped != 0 || result.Errors != 0 {
		t.Fatalf("completed %d, dropped %d, errors %d; want at least 3,800 of 4,000 and no loss",
			result.Completed, result.Dropped, result.Errors)
	}
	if !result.Verification.Passed {
		t.Fatalf("verification %+v", result.Verification)
	}
}
