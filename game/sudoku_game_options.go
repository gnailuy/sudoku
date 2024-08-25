package game

import (
	"github.com/gnailuy/sudoku/solver"
)

// Define the Sudoku game options struct.
type SudokuGameOptions struct {
	// Public fields.
	HintSolverKeys []string

	// Private fields.
	solverStore solver.SudokuSolverStore
}

// Constructor like function to create a default options object.
func NewDefaultSudokuGameOptions(solverStore solver.SudokuSolverStore) SudokuGameOptions {
	return SudokuGameOptions{
		HintSolverKeys: []string{},
		solverStore:    solverStore,
	}
}

// Function to get the hint solvers from the store.
func (options *SudokuGameOptions) GetHintSolvers() []solver.ISudokuSolver {
	hintSolvers := []solver.ISudokuSolver{}

	for _, key := range options.HintSolverKeys {
		solver := options.solverStore.GetSolverByKey(key)
		if solver != nil {
			hintSolvers = append(hintSolvers, solver)
		} else {
			panic("Bug: Invalid solver key: " + key)
		}
	}

	return hintSolvers
}
