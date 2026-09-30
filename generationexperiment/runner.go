// Package generationexperiment provides the isolated, resumable execution
// boundary for exact-grade generator experiments.
package generationexperiment

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gnailuy/sudoku/core"
)

const Version = 1

var arms = [...]string{"baseline", "candidate"}

// Manifest binds every experiment job to immutable seeds, budgets, policy,
// repository, and solver configuration identities.
type Manifest struct {
	Version          int      `json:"version"`
	Name             string   `json:"name"`
	RepositoryCommit string   `json:"repository_commit"`
	SolverConfigHash string   `json:"solver_config_hash"`
	PolicyVersion    string   `json:"policy_version"`
	Budget           Budget   `json:"budget"`
	Samples          []Sample `json:"samples"`
}

// Budget is shared by both arms for every sample.
type Budget struct {
	MaxDurationMS      int64 `json:"max_duration_ms"`
	MaxClassifications int   `json:"max_classifications"`
}

// Sample identifies one catalog seed and deterministic random seed.
type Sample struct {
	ID               string `json:"id"`
	Split            string `json:"split"`
	TargetDifficulty string `json:"target_difficulty"`
	SeedID           string `json:"seed_id"`
	SeedPuzzle       string `json:"seed_puzzle"`
	RandomSeed       int64  `json:"random_seed"`
}

// Job is one stable arm/sample pair passed to an Executor.
type Job struct {
	Index  int
	Arm    string
	Sample Sample
	Budget Budget
}

// Execution is the raw result returned by an experiment arm. Outcome must be
// one of exact-grade, wrong-grade, strategy-unsolved, invalid, non-unique, or
// timed-out. Duplicate classification is owned by the harness.
type Execution struct {
	Outcome         string `json:"outcome"`
	Puzzle          string `json:"puzzle,omitempty"`
	Difficulty      string `json:"difficulty,omitempty"`
	Score           int    `json:"score,omitempty"`
	MaxTechnique    string `json:"max_technique,omitempty"`
	TraceDigest     string `json:"trace_digest,omitempty"`
	DurationMS      int64  `json:"duration_ms"`
	Classifications int    `json:"classifications"`
}

// Executor runs one job without mutating the catalog or output directory.
type Executor interface {
	Execute(Job) (Execution, error)
}

// ExecutorFunc adapts a function to Executor.
type ExecutorFunc func(Job) (Execution, error)

func (fn ExecutorFunc) Execute(job Job) (Execution, error) { return fn(job) }

// Observation is one durable append-only job result.
type Observation struct {
	Version          int    `json:"version"`
	Index            int    `json:"index"`
	ManifestHash     string `json:"manifest_hash"`
	SampleID         string `json:"sample_id"`
	Arm              string `json:"arm"`
	Split            string `json:"split"`
	TargetDifficulty string `json:"target_difficulty"`
	SeedID           string `json:"seed_id"`
	RandomSeed       int64  `json:"random_seed"`
	Outcome          string `json:"outcome"`
	Puzzle           string `json:"puzzle,omitempty"`
	CanonicalPuzzle  string `json:"canonical_puzzle,omitempty"`
	Difficulty       string `json:"difficulty,omitempty"`
	Score            int    `json:"score,omitempty"`
	MaxTechnique     string `json:"max_technique,omitempty"`
	TraceDigest      string `json:"trace_digest,omitempty"`
	DurationMS       int64  `json:"duration_ms"`
	Classifications  int    `json:"classifications"`
}

// GroupReport preserves raw outcome counts for one arm and target grade.
type GroupReport struct {
	Attempts int            `json:"attempts"`
	Outcomes map[string]int `json:"outcomes"`
}

// Report is deterministically derived from the append-only observation log.
type Report struct {
	Version      int                    `json:"version"`
	ManifestName string                 `json:"manifest_name"`
	ManifestHash string                 `json:"manifest_hash"`
	Observed     int                    `json:"observed"`
	Total        int                    `json:"total"`
	Complete     bool                   `json:"complete"`
	Groups       map[string]GroupReport `json:"groups"`
}

// Result describes the durable state produced by Run.
type Result struct {
	ManifestHash string
	Appended     int
	Report       Report
}

type checkpoint struct {
	Version      int    `json:"version"`
	ManifestHash string `json:"manifest_hash"`
	NextIndex    int    `json:"next_index"`
}

