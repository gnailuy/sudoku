// Package db manages the SQLite puzzle catalog, provenance, and play records.
package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ conn *sql.DB }

const (
	SchemaVersion            = 5
	busyTimeoutMilliseconds  = 5000
	sqliteBusyCode           = 5
	journalModeRetryInterval = 10 * time.Millisecond
)

var ErrRebuildRequired = errors.New("database schema is obsolete; run `sudoku db rebuild --db <path> --yes`")
var ErrEvilCohortUnavailable = errors.New("the approved Evil serving cohort is unavailable")

func Open(path string) (*DB, error) {
	conn, err := openConnection(path)
	if err != nil {
		return nil, err
	}
	database := &DB{conn: conn}
	if err := database.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return database, nil
}

func openConnection(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", sqliteDataSource(path))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	if err := enableWAL(conn, path == ":memory:"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	return conn, nil
}

func (db *DB) Close() error { return db.conn.Close() }

// migrate creates the current schema but deliberately refuses pre-catalog
// schemas. Their puzzle strings do not contain enough provenance to backfill
// the new identity boundary safely.
func (db *DB) migrate() (err error) {
	conn, err := db.conn.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Close()
	if _, err = conn.ExecContext(context.Background(), `BEGIN IMMEDIATE`); err != nil {
		return fmt.Errorf("begin migration: %w", err)
	}
	defer func() {
		if err != nil {
			_, _ = conn.ExecContext(context.Background(), `ROLLBACK`)
		}
	}()

	legacy, err := tableExists(conn, "puzzles")
	if err != nil {
		return err
	}
	current, err := tableExists(conn, "base_puzzles")
	if err != nil {
		return err
	}
	if legacy || current {
		var version int
		if scanErr := conn.QueryRowContext(context.Background(), `SELECT version FROM schema_metadata WHERE singleton = 1`).Scan(&version); scanErr != nil {
			return ErrRebuildRequired
		}
		if version == 3 && current {
			if err = migrateVersion3To4(conn); err != nil {
				return err
			}
			version = 4
		}
		if version == 4 && current {
			if err = migrateVersion4To5(conn); err != nil {
				return err
			}
			version = 5
		}
		if version != SchemaVersion {
			return ErrRebuildRequired
		}
	}
	if err = createSchema(conn); err != nil {
		return err
	}
	if _, err = conn.ExecContext(context.Background(), `COMMIT`); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}
	return nil
}

