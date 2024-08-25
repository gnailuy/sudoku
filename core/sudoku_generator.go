package core

// Options to generate a Sudoku problem
type SudokuProblemOptions struct {
	MinimumFilledCells int
	MaximumIterations  int
	MaximumSolutions   int
}

// Constructor like function to create a new SudokuProblemOptions
func NewDefaultSudokuProblemOptions() SudokuProblemOptions {
	return SudokuProblemOptions{
		MinimumFilledCells: 17,
		MaximumIterations:  100,
		MaximumSolutions:   1,
	}
}

// Generate a solved Sudoku board randomly by solving an empty board
func GenerateSolvedBoard() SudokuBoard {
	board := NewEmptySudokuBoard()
	board.SolveRandomly()
	return board
}

// Exported function to generate a Sudoku problem from a solved board
func (solvedBoard SudokuBoard) GenerateSudokuProblem(options SudokuProblemOptions) SudokuBoard {
	board := solvedBoard

	// Remove numbers from the solved board to create a problem
	for i := 0; i < options.MaximumIterations; i++ {
		// Stop removing numbers because the board has reached the minimum number of filled cells
		if board.filledCells <= options.MinimumFilledCells {
			break
		}

		// Stop removing numbers because it is impossible to have a unique solution with less than 17 filled cells
		if options.MaximumSolutions == 1 && board.filledCells <= 17 {
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

		// Make sure the problem has no more than maximum solutions
		if board.CountSolutions() > options.MaximumSolutions {
			board.Set(cell, originalNumber)

			// Stop removing numbers as we have reached the maximum number of solutions
			break
		}
	}

	return board
}
