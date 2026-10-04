package db

import (
	"errors"
	"testing"
	"time"
)

func TestExternalIdentityUsesIssuerAndSubjectInsteadOfEmail(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	for _, user := range []User{{ID: "user-a", ProfileEmail: "same@example.test"}, {ID: "user-b", ProfileEmail: "same@example.test"}} {
		if err := database.CreateUser(user); err != nil {
			t.Fatal(err)
		}
	}
	identity := ExternalIdentity{Issuer: "https://issuer.example", Subject: "subject-a", UserID: "user-a", ProfileEmail: "old@example.test", DisplayName: "Old"}
	if err := database.LinkExternalIdentity(identity); err != nil {
		t.Fatal(err)
	}
	identity.ProfileEmail = "new@example.test"
	identity.DisplayName = "New"
	if err := database.LinkExternalIdentity(identity); err != nil {
		t.Fatalf("idempotent profile refresh: %v", err)
	}
	identity.UserID = "user-b"
	if err := database.LinkExternalIdentity(identity); !errors.Is(err, ErrIdentityAlreadyLinked) {
		t.Fatalf("cross-user relink error = %v, want ErrIdentityAlreadyLinked", err)
	}
	user, err := database.UserByExternalIdentity(identity.Issuer, identity.Subject)
	if err != nil || user == nil || user.ID != "user-a" || user.ProfileEmail != "new@example.test" || user.DisplayName != "New" {
		t.Fatalf("resolved user = %+v, %v", user, err)
	}
	missing, err := database.UserByExternalIdentity(identity.Issuer, "missing")
	if err != nil || missing != nil {
		t.Fatalf("missing identity = %+v, %v", missing, err)
	}
}

