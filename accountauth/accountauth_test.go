package accountauth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/oidcauth"
)

type fakeLoginCompleter struct {
	result oidcauth.Result
	err    error
}

func (fake *fakeLoginCompleter) Complete(_ context.Context, _, _ string) (oidcauth.Result, error) {
	return fake.result, fake.err
}

func newTestService(t *testing.T, database *db.DB, login *fakeLoginCompleter, now *time.Time, random []byte) *Service {
	t.Helper()
	service, err := New(Config{
		Login: login, Store: database, IdleLifetime: time.Hour, AbsoluteLifetime: 24 * time.Hour,
		Now: func() time.Time { return *now }, Random: bytes.NewReader(random),
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestCompleteCreatesDigestOnlySessionAndReusesProviderIdentity(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Date(2026, 10, 4, 17, 0, 0, 0, time.UTC)
	login := &fakeLoginCompleter{result: oidcauth.Result{
		Claims:   oidcauth.Claims{Issuer: oidcauth.GoogleIssuer, Subject: "google-subject", Email: "first@example.test", EmailVerified: true, Name: "First"},
		ReturnTo: "/sudoku/games",
	}}
	firstRandom := append(append(bytes.Repeat([]byte{1}, identifierBytes), bytes.Repeat([]byte{2}, identifierBytes)...), bytes.Repeat([]byte{3}, verifierBytes)...)
	first := newTestService(t, database, login, &now, firstRandom)
	firstResult, err := first.Complete(context.Background(), "state", "code", "")
	if err != nil {
		t.Fatal(err)
	}
	if firstResult.User.ProfileEmail != "first@example.test" || firstResult.ReturnTo != "/sudoku/games" || firstResult.Verifier == "" {
		t.Fatalf("first login = %+v", firstResult)
	}
	decodedVerifier, err := base64.RawURLEncoding.DecodeString(firstResult.Verifier)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(decodedVerifier)
	active, err := database.ActiveWebSession(digest[:], now)
	if err != nil || active == nil || active.UserID != firstResult.User.ID {
		t.Fatalf("active session = %+v, %v", active, err)
	}
	if raw, err := database.ActiveWebSession([]byte(firstResult.Verifier), now); err != nil || raw != nil {
		t.Fatalf("raw verifier was accepted as stored digest: %+v, %v", raw, err)
	}

	login.result.Claims.Email = "second@example.test"
	login.result.Claims.Name = "Second"
	now = now.Add(time.Minute)
	secondRandom := append(append(bytes.Repeat([]byte{4}, identifierBytes), bytes.Repeat([]byte{5}, identifierBytes)...), bytes.Repeat([]byte{6}, verifierBytes)...)
	second := newTestService(t, database, login, &now, secondRandom)
	secondResult, err := second.Complete(context.Background(), "state-2", "code-2", firstResult.Verifier)
	if err != nil {
		t.Fatal(err)
	}
	if secondResult.User.ID != firstResult.User.ID || secondResult.User.ProfileEmail != "second@example.test" || secondResult.User.DisplayName != "Second" {
		t.Fatalf("rotated login = %+v", secondResult)
	}
	if old, err := first.Authenticate(firstResult.Verifier); err != nil || old != nil {
		t.Fatalf("old session = %+v, %v", old, err)
	}
	now = now.Add(30 * time.Minute)
	current, err := second.Authenticate(secondResult.Verifier)
	if err != nil || current == nil || current.RotatedFromID != firstResult.SessionID || current.UserID != firstResult.User.ID || !current.IdleExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("current session = %+v, %v", current, err)
	}
	if resolved, err := database.UserByExternalIdentity(oidcauth.GoogleIssuer, "google-subject"); err != nil || resolved == nil || resolved.ID != firstResult.User.ID || resolved.ProfileEmail != "second@example.test" {
		t.Fatalf("resolved identity = %+v, %v", resolved, err)
	}
}

func TestCompleteDoesNotPersistUnverifiedProfileEmail(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	login := &fakeLoginCompleter{result: oidcauth.Result{Claims: oidcauth.Claims{
		Issuer: oidcauth.GoogleIssuer, Subject: "subject", Email: "unverified@example.test", EmailVerified: false, Name: "Profile",
	}}}
	random := bytes.Repeat([]byte{7}, 2*identifierBytes+verifierBytes)
	result, err := newTestService(t, database, login, &now, random).Complete(context.Background(), "state", "code", "not-a-valid-cookie")
	if err != nil {
		t.Fatal(err)
	}
	if result.User.ProfileEmail != "" || result.User.DisplayName != "Profile" {
		t.Fatalf("user = %+v", result.User)
	}
}

func TestAuthenticateAndLogoutRejectMalformedAndRevokeCurrentSession(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	login := &fakeLoginCompleter{result: oidcauth.Result{Claims: oidcauth.Claims{Issuer: oidcauth.GoogleIssuer, Subject: "subject"}}}
	service := newTestService(t, database, login, &now, bytes.Repeat([]byte{8}, 2*identifierBytes+verifierBytes))
	if session, err := service.Authenticate("malformed"); err != nil || session != nil {
		t.Fatalf("malformed authentication = %+v, %v", session, err)
	}
	if revoked, err := service.Logout("malformed"); err != nil || revoked {
		t.Fatalf("malformed logout = %v, %v", revoked, err)
	}
	result, err := service.Complete(context.Background(), "state", "code", "")
	if err != nil {
		t.Fatal(err)
	}
	if revoked, err := service.Logout(result.Verifier); err != nil || !revoked {
		t.Fatalf("logout = %v, %v", revoked, err)
	}
	if session, err := service.Authenticate(result.Verifier); err != nil || session != nil {
		t.Fatalf("revoked authentication = %+v, %v", session, err)
	}
}

func TestCompleteLeavesNoDurableStateWhenProviderFails(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	login := &fakeLoginCompleter{err: errors.New("provider failed")}
	service := newTestService(t, database, login, &now, bytes.Repeat([]byte{9}, 2*identifierBytes+verifierBytes))
	if _, err := service.Complete(context.Background(), "state", "code", ""); err == nil || !strings.Contains(err.Error(), "provider failed") {
		t.Fatalf("error = %v", err)
	}
	if user, err := database.UserByExternalIdentity(oidcauth.GoogleIssuer, "subject"); err != nil || user != nil {
		t.Fatalf("unexpected durable identity = %+v, %v", user, err)
	}
}

func TestNewValidatesSessionLifetimes(t *testing.T) {
	login := &fakeLoginCompleter{}
	for _, config := range []Config{
		{},
		{Login: login, Store: nil, AbsoluteLifetime: 24 * time.Hour},
		{Login: login, Store: &fakeStore{}, AbsoluteLifetime: 0},
		{Login: login, Store: &fakeStore{}, IdleLifetime: 24 * time.Hour, AbsoluteLifetime: 24 * time.Hour},
	} {
		if _, err := New(config); !errors.Is(err, ErrInvalidConfiguration) {
			t.Fatalf("New(%+v) error = %v", config, err)
		}
	}
}

type fakeStore struct{}

func (*fakeStore) RotateWebSessionForIdentity(db.WebSessionRotation) (db.User, db.WebSession, error) {
	return db.User{}, db.WebSession{}, nil
}
func (*fakeStore) ActiveWebSession([]byte, time.Time) (*db.WebSession, error) { return nil, nil }
func (*fakeStore) RevokeWebSession(string, string) (bool, error)              { return false, nil }
func (*fakeStore) RefreshActiveWebSession([]byte, time.Time, time.Duration) (*db.WebSession, error) {
	return nil, nil
}
