package generationexperiment

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/generator"
	"github.com/gnailuy/sudoku/solver"
)

// NewExecutor returns the production adapter for both experiment arms.
func NewExecutor(store solver.Store) Executor {
	return ExecutorFunc(func(job Job) (Execution, error) {
		switch job.Arm {
		case "baseline":
			return executeBaseline(store, job)
		case "candidate":
			return executeCandidate(store, job)
		default:
			return Execution{}, fmt.Errorf("unsupported experiment arm %q", job.Arm)
		}
	})
}

func executeBaseline(store solver.Store, job Job) (Execution, error) {
	difficulty, err := experimentDifficulty(job.Sample.TargetDifficulty)
	if err != nil {
		return Execution{}, err
	}
	opts := generator.NewBestEffortOptions(store, difficulty)
	opts.MaxDurationMs = job.Budget.MaxDurationMS
	opts.MaxRounds = job.Budget.MaxClassifications
	opts.Random = rand.New(rand.NewSource(job.Sample.RandomSeed))
	result := generator.GenerateBestEffort(opts)
	if result.Puzzle.IsEmpty() {
		return Execution{
			Outcome: "timed-out", DurationMS: boundedDuration(result.DurationMs, job.Budget.MaxDurationMS),
			Classifications: result.RoundsUsed,
		}, nil
	}
	return executionFromClassification(job, result.Puzzle, result.Classification,
		boundedDuration(result.DurationMs, job.Budget.MaxDurationMS), result.RoundsUsed, result.TimedOut), nil
}

func executeCandidate(store solver.Store, job Job) (Execution, error) {
	start := time.Now()
	deadline := start.Add(time.Duration(job.Budget.MaxDurationMS) * time.Millisecond)
	seed, err := generator.GenerateSudokuProblemFromString(job.Sample.SeedPuzzle)
	if err != nil {
		return Execution{}, fmt.Errorf("parse seed puzzle: %w", err)
	}
	if store.GetDefaultSolver().CountSolutions(seed) != 1 {
		return Execution{Outcome: "non-unique", DurationMS: boundedDuration(time.Since(start).Milliseconds(), job.Budget.MaxDurationMS)}, nil
	}
	if !time.Now().Before(deadline) {
		return Execution{Outcome: "timed-out", DurationMS: job.Budget.MaxDurationMS}, nil
	}
	solution := seed.Copy()
	deterministic, ok := store.GetDefaultSolver().(interface {
		SolveDeterministic(*core.Board) bool
	})
	if !ok || !deterministic.SolveDeterministic(&solution) {
		return Execution{Outcome: "invalid", DurationMS: boundedDuration(time.Since(start).Milliseconds(), job.Budget.MaxDurationMS)}, nil
	}
	if !time.Now().Before(deadline) {
		return Execution{Outcome: "timed-out", DurationMS: job.Budget.MaxDurationMS}, nil
	}

	random := rand.New(rand.NewSource(job.Sample.RandomSeed))
	current := seed.Copy()
	currentClass := solver.ClassifyPuzzle(store, current)
	classifications := 1
	var bestBoard core.Board
	var bestClass solver.Classification
	hasBest := false

	for classifications < job.Budget.MaxClassifications && time.Now().Before(deadline) {
		candidate, ok := mutatePuzzle(random, current, solution, job.Sample.TargetDifficulty, currentClass)
		if !ok {
			break
		}
		if store.GetDefaultSolver().CountSolutions(&candidate) != 1 {
			continue
		}
		classification := solver.ClassifyPuzzle(store, candidate)
		classifications++
		if !hasBest || betterCandidate(classification, bestClass, job.Sample.TargetDifficulty) {
			bestBoard, bestClass = candidate, classification
			hasBest = true
		}
		if classification.Solved && classification.Difficulty == job.Sample.TargetDifficulty {
			bestBoard, bestClass = candidate, classification
			break
		}
		if acceptCandidate(classification, currentClass, job.Sample.TargetDifficulty) {
			current, currentClass = candidate, classification
		}
	}

	duration := boundedDuration(time.Since(start).Milliseconds(), job.Budget.MaxDurationMS)
	timedOut := !time.Now().Before(deadline)
	if !hasBest {
		return Execution{Outcome: "timed-out", DurationMS: duration, Classifications: classifications}, nil
	}
	return executionFromClassification(job, bestBoard, bestClass, duration, classifications, timedOut), nil
}

