package walletapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// renewBefore is how long before expiry a token is replaced, so no request
// leaves with a token that expires on the way.
const renewBefore = 30 * time.Second

// tokenSource holds one client's client-credentials token and fetches a
// new one when it is about to expire, one fetch at a time.
type tokenSource struct {
	http     *http.Client
	tokenURL string
	clientID string
	secret   string
	now      func() time.Time

	mu      sync.Mutex
	current string
	expires time.Time
}

func newTokenSource(client *http.Client, tokenURL, clientID, secret string) *tokenSource {
	return &tokenSource{http: client, tokenURL: tokenURL, clientID: clientID, secret: secret, now: time.Now}
}

func (s *tokenSource) token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != "" && s.now().Add(renewBefore).Before(s.expires) {
		return s.current, nil
	}

	asked := s.now()
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {s.clientID},
		"client_secret": {s.secret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("token for %s: %w", s.clientID, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("token for %s: HTTP %d: %s", s.clientID, resp.StatusCode, body)
	}
	var answer struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil || answer.AccessToken == "" {
		return "", fmt.Errorf("token for %s: unreadable answer", s.clientID)
	}
	// Measured from when it was asked for, which is never later than when
	// the IdP started counting.
	s.current = answer.AccessToken
	s.expires = asked.Add(time.Duration(answer.ExpiresIn) * time.Second)
	return s.current, nil
}
