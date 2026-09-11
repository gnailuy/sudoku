package game

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

func TestSerializeRestoreRoundTripPreservesStateAndHistory(t *testing.T) {
	original := newTestGame()
	target := core.NewPosition(0, 2)
	peer := core.NewPosition(0, 3)

	actions := []Action{
		SetNotes{Position: target, Values: []int{4}},
		SetNotes{Position: peer, Values: []int{4}},
		SetValue{Position: target, Value: 4},
		SetValue{Position: peer, Value: 5}, // Conflicts with the row and remains visible as invalid input.
		Undo{},
	}
	for _, action := range actions {
		if _, err := original.Apply(action); err != nil {
			t.Fatalf("Apply(%T) returned error: %v", action, err)
		}
	}

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize returned error: %v", err)
	}
	restored, err := Restore(data, NewDefaultOptions(solver.NewStore()))
	if err != nil {
		t.Fatalf("Restore returned error: %v", err)
	}
	if restored.Snapshot() != original.Snapshot() {
		t.Fatalf("restored snapshot differs\noriginal: %+v\nrestored: %+v", original.Snapshot(), restored.Snapshot())
	}

	restoredData, err := restored.Serialize()
	if err != nil {
		t.Fatalf("restored Serialize returned error: %v", err)
	}
	if string(restoredData) != string(data) {
		t.Fatalf("serialized state is not stable\nfirst: %s\nagain: %s", data, restoredData)
	}

	if _, err := restored.Apply(Redo{}); err != nil {
		t.Fatalf("redo after restoration returned error: %v", err)
	}
	snapshot := restored.Snapshot()
	if snapshot.Values[peer.Row][peer.Column] != 5 || !snapshot.Invalid[peer.Row][peer.Column] || snapshot.Status != StatusInvalid {
		t.Fatalf("redo did not restore invalid input: %+v", snapshot)
	}
	if _, err := restored.Apply(Undo{}); err != nil {
		t.Fatalf("undo invalid input returned error: %v", err)
	}
	if _, err := restored.Apply(Undo{}); err != nil {
		t.Fatalf("undo value returned error: %v", err)
	}
	snapshot = restored.Snapshot()
	if snapshot.Values[target.Row][target.Column] != 0 || !snapshot.Notes[target.Row][target.Column].Has(4) || !snapshot.Notes[peer.Row][peer.Column].Has(4) {
		t.Fatalf("undo did not restore pre-cleanup notes: %+v", snapshot)
	}
}

func TestRestoreRejectsMalformedAndUnsupportedState(t *testing.T) {
	options := NewDefaultOptions(solver.NewStore())

	tests := []struct {
		name string
		data []byte
		code StateErrorCode
	}{
		{name: "malformed JSON", data: []byte(`{"version":`), code: StateErrorMalformed},
		{name: "trailing JSON", data: append(mustSerialize(t, newTestGame()), []byte(` {}`)...), code: StateErrorMalformed},
		{name: "unknown field", data: []byte(`{"version":1,"puzzle":"","current":{},"history":[],"cursor":-1,"extra":true}`), code: StateErrorMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Restore(test.data, options); !errors.Is(err, &StateError{Code: test.code}) {
				t.Fatalf("expected %s, got %v", test.code, err)
			}
		})
	}

	payload := decodeSerializedGame(t, mustSerialize(t, newTestGame()))
	payload.Version = StateVersion + 1
	if _, err := Restore(encodeSerializedGame(t, payload), options); !errors.Is(err, &StateError{Code: StateErrorUnsupportedVersion}) {
		t.Fatalf("expected unsupported version error, got %v", err)
	}
}

