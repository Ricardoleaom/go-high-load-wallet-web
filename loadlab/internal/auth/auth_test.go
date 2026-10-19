package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/auth"
	"github.com/Ricardoleaom/go-high-load-wallet-web/loadlab/internal/auth/authtest"
)

func newVerifier(t *testing.T, issuer *authtest.Issuer) *auth.Verifier {
	t.Helper()
	v, err := auth.NewVerifier(context.Background(), auth.Config{
		JWKSURL: issuer.JWKSURL(), Issuer: issuer.URL(), Audience: authtest.Audience, RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	return v
}

func TestOperatorsAreAccepted(t *testing.T) {
	t.Parallel()
	issuer := authtest.NewIssuer(t)
	v := newVerifier(t, issuer)

	name, err := v.Operator(issuer.OperatorToken(t))
	if err != nil || name != "operator" {
		t.Fatalf("Operator = %q, %v; want operator", name, err)
	}
}

func TestEverythingElseIsRefused(t *testing.T) {
	t.Parallel()
	issuer := authtest.NewIssuer(t)
	other := authtest.NewIssuer(t)
	v := newVerifier(t, issuer)
	operator := jwt.MapClaims{"realm_access": map[string]any{"roles": []string{"operator"}}}

	for name, token := range map[string]string{
		"a provider":        issuer.ProviderToken(t),
		"another issuer":    other.OperatorToken(t),
		"another audience":  issuer.Token(t, jwt.MapClaims{"aud": "someone-else", "realm_access": operator["realm_access"]}),
		"expired":           issuer.Token(t, jwt.MapClaims{"exp": time.Now().Add(-time.Minute).Unix(), "realm_access": operator["realm_access"]}),
		"without an expiry": issuer.Token(t, jwt.MapClaims{"exp": nil, "realm_access": operator["realm_access"]}),
		"not a token":       "not-a-token",
		"the internal role": issuer.Token(t, jwt.MapClaims{"realm_access": map[string]any{"roles": []string{"internal"}}}),
	} {
		if _, err := v.Operator(token); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
