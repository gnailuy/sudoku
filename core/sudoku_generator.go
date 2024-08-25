package core

// Options to generate a Sudoku problem
type SudokuProblemOptions struct {
	MinimumFilledCells int
	MaximumIterations  int
	MaximumSolutions   int
}

// Constructor like function to create a default SudokuProblemOptions
func NewDefaultSudokuProblemOptions() SudokuProblemOptions {
	return SudokuProblemOptions{
		MinimumFilledCells: 17,
		MaximumIterations:  60,
		MaximumSolutions:   1,
	}
}

// Function to generate a solved Sudoku board by solving an empty board randomly
func GenerateSolvedBoard() SudokuBoard {
	board := NewEmptySudokuBoard()
	board.SolveRandomly()
	return board
}

// Function to generate a Sudoku problem from a solved board
func (solvedBoard SudokuBoard) GenerateSudokuProblem(options SudokuProblemOptions) SudokuBoard {
	// Make a copy of the solved board in case the original board is needed somewhere else
	board := solvedBoard

	// Initially, all cells are filled
	nonEmptyCells := make([]Cell, 0)
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			nonEmptyCells = append(nonEmptyCells, NewCell(row, col))
		}
	}

	// Remove numbers randomly from the solved board to create a problem
	for i := 0; i < options.MaximumIterations; i++ {
		// Stop removing numbers because the board has reached the minimum number of filled cells
		if board.filledCells <= options.MinimumFilledCells {
			break
		}

		// Stop removing numbers because it is impossible to have a unique solution with less than 17 filled cells
		if options.MaximumSolutions == 1 && board.filledCells <= 17 {
			break
		}

		// Test the non-empty cells in a random order and remove the first cell that can be removed
		shuffleArray(nonEmptyCells)

		removedCellIndex := -1
		for j, cell := range nonEmptyCells {
			// Temporarily store the cell value
			originalNumber := board.Get(cell)

			// Update the board
			board.Unset(cell)

			// If the problem is solvable and has no more than maximum solutions, confirm the removal
			numberOfSolutions := board.CountSolutions()
			if numberOfSolutions > 0 && numberOfSolutions <= options.MaximumSolutions {
				removedCellIndex = j
				break
			}

			// If the problem is not solvable or has more than maximum solutions, revert the removal
			board.Set(cell, originalNumber)
		}

		// Remove the cell from the non-empty cells list
		if removedCellIndex >= 0 {
			nonEmptyCells = append(nonEmptyCells[:removedCellIndex], nonEmptyCells[removedCellIndex+1:]...)
		} else {
			// We did not find any cell to remove in this iteration, so we stop the process
			break
		}
	}

	return board
}
