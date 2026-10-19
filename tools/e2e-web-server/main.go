// Command e2e-web-server serves deterministic guest games for cross-repository browser acceptance.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
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
	"github.com/gnailuy/sudoku/webapi"
)

var corpusStrategies = map[string]bool{
	"hidden-triple": true, "w-wing": true, "swordfish": true, "naked-quad": true,
	"xyz-wing": true, "jellyfish": true, "unique-rectangle-2": true,
	"unique-rectangle-3": true, "unique-rectangle-4": true,
}

type provider struct{ nonce string }

func (p *provider) AuthorizationURL(state, nonce, _ string) string {
	p.nonce = nonce
	return "https://identity.example/authorize?state=" + url.QueryEscape(state)
}
func (p *provider) Exchange(context.Context, string, string) (oidcauth.Claims, error) {
	now := time.Now().UTC()
	return oidcauth.Claims{Issuer: oidcauth.GoogleIssuer, Subject: "e2e", Audience: []string{"e2e-client"}, Nonce: p.nonce, Email: "player@example.test", EmailVerified: true, Name: "Player", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}, nil
}

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "loopback listen address")
	origin := flag.String("origin", "http://127.0.0.1:4173", "browser origin")
	flag.Parse()

	var manifest struct {
		Cases []struct {
			Strategy string `json:"strategy"`
			Puzzle   string `json:"puzzle"`
		} `json:"cases"`
	}
	contents, err := os.ReadFile(filepath.Join("game", "testdata", "hint_strategy_games.json"))
	must(err)
	must(json.Unmarshal(contents, &manifest))
	var puzzles []string
	for _, entry := range manifest.Cases {
		if corpusStrategies[entry.Strategy] {
			puzzles = append(puzzles, entry.Puzzle)
		}
	}
	if len(puzzles) != len(corpusStrategies) {
		panic(fmt.Sprintf("resolved %d of %d full-game fixtures", len(puzzles), len(corpusStrategies)))
	}

	store := solver.NewStore()
	options := game.NewDefaultOptions(store)
	options.StrategySolverKeys = store.GetAllStrategySolverKeys()
	temporary, err := os.MkdirTemp("", "sudoku-web-e2e-")
	must(err)
	defer os.RemoveAll(temporary)
	database, err := db.Open(filepath.Join(temporary, "sudoku.db"))
	must(err)
	defer database.Close()
	key := make([]byte, 32)
	for i := range key {
		key[i] = 7
	}
	documents, err := guestdoc.New(guestdoc.Config{ActiveKeyID: "e2e", Keys: map[string][]byte{"e2e": key}, Lifetime: time.Hour})
	must(err)
	login, err := oidcauth.New(oidcauth.Config{Provider: &provider{}, Issuer: oidcauth.GoogleIssuer, ClientID: "e2e-client", ReturnPathPrefix: "/"})
	must(err)
	authentication, err := accountauth.New(accountauth.Config{Login: login, Store: database, IdleLifetime: time.Hour, AbsoluteLifetime: 2 * time.Hour})
	must(err)
	games, err := accountgame.New(accountgame.Config{Store: database, Options: options})
	must(err)
	claims, err := guestclaim.New(guestclaim.Config{Documents: documents, Store: database})
	must(err)
	var mutex sync.Mutex
	nextPuzzle := 0
	create := func() (game.Game, string, webapi.Difficulty, error) {
		mutex.Lock()
		if nextPuzzle >= len(puzzles) {
			mutex.Unlock()
			return game.Game{}, "", "", fmt.Errorf("full-game fixture corpus exhausted")
		}
		puzzle := puzzles[nextPuzzle]
		nextPuzzle++
		mutex.Unlock()
		board := core.NewEmptyBoard()
		board.FromString(puzzle)
		current := game.NewGame(board, options)
		classification := solver.ClassifyPuzzle(store, board)
		return current, db.BasePuzzleID(puzzle), webapi.Difficulty(classification.Difficulty), nil
	}
	guest := &webapi.GuestRuntime{Documents: documents, Options: options, Create: func(_, _ string) (game.Game, string, webapi.Difficulty, error) { return create() }}
	runtime := &webapi.AccountRuntime{
		Guest: guest, OIDC: login, Auth: authentication, Games: games, Claims: claims, Store: database,
		CookiePath: "/", CSRFKey: append([]byte(nil), key...),
		CreateOwned: func(_, _ string) (game.Game, string, string, webapi.Difficulty, error) {
			current, baseID, difficulty, err := create()
			return current, baseID, "e2e-run", difficulty, err
		},
	}
	server := webapi.NewServer()
	server.SetGuestRuntime(guest)
	server.SetAccountRuntime(runtime)
	httpServer := &http.Server{Addr: *listen, Handler: webapi.NewHandler(server, []string{*origin}), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	fmt.Printf("Sudoku web E2E server listening on http://%s\n", *listen)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	must(httpServer.Shutdown(shutdown))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
