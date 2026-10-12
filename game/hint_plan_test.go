package game

import (
	"errors"
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
