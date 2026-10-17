package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

// StateErrorCode identifies a serialized-state validation failure.
type StateErrorCode string

const (
	StateErrorMalformed      StateErrorCode = "malformed-state"
	StateErrorInvalidPuzzle  StateErrorCode = "invalid-puzzle"
	StateErrorInvalidSession StateErrorCode = "invalid-session"
	StateErrorInvalidHistory StateErrorCode = "invalid-history"
)

// StateError reports why serialized game state could not be restored.
type StateError struct {
	Code   StateErrorCode
	Detail string
	Err    error
}

func (err *StateError) Error() string {
	if err.Detail != "" {
		return err.Detail
	}
	return string(err.Code)
}

// Unwrap exposes the underlying JSON error, when present.
func (err *StateError) Unwrap() error { return err.Err }

// Is allows errors.Is to match state errors by code.
func (err *StateError) Is(target error) bool {
	other, ok := target.(*StateError)
	return ok && err.Code == other.Code
}

type serializedGame struct {
	Puzzle   string                    `json:"puzzle"`
	Mistakes int                       `json:"mistakes"`
	Current  serializedSessionState    `json:"current"`
	History  []serializedHistoryRecord `json:"history"`
	Cursor   int                       `json:"cursor"`
}

type serializedSessionState struct {
	Values     string                `json:"values"`
	Invalid    string                `json:"invalid"`
	Notes      []serializedCandidate `json:"notes"`
	Candidates []serializedCandidate `json:"candidates"`
}

type serializedCandidate struct {
	Row    int   `json:"row"`
	Column int   `json:"column"`
	Values []int `json:"values"`
}

type serializedHistoryRecord struct {
	Before serializedSessionState `json:"before"`
	After  serializedSessionState `json:"after"`
}

// Serialize returns the complete game session in the current JSON shape.
// Solver configuration is intentionally excluded and must be supplied when
// restoring the session.
func (game *Game) Serialize() ([]byte, error) {
	payload := serializedGame{
		Puzzle:   game.problemBoard.ToString(),
		Mistakes: game.mistakes,
		Current:  serializeSessionState(game.problemBoard, game.captureState()),
		History:  make([]serializedHistoryRecord, len(game.inputSequence)),
		Cursor:   game.inputCursor,
	}
	for index, record := range game.inputSequence {
		payload.History[index] = serializedHistoryRecord{
			Before: serializeSessionState(game.problemBoard, record.before),
			After:  serializeSessionState(game.problemBoard, record.after),
		}
	}
	return json.Marshal(payload)
}

// Restore validates a serialized session and constructs a game atomically.
// The supplied options provide host-owned solver configuration.
func Restore(data []byte, options Options) (Game, error) {
	var payload serializedGame
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return Game{}, stateError(StateErrorMalformed, "cannot decode serialized game state", err)
	}
	if err := requireJSONEnd(decoder); err != nil {
		return Game{}, stateError(StateErrorMalformed, "serialized game state has trailing data", err)
	}
	if payload.Mistakes < 0 {
		return Game{}, stateError(StateErrorInvalidSession, "serialized mistake count is invalid", nil)
	}

	problem, err := boardFromString(payload.Puzzle)
	if err != nil || !problem.IsValid() {
		return Game{}, stateError(StateErrorInvalidPuzzle, "serialized puzzle is invalid", err)
	}
	completeSolver := options.solverStore.GetDefaultSolver()
	problemCopy := problem.Copy()
	if completeSolver.CountSolutions(&problemCopy) == 0 {
		return Game{}, stateError(StateErrorInvalidPuzzle, "serialized puzzle has no solution", nil)
	}

	current, err := deserializeSessionState(problem, payload.Current, completeSolver)
	if err != nil {
		return Game{}, err
	}
	history := make([]historyRecord, len(payload.History))
	for index, record := range payload.History {
		before, beforeErr := deserializeSessionState(problem, record.Before, completeSolver)
		if beforeErr != nil {
			return Game{}, stateError(StateErrorInvalidHistory, fmt.Sprintf("history record %d has invalid before state", index), beforeErr)
		}
		after, afterErr := deserializeSessionState(problem, record.After, completeSolver)
		if afterErr != nil {
			return Game{}, stateError(StateErrorInvalidHistory, fmt.Sprintf("history record %d has invalid after state", index), afterErr)
		}
		history[index] = historyRecord{before: before, after: after}
	}
	if err := validateHistory(history, payload.Cursor); err != nil {
		return Game{}, err
	}

	game := NewGame(problem, options)
	game.mistakes = payload.Mistakes
	game.restoreState(current)
	game.inputSequence = history
	game.inputCursor = payload.Cursor
	return game, nil
}

func serializeSessionState(problem core.Board, state sessionState) serializedSessionState {
	values := core.NewEmptyBoard()
	invalid := core.NewEmptyBoard()
	notes := make([]serializedCandidate, 0)
	candidates := make([]serializedCandidate, 0)
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			position := core.NewPosition(row, column)
			if problem.Get(position) == 0 {
				if value := state.playBoard.Get(position); value != 0 {
					_ = values.Set(position, value)
				}
				if value := state.invalidInput.Get(position); value != 0 {
					_ = invalid.Set(position, value)
				}
			}
			if !state.notes[row][column].IsEmpty() {
				notes = append(notes, serializedCandidate{
					Row: row, Column: column, Values: state.notes[row][column].Values(),
				})
			}
			if !state.candidates[row][column].IsEmpty() {
				candidates = append(candidates, serializedCandidate{
					Row: row, Column: column, Values: state.candidates[row][column].Values(),
				})
			}
		}
	}
	return serializedSessionState{
		Values: values.ToString(), Invalid: invalid.ToString(), Notes: notes, Candidates: candidates,
	}
}

