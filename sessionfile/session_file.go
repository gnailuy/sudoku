// Package sessionfile provides presentation-neutral durable session transport.
package sessionfile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gnailuy/sudoku/privatefs"
)

// MaxSize is the largest accepted serialized session.
const MaxSize int64 = 1 << 20

// ErrTooLarge identifies session input beyond MaxSize.
var ErrTooLarge = errors.New("session file is too large")

// Read loads a bounded serialized session from a platform-private file.
func Read(path string) ([]byte, error) {
	if err := privatefs.ValidateFile(path); err != nil {
		return nil, fmt.Errorf("validate private session %q: %w", path, err)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open session %q: %w", path, err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxSize+1))
	if err != nil {
		return nil, fmt.Errorf("read session %q: %w", path, err)
	}
	if int64(len(data)) > MaxSize {
		return nil, fmt.Errorf("read session %q: %w (maximum %d bytes)", path, ErrTooLarge, MaxSize)
	}
	return data, nil
}

// Write atomically replaces path with a platform-private serialized session.
func Write(path string, data []byte) (err error) {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".sudoku-session-*")
	if err != nil {
		return fmt.Errorf("create temporary session beside %q: %w", path, err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = temporary.Close(); _ = os.Remove(temporaryPath) }()
	if err = privatefs.ProtectFile(temporaryPath); err != nil {
		return fmt.Errorf("protect temporary session for %q: %w", path, err)
	}
	if _, err = temporary.Write(data); err != nil {
		return fmt.Errorf("write temporary session for %q: %w", path, err)
	}
	if err = temporary.Sync(); err != nil {
		return fmt.Errorf("flush temporary session for %q: %w", path, err)
	}
	if err = temporary.Close(); err != nil {
		return fmt.Errorf("close temporary session for %q: %w", path, err)
	}
	if err = os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace session %q: %w", path, err)
	}
	return nil
}
