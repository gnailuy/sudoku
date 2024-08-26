package core

// Function to normalize a Sudoku board.
func (board *SudokuBoard) Normalize() {
	if !board.IsSolved() {
		panic("Bug: Cannot normalize an unsolved board")
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
