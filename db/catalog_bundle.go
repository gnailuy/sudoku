package db

import (
	"database/sql"
	"fmt"
)

// CatalogBundleRelease is one immutable embedded catalog release.
type CatalogBundleRelease struct {
	Version            int
	ArtifactDigest     string
	SolverConfigDigest string
	Puzzles            []Puzzle
	Cohorts            []CatalogBundleCohort
}

// CatalogBundleCohort is one immutable serving-policy release included in a bundle.
type CatalogBundleCohort struct {
	Definition CohortDefinition
	Members    []string
}

// CatalogBundleResult describes an idempotent bundle check or application.
type CatalogBundleResult struct {
	Status          string
	Version         int
	PuzzlesAdded    int
	ProvenanceAdded int
	CohortsAdded    int
	MembersAdded    int
}

// ApplyCatalogBundle validates and atomically applies one additive release.
// Dry-run executes the same checks and rolls back every catalog change.
func (db *DB) ApplyCatalogBundle(bundle CatalogBundleRelease, dryRun bool) (result CatalogBundleResult, err error) {
	result.Version = bundle.Version
	if bundle.Version < 1 || bundle.ArtifactDigest == "" || bundle.SolverConfigDigest == "" {
		return result, fmt.Errorf("catalog bundle release metadata is incomplete")
	}
	tx, err := db.conn.Begin()
	if err != nil {
		return result, fmt.Errorf("begin catalog bundle: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var existingDigest string
	err = tx.QueryRow(`SELECT artifact_digest FROM catalog_bundle_releases WHERE bundle_version = ?`, bundle.Version).Scan(&existingDigest)
	if err == nil {
		if existingDigest != bundle.ArtifactDigest {
			return result, fmt.Errorf("catalog bundle version %d conflicts with applied digest %s", bundle.Version, existingDigest)
		}
		result.Status = "already-applied"
		return result, nil
	}
	if err != sql.ErrNoRows {
		return result, fmt.Errorf("inspect catalog bundle release: %w", err)
	}
	var newest sql.NullInt64
	if err := tx.QueryRow(`SELECT MAX(bundle_version) FROM catalog_bundle_releases`).Scan(&newest); err != nil {
		return result, fmt.Errorf("inspect latest catalog bundle release: %w", err)
	}
	if newest.Valid && int(newest.Int64) > bundle.Version {
		result.Status = "newer-present"
		return result, nil
	}

	seenPuzzles := make(map[string]struct{}, len(bundle.Puzzles))
	for index := range bundle.Puzzles {
		puzzle := bundle.Puzzles[index]
		derivedID := BasePuzzleID(puzzle.Puzzle)
		if puzzle.BasePuzzleID == "" {
			puzzle.BasePuzzleID = derivedID
		}
		if puzzle.BasePuzzleID != derivedID || puzzle.Puzzle == "" || puzzle.Difficulty == "" || puzzle.MaxTechnique == "" || puzzle.Source == "" || puzzle.SourceRef == "" {
			return result, fmt.Errorf("catalog bundle puzzle %d is incomplete or has mismatched identity", index)
		}
		if _, duplicate := seenPuzzles[puzzle.BasePuzzleID]; duplicate {
			return result, fmt.Errorf("catalog bundle contains duplicate %s", puzzle.BasePuzzleID)
		}
		seenPuzzles[puzzle.BasePuzzleID] = struct{}{}

		var canonical, difficulty, technique string
		var score int
		err := tx.QueryRow(`SELECT canonical_puzzle, difficulty, score, max_technique FROM base_puzzles WHERE base_puzzle_id = ?`, puzzle.BasePuzzleID).Scan(&canonical, &difficulty, &score, &technique)
		switch err {
		case nil:
			if canonical != puzzle.Puzzle || difficulty != puzzle.Difficulty || score != puzzle.Score || technique != puzzle.MaxTechnique {
				return result, fmt.Errorf("catalog bundle puzzle %s conflicts with immutable catalog metadata", puzzle.BasePuzzleID)
			}
		case sql.ErrNoRows:
			if _, err := tx.Exec(`INSERT INTO base_puzzles (base_puzzle_id, canonical_puzzle, difficulty, score, max_technique) VALUES (?, ?, ?, ?, ?)`, puzzle.BasePuzzleID, puzzle.Puzzle, puzzle.Difficulty, puzzle.Score, puzzle.MaxTechnique); err != nil {
				return result, fmt.Errorf("insert catalog bundle puzzle %s: %w", puzzle.BasePuzzleID, err)
			}
			result.PuzzlesAdded++
		default:
			return result, fmt.Errorf("inspect catalog bundle puzzle %s: %w", puzzle.BasePuzzleID, err)
		}
		provenance, err := tx.Exec(`INSERT OR IGNORE INTO puzzle_provenance (base_puzzle_id, source, source_ref) VALUES (?, ?, ?)`, puzzle.BasePuzzleID, puzzle.Source, puzzle.SourceRef)
		if err != nil {
			return result, fmt.Errorf("insert catalog bundle provenance %s: %w", puzzle.BasePuzzleID, err)
		}
		if rows, rowsErr := provenance.RowsAffected(); rowsErr != nil {
			return result, fmt.Errorf("count catalog bundle provenance %s: %w", puzzle.BasePuzzleID, rowsErr)
		} else {
			result.ProvenanceAdded += int(rows)
		}
	}

	seenCohorts := make(map[string]struct{}, len(bundle.Cohorts))
	for index, cohort := range bundle.Cohorts {
		definition := cohort.Definition
		if definition.Name == "" || definition.Difficulty == "" || definition.RuleVersion < 1 || definition.ManifestHash == "" || definition.RepositoryCommit == "" || definition.SolverConfigDigest == "" || definition.EvidenceHash == "" || definition.MemberCount != len(cohort.Members) || definition.MemberCount == 0 {
			return result, fmt.Errorf("catalog bundle cohort %d is incomplete", index)
		}
		if _, duplicate := seenCohorts[definition.Name]; duplicate {
			return result, fmt.Errorf("catalog bundle contains duplicate cohort %s", definition.Name)
		}
		seenCohorts[definition.Name] = struct{}{}
		var current CohortDefinition
		err := tx.QueryRow(`SELECT cohort_name, difficulty, rule_version, manifest_hash, repository_commit, solver_config_digest, evidence_hash, member_count FROM serving_cohorts WHERE cohort_name = ?`, definition.Name).Scan(&current.Name, &current.Difficulty, &current.RuleVersion, &current.ManifestHash, &current.RepositoryCommit, &current.SolverConfigDigest, &current.EvidenceHash, &current.MemberCount)
		switch err {
		case nil:
			if current != definition {
				return result, fmt.Errorf("catalog bundle cohort %s conflicts with immutable serving policy", definition.Name)
			}
		case sql.ErrNoRows:
			if _, err := tx.Exec(`INSERT INTO serving_cohorts (cohort_name, difficulty, rule_version, manifest_hash, repository_commit, solver_config_digest, evidence_hash, member_count) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, definition.Name, definition.Difficulty, definition.RuleVersion, definition.ManifestHash, definition.RepositoryCommit, definition.SolverConfigDigest, definition.EvidenceHash, definition.MemberCount); err != nil {
				return result, fmt.Errorf("insert catalog bundle cohort %s: %w", definition.Name, err)
			}
			result.CohortsAdded++
		default:
			return result, fmt.Errorf("inspect catalog bundle cohort %s: %w", definition.Name, err)
		}
		seenMembers := make(map[string]struct{}, len(cohort.Members))
		for _, member := range cohort.Members {
			if _, duplicate := seenMembers[member]; duplicate {
				return result, fmt.Errorf("catalog bundle cohort %s contains duplicate member %s", definition.Name, member)
			}
			seenMembers[member] = struct{}{}
			var memberDifficulty string
			if err := tx.QueryRow(`SELECT difficulty FROM base_puzzles WHERE base_puzzle_id = ?`, member).Scan(&memberDifficulty); err != nil || memberDifficulty != definition.Difficulty {
				return result, fmt.Errorf("catalog bundle cohort member %s is absent or not exact %s", member, definition.Difficulty)
			}
			inserted, err := tx.Exec(`INSERT OR IGNORE INTO serving_cohort_members (cohort_name, base_puzzle_id) VALUES (?, ?)`, definition.Name, member)
			if err != nil {
				return result, fmt.Errorf("insert catalog bundle cohort member %s: %w", member, err)
			}
			if rows, rowsErr := inserted.RowsAffected(); rowsErr != nil {
				return result, fmt.Errorf("count catalog bundle cohort member %s: %w", member, rowsErr)
			} else {
				result.MembersAdded += int(rows)
			}
		}
		var actual int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM serving_cohort_members WHERE cohort_name = ?`, definition.Name).Scan(&actual); err != nil || actual != definition.MemberCount {
			return result, fmt.Errorf("catalog bundle cohort %s has %d members, want %d", definition.Name, actual, definition.MemberCount)
		}
	}

	if _, err := tx.Exec(`INSERT INTO catalog_bundle_releases (bundle_version, artifact_digest, solver_config_digest, puzzle_count, cohort_count) VALUES (?, ?, ?, ?, ?)`, bundle.Version, bundle.ArtifactDigest, bundle.SolverConfigDigest, len(bundle.Puzzles), len(bundle.Cohorts)); err != nil {
		return result, fmt.Errorf("record catalog bundle release: %w", err)
	}
	if dryRun {
		result.Status = "would-apply"
		return result, nil
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit catalog bundle: %w", err)
	}
	result.Status = "applied"
	return result, nil
}
