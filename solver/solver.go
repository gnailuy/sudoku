package solver

import "github.com/gnailuy/sudoku/core"

// Solver defines the shared metadata for all solver implementations.
type Solver interface {
	// GetKey returns the unique identifier of the solver.
	GetKey() string

	// GetDisplayName returns the human-readable name of the solver.
	GetDisplayName() string

	// GetDescription returns a description of the solver's approach.
	GetDescription() string

	// GetWeight returns the difficulty weight per application (for puzzle scoring).
	GetWeight() int
}

// StrategySolver applies a single solving technique. It is not guaranteed
// to fully solve the board — it only handles puzzles within its technique scope.
type StrategySolver interface {
	Solver

	// Apply finds the next move using this technique.
	// Returns nil if the technique cannot make progress.
	Apply(board *core.Board) *Move
}

// TeachingStrategy may expose the first meaningful deduction before solving
// orchestration continues through its candidate eliminations.
type TeachingStrategy interface {
	StrategySolver
	Hint(board *core.Board) *Move
}

// CompleteSolver can fully solve any valid Sudoku board.
type CompleteSolver interface {
	Solver

	// Solve attempts to solve the board in place.
	// Returns false if the board cannot be solved.
	Solve(board *core.Board) bool

	// Hint returns the next determinable move without modifying the board.
	// Returns nil if no hint can be generated.
	Hint(board *core.Board) *Move

	// CountSolutions returns the number of solutions for the board.
	// Returns 0 if unsolvable, 1 if already solved or uniquely solvable.
	CountSolutions(board *core.Board) int
}

// Base provides common fields for all solver implementations.
type Base struct {
	Key         string // The unique key of the solver.
	DisplayName string // The display name of the solver.
	Description string // The description of the solver.
	Weight      int    // Difficulty weight per application (for puzzle scoring).
}

// GetKey returns the key of the solver.
func (b Base) GetKey() string {
	return b.Key
}

// GetDisplayName returns the display name of the solver.
func (b Base) GetDisplayName() string {
	return b.DisplayName
}

// GetDescription returns the description of the solver.
func (b Base) GetDescription() string {
	return b.Description
}

// GetWeight returns the difficulty weight of the solver.
func (b Base) GetWeight() int {
	return b.Weight
}
