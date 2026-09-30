package generator

import (
	"math/rand"
	"testing"
	"time"

	"github.com/gnailuy/sudoku/solver"
)

func TestSeededSolvedBoardGenerationIsDeterministic(t *testing.T) {
	firstOptions := NewProblemOptions(solver.NewStore(), NewHardDifficulty())
	firstOptions.Random = rand.New(rand.NewSource(42))
	first := GenerateNormalizedSolvedBoard(firstOptions)
	first.RandomizeWithRand(firstOptions.Random)

	secondOptions := NewProblemOptions(solver.NewStore(), NewHardDifficulty())
	secondOptions.Random = rand.New(rand.NewSource(42))
	second := GenerateNormalizedSolvedBoard(secondOptions)
	second.RandomizeWithRand(secondOptions.Random)

	if first.ToString() != second.ToString() {
		t.Fatalf("seeded solved boards differ:\n%s\n%s", first.ToString(), second.ToString())
	}
}

func TestGenerateBestEffortEasy(t *testing.T) {
	store := solver.NewStore()
	opts := NewBestEffortOptions(store, NewEasyDifficulty())
	opts.MaxRounds = 1
	opts.MaxDurationMs = 30000 // Keep the single real generation round bounded.

	result := GenerateBestEffort(opts)

	if result.RoundsUsed < 1 {
		t.Fatal("Expected at least 1 round used")
	}
	if result.DurationMs < 0 {
		t.Fatal("Expected non-negative duration")
	}

	// Easy puzzles should almost always be generated successfully.
	if !result.Matched {
		t.Logf("Warning: easy puzzle generation did not match target difficulty "+
			"(got %s), but this is acceptable for best-effort",
			result.Classification.Difficulty)
	}

	// Verify the puzzle is valid and solvable.
	if !result.Puzzle.IsValid() {
		t.Fatal("Generated puzzle is not valid")
	}
	solutions := store.GetDefaultSolver().CountSolutions(&result.Puzzle)
	if solutions != 1 {
		t.Fatalf("Expected 1 solution, got %d", solutions)
	}
}

func TestGenerateBestEffortEnforcesHardDeadline(t *testing.T) {
	store := solver.NewStore()
	opts := NewBestEffortOptions(store, NewEasyDifficulty())
	opts.MaxRounds = 2
	opts.MaxDurationMs = 20

	release := make(chan struct{})
	defer close(release)
	start := time.Now()
	result := generateBestEffortWithRound(opts, func(BestEffortOptions, int) GenerationResult {
		<-release
		return GenerationResult{RoundsUsed: 1}
	})
	if !result.TimedOut || result.RoundsUsed != 0 {
		t.Fatalf("result = %+v, want timeout before a completed round", result)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("hard deadline returned after %s", elapsed)
	}
}

func TestGenerateBestEffortReturnsCompletedBestResultAtDeadline(t *testing.T) {
	store := solver.NewStore()
	opts := NewBestEffortOptions(store, NewHardDifficulty())
	opts.MaxRounds = 2
	opts.MaxDurationMs = 20

	release := make(chan struct{})
	defer close(release)
	result := generateBestEffortWithRound(opts, func(_ BestEffortOptions, round int) GenerationResult {
		if round == 1 {
			return GenerationResult{
				Classification: solver.Classification{Difficulty: "easy"},
				RoundsUsed:     1,
			}
		}
		<-release
		return GenerationResult{RoundsUsed: 2}
	})
	if !result.TimedOut || result.RoundsUsed != 1 || result.Classification.Difficulty != "easy" {
		t.Fatalf("result = %+v, want the completed first-round fallback at timeout", result)
	}
}

func TestGenerateBestEffortReportsAllCompletedClassifications(t *testing.T) {
	store := solver.NewStore()
	opts := NewBestEffortOptions(store, NewHardDifficulty())
	opts.MaxRounds = 2
	opts.MaxDurationMs = 1000

	result := generateBestEffortWithRound(opts, func(_ BestEffortOptions, round int) GenerationResult {
		return GenerationResult{
			Classification: solver.Classification{Difficulty: "easy"},
			RoundsUsed:     round,
		}
	})
	if result.RoundsUsed != 2 {
		t.Fatalf("RoundsUsed = %d, want two completed classifications", result.RoundsUsed)
	}
}

func TestGenerateBestEffortRoundLimited(t *testing.T) {
	store := solver.NewStore()
	opts := NewBestEffortOptions(store, NewHardDifficulty())
	opts.MaxRounds = 1
	opts.MaxDurationMs = 30000 // Long time — round limit should hit first.

	result := GenerateBestEffort(opts)

	if result.RoundsUsed > 1 {
		t.Fatalf("Expected at most 1 round, used %d", result.RoundsUsed)
	}
}

func TestDifficultyLevelName(t *testing.T) {
	tests := []struct {
		difficulty Difficulty
		expected   string
	}{
		{NewEasyDifficulty(), "easy"},
		{NewMediumDifficulty(), "medium"},
		{NewHardDifficulty(), "hard"},
		{NewExpertDifficulty(), "expert"},
		{NewEvilDifficulty(), "evil"},
	}

	for _, tt := range tests {
		got := difficultyLevelName(tt.difficulty)
		if got != tt.expected {
			t.Errorf("difficultyLevelName() = %q, want %q", got, tt.expected)
		}
	}
}

func TestIsBetterMatch(t *testing.T) {
	// Medium is closer to hard than easy.
	if !isBetterMatch("medium", "easy", "hard") {
		t.Error("medium should be better than easy for hard target")
	}
	// Expert is closer to hard than easy.
	if !isBetterMatch("expert", "easy", "hard") {
		t.Error("expert should be better than easy for hard target")
	}
	// Hard is the best match for hard.
	if !isBetterMatch("hard", "easy", "hard") {
		t.Error("hard should be better than easy for hard target")
	}
}
