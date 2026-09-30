package generationexperiment

import (
	"math/rand"
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

func TestMutationDirectionUsesClassificationFeedback(t *testing.T) {
	tests := []struct {
		name           string
		target         string
		classification solver.Classification
		wantRemove     bool
	}{
		{name: "raise grade by removing a clue", target: "hard", classification: solver.Classification{Solved: true, Difficulty: "easy"}, wantRemove: true},
		{name: "harvest around an exact seed by restoring a clue", target: "hard", classification: solver.Classification{Solved: true, Difficulty: "hard"}, wantRemove: false},
		{name: "lower grade by restoring a clue", target: "hard", classification: solver.Classification{Solved: true, Difficulty: "evil"}, wantRemove: false},
		{name: "repair stalled trace by restoring a clue", target: "evil", classification: solver.Classification{}, wantRemove: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := mutationDirection(test.target, test.classification); got != test.wantRemove {
				t.Fatalf("mutationDirection() = %t, want %t", got, test.wantRemove)
			}
		})
	}
}

func TestMutatePuzzleIsDeterministicForSeed(t *testing.T) {
	board := core.NewEmptyBoard()
	board.FromString(hardSeed)
	solution := board.Copy()
	backtracker := solver.NewBacktracker()
	if !backtracker.SolveDeterministic(&solution) {
		t.Fatal("fixture did not solve")
	}
	classification := solver.Classification{Solved: true, Difficulty: "hard"}
	first, ok := mutatePuzzle(rand.New(rand.NewSource(42)), board, solution, "hard", classification)
	if !ok {
		t.Fatal("first mutation unavailable")
	}
	second, ok := mutatePuzzle(rand.New(rand.NewSource(42)), board, solution, "hard", classification)
	if !ok {
		t.Fatal("second mutation unavailable")
	}
	if first.ToString() != second.ToString() {
		t.Fatalf("seeded mutations differ:\n%s\n%s", first.ToString(), second.ToString())
	}
	if first.GetFilledCellsCount() != board.GetFilledCellsCount()+1 {
		t.Fatalf("mutation changed clue count by %d, want +1", first.GetFilledCellsCount()-board.GetFilledCellsCount())
	}
}

func TestExecutionFromClassificationPreservesTimeout(t *testing.T) {
	board := core.NewEmptyBoard()
	board.FromString(hardSeed)
	job := Job{Sample: Sample{TargetDifficulty: "hard"}}
	execution := executionFromClassification(job, board, solver.Classification{Solved: true, Difficulty: "hard"}, 10, 1, true)
	if execution.Outcome != "timed-out" {
		t.Fatalf("outcome = %q, want timed-out", execution.Outcome)
	}
}
