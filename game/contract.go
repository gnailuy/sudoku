package game

import (
	"fmt"
	"strings"

	"github.com/gnailuy/sudoku/core"
)

// Status describes the current state of a game session.
type Status string

const (
	StatusInProgress Status = "in-progress"
	StatusInvalid    Status = "invalid"
	StatusSolved     Status = "solved"
)

// ErrorCode identifies an expected engine failure without requiring callers
// to parse user-facing text.
type ErrorCode string

const (
	ErrorInvalidAction  ErrorCode = "invalid-action"
	ErrorInvalidCell    ErrorCode = "invalid-cell"
	ErrorImmutableCell  ErrorCode = "immutable-cell"
	ErrorNoteNotAllowed ErrorCode = "note-not-allowed"
	ErrorNoUndo         ErrorCode = "no-undo"
	ErrorNoRedo         ErrorCode = "no-redo"
	ErrorNoHint         ErrorCode = "no-hint"
)

// EngineError is returned for invalid player actions. The game is unchanged
// when Apply returns an EngineError.
type EngineError struct {
	Code     ErrorCode
	Position *core.Position
	Detail   string
}

func (err *EngineError) Error() string {
	if err.Detail != "" {
		return err.Detail
	}
	return string(err.Code)
}

// Is allows errors.Is to match engine errors by code.
func (err *EngineError) Is(target error) bool {
	other, ok := target.(*EngineError)
	return ok && err.Code == other.Code
}

// ActionKind identifies a transition accepted by the engine.
type ActionKind string

const (
	ActionSetValue   ActionKind = "set-value"
	ActionClearValue ActionKind = "clear-value"
	ActionReset      ActionKind = "reset"
	ActionUndo       ActionKind = "undo"
	ActionRedo       ActionKind = "redo"
	ActionApplyHint  ActionKind = "apply-hint"
	ActionToggleNote ActionKind = "toggle-note"
	ActionSetNotes   ActionKind = "set-notes"
	ActionClearNotes ActionKind = "clear-notes"
	ActionRepair     ActionKind = "repair"
	ActionSolve      ActionKind = "solve"
)

// Action is a typed player intent. Its unexported method keeps the set of
// accepted actions closed to this package.
type Action interface {
	actionKind() ActionKind
}

// SetValue sets an editable cell to a digit from 1 through 9.
type SetValue struct {
	Position core.Position
	Value    int
}

func (SetValue) actionKind() ActionKind { return ActionSetValue }

// ClearValue clears an editable cell.
type ClearValue struct{ Position core.Position }

func (ClearValue) actionKind() ActionKind { return ActionClearValue }

// Reset restores the puzzle and clears action history.
type Reset struct{}

func (Reset) actionKind() ActionKind { return ActionReset }

// Undo reverses the most recent recorded action.
type Undo struct{}

func (Undo) actionKind() ActionKind { return ActionUndo }

// Redo reapplies the next action in history.
type Redo struct{}

func (Redo) actionKind() ActionKind { return ActionRedo }

// ApplyHint applies the current hint as a recorded value action.
type ApplyHint struct{}

func (ApplyHint) actionKind() ActionKind { return ActionApplyHint }

// ToggleNote adds or removes one manual note on an editable empty cell.
type ToggleNote struct {
	Position core.Position
	Value    int
}

func (ToggleNote) actionKind() ActionKind { return ActionToggleNote }

// SetNotes atomically replaces all manual notes on an editable empty cell.
type SetNotes struct {
	Position core.Position
	Values   []int
}

func (SetNotes) actionKind() ActionKind { return ActionSetNotes }

// ClearNotes removes all manual notes from an editable empty cell.
type ClearNotes struct{ Position core.Position }

func (ClearNotes) actionKind() ActionKind { return ActionClearNotes }

// Repair removes invalid player entries by returning to the most recent
// valid history state.
type Repair struct{}

func (Repair) actionKind() ActionKind { return ActionRepair }

// Solve completes the current session with the configured complete solver.
type Solve struct{}

func (Solve) actionKind() ActionKind { return ActionSolve }

