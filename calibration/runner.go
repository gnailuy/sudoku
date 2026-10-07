// Package calibration provides deterministic, resumable difficulty measurements.
package calibration

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

const Version = 2

// Manifest is an immutable, ordered corpus definition.
type Manifest struct {
	Version int      `json:"version"`
	Name    string   `json:"name"`
	Puzzles []Puzzle `json:"puzzles"`
}

// Puzzle is one stable corpus member.
type Puzzle struct {
	ID                 string             `json:"id"`
	Puzzle             string             `json:"puzzle"`
	PuzzleHash         string             `json:"puzzle_hash"`
	SourceCategory     string             `json:"source_category"`
	SourceID           string             `json:"source_id"`
	License            string             `json:"license"`
	Redistribution     string             `json:"redistribution"`
	CollectionMethod   string             `json:"collection_method"`
	Split              string             `json:"split"`
	ExpectedDifficulty string             `json:"expected_difficulty,omitempty"`
	OriginalRating     *OriginalRating    `json:"original_rating,omitempty"`
	Generator          *GeneratorMetadata `json:"generator,omitempty"`
}

// OriginalRating preserves a source's own scale without silently normalizing it.
type OriginalRating struct {
	System string `json:"system"`
	Label  string `json:"label"`
}

// GeneratorMetadata records how a generated corpus member was collected.
type GeneratorMetadata struct {
	RequestedDifficulty   string `json:"requested_difficulty"`
	Configuration         string `json:"configuration"`
	Attempt               int    `json:"attempt"`
	ElapsedMilliseconds   int64  `json:"elapsed_milliseconds"`
	ClueCount             int    `json:"clue_count"`
	ClassificationOutcome string `json:"classification_outcome"`
}

// Observation is one append-only classification result.
type Observation struct {
	Version            int                          `json:"version"`
	Index              int                          `json:"index"`
	ManifestHash       string                       `json:"manifest_hash"`
	ID                 string                       `json:"id"`
	PuzzleHash         string                       `json:"puzzle_hash"`
	SourceCategory     string                       `json:"source_category"`
	SourceID           string                       `json:"source_id"`
	Split              string                       `json:"split"`
	ExpectedDifficulty string                       `json:"expected_difficulty,omitempty"`
	Outcome            solver.ClassificationOutcome `json:"outcome"`
	Difficulty         string                       `json:"difficulty"`
	Score              int                          `json:"score"`
	MaxTechnique       string                       `json:"max_technique"`
	MoveCount          int                          `json:"move_count"`
	TraceHash          string                       `json:"trace_hash"`
	RepeatCount        int                          `json:"repeat_count"`
	MatchesExpected    *bool                        `json:"matches_expected,omitempty"`
}

type checkpoint struct {
	Version      int    `json:"version"`
	ManifestHash string `json:"manifest_hash"`
	NextIndex    int    `json:"next_index"`
}

// Report is deterministically derived from an observation log.
type Report struct {
	Version                 int                                  `json:"version"`
	ManifestName            string                               `json:"manifest_name"`
	ManifestHash            string                               `json:"manifest_hash"`
	Observed                int                                  `json:"observed"`
	Total                   int                                  `json:"total"`
	Complete                bool                                 `json:"complete"`
	Reproducible            int                                  `json:"reproducible"`
	ByOutcome               map[string]int                       `json:"by_outcome"`
	ByDifficulty            map[string]int                       `json:"by_difficulty"`
	BySource                map[string]GroupReport               `json:"by_source"`
	BySplit                 map[string]GroupReport               `json:"by_split"`
	MetricsByDifficulty     map[string]MetricReport              `json:"metrics_by_difficulty"`
	NeighboringScoreOverlap map[string]OverlapReport             `json:"neighboring_score_overlap"`
	ExternalAgreement       map[string]map[string]map[string]int `json:"external_agreement"`
	GenerationMismatch      map[string]map[string]int            `json:"generation_mismatch"`
	GenerationHitRate       map[string]RateReport                `json:"generation_hit_rate"`
	GenerationLatencyMS     map[string]NumericReport             `json:"generation_latency_ms"`
	GenerationRounds        map[string]NumericReport             `json:"generation_rounds"`
	Expected                int                                  `json:"expected"`
	Matched                 int                                  `json:"matched"`
}

