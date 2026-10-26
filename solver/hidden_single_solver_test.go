package solver

import (
	"reflect"
	"testing"

	"github.com/gnailuy/sudoku/core"
)

func TestHiddenSingleSolver_FindsInRow(t *testing.T) {
	// Build a board where digit 1 can only appear in one cell in row 0.
	// Fill row 0 cols 1-8 with values 2-9, but also put digit 1 into the
	// column 0 peers (rows 1,2) so it's truly a hidden single scenario.
	board := core.NewEmptyBoard()

	// Row 0: cols 1-8 filled with 2,3,4,5,6,7,8,9
	for c := 1; c <= 8; c++ {
		_ = board.Set(core.NewPosition(0, c), c+1)
	}
	// R0C0 is empty. Digit 1 is a candidate. It's actually a naked single here too,
	// but the hidden single solver should also find it (only cell in row 0 that can hold 1).

	solver := NewHiddenSingleSolver()
	move := solver.Apply(&board)

	if move == nil {
		t.Fatal("expected a move, got nil")
	}
	if move.Cell.Value != 1 {
		t.Errorf("expected value 1, got %d", move.Cell.Value)
	}
	if move.Technique != "hidden-single" {
		t.Errorf("expected technique 'hidden-single', got '%s'", move.Technique)
	}
}

func TestHiddenSingleSolver_FindsInBox(t *testing.T) {
	// Create a scenario where digit 5 can only go in one cell within box 0 (rows 0-2, cols 0-2).
	board := core.NewEmptyBoard()

	// Fill enough cells so that digit 5 is only possible in one cell of box 0.
	// Box 0 cells: (0,0), (0,1), (0,2), (1,0), (1,1), (1,2), (2,0), (2,1), (2,2)
	// Fill most of box 0:
	_ = board.Set(core.NewPosition(0, 0), 1)
	_ = board.Set(core.NewPosition(0, 1), 2)
	_ = board.Set(core.NewPosition(0, 2), 3)
	_ = board.Set(core.NewPosition(1, 0), 4)
	// (1,1) is empty
	_ = board.Set(core.NewPosition(1, 2), 6)
	_ = board.Set(core.NewPosition(2, 0), 7)
	_ = board.Set(core.NewPosition(2, 1), 8)
	_ = board.Set(core.NewPosition(2, 2), 9)

	// Now (1,1) is the only empty cell in box 0 → it must be 5.
	// This is actually a naked single, but hidden single should also find it
	// (digit 5 can only go in (1,1) within box 0).

	solver := NewHiddenSingleSolver()
	move := solver.Apply(&board)

	if move == nil {
		t.Fatal("expected a move, got nil")
	}
	if move.Cell.Value != 5 {
		t.Errorf("expected value 5, got %d", move.Cell.Value)
	}
	if move.Cell.Position.Row != 1 || move.Cell.Position.Column != 1 {
		t.Errorf("expected R1C1, got R%dC%d", move.Cell.Position.Row, move.Cell.Position.Column)
	}
}

func TestHiddenSingleSolver_TrueHiddenSingle(t *testing.T) {
	// Build a scenario where a cell has multiple candidates but one of them is
	// unique in the unit — a true hidden single (not naked single).
	board := core.NewEmptyBoard()

	// We'll construct a partial board where R0C0 has candidates {1, 5}
	// but 5 also appears as candidate in other row-0 cells, while 1 does NOT
	// appear in any other row-0 cell → hidden single for 1 in row 0 at R0C0.

	// Put 1 in the column 0 of every other row? No — then R0C0 can't have 1.
	// Instead: block 1 from all other empty cells in row 0.
	// Row 0: R0C0 empty, rest have values or 1 is blocked.
	_ = board.Set(core.NewPosition(0, 1), 2)
	_ = board.Set(core.NewPosition(0, 2), 3)
	_ = board.Set(core.NewPosition(0, 3), 4)
	_ = board.Set(core.NewPosition(0, 4), 5)
	_ = board.Set(core.NewPosition(0, 5), 6)
	_ = board.Set(core.NewPosition(0, 6), 7)
	// R0C7 and R0C8 are empty. Block digit 1 from them by putting 1 in their columns.
	_ = board.Set(core.NewPosition(1, 7), 1) // blocks 1 from column 7
	_ = board.Set(core.NewPosition(2, 8), 1) // blocks 1 from column 8

	// Now in row 0: R0C0 can hold {1, 8, 9}, R0C7 can hold {8, 9, ...} but NOT 1,
	// R0C8 can hold {8, 9, ...} but NOT 1. So 1 is a hidden single in row 0 at R0C0.

	solver := NewHiddenSingleSolver()
	move := solver.Apply(&board)

	if move == nil {
		t.Fatal("expected a move, got nil")
	}
	if move.Cell.Value != 1 {
		t.Errorf("expected value 1, got %d", move.Cell.Value)
	}
	if move.Cell.Position.Row != 0 || move.Cell.Position.Column != 0 {
		t.Errorf("expected R0C0, got R%dC%d", move.Cell.Position.Row, move.Cell.Position.Column)
	}
	if move.Technique != "hidden-single" {
		t.Errorf("expected technique 'hidden-single', got '%s'", move.Technique)
	}
}

func TestHiddenSingleSolver_MaintainedRowWitnesses(t *testing.T) {
	var board core.Board
	board.FromString("439658..1871234965256..7.......2..5.....8.49.593746.1.9.....5..3.....68.7..4691..")

	move := NewHiddenSingleSolver().Apply(&board)
	if move == nil || move.Cell.Position != core.NewPosition(3, 3) || move.Cell.Value != 9 {
		t.Fatalf("move = %+v, want r4c4=9", move)
	}
	want := []core.Cell{
		core.NewCell(core.NewPosition(6, 0), 9),
		core.NewCell(core.NewPosition(5, 1), 9),
		core.NewCell(core.NewPosition(0, 2), 9),
		core.NewCell(core.NewPosition(8, 5), 9),
		core.NewCell(core.NewPosition(1, 6), 9),
		core.NewCell(core.NewPosition(4, 7), 9),
	}
	if !reflect.DeepEqual(move.Evidence.Witnesses, want) {
		t.Fatalf("witnesses = %+v, want %+v", move.Evidence.Witnesses, want)
	}
	if len(move.Evidence.RuledOut) != 6 {
		t.Fatalf("ruled-out candidates = %+v, want six row cells", move.Evidence.RuledOut)
	}
}

func TestHiddenSingleSolver_NoProgress(t *testing.T) {
	// An empty board: every digit appears in all cells of every unit.
	board := core.NewEmptyBoard()
	solver := NewHiddenSingleSolver()
	move := solver.Apply(&board)

	if move != nil {
		t.Errorf("expected nil on empty board, got move: %v", move)
	}
}

func TestHiddenSingleSolver_SolvedBoard(t *testing.T) {
	board := buildSolvedBoard()
	solver := NewHiddenSingleSolver()
	move := solver.Apply(&board)

	if move != nil {
		t.Errorf("expected nil on solved board, got move: %v", move)
	}
}
