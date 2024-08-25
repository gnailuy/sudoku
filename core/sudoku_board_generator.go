package core

// Generate a solved Sudoku board randomly by solving an empty board
func generateSolvedBoard() SudokuBoard {
	var board SudokuBoard
	board.SolveRandomly()
	return board
}

// Exported function to generate a Sudoku problem
func GenerateSudokuProblem(iteration int) SudokuBoard {
	board := generateSolvedBoard()

	// Remove numbers from the solved board to create a problem
	for i := 0; i < iteration; i++ {
		// Stop removing numbers because it is impossible to have a unique solution with less than 17 filled cells
		if board.filledCells < 17 {
			break
		}

		// Find a non-zero cell to remove
		row, col := generateRandomNumber(0, 9), generateRandomNumber(0, 9)
		for board.Get(row, col) == 0 {
			row, col = generateRandomNumber(0, 9), generateRandomNumber(0, 9)
		}

		// Temporarily store the cell value
		originalValue := board.Get(row, col)

		// Update the board and reset the number of solutions
		board.Unset(row, col)

		// Make sure the board has a unique solution, otherwise revert the change
		if board.CountSolutions() > 1 {
			board.Set(row, col, originalValue)
		}
	}

	return board
}
