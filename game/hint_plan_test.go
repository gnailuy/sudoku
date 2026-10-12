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
	if first.ProtocolVersion != 1 || first.Strategy.ID != "naked-single" || len(first.Steps) != 2 || first.Conclusion.Placement == nil {
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