func deserializeSessionState(problem core.Board, payload serializedSessionState, completeSolver solver.CompleteSolver) (sessionState, error) {
	values, err := boardFromString(payload.Values)
	if err != nil {
		return sessionState{}, stateError(StateErrorInvalidSession, "session values are invalid", err)
	}
	invalid, err := boardFromString(payload.Invalid)
	if err != nil {
		return sessionState{}, stateError(StateErrorInvalidSession, "session invalid entries are invalid", err)
	}

	playBoard := problem.Copy()
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			position := core.NewPosition(row, column)
			value := values.Get(position)
			invalidValue := invalid.Get(position)
			if problem.Get(position) != 0 && (value != 0 || invalidValue != 0) {
				return sessionState{}, stateError(StateErrorInvalidSession, fmt.Sprintf("session changes puzzle cell at row %d, column %d", row, column), nil)
			}
			if value != 0 && invalidValue != 0 {
				return sessionState{}, stateError(StateErrorInvalidSession, fmt.Sprintf("session overlaps valid and invalid values at row %d, column %d", row, column), nil)
			}
			if value != 0 {
				_ = playBoard.Set(position, value)
			}
		}
	}
	if !playBoard.IsValid() {
		return sessionState{}, stateError(StateErrorInvalidSession, "session values violate Sudoku constraints", nil)
	}

	notes, err := deserializeDigitSets(problem, values, invalid, payload.Notes, "notes")
	if err != nil {
		return sessionState{}, err
	}
	candidates, err := deserializeDigitSets(problem, values, invalid, payload.Candidates, "candidates")
	if err != nil {
		return sessionState{}, err
	}

	state := sessionState{playBoard: playBoard, invalidInput: invalid, notes: notes, candidates: candidates}
	if err := validateSessionSemantics(state, completeSolver); err != nil {
		return sessionState{}, err
	}
	return state, nil
}

func deserializeDigitSets(problem, values, invalid core.Board, records []serializedCandidate, label string) ([9][9]core.CandidateSet, error) {
	var sets [9][9]core.CandidateSet
	seen := make(map[core.Position]struct{}, len(records))
	for _, candidate := range records {
		position := core.Position{Row: candidate.Row, Column: candidate.Column}
		if !position.IsValid() {
			return sets, stateError(StateErrorInvalidSession, "session "+label+" position is invalid", nil)
		}
		if _, exists := seen[position]; exists {
			return sets, stateError(StateErrorInvalidSession, fmt.Sprintf("session has duplicate %s at row %d, column %d", label, candidate.Row, candidate.Column), nil)
		}
		seen[position] = struct{}{}
		if len(candidate.Values) == 0 {
			return sets, stateError(StateErrorInvalidSession, "session contains an empty "+label+" record", nil)
		}
		if problem.Get(position) != 0 || values.Get(position) != 0 || invalid.Get(position) != 0 {
			return sets, stateError(StateErrorInvalidSession, fmt.Sprintf("session %s occupy a filled cell at row %d, column %d", label, candidate.Row, candidate.Column), nil)
		}
		for _, value := range candidate.Values {
			if value < 1 || value > 9 || sets[candidate.Row][candidate.Column].Has(value) {
				return sets, stateError(StateErrorInvalidSession, "session "+label+" values are invalid", nil)
			}
			sets[candidate.Row][candidate.Column].Add(value)
		}
	}
	return sets, nil
}

func validateSessionSemantics(state sessionState, completeSolver solver.CompleteSolver) error {
	playBoard := state.playBoard.Copy()
	if completeSolver.CountSolutions(&playBoard) == 0 {
		return stateError(StateErrorInvalidSession, "session values have no solution", nil)
	}
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			position := core.NewPosition(row, column)
			value := state.invalidInput.Get(position)
			if value == 0 {
				continue
			}
			withInvalid := state.playBoard.Copy()
			_ = withInvalid.Set(position, value)
			if withInvalid.IsValid() && completeSolver.CountSolutions(&withInvalid) != 0 {
				return stateError(StateErrorInvalidSession, fmt.Sprintf("invalid entry at row %d, column %d is solvable", row, column), nil)
			}
		}
	}
	return nil
}

func validateHistory(history []historyRecord, cursor int) error {
	if len(history) == 0 {
		if cursor != -1 {
			return stateError(StateErrorInvalidHistory, "empty history requires cursor -1", nil)
		}
		return nil
	}
	if cursor < -1 || cursor >= len(history) {
		return stateError(StateErrorInvalidHistory, "history cursor is out of range", nil)
	}
	for index := 1; index < len(history); index++ {
		if history[index-1].after != history[index].before {
			return stateError(StateErrorInvalidHistory, fmt.Sprintf("history records %d and %d are disconnected", index-1, index), nil)
		}
	}
	return nil
}

func boardFromString(value string) (core.Board, error) {
	if !core.IsValidSudokuString(value) {
		return core.Board{}, errors.New("expected an 81-character Sudoku string")
	}
	board := core.NewEmptyBoard()
	board.FromString(value)
	return board, nil
}

func requireJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func stateError(code StateErrorCode, detail string, err error) *StateError {
	return &StateError{Code: code, Detail: detail, Err: err}
}
