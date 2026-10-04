package oidcauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

type fakeProvider struct {
	state, nonce, challenge string
	code, verifier          string
	claims                  Claims
	err                     error
}

func (p *fakeProvider) AuthorizationURL(state, nonce, challenge string) string {
	p.state, p.nonce, p.challenge = state, nonce, challenge
	values := url.Values{"state": {state}, "nonce": {nonce}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}}
	return "https://accounts.example/authorize?" + values.Encode()
}

func (p *fakeProvider) Exchange(_ context.Context, code, verifier string) (Claims, error) {
	p.code, p.verifier = code, verifier
	if p.err != nil {
		return Claims{}, p.err
	}
	return p.claims, nil
}

func newTestManager(t *testing.T, provider *fakeProvider, now *time.Time) *Manager {
	t.Helper()
	manager, err := New(Config{
		Provider: provider, Issuer: GoogleIssuer, ClientID: "client-id",
		ReturnPathPrefix: "/sudoku/", Now: func() time.Time { return *now },
		Random: strings.NewReader(strings.Repeat("a", 32) + strings.Repeat("b", 32) + strings.Repeat("c", 32)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func TestBeginAndCompleteValidatesStateNoncePKCEAndClaims(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	provider := &fakeProvider{}
	manager := newTestManager(t, provider, &now)
	start, err := manager.Begin("/sudoku/games?from=login")
	if err != nil {
		t.Fatal(err)
	}
	if start.AuthorizationURL == "" || !start.ExpiresAt.Equal(now.Add(defaultTransactionLifetime)) {
		t.Fatalf("unexpected start: %+v", start)
	}
	challenge := sha256.Sum256([]byte(base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("c", 32)))))
	if provider.challenge != base64.RawURLEncoding.EncodeToString(challenge[:]) {
		t.Fatalf("challenge = %q", provider.challenge)
	}
	provider.claims = Claims{
		Issuer: GoogleIssuer, Subject: "subject-1", Audience: []string{"client-id"}, Nonce: provider.nonce,
		Email: "profile@example.test", EmailVerified: true, Name: "Profile",
		IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
	}
	result, err := manager.Complete(context.Background(), provider.state, "code-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.ReturnTo != "/sudoku/games?from=login" || result.Claims.Subject != "subject-1" || provider.code != "code-1" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if provider.verifier != base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("c", 32))) {
		t.Fatalf("verifier = %q", provider.verifier)
	}
	if _, err := manager.Complete(context.Background(), provider.state, "code-1"); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("replayed callback error = %v", err)
	}
}

func TestCompleteConsumesStateOnProviderFailure(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	provider := &fakeProvider{err: errors.New("exchange failed")}
	manager := newTestManager(t, provider, &now)
	if _, err := manager.Begin("/sudoku/"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Complete(context.Background(), provider.state, "bad-code"); !errors.Is(err, ErrProvider) {
		t.Fatalf("provider error = %v", err)
	}
	if _, err := manager.Complete(context.Background(), provider.state, "bad-code"); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("replayed provider error = %v", err)
	}
}

func TestCompleteRejectsExpiredTransactionAndInvalidClaims(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Claims, *time.Time)
		want   error
	}{
		{"expired transaction", func(_ *Claims, now *time.Time) { *now = now.Add(defaultTransactionLifetime) }, ErrExpiredTransaction},
		{"issuer", func(c *Claims, _ *time.Time) { c.Issuer = "https://issuer.example" }, ErrInvalidToken},
		{"audience", func(c *Claims, _ *time.Time) { c.Audience = []string{"other"} }, ErrInvalidToken},
		{"multiple audience", func(c *Claims, _ *time.Time) { c.Audience = []string{"client-id", "other"} }, ErrInvalidToken},
		{"subject", func(c *Claims, _ *time.Time) { c.Subject = "" }, ErrInvalidToken},
		{"nonce", func(c *Claims, _ *time.Time) { c.Nonce = "wrong" }, ErrInvalidToken},
		{"future issued", func(c *Claims, _ *time.Time) { c.IssuedAt = c.IssuedAt.Add(3 * time.Minute) }, ErrInvalidToken},
		{"expired token", func(c *Claims, now *time.Time) { c.ExpiresAt = *now }, ErrInvalidToken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
			provider := &fakeProvider{}
			manager := newTestManager(t, provider, &now)
			if _, err := manager.Begin("/sudoku/"); err != nil {
				t.Fatal(err)
			}
			provider.claims = Claims{Issuer: GoogleIssuer, Subject: "subject", Audience: []string{"client-id"}, Nonce: provider.nonce, IssuedAt: now, ExpiresAt: now.Add(time.Hour)}
			tt.mutate(&provider.claims, &now)
			if _, err := manager.Complete(context.Background(), provider.state, "code"); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestBeginRejectsUnsafeReturnDestinations(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	for _, destination := range []string{"", "sudoku/", "//evil.example/path", "https://evil.example/sudoku/", "/other/", "/sudoku\\evil", "/sudoku/#fragment"} {
		t.Run(destination, func(t *testing.T) {
			manager := newTestManager(t, &fakeProvider{}, &now)
			if _, err := manager.Begin(destination); !errors.Is(err, ErrInvalidReturnTo) {
				t.Fatalf("Begin(%q) error = %v", destination, err)
			}
		})
	}
}

func TestNewRejectsUnsafeConfiguration(t *testing.T) {
	provider := &fakeProvider{}
	for _, config := range []Config{
		{},
		{Provider: provider, Issuer: GoogleIssuer, ClientID: "client", ReturnPathPrefix: "sudoku/"},
		{Provider: provider, Issuer: GoogleIssuer, ClientID: "client", ReturnPathPrefix: "/sudoku", TransactionLifetime: time.Hour},
	} {
		if _, err := New(config); !errors.Is(err, ErrInvalidConfiguration) {
			t.Fatalf("New(%+v) error = %v", config, err)
		}
	}
}
