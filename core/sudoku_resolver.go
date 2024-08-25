package core

// Function to check if a number can be placed in a specific cell
func (board *SudokuBoard) isValidInput(cell Cell, number int) bool {
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
			if startRow+i != cell.Row &&
				startColumn+j != cell.Column &&
				board.Get(NewCell(startRow+i, startColumn+j)) == number {
				return false
			}
		}
	}

	return true
}

// Internal options for the solve function
type solveOptions struct {
	Randomly       bool  // Randomly generate candidate numbers. When counting solutions, this option is ignored
	CountSolutions bool  // Count the number of solutions instead of returning the first solution, default is false
	RowOrder       []int // Order of rows to generate candidate cells
	ColumnOrder    []int // Order of columns to generate candidate cells
}

// Constructor like function to create a new solveOptions object
func newSolveOptions(randomly, countSolutions bool) solveOptions {
	return solveOptions{
		Randomly:       randomly,
		CountSolutions: countSolutions,
		RowOrder:       generateNumberArray(0, 9, randomly),
		ColumnOrder:    generateNumberArray(0, 9, randomly),
	}
}

// Function to solve the Sudoku board using backtracking
func (board *SudokuBoard) solve(option solveOptions) bool {
	for _, row := range option.RowOrder {
		for _, column := range option.ColumnOrder {
			cell := NewCell(row, column)

			if board.Get(cell) == 0 {
				// When counting solutions, we do not need to generate candidate numbers randomly
				candidateNumbers := generateNumberArray(1, 10, !option.CountSolutions && option.Randomly)

				for _, num := range candidateNumbers {
					// Try to place a number in the cell and solve the board recursively if it is valid
					if board.isValidInput(cell, num) {
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

// Function to check if the Sudoku board is solved
func (board *SudokuBoard) IsSolved() bool {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			cell := NewCell(i, j)
			number := board.Get(cell)
			if number == 0 || !board.isValidInput(cell, number) {
				return false
			}
		}
	}
	return true
}

// Function to solve the Sudoku board using backtracking
func (board *SudokuBoard) Solve() {
	board.solve(newSolveOptions(false, false))
}

// Function to solve the Sudoku board using backtracking with random candidate numbers
func (board *SudokuBoard) SolveRandomly() {
	board.solve(newSolveOptions(true, false))
}

// Function to count the number of solutions for the Sudoku board
// Note that if the board is solved, we return 1
func (board *SudokuBoard) CountSolutions() int {
	// If the board is already solved, return 1
	if board.IsSolved() {
		return 1
	}

	// If there is any invalid cell, the board is not solvable, return 0
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			cell := NewCell(i, j)
			number := board.Get(cell)
			if number != 0 && !board.isValidInput(cell, number) {
				return 0
			}
		}
	}

	// If no invalid cell, we can count the number of solutions
	board.numberOfSolutions = 0
	board.solve(newSolveOptions(false, true))
	return board.numberOfSolutions
}

// Function to check if the Sudoku board is solvable
func (board *SudokuBoard) IsSolvable() bool {
	return board.CountSolutions() > 0
}
