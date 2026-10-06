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
	board.FromString(testPuzzle)
	options := game.NewDefaultOptions(solver.NewStore())
	current := game.NewGame(board, options)
	serialized, err := current.Serialize()
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.InsertPuzzle(db.Puzzle{Puzzle: testPuzzle, Difficulty: "easy", Score: 1, MaxTechnique: "single"}); err != nil {
		database.Close()
		t.Fatal(err)
	}
	basePuzzleID := db.BasePuzzleID(testPuzzle)
	if err := database.InsertPlayRun(db.PlayRun{ID: "run-a", BasePuzzleID: basePuzzleID, PresentedPuzzle: testPuzzle}); err != nil {
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
