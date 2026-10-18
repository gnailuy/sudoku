package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gnailuy/sudoku/catalogbundle"
	"github.com/gnailuy/sudoku/db"
	"github.com/spf13/cobra"
)

func defaultDBPath() (string, error) {
	return defaultDBPathFor(runtime.GOOS, os.Getenv, os.UserHomeDir)
}

func defaultDBPathFor(goos string, getenv func(string) string, userHomeDir func() (string, error)) (string, error) {
	switch goos {
	case "linux":
		dataHome := getenv("XDG_DATA_HOME")
		if dataHome == "" {
			home, err := userHomeDir()
			if err != nil {
				return "", fmt.Errorf("determine Linux application data directory: %w", err)
			}
			if home == "" {
				return "", fmt.Errorf("determine Linux application data directory: home directory is empty")
			}
			dataHome = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(dataHome, "sudoku", "puzzles.db"), nil
	case "darwin":
		home, err := userHomeDir()
		if err != nil {
			return "", fmt.Errorf("determine macOS application data directory: %w", err)
		}
		if home == "" {
			return "", fmt.Errorf("determine macOS application data directory: home directory is empty")
		}
		return filepath.Join(home, "Library", "Application Support", "Sudoku", "puzzles.db"), nil
	case "windows":
		dataHome := getenv("LOCALAPPDATA")
		if dataHome == "" {
			return "", fmt.Errorf("determine Windows application data directory: LOCALAPPDATA is empty")
		}
		return filepath.Join(dataHome, "Sudoku", "puzzles.db"), nil
	default:
		return "", fmt.Errorf("no managed database location is defined for %s", goos)
	}
}

func resolveDatabasePath(path string) (string, error) {
	if path != "" {
		return path, nil
	}
	return defaultDBPath()
}

// openCatalogDatabase opens an explicit database unchanged, while managed
// default databases receive the embedded additive catalog release.
func newDatabaseBootstrapCommand() *cobra.Command {
	var path string
	var check bool
	command := &cobra.Command{
		Use:   "bootstrap",
		Short: "Apply the embedded additive puzzle catalog",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			resolved, err := resolveDatabasePath(path)
			if err != nil {
				return err
			}
			if resolved != ":memory:" {
				if check {
					if _, err := os.Stat(resolved); err != nil {
						return fmt.Errorf("check catalog database: %w", err)
					}
				} else if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
					return fmt.Errorf("create database directory: %w", err)
				}
			}
			database, err := db.Open(resolved)
			if err != nil {
				return err
			}
			defer database.Close()
			release, err := catalogbundle.Default()
			if err != nil {
				return err
			}
			result, err := release.Apply(database, check)
			if err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "Database: %s\nBundle version: %d\nStatus: %s\nPuzzles added: %d\nProvenance added: %d\nCohorts added: %d\nCohort members added: %d\n", resolved, result.Version, result.Status, result.PuzzlesAdded, result.ProvenanceAdded, result.CohortsAdded, result.MembersAdded)
			return nil
		},
	}
	command.Flags().StringVar(&path, "db", "", "Puzzle database path (defaults to the managed application data directory)")
	command.Flags().BoolVar(&check, "check", false, "Validate without applying catalog rows or recording the release")
	return command
}

func openCatalogDatabase(path string) (*db.DB, string, error) {
	managed := path == ""
	resolved, err := resolveDatabasePath(path)
	if err != nil {
		return nil, "", err
	}
	if resolved != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
			return nil, "", fmt.Errorf("create database directory: %w", err)
		}
	}
	database, err := db.Open(resolved)
	if err != nil {
		return nil, "", err
	}
	if managed {
		release, releaseErr := catalogbundle.Default()
		if releaseErr != nil {
			_ = database.Close()
			return nil, "", releaseErr
		}
		if _, applyErr := release.Apply(database, false); applyErr != nil {
			_ = database.Close()
			return nil, "", applyErr
		}
	}
	return database, resolved, nil
}
