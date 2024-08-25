package solver

import "github.com/gnailuy/sudoku/core"

// Define the interface of a Sudoku solver.
type ISudokuSolver interface {
	// Return the key of the solver.
	GetKey() string

	// Return the display name of the solver.
	GetDisplayName() string

	// Return the description of the solver.
	GetDescription() string

	// Return if the solver is reliable.
	IsReliable() bool

	// Solve the Sudoku board, return false if the solver cannot fully solve the board.
	Solve(board *core.SudokuBoard) bool

	// Give a hint for the next step of the board, return nil if the solver cannot give a hint.
	Hint(board *core.SudokuBoard) *core.Cell

	// Count the number of solutions of the board. Return 0 if the solver cannot solve the board; return 1 if the board is already solved.
	CountSolutions(board *core.SudokuBoard) int
}

// Define the base solver embedding the key and other properties.
type BaseSolver struct {
	Key         string // The unique key of the solver.
	DisplayName string // The display name of the solver.
	Description string // The description of the solver.
	Reliable    bool   // If the solver is reliable, it will always solve a valid Sudoku board, otherwise, it may not be able to solve some boards.
}

// Function to get the key of the base solver.
func (solver BaseSolver) GetKey() string {
	return solver.Key
}

// Function to get the display name of the base solver.
func (solver BaseSolver) GetDisplayName() string {
	return solver.DisplayName
}

// Function to get the description of the base solver.
func (solver BaseSolver) GetDescription() string {
	return solver.Description
}

// Function to check if the base solver is reliable.
func (solver BaseSolver) IsReliable() bool {
	return solver.Reliable
}

// Function to implement the default solution counting logic on the base solver.
func (solver BaseSolver) CountSolutions(board *core.SudokuBoard) int {
	// Reliable solvers should always override this function.
	if solver.Reliable {
		panic("Bug: Reliable solver should override the CountSolutions function")
	}

	// Unreliable solvers should return 0 as they may not be able to fully solve the board.
	return 0
}
