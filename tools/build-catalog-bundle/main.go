// Command build-catalog-bundle reproduces the embedded starter catalog from
// the pinned public-domain Sudoku Exchange bank and frozen audit evidence.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gnailuy/sudoku/catalogbundle"
	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/evilcohort"
	"github.com/gnailuy/sudoku/solver"
)

const (
	diabolicalDigest = "08553d0c1145ea4d7c13008040f47ea8205d21fe1eaf8f4ab17a1a6981928b35"
	easyDigest       = "789aab6f52cc4588e0c3aa4269ad3282cfda7e1cc79ea693d16507decde3fc50"
	mediumDigest     = "c2a5f8fac99dcf215b25d46ece47f346375e51730b182de70a6f8f2107b2b38d"
	sourceCommit     = "d8c8ebaee0c08c412cfba96af1923dfa61c83317"
	perGrade         = 256
)

type auditEvidence struct {
	SchemaVersion int    `json:"schema_version"`
	ID            string `json:"id"`
	PuzzleHash    string `json:"puzzle_hash"`
	Outcome       string `json:"outcome"`
	Difficulty    string `json:"difficulty"`
	Score         int    `json:"score"`
	MaxTechnique  string `json:"max_technique"`
}

func main() {
	sourcePath := flag.String("source", "", "pinned diabolical.txt path")
	easySourcePath := flag.String("easy-source", "", "pinned easy.txt path")
	mediumSourcePath := flag.String("medium-source", "", "pinned medium.txt path")
	evidencePath := flag.String("evidence", "", "frozen full-catalog evidence.jsonl path")
	reportPath := flag.String("report", "", "frozen full-catalog report.json path")
	outputPath := flag.String("output", "catalogbundle/catalog.json", "output manifest path")
	flag.Parse()
	if *sourcePath == "" || *easySourcePath == "" || *mediumSourcePath == "" || *evidencePath == "" || *reportPath == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*sourcePath, *easySourcePath, *mediumSourcePath, *evidencePath, *reportPath, *outputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(sourcePath, easySourcePath, mediumSourcePath, evidencePath, reportPath, outputPath string) error {
	source, err := loadSource(sourcePath, diabolicalDigest)
	if err != nil {
		return err
	}
	cohort, err := evilcohort.Load(evidencePath, reportPath)
	if err != nil {
		return fmt.Errorf("load frozen Evil cohort: %w", err)
	}
	evilMembers := make(map[string]struct{}, len(cohort.BasePuzzleIDs))
	for _, id := range cohort.BasePuzzleIDs {
		evilMembers[id] = struct{}{}
	}

	evidence, err := os.Open(evidencePath)
	if err != nil {
		return fmt.Errorf("open audit evidence: %w", err)
	}
	defer evidence.Close()
	counts := map[string]int{}
	puzzles := make([]catalogbundle.Puzzle, 0, 4*perGrade+len(evilMembers))
	seenPuzzles := make(map[string]struct{}, cap(puzzles))
	seenMembers := make(map[string]struct{}, len(evilMembers))
	scanner := bufio.NewScanner(evidence)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		var item auditEvidence
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			return fmt.Errorf("decode evidence line %d: %w", line, err)
		}
		if item.SchemaVersion != 1 || item.Outcome != "solved" {
			continue
		}
		id := "bp_" + item.PuzzleHash
		selected := false
		if item.Difficulty == "evil" {
			_, selected = evilMembers[id]
		} else if counts[item.Difficulty] < perGrade {
			selected = item.Difficulty == "hard" || item.Difficulty == "expert"
		}
		if !selected {
			continue
		}
		digits, ok := source[item.ID]
		if !ok {
			return fmt.Errorf("source record %s is absent", item.ID)
		}
		canonical := core.CanonicalPuzzle(strings.ReplaceAll(digits, "0", "."))
		if db.BasePuzzleID(canonical) != id {
			return fmt.Errorf("source record %s does not match evidence identity", item.ID)
		}
		puzzles = append(puzzles, catalogbundle.Puzzle{
			BasePuzzleID:   id,
			Puzzle:         canonical,
			Difficulty:     item.Difficulty,
			Score:          item.Score,
			MaxTechnique:   item.MaxTechnique,
			Source:         "sudoku-exchange-diabolical@" + sourceCommit,
			SourceRef:      item.ID,
			License:        "Public domain (Unlicense)",
			Redistribution: "unrestricted",
		})
		counts[item.Difficulty]++
		seenPuzzles[id] = struct{}{}
		if item.Difficulty == "evil" {
			seenMembers[id] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read audit evidence: %w", err)
	}
	for _, selection := range []struct {
		path, digest, grade, source string
	}{
		{easySourcePath, easyDigest, "easy", "sudoku-exchange-easy@" + sourceCommit},
		{mediumSourcePath, mediumDigest, "medium", "sudoku-exchange-medium@" + sourceCommit},
	} {
		selected, err := selectClassifiedSource(selection.path, selection.digest, selection.source, selection.grade, perGrade, seenPuzzles)
		if err != nil {
			return err
		}
		puzzles = append(puzzles, selected...)
		counts[selection.grade] += len(selected)
	}
	for _, grade := range []string{"easy", "medium", "hard", "expert"} {
		if counts[grade] != perGrade {
			return fmt.Errorf("selected %d %s puzzles, want %d", counts[grade], grade, perGrade)
		}
	}
	if len(seenMembers) != evilcohort.MemberCount {
		return fmt.Errorf("selected %d Evil cohort members, want %d", len(seenMembers), evilcohort.MemberCount)
	}
	sort.Slice(puzzles, func(i, j int) bool { return puzzles[i].BasePuzzleID < puzzles[j].BasePuzzleID })
	manifest := catalogbundle.Manifest{
		FormatVersion:      1,
		BundleVersion:      2,
		SolverConfigDigest: solver.ConfigDigest(solver.NewStore()),
		Puzzles:            puzzles,
		Cohorts: []catalogbundle.Cohort{{
			Name:               evilcohort.Name,
			Difficulty:         "evil",
			RuleVersion:        evilcohort.RuleVersion,
			ManifestHash:       evilcohort.ManifestHash,
			RepositoryCommit:   evilcohort.RepositoryCommit,
			SolverConfigDigest: evilcohort.SolverConfigDigest,
			EvidenceHash:       evilcohort.EvidenceHash,
			Members:            cohort.BasePuzzleIDs,
		}},
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode catalog manifest: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("write catalog manifest: %w", err)
	}
	artifact := sha256.Sum256(data)
	fmt.Printf("wrote %d puzzles (%d each Easy-Medium-Hard-Expert, %d Evil) to %s\nsha256 %s\n", len(puzzles), perGrade, evilcohort.MemberCount, outputPath, hex.EncodeToString(artifact[:]))
	return nil
}

