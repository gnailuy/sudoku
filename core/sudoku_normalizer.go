package core

import "github.com/gnailuy/sudoku/util"

// Function to normalize a Sudoku board.
func (board *SudokuBoard) Normalize() {
	if !board.IsSolved() {
		panic("Bug: Normalizing an unsolved board is not allowed")
	}

	// Make a copy of the board.
	boardCopy := board.Copy()

	// Normalize the board to the smallest representation.
	for i := 0; i < 9; i++ {
		originalValue := boardCopy.Get(NewPosition(0, i))
		targetValue := i + 1

		for j := 0; j < 9; j++ {
			for k := 0; k < 9; k++ {
				if boardCopy.Get(NewPosition(j, k)) == originalValue {
					board.Set(NewPosition(j, k), targetValue)
				}
			}
		}
	}
}

// Function to randomize a normalized Sudoku board.
func (board *SudokuBoard) Randomize() {
	// Make a copy of the board.
	boardCopy := board.Copy()

	// Randomize the board with the below replacement plan.
	randomArray := util.GenerateNumberArray(1, 10, true)

	for j := 0; j < 9; j++ {
		for k := 0; k < 9; k++ {
			position := NewPosition(j, k)

			if boardCopy.Get(position) != 0 {
				targetValue := randomArray[boardCopy.Get(position)-1]
				board.Set(position, targetValue)
			}
		}
	}
}
