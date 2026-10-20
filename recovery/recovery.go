// Package recovery provides private, bounded TUI crash-recovery storage.
package recovery

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gnailuy/sudoku/privatefs"
	"github.com/gnailuy/sudoku/sessionfile"
)

const (
	formatVersion = 1
	retention     = 30 * 24 * time.Hour
)

// Record is one validated recovery record returned by discovery.
type Record struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Source    string
	Session   []byte
}

type envelope struct {
	Version   int       `json:"version"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Source    string    `json:"source,omitempty"`
	Session   []byte    `json:"session"`
}

// Store owns recovery records under one private directory.
type Store struct {
	directory string
	now       func() time.Time
}

// DefaultDirectory returns Sudoku's platform-native application-state location.
func DefaultDirectory() (string, error) {
	return defaultDirectoryFor(runtime.GOOS, os.Getenv, os.UserHomeDir)
}

func defaultDirectoryFor(goos string, getenv func(string) string, userHomeDir func() (string, error)) (string, error) {
	switch goos {
	case "windows":
		root := strings.TrimSpace(getenv("LOCALAPPDATA"))
		if root == "" {
			return "", errors.New("determine Windows recovery directory: LOCALAPPDATA is empty")
		}
		return filepath.Join(root, "Sudoku", "recovery"), nil
	case "darwin":
		home, err := userHomeDir()
		if err != nil {
			return "", fmt.Errorf("determine macOS recovery directory: %w", err)
		}
		if home == "" {
			return "", errors.New("determine macOS recovery directory: home directory is empty")
		}
		return filepath.Join(home, "Library", "Application Support", "Sudoku", "recovery"), nil
	case "linux":
		if state := strings.TrimSpace(getenv("XDG_STATE_HOME")); state != "" {
			return filepath.Join(state, "sudoku", "recovery"), nil
		}
		home, err := userHomeDir()
		if err != nil {
			return "", fmt.Errorf("determine Linux recovery directory: %w", err)
		}
		if home == "" {
			return "", errors.New("determine Linux recovery directory: home directory is empty")
		}
		return filepath.Join(home, ".local", "state", "sudoku", "recovery"), nil
	default:
		return "", fmt.Errorf("no recovery location is defined for %s", goos)
	}
}

// NewStore creates a recovery store handle without touching the filesystem.
func NewStore(directory string) Store { return Store{directory: directory, now: time.Now} }

// NewID returns a random opaque record identifier.
func NewID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("create recovery identifier: %w", err)
	}
	return hex.EncodeToString(value), nil
}

// Write atomically creates or replaces one private recovery record.
func (store Store) Write(id, source string, session []byte) error {
	if !validID(id) {
		return errors.New("invalid recovery identifier")
	}
	if !validSource(source) {
		return errors.New("invalid recovery source label")
	}
	if int64(len(session)) > sessionfile.MaxSize {
		return sessionfile.ErrTooLarge
	}
	if err := os.MkdirAll(store.directory, 0700); err != nil {
		return fmt.Errorf("create recovery directory: %w", err)
	}
	directoryInfo, err := os.Lstat(store.directory)
	if err != nil || directoryInfo.Mode()&os.ModeSymlink != 0 || !directoryInfo.IsDir() {
		return errors.New("recovery path is not a regular directory")
	}
	if err := privatefs.ProtectDirectory(store.directory); err != nil {
		return fmt.Errorf("protect recovery directory: %w", err)
	}
	now := store.now().UTC()
	created := now
	path := store.path(id)
	if existing, err := store.read(path); err == nil && existing.ID == id {
		created = existing.CreatedAt
	}
	data, err := json.Marshal(envelope{Version: formatVersion, ID: id, CreatedAt: created, UpdatedAt: now, Source: source, Session: session})
	if err != nil {
		return fmt.Errorf("encode recovery record: %w", err)
	}
	return sessionfile.Write(path, data)
}

// Delete removes one recovery record. A missing record is already deleted.
func (store Store) Delete(id string) error {
	if !validID(id) {
		return errors.New("invalid recovery identifier")
	}
	err := os.Remove(store.path(id))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("delete recovery record: %w", err)
	}
	return nil
}

// Discover returns recent valid records, newest first. Invalid and expired
// regular records are pruned. validate performs semantic session validation.
func (store Store) Discover(validate func([]byte) error) ([]Record, error) {
	directoryInfo, err := os.Lstat(store.directory)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect recovery directory: %w", err)
	}
	if directoryInfo.Mode()&os.ModeSymlink != 0 || !directoryInfo.IsDir() {
		return nil, errors.New("recovery path is not a regular directory")
	}
	if err := privatefs.ProtectDirectory(store.directory); err != nil {
		return nil, fmt.Errorf("protect recovery directory: %w", err)
	}
	entries, err := os.ReadDir(store.directory)
	if err != nil {
		return nil, fmt.Errorf("read recovery directory: %w", err)
	}
	now := store.now()
	cutoff := now.Add(-retention)
	records := make([]Record, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if filepath.Ext(name) != ".json" || !validID(strings.TrimSuffix(name, ".json")) {
			continue
		}
		path := filepath.Join(store.directory, name)
		info, infoErr := entry.Info()
		if infoErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			continue
		}
		if err := privatefs.ProtectFile(path); err != nil {
			continue
		}
		record, readErr := store.read(path)
		if readErr != nil || record.UpdatedAt.Before(cutoff) || record.UpdatedAt.After(now.Add(5*time.Minute)) || (validate != nil && validate(record.Session) != nil) {
			_ = os.Remove(path)
			continue
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].UpdatedAt.After(records[j].UpdatedAt) })
	return records, nil
}

func (store Store) read(path string) (Record, error) {
	if err := privatefs.ValidateFile(path); err != nil {
		return Record{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return Record{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Record{}, errors.New("recovery record is not a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return Record{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 2*sessionfile.MaxSize+1))
	if err != nil {
		return Record{}, err
	}
	if int64(len(data)) > 2*sessionfile.MaxSize {
		return Record{}, errors.New("recovery record is too large")
	}
	var decoded envelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		return Record{}, err
	}
	if decoded.Version != formatVersion || !validID(decoded.ID) || !validSource(decoded.Source) || filepath.Base(path) != decoded.ID+".json" || decoded.CreatedAt.IsZero() || decoded.UpdatedAt.IsZero() || decoded.UpdatedAt.Before(decoded.CreatedAt) || int64(len(decoded.Session)) > sessionfile.MaxSize {
		return Record{}, errors.New("invalid recovery record")
	}
	return Record{ID: decoded.ID, CreatedAt: decoded.CreatedAt, UpdatedAt: decoded.UpdatedAt, Source: decoded.Source, Session: decoded.Session}, nil
}

func (store Store) path(id string) string { return filepath.Join(store.directory, id+".json") }

func validID(id string) bool {
	if len(id) != 32 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func validSource(source string) bool {
	if !utf8.ValidString(source) || utf8.RuneCountInString(source) > 80 {
		return false
	}
	for _, value := range source {
		if unicode.IsControl(value) {
			return false
		}
	}
	return true
}
