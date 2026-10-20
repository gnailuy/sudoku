//go:build !windows

// Package privatefs enforces platform-native private file and directory storage.
package privatefs

import (
	"errors"
	"os"
)

// ValidateFile requires a regular, non-symlink, owner-only file.
func ValidateFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return errors.New("not a regular owner-only file")
	}
	return nil
}

// ProtectFile restricts an existing regular file to its owner.
func ProtectFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("not a regular file")
	}
	if err := os.Chmod(path, 0600); err != nil {
		return err
	}
	return ValidateFile(path)
}

// ValidateDirectory requires a regular, non-symlink, owner-only directory.
func ValidateDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return errors.New("not a regular owner-only directory")
	}
	return nil
}

// ProtectDirectory restricts an existing directory to its owner.
func ProtectDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("not a regular directory")
	}
	if err := os.Chmod(path, 0700); err != nil {
		return err
	}
	return ValidateDirectory(path)
}
