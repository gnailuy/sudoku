package cmd

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gnailuy/sudoku/accountauth"
	"github.com/gnailuy/sudoku/accountgame"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/guestclaim"
	"github.com/gnailuy/sudoku/guestdoc"
	"github.com/gnailuy/sudoku/oidcauth"
	"github.com/gnailuy/sudoku/privatefs"
	"github.com/gnailuy/sudoku/recovery"
	"github.com/gnailuy/sudoku/solver"
	"github.com/gnailuy/sudoku/webapi"
)

const (
	maxWebConfigBytes    = 64 << 10
	managedGuestKeyID    = "local-v1"
	managedGuestKeyFile  = "guest-sealing.key"
	defaultGuestLifetime = 30 * 24 * time.Hour
)

type webConfigFile struct {
	Guest   *guestConfigFile   `json:"guest"`
	Account *accountConfigFile `json:"account"`
}

type guestConfigFile struct {
	ActiveKeyID     string            `json:"active_key_id"`
	Keys            map[string]string `json:"keys"`
	LifetimeSeconds int64             `json:"lifetime_seconds"`
}

type accountConfigFile struct {
	CSRFKey                string           `json:"csrf_key"`
	Google                 googleConfigFile `json:"google"`
	ReturnPathPrefix       string           `json:"return_path_prefix"`
	CookiePath             string           `json:"cookie_path"`
	SessionIdleSeconds     int64            `json:"session_idle_seconds"`
	SessionAbsoluteSeconds int64            `json:"session_absolute_seconds"`
}

type googleConfigFile struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
}

func loadWebRuntime(ctx context.Context, path, databasePath string, options game.Options) (*webapi.GuestRuntime, *webapi.AccountRuntime, func() error, error) {
	config, err := readWebConfig(path)
	if err != nil {
		return nil, nil, nil, err
	}
	if path != "" && config.Guest == nil {
		return nil, nil, nil, errors.New("--web-config account mode requires a complete guest section")
	}

	var documents *guestdoc.Sealer
	if config.Guest == nil {
		key, err := loadManagedGuestKey()
		if err != nil {
			return nil, nil, nil, err
		}
		documents, err = guestdoc.New(guestdoc.Config{ActiveKeyID: managedGuestKeyID, Keys: map[string][]byte{managedGuestKeyID: key}, Lifetime: defaultGuestLifetime})
		if err != nil {
			return nil, nil, nil, fmt.Errorf("configure managed guest documents: %w", err)
		}
	} else {
		documents, err = configuredGuestDocuments(*config.Guest)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	createGuest := guestCreateFunc(databasePath, options)
	guest := &webapi.GuestRuntime{Documents: documents, Options: options, Create: createGuest}
	if config.Account == nil {
		return guest, nil, func() error { return nil }, nil
	}
	accountConfig := *config.Account
	if err := validateAccountConfig(accountConfig); err != nil {
		return nil, nil, nil, err
	}
	csrfKey, err := decodeSecret(accountConfig.CSRFKey)
	if err != nil || len(csrfKey) < 32 {
		return nil, nil, nil, errors.New("--web-config account.csrf_key must contain at least 32 base64url bytes")
	}
	provider, err := oidcauth.NewGoogleProvider(ctx, oidcauth.GoogleConfig{ClientID: accountConfig.Google.ClientID, ClientSecret: accountConfig.Google.ClientSecret, RedirectURL: accountConfig.Google.RedirectURL})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("configure Google OIDC: %w", err)
	}
	login, err := oidcauth.New(oidcauth.Config{Provider: provider, Issuer: oidcauth.GoogleIssuer, ClientID: accountConfig.Google.ClientID, ReturnPathPrefix: accountConfig.ReturnPathPrefix})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("configure OIDC transactions: %w", err)
	}
	database, resolvedDatabasePath, err := openCatalogDatabase(databasePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open account database: %w", err)
	}
	closeDatabase := func() error { return database.Close() }
	authentication, err := accountauth.New(accountauth.Config{Login: login, Store: database, IdleLifetime: time.Duration(accountConfig.SessionIdleSeconds) * time.Second, AbsoluteLifetime: time.Duration(accountConfig.SessionAbsoluteSeconds) * time.Second})
	if err != nil {
		_ = database.Close()
		return nil, nil, nil, fmt.Errorf("configure account sessions: %w", err)
	}
	games, err := accountgame.New(accountgame.Config{Store: database, Options: options})
	if err != nil {
		_ = database.Close()
		return nil, nil, nil, err
	}
	claims, err := guestclaim.New(guestclaim.Config{Documents: documents, Store: database})
	if err != nil {
		_ = database.Close()
		return nil, nil, nil, err
	}
	account := &webapi.AccountRuntime{
		Guest: guest, OIDC: login, Auth: authentication, Games: games, Claims: claims,
		Store: database, CookiePath: accountConfig.CookiePath, CSRFKey: csrfKey,
		CreateOwned: func(kind, value string) (game.Game, string, string, webapi.Difficulty, error) {
			return createOwnedSession(database, createSessionRequest(resolvedDatabasePath, kind, value))
		},
	}
	return guest, account, closeDatabase, nil
}

