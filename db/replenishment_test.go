package db

import (
	"path/filepath"
	"testing"
)

func replenishmentPuzzle(name string) Puzzle {
	return Puzzle{Puzzle: name, Difficulty: "expert", Score: 10, MaxTechnique: "x-wing", Source: "replenished", SourceRef: "seed:" + name}
}

func TestPublishPuzzleBatchIsAtomicAndIdempotent(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	batch := []Puzzle{replenishmentPuzzle("candidate-a"), replenishmentPuzzle("candidate-b")}
	published, err := database.PublishPuzzleBatch(batch)
	if err != nil || !published {
		t.Fatalf("publish = %v, %v", published, err)
	}
	published, err = database.PublishPuzzleBatch(batch)
	if err != nil || published {
		t.Fatalf("resume = %v, %v", published, err)
	}
	stats, err := database.GetStats()
	if err != nil || stats.Total != 2 {
		t.Fatalf("stats = %+v, %v", stats, err)
	}
}

func TestPublishPuzzleBatchRollsBackPartialCollision(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.InsertPuzzle(replenishmentPuzzle("candidate-a")); err != nil {
		t.Fatal(err)
	}
	_, err = database.PublishPuzzleBatch([]Puzzle{replenishmentPuzzle("candidate-a"), replenishmentPuzzle("candidate-b")})
	if err == nil {
		t.Fatal("expected partial collision")
	}
	stats, statsErr := database.GetStats()
	if statsErr != nil || stats.Total != 1 {
		t.Fatalf("atomic rollback stats = %+v, %v", stats, statsErr)
	}
}
