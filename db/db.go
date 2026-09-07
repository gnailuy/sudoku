// Package db manages the SQLite puzzle database for storing, deduplicating,
// and querying Sudoku puzzles by difficulty.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite connection for puzzle storage.
type DB struct {
	conn *sql.DB
}

const (
	busyTimeoutMilliseconds  = 5000
	sqliteBusyCode           = 5
	journalModeRetryInterval = 10 * time.Millisecond
)

// Open opens (or creates) a SQLite database at the given path and runs
// schema migrations. Use ":memory:" for an in-memory database.
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", sqliteDataSource(path))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// One pooled connection keeps each handle's operation order explicit. The
	// DSN reapplies the busy timeout if database/sql replaces that connection.
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	// Enable WAL mode for better concurrent read performance. SQLite can
	// return SQLITE_BUSY before its busy handler while another connection is
	// changing journal mode, so keep this initialization retry bounded by the
	// same timeout as ordinary lock waits.
	if err := enableWAL(conn, path == ":memory:"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// migrate creates the puzzles table and applies additive schema changes.
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

	if _, err = conn.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS puzzles (
			puzzle         TEXT PRIMARY KEY,
			difficulty     TEXT NOT NULL,
			score          INTEGER NOT NULL,
			max_technique  TEXT NOT NULL,
			source         TEXT,
			play_count     INTEGER NOT NULL DEFAULT 0,
			last_played_at TIMESTAMP,
			completion_count INTEGER NOT NULL DEFAULT 0,
			last_completed_at TIMESTAMP,
			created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("create puzzles table: %w", err)
	}

	columns, err := tableColumns(conn, "puzzles")
	if err != nil {
		return err
	}
	additions := []struct {
		name string
		sql  string
	}{
		{"play_count", `ALTER TABLE puzzles ADD COLUMN play_count INTEGER NOT NULL DEFAULT 0`},
		{"last_played_at", `ALTER TABLE puzzles ADD COLUMN last_played_at TIMESTAMP`},
		{"completion_count", `ALTER TABLE puzzles ADD COLUMN completion_count INTEGER NOT NULL DEFAULT 0`},
		{"last_completed_at", `ALTER TABLE puzzles ADD COLUMN last_completed_at TIMESTAMP`},
	}
	for _, addition := range additions {
		if !columns[addition.name] {
			if _, err = conn.ExecContext(context.Background(), addition.sql); err != nil {
				return fmt.Errorf("add %s: %w", addition.name, err)
			}
		}
	}
	if _, err = conn.ExecContext(context.Background(), `CREATE INDEX IF NOT EXISTS puzzles_acquisition_idx ON puzzles (difficulty, play_count, last_played_at)`); err != nil {
		return fmt.Errorf("create acquisition index: %w", err)
	}
	if _, err = conn.ExecContext(context.Background(), `COMMIT`); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}
	return nil
}

type queryContext interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func tableColumns(queryer queryContext, table string) (map[string]bool, error) {
	rows, err := queryer.QueryContext(context.Background(), `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return nil, fmt.Errorf("inspect %s columns: %w", table, err)
	}
	defer rows.Close()
	columns := make(map[string]bool)
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, fmt.Errorf("scan %s columns: %w", table, err)
		}
		columns[name] = true
	}
	return columns, rows.Err()
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
	return fmt.Sprintf("%s%s_pragma=busy_timeout%%3d%d", path, separator, busyTimeoutMilliseconds)
}
