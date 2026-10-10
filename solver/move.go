package solver

import (
	"fmt"

	"github.com/gnailuy/sudoku/core"
)

// Move represents a solver's recommended action on the board, including
// which technique found it and a human-readable explanation.
type Move struct {
	Cell            core.Cell // The cell to fill (position + value). Zero value for elimination-only moves.
	Technique       string    // Technique identifier, e.g. "backtracker", "naked-single".
	Reason          string    // Human-readable explanation for hints.
	EliminationOnly bool      // True if this move only eliminated candidates without placing a value.
	Evidence        *Evidence // Typed strategy facts used to compose a renderer-neutral teaching plan.
}

// IsPlacement reports whether this move places a value on the board.
func (m Move) IsPlacement() bool {
	return !m.EliminationOnly && m.Cell.Value != 0
}

// String returns a display-friendly representation of the move.
func (m Move) String() string {
	return fmt.Sprintf("%s at %s → %d (%s)",
		m.Technique, m.Cell.Position.ToString(), m.Cell.Value, m.Reason)
}
