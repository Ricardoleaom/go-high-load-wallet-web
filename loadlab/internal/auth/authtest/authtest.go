// Package authtest is a JWT issuer for tests: an RSA key, its JWKS over
// HTTP, and tokens shaped like the wagering realm's.
package authtest

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	Audience = "wallet-api"
	kid      = "test-key"
)

type Issuer struct {
	key    *rsa.PrivateKey
	server *httptest.Server
}

func NewIssuer(t testing.TB) *Issuer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	i := &Issuer{key: key}
	i.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": kid, "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	t.Cleanup(i.server.Close)
	return i
}

func (i *Issuer) URL() string     { return i.server.URL + "/realms/wagering" }
func (i *Issuer) JWKSURL() string { return i.server.URL + "/certs" }

// Token signs claims with the issuer, audience and a five-minute expiry
// added; a nil value removes a claim.
func (i *Issuer) Token(t testing.TB, claims jwt.MapClaims) string {
	t.Helper()
	all := jwt.MapClaims{"iss": i.URL(), "aud": Audience, "exp": time.Now().Add(5 * time.Minute).Unix(), "iat": time.Now().Unix()}
	for name, value := range claims {
		if value == nil {
			delete(all, name)
			continue
		}
		all[name] = value
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, all)
	token.Header["kid"] = kid
	signed, err := token.SignedString(i.key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}

func (i *Issuer) OperatorToken(t testing.TB) string {
	t.Helper()
	return i.Token(t, jwt.MapClaims{"preferred_username": "operator", "realm_access": map[string]any{"roles": []string{"operator"}}})
}

func (i *Issuer) ProviderToken(t testing.TB) string {
	t.Helper()
	return i.Token(t, jwt.MapClaims{"provider_id": "provider-a"})
}
