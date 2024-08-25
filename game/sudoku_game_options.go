package game

import (
	"github.com/gnailuy/sudoku/solver"
)

// Define the Sudoku game options struct.
type SudokuGameOptions struct {
	// Public fields.
	StrategySolverKeys []string

	// Private fields.
	solverStore solver.SudokuSolverStore
}

// Constructor like function to create a default options object.
func NewDefaultSudokuGameOptions(solverStore solver.SudokuSolverStore) SudokuGameOptions {
	return SudokuGameOptions{
		StrategySolverKeys: []string{},
		solverStore:        solverStore,
	}
}

// Function to get the strategy solvers from the store.
func (options *SudokuGameOptions) GetStrategySolvers() []solver.ISudokuSolver {
	strategySolvers := []solver.ISudokuSolver{}

	for _, key := range options.StrategySolverKeys {
		solver := options.solverStore.GetSolverByKey(key)
		if solver != nil {
			strategySolvers = append(strategySolvers, solver)
		} else {
			panic("Bug: Invalid solver key: " + key)
		}
	}

	return strategySolvers
}
