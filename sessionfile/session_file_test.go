package sessionfile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gnailuy/sudoku/privatefs"
)

func TestWriteAndRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	if err := Write(path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, []byte("second")); err != nil {
		t.Fatal(err)
	}
	data, err := Read(path)
	if err != nil || string(data) != "second" {
		t.Fatalf("Read() = %q, %v", data, err)
	}
	if err := privatefs.ValidateFile(path); err != nil {
		t.Fatalf("session file is not private: %v", err)
	}
}

func TestReadRejectsOversizedInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.json")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", int(MaxSize+1))), 0600); err != nil {
		t.Fatal(err)
	}
	if err := privatefs.ProtectFile(path); err != nil {
		t.Fatalf("protect oversized session fixture: %v", err)
	}
	_, err := Read(path)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Read() error = %v, want ErrTooLarge", err)
	}
}

func TestFailedWritePreservesDestination(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "destination")
	if err := os.Mkdir(destination, 0700); err != nil {
		t.Fatal(err)
	}
	if err := Write(destination, []byte("replacement")); err == nil {
		t.Fatal("Write unexpectedly replaced a directory")
	}
	if info, err := os.Stat(destination); err != nil || !info.IsDir() {
		t.Fatalf("destination changed: info=%v err=%v", info, err)
	}
}
