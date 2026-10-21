//go:build !windows

package privatefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProtectAndValidateNativeObjects(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(directory, 0777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(directory, 0777); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDirectory(directory); err == nil {
		t.Fatal("broad directory permissions unexpectedly validated")
	}
	if err := ProtectDirectory(directory); err != nil {
		t.Fatalf("protect directory: %v", err)
	}
	if info, err := os.Stat(directory); err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("protected directory mode = %v, %v", info.Mode().Perm(), err)
	}

	file := filepath.Join(directory, "secret")
	if err := os.WriteFile(file, []byte("secret"), 0666); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0666); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFile(file); err == nil {
		t.Fatal("broad file permissions unexpectedly validated")
	}
	if err := ProtectFile(file); err != nil {
		t.Fatalf("protect file: %v", err)
	}
	if info, err := os.Stat(file); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("protected file mode = %v, %v", info.Mode().Perm(), err)
	}
}

func TestRejectsLinksAndWrongObjectTypes(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "directory")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	for name, check := range map[string]func() error{
		"file as directory":  func() error { return ValidateDirectory(file) },
		"directory as file":  func() error { return ValidateFile(directory) },
		"validate file link": func() error { return ValidateFile(link) },
		"protect file link":  func() error { return ProtectFile(link) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := check(); err == nil {
				t.Fatal("unsupported object unexpectedly accepted")
			}
		})
	}
}
