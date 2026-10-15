package accountgame

import (
	"errors"
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/solver"
)

const testPuzzle = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."

func TestServiceAuthorizesOwnerAndPersistsRevisionedActions(t *testing.T) {
	service, database, options := newTestService(t)
	defer database.Close()

	state, err := service.Get("user-a", "game-a")
	if err != nil || state.Game.Revision != 0 || state.Snapshot.Values[0][0] != 0 {
		t.Fatalf("initial state = %+v, %v", state, err)
	}
	transition, err := service.Apply("user-a", "game-a", 0, game.SetValue{Position: core.NewPosition(0, 0), Value: 4})
	if err != nil {
		t.Fatal(err)
	}
	if transition.Game.Revision != 1 || transition.Snapshot.Values[0][0] != 4 || transition.Result.Action != game.ActionSetValue {
		t.Fatalf("transition = %+v", transition)
	}
	stored, err := database.AccountGameByID("user-a", "game-a")
	if err != nil || stored == nil || stored.Revision != 1 {
		t.Fatalf("stored game = %+v, %v", stored, err)
	}
	restored, err := game.Restore(stored.EngineState, options)
	if err != nil || restored.Snapshot().Values[0][0] != 4 {
		t.Fatalf("restored game = %+v, %v", restored.Snapshot(), err)
	}
	_, err = service.Apply("user-a", "game-a", 0, game.Undo{})
	var conflict *RevisionConflictError
	if !errors.As(err, &conflict) || conflict.CurrentRevision != 1 {
		t.Fatalf("stale action error = %v", err)
	}
}

func TestServicePersistsHistoricalTraversalAndTruncatesRedoBranch(t *testing.T) {
	service, database, _ := newTestService(t)
	defer database.Close()

	first := core.NewPosition(0, 0)
	second := core.NewPosition(0, 1)
	revision := int64(0)
	apply := func(action game.Action) State {
		t.Helper()
		transition, err := service.Apply("user-a", "game-a", revision, action)
		if err != nil {
			t.Fatalf("Apply(%T) at revision %d: %v", action, revision, err)
		}
		revision = transition.Game.Revision
		return transition.State
	}

	apply(game.SetNotes{Position: first, Values: []int{4}})
	apply(game.SetNotes{Position: second, Values: []int{5}})
	state := apply(game.Undo{})
	if !state.Snapshot.Notes[first.Row][first.Column].Has(4) || !state.Snapshot.Notes[second.Row][second.Column].IsEmpty() || !state.Snapshot.CanRedo {
		t.Fatalf("first persisted undo did not restore the exact previous state: %+v", state.Snapshot)
	}
	state = apply(game.Undo{})
	if !state.Snapshot.Notes[first.Row][first.Column].IsEmpty() || !state.Snapshot.CanRedo {
		t.Fatalf("second persisted undo did not restore the initial state: %+v", state.Snapshot)
	}
	state = apply(game.Redo{})
	if !state.Snapshot.Notes[first.Row][first.Column].Has(4) || !state.Snapshot.Notes[second.Row][second.Column].IsEmpty() || !state.Snapshot.CanRedo {
		t.Fatalf("persisted redo did not follow the original path: %+v", state.Snapshot)
	}
	state = apply(game.SetNotes{Position: first, Values: []int{4, 6}})
	if state.Snapshot.CanRedo || !state.Snapshot.Notes[first.Row][first.Column].Has(6) {
		t.Fatalf("manual action did not replace the historical branch: %+v", state.Snapshot)
	}
	if _, err := service.Apply("user-a", "game-a", revision, game.Redo{}); !errors.Is(err, &game.EngineError{Code: game.ErrorNoRedo}) {
		t.Fatalf("redo after a manual historical action = %v, want no-redo", err)
	}
	restored, err := service.Get("user-a", "game-a")
	if err != nil {
		t.Fatal(err)
	}
	if restored.Game.Revision != revision || restored.Snapshot != state.Snapshot {
		t.Fatalf("restored branched game differs from the persisted state: %+v", restored)
	}
}

