package solver

import "github.com/gnailuy/sudoku/core"

// Default solver object
type DefaultSolver struct {
	BaseSolver
}

// Constructor like function to create a default DefaultSolver object
func NewDefaultSolver() DefaultSolver {
	return DefaultSolver{
		BaseSolver: BaseSolver{
			Name:     "Default Solver",
			Reliable: true,
		},
	}
}

// Internal options for the solve function
type solveOptions struct {
	Randomly       bool  // Randomly generate candidate numbers. When counting solutions, this option is ignored
	CountSolutions bool  // Count the number of solutions instead of returning the first solution, default is false
	RowOrder       []int // Order of rows to generate candidate positions
	ColumnOrder    []int // Order of columns to generate candidate positions
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

// Internal state struct for the recursive backtracking solver
type solveState struct {
	numberOfSolutions int
}

// Function to solve the Sudoku board using backtracking
func solve(board *core.SudokuBoard, state *solveState, options solveOptions) bool {
	for _, row := range options.RowOrder {
		for _, column := range options.ColumnOrder {
			position := core.NewPosition(row, column)

			if board.Get(position) == 0 {
				// When counting solutions, we do not need to generate candidate values randomly
				candidateValues := generateNumberArray(1, 10, !options.CountSolutions && options.Randomly)

				for _, value := range candidateValues {
					// Try to place a value in the cell and solve the board recursively if it is valid
					if board.IsValidInput(position, value) {
						board.Set(position, value)
						if solve(board, state, options) {
							if options.CountSolutions {
								state.numberOfSolutions++ // Collect one solution when the board solved
							} else {
								return true // Return the first solution
							}
						}
						board.Unset(position)
					}
				}
				return false
			}
		}
	}
	return true
}

// Function to solve the Sudoku board with random candidate values
func (solver DefaultSolver) Solve(board *core.SudokuBoard) bool {
	if !board.IsValid() {
		return false
	}

	state := &solveState{}
	return solve(board, state, newSolveOptions(true, false))
}

// Function to count the number of solutions for the Sudoku board
// Note that if the board is already solved, we return 1
func (solver DefaultSolver) CountSolutions(board *core.SudokuBoard) int {
	// If the board is already solved, return 1
	if board.IsSolved() {
		return 1
	}

	// If there is any invalid cell, the board is not solvable, return 0
	if !board.IsValid() {
		return 0
	}

	// If no invalid cell, we can count the number of solutions
	state := &solveState{}
	solve(board, state, newSolveOptions(false, true))
	return state.numberOfSolutions
}