func readWebConfig(path string) (webConfigFile, error) {
	if path == "" {
		return webConfigFile{}, nil
	}
	info, err := os.Lstat(path)
	if err != nil {
		return webConfigFile{}, fmt.Errorf("read --web-config: %w", err)
	}
	if err := privatefs.ValidateFile(path); err != nil {
		return webConfigFile{}, errors.New("--web-config must be a regular private file")
	}
	if info.Size() > maxWebConfigBytes {
		return webConfigFile{}, errors.New("--web-config exceeds 64 KiB")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return webConfigFile{}, fmt.Errorf("read --web-config: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var config webConfigFile
	if err := decoder.Decode(&config); err != nil {
		return webConfigFile{}, fmt.Errorf("decode --web-config: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return webConfigFile{}, errors.New("decode --web-config: multiple JSON values")
		}
		return webConfigFile{}, fmt.Errorf("decode --web-config: %w", err)
	}
	if config.Guest == nil && config.Account == nil {
		return webConfigFile{}, errors.New("--web-config must contain a guest section")
	}
	return config, nil
}

func configuredGuestDocuments(config guestConfigFile) (*guestdoc.Sealer, error) {
	if config.LifetimeSeconds <= 0 {
		return nil, errors.New("--web-config guest.lifetime_seconds must be positive")
	}
	keys := make(map[string][]byte, len(config.Keys))
	for keyID, encoded := range config.Keys {
		value, err := decodeSecret(encoded)
		if err != nil || len(value) != 32 {
			return nil, fmt.Errorf("--web-config guest key %q must contain 32 base64url bytes", keyID)
		}
		keys[keyID] = value
	}
	documents, err := guestdoc.New(guestdoc.Config{ActiveKeyID: config.ActiveKeyID, Keys: keys, Lifetime: time.Duration(config.LifetimeSeconds) * time.Second})
	if err != nil {
		return nil, fmt.Errorf("configure guest documents: %w", err)
	}
	return documents, nil
}

func validateAccountConfig(config accountConfigFile) error {
	if config.Google.ClientID == "" || config.Google.ClientSecret == "" || config.Google.RedirectURL == "" || config.SessionIdleSeconds <= 0 || config.SessionAbsoluteSeconds <= config.SessionIdleSeconds || !validApplicationPath(config.CookiePath) || !validApplicationPath(config.ReturnPathPrefix) {
		return errors.New("--web-config contains an incomplete or invalid account section")
	}
	return nil
}

func loadManagedGuestKey() ([]byte, error) {
	databasePath, err := defaultDBPath()
	if err != nil {
		return nil, fmt.Errorf("locate managed guest key: %w", err)
	}
	directory := filepath.Dir(databasePath)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, fmt.Errorf("create managed guest key directory: %w", err)
	}
	if err := privatefs.ProtectDirectory(directory); err != nil {
		return nil, fmt.Errorf("protect managed guest key directory: %w", err)
	}
	path := filepath.Join(directory, managedGuestKeyFile)
	if key, err := readManagedGuestKey(path); err == nil {
		return key, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate managed guest key: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".guest-sealing-key-*")
	if err != nil {
		return nil, fmt.Errorf("stage managed guest key: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := privatefs.ProtectFile(temporaryPath); err != nil {
		_ = temporary.Close()
		return nil, fmt.Errorf("protect managed guest key: %w", err)
	}
	if _, err := temporary.Write(key); err != nil {
		_ = temporary.Close()
		return nil, fmt.Errorf("write managed guest key: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return nil, fmt.Errorf("sync managed guest key: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return nil, fmt.Errorf("close managed guest key: %w", err)
	}
	if err := os.Link(temporaryPath, path); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("install managed guest key: %w", err)
		}
		return readManagedGuestKey(path)
	}
	return key, nil
}

func readManagedGuestKey(path string) ([]byte, error) {
	if err := privatefs.ValidateFile(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return nil, errors.New("managed guest key must be a regular private file")
	}
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read managed guest key: %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("managed guest key must contain exactly 32 bytes")
	}
	return key, nil
}