// Snapshot is a detached read model. Its arrays may be modified by a caller
// without mutating the Game that produced it.
type Snapshot struct {
	Givens     [9][9]int
	Values     [9][9]int
	Invalid    [9][9]bool
	Notes      [9][9]core.CandidateSet
	Candidates [9][9]core.CandidateSet
	Status     Status
	CanUndo    bool
	CanRedo    bool
}

// String formats the detached session state for terminal summaries.
func (snapshot Snapshot) String() string {
	var result strings.Builder
	result.WriteString("Problem:\n")
	writeSnapshotValues(&result, snapshot.Givens)
	result.WriteByte('\n')

	if snapshot.Values != snapshot.Givens {
		status := "Valid"
		switch snapshot.Status {
		case StatusInvalid:
			status = "Invalid"
		case StatusSolved:
			status = "Solved"
		}
		fmt.Fprintf(&result, "Current board (%s):\n", status)
		writeSnapshotValues(&result, snapshot.Values)
		result.WriteByte('\n')
	}

	return result.String()
}

func writeSnapshotValues(result *strings.Builder, values [9][9]int) {
	for row := range values {
		for column := range values[row] {
			value := values[row][column]
			if value == 0 {
				result.WriteByte('.')
			} else {
				result.WriteByte(byte('0' + value))
			}
		}
	}
}

// CellChange describes one visible cell changed by an accepted action.
type CellChange struct {
	Position      core.Position
	Before        int
	After         int
	InvalidBefore bool
	InvalidAfter  bool
	NotesBefore   core.CandidateSet
	NotesAfter    core.CandidateSet
}

// AppliedHint describes the recommendation used by an ApplyHint action.
type AppliedHint struct {
	Position  core.Position
	Value     int
	Technique string
	Reason    string
}

// Result describes an accepted transition.
type Result struct {
	Action  ActionKind
	Changes []CellChange
	Status  Status
	CanUndo bool
	CanRedo bool
	Hint    *AppliedHint
}

// Snapshot returns a detached representation suitable for rendering.
func (game *Game) Snapshot() Snapshot {
	var snapshot Snapshot
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			position := core.NewPosition(row, column)
			snapshot.Givens[row][column] = game.problemBoard.Get(position)
			snapshot.Values[row][column] = game.Get(position)
			snapshot.Invalid[row][column] = game.invalidInput.Get(position) != 0
			snapshot.Notes[row][column] = game.notes[row][column]
			snapshot.Candidates[row][column] = game.playBoard.Candidates(position)
		}
	}
	snapshot.Status = game.status()
	snapshot.CanUndo = game.inputCursor >= 0
	snapshot.CanRedo = game.inputCursor < len(game.inputSequence)-1
	return snapshot
}

// ProblemBoard returns a detached copy of the immutable puzzle givens.
func (game *Game) ProblemBoard() core.Board { return game.problemBoard.Copy() }

// PlayBoard returns a detached copy of the solver-safe current board. Invalid
// player entries are represented separately in Snapshot.
func (game *Game) PlayBoard() core.Board { return game.playBoard.Copy() }

// Apply validates and applies one typed player action atomically.
func (game *Game) Apply(action Action) (Result, error) {
	before := game.Snapshot()
	if action == nil {
		return Result{}, &EngineError{Code: ErrorInvalidAction, Detail: "action is required"}
	}

	kind := action.actionKind()
	var err error
	var appliedHint *AppliedHint
	switch typed := action.(type) {
	case SetValue:
		if !typed.Position.IsValid() || typed.Value < 1 || typed.Value > 9 {
			return Result{}, invalidCellError(typed.Position, typed.Value)
		}
		err = game.addInputAndRecordHistory(core.Cell{Position: typed.Position, Value: typed.Value})
	case ClearValue:
		if !typed.Position.IsValid() {
			return Result{}, invalidCellError(typed.Position, 0)
		}
		err = game.addInputAndRecordHistory(core.Cell{Position: typed.Position, Value: 0})
	case Reset:
		game.reset()
	case Undo:
		err = game.undo()
	case Redo:
		err = game.redo()
	case ApplyHint:
		hint := game.Hint()
		if hint == nil {
			err = &EngineError{Code: ErrorNoHint, Detail: "no hint is available"}
		} else {
			err = game.addInputAndRecordHistory(hint.Cell)
			appliedHint = &AppliedHint{
				Position:  hint.Cell.Position,
				Value:     hint.Cell.Value,
				Technique: hint.Technique,
				Reason:    hint.Reason,
			}
		}
	case ToggleNote:
		err = game.toggleNote(typed.Position, typed.Value)
	case SetNotes:
		err = game.setNotes(typed.Position, typed.Values)
	case ClearNotes:
		err = game.clearNotes(typed.Position)
	case Repair:
		if game.repair() == 0 {
			err = &EngineError{Code: ErrorInvalidAction, Detail: "no invalid input to repair"}
		}
	case Solve:
		game.solve()
	default:
		return Result{}, &EngineError{Code: ErrorInvalidAction, Detail: fmt.Sprintf("unsupported action %T", action)}
	}
	if err != nil {
		return Result{}, err
	}

	after := game.Snapshot()
	result := resultFromSnapshots(kind, before, after)
	result.Hint = appliedHint
	return result, nil
}

