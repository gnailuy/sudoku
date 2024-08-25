package solver

import "github.com/gnailuy/sudoku/core"

// Define the base solver object containing the name and reliability of the solver.
type BaseSolver struct {
	Name        string // The name of the solver.
	Description string // The description of the solver.
	Reliable    bool   // If the solver is reliable, it will always solve a valid Sudoku board, otherwise, it may not be able to solve some boards.
}

// Define the interface of a Sudoku solver.
type ISudokuSolver interface {
	// Solve the Sudoku board, return false if the solver cannot solve the board.
	Solve(board *core.SudokuBoard) bool

	// Give a hint for the next step of the board, return nil if the solver cannot give a hint.
	Hint(board *core.SudokuBoard) *core.Cell

	// Count the number of solutions of the board, return 0 if the solver cannot solve the board, return 1 if the board is already solved.
	CountSolutions(board *core.SudokuBoard) int
}
