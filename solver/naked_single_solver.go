package solver

import (
	"fmt"

	"github.com/gnailuy/sudoku/core"
)

// NakedSingleSolver finds cells with exactly one candidate remaining.
// A naked single occurs when all other values in a cell's row, column,
// and box are already filled, leaving only one possibility.
type NakedSingleSolver struct {
	Base
}

// NewNakedSingleSolver creates a NakedSingleSolver and returns it.
func NewNakedSingleSolver() *NakedSingleSolver {
	return &NakedSingleSolver{
		Base: Base{
			Key:         "naked-single",
			DisplayName: "Naked Single",
			Description: "Finds cells where all but one candidate have been eliminated by row, column, and box peers.",
			Weight:      WeightNakedSingle,
		},
	}
}

// Apply scans all empty cells and returns the first one with exactly one candidate.
func (s *NakedSingleSolver) Apply(board *core.Board) *Move {
	for _, pos := range board.EmptyPositions() {
		candidates := board.Candidates(pos)
		if candidates.Count() == 1 {
			value := candidates.Values()[0]
			return &Move{
				Cell:      core.NewCell(pos, value),
				Technique: s.Key,
				Reason:    fmt.Sprintf("%s is the only candidate for %s", digitName(value), pos.ToString()),
				Evidence: &Evidence{Premises: []CandidateGroup{{
					Position: pos,
					Values:   []int{value},
				}}},
			}
		}
	}

	return nil
}

// digitName returns a human-friendly name for a digit.
func digitName(value int) string {
	return fmt.Sprintf("%d", value)
}
