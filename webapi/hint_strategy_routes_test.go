package webapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
	"github.com/gnailuy/sudoku/solver"
)

func TestGuestHTTPRoutesPreviewAndApplyEveryRegisteredHintStrategy(t *testing.T) {
	if os.Getenv("SUDOKU_API_HINT_MATRIX") != "1" {
		t.Skip("set SUDOKU_API_HINT_MATRIX=1 in the dedicated API acceptance lane")
	}
	var manifest struct {
		SchemaVersion int `json:"schema_version"`
		Cases         []struct {
			Strategy    string `json:"strategy"`
			DisplayName string `json:"display_name"`
			Grade       string `json:"grade"`
			Puzzle      string `json:"puzzle"`
		} `json:"cases"`
	}
	contents, err := os.ReadFile(filepath.Join("..", "game", "testdata", "hint_strategy_games.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 1 || len(manifest.Cases) != 23 {
		t.Fatalf("hint strategy manifest version/cases = %d/%d, want 1/23", manifest.SchemaVersion, len(manifest.Cases))
	}

	handler, closeRuntime := hintStrategyRouteHandler(t)
	defer closeRuntime()
	for _, testCase := range manifest.Cases {
		t.Run(testCase.Strategy, func(t *testing.T) {
			t.Parallel()
			created := request(t, handler, http.MethodPost, "/api/v1/guest/games", "application/json", `{"source":{"kind":"puzzle","puzzle":"`+testCase.Puzzle+`"}}`, nil)
			if created.Code != http.StatusCreated {
				t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
			}
			var current GuestGame
			if err := json.Unmarshal(created.Body.Bytes(), &current); err != nil {
				t.Fatal(err)
			}
			for number := 1; number <= 128; number++ {
				preview := request(t, handler, http.MethodPost, "/api/v1/guest/games/hint", "application/json", `{"document":"`+current.Document+`"}`, nil)
				if preview.Code != http.StatusOK {
					t.Fatalf("preview %d status=%d body=%s", number, preview.Code, preview.Body.String())
				}
				var hint HintPreview
				if err := json.Unmarshal(preview.Body.Bytes(), &hint); err != nil {
					t.Fatal(err)
				}
				matched := hint.Hint.Strategy.Id == testCase.Strategy
				if matched {
					strategy := hint.Hint.Strategy
					if strategy.Name.Id != "strategy."+testCase.Strategy+".name" || strategy.Name.Text != testCase.DisplayName || len(strategy.Name.Arguments) != 0 ||
						string(strategy.Grade) != testCase.Grade || strategy.GradeName.Id != "difficulty."+testCase.Grade+".name" || strategy.GradeName.Text == "" || len(strategy.GradeName.Arguments) != 0 || len(hint.Hint.Steps) < 2 {
						t.Fatalf("public strategy = %+v steps=%d", strategy, len(hint.Hint.Steps))
					}
					if hint.Hint.Summary.Id != "hint.strategy.summary" || hint.Hint.Summary.Text == "" || len(hint.Hint.Summary.Arguments) != 2 {
						t.Fatalf("summary descriptor = %+v", hint.Hint.Summary)
					}
					migratedReasons := map[string]string{
						"naked-pair": "subset-reservation", "naked-triple": "subset-reservation", "naked-quad": "subset-reservation",
						"hidden-pair": "hidden-subset-confinement", "hidden-triple": "hidden-subset-confinement", "hidden-quad": "hidden-subset-confinement",
						"pointing-pair": "intersection-confinement", "x-wing": "fish-confinement", "xy-chain": "implication-chain",
					}
					if want, migrated := migratedReasons[testCase.Strategy]; migrated {
						if hint.Hint.Reason == nil {
							t.Fatalf("migrated strategy %s has no typed reason", testCase.Strategy)
						}
						kind, err := hint.Hint.Reason.Discriminator()
						if err != nil || kind != want {
							t.Fatalf("migrated strategy %s reason = %q, %v; want %q", testCase.Strategy, kind, err, want)
						}
					} else if hint.Hint.Reason != nil {
						t.Fatalf("unmigrated strategy %s unexpectedly exposed a typed reason", testCase.Strategy)
					}
					marks := 0
					for _, step := range hint.Hint.Steps {
						marks += len(step.Marks)
						if step.Message.Id == "" || step.Message.Text == "" {
							t.Fatalf("step message descriptor = %+v", step.Message)
						}
					}
					if marks == 0 {
						t.Fatal("API hint exposed no typed semantic marks")
					}
					for index, step := range hint.Hint.Steps {
						if step.Narration == nil || len(*step.Narration) == 0 {
							t.Fatalf("structured narration missing from HTTP step: %+v", step)
						}
						for _, chunk := range *step.Narration {
							kind, err := chunk.Discriminator()
							if err != nil {
								t.Fatal(err)
							}
							if kind == "message" {
								value, err := chunk.AsHintNarrationMessage()
								if err != nil || value.Message.Id == "" || value.Message.Text == "" {
									t.Fatalf("narration message descriptor = %+v, %v", value, err)
								}
							}
						}
						switch testCase.Strategy {
						case "xy-wing":
							wantLinks := 2
							if index == 0 {
								wantLinks = 1
							}
							if len(step.Links) != wantLinks || step.Links[0].Kind != Implication {
								t.Fatalf("XY-Wing HTTP links at step %d = %+v, want %d implications", index+1, step.Links, wantLinks)
							}
						case "w-wing":
							if len(step.Links) != 1 || step.Links[0].Kind != Strong {
								t.Fatalf("W-Wing HTTP links at step %d = %+v, want one strong link", index+1, step.Links)
							}
						}
					}
				}
				body := fmt.Sprintf(`{"document":%q,"action":{"kind":"apply-hint","expected_revision":%d,"plan_id":%q}}`, current.Document, current.Revision, hint.Hint.PlanId)
				applied := request(t, handler, http.MethodPost, "/api/v1/guest/games/actions", "application/json", body, nil)
				if applied.Code != http.StatusOK {
					t.Fatalf("apply %d status=%d body=%s", number, applied.Code, applied.Body.String())
				}
				var response GuestActionResponse
				if err := json.Unmarshal(applied.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Revision != current.Revision+1 || response.Result.Action != ActionKindApplyHint {
					t.Fatalf("apply result revision/action = %d/%s", response.Revision, response.Result.Action)
				}
				if matched {
					return
				}
				current = GuestGame{Document: response.Document, Revision: response.Revision, ActualDifficulty: response.ActualDifficulty, Snapshot: response.Snapshot}
			}
			t.Fatalf("strategy %s was not reached within 128 HTTP hints", testCase.Strategy)
		})
	}
}

func hintStrategyRouteHandler(t *testing.T) (http.Handler, func()) {
	t.Helper()
	store := solver.NewStore()
	options := game.NewDefaultOptions(store)
	options.StrategySolverKeys = store.GetAllStrategySolverKeys()
	database, err := db.Open(filepath.Join(t.TempDir(), "sudoku.db"))
	if err != nil {
		t.Fatal(err)
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
	newGame := func(puzzle string) game.Game {
		board := core.NewEmptyBoard()
		board.FromString(puzzle)
		return game.NewGame(board, options)
	}
	guest := &GuestRuntime{Documents: documents, Options: options, Create: func(kind, value string) (game.Game, string, Difficulty, error) {
		if kind != "puzzle" {
			return game.Game{}, "", "", fmt.Errorf("unexpected source kind %q", kind)
		}
		return newGame(value), db.BasePuzzleID(value), Easy, nil
	}}
	runtime := &AccountRuntime{
		Guest: guest, OIDC: login, Auth: authentication, Games: games, Claims: claims, Store: database,
		CookiePath: "/app/", CSRFKey: append([]byte(nil), key...),
		CreateOwned: func(kind, value string) (game.Game, string, string, Difficulty, error) {
			return newGame(value), db.BasePuzzleID(value), "test-run", Easy, nil
		},
	}
	server := NewServer()
	server.SetGuestRuntime(guest)
	server.SetAccountRuntime(runtime)
	return NewHandler(server, nil), func() { _ = database.Close() }
}
