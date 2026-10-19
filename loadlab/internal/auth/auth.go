// Package auth admits the console's operators: tokens from the wagering
// realm, for the wallet API's audience, carrying the operator role.
package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	JWKSURL         string
	Issuer          string
	Audience        string
	RefreshInterval time.Duration
}

var ErrNotOperator = errors.New("the token does not carry the operator role")

type Verifier struct {
	keyfunc keyfunc.Keyfunc
	options []jwt.ParserOption
}

type claims struct {
	jwt.RegisteredClaims
	PreferredUsername string `json:"preferred_username"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

// NewVerifier fetches the JWKS, failing if it cannot, and re-reads it every
// RefreshInterval for as long as ctx lives.
func NewVerifier(ctx context.Context, cfg Config) (*Verifier, error) {
	fail := false
	kf, err := keyfunc.NewDefaultOverrideCtx(ctx, []string{cfg.JWKSURL}, keyfunc.Override{
		NoErrorReturnFirstHTTPReq: &fail,
		RefreshInterval:           cfg.RefreshInterval,
		RateLimitWaitMax:          2 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS from %s: %w", cfg.JWKSURL, err)
	}
	return &Verifier{
		keyfunc: kf,
		options: []jwt.ParserOption{
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithIssuer(cfg.Issuer),
			jwt.WithAudience(cfg.Audience),
			jwt.WithExpirationRequired(),
		},
	}, nil
}

// Operator returns the operator's name when token is a valid operator
// token.
func (v *Verifier) Operator(token string) (string, error) {
	var c claims
	if _, err := jwt.ParseWithClaims(token, &c, v.keyfunc.Keyfunc, v.options...); err != nil {
		return "", err
	}
	if !slices.Contains(c.RealmAccess.Roles, "operator") {
		return "", ErrNotOperator
	}
	if c.PreferredUsername != "" {
		return c.PreferredUsername, nil
	}
	return c.Subject, nil
}
