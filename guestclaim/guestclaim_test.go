package guestclaim

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/guestdoc"
	"github.com/gnailuy/sudoku/solver"
)

const testPuzzle = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."

func TestClaimIsIdempotentAndOwnerScoped(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	for _, userID := range []string{"user-a", "user-b"} {
		if err := database.CreateUser(db.User{ID: userID}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := database.InsertPuzzle(db.Puzzle{Puzzle: testPuzzle, Difficulty: "easy", Score: 1, MaxTechnique: "single"}); err != nil {
		t.Fatal(err)
	}
	sealer, err := guestdoc.New(guestdoc.Config{
		ActiveKeyID: "current", Keys: map[string][]byte{"current": bytes.Repeat([]byte{1}, 32)},
		Lifetime: 24 * time.Hour, Now: func() time.Time { return time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	board := core.NewEmptyBoard()
	board.FromString(testPuzzle)
	current := game.NewGame(board, game.NewDefaultOptions(solver.NewStore()))
	token, document, err := sealer.Create(&current, db.BasePuzzleID(testPuzzle), "easy")
	if err != nil {
		t.Fatal(err)
	}
	service, err := New(Config{Documents: sealer, Store: database, Random: bytes.NewReader(bytes.Repeat([]byte{2}, 128))})
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Claim("user-a", token)
	if err != nil || !first.Created || first.Game.Revision != document.Revision || string(first.Game.EngineState) != string(document.EngineState) {
		t.Fatalf("first claim = %+v, %v", first, err)
	}
	second, err := service.Claim("user-a", token)
	if err != nil || second.Created || second.Game.ID != first.Game.ID || second.Game.PlayRunID != first.Game.PlayRunID {
		t.Fatalf("idempotent claim = %+v, %v", second, err)
	}
	if _, err := service.Claim("user-b", token); !errors.Is(err, db.ErrGuestAlreadyClaimed) {
		t.Fatalf("cross-user claim error = %v", err)
	}
	run, err := database.PlayRunByID(first.Game.PlayRunID)
	if err != nil || run == nil || run.PresentedPuzzle != testPuzzle || run.Status != "active" {
		t.Fatalf("claimed play run = %+v, %v", run, err)
	}
	games, err := database.AccountGamesByUser("user-a", 10)
	if err != nil || len(games) != 1 {
		t.Fatalf("claimed games = %+v, %v", games, err)
	}
}

func TestClaimRejectsInvalidDocumentBeforePersistence(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	sealer, err := guestdoc.New(guestdoc.Config{
		ActiveKeyID: "current", Keys: map[string][]byte{"current": bytes.Repeat([]byte{1}, 32)}, Lifetime: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := New(Config{Documents: sealer, Store: database})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Claim("user-a", "tampered"); !errors.Is(err, &guestdoc.Error{Code: guestdoc.ErrorMalformed}) {
		t.Fatalf("invalid document error = %v", err)
	}
}
