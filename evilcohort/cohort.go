// Package evilcohort validates and selects the frozen Evil serving cohort.
package evilcohort

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const (
	Name               = "evil-v1"
	RuleVersion        = 1
	ManifestHash       = "65689133e5019f881923762ed74bbc66f682835ba89981ccb748454a6ed1b970"
	RepositoryCommit   = "d2f4a3c585f5cc0aa50b6d0bf09ed7e4c73e71b6"
	SolverConfigDigest = "40af00b6619ec7aa17baeb3b7ce741cf8a7b895a0bd6d0376bb357c462eedd26"
	EvidenceHash       = "b9692dbfe6d009e69b5e6f39d2082babfc718ad075bd009cc3e12b06a37b940c"
	MemberCount        = 2023
)

type Report struct {
	SchemaVersion      int    `json:"schema_version"`
	ManifestHash       string `json:"manifest_hash"`
	RepositoryCommit   string `json:"repository_commit"`
	SolverConfigDigest string `json:"solver_config_digest"`
}

type Evidence struct {
	SchemaVersion              int     `json:"schema_version"`
	PuzzleHash                 string  `json:"puzzle_hash"`
	Outcome                    string  `json:"outcome"`
	Difficulty                 string  `json:"difficulty"`
	OriginalRating             float64 `json:"original_rating"`
	AdvancedTechniqueDiversity int     `json:"advanced_technique_diversity"`
	AdvancedMoveDensity        float64 `json:"advanced_move_density"`
	EvilMoveCount              int     `json:"evil_move_count"`
}

type Selection struct {
	BasePuzzleIDs                            []string
	ExploratoryPopulation, HeldOutPopulation int
	ExploratoryMembers, HeldOutMembers       int
}

func Load(evidencePath, reportPath string) (Selection, error) {
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		return Selection{}, fmt.Errorf("read audit report: %w", err)
	}
	var report Report
	if err := json.Unmarshal(reportData, &report); err != nil {
		return Selection{}, fmt.Errorf("decode audit report: %w", err)
	}
	if report.SchemaVersion != 1 || report.ManifestHash != ManifestHash || report.RepositoryCommit != RepositoryCommit || report.SolverConfigDigest != SolverConfigDigest {
		return Selection{}, fmt.Errorf("audit report does not match the frozen Evil cohort evidence")
	}
	file, err := os.Open(evidencePath)
	if err != nil {
		return Selection{}, fmt.Errorf("open audit evidence: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	selection, err := selectEvidence(io.TeeReader(file, hash))
	if err != nil {
		return Selection{}, err
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != EvidenceHash {
		return Selection{}, fmt.Errorf("audit evidence SHA-256 is %s, want %s", got, EvidenceHash)
	}
	if selection.ExploratoryPopulation != 34389 || selection.HeldOutPopulation != 8628 || selection.ExploratoryMembers != 1604 || selection.HeldOutMembers != 419 || len(selection.BasePuzzleIDs) != MemberCount {
		return Selection{}, fmt.Errorf("audit evidence produced unexpected frozen counts: population %d/%d, members %d/%d", selection.ExploratoryPopulation, selection.HeldOutPopulation, selection.ExploratoryMembers, selection.HeldOutMembers)
	}
	return selection, nil
}

func selectEvidence(reader io.Reader) (Selection, error) {
	var result Selection
	seen := map[string]struct{}{}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		var evidence Evidence
		if err := json.Unmarshal(scanner.Bytes(), &evidence); err != nil {
			return Selection{}, fmt.Errorf("decode evidence line %d: %w", line, err)
		}
		if evidence.SchemaVersion != 1 || len(evidence.PuzzleHash) != 64 {
			return Selection{}, fmt.Errorf("invalid evidence line %d", line)
		}
		if evidence.Outcome != "solved" || evidence.Difficulty != "evil" {
			continue
		}
		heldOut, err := heldOut(evidence.PuzzleHash)
		if err != nil {
			return Selection{}, fmt.Errorf("evidence line %d: %w", line, err)
		}
		if heldOut {
			result.HeldOutPopulation++
		} else {
			result.ExploratoryPopulation++
		}
		if !eligible(evidence) {
			continue
		}
		id := "bp_" + evidence.PuzzleHash
		if _, duplicate := seen[id]; duplicate {
			return Selection{}, fmt.Errorf("duplicate eligible canonical puzzle %s", id)
		}
		seen[id] = struct{}{}
		result.BasePuzzleIDs = append(result.BasePuzzleIDs, id)
		if heldOut {
			result.HeldOutMembers++
		} else {
			result.ExploratoryMembers++
		}
	}
	if err := scanner.Err(); err != nil {
		return Selection{}, fmt.Errorf("read audit evidence: %w", err)
	}
	return result, nil
}

func eligible(e Evidence) bool {
	return e.OriginalRating >= 7.0 && e.AdvancedMoveDensity >= 1.0/6.0 && e.EvilMoveCount >= 2 && e.AdvancedTechniqueDiversity >= 3
}

func heldOut(puzzleHash string) (bool, error) {
	prefix, err := hex.DecodeString(puzzleHash[:16])
	if err != nil {
		return false, fmt.Errorf("invalid canonical puzzle hash")
	}
	var value uint64
	for _, part := range prefix {
		value = value<<8 | uint64(part)
	}
	return value%5 == 0, nil
}
