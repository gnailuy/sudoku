package game

import (
	"errors"
	"slices"
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

func TestSnapshotStringMatchesSessionSummary(t *testing.T) {
	game := newTestGame()
	if got, want := game.Snapshot().String(), game.ToString(); got != want {
		t.Fatalf("initial snapshot summary differs from game summary:\n%s\nwant:\n%s", got, want)
	}

	if _, err := game.Apply(SetValue{Position: core.NewPosition(0, 2), Value: 9}); err != nil {
		t.Fatal(err)
	}
	if got, want := game.Snapshot().String(), game.ToString(); got != want {
		t.Fatalf("invalid snapshot summary differs from game summary:\n%s\nwant:\n%s", got, want)
	}

	if _, err := game.Apply(Solve{}); err != nil {
		t.Fatal(err)
	}
	if got, want := game.Snapshot().String(), game.ToString(); got != want {
		t.Fatalf("solved snapshot summary differs from game summary:\n%s\nwant:\n%s", got, want)
	}
}

func TestSnapshotIsDetached(t *testing.T) {
	game := newTestGame()
	position := core.NewPosition(0, 2)

	snapshot := game.Snapshot()
	snapshot.Givens[0][0] = 0
	snapshot.Values[0][2] = 9
	snapshot.Invalid[0][2] = true
	snapshot.Candidates[0][2].Add(9)

	fresh := game.Snapshot()
	if fresh.Givens[0][0] != 5 {
		t.Fatal("mutating snapshot givens changed the game")
	}
	if fresh.Values[0][2] != 0 || fresh.Invalid[0][2] {
		t.Fatal("mutating snapshot state changed the game")
	}
	if fresh.Candidates[0][2].Has(9) {
		t.Fatal("mutating snapshot candidates changed the game")
	}

	playBoard := game.PlayBoard()
	_ = playBoard.Set(position, 9)
	if game.Get(position) != 0 {
		t.Fatal("mutating PlayBoard copy changed the game")
	}
}

func TestSnapshotCarriesOneAuthoritativeCandidateGrid(t *testing.T) {
	game := newTestGame()
	target := core.NewPosition(0, 2)
	initial := game.Snapshot()
	peer := core.Position{Row: -1, Column: -1}
	for column := 0; column < 9; column++ {
		if column != target.Column && initial.Candidates[target.Row][column].Has(4) {
			peer = core.NewPosition(target.Row, column)
			break
		}
	}
	if initial.Candidates[0][0] != 0 || initial.Candidates[target.Row][target.Column].IsEmpty() {
		t.Fatal("new game should initialize candidates only for empty cells")
	}
	if !peer.IsValid() {
		t.Fatal("test board needs a row peer that initially allows 4")
	}

	if _, err := game.Apply(SetCandidates{Position: target, Values: []int{9}}); err != nil {
		t.Fatal(err)
	}
	if got := game.Snapshot().Candidates[target.Row][target.Column].Values(); !slices.Equal(got, []int{9}) {
		t.Fatalf("candidate edit was not authoritative: %v", got)
	}

	beforeInvalid := game.Snapshot()
	if _, err := game.Apply(SetValue{Position: target, Value: 9}); err != nil {
		t.Fatal(err)
	}
	invalid := game.Snapshot()
	if !invalid.Invalid[0][2] || invalid.Values[0][2] != 9 || invalid.Candidates[target.Row][target.Column] != 0 {
		t.Fatal("wrong value should occupy only its own visible cell")
	}
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			if row == target.Row && column == target.Column {
				continue
			}
			if invalid.Candidates[row][column] != beforeInvalid.Candidates[row][column] {
				t.Fatal("wrong value changed peer candidates")
			}
		}
	}

	if _, err := game.Apply(SetValue{Position: target, Value: 4}); err != nil {
		t.Fatal(err)
	}
	accepted := game.Snapshot()
	if accepted.Candidates[0][2] != 0 || accepted.Candidates[peer.Row][peer.Column].Has(4) {
		t.Fatal("accepted value did not clear target and peer candidates")
	}
	if _, err := game.Apply(Undo{}); err != nil {
		t.Fatal(err)
	}
	if got := game.Snapshot().Candidates; got != invalid.Candidates {
		t.Fatal("undo did not restore candidates")
	}
	if _, err := game.Apply(Redo{}); err != nil {
		t.Fatal(err)
	}
	if got := game.Snapshot().Candidates; got != accepted.Candidates {
		t.Fatal("redo did not restore candidates")
	}
}

