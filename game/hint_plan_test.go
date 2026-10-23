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

func TestHintStepHasValidNarrationRequiresOnePayloadPerKind(t *testing.T) {
	position := core.NewPosition(3, 6)
	validTarget := HintTarget{Kind: HintTargetCell, Position: &position}
	validMessage := hintDescriptor("hint.strategy.observe")
	tests := []struct {
		name  string
		chunk HintNarrationChunk
		valid bool
	}{
		{name: "message", chunk: narrationMessage(validMessage), valid: true},
		{name: "target", chunk: narrationTarget(validTarget), valid: true},
		{name: "digit", chunk: narrationDigit(7), valid: true},
		{name: "empty message", chunk: HintNarrationChunk{Kind: HintNarrationMessage}, valid: false},
		{name: "conflicting target payload", chunk: HintNarrationChunk{Kind: HintNarrationTarget, Message: &validMessage, Target: &validTarget}, valid: false},
		{name: "missing target", chunk: HintNarrationChunk{Kind: HintNarrationTarget}, valid: false},
		{name: "invalid digit", chunk: HintNarrationChunk{Kind: HintNarrationDigit, Value: 10}, valid: false},
		{name: "unknown kind", chunk: HintNarrationChunk{Kind: "future"}, valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			step := HintStep{Narration: []HintNarrationChunk{test.chunk}}
			if got := step.HasValidNarration(); got != test.valid {
				t.Fatalf("HasValidNarration() = %v, want %v", got, test.valid)
			}
		})
	}
}

