package catalogbundle

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/gnailuy/sudoku/solver"
)

func TestEmbeddedCatalogBundleIsBoundAndCurrent(t *testing.T) {
	release, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if release.Manifest.FormatVersion != 1 || release.Manifest.BundleVersion != 1 || release.Manifest.SolverConfigDigest != solver.ConfigDigest(solver.NewStore()) {
		t.Fatalf("unexpected embedded release: %+v", release)
	}
	if len(release.Manifest.Puzzles) != 0 || len(release.Manifest.Cohorts) != 0 {
		t.Fatalf("foundation release must not decide the official dataset: %+v", release.Manifest)
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
