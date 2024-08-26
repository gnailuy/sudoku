package generator

import "github.com/gnailuy/sudoku/solver"

// Define the options to generate a Sudoku problem.
type SudokuGeneratorOptions struct {
	// Public fields.
	MaximumSolutions  int
	MaximumIterations int
	Difficulty        SudokuDifficulty

	// Private fields.
	solverStore solver.SudokuSolverStore
}

// Constructor like function to create a default options object.
func NewSudokuProblemOptions(solverStore solver.SudokuSolverStore, difficulty SudokuDifficulty) SudokuGeneratorOptions {
	return SudokuGeneratorOptions{
		MaximumSolutions:  1,
		MaximumIterations: 1024,
		Difficulty:        difficulty,
		solverStore:       solverStore,
	}
}