func TestWebSessionsExpireAndRevokeWithoutStoringVerifier(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.CreateUser(User{ID: "user-a"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	session := WebSession{ID: "session-a", UserID: "user-a", VerifierDigest: []byte("digest-a"), IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour)}
	if err := database.CreateWebSession(session); err != nil {
		t.Fatal(err)
	}
	active, err := database.ActiveWebSession([]byte("digest-a"), now)
	if err != nil || active == nil || active.UserID != "user-a" {
		t.Fatalf("active session = %+v, %v", active, err)
	}
	if expired, err := database.ActiveWebSession([]byte("digest-a"), now.Add(2*time.Hour)); err != nil || expired != nil {
		t.Fatalf("expired session = %+v, %v", expired, err)
	}
	if revoked, err := database.RevokeWebSession("user-b", "session-a"); err != nil || revoked {
		t.Fatalf("cross-user revoke = %v, %v", revoked, err)
	}
	if revoked, err := database.RevokeWebSession("user-a", "session-a"); err != nil || !revoked {
		t.Fatalf("owner revoke = %v, %v", revoked, err)
	}
	if active, err := database.ActiveWebSession([]byte("digest-a"), now); err != nil || active != nil {
		t.Fatalf("revoked session = %+v, %v", active, err)
	}
}

func TestAccountGamesAreOwnerScopedAndRevisionChecked(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	for _, userID := range []string{"user-a", "user-b"} {
		if err := database.CreateUser(User{ID: userID}); err != nil {
			t.Fatal(err)
		}
	}
	puzzle := "account-puzzle"
	if _, err := database.InsertPuzzle(Puzzle{Puzzle: puzzle, Difficulty: "easy", Score: 1, MaxTechnique: "single"}); err != nil {
		t.Fatal(err)
	}
	basePuzzleID := BasePuzzleID(puzzle)
	if err := database.InsertPlayRun(PlayRun{ID: "run-a", BasePuzzleID: basePuzzleID, PresentedPuzzle: puzzle}); err != nil {
		t.Fatal(err)
	}
	if err := database.CreateAccountGame(AccountGame{ID: "game-a", UserID: "user-a", BasePuzzleID: basePuzzleID, PlayRunID: "run-a", EngineState: []byte(`{"version":1}`), ActualDifficulty: "easy"}); err != nil {
		t.Fatal(err)
	}
	if other, err := database.AccountGameByID("user-b", "game-a"); err != nil || other != nil {
		t.Fatalf("cross-user read = %+v, %v", other, err)
	}
	if updated, err := database.UpdateAccountGame("user-b", "game-a", 0, []byte(`{"version":2}`)); err != nil || updated {
		t.Fatalf("cross-user update = %v, %v", updated, err)
	}
	if updated, err := database.UpdateAccountGame("user-a", "game-a", 1, []byte(`{"version":2}`)); err != nil || updated {
		t.Fatalf("stale update = %v, %v", updated, err)
	}
	if updated, err := database.UpdateAccountGame("user-a", "game-a", 0, []byte(`{"version":2}`)); err != nil || !updated {
		t.Fatalf("owner update = %v, %v", updated, err)
	}
	game, err := database.AccountGameByID("user-a", "game-a")
	if err != nil || game == nil || game.Revision != 1 || string(game.EngineState) != `{"version":2}` {
		t.Fatalf("updated game = %+v, %v", game, err)
	}
	if deleted, err := database.DeleteAccountGame("user-b", "game-a"); err != nil || deleted {
		t.Fatalf("cross-user delete = %v, %v", deleted, err)
	}
	games, err := database.AccountGamesByUser("user-a", 20)
	if err != nil || len(games) != 1 || games[0].ID != "game-a" || len(games[0].EngineState) != 0 {
		t.Fatalf("account game list = %+v, %v", games, err)
	}
	if games, err := database.AccountGamesByUser("user-b", 20); err != nil || len(games) != 0 {
		t.Fatalf("other account game list = %+v, %v", games, err)
	}
	if _, err := database.AccountGamesByUser("user-a", 101); err == nil {
		t.Fatal("expected bounded account game list to reject an excessive limit")
	}
}

func TestDeleteUserCascadesPrivateStateAndPreservesCatalog(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.CreateUser(User{ID: "user-a"}); err != nil {
		t.Fatal(err)
	}
	if err := database.LinkExternalIdentity(ExternalIdentity{Issuer: "https://issuer.example", Subject: "subject-a", UserID: "user-a"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := database.CreateWebSession(WebSession{ID: "session-a", UserID: "user-a", VerifierDigest: []byte("digest-a"), IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(2 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	puzzle := "preserved-catalog-puzzle"
	if _, err := database.InsertPuzzle(Puzzle{Puzzle: puzzle, Difficulty: "easy", Score: 1, MaxTechnique: "single"}); err != nil {
		t.Fatal(err)
	}
	basePuzzleID := BasePuzzleID(puzzle)
	if err := database.InsertPlayRun(PlayRun{ID: "run-a", BasePuzzleID: basePuzzleID, PresentedPuzzle: puzzle}); err != nil {
		t.Fatal(err)
	}
	if err := database.CreateAccountGame(AccountGame{ID: "game-a", UserID: "user-a", BasePuzzleID: basePuzzleID, PlayRunID: "run-a", EngineState: []byte("state"), ActualDifficulty: "easy"}); err != nil {
		t.Fatal(err)
	}
	if deleted, err := database.DeleteUser("user-a"); err != nil || !deleted {
		t.Fatalf("delete user = %v, %v", deleted, err)
	}
	for _, table := range []string{"external_identities", "web_sessions", "account_games"} {
		var count int
		if err := database.conn.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s count = %d, %v", table, count, err)
		}
	}
	if present, err := database.ContainsPuzzle(puzzle); err != nil || !present {
		t.Fatalf("catalog puzzle present = %v, %v", present, err)
	}
	if run, err := database.PlayRunByID("run-a"); err != nil || run == nil {
		t.Fatalf("play run = %+v, %v", run, err)
	}
}
