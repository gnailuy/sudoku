// Package replenisher builds and atomically publishes exact-grade catalog additions.
package replenisher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/generator"
	"github.com/gnailuy/sudoku/solver"
)

const (
	stateVersion = 1
	policyName   = "exact-seed-clue-addition-v1"
)

var ErrBudgetExhausted = errors.New("replenishment classification budget exhausted before the batch was complete")

type Config struct {
	Difficulty         string `json:"difficulty"`
	Count              int    `json:"count"`
	RandomSeed         int64  `json:"random_seed"`
	MaxClassifications int    `json:"max_classifications"`
	StatePath          string `json:"-"`
}

type State struct {
	Version         int         `json:"version"`
	Config          Config      `json:"config"`
	Classifications int         `json:"classifications"`
	Candidates      []db.Puzzle `json:"candidates"`
	Published       bool        `json:"published"`
}

type Result struct {
	Classifications int
	Accepted        int
	Published       bool
	Resumed         bool
}

type Catalog interface {
	ReplenishmentSeeds(string) ([]db.Puzzle, error)
	ContainsPuzzle(string) (bool, error)
	PublishPuzzleBatch([]db.Puzzle) (bool, error)
}

type Builder interface {
	Build(db.Puzzle, int64) (db.Puzzle, bool, error)
}
type BuilderFunc func(db.Puzzle, int64) (db.Puzzle, bool, error)

func (function BuilderFunc) Build(seed db.Puzzle, randomSeed int64) (db.Puzzle, bool, error) {
	return function(seed, randomSeed)
}

func NewBuilder(store solver.Store, difficulty string) Builder {
	return BuilderFunc(func(seed db.Puzzle, randomSeed int64) (db.Puzzle, bool, error) {
		board, err := generator.GenerateSudokuProblemFromString(seed.Puzzle)
		if err != nil {
			return db.Puzzle{}, false, fmt.Errorf("parse seed %s: %w", seed.BasePuzzleID, err)
		}
		solution := board.Copy()
		deterministic, ok := store.GetDefaultSolver().(interface{ SolveDeterministic(*core.Board) bool })
		if !ok || !deterministic.SolveDeterministic(&solution) {
			return db.Puzzle{}, false, fmt.Errorf("solve seed %s", seed.BasePuzzleID)
		}
		blanks := make([]core.Position, 0, 64)
		board.ForEachCell(func(position core.Position, value int) bool {
			if value == 0 {
				blanks = append(blanks, position)
			}
			return true
		})
		if len(blanks) == 0 {
			return db.Puzzle{}, false, nil
		}
		random := rand.New(rand.NewSource(randomSeed))
		position := blanks[random.Intn(len(blanks))]
		if err := board.Set(position, solution.Get(position)); err != nil {
			return db.Puzzle{}, false, err
		}
		canonical := core.CanonicalPuzzle(board.ToString())
		candidateBoard, err := generator.GenerateSudokuProblemFromString(canonical)
		if err != nil {
			return db.Puzzle{}, false, fmt.Errorf("parse canonical candidate: %w", err)
		}
		classification := solver.ClassifyPuzzle(store, *candidateBoard)
		if !classification.Solved || classification.Difficulty != difficulty {
			return db.Puzzle{}, false, nil
		}
		sourceRef, _ := json.Marshal(map[string]any{"seed_base_puzzle_id": seed.BasePuzzleID, "policy": policyName, "random_seed": randomSeed, "added_row": position.Row, "added_column": position.Column})
		return db.Puzzle{Puzzle: canonical, Difficulty: classification.Difficulty, Score: classification.Score,
			MaxTechnique: classification.MaxTechnique, Source: "replenished", SourceRef: string(sourceRef)}, true, nil
	})
}

func Run(ctx context.Context, catalog Catalog, builder Builder, config Config) (Result, error) {
	if config.Difficulty != "expert" && config.Difficulty != "evil" {
		return Result{}, fmt.Errorf("difficulty must be expert or evil")
	}
	if config.Count <= 0 || config.MaxClassifications <= 0 || config.MaxClassifications < config.Count {
		return Result{}, fmt.Errorf("count and classification budget must be positive, with budget at least count")
	}
	if config.StatePath == "" {
		return Result{}, fmt.Errorf("state path is required")
	}
	state, resumed, err := loadState(config)
	if err != nil {
		return Result{}, err
	}
	if state.Published {
		return Result{Classifications: state.Classifications, Accepted: len(state.Candidates), Resumed: true}, nil
	}
	seeds, err := catalog.ReplenishmentSeeds(config.Difficulty)
	if err != nil {
		return Result{}, err
	}
	if len(seeds) == 0 {
		return Result{}, fmt.Errorf("catalog has no exact-grade %s seeds", config.Difficulty)
	}
	staged := make(map[string]struct{}, len(state.Candidates))
	for _, candidate := range state.Candidates {
		staged[candidate.Puzzle] = struct{}{}
	}
	for len(state.Candidates) < config.Count && state.Classifications < config.MaxClassifications {
		select {
		case <-ctx.Done():
			return Result{Classifications: state.Classifications, Accepted: len(state.Candidates), Resumed: resumed}, ctx.Err()
		default:
		}
		attemptSeed := config.RandomSeed + int64(state.Classifications)*1_000_003
		random := rand.New(rand.NewSource(attemptSeed))
		seed := seeds[random.Intn(len(seeds))]
		candidate, accepted, err := builder.Build(seed, attemptSeed)
		state.Classifications++
		if err != nil {
			return Result{}, err
		}
		if accepted {
			if candidate.Difficulty != config.Difficulty {
				return Result{}, fmt.Errorf("builder returned non-exact candidate %q", candidate.Difficulty)
			}
			if _, duplicate := staged[candidate.Puzzle]; !duplicate {
				exists, err := catalog.ContainsPuzzle(candidate.Puzzle)
				if err != nil {
					return Result{}, err
				}
				if !exists {
					state.Candidates = append(state.Candidates, candidate)
					staged[candidate.Puzzle] = struct{}{}
				}
			}
		}
		if err := saveState(config.StatePath, state); err != nil {
			return Result{}, err
		}
	}
	if len(state.Candidates) != config.Count {
		return Result{Classifications: state.Classifications, Accepted: len(state.Candidates), Resumed: resumed}, ErrBudgetExhausted
	}
	published, err := catalog.PublishPuzzleBatch(state.Candidates)
	if err != nil {
		return Result{}, err
	}
	state.Published = true
	if err := saveState(config.StatePath, state); err != nil {
		return Result{}, err
	}
	return Result{Classifications: state.Classifications, Accepted: len(state.Candidates), Published: published, Resumed: resumed}, nil
}

func loadState(config Config) (State, bool, error) {
	data, err := os.ReadFile(config.StatePath)
	if errors.Is(err, os.ErrNotExist) {
		stateConfig := config
		stateConfig.StatePath = ""
		state := State{Version: stateVersion, Config: stateConfig}
		return state, false, saveState(config.StatePath, state)
	}
	if err != nil {
		return State{}, false, fmt.Errorf("read replenishment state: %w", err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, false, fmt.Errorf("parse replenishment state: %w", err)
	}
	expected := config
	expected.StatePath = ""
	if state.Version != stateVersion || state.Config != expected {
		return State{}, false, fmt.Errorf("replenishment state does not match command configuration")
	}
	return state, true, nil
}

func saveState(path string, state State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create replenishment state directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".replenish-*.tmp")
	if err != nil {
		return fmt.Errorf("create replenishment state: %w", err)
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace replenishment state: %w", err)
	}
	return nil
}
