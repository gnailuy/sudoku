package solver

import (
	"testing"

	"github.com/gnailuy/sudoku/core"
)

func TestNakedSingleSolver_FindsSingle(t *testing.T) {
	// Build a board where R1C1 has only one candidate.
	// Fill row 1 (cols 1-8) with 2-8, col 0 of rows 1-8 with values that
	// leave exactly one candidate for R0C0.
	board := core.NewEmptyBoard()

	// Row 0: fill cols 1-8 with 2,3,4,5,6,7,8,9
	for c := 1; c <= 8; c++ {
		_ = board.Set(core.NewPosition(0, c), c+1)
	}
	// R0C0 should now have only candidate 1 (all others eliminated by row peers).

	solver := NewNakedSingleSolver()
	move := solver.Apply(&board)

	if move == nil {
		t.Fatal("expected a move, got nil")
	}
	if move.Cell.Position.Row != 0 || move.Cell.Position.Column != 0 {
		t.Errorf("expected R0C0, got R%dC%d", move.Cell.Position.Row, move.Cell.Position.Column)
	}
	if move.Cell.Value != 1 {
		t.Errorf("expected value 1, got %d", move.Cell.Value)
	}
	if move.Technique != "naked-single" {
		t.Errorf("expected technique 'naked-single', got '%s'", move.Technique)
	}
	if len(move.Evidence.Witnesses) != 8 {
		t.Fatalf("witnesses = %+v, want one placed peer for every excluded digit", move.Evidence.Witnesses)
	}
}

func TestNakedSingleSolver_NoProgress(t *testing.T) {
	// An empty board has multiple candidates for every cell.
	board := core.NewEmptyBoard()
	solver := NewNakedSingleSolver()
	move := solver.Apply(&board)

	if move != nil {
		t.Errorf("expected nil, got move: %v", move)
	}
}

func TestNakedSingleSolver_SolvedBoard(t *testing.T) {
	// A fully solved board has no empty cells → no move.
	board := buildSolvedBoard()
	solver := NewNakedSingleSolver()
	move := solver.Apply(&board)

	if move != nil {
		t.Errorf("expected nil on solved board, got move: %v", move)
	}
}

// buildSolvedBoard returns a valid solved 9x9 Sudoku board for testing.
func buildSolvedBoard() core.Board {
	board := core.NewEmptyBoard()
	// A valid completed board (band-based construction).
	rows := [9][9]int{
		{1, 2, 3, 4, 5, 6, 7, 8, 9},
		{4, 5, 6, 7, 8, 9, 1, 2, 3},
		{7, 8, 9, 1, 2, 3, 4, 5, 6},
		{2, 3, 1, 5, 6, 4, 8, 9, 7},
		{5, 6, 4, 8, 9, 7, 2, 3, 1},
		{8, 9, 7, 2, 3, 1, 5, 6, 4},
		{3, 1, 2, 6, 4, 5, 9, 7, 8},
		{6, 4, 5, 9, 7, 8, 3, 1, 2},
		{9, 7, 8, 3, 1, 2, 6, 4, 5},
	}
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			_ = board.Set(core.NewPosition(r, c), rows[r][c])
		}
	}
	return board
}
