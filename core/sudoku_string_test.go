package core

import "testing"

// Test the ToString function.
func TestToString(t *testing.T) {
	board := NewEmptySudokuBoard()

	board.SetCell(NewCell(NewPosition(0, 0), 1))
	board.SetCell(NewCell(NewPosition(4, 4), 5))
	board.SetCell(NewCell(NewPosition(8, 8), 9))

	expected := "1.......................................5.......................................9"

	if board.ToString() != expected {
		t.Errorf("Expected string: %s, got %s", expected, board.ToString())
	}
}

// Test the FromString function.
func TestFromString(t *testing.T) {
	board := NewEmptySudokuBoard()
	board.FromString("1.......................................5.......................................9")

	if board.Get(NewPosition(0, 0)) != 1 {
		t.Errorf("Expected 1 at (0, 0), got %d", board.Get(NewPosition(0, 0)))
	}

	if board.Get(NewPosition(4, 4)) != 5 {
		t.Errorf("Expected 5 at (4, 4), got %d", board.Get(NewPosition(4, 4)))
	}

	if board.Get(NewPosition(8, 8)) != 9 {
		t.Errorf("Expected 9 at (8, 8), got %d", board.Get(NewPosition(8, 8)))
	}
}

// Test the FromString function with an invalid string.
func TestFromStringInvalid(t *testing.T) {
	board := NewEmptySudokuBoard()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic on an invalid string")
		}
	}()

	board.FromString("1.......................................x.......................................9")
}