func TestServicePreservesHintEliminationsAcrossPlayerPlacement(t *testing.T) {
	const puzzle = "................21..3..8.97..2..6.....7..4568.65.319....9..31...2..8..34.34.65..."
	store := solver.NewStore()
	service, database, options := newTestServiceWithPuzzle(t, puzzle, store.GetAllStrategySolverKeys())
	defer database.Close()

	board := core.NewEmptyBoard()
	board.FromString(puzzle)
	current := game.NewGame(board, options)
	for _, input := range []core.Cell{
		core.NewCell(core.NewPosition(3, 8), 3),
		core.NewCell(core.NewPosition(5, 8), 2),
		core.NewCell(core.NewPosition(8, 8), 9),
		core.NewCell(core.NewPosition(4, 0), 3),
		core.NewCell(core.NewPosition(4, 1), 1),
		core.NewCell(core.NewPosition(3, 7), 1),
		core.NewCell(core.NewPosition(7, 0), 5),
		core.NewCell(core.NewPosition(0, 5), 2),
		core.NewCell(core.NewPosition(2, 0), 2),
		core.NewCell(core.NewPosition(8, 6), 2),
	} {
		if _, err := current.Apply(game.SetValue{Position: input.Position, Value: input.Value}); err != nil {
			t.Fatal(err)
		}
	}
	for number := 0; number < 7; number++ {
		plan := current.Hint()
		if plan == nil || len(plan.Conclusion.Eliminations) == 0 {
			t.Fatalf("hint %d is not the reported elimination sequence: %+v", number+1, plan)
		}
		if _, err := current.Apply(game.ApplyHint{PlanID: plan.PlanID}); err != nil {
			t.Fatal(err)
		}
	}
	prepared, err := current.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	updated, err := database.UpdateAccountGame("user-a", "game-a", 0, prepared, string(current.Snapshot().Status))
	if err != nil || !updated {
		t.Fatalf("prepare reported account game = %v, %v", updated, err)
	}

	state, err := service.Get("user-a", "game-a")
	if err != nil {
		t.Fatal(err)
	}
	retained := core.NewPosition(3, 3)
	if state.Game.Revision != 1 || state.Snapshot.Candidates[retained.Row][retained.Column].Has(9) {
		t.Fatal("durable account game lost the reported elimination before placement")
	}
	placement := current.Hint()
	if placement == nil || placement.Conclusion.Placement == nil {
		t.Fatalf("expected the reported placement after eliminations: %+v", placement)
	}
	transition, err := service.Apply("user-a", "game-a", 1, game.SetValue{
		Position: placement.Conclusion.Placement.Position,
		Value:    placement.Conclusion.Placement.Value,
	})
	if err != nil {
		t.Fatal(err)
	}
	if transition.Game.Revision != 2 || transition.Snapshot.Candidates[retained.Row][retained.Column].Has(9) {
		t.Fatalf("account mutation changed retained elimination: %+v", transition.Snapshot.Candidates[retained.Row][retained.Column])
	}
	state, err = service.Get("user-a", "game-a")
	if err != nil {
		t.Fatal(err)
	}
	if state.Game.Revision != 2 || state.Snapshot.Candidates != transition.Snapshot.Candidates {
		t.Fatal("account restore changed the post-placement candidate grid")
	}
}

func TestServiceReportsCurrentRevisionWhenPersistenceLosesARace(t *testing.T) {
	_, database, options := newTestService(t)
	defer database.Close()

	service, err := New(Config{Store: &conflictingStore{DB: database}, Options: options})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Apply("user-a", "game-a", 0, game.SetValue{Position: core.NewPosition(0, 0), Value: 4})
	var conflict *RevisionConflictError
	if !errors.As(err, &conflict) || conflict.CurrentRevision != 1 {
		t.Fatalf("competing update error = %v", err)
	}
}

type conflictingStore struct{ *db.DB }

