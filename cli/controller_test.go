package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
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

func TestHintPreviewApplyAndCancel(t *testing.T) {
	controller := newTestController(t)
	before := controller.game.Snapshot()
	if controller.RunCommand("hint") {
		t.Fatal("hint preview reported a board change")
	}
	if controller.hintPlan == nil || controller.game.Snapshot() != before {
		t.Fatal("hint preview was not retained read-only")
	}
	formatted := formatPlacementHint(*controller.hintPlan)
	for _, marker := range []string{"Hint:", "Placement:", "hint apply", "hint cancel"} {
		if !strings.Contains(formatted, marker) {
			t.Fatalf("formatted hint missing %q:\n%s", marker, formatted)
		}
	}
	if !controller.RunCommand("hint apply") || controller.game.Snapshot() == before || controller.hintPlan != nil {
		t.Fatal("hint apply did not apply and consume the previewed plan")
	}
	if controller.RunCommand("hint apply") {
		t.Fatal("consumed hint plan was applied twice")
	}
	controller.RunCommand("hint")
	if controller.hintPlan == nil {
		t.Fatal("second hint preview missing")
	}
	if controller.RunCommand("hint cancel") || controller.hintPlan != nil {
		t.Fatal("hint cancel did not discard the preview without mutation")
	}
}

func TestStateChangeDiscardsCachedHintPlan(t *testing.T) {
	controller := newTestController(t)
	controller.RunCommand("hint")
	if controller.hintPlan == nil {
		t.Fatal("hint preview missing")
	}
	if !controller.RunCommand("add 1 1 4") {
		t.Fatal("test setup did not change the game")
	}
	if controller.hintPlan != nil {
		t.Fatal("state change did not discard the cached hint")
	}
	if controller.RunCommand("hint apply") {
		t.Fatal("discarded hint was applied")
	}
}

func TestNoOpAndFailedActionPreserveCachedHintPlan(t *testing.T) {
	controller := newTestController(t)
	controller.RunCommand("hint")
	if controller.RunCommand("clear 1 1") || controller.hintPlan == nil {
		t.Fatal("no-op clear discarded the cached hint")
	}
	if controller.RunCommand("add 1 3 4") || controller.hintPlan == nil {
		t.Fatal("failed immutable-cell action discarded the cached hint")
	}
}

func TestCandidateWorkRemainsHiddenAndUsesTUI(t *testing.T) {
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
	if controller.RunCommand("hint") {
		t.Fatal("candidate correction hint changed the game")
	}
	if controller.hintPlan != nil || controller.game.Snapshot() != before {
		t.Fatal("candidate correction hint was cached or mutated hidden state")
	}
	plan := game.HintPlan{Conclusion: game.HintConclusion{Eliminations: []solver.CandidateRef{{Position: core.NewPosition(0, 1), Value: 3}}}}
	if got := formatPlacementHint(plan); got != candidateHintLimitation+"\n" {
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
	step := game.HintStep{
		Message: "fallback",
		Narration: []game.HintNarrationChunk{
			{Kind: game.HintNarrationText, Text: "Look at "},
			{Kind: game.HintNarrationTarget, Target: &game.HintTarget{Kind: game.HintTargetCell, Position: &position}},
			{Kind: game.HintNarrationText, Text: "."},
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
	step := game.HintStep{
		Message: "complete fallback",
		Narration: []game.HintNarrationChunk{{
			Kind: game.HintNarrationTarget,
			Text: "conflicting payload",
		}},
	}
	if got := formatHintNarration(step, false); got != step.Message {
		t.Fatalf("invalid narration = %q, want fallback %q", got, step.Message)
	}
}
