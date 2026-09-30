package generationexperiment

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	hardSeed     = "1....7.9..3..2...8..96..5....53..9...1..8...26....4...3......1..4......7..7...3.."
	expertSeed   = ".....6....59.....82....8....45........3........6..3.54...325..6.................."
	resultPuzzle = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."
)

func TestRunResumesAndDerivesReport(t *testing.T) {
	manifestPath := writeManifest(t, validManifest())
	output := filepath.Join(t.TempDir(), "output")
	calls := 0
	executor := ExecutorFunc(func(job Job) (Execution, error) {
		calls++
		outcome := "wrong-grade"
		difficulty := "easy"
		puzzle := resultPuzzle
		if job.Arm == "candidate" {
			outcome = "timed-out"
			difficulty = ""
			puzzle = ""
		}
		return Execution{Outcome: outcome, Puzzle: puzzle, Difficulty: difficulty, DurationMS: 12, Classifications: 1}, nil
	})

	first, err := Run(manifestPath, output, executor)
	if err != nil {
		t.Fatal(err)
	}
	if first.Appended != 2 || first.Report.Observed != 2 || !first.Report.Complete {
		t.Fatalf("unexpected first result: %+v", first)
	}
	if calls != 2 {
		t.Fatalf("executor calls = %d, want 2", calls)
	}
	if got := first.Report.Groups["baseline/hard"].Outcomes["wrong-grade"]; got != 1 {
		t.Fatalf("baseline wrong-grade count = %d, want 1", got)
	}

	before, err := os.ReadFile(filepath.Join(output, "observations.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Run(manifestPath, output, executor)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(output, "observations.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if second.Appended != 0 || calls != 2 || string(before) != string(after) {
		t.Fatalf("resume changed durable observations: appended=%d calls=%d", second.Appended, calls)
	}
}

func TestRunResumesAfterExecutorFailure(t *testing.T) {
	manifestPath := writeManifest(t, validManifest())
	output := filepath.Join(t.TempDir(), "output")
	failed := false
	executor := ExecutorFunc(func(job Job) (Execution, error) {
		if job.Index == 1 && !failed {
			failed = true
			return Execution{}, errors.New("interrupted")
		}
		return Execution{Outcome: "timed-out", DurationMS: 10, Classifications: 1}, nil
	})
	if _, err := Run(manifestPath, output, executor); err == nil || !strings.Contains(err.Error(), "interrupted") {
		t.Fatalf("first run error = %v, want interruption", err)
	}
	result, err := Run(manifestPath, output, executor)
	if err != nil {
		t.Fatal(err)
	}
	if result.Appended != 1 || result.Report.Observed != 2 {
		t.Fatalf("resume result = %+v", result)
	}
}

func TestRunRejectsChangedManifest(t *testing.T) {
	manifest := validManifest()
	manifestPath := writeManifest(t, manifest)
	output := filepath.Join(t.TempDir(), "output")
	executor := ExecutorFunc(func(Job) (Execution, error) {
		return Execution{Outcome: "timed-out", DurationMS: 1, Classifications: 1}, nil
	})
	if _, err := Run(manifestPath, output, executor); err != nil {
		t.Fatal(err)
	}
	manifest.Name = "changed"
	data, _ := json.Marshal(manifest)
	if err := os.WriteFile(manifestPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(manifestPath, output, executor); err == nil || !strings.Contains(err.Error(), "active manifest") {
		t.Fatalf("changed manifest error = %v", err)
	}
}

func TestManifestRejectsCanonicalSeedReuseAcrossSplits(t *testing.T) {
	manifest := validManifest()
	duplicate := manifest.Samples[0]
	duplicate.ID = "held-out-hard"
	duplicate.SeedID = "catalog-hard-copy"
	duplicate.Split = "held-out"
	manifest.Samples = append(manifest.Samples, duplicate)
	manifestPath := writeManifest(t, manifest)
	_, _, err := loadManifest(manifestPath)
	if err == nil || !strings.Contains(err.Error(), "canonically identical") {
		t.Fatalf("duplicate seed error = %v", err)
	}
}

func TestRunRejectsExecutionBeyondSharedBudget(t *testing.T) {
	manifestPath := writeManifest(t, validManifest())
	executor := ExecutorFunc(func(Job) (Execution, error) {
		return Execution{Outcome: "timed-out", DurationMS: 1001, Classifications: 1}, nil
	})
	_, err := Run(manifestPath, filepath.Join(t.TempDir(), "output"), executor)
	if err == nil || !strings.Contains(err.Error(), "job budget") {
		t.Fatalf("budget violation error = %v", err)
	}
}

func TestHarnessOwnsCanonicalDuplicateAccounting(t *testing.T) {
	manifest := validManifest()
	manifest.Samples[0].SeedPuzzle = resultPuzzle
	manifestPath := writeManifest(t, manifest)
	output := filepath.Join(t.TempDir(), "output")
	executor := ExecutorFunc(func(Job) (Execution, error) {
		return Execution{Outcome: "exact-grade", Puzzle: resultPuzzle, Difficulty: "hard", DurationMS: 1, Classifications: 1}, nil
	})
	result, err := Run(manifestPath, output, executor)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Report.Groups["baseline/hard"].Outcomes["duplicate"]; got != 1 {
		t.Fatalf("duplicate count = %d, want 1", got)
	}
	if got := result.Report.Groups["candidate/hard"].Outcomes["duplicate"]; got != 1 {
		t.Fatalf("candidate duplicate count = %d, want 1", got)
	}
}

func validManifest() Manifest {
	return Manifest{
		Version: Version, Name: "exact-grade-pilot", RepositoryCommit: "0123456789abcdef",
		SolverConfigHash: "solver-config-v1", PolicyVersion: "trace-mutation-v1",
		Budget: Budget{MaxDurationMS: 1000, MaxClassifications: 10},
		Samples: []Sample{{
			ID: "exploratory-hard", Split: "exploratory", TargetDifficulty: "hard",
			SeedID: "catalog-hard-1", SeedPuzzle: hardSeed, RandomSeed: 42,
		}},
	}
}

func writeManifest(t *testing.T, manifest Manifest) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
