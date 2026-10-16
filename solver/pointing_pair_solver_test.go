package solver_test

import (
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

// TestPointingPairSolver_FindsPointingPairInRow builds a board where a candidate
// in a box is confined to one row, and eliminating it from the rest of that row
// creates a naked single.
func TestPointingPairSolver_FindsPointingPairInRow(t *testing.T) {
	board := core.NewEmptyBoard()

	// Goal: In box 0, digit 1 is a candidate only in row 0 at TWO cells
	// ((0,0) and (0,1)) → pointing pair.
	// Eliminate 1 from row 0 outside box 0 → (0,4) loses 1 → {2}.

	// Eliminate 1 from rows 1 and 2 entirely (outside box 0):
	_ = board.Set(core.NewPosition(1, 6), 1) // 1 in row 1
	_ = board.Set(core.NewPosition(2, 7), 1) // 1 in row 2

	// Row 0: leave (0,0), (0,1), (0,4) empty. Fill the rest.
	_ = board.Set(core.NewPosition(0, 2), 3)
	_ = board.Set(core.NewPosition(0, 3), 4)
	_ = board.Set(core.NewPosition(0, 5), 5)
	_ = board.Set(core.NewPosition(0, 6), 6)
	_ = board.Set(core.NewPosition(0, 7), 7)
	_ = board.Set(core.NewPosition(0, 8), 8)

	// Row 0 has {3,4,5,6,7,8}. Missing: {1,2,9}.
	// (0,0), (0,1), (0,4) ⊆ {1,2,9}.

	// Restrict (0,4) to {1,2} by placing 9 in col 4 (outside box 1):
	_ = board.Set(core.NewPosition(3, 4), 9)

	// (0,4) in box 1: (0,3)=4, (0,5)=5. No 1 in box 1. No 1 in col 4.
	// (0,4) candidates: {1,2,9} minus col 4 {9} = {1,2}. ✓

	// (0,0) in box 0: no extra constraints removing 1. Candidates ⊆ {1,2,9}. has1=true.
	// (0,1) in box 0: no extra constraints. Candidates ⊆ {1,2,9}. has1=true.
	// Box 0 rows 1,2: 1 eliminated. ✓ Pointing pair!

	// Elimination: remove 1 from (0,4) → {2}.

	s := solver.NewPointingPairSolver()
	move := s.Apply(&board)

	if move == nil {
		t.Fatal("Expected PointingPairSolver to find a move, got nil")
	}

	if move.Cell.Position != core.NewPosition(0, 4) {
		t.Errorf("Expected move at (1, 5), got %s", move.Cell.Position.ToString())
	}

	if move.Cell.Value != 2 {
		t.Errorf("Expected value 2, got %d", move.Cell.Value)
	}

	if move.Technique != "pointing-pair" {
		t.Errorf("Expected technique 'pointing-pair', got %q", move.Technique)
	}
}

// TestPointingPairSolver_FindsPointingPairInColumn builds a board where a
// candidate in a box is confined to one column, enabling elimination.
func TestPointingPairSolver_FindsPointingPairInColumn(t *testing.T) {
	board := core.NewEmptyBoard()

	// In box 0, digit 1 candidate only in col 0: (0,0) and (1,0).
	// Eliminate 1 from col 0 outside box 0 → (3,0) loses 1 → {2}.

	// Eliminate 1 from cols 1 and 2 within box 0's rows:
	_ = board.Set(core.NewPosition(0, 3), 1) // 1 in col... no, need to eliminate from col 1,2 within rows 0-2
	// Let me place 1 in col 1 and col 2 outside box 0 rows:
	_ = board.Set(core.NewPosition(5, 1), 1) // 1 in col 1
	_ = board.Set(core.NewPosition(6, 2), 1) // 1 in col 2

	// Now in box 0, cells in cols 1 and 2 all lose 1 by column.
	// Digit 1 in box 0 can only appear in col 0: (0,0), (1,0), (2,0).
	// For a pointing pair we need at least 2 in the same column. ✓

	// Target: (3,0) in col 0 outside box 0. Need {1,2} candidates.
	// Fill most of row 3 and col 0:
	_ = board.Set(core.NewPosition(3, 1), 3)
	_ = board.Set(core.NewPosition(3, 2), 4)
	_ = board.Set(core.NewPosition(3, 3), 5)
	_ = board.Set(core.NewPosition(3, 4), 6)
	_ = board.Set(core.NewPosition(3, 5), 7)
	_ = board.Set(core.NewPosition(3, 6), 8)
	_ = board.Set(core.NewPosition(3, 7), 9)

	// Col 0: fill rows 4-8:
	_ = board.Set(core.NewPosition(4, 0), 3)
	_ = board.Set(core.NewPosition(5, 0), 4)
	_ = board.Set(core.NewPosition(6, 0), 5)
	_ = board.Set(core.NewPosition(7, 0), 6)
	_ = board.Set(core.NewPosition(8, 0), 7)

	// (3,0) candidates: {1-9} minus row 3 {3,4,5,6,7,8,9} minus col 0 {3,4,5,6,7}
	// = {1,2}. Box 3 (rows 3-5, cols 0-2): (3,1)=3, (3,2)=4, (4,0)=3, (5,0)=4, (5,1)=1.
	// Wait (5,0)=4, (5,1)=1. So 1 is in box 3 at (5,1)? No, (5,1)=1 was set above for col 1.
	// (5,1)=1 is in box 3 (rows 3-5, cols 0-2). So (3,0) loses 1 → {2}. That's a naked single. ✗

	// Move (5,1)=1 to outside box 3: row 5, col 4 instead.
	board = core.NewEmptyBoard()

	_ = board.Set(core.NewPosition(5, 4), 1) // 1 in col... this doesn't help col 1
	// Let me just use rows to eliminate from box 0 cols 1 and 2.
	// Actually, I need digit 1 eliminated from box 0 except col 0.
	// Place 1 in rows 0,1,2 at cols 1 or 2 (outside box 0) to eliminate from those cells.
	// No — placing in those rows would eliminate from row, not just column.

	// Better: place 1 in cols 1 and 2 somewhere in rows 0-2 but outside box 0.
	// But cols 1,2 rows 0-2 IS box 0. That's impossible.

	// Alternative: place 1 in the two rows that share box 0 cols 1 and 2.
	// Actually, the simplest way: place 1 somewhere that eliminates it from
	// (0,1), (0,2), (1,1), (1,2), (2,1), (2,2) but not from (0,0), (1,0), (2,0).
	// Place 1 in row 0 outside box 0 → (0,0) loses 1 by row. That's wrong.

	// For column pointing pair: place 1 to eliminate from cols 1 and 2 only.
	board = core.NewEmptyBoard()
	_ = board.Set(core.NewPosition(4, 1), 1) // 1 in col 1
	_ = board.Set(core.NewPosition(5, 2), 1) // 1 in col 2

	// Box 0 col 0: (0,0), (1,0), (2,0) can have 1.
	// Box 0 col 1: (0,1), (1,1), (2,1) — col 1 has 1 at (4,1), eliminates 1 from entire col 1.
	// Box 0 col 2: (0,2), (1,2), (2,2) — col 2 has 1 at (5,2), eliminates 1 from entire col 2.
	// So digit 1 in box 0 → only col 0 → pointing pair in column!

	// Target: (3,0) outside box 0 in col 0.
	_ = board.Set(core.NewPosition(3, 1), 3)
	_ = board.Set(core.NewPosition(3, 2), 4)
	_ = board.Set(core.NewPosition(3, 3), 5)
	_ = board.Set(core.NewPosition(3, 4), 6)
	_ = board.Set(core.NewPosition(3, 5), 7)
	_ = board.Set(core.NewPosition(3, 6), 8)
	_ = board.Set(core.NewPosition(3, 7), 9)

	_ = board.Set(core.NewPosition(6, 0), 3)
	_ = board.Set(core.NewPosition(7, 0), 4)
	_ = board.Set(core.NewPosition(8, 0), 5)

	// (3,0): {1-9} minus row 3 {3,4,5,6,7,8,9} = {1,2} minus col 0 {3,4,5} = {1,2}.
	// Box 3 (rows 3-5, cols 0-2): (3,1)=3, (3,2)=4, (4,1)=1, (5,2)=1.
	// 1 is in box 3! So (3,0) loses 1 → {2}. Naked single again. ✗

	// The problem is that wherever I place 1 to eliminate it from cols 1,2 of box 0,
	// it also appears in the box of my target cell.

	// Solution: use a target further away. (6,0) is box 6 (rows 6-8, cols 0-2).
	// (6,0)=3 already. Use (8,0) = empty instead.
	board = core.NewEmptyBoard()
	_ = board.Set(core.NewPosition(4, 1), 1) // col 1
	_ = board.Set(core.NewPosition(5, 2), 1) // col 2

	// Fill col 0 rows 3-7 to leave (8,0) and box 0 col 0 empty:
	_ = board.Set(core.NewPosition(3, 0), 3)
	_ = board.Set(core.NewPosition(4, 0), 4)
	_ = board.Set(core.NewPosition(5, 0), 5)
	_ = board.Set(core.NewPosition(6, 0), 6)
	_ = board.Set(core.NewPosition(7, 0), 7)

	// Col 0 has {3,4,5,6,7}. (0,0),(1,0),(2,0),(8,0) are empty. ⊆ {1,2,8,9}.

	// Restrict (8,0) to {1,2}:
	_ = board.Set(core.NewPosition(8, 1), 8)
	_ = board.Set(core.NewPosition(8, 2), 9)
	_ = board.Set(core.NewPosition(8, 3), 3)
	_ = board.Set(core.NewPosition(8, 4), 4)
	_ = board.Set(core.NewPosition(8, 5), 5)
	_ = board.Set(core.NewPosition(8, 6), 6)
	_ = board.Set(core.NewPosition(8, 7), 7)
	// Row 8 has {8,9,3,4,5,6,7}. (8,0) ⊆ {1,2} minus col 0 minus box 8.
	// Box 8 (rows 6-8, cols 6-8): no 1. Col 0: no 1. ✓ (8,0)={1,2}.

	// Pointing pair: 1 in box 0 only in col 0 → eliminate from (8,0) → {2}.

	s := solver.NewPointingPairSolver()
	move := s.Apply(&board)

	if move == nil {
		t.Fatal("Expected PointingPairSolver to find a column pointing pair move, got nil")
	}

	if move.Cell.Value != 2 {
		t.Errorf("Expected value 2, got %d", move.Cell.Value)
	}

	if move.Technique != "pointing-pair" {
		t.Errorf("Expected technique 'pointing-pair', got %q", move.Technique)
	}
}

// TestPointingPairSolver_FindsBoxLineReduction builds a board where a candidate
// in a row is confined to one box, and eliminating it from the rest of that box
// creates a naked single.
func TestPointingPairSolver_FindsBoxLineReduction(t *testing.T) {
	board := core.NewEmptyBoard()

	// Eliminate 1 from row 0 cols 3-8 by placing 1 in each column.
	_ = board.Set(core.NewPosition(3, 3), 1)
	_ = board.Set(core.NewPosition(4, 4), 1)
	_ = board.Set(core.NewPosition(5, 5), 1)
	_ = board.Set(core.NewPosition(6, 6), 1)
	_ = board.Set(core.NewPosition(7, 7), 1)
	_ = board.Set(core.NewPosition(8, 8), 1)

	// Now in row 0, digit 1 can only be in cols 0,1,2 (box 0).
	// Box-line reduction: eliminate 1 from box 0 outside row 0.

	// Set up (1,0) to have candidates {1, 2}:
	_ = board.Set(core.NewPosition(1, 1), 3)
	_ = board.Set(core.NewPosition(1, 2), 4)
	_ = board.Set(core.NewPosition(1, 3), 5)
	_ = board.Set(core.NewPosition(1, 4), 6)
	_ = board.Set(core.NewPosition(1, 5), 7)
	_ = board.Set(core.NewPosition(1, 6), 8)
	_ = board.Set(core.NewPosition(1, 7), 9)

	_ = board.Set(core.NewPosition(2, 0), 3)
	_ = board.Set(core.NewPosition(3, 0), 4)
	_ = board.Set(core.NewPosition(4, 0), 5)
	_ = board.Set(core.NewPosition(5, 0), 6)
	_ = board.Set(core.NewPosition(6, 0), 7)
	_ = board.Set(core.NewPosition(7, 0), 8)
	_ = board.Set(core.NewPosition(8, 0), 9)

	s := solver.NewPointingPairSolver()
	move := s.Apply(&board)

	if move == nil {
		t.Fatal("Expected PointingPairSolver to find a box-line reduction move, got nil")
	}

	if move.Cell.Value != 2 {
		t.Errorf("Expected value 2, got %d", move.Cell.Value)
	}

	if move.Technique != "pointing-pair" {
		t.Errorf("Expected technique 'pointing-pair', got %q", move.Technique)
	}
}

// TestPointingPairSolver_NoProgress verifies nil when no pointing pairs exist.
func TestPointingPairSolver_NoProgress(t *testing.T) {
	board := core.NewEmptyBoard()
	s := solver.NewPointingPairSolver()

	move := s.Apply(&board)
	if move != nil {
		t.Errorf("Expected nil for empty board, got %v", move)
	}
}

// TestPointingPairSolver_SolvedBoard verifies nil for a fully solved board.
func TestPointingPairSolver_SolvedBoard(t *testing.T) {
	board := core.NewEmptyBoard()
	bt := solver.NewBacktracker()
	bt.Solve(&board)

	s := solver.NewPointingPairSolver()
	move := s.Apply(&board)
	if move != nil {
		t.Errorf("Expected nil for solved board, got %v", move)
	}
}
