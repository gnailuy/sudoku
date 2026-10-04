package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gnailuy/sudoku/accountauth"
	"github.com/gnailuy/sudoku/accountgame"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/guestclaim"
	"github.com/gnailuy/sudoku/guestdoc"
	"github.com/gnailuy/sudoku/oidcauth"
	"github.com/gnailuy/sudoku/solver"
	"github.com/gnailuy/sudoku/webapi"
)

const maxAccountsConfigBytes = 64 << 10

type accountsConfigFile struct {
	GuestKeyID             string            `json:"guest_key_id"`
	GuestKeys              map[string]string `json:"guest_keys"`
	GuestLifetimeSeconds   int64             `json:"guest_lifetime_seconds"`
	CSRFKey                string            `json:"csrf_key"`
	GoogleClientID         string            `json:"google_client_id"`
	GoogleClientSecret     string            `json:"google_client_secret"`
	GoogleRedirectURL      string            `json:"google_redirect_url"`
	ReturnPathPrefix       string            `json:"return_path_prefix"`
	CookiePath             string            `json:"cookie_path"`
	SessionIdleSeconds     int64             `json:"session_idle_seconds"`
	SessionAbsoluteSeconds int64             `json:"session_absolute_seconds"`
}

func loadAccountRuntime(ctx context.Context, path, databasePath string, options game.Options) (*webapi.AccountRuntime, func() error, error) {
	if path == "" {
		return nil, func() error { return nil }, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read --accounts-config: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0007 != 0 {
		return nil, nil, errors.New("--accounts-config must be a regular file without access for other users")
	}
	if info.Size() > maxAccountsConfigBytes {
		return nil, nil, errors.New("--accounts-config exceeds 64 KiB")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read --accounts-config: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var config accountsConfigFile
	if err := decoder.Decode(&config); err != nil {
		return nil, nil, fmt.Errorf("decode --accounts-config: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, nil, errors.New("decode --accounts-config: multiple JSON values")
		}
		return nil, nil, fmt.Errorf("decode --accounts-config: %w", err)
	}
	if config.GuestLifetimeSeconds <= 0 || config.SessionIdleSeconds <= 0 || config.SessionAbsoluteSeconds <= config.SessionIdleSeconds || !validApplicationPath(config.CookiePath) || !validApplicationPath(config.ReturnPathPrefix) {
		return nil, nil, errors.New("--accounts-config contains invalid lifetime or application-path policy")
	}
	keys := make(map[string][]byte, len(config.GuestKeys))
	for keyID, encoded := range config.GuestKeys {
		value, err := decodeSecret(encoded)
		if err != nil || len(value) != 32 {
			return nil, nil, fmt.Errorf("--accounts-config guest key %q must contain 32 base64url bytes", keyID)
		}
		keys[keyID] = value
	}
	csrfKey, err := decodeSecret(config.CSRFKey)
	if err != nil || len(csrfKey) < 32 {
		return nil, nil, errors.New("--accounts-config csrf_key must contain at least 32 base64url bytes")
	}
	documents, err := guestdoc.New(guestdoc.Config{ActiveKeyID: config.GuestKeyID, Keys: keys, Lifetime: time.Duration(config.GuestLifetimeSeconds) * time.Second})
	if err != nil {
		return nil, nil, fmt.Errorf("configure guest documents: %w", err)
	}
	provider, err := oidcauth.NewGoogleProvider(ctx, oidcauth.GoogleConfig{ClientID: config.GoogleClientID, ClientSecret: config.GoogleClientSecret, RedirectURL: config.GoogleRedirectURL})
	if err != nil {
		return nil, nil, fmt.Errorf("configure Google OIDC: %w", err)
	}
	login, err := oidcauth.New(oidcauth.Config{Provider: provider, Issuer: oidcauth.GoogleIssuer, ClientID: config.GoogleClientID, ReturnPathPrefix: config.ReturnPathPrefix})
	if err != nil {
		return nil, nil, fmt.Errorf("configure OIDC transactions: %w", err)
	}
	if databasePath == "" {
		databasePath = defaultDBPath()
	}
	database, err := db.Open(databasePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open account database: %w", err)
	}
	closeDatabase := func() error { return database.Close() }
	authentication, err := accountauth.New(accountauth.Config{Login: login, Store: database, IdleLifetime: time.Duration(config.SessionIdleSeconds) * time.Second, AbsoluteLifetime: time.Duration(config.SessionAbsoluteSeconds) * time.Second})
	if err != nil {
		_ = database.Close()
		return nil, nil, fmt.Errorf("configure account sessions: %w", err)
	}
	games, err := accountgame.New(accountgame.Config{Store: database, Options: options})
	if err != nil {
		_ = database.Close()
		return nil, nil, err
	}
	claims, err := guestclaim.New(guestclaim.Config{Documents: documents, Store: database})
	if err != nil {
		_ = database.Close()
		return nil, nil, err
	}
	classify := func(current game.Game) (webapi.Difficulty, string, error) {
		classification := solver.ClassifyPuzzle(solverStore, current.ProblemBoard())
		if classification.Outcome != solver.ClassificationSolved {
			return "", "", errors.New("puzzle is not solvable by the strategy classifier")
		}
		canonical := normalizePuzzleForDB(solverStore, current.ProblemBoard())
		return webapi.Difficulty(classification.Difficulty), db.BasePuzzleID(canonical), nil
	}
	createRequest := func(kind, value string) sessionRequest {
		request := sessionRequest{dbPath: databasePath}
		if kind == "difficulty" {
			request.level = value
		} else {
			request.input = value
		}
		return request
	}
	runtime := &webapi.AccountRuntime{
		Documents: documents, OIDC: login, Auth: authentication, Games: games, Claims: claims,
		Store: database, Options: options, CookiePath: config.CookiePath, CSRFKey: csrfKey,
		CreateGuest: func(kind, value string) (game.Game, string, webapi.Difficulty, error) {
			current, _, err := createSession(createRequest(kind, value), io.Discard, io.Discard)
			if err != nil {
				return game.Game{}, "", "", err
			}
			difficulty, baseID, err := classify(current)
			return current, baseID, difficulty, err
		},
		CreateOwned: func(kind, value string) (game.Game, string, string, webapi.Difficulty, error) {
			current, _, tracker, err := createTrackedSession(createRequest(kind, value), io.Discard, io.Discard)
			if err != nil {
				return game.Game{}, "", "", "", err
			}
			if tracker == nil || tracker.RunID() == "" {
				return game.Game{}, "", "", "", errors.New("unable to create durable play run")
			}
			difficulty, baseID, err := classify(current)
			return current, baseID, tracker.RunID(), difficulty, err
		},
	}
	return runtime, closeDatabase, nil
}

func decodeSecret(value string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err == nil {
		return decoded, nil
	}
	return base64.StdEncoding.DecodeString(value)
}

func validApplicationPath(value string) bool {
	return value == "/" || strings.HasPrefix(value, "/") && strings.HasSuffix(value, "/") && !strings.HasPrefix(value, "//") && !strings.ContainsAny(value, "\\?#")
}
