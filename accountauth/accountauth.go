// Package accountauth turns verified OIDC callbacks into revocable application
// sessions without exposing provider tokens or raw browser verifiers to storage.
package accountauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/oidcauth"
)

const (
	defaultIdleLifetime = 30 * 24 * time.Hour
	identifierBytes     = 16
	verifierBytes       = 32
)

var ErrInvalidConfiguration = errors.New("invalid account authentication configuration")

type loginCompleter interface {
	Complete(context.Context, string, string) (oidcauth.Result, error)
}

type sessionStore interface {
	RotateWebSessionForIdentity(db.WebSessionRotation) (db.User, db.WebSession, error)
	ActiveWebSession([]byte, time.Time) (*db.WebSession, error)
	RefreshActiveWebSession([]byte, time.Time, time.Duration) (*db.WebSession, error)
	RevokeWebSession(string, string) (bool, error)
}

// Config binds the verified OIDC boundary to durable account-session storage.
// AbsoluteLifetime is explicit deployment policy; IdleLifetime defaults to 30 days.
type Config struct {
	Login            loginCompleter
	Store            sessionStore
	IdleLifetime     time.Duration
	AbsoluteLifetime time.Duration
	Random           io.Reader
	Now              func() time.Time
}

// Service owns opaque verifier generation, digest-only persistence, rotation,
// authentication, and current-session logout.
type Service struct {
	login            loginCompleter
	store            sessionStore
	idleLifetime     time.Duration
	absoluteLifetime time.Duration
	random           io.Reader
	now              func() time.Time
}

// Login is the browser-facing result of a completed application login.
type Login struct {
	User              db.User
	SessionID         string
	Verifier          string
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
	ReturnTo          string
}

func New(config Config) (*Service, error) {
	if config.Login == nil || config.Store == nil || config.AbsoluteLifetime <= 0 {
		return nil, ErrInvalidConfiguration
	}
	idleLifetime := config.IdleLifetime
	if idleLifetime == 0 {
		idleLifetime = defaultIdleLifetime
	}
	if idleLifetime <= 0 || idleLifetime >= config.AbsoluteLifetime {
		return nil, ErrInvalidConfiguration
	}
	random := config.Random
	if random == nil {
		random = rand.Reader
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		login: config.Login, store: config.Store, idleLifetime: idleLifetime,
		absoluteLifetime: config.AbsoluteLifetime, random: random, now: now,
	}, nil
}

// Complete consumes a verified provider callback and atomically rotates any
// presented application session into a fresh digest-only session.
func (service *Service) Complete(ctx context.Context, state, code, currentVerifier string) (Login, error) {
	candidateUserID, err := service.randomID("usr_")
	if err != nil {
		return Login{}, fmt.Errorf("create candidate user identity: %w", err)
	}
	sessionID, err := service.randomID("ws_")
	if err != nil {
		return Login{}, fmt.Errorf("create web session identity: %w", err)
	}
	rawVerifier := make([]byte, verifierBytes)
	if _, err := io.ReadFull(service.random, rawVerifier); err != nil {
		return Login{}, fmt.Errorf("create web session verifier: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(rawVerifier)
	verifierDigest := sha256.Sum256(rawVerifier)
	previousDigest, _ := verifierDigestFromToken(currentVerifier)

	providerResult, err := service.login.Complete(ctx, state, code)
	if err != nil {
		return Login{}, err
	}
	now := service.now().UTC()
	profileEmail := ""
	if providerResult.Claims.EmailVerified {
		profileEmail = providerResult.Claims.Email
	}
	user, session, err := service.store.RotateWebSessionForIdentity(db.WebSessionRotation{
		Identity: db.ExternalIdentity{
			Issuer: providerResult.Claims.Issuer, Subject: providerResult.Claims.Subject,
			ProfileEmail: profileEmail, DisplayName: providerResult.Claims.Name,
		},
		CandidateUserID: candidateUserID,
		Session: db.WebSession{
			ID: sessionID, VerifierDigest: verifierDigest[:],
			IdleExpiresAt: now.Add(service.idleLifetime), AbsoluteExpiresAt: now.Add(service.absoluteLifetime),
		},
		PreviousVerifierDigest: previousDigest,
		Now:                    now,
	})
	if err != nil {
		return Login{}, err
	}
	return Login{
		User: user, SessionID: session.ID, Verifier: verifier,
		IdleExpiresAt: session.IdleExpiresAt, AbsoluteExpiresAt: session.AbsoluteExpiresAt,
		ReturnTo: providerResult.ReturnTo,
	}, nil
}

// Authenticate resolves an opaque browser verifier without storing or logging it.
func (service *Service) Authenticate(verifier string) (*db.WebSession, error) {
	digest, ok := verifierDigestFromToken(verifier)
	if !ok {
		return nil, nil
	}
	return service.store.RefreshActiveWebSession(digest, service.now().UTC(), service.idleLifetime)
}

// Logout revokes only the currently authenticated application session.
func (service *Service) Logout(verifier string) (bool, error) {
	digest, ok := verifierDigestFromToken(verifier)
	if !ok {
		return false, nil
	}
	session, err := service.store.ActiveWebSession(digest, service.now().UTC())
	if err != nil || session == nil {
		return false, err
	}
	return service.store.RevokeWebSession(session.UserID, session.ID)
}

func (service *Service) randomID(prefix string) (string, error) {
	value := make([]byte, identifierBytes)
	if _, err := io.ReadFull(service.random, value); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(value), nil
}

func verifierDigestFromToken(token string) ([]byte, bool) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != verifierBytes {
		return nil, false
	}
	digest := sha256.Sum256(decoded)
	return digest[:], true
}