func TestRestoredSnapshotRecomputesCandidates(t *testing.T) {
	original := newTestGame()
	if _, err := original.Apply(SetValue{Position: core.NewPosition(0, 2), Value: 4}); err != nil {
		t.Fatal(err)
	}
	data, err := original.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Restore(data, NewDefaultOptions(solver.NewStore()))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := restored.Snapshot().Candidates, original.Snapshot().Candidates; got != want {
		t.Fatal("restored candidates differ from original")
	}
}

func TestApplyValueActionsAndHistory(t *testing.T) {
	game := newTestGame()
	position := core.NewPosition(0, 2)

	result, err := game.Apply(SetValue{Position: position, Value: 4})
	if err != nil {
		t.Fatalf("Apply(SetValue) returned error: %v", err)
	}
	assertSingleChange(t, result, position, 0, 4)
	if result.Action != ActionSetValue || !result.CanUndo || result.CanRedo {
		t.Fatalf("unexpected set result: %+v", result)
	}

	result, err = game.Apply(Undo{})
	if err != nil {
		t.Fatalf("Apply(Undo) returned error: %v", err)
	}
	assertSingleChange(t, result, position, 4, 0)
	if !result.CanRedo {
		t.Fatal("undo result should allow redo")
	}

	result, err = game.Apply(Redo{})
	if err != nil {
		t.Fatalf("Apply(Redo) returned error: %v", err)
	}
	assertSingleChange(t, result, position, 0, 4)

	result, err = game.Apply(ClearValue{Position: position})
	if err != nil {
		t.Fatalf("Apply(ClearValue) returned error: %v", err)
	}
	assertSingleChange(t, result, position, 4, 0)
}

func TestApplySetCandidatesIsAtomicAndUndoable(t *testing.T) {
	game := newTestGame()
	position := core.NewPosition(0, 2)

	beforeInitial := game.Snapshot()
	result, err := game.Apply(SetCandidates{Position: position, Values: []int{1, 3, 9}})
	if err != nil {
		t.Fatalf("Apply(SetCandidates) returned error: %v", err)
	}
	if result.Action != ActionSetCandidates || len(result.Changes) != 1 {
		t.Fatalf("unexpected set-candidates result: %+v", result)
	}
	if got, want := result.Changes[0].CandidatesAfter.Values(), []int{1, 3, 9}; !slices.Equal(got, want) {
		t.Fatalf("notes=%v, want %v", got, want)
	}
	if _, err := game.Apply(Undo{}); err != nil {
		t.Fatal(err)
	}
	if game.Snapshot().Candidates != beforeInitial.Candidates {
		t.Fatal("undo did not restore candidates")
	}

	before := game.Snapshot()
	if _, err := game.Apply(SetCandidates{Position: position, Values: []int{2, 2}}); !errors.Is(err, &EngineError{Code: ErrorInvalidCell}) {
		t.Fatalf("duplicate notes returned %v", err)
	}
	if after := game.Snapshot(); after != before {
		t.Fatal("invalid set-candidates action mutated the game")
	}
}

func TestApplyReturnsTypedErrorsWithoutMutation(t *testing.T) {
	game := newTestGame()
	before := game.Snapshot()

	tests := []struct {
		name   string
		action Action
		code   ErrorCode
	}{
		{name: "invalid value", action: SetValue{Position: core.NewPosition(0, 2), Value: 10}, code: ErrorInvalidCell},
		{name: "given", action: SetValue{Position: core.NewPosition(0, 0), Value: 1}, code: ErrorImmutableCell},
		{name: "empty undo", action: Undo{}, code: ErrorNoUndo},
		{name: "empty redo", action: Redo{}, code: ErrorNoRedo},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := game.Apply(test.action)
			if !errors.Is(err, &EngineError{Code: test.code}) {
				t.Fatalf("expected error code %q, got %v", test.code, err)
			}
			if after := game.Snapshot(); after != before {
				t.Fatalf("failed action mutated game\nbefore: %+v\nafter: %+v", before, after)
			}
		})
	}
}

