package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/message"
	"github.com/gnailuy/sudoku/sessionfile"
	"github.com/gnailuy/sudoku/solver"
)

const controllerTestPuzzle = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."

func newTestController(t *testing.T) *Controller {
	t.Helper()
	board := core.NewEmptyBoard()
	board.FromString(controllerTestPuzzle)
	store := solver.NewStore()
	newGame := game.NewGame(board, game.NewDefaultOptions(store))
	return NewController(&newGame)
}

func TestParseDigitArguments(t *testing.T) {
	for _, input := range []string{"1 2 3", "123", "  1   2  3 "} {
		values, err := parseDigitArguments(input, 3)
		if err != nil || values[0] != 1 || values[1] != 2 || values[2] != 3 {
			t.Fatalf("parseDigitArguments(%q) = %v, %v", input, values, err)
		}
	}
	for _, input := range []string{"", "1 2", "1 2 3 4", "10 2 3", "a 2 3"} {
		if _, err := parseDigitArguments(input, 3); err == nil {
			t.Errorf("parseDigitArguments(%q) unexpectedly succeeded", input)
		}
	}
}

func TestCommandsRejectExtraArguments(t *testing.T) {
	controller := newTestController(t)
	if controller.RunCommand("undo now") {
		t.Fatal("undo with extra arguments changed the board")
	}
	if controller.game.Snapshot().CanUndo {
		t.Fatal("undo with extra arguments was applied")
	}
}

func TestHintExplainsAppliesAndUndoRestoresState(t *testing.T) {
	controller := newTestController(t)
	before := controller.game.Snapshot()
	plan := controller.game.Hint()
	if plan == nil || plan.Conclusion.Placement == nil {
		t.Fatal("test fixture did not produce a placement hint")
	}
	formatted := formatCLIHint(*plan)
	for _, marker := range []string{"Hint:", "Placement:"} {
		if !strings.Contains(formatted, marker) {
			t.Fatalf("formatted hint missing %q:\n%s", marker, formatted)
		}
	}
	for _, removed := range []string{"hint apply", "hint cancel"} {
		if strings.Contains(formatted, removed) {
			t.Fatalf("formatted hint retained %q:\n%s", removed, formatted)
		}
	}
	if !controller.RunCommand("hint") {
		t.Fatal("hint did not apply its exact plan")
	}
	after := controller.game.Snapshot()
	if after.Values == before.Values || !after.CanUndo {
		t.Fatal("hint did not create one undoable state change")
	}
	if !controller.RunCommand("undo") {
		t.Fatal("undo failed after hint")
	}
	restored := controller.game.Snapshot()
	if restored.Values != before.Values || restored.Candidates != before.Candidates {
		t.Fatal("undo did not restore the pre-hint state")
	}
}

func TestHintRejectsArgumentsWithoutMutation(t *testing.T) {
	controller := newTestController(t)
	before := controller.game.Snapshot()
	for _, command := range []string{"hint apply", "hint cancel"} {
		if controller.RunCommand(command) {
			t.Fatalf("%q changed the board", command)
		}
		if controller.game.Snapshot() != before {
			t.Fatalf("%q mutated state", command)
		}
	}
}

func TestUnavailableHintDoesNotMutateState(t *testing.T) {
	controller := newTestController(t)
	if _, err := controller.game.Apply(game.Solve{}); err != nil {
		t.Fatal(err)
	}
	before := controller.game.Snapshot()
	if controller.RunCommand("hint") {
		t.Fatal("unavailable hint reported a state change")
	}
	if controller.game.Snapshot() != before {
		t.Fatal("unavailable hint mutated state")
	}
}

