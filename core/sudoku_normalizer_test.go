package core

import (
	"testing"
)

// Test the Normalize function.
func TestNormalize(t *testing.T) {
	board := NewEmptySudokuBoard()
	board.FromString("583167294672394815149825376934678521267451983851932467316589742795246138428713659")

	// Normalize the board.
	board.Normalize()

	// Check the board.
	for i := 0; i < 9; i++ {
		if board.Get(NewPosition(0, i)) != i+1 {
			t.Errorf("Normalization failed at position (0, %d)", i)
		}
	}

	if !board.IsSolved() {
		t.Error("Normalization failed: the board is not solved after normalization")
	}

	// Check the board string.
	if board.ToString() != "123456789567389241498271365839562174756914823214837956345128697681795432972643518" {
		t.Error("Normalization failed: the board string is not correct")
	}
}
