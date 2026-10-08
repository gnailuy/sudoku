package difficultyaudit

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnailuy/sudoku/calibration"
	"github.com/gnailuy/sudoku/solver"
)

func TestRunProducesDeterministicEvidence(t *testing.T) {
	source, _, err := calibration.LoadManifest(filepath.Join("..", "calibration", "testdata", "mixed-pilot-v2.json"))
	if err != nil {
		t.Fatal(err)
	}
	puzzle := source.Puzzles[0]
	puzzle.OriginalRating = &calibration.OriginalRating{System: "reference", Label: "7.5"}
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if _, err := calibration.WriteManifest(manifestPath, calibration.Manifest{Version: calibration.Version, Name: "audit-test", Puzzles: []calibration.Puzzle{puzzle}}); err != nil {
		t.Fatal(err)
	}

	var outputs []string
	for i := 0; i < 2; i++ {
		output := filepath.Join(t.TempDir(), "audit")
		result, err := Run(Options{ManifestPath: manifestPath, OutputDir: output, RepositoryCommit: "0123456789abcdef", Workers: 2}, solver.NewStore())
		if err != nil {
			t.Fatal(err)
		}
		if result.Report.Total != 1 || result.Report.RatingCorrelations["all_score"].Count != 1 {
			t.Fatalf("unexpected report: %+v", result.Report)
		}
		outputs = append(outputs, output)
	}
	for _, name := range []string{"evidence.jsonl", "report.json", "report.md"} {
		first, err := os.ReadFile(filepath.Join(outputs[0], name))
		if err != nil {
			t.Fatal(err)
		}
		second, err := os.ReadFile(filepath.Join(outputs[1], name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("%s is not deterministic", name)
		}
	}
}

func TestRunRefusesExistingOutput(t *testing.T) {
	output := t.TempDir()
	_, err := Run(Options{ManifestPath: "unused", OutputDir: output, RepositoryCommit: "commit", Workers: 1}, solver.NewStore())
	if err == nil {
		t.Fatal("expected existing output rejection")
	}
}
