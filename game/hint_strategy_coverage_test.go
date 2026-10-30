package game

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/message"
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
				assertCanonicalDescriptor(t, first.Title)
				assertCanonicalDescriptor(t, first.Summary)
				assertCanonicalDescriptor(t, first.Strategy.Name)
				assertCanonicalDescriptor(t, first.Strategy.GradeName)
				for index, step := range first.Steps {
					assertCanonicalDescriptor(t, step.Message)
					if !step.HasValidNarration() {
						t.Fatalf("step %d has no complete valid narration: %+v", index+1, step)
					}
					for _, chunk := range step.Narration {
						if chunk.Message != nil {
							assertCanonicalDescriptor(t, *chunk.Message)
						}
					}
				}
				switch key {
				case "naked-pair", "naked-triple", "naked-quad":
					if first.Reason == nil || first.Reason.Kind != HintReasonSubsetReservation || first.Steps[1].Message.ID == "hint.strategy.compare" {
						t.Fatalf("naked subset did not use its typed family reason: %+v", first)
					}
				case "pointing-pair":
					if first.Reason == nil || first.Reason.Kind != HintReasonIntersectionConfinement || first.Steps[1].Message.ID == "hint.strategy.compare" {
						t.Fatalf("intersection did not use its typed family reason: %+v", first)
					}
				case "x-wing", "swordfish", "jellyfish":
					if first.Reason == nil || first.Reason.Kind != HintReasonFishConfinement || first.Steps[1].Message.ID == "hint.strategy.compare" {
						t.Fatalf("fish did not use its typed family reason: %+v", first)
					}
				case "xy-wing", "xyz-wing", "w-wing":
					if first.Steps[1].Message.ID == "hint.strategy.compare" || first.Steps[len(first.Steps)-1].Message.ID == "hint.strategy.conclude" {
						t.Fatalf("%s did not use complete relational-wing descriptors: %+v", key, first)
					}
				case "bug-plus-one", "unique-rectangle", "unique-rectangle-2", "unique-rectangle-3", "unique-rectangle-4":
					if first.Steps[1].Message.ID == "hint.strategy.compare" || first.Steps[len(first.Steps)-1].Message.ID == "hint.strategy.conclude" {
						t.Fatalf("%s did not use complete uniqueness descriptors: %+v", key, first)
					}
				case "simple-coloring":
					if first.Steps[1].Message.ID == "hint.strategy.compare" || first.Steps[len(first.Steps)-1].Message.ID == "hint.strategy.conclude" {
						t.Fatalf("%s did not use complete linked-candidate descriptors: %+v", key, first)
					}
				case "xy-chain":
					if first.Reason == nil || first.Reason.Kind != HintReasonImplicationChain || first.Reason.ImplicationChain == nil || len(first.Steps) != 6 {
						t.Fatalf("xy-chain did not use its approved six-step progression: %+v", first)
					}
				}
				if !planHasRole(*first, HintRolePremise) {
					t.Fatal("plan has no typed candidate premise")
				}
				if key == "simple-coloring" || key == "x-cycles" || key == "xy-chain" {
					if len(first.Steps[1].Nodes) == 0 || len(first.Steps[1].Links) == 0 {
						t.Fatal("relational plan has no explicit nodes and links")
					}
				} else if !planHasUnit(*first) {
					t.Fatal("plan has no typed affected unit")
				}
				last := first.Steps[len(first.Steps)-1]
				if last.Effect == nil {
					t.Fatal("final step has no exact effect")
				}
				if !narrationHasTypedConclusion(last.Narration) {
					t.Fatal("final narration has no typed digit and target")
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

func TestStrategicHintRegressionCorpusHasNoGenericFallbackOrPanic(t *testing.T) {
	puzzles := []string{
		".....1..2..134..5.26.....3......6..7.1..2.89.5.9.1.....528..9..196.5.3..7831...2.",
		"........1......23...2..456...1..6.....7..8...358142....8.9....31...8....5.3.71684",
		"...........1..2.34.3.15.6.........1...5.2.37.712....86...9......5.8.7..396723....",
		".....1..2..1.3..4..52...67....8..5....6.5..8.5.8.93....2.4.9...8.....2.976.3....4",
		".....1..2..1.2..3.24.3.5.1.....1.2...674..15.4..5....6...7...2...4...8759..284...",
		"........1.....123...2.4.5.6.....7......48...5.9..561..6.9.124..47..3.6.2528.6.7..",
		"........1.....234..1536.......7.86.9.4.......8.6.5......32...98..28..5..97.4..1..",
		".......12..1.234...5.4..3.6.......78..6.385...2.5..6.31....7..43....2..76..9.....",
		".....1..2....2314....54..36..2.3...7.74..62..35.1...8..4...5.7..183....55.3.1.8..",
	}
	wantStrategies := map[string]bool{
		"simple-coloring":    false,
		"x-cycles":           false,
		"xy-chain":           false,
		"xyz-wing":           false,
		"unique-rectangle":   false,
		"unique-rectangle-2": false,
		"unique-rectangle-3": false,
		"unique-rectangle-4": false,
	}
	store := solver.NewStore()
	options := NewDefaultOptions(store)
	options.StrategySolverKeys = store.GetAllStrategySolverKeys()
	for puzzleIndex, puzzle := range puzzles {
		current := NewGame(boardFromPuzzle(t, puzzle), options)
		for hintNumber := 0; hintNumber < 1024; hintNumber++ {
			plan := current.Hint()
			if plan == nil {
				if current.Snapshot().Status != StatusSolved {
					t.Fatalf("puzzle %d exhausted hints before solving", puzzleIndex)
				}
				break
			}
			wantStrategies[plan.Strategy.ID] = wantStrategies[plan.Strategy.ID] || plan.Strategy.ID != ""
			for _, step := range plan.Steps {
				if strings.HasPrefix(step.Message.ID, "hint.strategy.") {
					t.Fatalf("puzzle %d hint %d strategy %s used generic descriptor %s", puzzleIndex, hintNumber+1, plan.Strategy.ID, step.Message.ID)
				}
			}
			if _, err := current.Apply(ApplyHint{PlanID: plan.PlanID}); err != nil {
				t.Fatalf("puzzle %d hint %d apply: %v", puzzleIndex, hintNumber+1, err)
			}
			if hintNumber == 1023 {
				t.Fatalf("puzzle %d exceeded hint limit", puzzleIndex)
			}
		}
	}
	for strategy, seen := range wantStrategies {
		if !seen {
			t.Fatalf("regression corpus did not exercise %s", strategy)
		}
	}
}

func assertCanonicalDescriptor(t *testing.T, descriptor message.Descriptor) {
	t.Helper()
	rendered, err := canonicalEnglishMessages.Render(descriptor.ID, descriptor.Arguments...)
	if err != nil {
		t.Fatalf("descriptor %q is absent from canonical catalog: %v", descriptor.ID, err)
	}
	if rendered.Text != descriptor.Text {
		t.Fatalf("descriptor %q fallback = %q, canonical = %q", descriptor.ID, descriptor.Text, rendered.Text)
	}
}

func boardFromPuzzle(t *testing.T, puzzle string) core.Board {
	t.Helper()
	board := core.NewEmptyBoard()
	board.FromString(puzzle)
	return board
}

func narrationHasKind(chunks []HintNarrationChunk, kind HintNarrationKind) bool {
	for _, chunk := range chunks {
		if chunk.Kind == kind {
			return true
		}
	}
	return false
}

func narrationHasTypedConclusion(chunks []HintNarrationChunk) bool {
	if narrationHasKind(chunks, HintNarrationDigit) && narrationHasKind(chunks, HintNarrationTarget) {
		return true
	}
	for _, chunk := range chunks {
		if chunk.Message == nil {
			continue
		}
		hasDigit, hasCells := false, false
		for _, argument := range chunk.Message.Arguments {
			hasDigit = hasDigit || argument.Kind == message.ArgumentDigit || argument.Kind == message.ArgumentDigits || argument.Kind == message.ArgumentCandidate || argument.Kind == message.ArgumentCandidates
			hasCells = hasCells || argument.Kind == message.ArgumentCell || argument.Kind == message.ArgumentCells || argument.Kind == message.ArgumentCandidate || argument.Kind == message.ArgumentCandidates
		}
		if hasDigit && hasCells {
			return true
		}
	}
	return false
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