// GroupReport summarizes outcomes for a source category or corpus split.
type GroupReport struct {
	Observed             int        `json:"observed"`
	Solved               int        `json:"solved"`
	StrategyUnsolved     int        `json:"strategy_unsolved"`
	StrategyUnsolvedRate RateReport `json:"strategy_unsolved_rate"`
}

// MetricReport keeps distributions separate for each assigned tier.
type MetricReport struct {
	Observed int           `json:"observed"`
	Score    NumericReport `json:"score"`
	Moves    NumericReport `json:"moves"`
	Clues    NumericReport `json:"clues"`
}

// NumericReport uses deterministic nearest-rank percentiles.
type NumericReport struct {
	Count  int `json:"count"`
	Min    int `json:"min"`
	Median int `json:"median"`
	P95    int `json:"p95"`
	Max    int `json:"max"`
}

// OverlapReport describes intersection between observed neighboring-tier score ranges.
type OverlapReport struct {
	LowerTier string `json:"lower_tier"`
	UpperTier string `json:"upper_tier"`
	Overlaps  bool   `json:"overlaps"`
	Lower     int    `json:"lower,omitempty"`
	Upper     int    `json:"upper,omitempty"`
}

// RateReport includes a Wilson 95% confidence interval. A zero denominator
// produces all-zero values and remains explicit in the denominator field.
type RateReport struct {
	Numerator   int     `json:"numerator"`
	Denominator int     `json:"denominator"`
	Rate        float64 `json:"rate"`
	Lower95     float64 `json:"lower_95"`
	Upper95     float64 `json:"upper_95"`
}

// Result describes the durable files produced by Run.
type Result struct {
	ManifestHash string
	Appended     int
	Report       Report
}

// Run resumes or starts a measurement run in outputDir.
func Run(manifestPath, outputDir string, store solver.Store) (Result, error) {
	manifest, manifestHash, err := loadManifest(manifestPath)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create output directory: %w", err)
	}

	observationsPath := filepath.Join(outputDir, "observations.jsonl")
	observations, err := readObservations(observationsPath, manifest, manifestHash)
	if err != nil {
		return Result{}, err
	}
	checkpointPath := filepath.Join(outputDir, "checkpoint.json")
	if err := validateCheckpoint(checkpointPath, manifestHash, len(observations)); err != nil {
		return Result{}, err
	}
	// The observation log is authoritative. Rebuild a missing or lagging
	// checkpoint before resuming after an interruption.
	if err := writeJSONAtomic(checkpointPath, checkpoint{Version: Version, ManifestHash: manifestHash, NextIndex: len(observations)}); err != nil {
		return Result{}, err
	}

	log, err := os.OpenFile(observationsPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return Result{}, fmt.Errorf("open observation log: %w", err)
	}
	defer log.Close()

	appended := 0
	for index := len(observations); index < len(manifest.Puzzles); index++ {
		observation, err := measure(index, manifestHash, manifest.Puzzles[index], store)
		if err != nil {
			return Result{}, err
		}
		line, err := json.Marshal(observation)
		if err != nil {
			return Result{}, fmt.Errorf("encode observation %d: %w", index, err)
		}
		if _, err := log.Write(append(line, '\n')); err != nil {
			return Result{}, fmt.Errorf("append observation %d: %w", index, err)
		}
		if err := log.Sync(); err != nil {
			return Result{}, fmt.Errorf("sync observation %d: %w", index, err)
		}
		observations = append(observations, observation)
		appended++
		if err := writeJSONAtomic(checkpointPath, checkpoint{Version: Version, ManifestHash: manifestHash, NextIndex: index + 1}); err != nil {
			return Result{}, err
		}
	}

	report := deriveReport(manifest, manifestHash, observations)
	if err := writeJSONAtomic(filepath.Join(outputDir, "report.json"), report); err != nil {
		return Result{}, err
	}
	if err := writeAtomic(filepath.Join(outputDir, "report.md"), []byte(markdownReport(report)), 0o644); err != nil {
		return Result{}, err
	}
	return Result{ManifestHash: manifestHash, Appended: appended, Report: report}, nil
}

// PrepareManifest normalizes candidate records, adds content hashes, validates
// provenance, and writes the canonical immutable manifest consumed by Run.
func PrepareManifest(inputPath, outputPath string) (Manifest, error) {
	manifest, err := decodeManifest(inputPath)
	if err != nil {
		return Manifest{}, err
	}
	return WriteManifest(outputPath, manifest)
}