func guestCreateFunc(databasePath string, options game.Options) webapi.CreateGuestGameFunc {
	return func(kind, value string) (game.Game, string, webapi.Difficulty, error) {
		current, _, err := createSession(createSessionRequest(databasePath, kind, value), io.Discard, io.Discard)
		if err != nil {
			return game.Game{}, "", "", err
		}
		classification := solver.ClassifyPuzzle(solverStore, current.ProblemBoard())
		if classification.Outcome != solver.ClassificationSolved {
			return game.Game{}, "", "", errors.New("puzzle is not solvable by the strategy classifier")
		}
		canonical := normalizePuzzleForDB(solverStore, current.ProblemBoard())
		return current, db.BasePuzzleID(canonical), webapi.Difficulty(classification.Difficulty), nil
	}
}

func createSessionRequest(databasePath, kind, value string) sessionRequest {
	request := sessionRequest{dbPath: databasePath}
	if kind == "difficulty" {
		request.level = value
	} else {
		request.input = value
	}
	return request
}

func createOwnedSession(database *db.DB, request sessionRequest) (game.Game, string, string, webapi.Difficulty, error) {
	current, _, err := createSession(request, io.Discard, io.Discard)
	if err != nil {
		return game.Game{}, "", "", "", err
	}
	classification := solver.ClassifyPuzzle(solverStore, current.ProblemBoard())
	if classification.Outcome != solver.ClassificationSolved {
		return game.Game{}, "", "", "", errors.New("puzzle is not solvable by the strategy classifier")
	}
	canonical := normalizePuzzleForDB(solverStore, current.ProblemBoard())
	if _, err := database.InsertPuzzle(db.Puzzle{
		Puzzle: canonical, Difficulty: classification.Difficulty, Score: classification.Score,
		MaxTechnique: classification.MaxTechnique, Source: "account-play",
	}); err != nil {
		return game.Game{}, "", "", "", fmt.Errorf("persist account puzzle: %w", err)
	}
	runID, err := recovery.NewID()
	if err != nil {
		return game.Game{}, "", "", "", fmt.Errorf("create play run identity: %w", err)
	}
	baseID := db.BasePuzzleID(canonical)
	presented := current.ProblemBoard()
	if err := database.InsertPlayRun(db.PlayRun{
		ID: runID, BasePuzzleID: baseID, PresentedPuzzle: presented.ToString(),
	}); err != nil {
		return game.Game{}, "", "", "", fmt.Errorf("create durable play run: %w", err)
	}
	return current, baseID, runID, webapi.Difficulty(classification.Difficulty), nil
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
