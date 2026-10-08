// Package difficultyaudit produces deterministic per-puzzle evidence for
// comparing an independent source rating with the canonical strategy solver.
package difficultyaudit

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gnailuy/sudoku/calibration"
	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

const SchemaVersion = 1

type Options struct {
	ManifestPath, OutputDir, RepositoryCommit string
	Workers                                   int
}

type Evidence struct {
	SchemaVersion              int                          `json:"schema_version"`
	Index                      int                          `json:"index"`
	ID                         string                       `json:"id"`
	PuzzleHash                 string                       `json:"puzzle_hash"`
	SourceID                   string                       `json:"source_id"`
	Split                      string                       `json:"split"`
	OriginalRatingSystem       string                       `json:"original_rating_system"`
	OriginalRating             float64                      `json:"original_rating"`
	Outcome                    solver.ClassificationOutcome `json:"outcome"`
	Difficulty                 string                       `json:"difficulty"`
	Score                      int                          `json:"score"`
	MaxTechnique               string                       `json:"max_technique"`
	ClueCount                  int                          `json:"clue_count"`
	MoveCount                  int                          `json:"move_count"`
	AdvancedMoveCount          int                          `json:"advanced_move_count"`
	AdvancedTechniqueDiversity int                          `json:"advanced_technique_diversity"`
	AdvancedMoveDensity        float64                      `json:"advanced_move_density"`
	EvilMoveCount              int                          `json:"evil_move_count"`
	EvilTechniqueDiversity     int                          `json:"evil_technique_diversity"`
	LongestAdvancedRun         int                          `json:"longest_advanced_run"`
	TechniqueCounts            map[string]int               `json:"technique_counts"`
	TraceHash                  string                       `json:"trace_hash"`
}

type NumericSummary struct {
	Count  int     `json:"count"`
	Min    float64 `json:"min"`
	P25    float64 `json:"p25"`
	Median float64 `json:"median"`
	P75    float64 `json:"p75"`
	P95    float64 `json:"p95"`
	Max    float64 `json:"max"`
}
type Correlation struct {
	Count    int     `json:"count"`
	Pearson  float64 `json:"pearson"`
	Spearman float64 `json:"spearman"`
}
type Report struct {
	SchemaVersion           int                                  `json:"schema_version"`
	ManifestName            string                               `json:"manifest_name"`
	ManifestHash            string                               `json:"manifest_hash"`
	RepositoryCommit        string                               `json:"repository_commit"`
	SolverConfigDigest      string                               `json:"solver_config_digest"`
	GoVersion               string                               `json:"go_version"`
	OS                      string                               `json:"os"`
	Architecture            string                               `json:"architecture"`
	Total                   int                                  `json:"total"`
	ByOutcome               map[string]int                       `json:"by_outcome"`
	ByDifficulty            map[string]int                       `json:"by_difficulty"`
	MissingData             map[string]int                       `json:"missing_data"`
	MetricsByDifficulty     map[string]map[string]NumericSummary `json:"metrics_by_difficulty"`
	RatingCorrelations      map[string]Correlation               `json:"rating_correlations"`
	RatingQuartileAgreement map[string]map[string]int            `json:"rating_quartile_agreement"`
}
type Result struct{ Report Report }

