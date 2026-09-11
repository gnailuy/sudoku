package playrun

import (
	"errors"
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/solver"
)

type fakeRecorder struct {
	calls int
	found bool
	err   error
}

func (recorder *fakeRecorder) RecordCompletion(string) (bool, error) {
	recorder.calls++
	return recorder.found, recorder.err
}

func nearlySolvedGame(t *testing.T) game.Game {
	t.Helper()
	board := core.NewEmptyBoard()
	board.FromString(".23456789456789123789123456214365897365897214897214365531642978642978531978531642")
	return game.NewGame(board, game.NewDefaultOptions(solver.NewStore()))
}

func TestTrackerRecordsOnlyFirstPlayerCompletion(t *testing.T) {
	current := nearlySolvedGame(t)
	recorder := &fakeRecorder{found: true}
	tracker := New("normalized", recorder)
	if result, err := tracker.Apply(&current, game.SetValue{Position: core.NewPosition(0, 0), Value: 1}); err != nil || result.Status != game.StatusSolved {
		t.Fatalf("complete = %+v, %v", result, err)
	}
	if _, err := tracker.Apply(&current, game.Undo{}); err != nil {
		t.Fatal(err)
	}
	if _, err := tracker.Apply(&current, game.Redo{}); err != nil {
		t.Fatal(err)
	}
	if recorder.calls != 1 {
		t.Fatalf("completion calls = %d, want 1", recorder.calls)
	}
}

func TestTrackerCountsHintAssistedCompletion(t *testing.T) {
	current := nearlySolvedGame(t)
	recorder := &fakeRecorder{found: true}
	tracker := New("normalized", recorder)
	result, err := tracker.Apply(&current, game.ApplyHint{})
	if err != nil || result.Status != game.StatusSolved {
		t.Fatalf("hint completion = %+v, %v", result, err)
	}
	if recorder.calls != 1 {
		t.Fatalf("completion calls = %d, want 1", recorder.calls)
	}
}

func TestTrackerExcludesAutomaticSolve(t *testing.T) {
	current := nearlySolvedGame(t)
	recorder := &fakeRecorder{found: true}
	tracker := New("normalized", recorder)
	if _, err := tracker.Apply(&current, game.Solve{}); err != nil {
		t.Fatal(err)
	}
	if recorder.calls != 0 {
		t.Fatalf("completion calls = %d, want 0", recorder.calls)
	}
}

func TestTrackerWarnsAndRetriesCompletionPersistence(t *testing.T) {
	current := nearlySolvedGame(t)
	recorder := &fakeRecorder{found: true, err: errors.New("locked")}
	tracker := New("normalized", recorder)
	if _, err := tracker.Apply(&current, game.SetValue{Position: core.NewPosition(0, 0), Value: 1}); err != nil {
		t.Fatal(err)
	}
	if tracker.TakeWarning() == nil {
		t.Fatal("expected warning")
	}
	recorder.err = nil
	if _, err := tracker.Apply(&current, game.SetNotes{Position: core.NewPosition(0, 0), Values: []int{1}}); err == nil {
		t.Fatal("expected immutable-cell action to fail")
	}
	if _, err := tracker.Apply(&current, game.Reset{}); err != nil {
		t.Fatal(err)
	}
	if _, err := tracker.Apply(&current, game.SetValue{Position: core.NewPosition(0, 0), Value: 1}); err != nil {
		t.Fatal(err)
	}
	if recorder.calls != 2 || tracker.TakeWarning() != nil {
		t.Fatalf("calls=%d warning=%v", recorder.calls, tracker.TakeWarning())
	}
}
