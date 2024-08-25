package generator

import "github.com/gnailuy/sudoku/solver"

// Define the options to generate a Sudoku problem.
type SudokuGeneratorOptions struct {
	// Public fields.
	MinimumFilledCells int
	MaximumIterations  int
	MaximumSolutions   int
	SolverKeys         []string

	// Private fields.
	solverStore solver.SudokuSolverStore
}

// Constructor like function to create a default options object.
func NewDefaultSudokuProblemOptions(solverStore solver.SudokuSolverStore) SudokuGeneratorOptions {
	return SudokuGeneratorOptions{
		MinimumFilledCells: 17,
		MaximumIterations:  60,
		MaximumSolutions:   1,
		SolverKeys:         []string{"default"},
		solverStore:        solverStore,
	}
}