func createSchema(conn *sql.Conn) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS schema_metadata (singleton INTEGER PRIMARY KEY CHECK (singleton = 1), version INTEGER NOT NULL)`,
		`INSERT OR IGNORE INTO schema_metadata (singleton, version) VALUES (1, 5)`,
		`CREATE TABLE IF NOT EXISTS base_puzzles (
			base_puzzle_id TEXT PRIMARY KEY,
			canonical_puzzle TEXT NOT NULL UNIQUE,
			difficulty TEXT NOT NULL,
			score INTEGER NOT NULL,
			max_technique TEXT NOT NULL,
			play_count INTEGER NOT NULL DEFAULT 0,
			last_played_at TIMESTAMP,
			completion_count INTEGER NOT NULL DEFAULT 0,
			last_completed_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS puzzle_provenance (
			provenance_id INTEGER PRIMARY KEY AUTOINCREMENT,
			base_puzzle_id TEXT NOT NULL REFERENCES base_puzzles(base_puzzle_id) ON DELETE CASCADE,
			source TEXT NOT NULL,
			source_ref TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (base_puzzle_id, source, source_ref)
		)`,
		`CREATE TABLE IF NOT EXISTS serving_cohorts (
			cohort_name TEXT PRIMARY KEY,
			difficulty TEXT NOT NULL,
			rule_version INTEGER NOT NULL,
			manifest_hash TEXT NOT NULL,
			repository_commit TEXT NOT NULL,
			solver_config_digest TEXT NOT NULL,
			evidence_hash TEXT NOT NULL,
			member_count INTEGER NOT NULL CHECK (member_count > 0),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS serving_cohort_members (
			cohort_name TEXT NOT NULL REFERENCES serving_cohorts(cohort_name) ON DELETE CASCADE,
			base_puzzle_id TEXT NOT NULL REFERENCES base_puzzles(base_puzzle_id) ON DELETE CASCADE,
			PRIMARY KEY (cohort_name, base_puzzle_id)
		)`,
		`CREATE TABLE IF NOT EXISTS play_runs (
			play_run_id TEXT PRIMARY KEY,
			base_puzzle_id TEXT NOT NULL REFERENCES base_puzzles(base_puzzle_id),
			presented_puzzle TEXT NOT NULL,
			status TEXT NOT NULL CHECK (status IN ('active', 'completed', 'abandoned')),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			user_id TEXT PRIMARY KEY,
			profile_email TEXT NOT NULL DEFAULT '',
			display_name TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS external_identities (
			issuer TEXT NOT NULL,
			subject TEXT NOT NULL,
			user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			profile_email TEXT NOT NULL DEFAULT '',
			display_name TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (issuer, subject)
		)`,
		`CREATE TABLE IF NOT EXISTS web_sessions (
			web_session_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			verifier_digest BLOB NOT NULL UNIQUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			idle_expires_at TIMESTAMP NOT NULL,
			absolute_expires_at TIMESTAMP NOT NULL,
			rotated_from_id TEXT REFERENCES web_sessions(web_session_id),
			revoked_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS account_games (
			account_game_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			base_puzzle_id TEXT NOT NULL REFERENCES base_puzzles(base_puzzle_id),
			play_run_id TEXT NOT NULL UNIQUE REFERENCES play_runs(play_run_id),
			engine_state BLOB NOT NULL,
			revision INTEGER NOT NULL DEFAULT 0 CHECK (revision >= 0),
			actual_difficulty TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'in-progress' CHECK (status IN ('in-progress', 'invalid', 'solved')),
			elapsed_seconds INTEGER NOT NULL DEFAULT 0 CHECK (elapsed_seconds >= 0),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS guest_claims (
			claim_fingerprint BLOB PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			account_game_id TEXT NOT NULL UNIQUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS base_puzzles_acquisition_idx ON base_puzzles (difficulty, play_count, last_played_at)`,
		`CREATE INDEX IF NOT EXISTS puzzle_provenance_base_idx ON puzzle_provenance (base_puzzle_id)`,
		`CREATE INDEX IF NOT EXISTS serving_cohort_members_base_idx ON serving_cohort_members (base_puzzle_id, cohort_name)`,
		`CREATE INDEX IF NOT EXISTS play_runs_base_idx ON play_runs (base_puzzle_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS external_identities_user_idx ON external_identities (user_id)`,
		`CREATE INDEX IF NOT EXISTS web_sessions_user_idx ON web_sessions (user_id, revoked_at)`,
		`CREATE INDEX IF NOT EXISTS account_games_user_idx ON account_games (user_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS guest_claims_user_idx ON guest_claims (user_id, created_at)`,
	}
	for _, statement := range statements {
		if _, err := conn.ExecContext(context.Background(), statement); err != nil {
			return fmt.Errorf("create catalog schema: %w", err)
		}
	}
	return nil
}

func migrateVersion4To5(conn *sql.Conn) error {
	for _, statement := range []string{
		`CREATE TABLE serving_cohorts (
			cohort_name TEXT PRIMARY KEY,
			difficulty TEXT NOT NULL,
			rule_version INTEGER NOT NULL,
			manifest_hash TEXT NOT NULL,
			repository_commit TEXT NOT NULL,
			solver_config_digest TEXT NOT NULL,
			evidence_hash TEXT NOT NULL,
			member_count INTEGER NOT NULL CHECK (member_count > 0),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE serving_cohort_members (
			cohort_name TEXT NOT NULL REFERENCES serving_cohorts(cohort_name) ON DELETE CASCADE,
			base_puzzle_id TEXT NOT NULL REFERENCES base_puzzles(base_puzzle_id) ON DELETE CASCADE,
			PRIMARY KEY (cohort_name, base_puzzle_id)
		)`,
		`CREATE INDEX serving_cohort_members_base_idx ON serving_cohort_members (base_puzzle_id, cohort_name)`,
		`UPDATE schema_metadata SET version = 5 WHERE singleton = 1 AND version = 4`,
	} {
		if _, err := conn.ExecContext(context.Background(), statement); err != nil {
			return fmt.Errorf("upgrade catalog schema to version 5: %w", err)
		}
	}
	return nil
}

func migrateVersion3To4(conn *sql.Conn) error {
	for _, statement := range []string{
		`ALTER TABLE account_games ADD COLUMN status TEXT NOT NULL DEFAULT 'in-progress' CHECK (status IN ('in-progress', 'invalid', 'solved'))`,
		`ALTER TABLE account_games ADD COLUMN elapsed_seconds INTEGER NOT NULL DEFAULT 0 CHECK (elapsed_seconds >= 0)`,
	} {
		if _, err := conn.ExecContext(context.Background(), statement); err != nil {
			return fmt.Errorf("upgrade catalog schema to version 4: %w", err)
		}
	}
	rows, err := conn.QueryContext(context.Background(), `SELECT account_game_id, engine_state FROM account_games`)
	if err != nil {
		return fmt.Errorf("read account games for version 4: %w", err)
	}
	type update struct{ id, status string }
	updates := make([]update, 0)
	for rows.Next() {
		var id string
		var state []byte
		if err := rows.Scan(&id, &state); err != nil {
			rows.Close()
			return fmt.Errorf("scan account game for version 4: %w", err)
		}
		var payload struct {
			Puzzle  string `json:"puzzle"`
			Current struct {
				Values  string `json:"values"`
				Invalid string `json:"invalid"`
			} `json:"current"`
		}
		if err := json.Unmarshal(state, &payload); err != nil {
			rows.Close()
			return fmt.Errorf("decode account game for version 4: %w", err)
		}
		status := "in-progress"
		if strings.Trim(payload.Current.Invalid, ".0") != "" {
			status = "invalid"
		} else if nonEmptyCells(payload.Puzzle)+nonEmptyCells(payload.Current.Values) == 81 {
			status = "solved"
		}
		updates = append(updates, update{id: id, status: status})
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close account games for version 4: %w", err)
	}
	for _, item := range updates {
		if _, err := conn.ExecContext(context.Background(), `UPDATE account_games SET status = ? WHERE account_game_id = ?`, item.status, item.id); err != nil {
			return fmt.Errorf("backfill account game status for version 4: %w", err)
		}
	}
	if _, err := conn.ExecContext(context.Background(), `UPDATE schema_metadata SET version = 4 WHERE singleton = 1 AND version = 3`); err != nil {
		return fmt.Errorf("finish catalog schema version 4: %w", err)
	}
	return nil
}

func nonEmptyCells(board string) int {
	return len(strings.ReplaceAll(strings.ReplaceAll(board, ".", ""), "0", ""))
}

// Rebuild atomically discards the disposable development catalog and creates
// the current schema. The database file itself remains in place.
func Rebuild(path string) (err error) {
	conn, err := openConnection(path)
	if err != nil {
		return err
	}
	defer conn.Close()
	dedicated, err := conn.Conn(context.Background())
	if err != nil {
		return err
	}
	defer dedicated.Close()
	if _, err = dedicated.ExecContext(context.Background(), `BEGIN IMMEDIATE`); err != nil {
		return fmt.Errorf("begin rebuild: %w", err)
	}
	defer func() {
		if err != nil {
			_, _ = dedicated.ExecContext(context.Background(), `ROLLBACK`)
		}
	}()
	for _, table := range []string{"guest_claims", "account_games", "web_sessions", "external_identities", "users", "play_runs", "serving_cohort_members", "serving_cohorts", "puzzle_provenance", "base_puzzles", "puzzles", "schema_metadata"} {
		if _, err = dedicated.ExecContext(context.Background(), `DROP TABLE IF EXISTS `+table); err != nil {
			return fmt.Errorf("drop %s: %w", table, err)
		}
	}
	if err = createSchema(dedicated); err != nil {
		return err
	}
	if _, err = dedicated.ExecContext(context.Background(), `COMMIT`); err != nil {
		return fmt.Errorf("commit rebuild: %w", err)
	}
	return nil
}

func tableExists(queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, table string) (bool, error) {
	var count int
	if err := queryer.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
		return false, fmt.Errorf("inspect table %s: %w", table, err)
	}
	return count == 1, nil
}

func enableWAL(conn *sql.DB, inMemory bool) error {
	deadline := time.Now().Add(time.Duration(busyTimeoutMilliseconds) * time.Millisecond)
	for {
		var mode string
		err := conn.QueryRow(`PRAGMA journal_mode=WAL`).Scan(&mode)
		if err == nil {
			if strings.EqualFold(mode, "wal") || inMemory && strings.EqualFold(mode, "memory") {
				return nil
			}
			return fmt.Errorf("journal mode is %q", mode)
		}
		if !isSQLiteBusy(err) || time.Now().Add(journalModeRetryInterval).After(deadline) {
			return err
		}
		time.Sleep(journalModeRetryInterval)
	}
}

func isSQLiteBusy(err error) bool {
	var coded interface{ Code() int }
	return errors.As(err, &coded) && coded.Code()&0xff == sqliteBusyCode
}

func sqliteDataSource(path string) string {
	if path == ":memory:" {
		path = "file::memory:?mode=memory&cache=private"
	}
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%s_pragma=busy_timeout%%3d%d&_pragma=foreign_keys%%3don", filepath.Clean(path), separator, busyTimeoutMilliseconds)
}
