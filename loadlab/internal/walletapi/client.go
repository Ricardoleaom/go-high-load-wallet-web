// Package walletapi talks to go-high-load-wallet's HTTP API with the
// realm's service clients: bench-internal to open, read and reconcile
// wallets, bench-provider to submit and look up wagers.
package walletapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/load"
)

type Config struct {
	BaseURL          string
	TokenURL         string
	ProviderID       string
	ProviderClientID string
	ProviderSecret   string
	InternalClientID string
	InternalSecret   string
	Currency         string // BRL when empty
}

// Client implements load.Target over HTTP.
type Client struct {
	base       string
	providerID string
	currency   string
	http       *http.Client
	provider   *tokenSource
	internal   *tokenSource
}

var _ load.Target = (*Client)(nil)

// New builds a client whose connection pool is sized for thousands of
// requests a second to one host: idle connections are kept, not closed
// and reopened under load.
func New(cfg Config) *Client {
	transport := &http.Transport{
		Proxy:               nil,
		DialContext:         (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:        8192,
		MaxIdleConnsPerHost: 8192,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
	}
	client := &http.Client{Transport: transport}
	currency := cfg.Currency
	if currency == "" {
		currency = "BRL"
	}
	return &Client{
		base:       cfg.BaseURL,
		providerID: cfg.ProviderID,
		currency:   currency,
		http:       client,
		provider:   newTokenSource(client, cfg.TokenURL, cfg.ProviderClientID, cfg.ProviderSecret),
		internal:   newTokenSource(client, cfg.TokenURL, cfg.InternalClientID, cfg.InternalSecret),
	}
}

type money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func (c *Client) OpenWallet(ctx context.Context, playerID string, initial load.Cents) (string, error) {
	var wallet struct {
		ID string `json:"id"`
	}
	body := map[string]any{"playerId": playerID, "initialBalance": money{initial.String(), c.currency}}
	if err := c.call(ctx, c.internal, http.MethodPost, "/wallets", nil, body, http.StatusCreated, &wallet); err != nil {
		return "", err
	}
	return wallet.ID, nil
}

func (c *Client) Submit(ctx context.Context, op load.Operation) (load.Outcome, error) {
	payload, err := json.Marshal(map[string]any{
		"providerId":            c.providerID,
		"externalTransactionId": op.ExternalID,
		"playerId":              op.PlayerID,
		"walletId":              op.WalletID,
		"roundId":               "loadlab",
		"gameId":                "loadlab",
		"kind":                  op.Kind,
		"money":                 money{op.Amount.String(), c.currency},
	})
	if err != nil {
		return load.Outcome{}, err
	}
	token, err := c.provider.token(ctx)
	if err != nil {
		return load.Outcome{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/wagering/transactions", bytes.NewReader(payload))
	if err != nil {
		return load.Outcome{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", op.IdempotencyKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return load.Outcome{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	var answer struct {
		Status           string `json:"status"`
		FailureCode      string `json:"failureCode"`
		IdempotentReplay bool   `json:"idempotentReplay"`
	}
	// An error body ({code, error}) decodes to no status: still an answer.
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil && !errors.Is(err, io.EOF) {
		return load.Outcome{}, fmt.Errorf("reading the answer: %w", err)
	}
	return load.Outcome{
		HTTPStatus:  resp.StatusCode,
		Status:      answer.Status,
		FailureCode: answer.FailureCode,
		Replay:      answer.IdempotentReplay,
	}, nil
}

func (c *Client) Lookup(ctx context.Context, externalID string) (string, bool, error) {
	var tx struct {
		Status string `json:"status"`
	}
	path := "/providers/" + url.PathEscape(c.providerID) + "/wagering/transactions/" + url.PathEscape(externalID)
	err := c.call(ctx, c.provider, http.MethodGet, path, nil, nil, http.StatusOK, &tx)
	var status *statusError
	if errors.As(err, &status) && status.code == http.StatusNotFound {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return tx.Status, true, nil
}

func (c *Client) Balance(ctx context.Context, walletID string) (load.Cents, error) {
	var wallet struct {
		Balance money `json:"balance"`
	}
	if err := c.call(ctx, c.internal, http.MethodGet, "/wallets/"+url.PathEscape(walletID), nil, nil, http.StatusOK, &wallet); err != nil {
		return 0, err
	}
	return load.ParseCents(wallet.Balance.Amount)
}

func (c *Client) Reconcile(ctx context.Context, walletID string) (bool, error) {
	var result struct {
		Consistent bool `json:"consistent"`
	}
	path := "/wallets/" + url.PathEscape(walletID) + "/reconciliation"
	if err := c.call(ctx, c.internal, http.MethodPost, path, nil, nil, http.StatusOK, &result); err != nil {
		return false, err
	}
	return result.Consistent, nil
}

type statusError struct {
	code int
	body string
}

func (e *statusError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.code, e.body) }

// call sends one request with the source's token and decodes the answer
// into out when it has the expected status.
func (c *Client) call(ctx context.Context, tokens *tokenSource, method, path string, query url.Values, body any, want int, out any) error {
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(raw)
	}
	target := c.base + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, target, payload)
	if err != nil {
		return err
	}
	token, err := tokens.token(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != want {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return &statusError{code: resp.StatusCode, body: string(bytes.TrimSpace(snippet))}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