// WriteManifest normalizes, validates, and atomically writes a new immutable
// corpus manifest. Existing output is never replaced.
func WriteManifest(outputPath string, manifest Manifest) (Manifest, error) {
	for i := range manifest.Puzzles {
		manifest.Puzzles[i].Puzzle = normalizePuzzle(manifest.Puzzles[i].Puzzle)
		manifest.Puzzles[i].PuzzleHash = hash([]byte(manifest.Puzzles[i].Puzzle))
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	if _, err := os.Stat(outputPath); err == nil {
		return Manifest{}, fmt.Errorf("output manifest already exists: %s", outputPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, fmt.Errorf("inspect output manifest: %w", err)
	}
	if err := writeJSONAtomic(outputPath, manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func loadManifest(path string) (Manifest, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, "", fmt.Errorf("read manifest: %w", err)
	}
	manifest, err := decodeManifestBytes(data)
	if err != nil {
		return Manifest{}, "", err
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, "", err
	}
	return manifest, hash(data), nil
}

func decodeManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	return decodeManifestBytes(data)
}

func decodeManifestBytes(data []byte) (Manifest, error) {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func validateManifest(manifest Manifest) error {
	if manifest.Version != Version {
		return fmt.Errorf("unsupported manifest version %d", manifest.Version)
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return errors.New("manifest name is required")
	}
	if len(manifest.Puzzles) == 0 {
		return errors.New("manifest must contain at least one puzzle")
	}
	seenIDs := make(map[string]struct{}, len(manifest.Puzzles))
	seenHashes := make(map[string]struct{}, len(manifest.Puzzles))
	validDifficulties := map[string]bool{"": true, "easy": true, "medium": true, "hard": true, "expert": true, "evil": true}
	validCategories := map[string]bool{"external": true, "generated": true, "imported": true, "pathological": true}
	validRedistribution := map[string]bool{"permitted": true, "restricted": true}
	validSplits := map[string]bool{"exploratory": true, "held-out": true}
	for i, puzzle := range manifest.Puzzles {
		if strings.TrimSpace(puzzle.ID) == "" {
			return fmt.Errorf("puzzle %d: id is required", i)
		}
		if _, duplicate := seenIDs[puzzle.ID]; duplicate {
			return fmt.Errorf("puzzle %d: duplicate id %q", i, puzzle.ID)
		}
		seenIDs[puzzle.ID] = struct{}{}
		if !core.IsValidSudokuString(puzzle.Puzzle) {
			return fmt.Errorf("puzzle %d (%s): invalid Sudoku string", i, puzzle.ID)
		}
		if normalizePuzzle(puzzle.Puzzle) != puzzle.Puzzle {
			return fmt.Errorf("puzzle %d (%s): puzzle must use canonical dot notation", i, puzzle.ID)
		}
		expectedHash := hash([]byte(puzzle.Puzzle))
		if puzzle.PuzzleHash != expectedHash {
			return fmt.Errorf("puzzle %d (%s): puzzle_hash does not match normalized puzzle", i, puzzle.ID)
		}
		if _, duplicate := seenHashes[puzzle.PuzzleHash]; duplicate {
			return fmt.Errorf("puzzle %d (%s): duplicate puzzle hash %s", i, puzzle.ID, puzzle.PuzzleHash)
		}
		seenHashes[puzzle.PuzzleHash] = struct{}{}
		if !validCategories[puzzle.SourceCategory] {
			return fmt.Errorf("puzzle %d (%s): invalid source_category %q", i, puzzle.ID, puzzle.SourceCategory)
		}
		for field, value := range map[string]string{"source_id": puzzle.SourceID, "license": puzzle.License, "collection_method": puzzle.CollectionMethod} {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("puzzle %d (%s): %s is required", i, puzzle.ID, field)
			}
		}
		if !validRedistribution[puzzle.Redistribution] {
			return fmt.Errorf("puzzle %d (%s): invalid redistribution %q", i, puzzle.ID, puzzle.Redistribution)
		}
		if !validSplits[puzzle.Split] {
			return fmt.Errorf("puzzle %d (%s): invalid split %q", i, puzzle.ID, puzzle.Split)
		}
		if !validDifficulties[puzzle.ExpectedDifficulty] {
			return fmt.Errorf("puzzle %d (%s): invalid expected difficulty %q", i, puzzle.ID, puzzle.ExpectedDifficulty)
		}
		if puzzle.OriginalRating != nil && (strings.TrimSpace(puzzle.OriginalRating.System) == "" || strings.TrimSpace(puzzle.OriginalRating.Label) == "") {
			return fmt.Errorf("puzzle %d (%s): original_rating requires system and label", i, puzzle.ID)
		}
		if puzzle.SourceCategory == "generated" {
			generator := puzzle.Generator
			if generator == nil {
				return fmt.Errorf("puzzle %d (%s): generated source requires generator metadata", i, puzzle.ID)
			}
			if !validDifficulties[generator.RequestedDifficulty] || generator.RequestedDifficulty == "" || strings.TrimSpace(generator.Configuration) == "" || generator.Attempt < 1 || generator.ElapsedMilliseconds < 0 || generator.ClueCount < 0 || generator.ClueCount > 81 || (generator.ClassificationOutcome != string(solver.ClassificationSolved) && generator.ClassificationOutcome != string(solver.ClassificationStrategyUnsolved)) {
				return fmt.Errorf("puzzle %d (%s): invalid generator metadata", i, puzzle.ID)
			}
		} else if puzzle.Generator != nil {
			return fmt.Errorf("puzzle %d (%s): generator metadata requires generated source_category", i, puzzle.ID)
		}
	}
	return nil
}

func normalizePuzzle(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		if r == '0' {
			return '.'
		}
		return r
	}, value)
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("decode manifest: trailing JSON value")
		}
		return fmt.Errorf("decode manifest: %w", err)
	}
	return nil
}

