package config

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultsPointAtTheLocalBackend(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.WalletAPIURL != "http://localhost:8080" || cfg.MaxRate != 5000 || cfg.MaxDuration != 10*time.Minute ||
		cfg.TokenURL != "http://localhost:8081/realms/wagering/protocol/openid-connect/token" || len(cfg.AllowedOrigins) != 3 {
		t.Fatalf("defaults %+v", cfg)
	}
}

func TestInvalidValuesFailNamingTheVariable(t *testing.T) {
	for key, value := range map[string]string{
		"MAX_RATE":             "0",
		"MAX_IN_FLIGHT":        "lots",
		"MAX_DURATION":         "-1m",
		"CORS_ALLOWED_ORIGINS": "*",
	} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, value)
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("Load with %s=%q: %v", key, value, err)
			}
		})
	}
}
