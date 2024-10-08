package solver

import (
	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/util"
)

// Define the default solver object.
type DefaultSolver struct {
	BaseSolver
}

// Constructor like function to create a default DefaultSolver object.
func NewDefaultSolver() DefaultSolver {
	return DefaultSolver{
		BaseSolver{
			Key:         "default",
			DisplayName: "Default Solver",
			Description: `Default solver using recursive backtracking in a random order.`,
			Reliable:    true,
		},
	}
}

// Define the internal options for the solve function.
type solveOptions struct {
	Randomly       bool  // Randomly generate candidate numbers. When counting solutions, this option is ignored.
	HintOnly       bool  // Only generate a solve path for hint generation without solving the board.
	CountSolutions bool  // Count the number of solutions instead of returning the first solution, default is false.
	RowOrder       []int // Order of rows to generate candidate positions.
	ColumnOrder    []int // Order of columns to generate candidate positions.
}

// Constructor like function to create a new solveOptions object.
func newSolveOptions(randomly, hintOnly, countSolutions bool) solveOptions {
	return solveOptions{
		Randomly:       randomly,
		HintOnly:       hintOnly,
		CountSolutions: countSolutions,
		RowOrder:       util.GenerateNumberArray(0, 9, randomly),
		ColumnOrder:    util.GenerateNumberArray(0, 9, randomly),
	}
}

// Internal state struct for the recursive backtracking solver.
type solveState struct {
	numberOfSolutions int
	solvePath         []core.Cell
}

// Function to solve the Sudoku board using backtracking.
func solve(board *core.SudokuBoard, state *solveState, options solveOptions) bool {
	for _, row := range options.RowOrder {
		for _, column := range options.ColumnOrder {
			position := core.NewPosition(row, column)

			if board.Get(position) == 0 {
				// When counting solutions, we do not need to generate candidate values randomly.
				candidateValues := util.GenerateNumberArray(1, 10, !options.CountSolutions && options.Randomly)

				for _, value := range candidateValues {
					// Try to place a value in the cell and solve the board recursively if it is valid.
					if board.IsValidInput(position, value) {
						board.Set(position, value)
						state.solvePath = append(state.solvePath, core.NewCell(position, value))

						if solve(board, state, options) {
							if options.CountSolutions {
								// Collect one solution when the board solved.
								state.numberOfSolutions++
							} else {
								// Return the first solution.
								return true
							}
						}

						board.Unset(position)
						state.solvePath = state.solvePath[:len(state.solvePath)-1]
					}
				}

				return false
			}
		}
	}

	// If we are only generating hints, restore the board to the original state.
	if options.HintOnly {
		for _, cell := range state.solvePath {
			board.Unset(cell.Position)
		}
	}

	return true
}

// Function to solve the Sudoku board with random candidate values.
func (solver DefaultSolver) Solve(board *core.SudokuBoard) bool {
	if !board.IsValid() {
		return false
	}

	state := &solveState{}
	return solve(board, state, newSolveOptions(true, false, false))
}

// Function to generate a hint for the Sudoku board without solving the board.
func (solver DefaultSolver) Hint(board *core.SudokuBoard) *core.Cell {
	if !board.IsValid() {
		return nil
	}

	state := &solveState{}
	solve(board, state, newSolveOptions(true, true, false))

	if len(state.solvePath) > 0 {
		return &state.solvePath[0]
	}

	return nil
}

// Function to count the number of solutions for the Sudoku board.
// Note that if the board is already solved, we return 1 as doing nothing is also a solution.
func (solver DefaultSolver) CountSolutions(board *core.SudokuBoard) int {
	// If the board is already solved, return 1.
	if board.IsSolved() {
		return 1
	}

	// If there is any invalid cell, the board is not solvable, return 0.
	if !board.IsValid() {
		return 0
	}

	// If no invalid cell, we can count the number of solutions.
	state := &solveState{}
	solve(board, state, newSolveOptions(false, false, true))
	return state.numberOfSolutions
}
