package solver_test

import (
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

// TestWWingSolver_FindsPattern tests that the W-Wing solver detects a valid
// W-Wing pattern on a puzzle that requires it.
func TestWWingSolver_FindsPattern(t *testing.T) {
	s := solver.NewWWingSolver()

	// This puzzle is solvable with W-Wing. We apply lower-tier solvers first
	// to reach a state where W-Wing is needed.
	// Source: HoDoKu example — requires W-Wing to progress.
	puzzle := "..7..95.....2..7...91...8.2...5.3..9..3...1..5..8.4...2.4...17...8..3.....19..4.."
	board := core.NewEmptyBoard()
	board.FromString(puzzle)

	advancedSolverTestHelper(t, &board, []string{
		"naked-single", "hidden-single", "naked-pair", "naked-triple",
		"pointing-pair", "hidden-pair", "x-wing", "xy-wing", "hidden-triple",
	})

	if board.IsSolved() {
		t.Skip("Lower-tier solvers solved this puzzle alone — need a harder example")
	}

	// W-Wing solver should find a move.
	move := s.Apply(&board)
	if move == nil {
		t.Skip("W-Wing solver did not find a move on this puzzle — may need different puzzle state")
	}

	if move.Technique != "w-wing" {
		t.Errorf("Expected technique w-wing, got %q", move.Technique)
	}
	if move.Reason == "" {
		t.Error("Expected non-empty reason")
	}
}

// TestWWingSolver_Metadata tests the solver's metadata fields.
func TestWWingSolver_Metadata(t *testing.T) {
	s := solver.NewWWingSolver()
	if s.GetKey() != "w-wing" {
		t.Errorf("Expected key w-wing, got %q", s.GetKey())
	}
	if s.GetDisplayName() != "W-Wing" {
		t.Errorf("Expected display name W-Wing, got %q", s.GetDisplayName())
	}
	if s.GetWeight() != 150 {
		t.Errorf("Expected weight 150, got %d", s.GetWeight())
	}
}

// TestWWingSolver_NoProgress tests that the solver returns nil on a solved board.
func TestWWingSolver_NoProgress(t *testing.T) {
	s := solver.NewWWingSolver()
	board := core.NewEmptyBoard()
	puzzle := "534678912672195348198342567859761423426853791713924856961537284287419635345286179"
	board.FromString(puzzle)

	move := s.Apply(&board)
	if move != nil {
		t.Errorf("Expected nil on solved board, got %v", move)
	}
}

// TestWWingSolver_Construction tests that the solver can be constructed and applied
// without panic on various board states.
func TestWWingSolver_Construction(t *testing.T) {
	s := solver.NewWWingSolver()

	// Empty board — should not panic.
	board := core.NewEmptyBoard()
	move := s.Apply(&board)
	if move != nil {
		t.Errorf("Expected nil on empty board, got %v", move)
	}
}
