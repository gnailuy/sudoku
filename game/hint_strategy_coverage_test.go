package game

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

type hintStrategyGameManifest struct {
	SchemaVersion int `json:"schema_version"`
	Cases         []struct {
		Strategy string `json:"strategy"`
		Puzzle   string `json:"puzzle"`
	} `json:"cases"`
}

// realGamesForHintStrategies loads complete, valid games from the maintained
// calibration corpus and catalog audit. The black-box line-CLI matrix consumes
// the same manifest, so package and user-perspective coverage cannot drift.
func realGamesForHintStrategies(t *testing.T) map[string]string {
	t.Helper()
	contents, err := os.ReadFile("testdata/hint_strategy_games.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest hintStrategyGameManifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 1 || len(manifest.Cases) != 23 {
		t.Fatalf("hint strategy manifest version/cases = %d/%d, want 1/23", manifest.SchemaVersion, len(manifest.Cases))
	}
	games := make(map[string]string, len(manifest.Cases))
	for _, entry := range manifest.Cases {
		if entry.Strategy == "" || entry.Puzzle == "" {
			t.Fatal("hint strategy manifest contains an empty strategy or puzzle")
		}
		if _, exists := games[entry.Strategy]; exists {
			t.Fatalf("duplicate hint strategy fixture %q", entry.Strategy)
		}
		games[entry.Strategy] = entry.Puzzle
	}
	return games
}

func TestEveryRegisteredStrategyBuildsTypedPlanFromRealGame(t *testing.T) {
	store := solver.NewStore()
	keys := store.GetAllStrategySolverKeys()
	realGameForHintStrategy := realGamesForHintStrategies(t)
	if len(keys) != 23 {
		t.Fatalf("registered strategy count = %d, want 23", len(keys))
	}

	for _, key := range keys {
		key := key
		t.Run(key, func(t *testing.T) {
			puzzle, ok := realGameForHintStrategy[key]
			if !ok {
				t.Fatalf("missing real-game fixture for %s", key)
			}
			options := NewDefaultOptions(store)
			options.StrategySolverKeys = keys
			board := boardFromPuzzle(t, puzzle)
			current := NewGame(board, options)

			for hintNumber := 1; hintNumber <= 128; hintNumber++ {
				first := current.Hint()
				if first == nil {
					t.Fatalf("real game exhausted hints before reaching %s", key)
				}
				if first.Strategy.ID != key {
					if _, err := current.Apply(ApplyHint{PlanID: first.PlanID}); err != nil {
						t.Fatalf("apply preceding %s hint: %v", first.Strategy.ID, err)
					}
					continue
				}

				second := current.Hint()
				if second == nil || first.PlanID == "" || first.PlanID != second.PlanID {
					t.Fatalf("plan identity is not deterministic: %q / %q", first.PlanID, second.PlanID)
				}
				if len(first.Steps) < 2 {
					t.Fatalf("steps = %d, want an ordered teaching sequence", len(first.Steps))
				}
				if !planHasRole(*first, HintRolePremise) {
					t.Fatal("plan has no typed candidate premise")
				}
				if !planHasUnit(*first) {
					t.Fatal("plan has no typed affected unit")
				}
				last := first.Steps[len(first.Steps)-1]
				if last.Effect == nil {
					t.Fatal("final step has no exact effect")
				}
				if first.Conclusion.Placement == nil && len(first.Conclusion.Eliminations) == 0 {
					t.Fatal("plan has no placement or elimination conclusion")
				}
				return
			}
			t.Fatalf("real game did not reach strategy %s within 128 hints", key)
		})
	}
}

func boardFromPuzzle(t *testing.T, puzzle string) core.Board {
	t.Helper()
	board := core.NewEmptyBoard()
	board.FromString(puzzle)
	return board
}

func planHasUnit(plan HintPlan) bool {
	for _, step := range plan.Steps {
		for _, mark := range step.Marks {
			switch mark.Target.Kind {
			case HintTargetRow, HintTargetColumn, HintTargetBox:
				return true
			}
		}
	}
	return false
}

func planHasRole(plan HintPlan, role HintRole) bool {
	for _, step := range plan.Steps {
		for _, mark := range step.Marks {
			if mark.Role == role {
				return true
			}
		}
	}
	return false
}