func TestApplyHintAndReset(t *testing.T) {
	game := newTestGame()

	plan := game.Hint()
	if plan == nil {
		t.Fatal("Hint returned nil")
	}
	result, err := game.Apply(ApplyHint{PlanID: plan.PlanID})
	if err != nil {
		t.Fatalf("Apply(ApplyHint) returned error: %v", err)
	}
	if result.Action != ActionApplyHint || len(result.Changes) == 0 || !result.CanUndo || result.Hint == nil {
		t.Fatalf("unexpected hint result: %+v", result)
	}
	if result.Hint.PlanID != plan.PlanID || result.Hint.Summary == "" || result.Hint.Conclusion.Placement == nil {
		t.Fatalf("hint metadata does not describe the applied plan: %+v", result)
	}
	if result.Hint.Conclusion.Placement.Position != result.Changes[0].Position || result.Hint.Conclusion.Placement.Value != result.Changes[0].After {
		t.Fatalf("hint conclusion does not match the applied change: %+v", result)
	}

	result, err = game.Apply(Reset{})
	if err != nil {
		t.Fatalf("Apply(Reset) returned error: %v", err)
	}
	if result.Action != ActionReset || result.CanUndo || result.CanRedo {
		t.Fatalf("unexpected reset result: %+v", result)
	}
	if game.Snapshot().Status != StatusInProgress {
		t.Fatal("reset game should be in progress")
	}
}

func TestMistakeCountTracksConfirmedInvalidValuesOutsideHistory(t *testing.T) {
	game := newTestGame()
	position := core.NewPosition(0, 2)

	if snapshot := game.Snapshot(); snapshot.Mistakes != 0 {
		t.Fatalf("new game mistakes = %d, want 0", snapshot.Mistakes)
	}
	if _, err := game.Apply(SetValue{Position: position, Value: 9}); err != nil {
		t.Fatal(err)
	}
	if snapshot := game.Snapshot(); snapshot.Mistakes != 1 || !snapshot.Invalid[0][2] {
		t.Fatalf("invalid value snapshot = %+v", snapshot)
	}
	for _, action := range []Action{Undo{}, Redo{}, ClearValue{Position: position}, SetCandidates{Position: position, Values: []int{4}}} {
		if _, err := game.Apply(action); err != nil {
			t.Fatalf("Apply(%T) returned error: %v", action, err)
		}
	}
	plan := game.Hint()
	if plan == nil {
		t.Fatal("Hint returned nil")
	}
	if _, err := game.Apply(ApplyHint{PlanID: plan.PlanID}); err != nil {
		t.Fatalf("Apply(ApplyHint) returned error: %v", err)
	}
	if snapshot := game.Snapshot(); snapshot.Mistakes != 1 {
		t.Fatalf("non-mistake actions changed count to %d", snapshot.Mistakes)
	}
	if _, err := game.Apply(Reset{}); err != nil {
		t.Fatal(err)
	}
	if snapshot := game.Snapshot(); snapshot.Mistakes != 1 {
		t.Fatalf("reset of the same puzzle changed count to %d", snapshot.Mistakes)
	}
	newGame := newTestGame()
	if snapshot := newGame.Snapshot(); snapshot.Mistakes != 0 {
		t.Fatalf("new puzzle mistakes = %d, want 0", snapshot.Mistakes)
	}
}

func TestRestartBeginsCleanAttemptOnSamePuzzle(t *testing.T) {
	game := newTestGame()
	if _, err := game.Apply(SetValue{Position: core.NewPosition(0, 2), Value: 9}); err != nil {
		t.Fatal(err)
	}
	if _, err := game.Apply(SetCandidates{Position: core.NewPosition(0, 3), Values: []int{2, 4}}); err != nil {
		t.Fatal(err)
	}

	result, err := game.Apply(Restart{})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := game.Snapshot()
	if result.Action != ActionRestart || snapshot.Status != StatusInProgress || snapshot.Mistakes != 0 || snapshot.CanUndo || snapshot.CanRedo {
		t.Fatalf("restart result = %+v, snapshot = %+v", result, snapshot)
	}
	if snapshot.Values != snapshot.Givens || snapshot.Invalid != [9][9]bool{} || snapshot.Candidates != legalCandidateGrid(&game.playBoard) {
		t.Fatal("restart did not restore the board and initial legal candidates")
	}
}

