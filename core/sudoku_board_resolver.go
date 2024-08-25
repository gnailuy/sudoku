package core

// Function to check if a number can be placed in a specific cell
func (board *SudokuBoard) isValid(row, col, num int) bool {
	// Check the row
	for i := 0; i < 9; i++ {
		if board.Get(row, i) == num {
			return false
		}
	}

	// Check the column
	for i := 0; i < 9; i++ {
		if board.Get(i, col) == num {
			return false
		}
	}

	// Check the 3x3 sub-grid
	startRow, startCol := row/3*3, col/3*3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if board.Get(startRow+i, startCol+j) == num {
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
		for col := 0; col < 9; col++ {
			if board.Get(row, col) == 0 {
				// When counting solutions, we do not need to generate candidate numbers randomly
				candidateNumbers := generateCellCandidates(!option.CountSolutions && option.Randomly)
				for _, num := range candidateNumbers {
					if board.isValid(row, col, num) {
						board.Set(row, col, num)
						if board.solve(option) {
							if option.CountSolutions {
								board.numberOfSolutions++
							} else {
								return true
							}
						}
						board.Unset(row, col)
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

// Export function to count the number of solutions for the Sudoku board
func (board *SudokuBoard) CountSolutions() int {
	board.numberOfSolutions = 0
	board.solve(solveOptions{
		CountSolutions: true,
	})
	return board.numberOfSolutions
}
