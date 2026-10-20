package session

import (
	"path/filepath"
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/sessionfile"
	"github.com/gnailuy/sudoku/solver"
)

const knownPuzzle = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."

func newService(t *testing.T) *Service {
	t.Helper()
	store := solver.NewStore()
	service, err := New(Config{Solver: &store, OpenCatalog: db.Open})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestCreateFromPuzzleReturnsStructuredSource(t *testing.T) {
	result, err := newService(t).Create(Request{Puzzle: knownPuzzle, DatabasePath: filepath.Join(t.TempDir(), "puzzles.db")})
	if err != nil {
		t.Fatal(err)
	}
	problem := result.Game.ProblemBoard()
	if result.Source != SourcePuzzleInput || problem.ToString() != knownPuzzle {
		t.Fatalf("result = source %q, puzzle %q", result.Source, problem.ToString())
	}
}

func TestCatalogFirst(t *testing.T) {
	for _, level := range []string{"hard", "expert", "evil"} {
		if !catalogFirst(level) {
			t.Errorf("catalogFirst(%q) = false", level)
		}
	}
	for _, level := range []string{"easy", "medium", "invalid"} {
		if catalogFirst(level) {
			t.Errorf("catalogFirst(%q) = true", level)
		}
	}
}

func TestCreateFromCatalogReturnsStructuredSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "puzzles.db")
	catalog, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	store := solver.NewStore()
	board, err := core.ParseBoard(knownPuzzle)
	if err != nil {
		t.Fatal(err)
	}
	classification := solver.ClassifyPuzzle(store, board)
	if _, err := catalog.InsertPuzzle(db.Puzzle{Puzzle: knownPuzzle, Difficulty: classification.Difficulty, Score: classification.Score, MaxTechnique: classification.MaxTechnique, Source: "test"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Close(); err != nil {
		t.Fatal(err)
	}
	service, err := New(Config{Solver: &store, OpenCatalog: db.Open})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Create(Request{Difficulty: classification.Difficulty, DatabasePath: path, FromDatabase: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != SourceCatalog || result.RequestedDifficulty != classification.Difficulty {
		t.Fatalf("result = %+v", result)
	}
}

func TestCreateRestoresSavedSession(t *testing.T) {
	store := solver.NewStore()
	board, err := core.ParseBoard(knownPuzzle)
	if err != nil {
		t.Fatal(err)
	}
	current := game.NewGame(board, game.NewDefaultOptions(store))
	data, err := current.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "session.json")
	if err := sessionfile.Write(path, data); err != nil {
		t.Fatal(err)
	}
	service, err := New(Config{Solver: &store, OpenCatalog: db.Open})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Create(Request{ResumePath: path})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != SourceSavedSession || result.ResumePath != path || result.Game.Snapshot() != current.Snapshot() {
		t.Fatalf("restored result = %+v", result)
	}
}

func TestCreateRejectsMissingDependencies(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("missing dependencies unexpectedly succeeded")
	}
}
