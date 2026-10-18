// Package catalogbundle validates and applies the embedded additive puzzle catalog.
package catalogbundle

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/solver"
)

//go:embed catalog.json
var embeddedCatalog []byte

//go:embed catalog.sha256
var embeddedDigest string

// Manifest is the reviewable, generated catalog artifact contract.
type Manifest struct {
	FormatVersion      int      `json:"format_version"`
	BundleVersion      int      `json:"bundle_version"`
	SolverConfigDigest string   `json:"solver_config_digest"`
	Puzzles            []Puzzle `json:"puzzles"`
	Cohorts            []Cohort `json:"cohorts"`
}

// Puzzle is one immutable canonical puzzle and its redistribution metadata.
type Puzzle struct {
	BasePuzzleID   string `json:"base_puzzle_id"`
	Puzzle         string `json:"puzzle"`
	Difficulty     string `json:"difficulty"`
	Score          int    `json:"score"`
	MaxTechnique   string `json:"max_technique"`
	Source         string `json:"source"`
	SourceRef      string `json:"source_ref"`
	License        string `json:"license"`
	Redistribution string `json:"redistribution"`
}

// Cohort is one separately approved immutable serving-policy definition.
type Cohort struct {
	Name               string   `json:"name"`
	Difficulty         string   `json:"difficulty"`
	RuleVersion        int      `json:"rule_version"`
	ManifestHash       string   `json:"manifest_hash"`
	RepositoryCommit   string   `json:"repository_commit"`
	SolverConfigDigest string   `json:"solver_config_digest"`
	EvidenceHash       string   `json:"evidence_hash"`
	Members            []string `json:"members"`
}

// Release is a validated manifest plus the exact artifact digest.
type Release struct {
	Manifest Manifest
	Digest   string
}

// Default returns the embedded release after verifying both artifact and solver configuration digests.
func Default() (Release, error) {
	// The embedded artifact was canonicalized by the reproducible generator and
	// is bound by its digest. Avoid repeating the expensive symmetry search on
	// every process start; Parse retains full validation for external artifacts.
	return parse(embeddedCatalog, strings.TrimSpace(embeddedDigest), solver.NewStore(), false)
}

// Parse validates a catalog artifact against its external digest and solver registry.
func Parse(data []byte, expectedDigest string, store solver.Store) (Release, error) {
	return parse(data, expectedDigest, store, true)
}

func parse(data []byte, expectedDigest string, store solver.Store, validateCanonical bool) (Release, error) {
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	if len(expectedDigest) != 64 || digest != expectedDigest {
		return Release{}, fmt.Errorf("catalog artifact digest is %s, want %s", digest, expectedDigest)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Release{}, fmt.Errorf("decode catalog artifact: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Release{}, fmt.Errorf("decode catalog artifact: multiple JSON values")
		}
		return Release{}, fmt.Errorf("decode catalog artifact: %w", err)
	}
	if manifest.FormatVersion != 1 || manifest.BundleVersion < 1 {
		return Release{}, fmt.Errorf("unsupported catalog artifact format or bundle version")
	}
	currentDigest := solver.ConfigDigest(store)
	if manifest.SolverConfigDigest != currentDigest {
		return Release{}, fmt.Errorf("catalog solver configuration is %s, current configuration is %s", manifest.SolverConfigDigest, currentDigest)
	}
	seenPuzzles := make(map[string]struct{}, len(manifest.Puzzles))
	for index, puzzle := range manifest.Puzzles {
		if len(puzzle.Puzzle) != 81 || db.BasePuzzleID(puzzle.Puzzle) != puzzle.BasePuzzleID || (validateCanonical && core.CanonicalPuzzle(puzzle.Puzzle) != puzzle.Puzzle) {
			return Release{}, fmt.Errorf("catalog puzzle %d is not canonical or has mismatched identity", index)
		}
		if !validDifficulty(puzzle.Difficulty) || puzzle.MaxTechnique == "" || puzzle.Source == "" || puzzle.SourceRef == "" || puzzle.License == "" || puzzle.Redistribution == "" {
			return Release{}, fmt.Errorf("catalog puzzle %d has incomplete immutable metadata", index)
		}
		if _, duplicate := seenPuzzles[puzzle.BasePuzzleID]; duplicate {
			return Release{}, fmt.Errorf("catalog contains duplicate %s", puzzle.BasePuzzleID)
		}
		seenPuzzles[puzzle.BasePuzzleID] = struct{}{}
	}
	seenCohorts := make(map[string]struct{}, len(manifest.Cohorts))
	for index, cohort := range manifest.Cohorts {
		if cohort.Name == "" || !validDifficulty(cohort.Difficulty) || cohort.RuleVersion < 1 || cohort.ManifestHash == "" || cohort.RepositoryCommit == "" || cohort.SolverConfigDigest == "" || cohort.EvidenceHash == "" || len(cohort.Members) == 0 {
			return Release{}, fmt.Errorf("catalog cohort %d has incomplete immutable metadata", index)
		}
		if _, duplicate := seenCohorts[cohort.Name]; duplicate {
			return Release{}, fmt.Errorf("catalog contains duplicate cohort %s", cohort.Name)
		}
		seenCohorts[cohort.Name] = struct{}{}
		seenMembers := make(map[string]struct{}, len(cohort.Members))
		for _, member := range cohort.Members {
			if _, duplicate := seenMembers[member]; duplicate {
				return Release{}, fmt.Errorf("catalog cohort %s contains duplicate member %s", cohort.Name, member)
			}
			seenMembers[member] = struct{}{}
		}
	}
	return Release{Manifest: manifest, Digest: digest}, nil
}

// Apply checks or atomically applies the validated release.
func (release Release) Apply(database *db.DB, dryRun bool) (db.CatalogBundleResult, error) {
	puzzles := make([]db.Puzzle, len(release.Manifest.Puzzles))
	for index, puzzle := range release.Manifest.Puzzles {
		puzzles[index] = db.Puzzle{BasePuzzleID: puzzle.BasePuzzleID, Puzzle: puzzle.Puzzle, Difficulty: puzzle.Difficulty, Score: puzzle.Score, MaxTechnique: puzzle.MaxTechnique, Source: puzzle.Source, SourceRef: puzzle.SourceRef}
	}
	cohorts := make([]db.CatalogBundleCohort, len(release.Manifest.Cohorts))
	for index, cohort := range release.Manifest.Cohorts {
		cohorts[index] = db.CatalogBundleCohort{Definition: db.CohortDefinition{Name: cohort.Name, Difficulty: cohort.Difficulty, RuleVersion: cohort.RuleVersion, ManifestHash: cohort.ManifestHash, RepositoryCommit: cohort.RepositoryCommit, SolverConfigDigest: cohort.SolverConfigDigest, EvidenceHash: cohort.EvidenceHash, MemberCount: len(cohort.Members)}, Members: cohort.Members}
	}
	return database.ApplyCatalogBundle(db.CatalogBundleRelease{Version: release.Manifest.BundleVersion, ArtifactDigest: release.Digest, SolverConfigDigest: release.Manifest.SolverConfigDigest, Puzzles: puzzles, Cohorts: cohorts}, dryRun)
}

func validDifficulty(value string) bool {
	switch value {
	case "easy", "medium", "hard", "expert", "evil":
		return true
	default:
		return false
	}
}