func TestHintRepairsWrongValueBeforeStrategy(t *testing.T) {
	game := newTestGame()
	position := core.NewPosition(0, 2)
	if _, err := game.Apply(SetValue{Position: position, Value: 9}); err != nil {
		t.Fatal(err)
	}
	plan := game.Hint()
	if plan == nil || plan.Kind != HintKindValueCorrection || plan.Conclusion.ClearValue == nil || *plan.Conclusion.ClearValue != position {
		t.Fatalf("unexpected value correction plan: %+v", plan)
	}
	if _, err := game.Apply(ApplyHint{PlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	if snapshot := game.Snapshot(); snapshot.Values[position.Row][position.Column] != 0 || snapshot.Invalid[position.Row][position.Column] {
		t.Fatalf("value correction did not clear the wrong value: %+v", snapshot)
	}
}

func TestHintRepairsCandidateCellBeforeStrategy(t *testing.T) {
	game := newTestGame()
	solution := game.PlayBoard()
	if !game.completeSolver.Solve(&solution) {
		t.Fatal("test puzzle has no solution")
	}
	position := core.NewPosition(0, 2)
	board := game.PlayBoard()
	legal := board.Candidates(position)
	required := solution.Get(position)
	broken := legal
	broken.Remove(required)
	for value := 1; value <= 9; value++ {
		if !legal.Has(value) {
			broken.Add(value)
			break
		}
	}
	if broken == legal || broken.Has(required) || (broken &^ legal).IsEmpty() {
		t.Fatalf("test setup did not create both correction classes: legal=%v broken=%v", legal.Values(), broken.Values())
	}
	if _, err := game.Apply(SetCandidates{Position: position, Values: broken.Values()}); err != nil {
		t.Fatal(err)
	}
	beforeCorrection := game.Snapshot()
	plan := game.Hint()
	if plan == nil || plan.Kind != HintKindCandidateCorrection || plan.Conclusion.SetCandidates == nil || plan.Conclusion.SetCandidates.Position != position || plan.Conclusion.SetCandidates.Values != legal {
		t.Fatalf("unexpected candidate correction plan: %+v", plan)
	}
	if _, err := game.Apply(ApplyHint{PlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	if got := game.Snapshot().Candidates[position.Row][position.Column]; got != plan.Conclusion.SetCandidates.Values {
		t.Fatalf("candidate correction = %v, want %v", got.Values(), plan.Conclusion.SetCandidates.Values.Values())
	}
	if _, err := game.Apply(Undo{}); err != nil {
		t.Fatal(err)
	}
	if got := game.Snapshot(); got.Candidates != beforeCorrection.Candidates {
		t.Fatal("one undo did not restore the complete pre-correction candidate grid")
	}
}

func TestHintResetsCorruptCandidateCellToFullLocalLegality(t *testing.T) {
	base := newTestGame()
	position := core.NewPosition(0, 2)
	board := base.PlayBoard()
	legal := board.Candidates(position)
	solution := base.PlayBoard()
	if !base.completeSolver.Solve(&solution) {
		t.Fatal("test puzzle has no solution")
	}
	required := solution.Get(position)
	var otherLegal, impossible int
	for value := 1; value <= 9; value++ {
		if legal.Has(value) && value != required && otherLegal == 0 {
			otherLegal = value
		}
		if !legal.Has(value) && impossible == 0 {
			impossible = value
		}
	}
	if otherLegal == 0 || impossible == 0 {
		t.Fatalf("test setup needs multiple legal candidates and one impossible digit: %v", legal.Values())
	}

	tests := []struct {
		name   string
		values []int
	}{
		{name: "reduced set with impossible digit", values: []int{required, impossible}},
		{name: "missing solution-supporting digit", values: []int{otherLegal}},
		{name: "empty set", values: []int{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			game := newTestGame()
			if _, err := game.Apply(SetCandidates{Position: position, Values: test.values}); err != nil {
				t.Fatal(err)
			}
			plan := game.Hint()
			if plan == nil || plan.Kind != HintKindCandidateCorrection || plan.Conclusion.SetCandidates == nil {
				t.Fatalf("expected candidate correction plan, got %+v", plan)
			}
			if got := plan.Conclusion.SetCandidates.Values; got != legal {
				t.Fatalf("candidate correction = %v, want full locally legal set %v", got.Values(), legal.Values())
			}
			for _, step := range plan.Steps {
				message := strings.ToLower(step.Message.Text)
				if strings.Contains(message, "solution") || strings.Contains(message, "required candidate") || strings.Contains(message, "keep every valid") {
					t.Fatalf("candidate correction leaks solution-specific reasoning: %q", step.Message.Text)
				}
			}
		})
	}
}

func TestHintPreservesLocallyLegalReducedCandidates(t *testing.T) {
	game := newTestGame()
	position := core.NewPosition(0, 2)
	solution := game.PlayBoard()
	if !game.completeSolver.Solve(&solution) {
		t.Fatal("test puzzle has no solution")
	}
	if _, err := game.Apply(SetCandidates{Position: position, Values: []int{solution.Get(position)}}); err != nil {
		t.Fatal(err)
	}
	if plan := game.Hint(); plan != nil && plan.Kind == HintKindCandidateCorrection {
		t.Fatalf("locally legal reduced candidates were treated as corrupt: %+v", plan)
	}
}

func TestReportedSimpleColoringUsesReducedAuthoritativeCandidates(t *testing.T) {
	store := solver.NewStore()
	options := NewDefaultOptions(store)
	options.StrategySolverKeys = store.GetAllStrategySolverKeys()
	current := NewGame(boardFromPuzzle(t, realGamesForHintStrategies(t)["simple-coloring"]), options)

	var coloring *HintPlan
	for hintNumber := 1; hintNumber <= 128; hintNumber++ {
		coloring = current.Hint()
		if coloring == nil {
			t.Fatal("exhausted hints before Simple Coloring")
		}
		if coloring.Kind == HintKindStrategy && coloring.Strategy.ID == "simple-coloring" {
			break
		}
		if _, err := current.Apply(ApplyHint{PlanID: coloring.PlanID}); err != nil {
			t.Fatal(err)
		}
	}
	if coloring == nil || coloring.Strategy.ID != "simple-coloring" || len(coloring.Steps) < 3 {
		t.Fatalf("did not reach the reported strategy shape: %+v", coloring)
	}

	solution := current.PlayBoard()
	if !current.completeSolver.Solve(&solution) {
		t.Fatal("fixture has no solution")
	}
	removed := make([]solver.CandidateRef, 0, 2)
	seen := map[solver.CandidateRef]bool{}
	for _, node := range coloring.Steps[2].Nodes {
		if node.Target.Position == nil || node.Target.Kind != HintTargetCandidate {
			continue
		}
		ref := solver.CandidateRef{Position: *node.Target.Position, Value: node.Target.Value}
		if seen[ref] || solution.Get(ref.Position) == ref.Value {
			continue
		}
		seen[ref] = true
		values := current.Snapshot().Candidates[ref.Position.Row][ref.Position.Column]
		values.Remove(ref.Value)
		if _, err := current.Apply(SetCandidates{Position: ref.Position, Values: values.Values()}); err != nil {
			t.Fatal(err)
		}
		removed = append(removed, ref)
		if len(removed) == 2 {
			break
		}
	}
	if len(removed) != 2 {
		t.Fatalf("Simple Coloring fixture provided only %d removable non-solution nodes", len(removed))
	}

	after := current.Hint()
	if after == nil || after.Kind != HintKindStrategy {
		t.Fatalf("valid reduced candidates did not proceed to a strategy hint: %+v", after)
	}
	for _, step := range after.Steps {
		for _, node := range step.Nodes {
			if node.Target.Position == nil {
				continue
			}
			for _, ref := range removed {
				if *node.Target.Position == ref.Position && node.Target.Value == ref.Value {
					t.Fatalf("removed candidate reappeared as a hint node: %+v", ref)
				}
			}
		}
	}
}

func TestHintPlanIDIgnoresCanonicalFallbackWording(t *testing.T) {
	game := gameWithStrategies(t, testProblem(), "naked-single")
	plan := game.Hint()
	if plan == nil {
		t.Fatal("expected a hint plan")
	}
	want := plan.PlanID
	plan.Title.Text = "translated title"
	plan.Summary.Text = "translated summary"
	plan.Strategy.Name.Text = "translated strategy"
	plan.Strategy.GradeName.Text = "translated grade"
	for index := range plan.Steps {
		plan.Steps[index].Message.Text = "translated step"
		for chunkIndex := range plan.Steps[index].Narration {
			if plan.Steps[index].Narration[chunkIndex].Message != nil {
				plan.Steps[index].Narration[chunkIndex].Message.Text = "translated narration"
			}
		}
	}
	if got := deterministicPlanID(game.Snapshot(), *plan); got != want {
		t.Fatalf("fallback wording changed plan id: got %q, want %q", got, want)
	}
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
	for _, step := range first.Steps {
		if len(step.Narration) == 0 {
			t.Fatalf("naked-single step lacks structured narration: %+v", step)
		}
	}
	if first.Steps[0].Message.ID != "hint.strategy.observe" || first.Steps[1].Message.ID != "hint.strategy.conclude" || first.Summary.ID != "hint.strategy.summary" {
		t.Fatalf("naked-single descriptors are incomplete: %+v", first)
	}
	if _, err := game.Apply(ApplyHint{PlanID: "wrong"}); !errors.Is(err, &EngineError{Code: ErrorStaleHint}) {
		t.Fatalf("wrong plan id error = %v", err)
	}
	result, err := game.Apply(ApplyHint{PlanID: first.PlanID})
	if err != nil || len(result.Changes) == 0 || result.Hint == nil || result.Hint.PlanID != first.PlanID {
		t.Fatalf("apply plan = %+v, %v", result, err)
	}
}

func TestHiddenSingleHintPlanHasUnitEvidence(t *testing.T) {
	game := gameWithStrategies(t, testProblem(), "hidden-single")
	plan := game.Hint()
	if plan == nil || plan.Strategy.ID != "hidden-single" || len(plan.Steps) != 3 {
		t.Fatalf("unexpected hidden-single plan: %+v", plan)
	}
	for _, step := range plan.Steps {
		if len(step.Narration) == 0 {
			t.Fatalf("hidden-single step lacks structured narration: %+v", step)
		}
	}
	if plan.Steps[0].Kind != HintStepObserve || plan.Steps[1].Kind != HintStepExplain || plan.Steps[2].Kind != HintStepApply {
		t.Fatalf("unexpected step order: %+v", plan.Steps)
	}
	if plan.Steps[0].Message.ID != "hint.strategy.observe" || plan.Steps[1].Message.ID != "hint.strategy.compare" || plan.Steps[2].Message.ID != "hint.strategy.conclude" {
		t.Fatalf("hidden-single descriptors are incomplete: %+v", plan.Steps)
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
	if plan.Reason == nil || plan.Reason.Kind != HintReasonSubsetReservation || plan.Reason.SubsetReservation == nil {
		t.Fatalf("naked-pair reason is incomplete: %+v", plan.Reason)
	}
	if plan.Steps[0].Message.ID != "hint.subset-reservation.observe" || plan.Steps[1].Message.ID != "hint.subset-reservation.explain" || plan.Steps[2].Message.ID != "hint.subset-reservation.apply" {
		t.Fatalf("naked-pair descriptors are incomplete: %+v", plan.Steps)
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
	if after := game.Snapshot(); after.Candidates != expectedNotes {
		t.Fatalf("elimination did not persist candidate progress: expected=%v candidates=%v", expectedNotes, after.Candidates)
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
	if after := game.Snapshot(); after.Candidates != before.Candidates {
		t.Fatalf("undo did not restore notes: before=%v after=%v", before.Candidates, after.Candidates)
	}
}

func TestReportedAccountGamePreservesEliminationsAcrossPlayerPlacement(t *testing.T) {
	var board core.Board
	board.FromString("................21..3..8.97..2..6.....7..4568.65.319....9..31...2..8..34.34.65...")
	store := solver.NewStore()
	options := NewDefaultOptions(store)
	options.StrategySolverKeys = store.GetAllStrategySolverKeys()
	current := NewGame(board, options)

	// Reproduce the reported account game's exact value-entry order before
	// automatic notes and the first seven elimination hints were applied.
	for _, input := range []core.Cell{
		core.NewCell(core.NewPosition(3, 8), 3),
		core.NewCell(core.NewPosition(5, 8), 2),
		core.NewCell(core.NewPosition(8, 8), 9),
		core.NewCell(core.NewPosition(4, 0), 3),
		core.NewCell(core.NewPosition(4, 1), 1),
		core.NewCell(core.NewPosition(3, 7), 1),
		core.NewCell(core.NewPosition(7, 0), 5),
		core.NewCell(core.NewPosition(0, 5), 2),
		core.NewCell(core.NewPosition(2, 0), 2),
		core.NewCell(core.NewPosition(8, 6), 2),
	} {
		if _, err := current.Apply(SetValue{Position: input.Position, Value: input.Value}); err != nil {
			t.Fatal(err)
		}
	}
	consumedEliminations := make([][]solver.CandidateRef, 0, 7)
	for number := 0; number < 7; number++ {
		plan := current.Hint()
		if plan == nil || len(plan.Conclusion.Eliminations) == 0 {
			t.Fatalf("hint %d is not the reported elimination sequence: %+v", number+1, plan)
		}
		consumedEliminations = append(consumedEliminations, append([]solver.CandidateRef(nil), plan.Conclusion.Eliminations...))
		if _, err := current.Apply(ApplyHint{PlanID: plan.PlanID}); err != nil {
			t.Fatal(err)
		}
	}

	retained := core.NewPosition(3, 3)
	if current.Snapshot().Candidates[retained.Row][retained.Column].Has(9) {
		t.Fatal("reported naked-pair elimination was not present before placement")
	}
	placement := current.Hint()
	if placement == nil || placement.Conclusion.Placement == nil || placement.Conclusion.Placement.Position != core.NewPosition(8, 7) || placement.Conclusion.Placement.Value != 8 {
		t.Fatalf("unexpected reported placement hint: %+v", placement)
	}
	if _, err := current.Apply(SetValue{Position: placement.Conclusion.Placement.Position, Value: placement.Conclusion.Placement.Value}); err != nil {
		t.Fatal(err)
	}
	if current.Snapshot().Candidates[retained.Row][retained.Column].Has(9) {
		t.Fatal("entering the reported hidden single erased prior hint eliminations")
	}
	serialized, err := current.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Restore(serialized, options)
	if err != nil {
		t.Fatalf("placement left eliminations that could not be restored: %v", err)
	}
	if restored.Snapshot().Candidates[retained.Row][retained.Column].Has(9) {
		t.Fatal("restoring the reported game erased prior hint eliminations")
	}
	if restored.Snapshot().Candidates != current.Snapshot().Candidates {
		t.Fatal("account-game serialization changed the candidate grid")
	}
	for label, next := range map[string]*HintPlan{"current": current.Hint(), "restored": restored.Hint()} {
		if next == nil {
			t.Fatalf("%s game has no next hint after the reported placement", label)
		}
		for _, consumed := range consumedEliminations {
			if sameEliminations(next.Conclusion.Eliminations, consumed) {
				t.Fatalf("%s game repeated a consumed elimination after placement: %+v", label, next)
			}
		}
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
				if strings.Contains(step.Message.Text, plan.Strategy.Name.Text) {
					t.Fatalf("strategy vocabulary appears before the reasoning: %q", step.Message.Text)
				}
			}
			if key == "naked-pair" || key == "naked-triple" || key == "naked-quad" {
				if plan.Reason == nil || plan.Reason.Kind != HintReasonSubsetReservation {
					t.Fatalf("naked-subset contract = reason %+v, conclusion %+v", plan.Reason, plan.Steps[2].Message)
				}
				expectedMessage := "hint.subset-reservation.apply"
				if plan.Conclusion.Placement != nil {
					expectedMessage = "hint.subset-reservation.place"
				}
				if plan.Steps[2].Message.ID != expectedMessage {
					t.Fatalf("naked-subset action descriptor = %q, want %q", plan.Steps[2].Message.ID, expectedMessage)
				}
				reason := plan.Reason.SubsetReservation
				if reason == nil || len(reason.PremiseCells) != len(reason.ReservedDigits) || len(reason.PremiseCells) < 2 || len(reason.ActionTargets) == 0 {
					t.Fatalf("naked-subset reason does not preserve its complete cardinality and action: %+v", reason)
				}
			} else if key == "hidden-pair" || key == "hidden-triple" || key == "hidden-quad" {
				if plan.Reason == nil || plan.Reason.Kind != HintReasonHiddenSubsetConfinement || plan.Reason.HiddenSubsetConfinement == nil {
					t.Fatalf("hidden-subset contract = reason %+v, conclusion %+v", plan.Reason, plan.Steps[2].Message)
				}
				expectedMessage := "hint.hidden-subset-confinement.apply"
				if plan.Conclusion.Placement != nil {
					expectedMessage = "hint.hidden-subset-confinement.place"
				}
				if plan.Steps[0].Message.ID != "hint.hidden-subset-confinement.observe" || plan.Steps[1].Message.ID != "hint.hidden-subset-confinement.explain" || plan.Steps[2].Message.ID != expectedMessage {
					t.Fatalf("hidden-subset descriptors are incomplete: %+v", plan.Steps)
				}
				reason := plan.Reason.HiddenSubsetConfinement
				if len(reason.PremiseCells) != len(reason.ConfinedDigits) || len(reason.PremiseCells) < 2 || len(reason.ActionTargets) == 0 {
					t.Fatalf("hidden-subset reason does not preserve its complete cardinality and action: %+v", reason)
				}
			} else if key == "pointing-pair" {
				if plan.Reason == nil || plan.Reason.Kind != HintReasonIntersectionConfinement || plan.Reason.IntersectionConfinement == nil {
					t.Fatalf("intersection contract = reason %+v", plan.Reason)
				}
				expectedMessage := "hint.intersection-confinement.apply"
				if plan.Conclusion.Placement != nil {
					expectedMessage = "hint.intersection-confinement.place"
				}
				if plan.Steps[0].Message.ID != "hint.intersection-confinement.observe" || plan.Steps[1].Message.ID != "hint.intersection-confinement.explain" || plan.Steps[2].Message.ID != expectedMessage {
					t.Fatalf("intersection descriptors are incomplete: %+v", plan.Steps)
				}
				reason := plan.Reason.IntersectionConfinement
				if len(reason.PremiseCandidates) < 2 || len(reason.ActionTargets) == 0 || reason.SourceUnit == reason.IntersectingUnit {
					t.Fatalf("intersection reason is incomplete: %+v", reason)
				}
			} else if plan.Steps[2].Message.ID != "hint.strategy.conclude" || len(plan.Steps[2].Message.Arguments) != 1 {
				t.Fatalf("conclusion descriptor = %+v", plan.Steps[2].Message)
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
				if strings.Contains(step.Message.Text, plan.Strategy.Name.Text) {
					t.Fatalf("strategy vocabulary appears before the reasoning: %q", step.Message.Text)
				}
				if strings.Contains(step.Message.Text, "(") {
					t.Fatalf("family plan exposes coordinate shorthand: %q", step.Message.Text)
				}
			}
			if key == "x-wing" || key == "swordfish" || key == "jellyfish" {
				if plan.Reason == nil || plan.Reason.Kind != HintReasonFishConfinement || plan.Steps[2].Message.ID != "hint.fish-confinement.apply" {
					t.Fatalf("fish contract = reason %+v, conclusion %+v", plan.Reason, plan.Steps[2].Message)
				}
				expectedSize := map[string]int{"x-wing": 2, "swordfish": 3, "jellyfish": 4}[key]
				reason := plan.Reason.FishConfinement
				if reason == nil || len(reason.BaseUnits) != expectedSize || len(reason.CoverUnits) != expectedSize || len(reason.PatternCandidates) < expectedSize || len(reason.ActionTargets) == 0 {
					t.Fatalf("fish reason is incomplete: %+v", reason)
				}
			} else if plan.Steps[2].Message.ID != "hint.strategy.conclude" || len(plan.Steps[2].Message.Arguments) != 1 {
				t.Fatalf("conclusion descriptor = %+v", plan.Steps[2].Message)
			}
			switch key {
			case "xy-wing":
				if len(plan.Steps[0].Links) != 1 || len(plan.Steps[1].Links) != 2 || len(plan.Steps[2].Links) != 2 {
					t.Fatalf("XY-Wing implication scene is not progressive and persistent: %+v", plan.Steps)
				}
				for _, step := range plan.Steps {
					if len(step.Nodes) == 0 || step.Links[0].Kind != solver.LinkImplication {
						t.Fatalf("XY-Wing step lacks implication nodes and arrows: %+v", step)
					}
				}
			case "w-wing":
				for _, step := range plan.Steps {
					if len(step.Nodes) == 0 || len(step.Links) != 1 || step.Links[0].Kind != solver.LinkStrong {
						t.Fatalf("W-Wing step lacks its persistent strong-link scene: %+v", step)
					}
				}
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
				if strings.Contains(step.Message.Text, plan.Strategy.Name.Text) {
					t.Fatalf("strategy vocabulary appears before the reasoning: %q", step.Message.Text)
				}
				if strings.Contains(step.Message.Text, "(") {
					t.Fatalf("family plan exposes coordinate shorthand: %q", step.Message.Text)
				}
			}
			if plan.Steps[2].Message.ID != "hint.strategy.conclude" || len(plan.Steps[2].Message.Arguments) != 1 {
				t.Fatalf("conclusion descriptor = %+v", plan.Steps[2].Message)
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
			if plan == nil || plan.Strategy.ID != key {
				t.Fatalf("unexpected family plan: %+v", plan)
			}
			if key == "xy-chain" {
				if plan.Reason == nil || plan.Reason.ImplicationChain == nil || len(plan.Steps) != len(plan.Reason.ImplicationChain.Links)+2 {
					t.Fatalf("unexpected logic-sized XY-chain plan: %+v", plan)
				}
			} else if len(plan.Steps) != 3 {
				t.Fatalf("unexpected family plan: %+v", plan)
			}
			for _, step := range plan.Steps[:2] {
				if strings.Contains(step.Message.Text, plan.Strategy.Name.Text) {
					t.Fatalf("strategy vocabulary appears before the reasoning: %q", step.Message.Text)
				}
				if strings.Contains(step.Message.Text, "r1c") || strings.Contains(step.Message.Text, "(") {
					t.Fatalf("family plan exposes compact notation: %q", step.Message.Text)
				}
			}
			last := plan.Steps[len(plan.Steps)-1]
			if key == "xy-chain" {
				if plan.Reason == nil || plan.Reason.Kind != HintReasonImplicationChain || last.Message.ID != "hint.implication-chain.apply" {
					t.Fatalf("xy-chain pilot contract = reason %+v, conclusion %+v", plan.Reason, last.Message)
				}
			} else if last.Message.ID != "hint.strategy.conclude" || len(last.Message.Arguments) != 1 {
				t.Fatalf("conclusion descriptor = %+v", last.Message)
			}
			if len(plan.Steps[1].Links) <= len(plan.Steps[0].Links) {
				t.Fatalf("complex scene is not progressive: %+v", plan.Steps)
			}
			for _, mark := range plan.Steps[1].Marks {
				if mark.Target.Kind == HintTargetRow || mark.Target.Kind == HintTargetColumn || mark.Target.Kind == HintTargetBox {
					t.Fatalf("chain scene paints broad unit scope: %+v", mark)
				}
			}
			switch key {
			case "simple-coloring":
				if len(plan.Steps[1].Links) == 0 || plan.Steps[1].Nodes[0].Group == "" {
					t.Fatalf("coloring plan does not teach alternating groups: %+v", plan.Steps)
				}
			case "x-cycles":
				if len(plan.Steps[1].Links) < 3 || plan.Steps[1].Nodes[0].Order == 0 {
					t.Fatalf("cycle plan does not teach alternating links: %+v", plan.Steps)
				}
			case "xy-chain":
				for _, step := range plan.Steps {
					if len(step.Narration) == 0 {
						t.Fatalf("XY-chain step lacks structured narration: %+v", step)
					}
				}
				if len(plan.Steps[1].Links) == 0 || plan.Steps[1].Links[0].Kind != solver.LinkImplication {
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
	if len(steps) != 3 || steps[0].Marks[0].Target.Kind != HintTargetRow || steps[1].Marks[len(steps[1].Marks)-1].Target.Kind != HintTargetBox {
		t.Fatalf("unexpected box-line scene: %+v", steps)
	}
	if steps[0].Message.ID != "hint.intersection-confinement.observe" || steps[1].Message.ID != "hint.intersection-confinement.explain" {
		t.Fatalf("box-line relationship uses generic descriptors: %+v", steps)
	}
}
