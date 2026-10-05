package webapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gnailuy/sudoku/accountauth"
	"github.com/gnailuy/sudoku/accountgame"
	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/guestclaim"
	"github.com/gnailuy/sudoku/guestdoc"
	"github.com/gnailuy/sudoku/oidcauth"
	"github.com/gnailuy/sudoku/recovery"
	"github.com/gnailuy/sudoku/solver"
)

type accountRouteProvider struct{ nonce string }

func (provider *accountRouteProvider) AuthorizationURL(state, nonce, _ string) string {
	provider.nonce = nonce
	return "https://identity.example/authorize?state=" + url.QueryEscape(state)
}

func (provider *accountRouteProvider) Exchange(context.Context, string, string) (oidcauth.Claims, error) {
	now := time.Now().UTC()
	return oidcauth.Claims{Issuer: oidcauth.GoogleIssuer, Subject: "subject-1", Audience: []string{"client-1"}, Nonce: provider.nonce, Email: "player@example.test", EmailVerified: true, Name: "Player", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}, nil
}

func TestGuestAndAccountRoutesUseSealedStateCookieAuthAndCSRF(t *testing.T) {
	store := solver.NewStore()
	options := game.NewDefaultOptions(store)
	options.StrategySolverKeys = store.GetAllStrategySolverKeys()
	database, err := db.Open(filepath.Join(t.TempDir(), "sudoku.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	baseID := db.BasePuzzleID(knownPuzzle)
	if inserted, err := database.InsertPuzzle(db.Puzzle{BasePuzzleID: baseID, Puzzle: knownPuzzle, Difficulty: "easy", Score: 1, MaxTechnique: "single", Source: "test"}); err != nil || !inserted {
		t.Fatalf("insert puzzle = %v, %v", inserted, err)
	}
	key := make([]byte, 32)
	for i := range key {
		key[i] = 7
	}
	documents, err := guestdoc.New(guestdoc.Config{ActiveKeyID: "test", Keys: map[string][]byte{"test": key}, Lifetime: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	provider := &accountRouteProvider{}
	login, err := oidcauth.New(oidcauth.Config{Provider: provider, Issuer: oidcauth.GoogleIssuer, ClientID: "client-1", ReturnPathPrefix: "/app/"})
	if err != nil {
		t.Fatal(err)
	}
	authentication, err := accountauth.New(accountauth.Config{Login: login, Store: database, IdleLifetime: 24 * time.Hour, AbsoluteLifetime: 30 * 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	games, err := accountgame.New(accountgame.Config{Store: database, Options: options})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := guestclaim.New(guestclaim.Config{Documents: documents, Store: database})
	if err != nil {
		t.Fatal(err)
	}
	newGame := func() game.Game {
		board := core.NewEmptyBoard()
		board.FromString(knownPuzzle)
		return game.NewGame(board, options)
	}
	runtime := &AccountRuntime{
		Documents: documents, OIDC: login, Auth: authentication, Games: games, Claims: claims, Store: database,
		Options: options, CookiePath: "/app/", CSRFKey: append([]byte(nil), key...),
		CreateGuest: func(_, _ string) (game.Game, string, Difficulty, error) {
			return newGame(), baseID, Easy, nil
		},
		CreateOwned: func(_, _ string) (game.Game, string, string, Difficulty, error) {
			runID, err := recovery.NewID()
			if err != nil {
				return game.Game{}, "", "", "", err
			}
			if err := database.InsertPlayRun(db.PlayRun{ID: runID, BasePuzzleID: baseID, PresentedPuzzle: knownPuzzle}); err != nil {
				return game.Game{}, "", "", "", err
			}
			return newGame(), baseID, runID, Easy, nil
		},
	}
	registry, err := NewRegistry(recovery.NewStore(filepath.Join(t.TempDir(), "recovery")), options)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(registry, func(_, _ string) (game.Game, error) { return newGame(), nil })
	server.SetAccountRuntime(runtime)
	handler := NewHandler(server, "", nil)

	created := request(t, handler, http.MethodPost, "/api/v1/guest/games", "application/json", `{"source":{"kind":"puzzle","puzzle":"`+knownPuzzle+`"}}`, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("guest create status=%d body=%s", created.Code, created.Body.String())
	}
	var guest GuestGame
	if err := json.Unmarshal(created.Body.Bytes(), &guest); err != nil || guest.Document == "" || guest.Revision != 0 {
		t.Fatalf("guest create = %+v, %v", guest, err)
	}
	action := request(t, handler, http.MethodPost, "/api/v1/guest/games/actions", "application/json", `{"document":"`+guest.Document+`","action":{"kind":"set-value","expected_revision":0,"row":1,"column":1,"value":4}}`, nil)
	if action.Code != http.StatusOK {
		t.Fatalf("guest action status=%d body=%s", action.Code, action.Body.String())
	}
	var guestAction GuestActionResponse
	if err := json.Unmarshal(action.Body.Bytes(), &guestAction); err != nil || guestAction.Revision != 1 || guestAction.Snapshot.Values[0][0] != 4 {
		t.Fatalf("guest action = %+v, %v", guestAction, err)
	}

	unauthenticated := request(t, handler, http.MethodGet, "/api/v1/account", "", "", nil)
	if unauthenticated.Code != http.StatusUnauthorized || !strings.Contains(unauthenticated.Body.String(), "account-unauthorized") {
		t.Fatalf("unauthenticated account status=%d body=%s", unauthenticated.Code, unauthenticated.Body.String())
	}

	loginAccount := func() (string, Account) {
		t.Helper()
		begin := request(t, handler, http.MethodGet, "/api/v1/auth/google/start?return_to=%2Fapp%2Fgame", "", "", nil)
		if begin.Code != http.StatusSeeOther {
			t.Fatalf("login begin status=%d body=%s", begin.Code, begin.Body.String())
		}
		location, err := url.Parse(begin.Header().Get("Location"))
		if err != nil || location.Query().Get("state") == "" {
			t.Fatalf("login location = %q, %v", begin.Header().Get("Location"), err)
		}
		callback := request(t, handler, http.MethodGet, "/api/v1/auth/google/callback?state="+url.QueryEscape(location.Query().Get("state"))+"&code=accepted", "", "", nil)
		if callback.Code != http.StatusSeeOther || callback.Header().Get("Location") != "/app/game" {
			t.Fatalf("callback status=%d headers=%v body=%s", callback.Code, callback.Header(), callback.Body.String())
		}
		cookie := strings.SplitN(callback.Header().Get("Set-Cookie"), ";", 2)[0]
		accountResponse := request(t, handler, http.MethodGet, "/api/v1/account", "", "", map[string]string{"Cookie": cookie})
		if accountResponse.Code != http.StatusOK {
			t.Fatalf("account status=%d body=%s", accountResponse.Code, accountResponse.Body.String())
		}
		var account Account
		if err := json.Unmarshal(accountResponse.Body.Bytes(), &account); err != nil || account.CsrfToken == "" || account.Email != "player@example.test" {
			t.Fatalf("account = %+v, %v", account, err)
		}
		return cookie, account
	}

	cookie, account := loginAccount()
	headers := map[string]string{"Cookie": cookie, "X-Sudoku-CSRF": account.CsrfToken}
	owned := request(t, handler, http.MethodPost, "/api/v1/account/games", "application/json", `{"source":{"kind":"puzzle","puzzle":"`+knownPuzzle+`"}}`, headers)
	if owned.Code != http.StatusCreated {
		t.Fatalf("account create status=%d body=%s", owned.Code, owned.Body.String())
	}
	claimBody := `{"document":"` + guestAction.Document + `"}`
	claimed := request(t, handler, http.MethodPost, "/api/v1/account/games/claim", "application/json", claimBody, headers)
	if claimed.Code != http.StatusCreated {
		t.Fatalf("claim status=%d body=%s", claimed.Code, claimed.Body.String())
	}
	var accountGame AccountGame
	if err := json.Unmarshal(claimed.Body.Bytes(), &accountGame); err != nil || accountGame.Revision != 1 || accountGame.Snapshot.Values[0][0] != 4 {
		t.Fatalf("claimed game = %+v, %v", accountGame, err)
	}
	retry := request(t, handler, http.MethodPost, "/api/v1/account/games/claim", "application/json", claimBody, headers)
	if retry.Code != http.StatusOK {
		t.Fatalf("claim retry status=%d body=%s", retry.Code, retry.Body.String())
	}
	badCSRF := request(t, handler, http.MethodPost, "/api/v1/account/games/"+accountGame.Id+"/actions", "application/json", `{"kind":"clear-value","expected_revision":1,"row":1,"column":1}`, map[string]string{"Cookie": cookie, "X-Sudoku-CSRF": strings.Repeat("x", 43)})
	if badCSRF.Code != http.StatusForbidden {
		t.Fatalf("bad csrf status=%d body=%s", badCSRF.Code, badCSRF.Body.String())
	}
	mutated := request(t, handler, http.MethodPost, "/api/v1/account/games/"+accountGame.Id+"/actions", "application/json", `{"kind":"clear-value","expected_revision":1,"row":1,"column":1}`, headers)
	if mutated.Code != http.StatusOK {
		t.Fatalf("account action status=%d body=%s", mutated.Code, mutated.Body.String())
	}
	listed := request(t, handler, http.MethodGet, "/api/v1/account/games", "", "", map[string]string{"Cookie": cookie})
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), accountGame.Id) {
		t.Fatalf("account list status=%d body=%s", listed.Code, listed.Body.String())
	}
	logout := request(t, handler, http.MethodPost, "/api/v1/auth/logout", "application/json", `{}`, headers)
	if logout.Code != http.StatusNoContent || !strings.Contains(logout.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("logout status=%d headers=%v body=%s", logout.Code, logout.Header(), logout.Body.String())
	}

	cookie, account = loginAccount()
	headers = map[string]string{"Cookie": cookie, "X-Sudoku-CSRF": account.CsrfToken}
	revoked := request(t, handler, http.MethodPost, "/api/v1/account/sessions/revoke", "application/json", `{}`, headers)
	if revoked.Code != http.StatusNoContent || !strings.Contains(revoked.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("revoke status=%d headers=%v body=%s", revoked.Code, revoked.Header(), revoked.Body.String())
	}

	cookie, account = loginAccount()
	headers = map[string]string{"Cookie": cookie, "X-Sudoku-CSRF": account.CsrfToken}
	deleted := request(t, handler, http.MethodDelete, "/api/v1/account", "", "", headers)
	if deleted.Code != http.StatusNoContent || !strings.Contains(deleted.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("delete account status=%d headers=%v body=%s", deleted.Code, deleted.Header(), deleted.Body.String())
	}
	if present, err := database.ContainsPuzzle(knownPuzzle); err != nil || !present {
		t.Fatalf("shared puzzle after account deletion present=%v err=%v", present, err)
	}
}

func TestAccountRoutesRemainUnavailableWithoutPrivateConfiguration(t *testing.T) {
	handler, _, _ := testHandler(t, "", nil)
	for _, test := range []struct {
		method      string
		path        string
		contentType string
		body        string
	}{
		{method: http.MethodPost, path: "/api/v1/guest/games", contentType: "application/json", body: `{"source":{"kind":"puzzle","puzzle":"` + knownPuzzle + `"}}`},
		{method: http.MethodGet, path: "/api/v1/account"},
	} {
		response := request(t, handler, test.method, test.path, test.contentType, test.body, nil)
		if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "account-unavailable") {
			t.Fatalf("%s %s status=%d body=%s", test.method, test.path, response.Code, response.Body.String())
		}
	}
}