func selectClassifiedSource(path, digest, sourceName, grade string, count int, seen map[string]struct{}) ([]catalogbundle.Puzzle, error) {
	records, err := loadSource(path, digest)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(records))
	for id := range records {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	store := solver.NewStore()
	selected := make([]catalogbundle.Puzzle, 0, count)
	for _, id := range ids {
		canonical := core.CanonicalPuzzle(strings.ReplaceAll(records[id], "0", "."))
		baseID := db.BasePuzzleID(canonical)
		if _, duplicate := seen[baseID]; duplicate {
			continue
		}
		var board core.Board
		board.FromString(canonical)
		classification := solver.ClassifyPuzzle(store, board)
		if classification.Outcome != solver.ClassificationSolved || classification.Difficulty != grade {
			continue
		}
		selected = append(selected, catalogbundle.Puzzle{
			BasePuzzleID:   baseID,
			Puzzle:         canonical,
			Difficulty:     grade,
			Score:          classification.Score,
			MaxTechnique:   classification.MaxTechnique,
			Source:         sourceName,
			SourceRef:      id,
			License:        "Public domain (Unlicense)",
			Redistribution: "unrestricted",
		})
		seen[baseID] = struct{}{}
		if len(selected) == count {
			return selected, nil
		}
	}
	return nil, fmt.Errorf("selected %d %s puzzles from %s, want %d", len(selected), grade, path, count)
}

func loadSource(path, expectedDigest string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read source bank: %w", err)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != expectedDigest {
		return nil, fmt.Errorf("source bank SHA-256 is %s, want %s", got, expectedDigest)
	}
	records := make(map[string]string, len(data)/100)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for line := 1; scanner.Scan(); line++ {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 || len(fields[0]) != 12 || len(fields[1]) != 81 {
			return nil, fmt.Errorf("invalid source record at line %d", line)
		}
		id := "sha1:" + fields[0]
		if _, duplicate := records[id]; duplicate {
			return nil, fmt.Errorf("duplicate source record %s", id)
		}
		records[id] = fields[1]
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read source bank: %w", err)
	}
	return records, nil
}
