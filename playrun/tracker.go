// Package playrun centralizes per-play-run outcome tracking around game actions.
package playrun

import (
	"fmt"

	"github.com/gnailuy/sudoku/game"
)

// CompletionRecorder persists a completion for an existing normalized puzzle.
type CompletionRecorder interface {
	RecordCompletion(string) (bool, error)
}

// Tracker records at most one player-driven completion in a play run.
type Tracker struct {
	puzzle   string
	runID    string
	recorder CompletionRecorder
	recorded bool
	warning  error
}

func New(puzzle string, recorder CompletionRecorder) *Tracker {
	return &Tracker{puzzle: puzzle, recorder: recorder}
}

// NewLinked records completion against one durable presentation-specific run.
func NewLinked(puzzle, runID string, recorder CompletionRecorder) *Tracker {
	return &Tracker{puzzle: puzzle, runID: runID, recorder: recorder}
}

// RunID returns the durable play-run identifier, when this tracker is linked.
func (tracker *Tracker) RunID() string {
	if tracker == nil {
		return ""
	}
	return tracker.runID
}

// Apply delegates to the game, then observes the accepted result.
func (tracker *Tracker) Apply(current *game.Game, action game.Action) (game.Result, error) {
	before := current.Snapshot().Status
	result, err := current.Apply(action)
	if err != nil {
		return result, err
	}
	tracker.Observe(before, result)
	return result, nil
}

// Observe records an already accepted transition. It is useful when a frontend
// must commit its own persistence before completion statistics.
func (tracker *Tracker) Observe(before game.Status, result game.Result) {
	if tracker == nil || tracker.recorded || before == game.StatusSolved || result.Action == game.ActionSolve || result.Status != game.StatusSolved {
		return
	}
	ok, recordErr := tracker.recorder.RecordCompletion(tracker.puzzle)
	if recordErr != nil {
		tracker.warning = fmt.Errorf("completion statistics were not saved: %w", recordErr)
		return
	}
	if !ok {
		tracker.warning = fmt.Errorf("completion statistics were not saved: puzzle is absent from the selected database")
		return
	}
	tracker.recorded = true
	tracker.warning = nil
}

// TakeWarning returns and clears the latest persistence warning.
func (tracker *Tracker) TakeWarning() error {
	if tracker == nil {
		return nil
	}
	err := tracker.warning
	tracker.warning = nil
	return err
}
