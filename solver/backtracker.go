package solver

import (
	"fmt"
	"math/rand"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/util"
)

// Backtracker implements a recursive backtracking solver.
// It satisfies CompleteSolver — it can fully solve any valid board.
type Backtracker struct {
	Base
}

// NewBacktracker creates a new backtracking solver.
func NewBacktracker() Backtracker {
	return Backtracker{
		Base{
			Key:         "default",
			DisplayName: "Backtracking Solver",
			Description: `Default solver using recursive backtracking in a random order.`,
		},
	}
}

// Define the internal options for the solve function.
type solveOptions struct {
	Randomly       bool       // Randomly generate candidate numbers. When counting solutions, this option is ignored.
	HintOnly       bool       // Only generate a solve path for hint generation without solving the board.
	CountSolutions bool       // Count the number of solutions instead of returning the first solution, default is false.
	RowOrder       []int      // Order of rows to generate candidate positions.
	ColumnOrder    []int      // Order of columns to generate candidate positions.
	Random         *rand.Rand // Optional deterministic source for randomized solving.
}

// Constructor like function to create a new solveOptions object.
func newSolveOptions(randomly, hintOnly, countSolutions bool) solveOptions {
	return newSolveOptionsWithRand(randomly, hintOnly, countSolutions, nil)
}

func newSolveOptionsWithRand(randomly, hintOnly, countSolutions bool, source *rand.Rand) solveOptions {
	return solveOptions{
		Randomly:       randomly,
		HintOnly:       hintOnly,
		CountSolutions: countSolutions,
		RowOrder:       util.GenerateNumberArrayWithRand(0, 9, randomly, source),
		ColumnOrder:    util.GenerateNumberArrayWithRand(0, 9, randomly, source),
		Random:         source,
	}
}

// Internal state struct for the recursive backtracking solver.
type solveState struct {
	numberOfSolutions int
	solvePath         []core.Cell
}

// Function to solve the Sudoku board using backtracking.
func solve(board *core.Board, state *solveState, options solveOptions) bool {
	for _, row := range options.RowOrder {
		for _, column := range options.ColumnOrder {
			position := core.NewPosition(row, column)

			if board.Get(position) == 0 {
				// Compute candidate values for this cell.
				candidateValues := board.Candidates(position).Values()

				// When solving randomly (not counting), shuffle candidates.
				if !options.CountSolutions && options.Randomly {
					util.ShuffleArrayWithRand(candidateValues, options.Random)
				}

				for _, value := range candidateValues {
					_ = board.Set(position, value) // value is a valid candidate
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

// Solve solves the board in place using backtracking with random candidate order.
func (s Backtracker) Solve(board *core.Board) bool {
	return s.SolveWithRand(board, nil)
}

// SolveWithRand solves with source when non-nil, allowing reproducible
// experiment generation without changing interactive random behavior.
func (s Backtracker) SolveWithRand(board *core.Board, source *rand.Rand) bool {
	if !board.IsValid() {
		return false
	}

	state := &solveState{}
	return solve(board, state, newSolveOptionsWithRand(true, false, false, source))
}

// SolveDeterministic solves the board in place with stable candidate ordering
// and a minimum-remaining-values search. It is intended for repeatable
// verification and callers that do not need randomized full-board generation.
func (s Backtracker) SolveDeterministic(board *core.Board) bool {
	if !board.IsValid() {
		return false
	}

	return solveDeterministic(board)
}

func solveDeterministic(board *core.Board) bool {
	var position core.Position
	var candidates []int
	found := false

search:
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			candidatePosition := core.NewPosition(row, column)
			if board.Get(candidatePosition) != 0 {
				continue
			}

			candidateValues := board.Candidates(candidatePosition).Values()
			if len(candidateValues) == 0 {
				return false
			}
			if !found || len(candidateValues) < len(candidates) {
				position = candidatePosition
				candidates = candidateValues
				found = true
				if len(candidates) == 1 {
					break search
				}
			}
		}
	}

	if !found {
		return true
	}

	for _, value := range candidates {
		_ = board.Set(position, value)
		if solveDeterministic(board) {
			return true
		}
		board.Unset(position)
	}

	return false
}

// Hint returns the next determinable move without modifying the board.
func (s Backtracker) Hint(board *core.Board) *Move {
	if !board.IsValid() {
		return nil
	}

	state := &solveState{}
	solve(board, state, newSolveOptions(true, true, false))

	if len(state.solvePath) > 0 {
		cell := state.solvePath[0]
		return &Move{
			Cell:      cell,
			Technique: "backtracker",
			Reason:    fmt.Sprintf("backtracking finds %d at %s", cell.Value, cell.Position.ToString()),
		}
	}

	return nil
}

// CountSolutions returns the number of solutions for the board.
func (s Backtracker) CountSolutions(board *core.Board) int {
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
