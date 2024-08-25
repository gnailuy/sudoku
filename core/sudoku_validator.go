package core

// Function to check if a number can be placed in a specific cell
func (board *SudokuBoard) IsValidInput(cell Cell, number int) bool {
	// Check the row
	for i := 0; i < 9; i++ {
		if i != cell.Column && board.Get(NewCell(cell.Row, i)) == number {
			return false
		}
	}

	// Check the column
	for i := 0; i < 9; i++ {
		if i != cell.Row && board.Get(NewCell(i, cell.Column)) == number {
			return false
		}
	}

	// Check the 3x3 sub-grid
	startRow, startColumn := cell.Row/3*3, cell.Column/3*3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if startRow+i != cell.Row && startColumn+j != cell.Column &&
				board.Get(NewCell(startRow+i, startColumn+j)) == number {
				return false
			}
		}
	}

	return true
}

// Function to check if the Sudoku board is valid
func (board *SudokuBoard) IsValid() bool {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			cell := NewCell(i, j)
			number := board.Get(cell)
			if number != 0 && !board.IsValidInput(cell, number) {
				return false
			}
		}
	}
	return true
}

// Function to check if the Sudoku board is solved
func (board *SudokuBoard) IsSolved() bool {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			cell := NewCell(i, j)
			number := board.Get(cell)
			if number == 0 || !board.IsValidInput(cell, number) {
				return false
			}
		}
	}
	return true
}

// Function to check if the board is empty
func (board *SudokuBoard) IsEmpty() bool {
	return board.filledCells == 0
}
