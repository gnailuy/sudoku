package catalogbundle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/evilcohort"
	"github.com/gnailuy/sudoku/solver"
)

func TestEmbeddedCatalogBundleIsBoundAndCurrent(t *testing.T) {
	release, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if release.Manifest.FormatVersion != 1 || release.Manifest.BundleVersion != 2 || release.Manifest.SolverConfigDigest != solver.ConfigDigest(solver.NewStore()) {
		t.Fatalf("unexpected embedded release: %+v", release)
	}
	counts := map[string]int{}
	for _, puzzle := range release.Manifest.Puzzles {
		counts[puzzle.Difficulty]++
	}
	if len(release.Manifest.Puzzles) != 3047 || counts["easy"] != 256 || counts["medium"] != 256 || counts["hard"] != 256 || counts["expert"] != 256 || counts["evil"] != evilcohort.MemberCount {
		t.Fatalf("unexpected official catalog counts: total=%d counts=%v", len(release.Manifest.Puzzles), counts)
	}
	if len(release.Manifest.Cohorts) != 1 || release.Manifest.Cohorts[0].Name != evilcohort.Name || len(release.Manifest.Cohorts[0].Members) != evilcohort.MemberCount {
		t.Fatalf("unexpected official cohort: %+v", release.Manifest.Cohorts)
	}
}

func TestCatalogBundleRejectsNoncanonicalExternalArtifact(t *testing.T) {
	release, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	manifest := release.Manifest
	manifest.Puzzles = append([]Puzzle(nil), manifest.Puzzles[:1]...)
	manifest.Cohorts = nil
	puzzle := &manifest.Puzzles[0]
	puzzle.Puzzle = puzzle.Puzzle[9:18] + puzzle.Puzzle[:9] + puzzle.Puzzle[18:]
	puzzle.BasePuzzleID = db.BasePuzzleID(puzzle.Puzzle)
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if _, err := Parse(data, hex.EncodeToString(sum[:]), solver.NewStore()); err == nil {
		t.Fatal("noncanonical external artifact was accepted")
	}
}

func TestCatalogBundleRejectsDigestAndSchemaDrift(t *testing.T) {
	if _, err := Parse([]byte(`{}`), strings.Repeat("0", 64), solver.NewStore()); err == nil {
		t.Fatal("digest mismatch was accepted")
	}
	data := []byte(`{"format_version":1,"bundle_version":1,"solver_config_digest":"wrong","puzzles":[],"cohorts":[]}`)
	sum := sha256.Sum256(data)
	if _, err := Parse(data, hex.EncodeToString(sum[:]), solver.NewStore()); err == nil {
		t.Fatal("solver configuration drift was accepted")
	}
}