func TestApplyRepairAndSolve(t *testing.T) {
	game := newTestGame()
	position := core.NewPosition(0, 2)

	if _, err := game.Apply(SetValue{Position: position, Value: 9}); err != nil {
		t.Fatalf("Apply invalid SetValue returned error: %v", err)
	}
	if game.Snapshot().Status != StatusInvalid {
		t.Fatal("wrong player value should make the snapshot invalid")
	}

	result, err := game.Apply(Repair{})
	if err != nil {
		t.Fatalf("Apply(Repair) returned error: %v", err)
	}
	if result.Action != ActionRepair || result.Status != StatusInProgress || len(result.Changes) != 1 {
		t.Fatalf("unexpected repair result: %+v", result)
	}

	result, err = game.Apply(Solve{})
	if err != nil {
		t.Fatalf("Apply(Solve) returned error: %v", err)
	}
	if result.Action != ActionSolve || result.Status != StatusSolved || len(result.Changes) == 0 {
		t.Fatalf("unexpected solve result: %+v", result)
	}
}

func TestApplyNilAction(t *testing.T) {
	game := newTestGame()
	before := game.Snapshot()
	_, err := game.Apply(nil)
	if !errors.Is(err, &EngineError{Code: ErrorInvalidAction}) {
		t.Fatalf("expected invalid action error, got %v", err)
	}
	if game.Snapshot() != before {
		t.Fatal("nil action mutated game")
	}
}

func assertSingleChange(t *testing.T, result Result, position core.Position, before, after int) {
	t.Helper()
	for _, change := range result.Changes {
		if change.Position == position {
			if change.Before != before || change.After != after {
				t.Fatalf("unexpected target change: %+v", change)
			}
			return
		}
	}
	t.Fatalf("missing target change at %+v: %+v", position, result.Changes)
}

func TestCandidateActionsAndUnifiedHistory(t *testing.T) {
	game := newTestGame()
	position := core.NewPosition(0, 2)
	initial := game.Snapshot()

	result, err := game.Apply(SetCandidates{Position: position, Values: []int{4}})
	if err != nil {
		t.Fatalf("Apply(SetCandidates) returned error: %v", err)
	}
	if result.Action != ActionSetCandidates || !result.Changes[0].CandidatesAfter.Has(4) {
		t.Fatalf("unexpected toggle result: %+v", result)
	}
	if !game.Snapshot().Candidates[0][2].Has(4) {
		t.Fatal("note was not stored in snapshot")
	}

	if _, err = game.Apply(SetValue{Position: position, Value: 4}); err != nil {
		t.Fatalf("Apply(SetValue) returned error: %v", err)
	}
	if !game.Snapshot().Candidates[0][2].IsEmpty() {
		t.Fatal("setting a value should clear notes in that cell")
	}

	if _, err = game.Apply(Undo{}); err != nil {
		t.Fatalf("undo value returned error: %v", err)
	}
	snapshot := game.Snapshot()
	if snapshot.Values[0][2] != 0 || !snapshot.Candidates[0][2].Has(4) {
		t.Fatalf("undo did not restore value and notes atomically: %+v", snapshot)
	}

	if _, err = game.Apply(Undo{}); err != nil {
		t.Fatalf("undo note returned error: %v", err)
	}
	if game.Snapshot().Candidates != initial.Candidates {
		t.Fatal("second undo should restore the initial candidates")
	}

	if _, err = game.Apply(Redo{}); err != nil {
		t.Fatalf("redo note returned error: %v", err)
	}
	if !game.Snapshot().Candidates[0][2].Has(4) {
		t.Fatal("redo should restore the note")
	}
}

func TestSetValueCleansPeerNotesAtomically(t *testing.T) {
	game := newTestGame()
	target := core.NewPosition(0, 2)
	peers := []core.Position{
		core.NewPosition(0, 3), // row
		core.NewPosition(1, 2), // column and box
	}
	nonPeer := core.NewPosition(4, 1)

	for _, position := range append(peers, target, nonPeer) {
		if _, err := game.Apply(SetCandidates{Position: position, Values: []int{4}}); err != nil {
			t.Fatalf("add note at %v: %v", position, err)
		}
	}

	if _, err := game.Apply(SetValue{Position: target, Value: 4}); err != nil {
		t.Fatalf("set value: %v", err)
	}
	snapshot := game.Snapshot()
	if !snapshot.Candidates[target.Row][target.Column].IsEmpty() {
		t.Fatal("target notes were not cleared")
	}
	for _, position := range peers {
		if snapshot.Candidates[position.Row][position.Column].Has(4) {
			t.Fatalf("peer note was not removed at %v", position)
		}
	}
	if !snapshot.Candidates[nonPeer.Row][nonPeer.Column].Has(4) {
		t.Fatal("non-peer note should be preserved")
	}

	if _, err := game.Apply(Undo{}); err != nil {
		t.Fatalf("undo set value: %v", err)
	}
	snapshot = game.Snapshot()
	for _, position := range append(peers, target, nonPeer) {
		if !snapshot.Candidates[position.Row][position.Column].Has(4) {
			t.Fatalf("undo did not restore note at %v", position)
		}
	}
}

