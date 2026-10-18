package db

import (
	"strings"
	"testing"
)

func TestCatalogBundleApplyIsAtomicAdditiveAndIdempotent(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	puzzle := strings.Repeat(".", 81)
	first := CatalogBundleRelease{Version: 1, ArtifactDigest: strings.Repeat("a", 64), SolverConfigDigest: "solver", Puzzles: []Puzzle{{Puzzle: puzzle, Difficulty: "easy", Score: 4, MaxTechnique: "Naked Single", Source: "official", SourceRef: "release-1"}}}
	preview, err := database.ApplyCatalogBundle(first, true)
	if err != nil || preview.Status != "would-apply" || preview.PuzzlesAdded != 1 || preview.ProvenanceAdded != 1 {
		t.Fatalf("preview = %+v, %v", preview, err)
	}
	stats, err := database.GetStats()
	if err != nil || stats.Total != 0 {
		t.Fatalf("dry run changed catalog: %+v, %v", stats, err)
	}
	applied, err := database.ApplyCatalogBundle(first, false)
	if err != nil || applied.Status != "applied" || applied.PuzzlesAdded != 1 || applied.ProvenanceAdded != 1 {
		t.Fatalf("apply = %+v, %v", applied, err)
	}
	repeated, err := database.ApplyCatalogBundle(first, false)
	if err != nil || repeated.Status != "already-applied" {
		t.Fatalf("repeat = %+v, %v", repeated, err)
	}
	selected, err := database.AcquireForPlay("easy")
	if err != nil || selected == nil || selected.PlayCount != 1 {
		t.Fatalf("selected = %+v, %v", selected, err)
	}
	second := CatalogBundleRelease{Version: 2, ArtifactDigest: strings.Repeat("b", 64), SolverConfigDigest: "solver", Puzzles: []Puzzle{{Puzzle: puzzle, Difficulty: "easy", Score: 4, MaxTechnique: "Naked Single", Source: "official", SourceRef: "release-2"}}}
	upgraded, err := database.ApplyCatalogBundle(second, false)
	if err != nil || upgraded.Status != "applied" || upgraded.PuzzlesAdded != 0 || upgraded.ProvenanceAdded != 1 {
		t.Fatalf("upgrade = %+v, %v", upgraded, err)
	}
	selected, err = database.AcquireForPlay("easy")
	if err != nil || selected == nil || selected.PlayCount != 2 {
		t.Fatalf("upgrade reset acquisition history: %+v, %v", selected, err)
	}
	provenance, err := database.ProvenanceFor(puzzle)
	if err != nil || len(provenance) != 2 {
		t.Fatalf("provenance = %+v, %v", provenance, err)
	}
}

func TestCatalogBundleConflictRollsBackWholeRelease(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	firstPuzzle := strings.Repeat(".", 81)
	if _, err := database.InsertPuzzle(Puzzle{Puzzle: firstPuzzle, Difficulty: "easy", Score: 1, MaxTechnique: "Naked Single", Source: "existing"}); err != nil {
		t.Fatal(err)
	}
	secondPuzzle := "1" + strings.Repeat(".", 80)
	release := CatalogBundleRelease{Version: 1, ArtifactDigest: strings.Repeat("c", 64), SolverConfigDigest: "solver", Puzzles: []Puzzle{
		{Puzzle: secondPuzzle, Difficulty: "easy", Score: 1, MaxTechnique: "Naked Single", Source: "official", SourceRef: "new"},
		{Puzzle: firstPuzzle, Difficulty: "hard", Score: 99, MaxTechnique: "X-Wing", Source: "official", SourceRef: "conflict"},
	}}
	if _, err := database.ApplyCatalogBundle(release, false); err == nil {
		t.Fatal("conflicting immutable metadata was accepted")
	}
	present, err := database.ContainsPuzzle(secondPuzzle)
	if err != nil || present {
		t.Fatalf("partial release committed: present=%v err=%v", present, err)
	}
}
