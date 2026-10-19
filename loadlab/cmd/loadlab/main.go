// Command loadlab offers load to go-high-load-wallet on an operator's
// request: an open-model generator that seeds its own wallets, follows
// every answer, and checks the money when the load ends.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/auth"
	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/config"
	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/load"
	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/server"
	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/walletapi"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("loadlab stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	verifier, err := waitForVerifier(ctx, auth.Config{
		JWKSURL: cfg.JWKSURL, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, RefreshInterval: time.Hour,
	})
	if err != nil {
		return err
	}
	target := walletapi.New(walletapi.Config{
		BaseURL:          cfg.WalletAPIURL,
		TokenURL:         cfg.TokenURL,
		ProviderID:       cfg.ProviderID,
		ProviderClientID: cfg.ProviderClientID,
		ProviderSecret:   cfg.ProviderSecret,
		InternalClientID: cfg.InternalClientID,
		InternalSecret:   cfg.InternalSecret,
	})
	lab := server.NewLab(ctx, target, load.Limits{
		MaxRate: cfg.MaxRate, MaxDuration: cfg.MaxDuration, MaxWallets: cfg.MaxWallets, MaxInFlight: cfg.MaxInFlight,
	})

	// No write timeout: the event stream lasts as long as a run.
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.NewHandler(lab, verifier, cfg.AllowedOrigins),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errs := make(chan error, 1)
	go func() {
		slog.Info("loadlab listening", "addr", cfg.Addr, "walletApi", cfg.WalletAPIURL, "maxRate", cfg.MaxRate)
		errs <- srv.ListenAndServe()
	}()

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// waitForVerifier retries the first JWKS fetch for two minutes, the time
// Keycloak may take to start beside it.
func waitForVerifier(ctx context.Context, cfg auth.Config) (*auth.Verifier, error) {
	deadline := time.Now().Add(2 * time.Minute)
	for {
		verifier, err := auth.NewVerifier(ctx, cfg)
		if err == nil {
			return verifier, nil
		}
		if time.Now().After(deadline) {
			return nil, err
		}
		slog.Warn("waiting for the identity provider", "error", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