func readObservations(path string, manifest Manifest, manifestHash string) ([]Observation, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open observation log: %w", err)
	}
	defer file.Close()

	var observations []Observation
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		index := len(observations)
		var observation Observation
		if err := json.Unmarshal(scanner.Bytes(), &observation); err != nil {
			return nil, fmt.Errorf("decode observation %d: %w", index, err)
		}
		if index >= len(manifest.Puzzles) {
			return nil, fmt.Errorf("observation %d exceeds manifest length", index)
		}
		puzzle := manifest.Puzzles[index]
		if observation.Version != Version || observation.Index != index || observation.ManifestHash != manifestHash || observation.ID != puzzle.ID || observation.PuzzleHash != hash([]byte(puzzle.Puzzle)) {
			return nil, fmt.Errorf("observation %d does not match immutable manifest prefix", index)
		}
		observations = append(observations, observation)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read observation log: %w", err)
	}
	return observations, nil
}

func validateCheckpoint(path, manifestHash string, observed int) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read checkpoint: %w", err)
	}
	var value checkpoint
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode checkpoint: %w", err)
	}
	if value.Version != Version || value.ManifestHash != manifestHash || value.NextIndex < 0 || value.NextIndex > observed {
		return errors.New("checkpoint does not match the immutable manifest and observation log")
	}
	return nil
}

func measure(index int, manifestHash string, puzzle Puzzle, store solver.Store) (Observation, error) {
	board := core.NewEmptyBoard()
	board.FromString(puzzle.Puzzle)
	classification := solver.ClassifyPuzzle(store, board)
	repeated := solver.ClassifyPuzzle(store, board)
	if classificationSignature(classification) != classificationSignature(repeated) {
		return Observation{}, fmt.Errorf("puzzle %d (%s): repeated classification was not identical", index, puzzle.ID)
	}
	observation := Observation{
		Version: Version, Index: index, ManifestHash: manifestHash, ID: puzzle.ID,
		PuzzleHash: puzzle.PuzzleHash, SourceCategory: puzzle.SourceCategory,
		SourceID: puzzle.SourceID, Split: puzzle.Split,
		ExpectedDifficulty: puzzle.ExpectedDifficulty, Outcome: classification.Outcome,
		Difficulty: classification.Difficulty, Score: classification.Score,
		MaxTechnique: classification.MaxTechnique, MoveCount: len(classification.Moves),
		TraceHash: traceHash(classification.Moves), RepeatCount: 2,
	}
	if puzzle.ExpectedDifficulty != "" {
		matched := classification.Difficulty == puzzle.ExpectedDifficulty
		observation.MatchesExpected = &matched
	}
	return observation, nil
}

func classificationSignature(classification solver.Classification) string {
	return fmt.Sprintf("%s\x00%s\x00%d\x00%s\x00%d\x00%s", classification.Outcome, classification.Difficulty, classification.Score, classification.MaxTechnique, len(classification.Moves), traceHash(classification.Moves))
}

