package game

import (
	"errors"
	"strings"
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

func gameWithStrategies(t *testing.T, board core.Board, keys ...string) Game {
	t.Helper()
	options := NewDefaultOptions(solver.NewStore())
	options.StrategySolverKeys = keys
	return NewGame(board, options)
}

func TestNakedSingleHintPlanIsDeterministicAndApplicable(t *testing.T) {
	game := gameWithStrategies(t, testProblem(), "naked-single")
	first, second := game.Hint(), game.Hint()
	if first == nil || second == nil || first.PlanID != second.PlanID {
		t.Fatalf("repeated hint plans differ: %+v / %+v", first, second)
	}
	if first.Strategy.ID != "naked-single" || len(first.Steps) != 2 || first.Conclusion.Placement == nil {
		t.Fatalf("unexpected naked-single plan: %+v", first)
	}
	if strings.Contains(first.Steps[0].Message, "Naked Single") || !strings.Contains(first.Steps[0].Message, "row ") || !strings.Contains(first.Steps[0].Message, ", column ") {
		t.Fatalf("naked-single observation is not beginner-first: %q", first.Steps[0].Message)
	}
	if !strings.Contains(first.Steps[1].Message, "Because") || !strings.Contains(first.Steps[1].Message, "called a Naked Single") {
		t.Fatalf("naked-single conclusion does not explain then name the strategy: %q", first.Steps[1].Message)
	}
	if strings.Contains(first.Summary, compactCellName(first.Conclusion.Placement.Position)) || !strings.Contains(first.Summary, cellName(first.Conclusion.Placement.Position)) || !strings.HasPrefix(first.Summary, first.Steps[0].Message) {
		t.Fatalf("naked-single summary is not ordered beginner prose: %q", first.Summary)
	}
	if _, err := game.Apply(ApplyHint{PlanID: "wrong"}); !errors.Is(err, &EngineError{Code: ErrorStaleHint}) {
		t.Fatalf("wrong plan id error = %v", err)
	}
	result, err := game.Apply(ApplyHint{PlanID: first.PlanID})
	if err != nil || len(result.Changes) != 1 || result.Hint == nil || result.Hint.PlanID != first.PlanID {
		t.Fatalf("apply plan = %+v, %v", result, err)
	}
}

func TestHiddenSingleHintPlanHasUnitEvidence(t *testing.T) {
	game := gameWithStrategies(t, testProblem(), "hidden-single")
	plan := game.Hint()
	if plan == nil || plan.Strategy.ID != "hidden-single" || len(plan.Steps) != 3 {
		t.Fatalf("unexpected hidden-single plan: %+v", plan)
	}
	if plan.Steps[0].Kind != HintStepObserve || plan.Steps[1].Kind != HintStepCompare || plan.Steps[2].Kind != HintStepConclude {
		t.Fatalf("unexpected step order: %+v", plan.Steps)
	}
	if strings.Contains(plan.Steps[0].Message, "Hidden Single") || strings.Contains(plan.Steps[1].Message, "Hidden Single") {
		t.Fatalf("hidden-single name appears before the reasoning: %+v", plan.Steps)
	}
	if !strings.Contains(plan.Steps[1].Message, "row ") || !strings.Contains(plan.Steps[1].Message, ", column ") || !strings.Contains(plan.Steps[2].Message, "called a Hidden Single") {
		t.Fatalf("hidden-single steps do not use plain location prose and delayed naming: %+v", plan.Steps)
	}
}

func TestNakedPairHintAppliesAtomicNoteEliminationAndUndo(t *testing.T) {
	var board core.Board
	board.FromString(".5..4....4.1.....3.8753.1.48............8..7..7...1.497.39....5..84.2937945....2.")
	store := solver.NewStore()
	pair := solver.NewNakedPairSolver()
	for pair.Hint(&board) == nil {
		var move *solver.Move
		for _, key := range []string{"naked-single", "hidden-single"} {
			if move = store.GetStrategySolverByKey(key).Apply(&board); move != nil {
				break
			}
		}
		if move == nil || !move.IsPlacement() {
			t.Fatal("basic deductions did not reach the naked-pair fixture")
		}
		if err := board.SetCell(move.Cell); err != nil {
			t.Fatal(err)
		}
	}
	game := gameWithStrategies(t, board, "naked-pair")
	plan := game.Hint()
	if plan == nil || plan.Strategy.ID != "naked-pair" || len(plan.Steps) != 3 || len(plan.Conclusion.Eliminations) == 0 || plan.Conclusion.Placement != nil {
		t.Fatalf("unexpected naked-pair plan: %+v", plan)
	}
	if strings.Contains(plan.Steps[0].Message, "Naked Pair") || strings.Contains(plan.Steps[1].Message, "Naked Pair") {
		t.Fatalf("naked-pair name appears before the visible facts and relationship: %+v", plan.Steps)
	}
	if !strings.Contains(plan.Steps[1].Message, "in some order") || !strings.Contains(plan.Steps[2].Message, "Because") || !strings.Contains(plan.Steps[2].Message, "called a Naked Pair") {
		t.Fatalf("naked-pair steps do not explain the reservation before naming it: %+v", plan.Steps)
	}
	before := game.Snapshot()
	result, err := game.Apply(ApplyHint{PlanID: plan.PlanID})
	if err != nil || len(result.Changes) == 0 || !result.CanUndo {
		t.Fatalf("apply elimination plan = %+v, %v", result, err)
	}
	expectedNotes := before.Candidates
	for _, elimination := range plan.Conclusion.Eliminations {
		expectedNotes[elimination.Position.Row][elimination.Position.Column].Remove(elimination.Value)
	}
	if after := game.Snapshot(); after.Notes != expectedNotes || after.Candidates != expectedNotes {
		t.Fatalf("elimination did not persist candidate progress: expected=%v notes=%v candidates=%v", expectedNotes, after.Notes, after.Candidates)
	}
	next := game.Hint()
	if next != nil && sameEliminations(next.Conclusion.Eliminations, plan.Conclusion.Eliminations) {
		t.Fatalf("next hint repeated the consumed elimination: before=%+v after=%+v", plan, next)
	}
	data, err := game.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	options := NewDefaultOptions(solver.NewStore())
	options.StrategySolverKeys = []string{"naked-pair"}
	restored, err := Restore(data, options)
	if err != nil {
		t.Fatal(err)
	}
	restoredNext := restored.Hint()
	if restoredNext != nil && sameEliminations(restoredNext.Conclusion.Eliminations, plan.Conclusion.Eliminations) {
		t.Fatalf("restored game repeated the consumed elimination: before=%+v after=%+v", plan, restoredNext)
	}
	if _, err := game.Apply(Undo{}); err != nil {
		t.Fatal(err)
	}
	if after := game.Snapshot(); after.Notes != before.Notes {
		t.Fatalf("undo did not restore notes: before=%v after=%v", before.Notes, after.Notes)
	}
}

func sameEliminations(left, right []solver.CandidateRef) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func TestSubsetAndIntersectionFamiliesTeachBeforeNaming(t *testing.T) {
	games := realGamesForHintStrategies(t)
	store := solver.NewStore()
	keys := store.GetAllStrategySolverKeys()
	families := []string{
		"naked-pair", "naked-triple", "naked-quad",
		"hidden-pair", "hidden-triple", "hidden-quad",
		"pointing-pair",
	}
	for _, key := range families {
		t.Run(key, func(t *testing.T) {
			options := NewDefaultOptions(store)
			options.StrategySolverKeys = keys
			current := NewGame(boardFromPuzzle(t, games[key]), options)
			var plan *HintPlan
			for number := 0; number < 128; number++ {
				plan = current.Hint()
				if plan == nil {
					t.Fatalf("exhausted hints before %s", key)
				}
				if plan.Strategy.ID == key {
					break
				}
				if _, err := current.Apply(ApplyHint{PlanID: plan.PlanID}); err != nil {
					t.Fatal(err)
				}
			}
			if plan == nil || plan.Strategy.ID != key || len(plan.Steps) != 3 {
				t.Fatalf("unexpected family plan: %+v", plan)
			}
			for _, step := range plan.Steps[:2] {
				if strings.Contains(step.Message, plan.Strategy.DisplayName) {
					t.Fatalf("strategy vocabulary appears before the reasoning: %q", step.Message)
				}
			}
			last := plan.Steps[2].Message
			expectedName := plan.Strategy.DisplayName
			if key == "pointing-pair" {
				expectedName = "Pointing Pair"
			}
			if !strings.Contains(last, "Because") || !strings.Contains(last, expectedName) {
				t.Fatalf("conclusion does not explain then name the strategy: %q", last)
			}
			for _, step := range plan.Steps {
				if strings.Contains(step.Message, "(") {
					t.Fatalf("family plan exposes coordinate shorthand: %q", step.Message)
				}
			}
		})
	}
}

func TestFishAndRelationalFamiliesTeachBeforeNaming(t *testing.T) {
	games := realGamesForHintStrategies(t)
	store := solver.NewStore()
	keys := store.GetAllStrategySolverKeys()
	families := []string{"x-wing", "swordfish", "jellyfish", "xy-wing", "w-wing", "xyz-wing"}
	for _, key := range families {
		t.Run(key, func(t *testing.T) {
			options := NewDefaultOptions(store)
			options.StrategySolverKeys = keys
			current := NewGame(boardFromPuzzle(t, games[key]), options)
			var plan *HintPlan
			for number := 0; number < 128; number++ {
				plan = current.Hint()
				if plan == nil {
					t.Fatalf("exhausted hints before %s", key)
				}
				if plan.Strategy.ID == key {
					break
				}
				if _, err := current.Apply(ApplyHint{PlanID: plan.PlanID}); err != nil {
					t.Fatal(err)
				}
			}
			if plan == nil || plan.Strategy.ID != key || len(plan.Steps) != 3 {
				t.Fatalf("unexpected family plan: %+v", plan)
			}
			for _, step := range plan.Steps[:2] {
				if strings.Contains(step.Message, plan.Strategy.DisplayName) {
					t.Fatalf("strategy vocabulary appears before the reasoning: %q", step.Message)
				}
				if strings.Contains(step.Message, "(") {
					t.Fatalf("family plan exposes coordinate shorthand: %q", step.Message)
				}
			}
			if !strings.Contains(plan.Steps[2].Message, "Because") || !strings.Contains(plan.Steps[2].Message, plan.Strategy.DisplayName) {
				t.Fatalf("conclusion does not explain then name the strategy: %q", plan.Steps[2].Message)
			}
			if key == "x-wing" || key == "swordfish" || key == "jellyfish" {
				if !strings.Contains(plan.Steps[0].Message, "highlighted intersections") || !strings.Contains(plan.Steps[1].Message, "confine candidate") {
					t.Fatalf("fish plan does not teach base and cross-line confinement: %+v", plan.Steps)
				}
			} else if !strings.Contains(plan.Steps[0].Message, "pivot") && !strings.Contains(plan.Steps[0].Message, "strong link") {
				t.Fatalf("relational plan does not identify its cells and link: %+v", plan.Steps)
			}
		})
	}
}

func TestUniquenessFamilyTeachesBeforeNaming(t *testing.T) {
	games := realGamesForHintStrategies(t)
	store := solver.NewStore()
	keys := store.GetAllStrategySolverKeys()
	families := []string{"bug-plus-one", "unique-rectangle", "unique-rectangle-2", "unique-rectangle-3", "unique-rectangle-4"}
	for _, key := range families {
		t.Run(key, func(t *testing.T) {
			options := NewDefaultOptions(store)
			options.StrategySolverKeys = keys
			current := NewGame(boardFromPuzzle(t, games[key]), options)
			var plan *HintPlan
			for number := 0; number < 128; number++ {
				plan = current.Hint()
				if plan == nil {
					t.Fatalf("exhausted hints before %s", key)
				}
				if plan.Strategy.ID == key {
					break
				}
				if _, err := current.Apply(ApplyHint{PlanID: plan.PlanID}); err != nil {
					t.Fatal(err)
				}
			}
			if plan == nil || plan.Strategy.ID != key || len(plan.Steps) != 3 {
				t.Fatalf("unexpected family plan: %+v", plan)
			}
			for _, step := range plan.Steps[:2] {
				if strings.Contains(step.Message, plan.Strategy.DisplayName) {
					t.Fatalf("strategy vocabulary appears before the reasoning: %q", step.Message)
				}
				if strings.Contains(step.Message, "(") {
					t.Fatalf("family plan exposes coordinate shorthand: %q", step.Message)
				}
			}
			if !strings.Contains(plan.Steps[2].Message, "Because") || !strings.Contains(plan.Steps[2].Message, plan.Strategy.DisplayName) {
				t.Fatalf("conclusion does not explain then name the strategy: %q", plan.Steps[2].Message)
			}
			if key == "bug-plus-one" {
				if !strings.Contains(plan.Steps[0].Message, "only unsolved cell with three candidates") || !strings.Contains(plan.Steps[1].Message, "extra candidate") {
					t.Fatalf("BUG+1 plan does not teach the bivalue exception: %+v", plan.Steps)
				}
			} else {
				if !strings.Contains(plan.Steps[0].Message, "rectangle corners") || !strings.Contains(plan.Steps[1].Message, "two solution") && !strings.Contains(plan.Steps[1].Message, "two-solution") && !strings.Contains(plan.Steps[1].Message, "At least one") && !strings.Contains(plan.Steps[1].Message, "reserve") && !strings.Contains(plan.Steps[1].Message, "appears in exactly") {
					t.Fatalf("rectangle plan does not teach its uniqueness relationship: %+v", plan.Steps)
				}
			}
		})
	}
}

func TestColoringAndChainFamiliesTeachBeforeNaming(t *testing.T) {
	games := realGamesForHintStrategies(t)
	store := solver.NewStore()
	keys := store.GetAllStrategySolverKeys()
	for _, key := range []string{"simple-coloring", "x-cycles", "xy-chain"} {
		t.Run(key, func(t *testing.T) {
			options := NewDefaultOptions(store)
			options.StrategySolverKeys = keys
			current := NewGame(boardFromPuzzle(t, games[key]), options)
			var plan *HintPlan
			for number := 0; number < 128; number++ {
				plan = current.Hint()
				if plan == nil {
					t.Fatalf("exhausted hints before %s", key)
				}
				if plan.Strategy.ID == key {
					break
				}
				if _, err := current.Apply(ApplyHint{PlanID: plan.PlanID}); err != nil {
					t.Fatal(err)
				}
			}
			if plan == nil || plan.Strategy.ID != key || len(plan.Steps) != 3 {
				t.Fatalf("unexpected family plan: %+v", plan)
			}
			for _, step := range plan.Steps[:2] {
				if strings.Contains(step.Message, plan.Strategy.DisplayName) {
					t.Fatalf("strategy vocabulary appears before the reasoning: %q", step.Message)
				}
				if strings.Contains(step.Message, "r1c") || strings.Contains(step.Message, "(") {
					t.Fatalf("family plan exposes compact notation: %q", step.Message)
				}
			}
			if !strings.Contains(plan.Steps[2].Message, "Because") || !strings.Contains(plan.Steps[2].Message, plan.Strategy.DisplayName) {
				t.Fatalf("conclusion does not explain then name the strategy: %q", plan.Steps[2].Message)
			}
			if len(plan.Steps[0].Links) != 1 || len(plan.Steps[1].Links) <= len(plan.Steps[0].Links) {
				t.Fatalf("complex scene is not progressive: %+v", plan.Steps)
			}
			for _, mark := range plan.Steps[1].Marks {
				if mark.Target.Kind == HintTargetRow || mark.Target.Kind == HintTargetColumn || mark.Target.Kind == HintTargetBox {
					t.Fatalf("chain scene paints broad unit scope: %+v", mark)
				}
			}
			switch key {
			case "simple-coloring":
				if !strings.Contains(plan.Steps[0].Message, "Exactly one end") || !strings.Contains(plan.Steps[1].Message, "two groups") || len(plan.Steps[1].Links) == 0 || plan.Steps[1].Nodes[0].Group == "" {
					t.Fatalf("coloring plan does not teach alternating groups: %+v", plan.Steps)
				}
			case "x-cycles":
				if !strings.Contains(plan.Steps[0].Message, "solid or dashed") || !strings.Contains(plan.Steps[1].Message, "returns") || len(plan.Steps[1].Links) < 3 || plan.Steps[1].Nodes[0].Order == 0 {
					t.Fatalf("cycle plan does not teach alternating links: %+v", plan.Steps)
				}
			case "xy-chain":
				if !strings.Contains(plan.Steps[0].Message, "first two-candidate cells") || !strings.Contains(plan.Steps[1].Message, "implication continues") || !strings.Contains(plan.Steps[1].Message, "at least one endpoint") || len(plan.Steps[1].Links) == 0 || plan.Steps[1].Links[0].Kind != solver.LinkImplication {
					t.Fatalf("XY-chain plan does not teach the endpoint implication: %+v", plan.Steps)
				}
			}
		})
	}
}

func TestBoxLineReductionUsesItsSpecificRelationshipName(t *testing.T) {
	source := solver.UnitRef{Kind: solver.UnitRow, Index: 3}
	box := solver.UnitRef{Kind: solver.UnitBox, Index: 3}
	move := &solver.Move{
		Cell:      core.NewCell(core.NewPosition(4, 2), 7),
		Technique: "pointing-pair",
		Evidence: &solver.Evidence{
			Unit:  &source,
			Units: []solver.UnitRef{source, box},
			Premises: []solver.CandidateGroup{
				{Position: core.NewPosition(3, 0), Values: []int{9}},
				{Position: core.NewPosition(3, 1), Values: []int{9}},
			},
		},
	}
	steps := intersectionSteps(move)
	if !strings.Contains(steps[0].Message, "row 4") || !strings.Contains(steps[0].Message, "box 4") || !strings.Contains(steps[2].Message, "Box-Line Reduction") {
		t.Fatalf("unexpected box-line teaching plan: %+v", steps)
	}
}