func TestInvalidValuePreservesPeerNotesAndCandidates(t *testing.T) {
	game := newTestGame()
	target := core.NewPosition(0, 2)
	peer := core.NewPosition(0, 3)

	for _, position := range []core.Position{target, peer} {
		if _, err := game.Apply(SetCandidates{Position: position, Values: []int{9}}); err != nil {
			t.Fatalf("add note at %v: %v", position, err)
		}
	}
	beforeCandidates := game.Snapshot().Candidates

	if _, err := game.Apply(SetValue{Position: target, Value: 9}); err != nil {
		t.Fatalf("set invalid value: %v", err)
	}
	snapshot := game.Snapshot()
	if !snapshot.Invalid[target.Row][target.Column] {
		t.Fatal("test value should be visible and invalid")
	}
	if !snapshot.Candidates[target.Row][target.Column].IsEmpty() {
		t.Fatal("the occupied target should clear its own notes")
	}
	if !snapshot.Candidates[peer.Row][peer.Column].Has(9) {
		t.Fatal("an invalid value should not remove a peer note")
	}
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			if row == target.Row && column == target.Column {
				continue
			}
			if snapshot.Candidates[row][column] != beforeCandidates[row][column] {
				t.Fatal("an invalid value changed peer candidates")
			}
		}
	}

	if _, err := game.Apply(Undo{}); err != nil {
		t.Fatalf("undo invalid value: %v", err)
	}
	snapshot = game.Snapshot()
	if !snapshot.Candidates[target.Row][target.Column].Has(9) || !snapshot.Candidates[peer.Row][peer.Column].Has(9) {
		t.Fatal("undo should restore the exact note state")
	}
}

func TestSetCandidatesAndRedoTruncation(t *testing.T) {
	game := newTestGame()
	position := core.NewPosition(0, 2)
	if _, err := game.Apply(SetCandidates{Position: position, Values: []int{3}}); err != nil {
		t.Fatal(err)
	}
	if _, err := game.Apply(SetCandidates{Position: position, Values: []int{3, 4}}); err != nil {
		t.Fatal(err)
	}
	if _, err := game.Apply(SetCandidates{Position: position}); err != nil {
		t.Fatal(err)
	}
	if !game.Snapshot().Candidates[0][2].IsEmpty() {
		t.Fatal("clear notes did not clear all notes")
	}
	if _, err := game.Apply(Undo{}); err != nil {
		t.Fatal(err)
	}
	if _, err := game.Apply(SetCandidates{Position: position, Values: []int{3, 4, 5}}); err != nil {
		t.Fatal(err)
	}
	if _, err := game.Apply(Redo{}); !errors.Is(err, &EngineError{Code: ErrorNoRedo}) {
		t.Fatalf("new note action should truncate redo history, got %v", err)
	}
}

func TestNoteErrorsDoNotMutateGame(t *testing.T) {
	game := newTestGame()
	filled := core.NewPosition(0, 2)
	if _, err := game.Apply(SetValue{Position: filled, Value: 4}); err != nil {
		t.Fatal(err)
	}
	before := game.Snapshot()

	tests := []struct {
		action Action
		code   ErrorCode
	}{
		{SetCandidates{Position: core.NewPosition(0, 0), Values: []int{4}}, ErrorImmutableCell},
		{SetCandidates{Position: filled, Values: []int{4}}, ErrorNoteNotAllowed},
		{SetCandidates{Position: core.NewPosition(0, 3), Values: []int{0}}, ErrorInvalidCell},
		{SetCandidates{Position: core.Position{Row: 9, Column: 0}}, ErrorInvalidCell},
	}
	for _, test := range tests {
		if _, err := game.Apply(test.action); !errors.Is(err, &EngineError{Code: test.code}) {
			t.Fatalf("expected %s, got %v", test.code, err)
		}
		if after := game.Snapshot(); after != before {
			t.Fatalf("failed note action mutated game\nbefore: %+v\nafter: %+v", before, after)
		}
	}
}