func (game *Game) status() Status {
	if game.IsSolved() {
		return StatusSolved
	}
	if !game.IsValid() {
		return StatusInvalid
	}
	return StatusInProgress
}

func resultFromSnapshots(kind ActionKind, before, after Snapshot) Result {
	result := Result{
		Action:  kind,
		Status:  after.Status,
		CanUndo: after.CanUndo,
		CanRedo: after.CanRedo,
		Changes: make([]CellChange, 0),
	}
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			if before.Values[row][column] == after.Values[row][column] &&
				before.Invalid[row][column] == after.Invalid[row][column] &&
				before.Notes[row][column] == after.Notes[row][column] {
				continue
			}
			result.Changes = append(result.Changes, CellChange{
				Position:      core.NewPosition(row, column),
				Before:        before.Values[row][column],
				After:         after.Values[row][column],
				InvalidBefore: before.Invalid[row][column],
				InvalidAfter:  after.Invalid[row][column],
				NotesBefore:   before.Notes[row][column],
				NotesAfter:    after.Notes[row][column],
			})
		}
	}
	return result
}

func (game *Game) toggleNote(position core.Position, value int) error {
	if !position.IsValid() || value < 1 || value > 9 {
		return invalidCellError(position, value)
	}
	if err := game.validateNoteCell(position); err != nil {
		return err
	}
	before := game.captureState()
	if game.notes[position.Row][position.Column].Has(value) {
		game.notes[position.Row][position.Column].Remove(value)
	} else {
		game.notes[position.Row][position.Column].Add(value)
	}
	game.recordTransition(before)
	return nil
}

func (game *Game) setNotes(position core.Position, values []int) error {
	if !position.IsValid() {
		return invalidCellError(position, 0)
	}
	if err := game.validateNoteCell(position); err != nil {
		return err
	}
	var notes core.CandidateSet
	for _, value := range values {
		if value < 1 || value > 9 || notes.Has(value) {
			return invalidCellError(position, value)
		}
		notes.Add(value)
	}
	before := game.captureState()
	game.notes[position.Row][position.Column] = notes
	game.recordTransition(before)
	return nil
}

func (game *Game) clearNotes(position core.Position) error {
	if !position.IsValid() {
		return invalidCellError(position, 0)
	}
	if err := game.validateNoteCell(position); err != nil {
		return err
	}
	before := game.captureState()
	game.notes[position.Row][position.Column] = 0
	game.recordTransition(before)
	return nil
}

func (game *Game) validateNoteCell(position core.Position) error {
	positionCopy := position
	if game.problemBoard.Get(position) != 0 {
		return &EngineError{Code: ErrorImmutableCell, Position: &positionCopy, Detail: "cannot add notes to a problem cell"}
	}
	if game.Get(position) != 0 {
		return &EngineError{Code: ErrorNoteNotAllowed, Position: &positionCopy, Detail: "notes are allowed only on empty cells"}
	}
	return nil
}

func invalidCellError(position core.Position, value int) *EngineError {
	positionCopy := position
	return &EngineError{
		Code:     ErrorInvalidCell,
		Position: &positionCopy,
		Detail:   fmt.Sprintf("invalid cell at row %d, column %d with value %d", position.Row, position.Column, value),
	}
}
