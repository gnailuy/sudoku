package solver

import "github.com/gnailuy/sudoku/core"

// Base solver object containing the name and reliability of the solver
type BaseSolver struct {
	Name        string // The name of the solver
	Description string // The description of the solver
	Reliable    bool   // If the solver is reliable, it will always solve a valid Sudoku board, otherwise, it may not be able to solve some boards
}

type ISudokuSolver interface {
	Solve(board *core.SudokuBoard) bool
	Hint(board *core.SudokuBoard) *core.Cell
	CountSolutions(board *core.SudokuBoard) int
}
