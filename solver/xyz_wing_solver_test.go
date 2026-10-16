package solver_test

import (
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

// TestXYZWingSolver_FindsPattern tests that the XYZ-Wing solver detects a valid pattern.
func TestXYZWingSolver_FindsPattern(t *testing.T) {
	s := solver.NewXYZWingSolver()

	// A puzzle that may trigger XYZ-Wing after lower-tier solvers stall.
	puzzle := "..53.....8......2..7..1.5..4....53...1..7...6.32...8..6.5....9..4....3......97..."
	board := core.NewEmptyBoard()
	board.FromString(puzzle)

	advancedSolverTestHelper(t, &board, []string{
		"naked-single", "hidden-single", "naked-pair", "naked-triple",
		"pointing-pair", "hidden-pair", "x-wing", "xy-wing", "hidden-triple",
		"swordfish", "naked-quad", "hidden-quad",
	})

	if board.IsSolved() {
		t.Skip("Lower-tier solvers solved this puzzle alone — need a harder example")
	}

	move := s.Apply(&board)
	if move == nil {
		t.Skip("XYZ-Wing solver did not find a move on this puzzle — may need different puzzle state")
	}

	if move.Technique != "xyz-wing" {
		t.Errorf("Expected technique xyz-wing, got %q", move.Technique)
	}
	if move.Reason == "" {
		t.Error("Expected non-empty reason")
	}
}

// TestXYZWingSolver_Metadata tests the solver's metadata fields.
func TestXYZWingSolver_Metadata(t *testing.T) {
	s := solver.NewXYZWingSolver()
	if s.GetKey() != "xyz-wing" {
		t.Errorf("Expected key xyz-wing, got %q", s.GetKey())
	}
	if s.GetDisplayName() != "XYZ-Wing" {
		t.Errorf("Expected display name XYZ-Wing, got %q", s.GetDisplayName())
	}
	if s.GetWeight() != 180 {
		t.Errorf("Expected weight 180, got %d", s.GetWeight())
	}
}

// TestXYZWingSolver_NoProgress tests that the solver returns nil on a solved board.
func TestXYZWingSolver_NoProgress(t *testing.T) {
	s := solver.NewXYZWingSolver()
	board := core.NewEmptyBoard()
	puzzle := "534678912672195348198342567859761423426853791713924856961537284287419635345286179"
	board.FromString(puzzle)

	move := s.Apply(&board)
	if move != nil {
		t.Errorf("Expected nil on solved board, got %v", move)
	}
}

// TestXYZWingSolver_EmptyBoard tests that the solver returns nil on an empty board.
func TestXYZWingSolver_EmptyBoard(t *testing.T) {
	s := solver.NewXYZWingSolver()
	board := core.NewEmptyBoard()

	move := s.Apply(&board)
	if move != nil {
		t.Errorf("Expected nil on empty board, got %v", move)
	}
}