// Run writes one immutable evidence directory. Rows remain in manifest order
// regardless of worker completion order.
func Run(options Options, store solver.Store) (Result, error) {
	if options.Workers < 1 {
		return Result{}, errors.New("workers must be positive")
	}
	if strings.TrimSpace(options.RepositoryCommit) == "" {
		return Result{}, errors.New("repository commit is required")
	}
	if _, err := os.Stat(options.OutputDir); err == nil {
		return Result{}, fmt.Errorf("output directory already exists: %s", options.OutputDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Result{}, err
	}
	manifest, manifestHash, err := calibration.LoadManifest(options.ManifestPath)
	if err != nil {
		return Result{}, err
	}

	evidence := make([]Evidence, len(manifest.Puzzles))
	type job struct {
		index  int
		puzzle calibration.Puzzle
	}
	type answer struct {
		index    int
		evidence Evidence
		err      error
	}
	jobs, answers := make(chan job), make(chan answer, options.Workers)
	var wg sync.WaitGroup
	for range options.Workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			workerStore := solver.NewStore()
			for item := range jobs {
				row, rowErr := measure(item.index, item.puzzle, workerStore)
				answers <- answer{item.index, row, rowErr}
			}
		}()
	}
	go func() {
		for i, puzzle := range manifest.Puzzles {
			jobs <- job{i, puzzle}
		}
		close(jobs)
		wg.Wait()
		close(answers)
	}()
	for answer := range answers {
		if answer.err != nil {
			return Result{}, answer.err
		}
		evidence[answer.index] = answer.evidence
	}

	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return Result{}, err
	}
	if err := writeJSONL(filepath.Join(options.OutputDir, "evidence.jsonl"), evidence); err != nil {
		return Result{}, err
	}
	report := deriveReport(manifest, manifestHash, options.RepositoryCommit, solverDigest(store), evidence)
	if err := writeJSON(filepath.Join(options.OutputDir, "report.json"), report); err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(filepath.Join(options.OutputDir, "report.md"), []byte(markdown(report)), 0o644); err != nil {
		return Result{}, err
	}
	return Result{Report: report}, nil
}

