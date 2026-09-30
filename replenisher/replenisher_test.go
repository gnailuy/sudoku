package replenisher

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/gnailuy/sudoku/db"
)

type fakeCatalog struct {
	seeds     []db.Puzzle
	existing  map[string]bool
	published []db.Puzzle
}

func (catalog *fakeCatalog) ReplenishmentSeeds(string) ([]db.Puzzle, error) {
	return catalog.seeds, nil
}
func (catalog *fakeCatalog) ContainsPuzzle(puzzle string) (bool, error) {
	return catalog.existing[puzzle], nil
}
func (catalog *fakeCatalog) PublishPuzzleBatch(puzzles []db.Puzzle) (bool, error) {
	catalog.published = append([]db.Puzzle(nil), puzzles...)
	return true, nil
}

func candidate(number int) db.Puzzle {
	return db.Puzzle{Puzzle: fmt.Sprintf("candidate-%d", number), Difficulty: "expert", Score: number,
		MaxTechnique: "x-wing", Source: "replenished", SourceRef: fmt.Sprintf("seed-%d", number)}
}

func TestRunResumesWithoutPublishingPartialBatch(t *testing.T) {
	catalog := &fakeCatalog{seeds: []db.Puzzle{{BasePuzzleID: "seed", Puzzle: "seed", Difficulty: "expert"}}, existing: map[string]bool{}}
	state := filepath.Join(t.TempDir(), "state.json")
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	builder := BuilderFunc(func(db.Puzzle, int64) (db.Puzzle, bool, error) {
		calls++
		if calls == 1 {
			cancel()
		}
		return candidate(calls), true, nil
	})
	config := Config{Difficulty: "expert", Count: 2, RandomSeed: 7, MaxClassifications: 4, StatePath: state}
	result, err := Run(ctx, catalog, builder, config)
	if !errors.Is(err, context.Canceled) || result.Accepted != 1 {
		t.Fatalf("first run = %+v, %v", result, err)
	}
	if len(catalog.published) != 0 {
		t.Fatal("partial batch was published")
	}
	result, err = Run(context.Background(), catalog, builder, config)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if !result.Resumed || !result.Published || len(catalog.published) != 2 {
		t.Fatalf("resume result = %+v, published=%d", result, len(catalog.published))
	}
}

func TestRunRejectsExistingAndStagedDuplicates(t *testing.T) {
	catalog := &fakeCatalog{seeds: []db.Puzzle{{BasePuzzleID: "seed", Puzzle: "seed", Difficulty: "expert"}}, existing: map[string]bool{"candidate-1": true}}
	calls := 0
	builder := BuilderFunc(func(db.Puzzle, int64) (db.Puzzle, bool, error) {
		calls++
		if calls < 3 {
			return candidate(1), true, nil
		}
		return candidate(2), true, nil
	})
	config := Config{Difficulty: "expert", Count: 1, RandomSeed: 7, MaxClassifications: 3, StatePath: filepath.Join(t.TempDir(), "state.json")}
	result, err := Run(context.Background(), catalog, builder, config)
	if err != nil || result.Classifications != 3 || len(catalog.published) != 1 || catalog.published[0].Puzzle != "candidate-2" {
		t.Fatalf("result=%+v err=%v published=%+v", result, err, catalog.published)
	}
}