func traceHash(moves []solver.Move) string {
	type traceMove struct {
		Row             int    `json:"row"`
		Column          int    `json:"column"`
		Value           int    `json:"value"`
		Technique       string `json:"technique"`
		Reason          string `json:"reason"`
		EliminationOnly bool   `json:"elimination_only"`
	}
	trace := make([]traceMove, len(moves))
	for i, move := range moves {
		trace[i] = traceMove{move.Cell.Position.Row, move.Cell.Position.Column, move.Cell.Value, move.Technique, move.Reason, move.EliminationOnly}
	}
	data, _ := json.Marshal(trace)
	return hash(data)
}

func deriveReport(manifest Manifest, manifestHash string, observations []Observation) Report {
	report := Report{
		Version: Version, ManifestName: manifest.Name, ManifestHash: manifestHash,
		Observed: len(observations), Total: len(manifest.Puzzles), Complete: len(observations) == len(manifest.Puzzles),
		ByOutcome: map[string]int{}, ByDifficulty: map[string]int{},
		BySource: map[string]GroupReport{}, BySplit: map[string]GroupReport{}, MetricsByDifficulty: map[string]MetricReport{}, NeighboringScoreOverlap: map[string]OverlapReport{},
		ExternalAgreement: map[string]map[string]map[string]int{}, GenerationMismatch: map[string]map[string]int{},
		GenerationHitRate: map[string]RateReport{}, GenerationLatencyMS: map[string]NumericReport{}, GenerationRounds: map[string]NumericReport{},
	}
	type metricValues struct{ scores, moves, clues []int }
	metrics := map[string]*metricValues{}
	generationHits := map[string][2]int{}
	generationLatency := map[string][]int{}
	generationRounds := map[string][]int{}

	for _, observation := range observations {
		if observation.RepeatCount >= 2 {
			report.Reproducible++
		}
		puzzle := manifest.Puzzles[observation.Index]
		difficulty := observation.Difficulty
		if difficulty == "" {
			difficulty = "unassigned"
		}
		report.ByOutcome[string(observation.Outcome)]++
		report.ByDifficulty[difficulty]++
		report.BySource[puzzle.SourceCategory] = addGroup(report.BySource[puzzle.SourceCategory], observation.Outcome)
		report.BySplit[puzzle.Split] = addGroup(report.BySplit[puzzle.Split], observation.Outcome)

		values := metrics[difficulty]
		if values == nil {
			values = &metricValues{}
			metrics[difficulty] = values
		}
		values.scores = append(values.scores, observation.Score)
		values.moves = append(values.moves, observation.MoveCount)
		values.clues = append(values.clues, strings.Count(puzzle.Puzzle, "1")+strings.Count(puzzle.Puzzle, "2")+strings.Count(puzzle.Puzzle, "3")+strings.Count(puzzle.Puzzle, "4")+strings.Count(puzzle.Puzzle, "5")+strings.Count(puzzle.Puzzle, "6")+strings.Count(puzzle.Puzzle, "7")+strings.Count(puzzle.Puzzle, "8")+strings.Count(puzzle.Puzzle, "9"))

		if puzzle.SourceCategory == "external" && puzzle.OriginalRating != nil {
			system := puzzle.OriginalRating.System
			label := puzzle.OriginalRating.Label
			if report.ExternalAgreement[system] == nil {
				report.ExternalAgreement[system] = map[string]map[string]int{}
			}
			if report.ExternalAgreement[system][label] == nil {
				report.ExternalAgreement[system][label] = map[string]int{}
			}
			report.ExternalAgreement[system][label][difficulty]++
		}
		if puzzle.SourceCategory == "generated" && puzzle.Generator != nil {
			target := puzzle.Generator.RequestedDifficulty
			if report.GenerationMismatch[target] == nil {
				report.GenerationMismatch[target] = map[string]int{}
			}
			report.GenerationMismatch[target][difficulty]++
			hit := generationHits[target]
			hit[1]++
			if difficulty == target && observation.Outcome == solver.ClassificationSolved {
				hit[0]++
			}
			generationHits[target] = hit
			generationLatency[target] = append(generationLatency[target], int(puzzle.Generator.ElapsedMilliseconds))
			generationRounds[target] = append(generationRounds[target], puzzle.Generator.Attempt)
		}
		if observation.MatchesExpected != nil {
			report.Expected++
			if *observation.MatchesExpected {
				report.Matched++
			}
		}
	}
	for key, group := range report.BySource {
		group.StrategyUnsolvedRate = rateReport(group.StrategyUnsolved, group.Observed)
		report.BySource[key] = group
	}
	for key, group := range report.BySplit {
		group.StrategyUnsolvedRate = rateReport(group.StrategyUnsolved, group.Observed)
		report.BySplit[key] = group
	}
	for difficulty, values := range metrics {
		report.MetricsByDifficulty[difficulty] = MetricReport{Observed: len(values.scores), Score: numericReport(values.scores), Moves: numericReport(values.moves), Clues: numericReport(values.clues)}
	}
	tiers := []string{"easy", "medium", "hard", "expert", "evil"}
	for i := 0; i < len(tiers)-1; i++ {
		lower, lowerOK := report.MetricsByDifficulty[tiers[i]]
		upper, upperOK := report.MetricsByDifficulty[tiers[i+1]]
		if !lowerOK || !upperOK {
			continue
		}
		overlap := OverlapReport{LowerTier: tiers[i], UpperTier: tiers[i+1]}
		overlap.Lower = max(lower.Score.Min, upper.Score.Min)
		overlap.Upper = min(lower.Score.Max, upper.Score.Max)
		overlap.Overlaps = overlap.Lower <= overlap.Upper
		if !overlap.Overlaps {
			overlap.Lower, overlap.Upper = 0, 0
		}
		report.NeighboringScoreOverlap[tiers[i]+"-"+tiers[i+1]] = overlap
	}
	for target, hit := range generationHits {
		report.GenerationHitRate[target] = rateReport(hit[0], hit[1])
		report.GenerationLatencyMS[target] = numericReport(generationLatency[target])
		report.GenerationRounds[target] = numericReport(generationRounds[target])
	}
	return report
}