func TestRestoreRejectsInvalidPuzzleAndSession(t *testing.T) {
	options := NewDefaultOptions(solver.NewStore())
	base := decodeSerializedGame(t, mustSerialize(t, newTestGame()))

	t.Run("invalid puzzle", func(t *testing.T) {
		payload := base
		payload.Puzzle = "55" + payload.Puzzle[2:]
		if _, err := Restore(encodeSerializedGame(t, payload), options); !errors.Is(err, &StateError{Code: StateErrorInvalidPuzzle}) {
			t.Fatalf("expected invalid puzzle error, got %v", err)
		}
	})

	t.Run("value overwrites given", func(t *testing.T) {
		payload := base
		payload.Current.Values = "1" + payload.Current.Values[1:]
		if _, err := Restore(encodeSerializedGame(t, payload), options); !errors.Is(err, &StateError{Code: StateErrorInvalidSession}) {
			t.Fatalf("expected invalid session error, got %v", err)
		}
	})

	t.Run("notes occupy given", func(t *testing.T) {
		payload := base
		payload.Current.Notes = []serializedNote{{Row: 0, Column: 0, Values: []int{1}}}
		if _, err := Restore(encodeSerializedGame(t, payload), options); !errors.Is(err, &StateError{Code: StateErrorInvalidSession}) {
			t.Fatalf("expected invalid session error, got %v", err)
		}
	})

	t.Run("duplicate note values", func(t *testing.T) {
		payload := base
		payload.Current.Notes = []serializedNote{{Row: 0, Column: 2, Values: []int{4, 4}}}
		if _, err := Restore(encodeSerializedGame(t, payload), options); !errors.Is(err, &StateError{Code: StateErrorInvalidSession}) {
			t.Fatalf("expected invalid session error, got %v", err)
		}
	})

	t.Run("accepted value makes puzzle unsolvable", func(t *testing.T) {
		payload := base
		payload.Current.Values = payload.Current.Values[:2] + "1" + payload.Current.Values[3:]
		if _, err := Restore(encodeSerializedGame(t, payload), options); !errors.Is(err, &StateError{Code: StateErrorInvalidSession}) {
			t.Fatalf("expected invalid session error, got %v", err)
		}
	})

	t.Run("solvable value marked invalid", func(t *testing.T) {
		payload := base
		payload.Current.Invalid = payload.Current.Invalid[:2] + "4" + payload.Current.Invalid[3:]
		if _, err := Restore(encodeSerializedGame(t, payload), options); !errors.Is(err, &StateError{Code: StateErrorInvalidSession}) {
			t.Fatalf("expected invalid session error, got %v", err)
		}
	})
}

func TestRestoreRejectsInvalidHistory(t *testing.T) {
	game := newTestGame()
	if _, err := game.Apply(SetNotes{Position: core.NewPosition(0, 2), Values: []int{4}}); err != nil {
		t.Fatal(err)
	}
	if _, err := game.Apply(SetNotes{Position: core.NewPosition(0, 3), Values: []int{4}}); err != nil {
		t.Fatal(err)
	}
	base := decodeSerializedGame(t, mustSerialize(t, game))
	options := NewDefaultOptions(solver.NewStore())

	t.Run("cursor out of range", func(t *testing.T) {
		payload := base
		payload.Cursor = len(payload.History)
		if _, err := Restore(encodeSerializedGame(t, payload), options); !errors.Is(err, &StateError{Code: StateErrorInvalidHistory}) {
			t.Fatalf("expected invalid history error, got %v", err)
		}
	})

	t.Run("disconnected records", func(t *testing.T) {
		payload := base
		payload.History = append([]serializedHistoryRecord(nil), base.History...)
		payload.History[1].Before = payload.History[0].Before
		if _, err := Restore(encodeSerializedGame(t, payload), options); !errors.Is(err, &StateError{Code: StateErrorInvalidHistory}) {
			t.Fatalf("expected invalid history error, got %v", err)
		}
	})

}

func TestSerializeRestorePreservesStateOutsideActionHistory(t *testing.T) {
	original := newTestGame()
	if _, err := original.Apply(SetNotes{Position: core.NewPosition(0, 2), Values: []int{4}}); err != nil {
		t.Fatal(err)
	}
	if _, err := original.Apply(SetValue{Position: core.NewPosition(0, 3), Value: 5}); err != nil {
		t.Fatal(err)
	}
	// Solve intentionally replaces the visible board without adding a history record.
	if _, err := original.Apply(Solve{}); err != nil {
		t.Fatal(err)
	}
	if snapshot := original.Snapshot(); snapshot.Status != StatusSolved || snapshot.Invalid[0][3] || !snapshot.Notes[0][2].IsEmpty() {
		t.Fatalf("Solve did not produce a self-consistent solved session: %+v", snapshot)
	}

	restored, err := Restore(mustSerialize(t, original), NewDefaultOptions(solver.NewStore()))
	if err != nil {
		t.Fatalf("Restore returned error: %v", err)
	}
	if restored.Snapshot() != original.Snapshot() {
		t.Fatal("restoration did not preserve the current state independently of history")
	}
	if _, err := restored.Apply(Undo{}); err != nil {
		t.Fatalf("restored undo returned error: %v", err)
	}
	if _, err := original.Apply(Undo{}); err != nil {
		t.Fatalf("original undo returned error: %v", err)
	}
	if restored.Snapshot() != original.Snapshot() {
		t.Fatal("restoration did not preserve solve-action undo behavior")
	}
}

func mustSerialize(t *testing.T, game Game) []byte {
	t.Helper()
	data, err := game.Serialize()
	if err != nil {
		t.Fatalf("Serialize returned error: %v", err)
	}
	return data
}

func decodeSerializedGame(t *testing.T, data []byte) serializedGame {
	t.Helper()
	var payload serializedGame
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode serialized game: %v", err)
	}
	return payload
}

func encodeSerializedGame(t *testing.T, payload serializedGame) []byte {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode serialized game: %v", err)
	}
	return data
}
