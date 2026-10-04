// Package oidcauth owns the short-lived Google OIDC login transaction.
// It keeps state, nonce, and PKCE validation separate from HTTP and durable
// application-session persistence.
package oidcauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultTransactionLifetime = 10 * time.Minute
	maximumTransactionLifetime = 15 * time.Minute
	tokenClockSkew             = time.Minute
)

var (
	ErrInvalidConfiguration = errors.New("invalid OIDC configuration")
	ErrInvalidReturnTo      = errors.New("invalid login return destination")
	ErrInvalidState         = errors.New("invalid or consumed login state")
	ErrExpiredTransaction   = errors.New("login transaction expired")
	ErrProvider             = errors.New("OIDC provider rejected the callback")
	ErrInvalidToken         = errors.New("invalid OIDC token")
)

// Claims are the verified identity attributes needed by the application.
// Email and name remain mutable profile attributes; Issuer and Subject are the
// only external identity key.
type Claims struct {
	Issuer        string
	Subject       string
	Audience      []string
	Nonce         string
	Email         string
	EmailVerified bool
	Name          string
	IssuedAt      time.Time
	ExpiresAt     time.Time
}

// Provider performs provider-specific authorization and token verification.
// Exchange must verify the authorization response signature and code exchange
// before returning claims.
type Provider interface {
	AuthorizationURL(state, nonce, codeChallenge string) string
	Exchange(context.Context, string, string) (Claims, error)
}

type Config struct {
	Provider            Provider
	Issuer              string
	ClientID            string
	ReturnPathPrefix    string
	TransactionLifetime time.Duration
	Now                 func() time.Time
	Random              io.Reader
}

type Start struct {
	AuthorizationURL string
	ExpiresAt        time.Time
}

type Result struct {
	Claims   Claims
	ReturnTo string
}

type transaction struct {
	nonce        string
	codeVerifier string
	returnTo     string
	expiresAt    time.Time
}

// Manager keeps bounded, single-use login transactions in process memory.
// Runtime wiring may replace process-local storage later without changing the
// validation boundary.
type Manager struct {
	provider Provider
	issuer   string
	clientID string
	prefix   string
	lifetime time.Duration
	now      func() time.Time
	random   io.Reader

	mu           sync.Mutex
	transactions map[[sha256.Size]byte]transaction
}

func New(config Config) (*Manager, error) {
	if config.Provider == nil || config.Issuer == "" || config.ClientID == "" {
		return nil, ErrInvalidConfiguration
	}
	if !validPrefix(config.ReturnPathPrefix) {
		return nil, fmt.Errorf("%w: return path prefix", ErrInvalidConfiguration)
	}
	lifetime := config.TransactionLifetime
	if lifetime == 0 {
		lifetime = defaultTransactionLifetime
	}
	if lifetime <= 0 || lifetime > maximumTransactionLifetime {
		return nil, fmt.Errorf("%w: transaction lifetime", ErrInvalidConfiguration)
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	random := config.Random
	if random == nil {
		random = rand.Reader
	}
	return &Manager{
		provider:     config.Provider,
		issuer:       config.Issuer,
		clientID:     config.ClientID,
		prefix:       config.ReturnPathPrefix,
		lifetime:     lifetime,
		now:          now,
		random:       random,
		transactions: make(map[[sha256.Size]byte]transaction),
	}, nil
}

// Begin creates one state/nonce/PKCE transaction and returns the provider URL.
func (m *Manager) Begin(returnTo string) (Start, error) {
	if !validReturnTo(returnTo, m.prefix) {
		return Start{}, ErrInvalidReturnTo
	}
	state, err := randomValue(m.random)
	if err != nil {
		return Start{}, fmt.Errorf("create login state: %w", err)
	}
	nonce, err := randomValue(m.random)
	if err != nil {
		return Start{}, fmt.Errorf("create login nonce: %w", err)
	}
	verifier, err := randomValue(m.random)
	if err != nil {
		return Start{}, fmt.Errorf("create PKCE verifier: %w", err)
	}
	challengeDigest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeDigest[:])
	now := m.now().UTC()
	expiresAt := now.Add(m.lifetime)

	m.mu.Lock()
	m.pruneExpiredLocked(now)
	m.transactions[stateDigest(state)] = transaction{
		nonce: nonce, codeVerifier: verifier, returnTo: returnTo, expiresAt: expiresAt,
	}
	m.mu.Unlock()

	return Start{
		AuthorizationURL: m.provider.AuthorizationURL(state, nonce, challenge),
		ExpiresAt:        expiresAt,
	}, nil
}

// Complete consumes state before exchanging the code, so callback retries or
// provider failures cannot replay a login transaction.
func (m *Manager) Complete(ctx context.Context, state, code string) (Result, error) {
	if state == "" || code == "" {
		return Result{}, ErrInvalidState
	}
	now := m.now().UTC()
	key := stateDigest(state)
	m.mu.Lock()
	tx, ok := m.transactions[key]
	if ok {
		delete(m.transactions, key)
	}
	m.pruneExpiredLocked(now)
	m.mu.Unlock()
	if !ok {
		return Result{}, ErrInvalidState
	}
	if !now.Before(tx.expiresAt) {
		return Result{}, ErrExpiredTransaction
	}

	claims, err := m.provider.Exchange(ctx, code, tx.codeVerifier)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrProvider, err)
	}
	if err := m.validateClaims(claims, tx.nonce, now); err != nil {
		return Result{}, err
	}
	return Result{Claims: claims, ReturnTo: tx.returnTo}, nil
}

func (m *Manager) validateClaims(claims Claims, nonce string, now time.Time) error {
	if claims.Issuer != m.issuer || claims.Subject == "" {
		return ErrInvalidToken
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != m.clientID {
		return ErrInvalidToken
	}
	if subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(nonce)) != 1 {
		return ErrInvalidToken
	}
	if claims.IssuedAt.IsZero() || claims.ExpiresAt.IsZero() || claims.ExpiresAt.Before(claims.IssuedAt) {
		return ErrInvalidToken
	}
	if claims.IssuedAt.After(now.Add(tokenClockSkew)) || !claims.ExpiresAt.After(now) {
		return ErrInvalidToken
	}
	return nil
}

func (m *Manager) pruneExpiredLocked(now time.Time) {
	for key, tx := range m.transactions {
		if !now.Before(tx.expiresAt) {
			delete(m.transactions, key)
		}
	}
}

func stateDigest(state string) [sha256.Size]byte {
	return sha256.Sum256([]byte(state))
}

func randomValue(source io.Reader) (string, error) {
	value := make([]byte, 32)
	if _, err := io.ReadFull(source, value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func validPrefix(prefix string) bool {
	if prefix == "/" {
		return true
	}
	return strings.HasPrefix(prefix, "/") && strings.HasSuffix(prefix, "/") &&
		!strings.HasPrefix(prefix, "//") && !strings.Contains(prefix, "\\") &&
		!strings.ContainsAny(prefix, "?#")
}

func validReturnTo(returnTo, prefix string) bool {
	if returnTo == "" || !strings.HasPrefix(returnTo, "/") || strings.HasPrefix(returnTo, "//") || strings.ContainsAny(returnTo, "\\#") {
		return false
	}
	parsed, err := url.ParseRequestURI(returnTo)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil || parsed.Fragment != "" || parsed.Opaque != "" {
		return false
	}
	return prefix == "/" || strings.HasPrefix(parsed.Path, prefix)
}