// Run starts or resumes a manifest-bound experiment in outputDir.
func Run(manifestPath, outputDir string, executor Executor) (Result, error) {
	if executor == nil {
		return Result{}, errors.New("executor is required")
	}
	manifest, manifestHash, err := loadManifest(manifestPath)
	if err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(outputDir) == "" {
		return Result{}, errors.New("output directory is required")
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
	if err := writeJSONAtomic(checkpointPath, checkpoint{Version: Version, ManifestHash: manifestHash, NextIndex: len(observations)}); err != nil {
		return Result{}, err
	}

	log, err := os.OpenFile(observationsPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return Result{}, fmt.Errorf("open observation log: %w", err)
	}
	defer log.Close()

	seedCanonicals := make(map[string]struct{}, len(manifest.Samples))
	for _, sample := range manifest.Samples {
		seedCanonicals[core.CanonicalPuzzle(sample.SeedPuzzle)] = struct{}{}
	}
	resultCanonicals := make(map[string]struct{}, len(observations))
	for _, observation := range observations {
		if observation.CanonicalPuzzle != "" {
			resultCanonicals[observation.CanonicalPuzzle] = struct{}{}
		}
	}

	appended := 0
	total := len(manifest.Samples) * len(arms)
	for index := len(observations); index < total; index++ {
		sample := manifest.Samples[index/len(arms)]
		job := Job{Index: index, Arm: arms[index%len(arms)], Sample: sample, Budget: manifest.Budget}
		execution, err := executor.Execute(job)
		if err != nil {
			return Result{}, fmt.Errorf("execute job %d: %w", index, err)
		}
		observation, err := makeObservation(index, manifestHash, job, execution, seedCanonicals, resultCanonicals)
		if err != nil {
			return Result{}, fmt.Errorf("job %d: %w", index, err)
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
		if observation.CanonicalPuzzle != "" {
			resultCanonicals[observation.CanonicalPuzzle] = struct{}{}
		}
		appended++
		if err := writeJSONAtomic(checkpointPath, checkpoint{Version: Version, ManifestHash: manifestHash, NextIndex: index + 1}); err != nil {
			return Result{}, err
		}
	}

	report := deriveReport(manifest, manifestHash, observations)
	if err := writeJSONAtomic(filepath.Join(outputDir, "report.json"), report); err != nil {
		return Result{}, err
	}
	return Result{ManifestHash: manifestHash, Appended: appended, Report: report}, nil
}

func loadManifest(path string) (Manifest, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, "", fmt.Errorf("read manifest: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, "", fmt.Errorf("decode manifest: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Manifest{}, "", err
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, "", err
	}
	return manifest, hash(data), nil
}

func validateManifest(manifest Manifest) error {
	if manifest.Version != Version {
		return fmt.Errorf("unsupported manifest version %d", manifest.Version)
	}
	for name, value := range map[string]string{
		"name": manifest.Name, "repository_commit": manifest.RepositoryCommit,
		"solver_config_hash": manifest.SolverConfigHash, "policy_version": manifest.PolicyVersion,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if manifest.Budget.MaxDurationMS <= 0 || manifest.Budget.MaxClassifications <= 0 {
		return errors.New("budget limits must be positive")
	}
	if len(manifest.Samples) == 0 {
		return errors.New("manifest must contain at least one sample")
	}
	validSplit := map[string]bool{"exploratory": true, "held-out": true}
	validGrade := map[string]bool{"hard": true, "expert": true, "evil": true}
	seenID := map[string]struct{}{}
	seenSeed := map[string]string{}
	for index, sample := range manifest.Samples {
		if strings.TrimSpace(sample.ID) == "" || strings.TrimSpace(sample.SeedID) == "" {
			return fmt.Errorf("sample %d: id and seed_id are required", index)
		}
		if _, ok := seenID[sample.ID]; ok {
			return fmt.Errorf("sample %d: duplicate id %q", index, sample.ID)
		}
		seenID[sample.ID] = struct{}{}
		if !validSplit[sample.Split] {
			return fmt.Errorf("sample %d (%s): invalid split %q", index, sample.ID, sample.Split)
		}
		if !validGrade[sample.TargetDifficulty] {
			return fmt.Errorf("sample %d (%s): invalid target difficulty %q", index, sample.ID, sample.TargetDifficulty)
		}
		if !core.IsValidSudokuString(sample.SeedPuzzle) || strings.Contains(sample.SeedPuzzle, "0") {
			return fmt.Errorf("sample %d (%s): seed_puzzle must be valid dot notation", index, sample.ID)
		}
		canonical := core.CanonicalPuzzle(sample.SeedPuzzle)
		if prior, ok := seenSeed[canonical]; ok {
			return fmt.Errorf("sample %d (%s): seed is canonically identical to %s", index, sample.ID, prior)
		}
		seenSeed[canonical] = sample.ID
	}
	return nil
}

func makeObservation(index int, manifestHash string, job Job, execution Execution, seeds, results map[string]struct{}) (Observation, error) {
	validOutcome := map[string]bool{
		"exact-grade": true, "wrong-grade": true, "strategy-unsolved": true,
		"invalid": true, "non-unique": true, "timed-out": true,
	}
	if !validOutcome[execution.Outcome] {
		return Observation{}, fmt.Errorf("invalid outcome %q", execution.Outcome)
	}
	if execution.DurationMS < 0 || execution.DurationMS > job.Budget.MaxDurationMS || execution.Classifications < 0 || execution.Classifications > job.Budget.MaxClassifications {
		return Observation{}, errors.New("execution metrics violate the job budget")
	}
	if execution.Outcome == "exact-grade" && execution.Difficulty != job.Sample.TargetDifficulty {
		return Observation{}, errors.New("exact-grade result does not match target difficulty")
	}
	if execution.Outcome == "wrong-grade" && (execution.Difficulty == "" || execution.Difficulty == job.Sample.TargetDifficulty) {
		return Observation{}, errors.New("wrong-grade result must carry a different assigned difficulty")
	}

	canonical := ""
	if execution.Puzzle != "" {
		if !core.IsValidSudokuString(execution.Puzzle) {
			return Observation{}, errors.New("result puzzle is invalid")
		}
		canonical = core.CanonicalPuzzle(execution.Puzzle)
		if _, duplicate := seeds[canonical]; duplicate {
			execution.Outcome = "duplicate"
		} else if _, duplicate := results[canonical]; duplicate {
			execution.Outcome = "duplicate"
		}
	}
	return Observation{
		Version: Version, Index: index, ManifestHash: manifestHash,
		SampleID: job.Sample.ID, Arm: job.Arm, Split: job.Sample.Split,
		TargetDifficulty: job.Sample.TargetDifficulty, SeedID: job.Sample.SeedID,
		RandomSeed: job.Sample.RandomSeed, Outcome: execution.Outcome,
		Puzzle: execution.Puzzle, CanonicalPuzzle: canonical,
		Difficulty: execution.Difficulty, Score: execution.Score,
		MaxTechnique: execution.MaxTechnique, TraceDigest: execution.TraceDigest,
		DurationMS: execution.DurationMS, Classifications: execution.Classifications,
	}, nil
}

func readObservations(path string, manifest Manifest, manifestHash string) ([]Observation, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open observations: %w", err)
	}
	defer file.Close()
	var observations []Observation
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var observation Observation
		if err := json.Unmarshal(scanner.Bytes(), &observation); err != nil {
			return nil, fmt.Errorf("decode observation %d: %w", len(observations), err)
		}
		index := len(observations)
		if observation.Version != Version || observation.Index != index || observation.ManifestHash != manifestHash {
			return nil, fmt.Errorf("observation %d does not match the active manifest", index)
		}
		sample := manifest.Samples[index/len(arms)]
		if observation.SampleID != sample.ID || observation.Arm != arms[index%len(arms)] || observation.SeedID != sample.SeedID {
			return nil, fmt.Errorf("observation %d does not match its manifest job", index)
		}
		observations = append(observations, observation)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read observations: %w", err)
	}
	if len(observations) > len(manifest.Samples)*len(arms) {
		return nil, errors.New("observation log exceeds manifest jobs")
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
	if value.Version != Version || value.ManifestHash != manifestHash {
		return errors.New("checkpoint does not match the active manifest")
	}
	if value.NextIndex > observed {
		return errors.New("checkpoint is ahead of the observation log")
	}
	return nil
}

func deriveReport(manifest Manifest, manifestHash string, observations []Observation) Report {
	groups := make(map[string]GroupReport)
	for _, observation := range observations {
		key := observation.Arm + "/" + observation.TargetDifficulty
		group := groups[key]
		if group.Outcomes == nil {
			group.Outcomes = make(map[string]int)
		}
		group.Attempts++
		group.Outcomes[observation.Outcome]++
		groups[key] = group
	}
	return Report{
		Version: Version, ManifestName: manifest.Name, ManifestHash: manifestHash,
		Observed: len(observations), Total: len(manifest.Samples) * len(arms),
		Complete: len(observations) == len(manifest.Samples)*len(arms), Groups: groups,
	}
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}
	return writeAtomic(path, append(data, '\n'))
}

func writeAtomic(path string, data []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	return nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("manifest contains multiple JSON values")
		}
		return fmt.Errorf("decode trailing manifest data: %w", err)
	}
	return nil
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// SortedGroupKeys returns report group keys in stable presentation order.
func SortedGroupKeys(report Report) []string {
	keys := make([]string, 0, len(report.Groups))
	for key := range report.Groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
