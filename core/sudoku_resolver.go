package core

// Function to check if a number can be placed in a specific cell
func (board *SudokuBoard) isValid(cell Cell, number int) bool {
	// Check the row
	for i := 0; i < 9; i++ {
		if board.Get(NewCell(cell.Row, i)) == number {
			return false
		}
	}

	// Check the column
	for i := 0; i < 9; i++ {
		if board.Get(NewCell(i, cell.Column)) == number {
			return false
		}
	}

	// Check the 3x3 sub-grid
	startRow, startColumn := cell.Row/3*3, cell.Column/3*3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if board.Get(NewCell(startRow+i, startColumn+j)) == number {
				return false
			}
		}
	}

	return true
}

// Options for the solve function
type solveOptions struct {
	Randomly       bool // Randomly generate candidate numbers. When counting solutions, this option is ignored
	CountSolutions bool // Count the number of solutions instead of returning the first solution, default is false
}

// Function to solve the Sudoku board using backtracking
func (board *SudokuBoard) solve(option solveOptions) bool {
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			cell := NewCell(row, column)

			if board.Get(cell) == 0 {
				// When counting solutions, we do not need to generate candidate numbers randomly
				candidateNumbers := generateCellCandidates(!option.CountSolutions && option.Randomly)

				for _, num := range candidateNumbers {
					// Try to place a number in the cell and solve the board recursively if it is valid
					if board.isValid(cell, num) {
						board.Set(cell, num)
						if board.solve(option) {
							if option.CountSolutions {
								board.numberOfSolutions++ // Collect one solution when the board solved
							} else {
								return true // Return the first solution
							}
						}
						board.Unset(cell)
					}
				}
				return false
			}
		}
	}
	return true
}

// Exported function to solve the Sudoku board using backtracking
func (board *SudokuBoard) Solve() {
	board.solve(solveOptions{
		Randomly: false,
	})
}

// Exported function to solve the Sudoku board using backtracking with random candidate numbers
func (board *SudokuBoard) SolveRandomly() {
	board.solve(solveOptions{
		Randomly: true,
	})
}

// Exported function to count the number of solutions for the Sudoku board
func (board *SudokuBoard) CountSolutions() int {
	board.numberOfSolutions = 0
	board.solve(solveOptions{
		CountSolutions: true,
	})
	return board.numberOfSolutions
}
