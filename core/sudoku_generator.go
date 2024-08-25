package core

// Generate a solved Sudoku board randomly by solving an empty board
func GenerateSolvedBoard() SudokuBoard {
	board := NewEmptySudokuBoard()
	board.SolveRandomly()
	return board
}

// Exported function to generate a Sudoku problem from a solved board
func (solvedBoard SudokuBoard) GenerateSudokuProblem(iteration int) SudokuBoard {
	board := solvedBoard

	// Remove numbers from the solved board to create a problem
	for i := 0; i < iteration; i++ {
		// Stop removing numbers because it is impossible to have a unique solution with less than 17 filled cells
		if board.filledCells < 17 {
			break
		}

		// Find a non-zero cell to remove
		cell := NewCell(generateRandomNumber(0, 9), generateRandomNumber(0, 9))
		for board.Get(cell) == 0 {
			cell = NewCell(generateRandomNumber(0, 9), generateRandomNumber(0, 9))
		}

		// Temporarily store the cell value
		originalNumber := board.Get(cell)

		// Update the board
		board.Unset(cell)

		// Make sure the board has a unique solution, otherwise revert the change
		if board.CountSolutions() > 1 {
			board.Set(cell, originalNumber)
		}
	}

	return board
}
