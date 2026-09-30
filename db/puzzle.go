package db

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
)

// Puzzle represents a stored puzzle record.
type Puzzle struct {
	BasePuzzleID    string // Stable content-derived identifier for the canonical base puzzle.
	Puzzle          string // Normalized 81-char puzzle string.
	Difficulty      string // Difficulty level name (easy/medium/hard/expert/evil).
	Score           int    // Total difficulty score.
	MaxTechnique    string // Highest-tier technique required.
	Source          string // Origin: "generated", "imported", or source name.
	SourceRef       string // Optional immutable record or upstream identifier.
	PlayCount       int    // Number of times selected for play.
	LastPlayedAt    string // SQLite timestamp of the latest selection, or empty.
	CompletionCount int    // Number of player-driven completions.
	LastCompletedAt string // SQLite timestamp of the latest completion, or empty.
}

// BasePuzzleID derives the stable catalog identifier from canonical puzzle content.
func BasePuzzleID(canonicalPuzzle string) string {
	sum := sha256.Sum256([]byte(canonicalPuzzle))
	return fmt.Sprintf("bp_%x", sum[:])
}

// RecordCompletion atomically records a completion for an existing canonical puzzle.
func (db *DB) RecordCompletion(puzzle string) (bool, error) {
	result, err := db.conn.Exec(`UPDATE base_puzzles SET completion_count = completion_count + 1, last_completed_at = CURRENT_TIMESTAMP WHERE canonical_puzzle = ?`, puzzle)
	if err != nil {
		return false, fmt.Errorf("record completion: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("record completion rows affected: %w", err)
	}
	return rows == 1, nil
}

// AcquireForPlay atomically selects and marks an exact-difficulty puzzle.
func (db *DB) AcquireForPlay(difficulty string) (*Puzzle, error) {
	row := db.conn.QueryRow(`
		UPDATE base_puzzles SET play_count = play_count + 1, last_played_at = CURRENT_TIMESTAMP
		WHERE canonical_puzzle = (SELECT canonical_puzzle FROM base_puzzles WHERE difficulty = ?
			ORDER BY play_count ASC, last_played_at ASC, RANDOM() LIMIT 1)
		RETURNING base_puzzle_id, canonical_puzzle, difficulty, score, max_technique, COALESCE((SELECT source FROM puzzle_provenance WHERE base_puzzle_id = base_puzzles.base_puzzle_id ORDER BY provenance_id LIMIT 1), ''),
			play_count, COALESCE(CAST(last_played_at AS TEXT), '')`, difficulty)
	return scanPuzzle(row, "acquire puzzle")
}

// MarkForPlay atomically records selection of a specific stored puzzle.
func (db *DB) MarkForPlay(puzzle string) (*Puzzle, error) {
	row := db.conn.QueryRow(`
		UPDATE base_puzzles SET play_count = play_count + 1, last_played_at = CURRENT_TIMESTAMP
		WHERE canonical_puzzle = ?
		RETURNING base_puzzle_id, canonical_puzzle, difficulty, score, max_technique, COALESCE((SELECT source FROM puzzle_provenance WHERE base_puzzle_id = base_puzzles.base_puzzle_id ORDER BY provenance_id LIMIT 1), ''),
			play_count, COALESCE(CAST(last_played_at AS TEXT), '')`, puzzle)
	return scanPuzzle(row, "mark puzzle played")
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPuzzle(row rowScanner, operation string) (*Puzzle, error) {
	var p Puzzle
	err := row.Scan(&p.BasePuzzleID, &p.Puzzle, &p.Difficulty, &p.Score, &p.MaxTechnique, &p.Source, &p.PlayCount, &p.LastPlayedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	return &p, nil
}

// InsertPuzzle stores a puzzle if it does not already exist.
// Returns true if the puzzle was inserted (new), false if it was a duplicate.
func (db *DB) InsertPuzzle(p Puzzle) (bool, error) {
	derivedID := BasePuzzleID(p.Puzzle)
	if p.BasePuzzleID == "" {
		p.BasePuzzleID = derivedID
	} else if p.BasePuzzleID != derivedID {
		return false, fmt.Errorf("base puzzle id does not match canonical puzzle")
	}
	tx, err := db.conn.Begin()
	if err != nil {
		return false, fmt.Errorf("begin puzzle insert: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.Exec(
		`INSERT OR IGNORE INTO base_puzzles (base_puzzle_id, canonical_puzzle, difficulty, score, max_technique)
		 VALUES (?, ?, ?, ?, ?)`,
		p.BasePuzzleID, p.Puzzle, p.Difficulty, p.Score, p.MaxTechnique,
	)
	if err != nil {
		return false, fmt.Errorf("insert base puzzle: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("base puzzle rows affected: %w", err)
	}
	if p.Source != "" {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO puzzle_provenance (base_puzzle_id, source, source_ref) VALUES (?, ?, ?)`, p.BasePuzzleID, p.Source, p.SourceRef); err != nil {
			return false, fmt.Errorf("insert puzzle provenance: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit puzzle insert: %w", err)
	}
	return rows > 0, nil
}

// ReplenishmentSeeds returns stable exact-grade catalog seeds without
// changing acquisition history.
func (db *DB) ReplenishmentSeeds(difficulty string) ([]Puzzle, error) {
	rows, err := db.conn.Query(`SELECT base_puzzle_id, canonical_puzzle, difficulty, score, max_technique
		FROM base_puzzles WHERE difficulty = ? ORDER BY base_puzzle_id`, difficulty)
	if err != nil {
		return nil, fmt.Errorf("query replenishment seeds: %w", err)
	}
	defer rows.Close()
	var seeds []Puzzle
	for rows.Next() {
		var seed Puzzle
		if err := rows.Scan(&seed.BasePuzzleID, &seed.Puzzle, &seed.Difficulty, &seed.Score, &seed.MaxTechnique); err != nil {
			return nil, fmt.Errorf("scan replenishment seed: %w", err)
		}
		seeds = append(seeds, seed)
	}
	return seeds, rows.Err()
}

// ContainsPuzzle reports whether canonical content already belongs to the catalog.
func (db *DB) ContainsPuzzle(canonicalPuzzle string) (bool, error) {
	var present int
	err := db.conn.QueryRow(`SELECT EXISTS(SELECT 1 FROM base_puzzles WHERE canonical_puzzle = ?)`, canonicalPuzzle).Scan(&present)
	if err != nil {
		return false, fmt.Errorf("check catalog puzzle: %w", err)
	}
	return present == 1, nil
}

// PublishPuzzleBatch inserts a completely validated replenishment batch in
// one transaction. A fully matching existing batch is an idempotent resume;
// any partial collision aborts without publishing another row.
func (db *DB) PublishPuzzleBatch(puzzles []Puzzle) (bool, error) {
	if len(puzzles) == 0 {
		return false, fmt.Errorf("replenishment batch is empty")
	}
	seen := make(map[string]struct{}, len(puzzles))
	for index := range puzzles {
		puzzle := &puzzles[index]
		if puzzle.Puzzle == "" || puzzle.Difficulty == "" || puzzle.MaxTechnique == "" || puzzle.Source == "" || puzzle.SourceRef == "" {
			return false, fmt.Errorf("replenishment puzzle %d is incomplete", index)
		}
		derivedID := BasePuzzleID(puzzle.Puzzle)
		if puzzle.BasePuzzleID == "" {
			puzzle.BasePuzzleID = derivedID
		} else if puzzle.BasePuzzleID != derivedID {
			return false, fmt.Errorf("replenishment puzzle %d has mismatched identity", index)
		}
		if _, duplicate := seen[puzzle.BasePuzzleID]; duplicate {
			return false, fmt.Errorf("replenishment batch contains duplicate %s", puzzle.BasePuzzleID)
		}
		seen[puzzle.BasePuzzleID] = struct{}{}
	}
	tx, err := db.conn.Begin()
	if err != nil {
		return false, fmt.Errorf("begin replenishment publication: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(puzzles)), ",")
	args := make([]any, len(puzzles))
	for index, puzzle := range puzzles {
		args[index] = puzzle.BasePuzzleID
	}
	var existing int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM base_puzzles WHERE base_puzzle_id IN (`+placeholders+`)`, args...).Scan(&existing); err != nil {
		return false, fmt.Errorf("check replenishment publication: %w", err)
	}
	if existing != 0 && existing != len(puzzles) {
		return false, fmt.Errorf("replenishment batch partially collides with the catalog")
	}
	if existing == len(puzzles) {
		for _, puzzle := range puzzles {
			var count int
			err := tx.QueryRow(`SELECT COUNT(*) FROM base_puzzles b JOIN puzzle_provenance p USING (base_puzzle_id)
				WHERE b.base_puzzle_id = ? AND b.canonical_puzzle = ? AND b.difficulty = ? AND b.score = ? AND b.max_technique = ?
				AND p.source = ? AND p.source_ref = ?`, puzzle.BasePuzzleID, puzzle.Puzzle, puzzle.Difficulty, puzzle.Score,
				puzzle.MaxTechnique, puzzle.Source, puzzle.SourceRef).Scan(&count)
			if err != nil {
				return false, fmt.Errorf("verify resumed replenishment batch: %w", err)
			}
			if count != 1 {
				return false, fmt.Errorf("existing replenishment batch does not match staged content")
			}
		}
		return false, nil
	}
	for _, puzzle := range puzzles {
		if _, err := tx.Exec(`INSERT INTO base_puzzles (base_puzzle_id, canonical_puzzle, difficulty, score, max_technique)
			VALUES (?, ?, ?, ?, ?)`, puzzle.BasePuzzleID, puzzle.Puzzle, puzzle.Difficulty, puzzle.Score, puzzle.MaxTechnique); err != nil {
			return false, fmt.Errorf("publish replenishment puzzle: %w", err)
		}
		if _, err := tx.Exec(`INSERT INTO puzzle_provenance (base_puzzle_id, source, source_ref) VALUES (?, ?, ?)`,
			puzzle.BasePuzzleID, puzzle.Source, puzzle.SourceRef); err != nil {
			return false, fmt.Errorf("publish replenishment provenance: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit replenishment publication: %w", err)
	}
	return true, nil
}

// GetRandom returns a random puzzle at the specified difficulty level,
// or nil if none exists.
func (db *DB) GetRandom(difficulty string) (*Puzzle, error) {
	row := db.conn.QueryRow(
		`SELECT b.base_puzzle_id, b.canonical_puzzle, b.difficulty, b.score, b.max_technique,
			COALESCE((SELECT source FROM puzzle_provenance p WHERE p.base_puzzle_id = b.base_puzzle_id ORDER BY provenance_id LIMIT 1), '')
		 FROM base_puzzles b
		 WHERE difficulty = ?
		 ORDER BY RANDOM()
		 LIMIT 1`,
		difficulty,
	)

	var p Puzzle
	err := row.Scan(&p.BasePuzzleID, &p.Puzzle, &p.Difficulty, &p.Score, &p.MaxTechnique, &p.Source)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get random puzzle: %w", err)
	}

	return &p, nil
}

// Stats holds per-difficulty puzzle counts.
type Stats struct {
	Total   int
	ByLevel map[string]int
}

// GetStats returns the total puzzle count and per-difficulty breakdown.
func (db *DB) GetStats() (*Stats, error) {
	rows, err := db.conn.Query(
		`SELECT difficulty, COUNT(*) FROM base_puzzles GROUP BY difficulty`,
	)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}
	defer rows.Close()

	stats := &Stats{ByLevel: make(map[string]int)}
	for rows.Next() {
		var level string
		var count int
		if err := rows.Scan(&level, &count); err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		stats.ByLevel[level] = count
		stats.Total += count
	}

	return stats, rows.Err()
}

// PlayStats is one aggregate row from a single statistics snapshot.
type PlayStats struct {
	Level                                         string
	Stored, NeverSelected, Selected, Acquisitions int
	Completed, Completions                        int
	LatestSelection, LatestCompletion             string
}

// PlayStatistics returns per-grade rows followed by an overall row from one read transaction.
func (db *DB) PlayStatistics(level string) ([]PlayStats, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin statistics snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	where, args := "", []any{}
	if level != "" {
		where, args = " WHERE difficulty = ?", append(args, level)
	}
	rows, err := tx.Query(`SELECT difficulty, COUNT(*),
		SUM(CASE WHEN play_count = 0 THEN 1 ELSE 0 END),
		SUM(CASE WHEN play_count > 0 THEN 1 ELSE 0 END), COALESCE(SUM(play_count), 0),
		SUM(CASE WHEN completion_count > 0 THEN 1 ELSE 0 END), COALESCE(SUM(completion_count), 0),
		COALESCE(CAST(MAX(last_played_at) AS TEXT), ''), COALESCE(CAST(MAX(last_completed_at) AS TEXT), '')
		FROM base_puzzles`+where+` GROUP BY difficulty ORDER BY difficulty`, args...)
	if err != nil {
		return nil, fmt.Errorf("query statistics: %w", err)
	}
	var result []PlayStats
	for rows.Next() {
		var row PlayStats
		if err := rows.Scan(&row.Level, &row.Stored, &row.NeverSelected, &row.Selected, &row.Acquisitions, &row.Completed, &row.Completions, &row.LatestSelection, &row.LatestCompletion); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan statistics: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	byLevel := make(map[string]PlayStats, len(result))
	for _, row := range result {
		byLevel[row.Level] = row
	}
	result = result[:0]
	levels := []string{"easy", "medium", "hard", "expert", "evil"}
	if level != "" {
		levels = []string{level}
	}
	for _, included := range levels {
		row := byLevel[included]
		row.Level = included
		result = append(result, row)
	}
	overall := PlayStats{Level: "overall"}
	err = tx.QueryRow(`SELECT COUNT(*),
		COALESCE(SUM(CASE WHEN play_count = 0 THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN play_count > 0 THEN 1 ELSE 0 END), 0), COALESCE(SUM(play_count), 0),
		COALESCE(SUM(CASE WHEN completion_count > 0 THEN 1 ELSE 0 END), 0), COALESCE(SUM(completion_count), 0),
		COALESCE(CAST(MAX(last_played_at) AS TEXT), ''), COALESCE(CAST(MAX(last_completed_at) AS TEXT), '')
		FROM base_puzzles`+where, args...).Scan(&overall.Stored, &overall.NeverSelected, &overall.Selected, &overall.Acquisitions, &overall.Completed, &overall.Completions, &overall.LatestSelection, &overall.LatestCompletion)
	if err != nil {
		return nil, fmt.Errorf("query overall statistics: %w", err)
	}
	result = append(result, overall)
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit statistics snapshot: %w", err)
	}
	return result, nil
}

// HistoryPreview reports the rows and counters selected for a reset.
type HistoryPreview struct {
	Rows, Acquisitions, Completions int
}

func (db *DB) PreviewHistoryReset(level string) (HistoryPreview, error) {
	where, args := "", []any{}
	if level != "" {
		where, args = " WHERE difficulty = ?", append(args, level)
	}
	var preview HistoryPreview
	err := db.conn.QueryRow(`SELECT COUNT(*), COALESCE(SUM(play_count), 0), COALESCE(SUM(completion_count), 0) FROM base_puzzles`+where, args...).Scan(&preview.Rows, &preview.Acquisitions, &preview.Completions)
	if err != nil {
		return preview, fmt.Errorf("preview history reset: %w", err)
	}
	return preview, nil
}

// ResetHistory atomically clears the requested history dimension.
func (db *DB) ResetHistory(history, level string) error {
	set := map[string]string{
		"acquisition": "play_count = 0, last_played_at = NULL",
		"completion":  "completion_count = 0, last_completed_at = NULL",
		"all":         "play_count = 0, last_played_at = NULL, completion_count = 0, last_completed_at = NULL",
	}[history]
	if set == "" {
		return fmt.Errorf("invalid history %q", history)
	}
	where, args := "", []any{}
	if level != "" {
		where, args = " WHERE difficulty = ?", append(args, level)
	}
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("begin history reset: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`UPDATE base_puzzles SET `+set+where, args...); err != nil {
		return fmt.Errorf("reset history: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit history reset: %w", err)
	}
	return nil
}