func addGroup(group GroupReport, outcome solver.ClassificationOutcome) GroupReport {
	group.Observed++
	if outcome == solver.ClassificationSolved {
		group.Solved++
	} else if outcome == solver.ClassificationStrategyUnsolved {
		group.StrategyUnsolved++
	}
	return group
}

func numericReport(values []int) NumericReport {
	if len(values) == 0 {
		return NumericReport{}
	}
	ordered := append([]int(nil), values...)
	sort.Ints(ordered)
	at := func(p float64) int {
		index := int(math.Ceil(p*float64(len(ordered)))) - 1
		if index < 0 {
			index = 0
		}
		return ordered[index]
	}
	return NumericReport{Count: len(ordered), Min: ordered[0], Median: at(0.5), P95: at(0.95), Max: ordered[len(ordered)-1]}
}

func rateReport(numerator, denominator int) RateReport {
	report := RateReport{Numerator: numerator, Denominator: denominator}
	if denominator == 0 {
		return report
	}
	const z = 1.959963984540054
	n := float64(denominator)
	p := float64(numerator) / n
	denom := 1 + z*z/n
	center := (p + z*z/(2*n)) / denom
	delta := z * math.Sqrt((p*(1-p)+z*z/(4*n))/n) / denom
	report.Rate, report.Lower95, report.Upper95 = p, math.Max(0, center-delta), math.Min(1, center+delta)
	return report
}

