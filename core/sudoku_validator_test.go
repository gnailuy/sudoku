package core

import "testing"

// Function to test the IsValidInput function.
func TestIsValidInput(t *testing.T) {
	board := NewEmptySudokuBoard()
	board.Set(NewPosition(4, 4), 1)

	// Test the row.
	if board.IsValidInput(NewPosition(4, 0), 1) {
		t.Error("The value 1 cannot be placed in the same row")
	}

	// Test the column.
	if board.IsValidInput(NewPosition(0, 4), 1) {
		t.Error("The value 1 cannot be placed in the same column")
	}

	// Test the 3x3 sub-grid.
	if board.IsValidInput(NewPosition(3, 3), 1) {
		t.Error("The value 1 cannot be placed in the same 3x3 sub-grid")
	}

	// Test the same position.
	if !board.IsValidInput(NewPosition(4, 4), 1) {
		t.Error("The same value 1 can be placed in the same position")
	}

	// Test the valid position.
	if !board.IsValidInput(NewPosition(0, 0), 1) {
		t.Error("The value 1 can be placed in the position (0, 0)")
	}
}

// Function to test the IsValid function.
func TestIsValid(t *testing.T) {
	board := NewEmptySudokuBoard()
	board.FromString("583.67..46723.48...4.8253.6934..852.2.74519.3851.3.4673..589742.952461.84.87..659")

	// Test the current board is valid.
	if !board.IsValid() {
		t.Error("The board is valid")
	}

	// Test the invalid board.
	board.Set(NewPosition(2, 0), 5)
	if board.IsValid() {
		t.Error("The board is invalid")
	}
}

// Function to test the IsSolved function.
func TestIsSolved(t *testing.T) {
	board := NewEmptySudokuBoard()
	board.FromString("583167294672394815149825376934678521267451983851932467316589742795246138428713659")

	// Test the current board is solved.
	if !board.IsSolved() {
		t.Error("The board is solved")
	}

	// Test the filled but invalid board.
	board.Set(NewPosition(0, 0), 1)
	if board.IsSolved() {
		t.Error("The board is filled but not solved")
	}
}
