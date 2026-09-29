package db

import (
	"database/sql"
	"fmt"
)

// Provenance identifies one independently traceable source for a base puzzle.
type Provenance struct {
	Source    string
	SourceRef string
}

// ProvenanceFor returns every source attached to canonical puzzle content.
func (db *DB) ProvenanceFor(puzzle string) ([]Provenance, error) {
	rows, err := db.conn.Query(`SELECT p.source, p.source_ref
		FROM puzzle_provenance p JOIN base_puzzles b USING (base_puzzle_id)
		WHERE b.canonical_puzzle = ? ORDER BY p.provenance_id`, puzzle)
	if err != nil {
		return nil, fmt.Errorf("query puzzle provenance: %w", err)
	}
	defer rows.Close()
	var result []Provenance
	for rows.Next() {
		var provenance Provenance
		if err := rows.Scan(&provenance.Source, &provenance.SourceRef); err != nil {
			return nil, err
		}
		result = append(result, provenance)
	}
	return result, rows.Err()
}

// PlayRun stores presentation-specific state separately from catalog identity.
type PlayRun struct {
	ID              string
	BasePuzzleID    string
	PresentedPuzzle string
	Status          string
}

// InsertPlayRun starts one caller-identified play run for an existing base puzzle.
func (db *DB) InsertPlayRun(run PlayRun) error {
	if run.ID == "" || run.BasePuzzleID == "" || run.PresentedPuzzle == "" {
		return fmt.Errorf("play run identity, base puzzle, and presentation are required")
	}
	if run.Status == "" {
		run.Status = "active"
	}
	if run.Status != "active" {
		return fmt.Errorf("new play run status must be active")
	}
	_, err := db.conn.Exec(`INSERT INTO play_runs (play_run_id, base_puzzle_id, presented_puzzle, status) VALUES (?, ?, ?, ?)`, run.ID, run.BasePuzzleID, run.PresentedPuzzle, run.Status)
	if err != nil {
		return fmt.Errorf("insert play run: %w", err)
	}
	return nil
}

// UpdatePlayRunStatus closes or reopens one durable play-run record.
func (db *DB) UpdatePlayRunStatus(id, status string) (bool, error) {
	if status != "active" && status != "completed" && status != "abandoned" {
		return false, fmt.Errorf("invalid play run status %q", status)
	}
	result, err := db.conn.Exec(`UPDATE play_runs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE play_run_id = ?`, status, id)
	if err != nil {
		return false, fmt.Errorf("update play run: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}

// CompletePlayRun atomically closes one active play run and records exactly
// one completion against its stable base-puzzle identity.
func (db *DB) CompletePlayRun(id string) (bool, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return false, fmt.Errorf("begin play-run completion: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var basePuzzleID string
	err = tx.QueryRow(`UPDATE play_runs SET status = 'completed', updated_at = CURRENT_TIMESTAMP
		WHERE play_run_id = ? AND status = 'active' RETURNING base_puzzle_id`, id).Scan(&basePuzzleID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("complete play run: %w", err)
	}
	result, err := tx.Exec(`UPDATE base_puzzles SET completion_count = completion_count + 1,
		last_completed_at = CURRENT_TIMESTAMP WHERE base_puzzle_id = ?`, basePuzzleID)
	if err != nil {
		return false, fmt.Errorf("record play-run completion: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		if err != nil {
			return false, fmt.Errorf("record play-run completion rows affected: %w", err)
		}
		return false, fmt.Errorf("record play-run completion: base puzzle is absent")
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit play-run completion: %w", err)
	}
	return true, nil
}

// PlayRunByID reads one play run without conflating it with its base puzzle.
func (db *DB) PlayRunByID(id string) (*PlayRun, error) {
	var run PlayRun
	err := db.conn.QueryRow(`SELECT play_run_id, base_puzzle_id, presented_puzzle, status FROM play_runs WHERE play_run_id = ?`, id).Scan(&run.ID, &run.BasePuzzleID, &run.PresentedPuzzle, &run.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get play run: %w", err)
	}
	return &run, nil
}