func TestCandidateWorkStaysHiddenAndAppliesAtomically(t *testing.T) {
	controller := newTestController(t)
	initial := controller.game.Snapshot()
	if controller.RunCommand("note 1 1 1") || controller.game.Snapshot() != initial {
		t.Fatal("removed line-CLI note command changed the game")
	}
	position := core.NewPosition(0, 0)
	if _, err := controller.game.Apply(game.SetCandidates{Position: position, Values: []int{1}}); err != nil {
		t.Fatal(err)
	}
	before := controller.game.Snapshot()
	if strings.Contains(renderBoard(before), "+-----------+") {
		t.Fatal("line CLI rendered candidate cells instead of the compact value board")
	}
	plan := controller.game.Hint()
	if plan == nil || plan.Conclusion.Placement != nil {
		t.Fatal("test fixture did not produce candidate work")
	}
	formatted := formatCLIHint(*plan)
	for _, marker := range []string{"Hint:", "updates candidates", "sudoku tui"} {
		if !strings.Contains(formatted, marker) {
			t.Fatalf("candidate hint missing %q:\n%s", marker, formatted)
		}
	}
	if strings.Contains(formatted, "hint apply") || strings.Contains(formatted, "hint cancel") {
		t.Fatalf("candidate hint retained two-step instructions:\n%s", formatted)
	}
	if !controller.RunCommand("hint") || controller.game.Snapshot() == before {
		t.Fatal("candidate correction was not applied atomically")
	}
	if !controller.RunCommand("undo") {
		t.Fatal("candidate correction was not undoable")
	}
	restored := controller.game.Snapshot()
	if restored.Values != before.Values || restored.Candidates != before.Candidates {
		t.Fatal("undo did not restore candidate state")
	}
	if !controller.RunCommand("hint") {
		t.Fatal("candidate correction could not be applied again")
	}
	if !controller.RunCommand("hint") {
		t.Fatal("next hint did not continue from updated candidate state")
	}
	manual := game.HintPlan{Conclusion: game.HintConclusion{Eliminations: []solver.CandidateRef{{Position: core.NewPosition(0, 1), Value: 3}}}}
	if got := formatCLIHint(manual); !strings.Contains(got, candidateHintLimitation) {
		t.Fatalf("candidate hint text=%q", got)
	}
}

func TestSaveCommandRoundTrip(t *testing.T) {
	controller := newTestController(t)
	before := controller.game.Snapshot().Candidates[0][0].Has(5)
	values := []int{5}
	if before {
		values = nil
	}
	if _, err := controller.game.Apply(game.SetCandidates{Position: core.NewPosition(0, 0), Values: values}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "saved.json")
	if controller.RunCommand("save " + path) {
		t.Fatal("save should not report a board change")
	}
	data, err := sessionfile.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	store := solver.NewStore()
	restored, err := game.Restore(data, game.NewDefaultOptions(store))
	if err != nil {
		t.Fatal(err)
	}
	if restored.Snapshot().Candidates[0][0].Has(5) == before {
		t.Fatal("saved session did not preserve candidate edit")
	}
}

func TestFormatHintNarrationUsesCLICoordinates(t *testing.T) {
	position := core.NewPosition(3, 6)
	prefix := message.Descriptor{ID: "test.prefix", Arguments: []message.Argument{}, Text: "Look at "}
	suffix := message.Descriptor{ID: "test.suffix", Arguments: []message.Argument{}, Text: "."}
	step := game.HintStep{
		Message: message.Descriptor{ID: "test.fallback", Arguments: []message.Argument{}, Text: "fallback"},
		Narration: []game.HintNarrationChunk{
			{Kind: game.HintNarrationMessage, Message: &prefix},
			{Kind: game.HintNarrationTarget, Target: &game.HintTarget{Kind: game.HintTargetCell, Position: &position}},
			{Kind: game.HintNarrationMessage, Message: &suffix},
		},
	}
	if got := formatHintNarration(step, false); got != "Look at row 4, column 7." {
		t.Fatalf("CLI narration = %q", got)
	}
	if got := formatHintNarration(step, true); got != "Look at r4c7." {
		t.Fatalf("compact narration = %q", got)
	}
}

func TestFormatHintNarrationFallsBackForInvalidVariant(t *testing.T) {
	conflict := message.Descriptor{ID: "test.conflict", Arguments: []message.Argument{}, Text: "conflicting payload"}
	step := game.HintStep{
		Message: message.Descriptor{ID: "test.fallback", Arguments: []message.Argument{}, Text: "complete fallback"},
		Narration: []game.HintNarrationChunk{{
			Kind:    game.HintNarrationTarget,
			Message: &conflict,
		}},
	}
	if got := formatHintNarration(step, false); got != step.Message.Text {
		t.Fatalf("invalid narration = %q, want fallback %q", got, step.Message.Text)
	}
}