func mutatePuzzle(random *rand.Rand, board, solution core.Board, target string, classification solver.Classification) (core.Board, bool) {
	remove := mutationDirection(target, classification)
	positions := make([]core.Position, 0, 81)
	board.ForEachCell(func(position core.Position, value int) bool {
		if (remove && value != 0) || (!remove && value == 0) {
			positions = append(positions, position)
		}
		return true
	})
	if len(positions) == 0 {
		return core.Board{}, false
	}
	position := positions[random.Intn(len(positions))]
	candidate := board.Copy()
	if remove {
		candidate.Unset(position)
	} else {
		_ = candidate.Set(position, solution.Get(position))
	}
	return candidate, true
}

func mutationDirection(target string, classification solver.Classification) bool {
	if !classification.Solved {
		return false
	}
	current := difficultyRank(classification.Difficulty)
	wanted := difficultyRank(target)
	return current < wanted
}

func acceptCandidate(candidate, current solver.Classification, target string) bool {
	return candidateDistance(candidate, target) <= candidateDistance(current, target)
}

func betterCandidate(candidate, current solver.Classification, target string) bool {
	candidateDelta := candidateDistance(candidate, target)
	currentDelta := candidateDistance(current, target)
	if candidateDelta != currentDelta {
		return candidateDelta < currentDelta
	}
	if candidate.Solved != current.Solved {
		return candidate.Solved
	}
	return candidate.Score > current.Score
}

func candidateDistance(classification solver.Classification, target string) int {
	if !classification.Solved {
		return 100
	}
	distance := difficultyRank(classification.Difficulty) - difficultyRank(target)
	if distance < 0 {
		return -distance
	}
	return distance
}

func difficultyRank(value string) int {
	switch value {
	case "easy":
		return 0
	case "medium":
		return 1
	case "hard":
		return 2
	case "expert":
		return 3
	case "evil":
		return 4
	default:
		return 100
	}
}

func executionFromClassification(job Job, board core.Board, classification solver.Classification, duration int64, classifications int, timedOut bool) Execution {
	outcome := "strategy-unsolved"
	if timedOut {
		outcome = "timed-out"
	} else if classification.Solved {
		outcome = "wrong-grade"
		if classification.Difficulty == job.Sample.TargetDifficulty {
			outcome = "exact-grade"
		}
	}
	return Execution{
		Outcome: outcome, Puzzle: board.ToString(), Difficulty: classification.Difficulty,
		Score: classification.Score, MaxTechnique: classification.MaxTechnique,
		TraceDigest: traceDigest(classification.Moves), DurationMS: duration,
		Classifications: classifications,
	}
}

func traceDigest(moves []solver.Move) string {
	hasher := sha256.New()
	for _, move := range moves {
		_, _ = fmt.Fprintf(hasher, "%s|%d|%d|%d|%t\n", move.Technique,
			move.Cell.Position.Row, move.Cell.Position.Column, move.Cell.Value, move.EliminationOnly)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func experimentDifficulty(level string) (generator.Difficulty, error) {
	switch level {
	case "hard":
		return generator.NewHardDifficulty(), nil
	case "expert":
		return generator.NewExpertDifficulty(), nil
	case "evil":
		return generator.NewEvilDifficulty(), nil
	default:
		return generator.Difficulty{}, errors.New("experiment target must be hard, expert, or evil")
	}
}

func boundedDuration(value, maximum int64) int64 {
	if value < 0 {
		return 0
	}
	if value > maximum {
		return maximum
	}
	return value
}
