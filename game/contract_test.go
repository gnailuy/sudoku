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

func TestSnapshotCandidatesFollowSolverSafeBoard(t *testing.T) {
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
		t.Fatal("snapshot should expose candidates only for empty cells")
	}
	playBoard := game.PlayBoard()
	if got, want := initial.Candidates[target.Row][target.Column], playBoard.Candidates(target); got != want {
		t.Fatalf("target candidates=%v, want %v", got.Values(), want.Values())
	}
	if !peer.IsValid() {
		t.Fatal("test board needs a row peer that initially allows 4")
	}

	if _, err := game.Apply(SetNotes{Position: target, Values: []int{9}}); err != nil {
		t.Fatal(err)
	}
	if got := game.Snapshot().Candidates; got != initial.Candidates {
		t.Fatal("manual notes changed derived candidates")
	}

	if _, err := game.Apply(SetValue{Position: target, Value: 9}); err != nil {
		t.Fatal(err)
	}
	invalid := game.Snapshot()
	if !invalid.Invalid[0][2] || invalid.Values[0][2] != 9 {
		t.Fatal("test value should be visible and invalid")
	}
	if invalid.Candidates != initial.Candidates {
		t.Fatal("an invalid visible entry constrained solver-safe candidates")
	}

	if _, err := game.Apply(SetValue{Position: target, Value: 4}); err != nil {
		t.Fatal(err)
	}
	accepted := game.Snapshot()
	if accepted.Candidates[0][2] != 0 || accepted.Candidates[peer.Row][peer.Column].Has(4) {
		t.Fatal("accepted value did not refresh target and peer candidates")
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

func TestApplySetNotesIsAtomicAndUndoable(t *testing.T) {
	game := newTestGame()
	position := core.NewPosition(0, 2)

	result, err := game.Apply(SetNotes{Position: position, Values: []int{1, 3, 9}})
	if err != nil {
		t.Fatalf("Apply(SetNotes) returned error: %v", err)
	}
	if result.Action != ActionSetNotes || len(result.Changes) != 1 {
		t.Fatalf("unexpected set-notes result: %+v", result)
	}
	if got, want := result.Changes[0].NotesAfter.Values(), []int{1, 3, 9}; !slices.Equal(got, want) {
		t.Fatalf("notes=%v, want %v", got, want)
	}
	if _, err := game.Apply(Undo{}); err != nil {
		t.Fatal(err)
	}
	if !game.Snapshot().Notes[0][2].IsEmpty() {
		t.Fatal("undo did not restore notes")
	}

	before := game.Snapshot()
	if _, err := game.Apply(SetNotes{Position: position, Values: []int{2, 2}}); !errors.Is(err, &EngineError{Code: ErrorInvalidCell}) {
		t.Fatalf("duplicate notes returned %v", err)
	}
	if after := game.Snapshot(); after != before {
		t.Fatal("invalid set-notes action mutated the game")
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

	result, err := game.Apply(ApplyHint{})
	if err != nil {
		t.Fatalf("Apply(ApplyHint) returned error: %v", err)
	}
	if result.Action != ActionApplyHint || len(result.Changes) != 1 || !result.CanUndo || result.Hint == nil {
		t.Fatalf("unexpected hint result: %+v", result)
	}
	if result.Hint.Position != result.Changes[0].Position || result.Hint.Value != result.Changes[0].After || result.Hint.Reason == "" {
		t.Fatalf("hint metadata does not describe the applied change: %+v", result)
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
	if len(result.Changes) != 1 {
		t.Fatalf("expected one cell change, got %+v", result.Changes)
	}
	change := result.Changes[0]
	if change.Position != position || change.Before != before || change.After != after {
		t.Fatalf("unexpected cell change: %+v", change)
	}
}

func TestNoteActionsAndUnifiedHistory(t *testing.T) {
	game := newTestGame()
	position := core.NewPosition(0, 2)

	result, err := game.Apply(SetNotes{Position: position, Values: []int{4}})
	if err != nil {
		t.Fatalf("Apply(SetNotes) returned error: %v", err)
	}
	if result.Action != ActionSetNotes || !result.Changes[0].NotesAfter.Has(4) {
		t.Fatalf("unexpected toggle result: %+v", result)
	}
	if !game.Snapshot().Notes[0][2].Has(4) {
		t.Fatal("note was not stored in snapshot")
	}

	if _, err = game.Apply(SetValue{Position: position, Value: 4}); err != nil {
		t.Fatalf("Apply(SetValue) returned error: %v", err)
	}
	if !game.Snapshot().Notes[0][2].IsEmpty() {
		t.Fatal("setting a value should clear notes in that cell")
	}

	if _, err = game.Apply(Undo{}); err != nil {
		t.Fatalf("undo value returned error: %v", err)
	}
	snapshot := game.Snapshot()
	if snapshot.Values[0][2] != 0 || !snapshot.Notes[0][2].Has(4) {
		t.Fatalf("undo did not restore value and notes atomically: %+v", snapshot)
	}

	if _, err = game.Apply(Undo{}); err != nil {
		t.Fatalf("undo note returned error: %v", err)
	}
	if !game.Snapshot().Notes[0][2].IsEmpty() {
		t.Fatal("second undo should remove the note")
	}

	if _, err = game.Apply(Redo{}); err != nil {
		t.Fatalf("redo note returned error: %v", err)
	}
	if !game.Snapshot().Notes[0][2].Has(4) {
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
		if _, err := game.Apply(SetNotes{Position: position, Values: []int{4}}); err != nil {
			t.Fatalf("add note at %v: %v", position, err)
		}
	}

	if _, err := game.Apply(SetValue{Position: target, Value: 4}); err != nil {
		t.Fatalf("set value: %v", err)
	}
	snapshot := game.Snapshot()
	if !snapshot.Notes[target.Row][target.Column].IsEmpty() {
		t.Fatal("target notes were not cleared")
	}
	for _, position := range peers {
		if snapshot.Notes[position.Row][position.Column].Has(4) {
			t.Fatalf("peer note was not removed at %v", position)
		}
	}
	if !snapshot.Notes[nonPeer.Row][nonPeer.Column].Has(4) {
		t.Fatal("non-peer note should be preserved")
	}

	if _, err := game.Apply(Undo{}); err != nil {
		t.Fatalf("undo set value: %v", err)
	}
	snapshot = game.Snapshot()
	for _, position := range append(peers, target, nonPeer) {
		if !snapshot.Notes[position.Row][position.Column].Has(4) {
			t.Fatalf("undo did not restore note at %v", position)
		}
	}
}

func TestSetNotesAndRedoTruncation(t *testing.T) {
	game := newTestGame()
	position := core.NewPosition(0, 2)
	if _, err := game.Apply(SetNotes{Position: position, Values: []int{3}}); err != nil {
		t.Fatal(err)
	}
	if _, err := game.Apply(SetNotes{Position: position, Values: []int{3, 4}}); err != nil {
		t.Fatal(err)
	}
	if _, err := game.Apply(SetNotes{Position: position}); err != nil {
		t.Fatal(err)
	}
	if !game.Snapshot().Notes[0][2].IsEmpty() {
		t.Fatal("clear notes did not clear all notes")
	}
	if _, err := game.Apply(Undo{}); err != nil {
		t.Fatal(err)
	}
	if _, err := game.Apply(SetNotes{Position: position, Values: []int{3, 4, 5}}); err != nil {
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
		{SetNotes{Position: core.NewPosition(0, 0), Values: []int{4}}, ErrorImmutableCell},
		{SetNotes{Position: filled, Values: []int{4}}, ErrorNoteNotAllowed},
		{SetNotes{Position: core.NewPosition(0, 3), Values: []int{0}}, ErrorInvalidCell},
		{SetNotes{Position: core.Position{Row: 9, Column: 0}}, ErrorInvalidCell},
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
