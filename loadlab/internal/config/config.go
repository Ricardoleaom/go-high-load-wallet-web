// Package config reads the load lab's settings from the environment. The
// defaults point at the backend's Compose stack on this machine, with the
// development realm's bench clients.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr string

	WalletAPIURL string
	TokenURL     string
	JWKSURL      string
	JWTIssuer    string
	JWTAudience  string

	ProviderID       string
	ProviderClientID string
	ProviderSecret   string
	InternalClientID string
	InternalSecret   string

	AllowedOrigins []string

	MaxRate     int
	MaxDuration time.Duration
	MaxWallets  int
	MaxInFlight int
}

func Load() (Config, error) {
	realm := get("KEYCLOAK_REALM_URL", "http://localhost:8081/realms/wagering")
	cfg := Config{
		Addr:             get("LOADLAB_ADDR", "127.0.0.1:8090"),
		WalletAPIURL:     strings.TrimRight(get("WALLET_API_URL", "http://localhost:8080"), "/"),
		TokenURL:         get("TOKEN_URL", realm+"/protocol/openid-connect/token"),
		JWKSURL:          get("JWKS_URL", realm+"/protocol/openid-connect/certs"),
		JWTIssuer:        get("JWT_ISSUER", "http://localhost:8081/realms/wagering"),
		JWTAudience:      get("JWT_AUDIENCE", "wallet-api"),
		ProviderID:       get("PROVIDER_ID", "bench-provider"),
		ProviderClientID: get("PROVIDER_CLIENT_ID", "bench-provider"),
		ProviderSecret:   get("PROVIDER_CLIENT_SECRET", "bench-provider-secret"),
		InternalClientID: get("INTERNAL_CLIENT_ID", "bench-internal"),
		InternalSecret:   get("INTERNAL_CLIENT_SECRET", "bench-internal-secret"),
	}
	var errs []error
	collect := func(err error) {
		if err != nil {
			errs = append(errs, err)
		}
	}
	var err error
	cfg.AllowedOrigins, err = origins("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:4173,http://localhost:8088")
	collect(err)
	cfg.MaxRate, err = positiveInt("MAX_RATE", 5000)
	collect(err)
	cfg.MaxWallets, err = positiveInt("MAX_WALLETS", 10000)
	collect(err)
	cfg.MaxInFlight, err = positiveInt("MAX_IN_FLIGHT", 4096)
	collect(err)
	cfg.MaxDuration, err = duration("MAX_DURATION", 10*time.Minute)
	collect(err)
	return cfg, errors.Join(errs...)
}

func get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func positiveInt(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("%s %q is not a positive integer", key, raw)
	}
	return n, nil
}

func duration(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s %q is not a positive duration such as 10m", key, raw)
	}
	return d, nil
}

// origins reads a comma-separated list of scheme://host[:port] origins; a
// wildcard is refused, as in the wallet API.
func origins(key, fallback string) ([]string, error) {
	var list []string
	for _, raw := range strings.Split(get(key, fallback), ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.Path != "" || u.RawQuery != "" {
			return nil, fmt.Errorf("%s %q is not an origin such as http://localhost:5173", key, raw)
		}
		list = append(list, raw)
	}
	return list, nil
}