func markdownReport(report Report) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Difficulty Measurement Report\n\n- Manifest: `%s`\n- Hash: `%s`\n- Progress: %d/%d\n- Complete: %t\n- Reproducible classifications: %d/%d\n", report.ManifestName, report.ManifestHash, report.Observed, report.Total, report.Complete, report.Reproducible, report.Observed)
	if report.Expected > 0 {
		fmt.Fprintf(&builder, "- Expected matches: %d/%d\n", report.Matched, report.Expected)
	}
	writeCountSection(&builder, "Outcomes", report.ByOutcome)
	writeCountSection(&builder, "Difficulties", report.ByDifficulty)

	fmt.Fprint(&builder, "\n## Source Groups\n\n")
	for _, key := range sortedKeys(report.BySource) {
		group := report.BySource[key]
		fmt.Fprintf(&builder, "- %s: %d observed, %d solved, %d strategy-unsolved (%.1f%%, 95%% CI %.1f–%.1f%%)\n", key, group.Observed, group.Solved, group.StrategyUnsolved, 100*group.StrategyUnsolvedRate.Rate, 100*group.StrategyUnsolvedRate.Lower95, 100*group.StrategyUnsolvedRate.Upper95)
	}
	fmt.Fprint(&builder, "\n## Split Groups\n\n")
	for _, key := range sortedKeys(report.BySplit) {
		group := report.BySplit[key]
		fmt.Fprintf(&builder, "- %s: %d observed, %d solved, %d strategy-unsolved (%.1f%%, 95%% CI %.1f–%.1f%%)\n", key, group.Observed, group.Solved, group.StrategyUnsolved, 100*group.StrategyUnsolvedRate.Rate, 100*group.StrategyUnsolvedRate.Lower95, 100*group.StrategyUnsolvedRate.Upper95)
	}
	fmt.Fprint(&builder, "\n## Tier Distributions\n\n| Tier | n | Score min/median/p95/max | Moves min/median/p95/max | Clues min/median/p95/max |\n|---|---:|---|---|---|\n")
	for _, key := range sortedKeys(report.MetricsByDifficulty) {
		metric := report.MetricsByDifficulty[key]
		fmt.Fprintf(&builder, "| %s | %d | %s | %s | %s |\n", key, metric.Observed, formatNumeric(metric.Score), formatNumeric(metric.Moves), formatNumeric(metric.Clues))
	}
	fmt.Fprint(&builder, "\n## Neighboring-Tier Score-Range Overlap\n\n")
	for _, key := range sortedKeys(report.NeighboringScoreOverlap) {
		overlap := report.NeighboringScoreOverlap[key]
		if overlap.Overlaps {
			fmt.Fprintf(&builder, "- %s / %s: overlap %d–%d\n", overlap.LowerTier, overlap.UpperTier, overlap.Lower, overlap.Upper)
		} else {
			fmt.Fprintf(&builder, "- %s / %s: no observed range overlap\n", overlap.LowerTier, overlap.UpperTier)
		}
	}
	fmt.Fprint(&builder, "\n## Optional External Source Comparison\n")
	for _, system := range sortedKeys(report.ExternalAgreement) {
		fmt.Fprintf(&builder, "\n### %s\n\n", system)
		for _, label := range sortedKeys(report.ExternalAgreement[system]) {
			fmt.Fprintf(&builder, "- %s: %s\n", label, formatCounts(report.ExternalAgreement[system][label]))
		}
	}
	fmt.Fprint(&builder, "\n## Generation Measurements\n\n")
	for _, target := range sortedKeys(report.GenerationMismatch) {
		hit := report.GenerationHitRate[target]
		fmt.Fprintf(&builder, "- %s: hits %d/%d (%.1f%%, 95%% CI %.1f–%.1f%%); observed {%s}; latency ms %s; rounds %s\n", target, hit.Numerator, hit.Denominator, 100*hit.Rate, 100*hit.Lower95, 100*hit.Upper95, formatCounts(report.GenerationMismatch[target]), formatNumeric(report.GenerationLatencyMS[target]), formatNumeric(report.GenerationRounds[target]))
	}
	fmt.Fprint(&builder, "\n## Method and Limits\n\n- Every puzzle was classified twice; only exact outcome, score, maximum-technique, move-count, and trace agreement was recorded.\n- Numeric summaries use deterministic nearest-rank percentiles. Rate intervals are Wilson 95% confidence intervals.\n- External source scales remain separate and do not validate or redefine canonical strategy grades.\n- Generated samples test generator behavior, not external source agreement. Random seeds are absent because the public generator boundary does not expose them.\n- Pilot strata are deliberately small. The report identifies expansion targets and does not justify policy changes by itself.\n")
	return builder.String()
}

func writeCountSection(builder *strings.Builder, name string, values map[string]int) {
	fmt.Fprintf(builder, "\n## %s\n\n", name)
	for _, key := range sortedKeys(values) {
		fmt.Fprintf(builder, "- %s: %d\n", key, values[key])
	}
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func formatNumeric(value NumericReport) string {
	return fmt.Sprintf("%d/%d/%d/%d", value.Min, value.Median, value.P95, value.Max)
}

func formatCounts(values map[string]int) string {
	parts := make([]string, 0, len(values))
	for _, key := range sortedKeys(values) {
		parts = append(parts, fmt.Sprintf("%s=%d", key, values[key]))
	}
	return strings.Join(parts, ", ")
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}
	return writeAtomic(path, append(data, '\n'), 0o644)
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return fmt.Errorf("create temporary %s: %w", filepath.Base(path), err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", filepath.Base(path), err)
	}
	return nil
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