func (store *conflictingStore) UpdateAccountGame(userID, accountGameID string, expectedRevision int64, engineState []byte, status string) (bool, error) {
	current, err := store.AccountGameByID(userID, accountGameID)
	if err != nil || current == nil {
		return false, err
	}
	if _, err := store.DB.UpdateAccountGame(userID, accountGameID, expectedRevision, current.EngineState, status); err != nil {
		return false, err
	}
	return false, nil
}

func TestServiceHidesOtherOwnersGamesAcrossOperations(t *testing.T) {
	service, database, _ := newTestService(t)
	defer database.Close()

	if _, err := service.Get("user-b", "game-a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-owner get error = %v", err)
	}
	if _, err := service.Apply("user-b", "game-a", 0, game.Undo{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-owner apply error = %v", err)
	}
	if deleted, err := service.Delete("user-b", "game-a"); !errors.Is(err, ErrNotFound) || deleted {
		t.Fatalf("cross-owner delete = %v, %v", deleted, err)
	}
	games, err := service.List("user-b", 20)
	if err != nil || len(games) != 0 {
		t.Fatalf("cross-owner list = %+v, %v", games, err)
	}
	ownerGames, err := service.List("user-a", 20)
	if err != nil || len(ownerGames) != 1 || ownerGames[0].ID != "game-a" || len(ownerGames[0].EngineState) != 0 {
		t.Fatalf("owner list = %+v, %v", ownerGames, err)
	}
	if deleted, err := service.Delete("user-a", "game-a"); err != nil || !deleted {
		t.Fatalf("owner delete = %v, %v", deleted, err)
	}
	if _, err := service.Get("user-a", "game-a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted game error = %v", err)
	}
}

func TestServiceRejectsMissingAuthenticatedIdentity(t *testing.T) {
	service, database, _ := newTestService(t)
	defer database.Close()

	if _, err := service.List("", 20); !errors.Is(err, ErrAuthenticatedIdentityRequired) {
		t.Fatalf("list identity error = %v", err)
	}
	if _, err := service.Get("", "game-a"); !errors.Is(err, ErrAuthenticatedIdentityRequired) {
		t.Fatalf("get identity error = %v", err)
	}
	if _, err := service.Apply("", "game-a", 0, game.Undo{}); !errors.Is(err, ErrAuthenticatedIdentityRequired) {
		t.Fatalf("apply identity error = %v", err)
	}
	if _, err := service.Delete("", "game-a"); !errors.Is(err, ErrAuthenticatedIdentityRequired) {
		t.Fatalf("delete identity error = %v", err)
	}
}

func newTestService(t *testing.T) (*Service, *db.DB, game.Options) {
	t.Helper()
	return newTestServiceWithPuzzle(t, testPuzzle, nil)
}

func newTestServiceWithPuzzle(t *testing.T, puzzle string, strategyKeys []string) (*Service, *db.DB, game.Options) {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	for _, userID := range []string{"user-a", "user-b"} {
		if err := database.CreateUser(db.User{ID: userID}); err != nil {
			database.Close()
			t.Fatal(err)
		}
	}
	board := core.NewEmptyBoard()
	board.FromString(puzzle)
	options := game.NewDefaultOptions(solver.NewStore())
	if strategyKeys != nil {
		options.StrategySolverKeys = strategyKeys
	}
	current := game.NewGame(board, options)
	serialized, err := current.Serialize()
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.InsertPuzzle(db.Puzzle{Puzzle: puzzle, Difficulty: "easy", Score: 1, MaxTechnique: "single"}); err != nil {
		database.Close()
		t.Fatal(err)
	}
	basePuzzleID := db.BasePuzzleID(puzzle)
	if err := database.InsertPlayRun(db.PlayRun{ID: "run-a", BasePuzzleID: basePuzzleID, PresentedPuzzle: puzzle}); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if err := database.CreateAccountGame(db.AccountGame{
		ID: "game-a", UserID: "user-a", BasePuzzleID: basePuzzleID, PlayRunID: "run-a",
		EngineState: serialized, ActualDifficulty: "easy", Status: "in-progress",
	}); err != nil {
		database.Close()
		t.Fatal(err)
	}
	service, err := New(Config{Store: database, Options: options})
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	return service, database, options
}