func measure(index int, puzzle calibration.Puzzle, store solver.Store) (Evidence, error) {
	if puzzle.OriginalRating == nil {
		return Evidence{}, fmt.Errorf("puzzle %d (%s): original rating is required", index, puzzle.ID)
	}
	rating, err := strconv.ParseFloat(puzzle.OriginalRating.Label, 64)
	if err != nil || math.IsNaN(rating) || math.IsInf(rating, 0) {
		return Evidence{}, fmt.Errorf("puzzle %d (%s): original rating is not finite", index, puzzle.ID)
	}
	board := core.NewEmptyBoard()
	board.FromString(puzzle.Puzzle)
	classification := solver.ClassifyPuzzle(store, board)
	repeated := solver.ClassifyPuzzle(store, board)
	if classification.Outcome != repeated.Outcome || classification.Difficulty != repeated.Difficulty || classification.Score != repeated.Score || classification.MaxTechnique != repeated.MaxTechnique || traceHash(classification.Moves) != traceHash(repeated.Moves) {
		return Evidence{}, fmt.Errorf("puzzle %d (%s): repeated classification was not identical", index, puzzle.ID)
	}
	counts := map[string]int{}
	advanced, evil, longest, run := 0, 0, 0, 0
	advancedTechniques, evilTechniques := map[string]struct{}{}, map[string]struct{}{}
	for _, move := range classification.Moves {
		counts[move.Technique]++
		_, tier, ok := solver.StrategyTierForTechnique(move.Technique)
		if ok && tier >= 2 {
			advanced++
			advancedTechniques[move.Technique] = struct{}{}
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
		if ok && tier == 4 {
			evil++
			evilTechniques[move.Technique] = struct{}{}
		}
	}
	density := 0.0
	if len(classification.Moves) > 0 {
		density = float64(advanced) / float64(len(classification.Moves))
	}
	return Evidence{SchemaVersion: SchemaVersion, Index: index, ID: puzzle.ID, PuzzleHash: puzzle.PuzzleHash, SourceID: puzzle.SourceID, Split: puzzle.Split,
		OriginalRatingSystem: puzzle.OriginalRating.System, OriginalRating: rating, Outcome: classification.Outcome, Difficulty: classification.Difficulty,
		Score: classification.Score, MaxTechnique: classification.MaxTechnique, ClueCount: 81 - strings.Count(puzzle.Puzzle, "."), MoveCount: len(classification.Moves),
		AdvancedMoveCount: advanced, AdvancedTechniqueDiversity: len(advancedTechniques), AdvancedMoveDensity: density, EvilMoveCount: evil,
		EvilTechniqueDiversity: len(evilTechniques), LongestAdvancedRun: longest, TechniqueCounts: counts, TraceHash: traceHash(classification.Moves)}, nil
}

func solverDigest(store solver.Store) string {
	type entry struct {
		Technique string `json:"technique"`
		Tier      string `json:"tier"`
		Weight    int    `json:"weight"`
	}
	var entries []entry
	for _, key := range store.GetAllStrategySolverKeys() {
		tier, _, _ := solver.StrategyTierForTechnique(key)
		entries = append(entries, entry{key, tier, store.GetStrategySolverByKey(key).GetWeight()})
	}
	data, _ := json.Marshal(entries)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
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
	for i, m := range moves {
		trace[i] = traceMove{m.Cell.Position.Row, m.Cell.Position.Column, m.Cell.Value, m.Technique, m.Reason, m.EliminationOnly}
	}
	data, _ := json.Marshal(trace)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func deriveReport(manifest calibration.Manifest, manifestHash, commit, digest string, rows []Evidence) Report {
	r := Report{SchemaVersion: SchemaVersion, ManifestName: manifest.Name, ManifestHash: manifestHash, RepositoryCommit: commit, SolverConfigDigest: digest, GoVersion: runtime.Version(), OS: runtime.GOOS, Architecture: runtime.GOARCH, Total: len(rows), ByOutcome: map[string]int{}, ByDifficulty: map[string]int{}, MissingData: map[string]int{"original_rating": 0, "difficulty": 0, "max_technique": 0}, MetricsByDifficulty: map[string]map[string]NumericSummary{}, RatingCorrelations: map[string]Correlation{}, RatingQuartileAgreement: map[string]map[string]int{}}
	type series struct{ rating, score, evil, advanced, density, moves, clues []float64 }
	groups := map[string]*series{"all": {}}
	for _, row := range rows {
		r.ByOutcome[string(row.Outcome)]++
		d := row.Difficulty
		if d == "" {
			r.MissingData["difficulty"]++
			d = "unassigned"
		}
		if row.MaxTechnique == "" {
			r.MissingData["max_technique"]++
		}
		r.ByDifficulty[d]++
		for _, key := range []string{"all", d} {
			if groups[key] == nil {
				groups[key] = &series{}
			}
			g := groups[key]
			g.rating = append(g.rating, row.OriginalRating)
			g.score = append(g.score, float64(row.Score))
			g.evil = append(g.evil, float64(row.EvilMoveCount))
			g.advanced = append(g.advanced, float64(row.AdvancedTechniqueDiversity))
			g.density = append(g.density, row.AdvancedMoveDensity)
			g.moves = append(g.moves, float64(row.MoveCount))
			g.clues = append(g.clues, float64(row.ClueCount))
		}
	}
	for key, g := range groups {
		r.MetricsByDifficulty[key] = map[string]NumericSummary{"original_rating": summary(g.rating), "score": summary(g.score), "evil_move_count": summary(g.evil), "advanced_technique_diversity": summary(g.advanced), "advanced_move_density": summary(g.density), "move_count": summary(g.moves), "clue_count": summary(g.clues)}
		r.RatingCorrelations[key+"_score"] = correlation(g.rating, g.score)
		r.RatingCorrelations[key+"_evil_move_count"] = correlation(g.rating, g.evil)
		r.RatingCorrelations[key+"_advanced_diversity"] = correlation(g.rating, g.advanced)
		r.RatingCorrelations[key+"_advanced_density"] = correlation(g.rating, g.density)
	}
	all := groups["all"]
	cuts := []float64{percentile(all.rating, .25), percentile(all.rating, .5), percentile(all.rating, .75)}
	for _, row := range rows {
		q := quartile(row.OriginalRating, cuts)
		d := row.Difficulty
		if d == "" {
			d = "unassigned"
		}
		if r.RatingQuartileAgreement[q] == nil {
			r.RatingQuartileAgreement[q] = map[string]int{}
		}
		r.RatingQuartileAgreement[q][d]++
	}
	return r
}

func summary(values []float64) NumericSummary {
	if len(values) == 0 {
		return NumericSummary{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	return NumericSummary{len(sorted), sorted[0], percentileSorted(sorted, .25), percentileSorted(sorted, .5), percentileSorted(sorted, .75), percentileSorted(sorted, .95), sorted[len(sorted)-1]}
}
func percentile(values []float64, p float64) float64 {
	v := append([]float64(nil), values...)
	sort.Float64s(v)
	return percentileSorted(v, p)
}
func percentileSorted(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	i := int(math.Ceil(p*float64(len(v)))) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(v) {
		i = len(v) - 1
	}
	return v[i]
}
func quartile(v float64, c []float64) string {
	if v <= c[0] {
		return "q1"
	}
	if v <= c[1] {
		return "q2"
	}
	if v <= c[2] {
		return "q3"
	}
	return "q4"
}
func correlation(x, y []float64) Correlation {
	return Correlation{len(x), pearson(x, y), pearson(ranks(x), ranks(y))}
}
func pearson(x, y []float64) float64 {
	if len(x) == 0 || len(x) != len(y) {
		return 0
	}
	var sx, sy float64
	for i := range x {
		sx += x[i]
		sy += y[i]
	}
	mx, my := sx/float64(len(x)), sy/float64(len(y))
	var n, dx, dy float64
	for i := range x {
		a, b := x[i]-mx, y[i]-my
		n += a * b
		dx += a * a
		dy += b * b
	}
	if dx == 0 || dy == 0 {
		return 0
	}
	return n / math.Sqrt(dx*dy)
}
func ranks(v []float64) []float64 {
	type pair struct {
		v float64
		i int
	}
	p := make([]pair, len(v))
	for i, x := range v {
		p[i] = pair{x, i}
	}
	sort.SliceStable(p, func(i, j int) bool { return p[i].v < p[j].v })
	r := make([]float64, len(v))
	for i := 0; i < len(p); {
		j := i + 1
		for j < len(p) && p[j].v == p[i].v {
			j++
		}
		rank := (float64(i+1) + float64(j)) / 2
		for k := i; k < j; k++ {
			r[p[k].i] = rank
		}
		i = j
	}
	return r
}

func writeJSONL(path string, rows []Evidence) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, row := range rows {
		b, err := json.Marshal(row)
		if err != nil {
			return err
		}
		if _, err = w.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return w.Flush()
}
func writeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
func markdown(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Full-Catalog Difficulty Audit\n\n- Manifest: `%s`\n- Manifest SHA-256: `%s`\n- Repository commit: `%s`\n- Solver configuration: `%s`\n- Runtime: `%s %s/%s`\n- Records: %d\n\n## Outcomes\n\n", r.ManifestName, r.ManifestHash, r.RepositoryCommit, r.SolverConfigDigest, r.GoVersion, r.OS, r.Architecture, r.Total)
	for _, k := range sortedIntKeys(r.ByOutcome) {
		fmt.Fprintf(&b, "- %s: %d\n", k, r.ByOutcome[k])
	}
	b.WriteString("\n## Assigned strategy grades\n\n")
	for _, k := range sortedIntKeys(r.ByDifficulty) {
		fmt.Fprintf(&b, "- %s: %d\n", k, r.ByDifficulty[k])
	}
	b.WriteString("\n## Independent-rating correlations\n\n| Scope / metric | n | Pearson | Spearman |\n|---|---:|---:|---:|\n")
	for _, k := range sortedCorrelationKeys(r.RatingCorrelations) {
		c := r.RatingCorrelations[k]
		fmt.Fprintf(&b, "| %s | %d | %.6f | %.6f |\n", k, c.Count, c.Pearson, c.Spearman)
	}
	b.WriteString("\n`evidence.jsonl` contains the immutable per-puzzle feature vectors; `report.json` contains complete distributions and cross-model quartile counts.\n")
	return b.String()
}
func sortedIntKeys(m map[string]int) []string {
	k := make([]string, 0, len(m))
	for x := range m {
		k = append(k, x)
	}
	sort.Strings(k)
	return k
}
func sortedCorrelationKeys(m map[string]Correlation) []string {
	k := make([]string, 0, len(m))
	for x := range m {
		k = append(k, x)
	}
	sort.Strings(k)
	return k
}
