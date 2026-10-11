package db

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestRebuildWebStatePreservesCatalogAndCohort(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sudoku.db")
	database, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	puzzle := "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."
	baseID := BasePuzzleID(puzzle)
	if _, err := database.InsertPuzzle(Puzzle{BasePuzzleID: baseID, Puzzle: puzzle, Difficulty: "evil", Score: 1, MaxTechnique: "single", Source: "test"}); err != nil {
		t.Fatal(err)
	}
	definition := CohortDefinition{Name: "test", Difficulty: "evil", RuleVersion: 1, ManifestHash: "manifest", RepositoryCommit: "commit", SolverConfigDigest: "solver", EvidenceHash: "evidence", MemberCount: 1}
	if err := database.MaterializeServingCohort(definition, []string{baseID}); err != nil {
		t.Fatal(err)
	}
	if err := database.CreateUser(User{ID: "user-1"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := database.CreateWebSession(WebSession{ID: "session-1", UserID: "user-1", VerifierDigest: []byte("digest"), IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(2 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := database.conn.Exec(`ALTER TABLE application_sessions RENAME TO web_sessions`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.conn.Exec(`UPDATE schema_metadata SET version = 5 WHERE singleton = 1`); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := Open(path); !errors.Is(err, ErrWebRebuildRequired) {
		t.Fatalf("open error=%v, want %v", err, ErrWebRebuildRequired)
	}
	if err := RebuildWebState(path); err != nil {
		t.Fatal(err)
	}
	database, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if present, err := database.ContainsPuzzle(puzzle); err != nil || !present {
		t.Fatalf("catalog present=%v err=%v", present, err)
	}
	var cohorts, users, sessions int
	if err := database.conn.QueryRow(`SELECT COUNT(*) FROM serving_cohort_members WHERE cohort_name = 'test' AND base_puzzle_id = ?`, baseID).Scan(&cohorts); err != nil {
		t.Fatal(err)
	}
	if err := database.conn.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := database.conn.QueryRow(`SELECT COUNT(*) FROM application_sessions`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if cohorts != 1 || users != 0 || sessions != 0 {
		t.Fatalf("cohorts=%d users=%d sessions=%d", cohorts, users, sessions)
	}
}
