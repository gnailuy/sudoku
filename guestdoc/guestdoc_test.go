package guestdoc

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/solver"
)

const testPuzzle = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."

func TestCreateOpenAndApplyGuestDocument(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	sealer := newTestSealer(t, now, map[string][]byte{"current": testKey(1)}, "current")
	current, options := testGame(t)
	basePuzzleID := db.BasePuzzleID(testPuzzle)

	token, created, err := sealer.Create(&current, basePuzzleID, "easy")
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || created.Revision != 0 || created.ExpiresAt.Sub(created.IssuedAt) != 24*time.Hour {
		t.Fatalf("created document = %+v", created)
	}
	opened, err := sealer.Open(token)
	if err != nil || opened.ID != created.ID || opened.BasePuzzleID != basePuzzleID || opened.PresentedPuzzle != testPuzzle {
		t.Fatalf("opened document = %+v, %v", opened, err)
	}

	transition, err := sealer.Apply(token, 0, game.SetValue{Position: core.NewPosition(0, 0), Value: 4}, options)
	if err != nil {
		t.Fatal(err)
	}
	if transition.Document.Revision != 1 || transition.Token == token || transition.Snapshot.Values[0][0] != 4 || transition.Result.Action != game.ActionSetValue {
		t.Fatalf("transition = %+v", transition)
	}
	reopened, err := sealer.Open(transition.Token)
	if err != nil || reopened.Revision != 1 || reopened.IssuedAt != created.IssuedAt || reopened.ExpiresAt != created.ExpiresAt {
		t.Fatalf("replacement document = %+v, %v", reopened, err)
	}
	restored, err := game.Restore(reopened.EngineState, options)
	if err != nil || restored.Snapshot().Values[0][0] != 4 {
		t.Fatalf("restored guest game = %+v, %v", restored.Snapshot(), err)
	}
	if _, err := sealer.Apply(transition.Token, 0, game.Undo{}, options); !errors.Is(err, &Error{Code: ErrorRevisionConflict}) {
		t.Fatalf("stale revision error = %v", err)
	}
}

func TestGuestDocumentRejectsTamperingExpiryAndUnknownKeys(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	sealer := newTestSealer(t, now, map[string][]byte{"current": testKey(1)}, "current")
	current, _ := testGame(t)
	token, _, err := sealer.Create(&current, db.BasePuzzleID(testPuzzle), "easy")
	if err != nil {
		t.Fatal(err)
	}

	tampered := token[:len(token)-1] + differentLastCharacter(token[len(token)-1:])
	if _, err := sealer.Open(tampered); !errors.Is(err, &Error{Code: ErrorAuthentication}) && !errors.Is(err, &Error{Code: ErrorMalformed}) {
		t.Fatalf("tampered document error = %v", err)
	}

	rotated := newTestSealer(t, now, map[string][]byte{"next": testKey(2)}, "next")
	if _, err := rotated.Open(token); !errors.Is(err, &Error{Code: ErrorUnknownKey}) {
		t.Fatalf("unknown key error = %v", err)
	}

	sealer.now = func() time.Time { return now.Add(24 * time.Hour) }
	if _, err := sealer.Open(token); !errors.Is(err, &Error{Code: ErrorExpired}) {
		t.Fatalf("expired document error = %v", err)
	}
}

func TestGuestKeyRotationDecryptsOldAndResealsWithActiveKey(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	old := newTestSealer(t, now, map[string][]byte{"old": testKey(1)}, "old")
	current, options := testGame(t)
	token, _, err := old.Create(&current, db.BasePuzzleID(testPuzzle), "easy")
	if err != nil {
		t.Fatal(err)
	}
	rotated := newTestSealer(t, now, map[string][]byte{"old": testKey(1), "new": testKey(2)}, "new")
	transition, err := rotated.Apply(token, 0, game.SetNotes{Position: core.NewPosition(0, 0), Values: []int{4, 5}}, options)
	if err != nil {
		t.Fatal(err)
	}
	wrapped := decodeEnvelope(t, transition.Token)
	if wrapped.KeyID != "new" {
		t.Fatalf("replacement key ID = %q", wrapped.KeyID)
	}
	if _, err := rotated.Open(transition.Token); err != nil {
		t.Fatal(err)
	}
}

func TestGuestDocumentRejectsPayloadPuzzleMismatch(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	sealer := newTestSealer(t, now, map[string][]byte{"current": testKey(1)}, "current")
	current, _ := testGame(t)
	_, document, err := sealer.Create(&current, db.BasePuzzleID(testPuzzle), "easy")
	if err != nil {
		t.Fatal(err)
	}
	document.PresentedPuzzle = strings.Repeat(".", 81)
	if _, err := sealer.Seal(document); !errors.Is(err, &Error{Code: ErrorInvalidPayload}) {
		t.Fatalf("mismatched puzzle error = %v", err)
	}
}

func newTestSealer(t *testing.T, now time.Time, keys map[string][]byte, active string) *Sealer {
	t.Helper()
	sealer, err := New(Config{
		ActiveKeyID: active,
		Keys:        keys,
		Lifetime:    24 * time.Hour,
		Now:         func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return sealer
}

func testGame(t *testing.T) (game.Game, game.Options) {
	t.Helper()
	board := core.NewEmptyBoard()
	board.FromString(testPuzzle)
	options := game.NewDefaultOptions(solver.NewStore())
	return game.NewGame(board, options), options
}

func testKey(fill byte) []byte {
	key := make([]byte, 32)
	for index := range key {
		key[index] = fill
	}
	return key
}

func differentLastCharacter(value string) string {
	if value == "A" {
		return "B"
	}
	return "A"
}

func decodeEnvelope(t *testing.T, token string) envelope {
	t.Helper()
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatal(err)
	}
	var wrapped envelope
	if err := json.Unmarshal(data, &wrapped); err != nil {
		t.Fatal(err)
	}
	return wrapped
}
