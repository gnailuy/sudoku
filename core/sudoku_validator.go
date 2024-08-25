package core

// Function to check if a value can be placed in a specific position.
func (board SudokuBoard) IsValidInput(position Position, value int) bool {
	if !position.IsValid() || value < 1 || value > 9 {
		return false
	}

	// Check the row.
	for i := 0; i < 9; i++ {
		if i != position.Column && board.Get(NewPosition(position.Row, i)) == value {
			return false
		}
	}

	// Check the column.
	for i := 0; i < 9; i++ {
		if i != position.Row && board.Get(NewPosition(i, position.Column)) == value {
			return false
		}
	}

	// Check the 3x3 sub-grid.
	startRow, startColumn := position.Row/3*3, position.Column/3*3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if startRow+i != position.Row && startColumn+j != position.Column &&
				board.Get(NewPosition(startRow+i, startColumn+j)) == value {
				return false
			}
		}
	}

	return true
}

// Function to check if the Sudoku board is valid.
func (board SudokuBoard) IsValid() bool {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			position := NewPosition(i, j)
			value := board.Get(position)
			if value != 0 && !board.IsValidInput(position, value) {
				return false
			}
		}
	}
	return true
}

// Function to check if the Sudoku board is solved.
func (board SudokuBoard) IsSolved() bool {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			position := NewPosition(i, j)
			value := board.Get(position)
			if value == 0 || !board.IsValidInput(position, value) {
				return false
			}
		}
	}
	return true
}

// Function to check if the board is empty.
func (board SudokuBoard) IsEmpty() bool {
	return board.filledCellsCount == 0
}
