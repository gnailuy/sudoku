package tui

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/recovery"
	"github.com/gnailuy/sudoku/solver"
)

const testPuzzle = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."

func testModel(t *testing.T) Model {
	t.Helper()
	board := core.NewEmptyBoard()
	board.FromString(testPuzzle)
	current := game.NewGame(board, game.NewDefaultOptions(solver.NewStore()))
	model := NewModel(current, "")
	model.width, model.height = 80, 42
	model.theme = darkTheme
	return model
}

func sendKey(t *testing.T, model Model, key tea.KeyMsg) Model {
	t.Helper()
	updated, _ := model.Update(key)
	return updated.(Model)
}

func TestNavigationAndValueNoteModes(t *testing.T) {
	model := testModel(t)
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRight})
	if model.column != 1 {
		t.Fatalf("column=%d", model.column)
	}
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	if model.snapshot.Values[0][1] != 5 || !model.dirty {
		t.Fatal("value was not applied")
	}
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	if !model.snapshot.Notes[1][1].Has(4) {
		t.Fatal("note was not applied")
	}
}

func TestAutomaticCandidateToggleIsPresentationOnly(t *testing.T) {
	model := testModel(t)
	before := model.snapshot
	beforeSerialized, err := model.game.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	afterSerialized, err := model.game.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !model.autoCandidates || model.dirty || model.snapshot != before || string(afterSerialized) != string(beforeSerialized) {
		t.Fatal("automatic-candidate toggle mutated session state")
	}
	view := ansi.Strip(model.View())
	cell := cellContent(model, 0, 0, 0) + cellContent(model, 0, 0, 1) + cellContent(model, 0, 0, 2)
	if !strings.Contains(view, "AUTO ON") || strings.TrimSpace(cell) == "" {
		t.Fatal("automatic candidates were not exposed in status and cells")
	}
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if model.autoCandidates || model.dirty || !strings.Contains(ansi.Strip(model.View()), "AUTO OFF") {
		t.Fatal("automatic candidates did not turn off cleanly")
	}
}

func TestStatusDisplaysAuthoritativeMistakeCount(t *testing.T) {
	model := testModel(t)
	if view := ansi.Strip(model.View()); !strings.Contains(view, "Mistakes: 0") {
		t.Fatalf("initial view omitted mistake count: %q", view)
	}

	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if !model.snapshot.Invalid[0][0] || model.snapshot.Mistakes != 1 {
		t.Fatalf("invalid input snapshot = %+v", model.snapshot)
	}
	if view := ansi.Strip(model.View()); !strings.Contains(view, "Mistakes: 1") {
		t.Fatalf("updated view omitted mistake count: %q", view)
	}

	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if view := ansi.Strip(model.View()); !strings.Contains(view, "Mistakes: 1") {
		t.Fatalf("undo incorrectly changed displayed mistake count: %q", view)
	}
}

func TestFirstNoteEditAdoptsAutomaticCandidatesAtomically(t *testing.T) {
	model := testModel(t)
	model.autoCandidates = true
	model.mode = noteMode
	target := core.NewPosition(0, 0)
	before := model.snapshot
	value := 1

	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune('0' + value)}})

	if model.autoCandidates || model.mode != noteMode {
		t.Fatal("candidate adoption did not enter manual-note editing")
	}
	if model.snapshot.Notes[target.Row][target.Column].Has(value) == before.Candidates[target.Row][target.Column].Has(value) {
		t.Fatal("candidate adoption did not apply the initiating note toggle")
	}
	peer := core.NewPosition(0, 1)
	if model.snapshot.Notes[peer.Row][peer.Column] != before.Candidates[peer.Row][peer.Column] {
		t.Fatal("candidate adoption did not copy the complete candidate grid")
	}
	if !model.dirty || !strings.Contains(model.message, "Candidates copied") {
		t.Fatalf("adoption dirty=%v message=%q", model.dirty, model.message)
	}

	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if model.snapshot.Notes != before.Notes {
		t.Fatal("one undo did not restore the complete prior note map")
	}
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if model.snapshot.Notes[target.Row][target.Column].Has(value) == before.Candidates[target.Row][target.Column].Has(value) {
		t.Fatal("one redo did not reapply candidate adoption")
	}
}

func TestAutomaticCandidatesRefreshAndCoexistWithNotes(t *testing.T) {
	model := testModel(t)
	model.autoCandidates = true
	noteTarget := core.NewPosition(0, 0)
	legalNote := model.snapshot.Candidates[noteTarget.Row][noteTarget.Column].Values()[0]
	model.snapshot.Notes[noteTarget.Row][noteTarget.Column].Add(legalNote)
	model.snapshot.Notes[noteTarget.Row][noteTarget.Column].Add(9)
	content := cellContent(model, noteTarget.Row, noteTarget.Column, 0) + cellContent(model, noteTarget.Row, noteTarget.Column, 1) + cellContent(model, noteTarget.Row, noteTarget.Column, 2)
	if !strings.Contains(content, string(rune('0'+legalNote))) || !strings.Contains(content, "9") {
		t.Fatalf("combined candidates omitted legal or stale manual note: %q", content)
	}

	target := core.Position{Row: -1, Column: -1}
	peer := core.Position{Row: -1, Column: -1}
	legal := 0
	for targetRow := 0; targetRow < 9 && !peer.IsValid(); targetRow++ {
		for targetColumn := 0; targetColumn < 9 && !peer.IsValid(); targetColumn++ {
			position := core.NewPosition(targetRow, targetColumn)
			for _, candidate := range model.snapshot.Candidates[targetRow][targetColumn].Values() {
				for row := 0; row < 9 && !peer.IsValid(); row++ {
					for column := 0; column < 9; column++ {
						possiblePeer := core.NewPosition(row, column)
						isPeer := row == targetRow || column == targetColumn || (row/3 == targetRow/3 && column/3 == targetColumn/3)
						if possiblePeer != position && isPeer && model.snapshot.Candidates[row][column].Has(candidate) {
							target, peer, legal = position, possiblePeer, candidate
							break
						}
					}
				}
			}
		}
	}
	if !peer.IsValid() {
		t.Fatal("test puzzle needs two peer cells sharing a candidate")
	}
	before := model.snapshot.Candidates[peer.Row][peer.Column]
	model.row, model.column = target.Row, target.Column
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune('0' + legal)}})
	if model.snapshot.Candidates[target.Row][target.Column] != 0 || model.snapshot.Candidates[peer.Row][peer.Column] == before {
		t.Fatal("candidate display did not refresh after a value action")
	}
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if model.snapshot.Candidates[peer.Row][peer.Column] != before {
		t.Fatal("candidate display did not refresh after undo")
	}
}

func TestUndoRedoAndQuitConfirmation(t *testing.T) {
	model := testModel(t)
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if model.snapshot.Values[0][0] != 0 {
		t.Fatal("undo failed")
	}
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if model.snapshot.Values[0][0] != 1 {
		t.Fatal("redo failed")
	}
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if model.modal != quitModal {
		t.Fatal("dirty quit did not ask for confirmation")
	}
}

func TestResizeFallbackAndCleanStableLayout(t *testing.T) {
	model := testModel(t)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	model = updated.(Model)
	if !strings.Contains(model.View(), "Terminal too small") {
		t.Fatal("missing resize fallback")
	}
	model.width, model.height = 80, 42
	view := ansi.Strip(model.View())
	for _, marker := range []string{"  3  ", "VALUE  •", "┏", "╋", "i hint", "? help"} {
		if !strings.Contains(view, marker) {
			t.Errorf("view missing %q", marker)
		}
	}
	for _, oldMarker := range []string{"<3>", "{", "}", "( . )", " . "} {
		if strings.Contains(view, oldMarker) {
			t.Errorf("view retained old cell marker %q", oldMarker)
		}
	}
}

func TestFocusStopsAtBoardEdges(t *testing.T) {
	model := testModel(t)
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyUp})
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyLeft})
	if model.row != 0 || model.column != 0 {
		t.Fatalf("focus escaped top-left: %d,%d", model.row, model.column)
	}
	model.row, model.column = 8, 8
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRight})
	if model.row != 8 || model.column != 8 {
		t.Fatalf("focus escaped bottom-right: %d,%d", model.row, model.column)
	}
}

func TestSaveUsesSerializedSessionAndClearsDirty(t *testing.T) {
	model := testModel(t)
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	model.modal, model.savePath = saveModal, "game.json"
	var path string
	model.writeSession = func(got string, data []byte) error {
		path = got
		if len(data) == 0 {
			t.Fatal("empty serialized session")
		}
		return nil
	}
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if path != "game.json" || model.dirty || model.modal != noModal {
		t.Fatalf("save state path=%q dirty=%v modal=%v", path, model.dirty, model.modal)
	}
}

func TestHintPreviewDoesNotMutateUntilEnter(t *testing.T) {
	model := testModel(t)
	before := model.snapshot
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if model.hint == nil || model.snapshot != before {
		t.Fatal("hint preview missing or mutated game")
	}
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.snapshot == before || !model.dirty {
		t.Fatal("hint was not applied")
	}
}

func TestHelpOverlayOpensAndCloses(t *testing.T) {
	model := testModel(t)
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if model.modal != helpModal || !strings.Contains(ansi.Strip(model.View()), "KEYBOARD HELP") {
		t.Fatal("help overlay did not open")
	}
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyEscape})
	if model.modal != noModal {
		t.Fatal("help overlay did not close")
	}
}

func TestSemanticBackgroundsUseDeterministicThemeColors(t *testing.T) {
	model := testModel(t)
	view := model.View()
	for _, pattern := range []string{`\x1b\[[0-9;]*48;2;117;91;24m`, `\x1b\[[0-9;]*48;2;27;48;56m`} {
		if !regexp.MustCompile(pattern).MatchString(view) {
			t.Fatalf("dark view missing background pattern %q", pattern)
		}
	}
	model.theme = lightTheme
	view = model.View()
	for _, pattern := range []string{`\x1b\[[0-9;]*48;2;246;195;83m`, `\x1b\[[0-9;]*48;2;232;242;243m`} {
		if !regexp.MustCompile(pattern).MatchString(view) {
			t.Fatalf("light view missing background pattern %q", pattern)
		}
	}
}

func TestDeterministicThemeSelectionAndNoColorFallback(t *testing.T) {
	t.Setenv("SUDOKU_THEME", "light")
	if got := themeFromEnvironment(); got != lightTheme {
		t.Fatalf("light theme=%v", got)
	}
	t.Setenv("NO_COLOR", "1")
	if got := themeFromEnvironment(); got != noColorTheme {
		t.Fatalf("NO_COLOR theme=%v", got)
	}

	model := testModel(t)
	model.theme = noColorTheme
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	view := model.View()
	for _, colorCode := range []string{"\x1b[30m", "\x1b[31m", "\x1b[32m", "\x1b[33m", "\x1b[34m", "\x1b[35m", "\x1b[36m", "\x1b[37m", "\x1b[38;", "\x1b[40m", "\x1b[41m", "\x1b[42m", "\x1b[43m", "\x1b[44m", "\x1b[45m", "\x1b[46m", "\x1b[47m", "\x1b[48;"} {
		if strings.Contains(view, colorCode) {
			t.Fatalf("no-color view contains %q", colorCode)
		}
	}
	if !regexp.MustCompile(`\x1b\[[0-9;]*7[0-9;]*m`).MatchString(view) {
		t.Fatal("no-color focus is not distinguished with reverse video")
	}
	if !regexp.MustCompile(`\x1b\[[0-9;]*4[0-9;]*m`).MatchString(view) {
		t.Fatal("no-color invalid value is not underlined")
	}
}

func TestFailedSaveKeepsSessionDirty(t *testing.T) {
	model := testModel(t)
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	model.modal, model.savePath = saveModal, "game.json"
	model.writeSession = func(string, []byte) error { return errors.New("disk full") }
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if !model.dirty || !strings.Contains(model.message, "disk full") {
		t.Fatalf("failed save state dirty=%v message=%q", model.dirty, model.message)
	}
}

func TestSmallTerminalIgnoresGameInput(t *testing.T) {
	model := testModel(t)
	model.width, model.height = 30, 10
	before := model.snapshot
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if model.snapshot != before || model.dirty {
		t.Fatal("small terminal accepted game input")
	}
}

func TestAutosaveDebouncesToNewestGeneration(t *testing.T) {
	store := recovery.NewStore(filepath.Join(t.TempDir(), "recovery"))
	model := testModel(t)
	model = NewModelWithRecovery(*model.game, "", RecoveryOptions{Store: &store})
	model.width, model.height = 80, 42

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	model = updated.(Model)
	firstGeneration := model.recoveryGeneration
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	model = updated.(Model)
	secondGeneration := model.recoveryGeneration
	if firstGeneration == secondGeneration || model.recoveryID == "" {
		t.Fatal("mutations did not create distinct recovery generations")
	}

	updated, command := model.Update(autosaveDueMsg{generation: firstGeneration})
	model = updated.(Model)
	if command != nil {
		t.Fatal("stale debounce wrote a recovery record")
	}
	updated, command = model.Update(autosaveDueMsg{generation: secondGeneration})
	model = updated.(Model)
	if command == nil {
		t.Fatal("latest debounce did not start a write")
	}
	updated, _ = model.Update(command())
	model = updated.(Model)
	if model.recoveryWarning != "" || model.recoveryWriting {
		t.Fatalf("autosave warning=%q writing=%v", model.recoveryWarning, model.recoveryWriting)
	}

	records, err := store.Discover(nil)
	if err != nil || len(records) != 1 {
		t.Fatalf("records=%v err=%v", records, err)
	}
	restored, err := game.Restore(records[0].Session, game.NewDefaultOptions(solver.NewStore()))
	if err != nil || restored.Snapshot().Values[0][0] != 2 {
		t.Fatalf("newest state was not recovered: err=%v", err)
	}
}

func TestRecoverySelectionAndExplicitSaveCleanup(t *testing.T) {
	store := recovery.NewStore(filepath.Join(t.TempDir(), "recovery"))
	recovered := testModel(t)
	result, err := recovered.game.Apply(game.SetValue{Position: core.NewPosition(0, 0), Value: 4})
	if err != nil || result.Action != game.ActionSetValue {
		t.Fatal(err)
	}
	id, _ := recovery.NewID()
	data, _ := recovered.game.Serialize()
	if err := store.Write(id, "Recovered test", data); err != nil {
		t.Fatal(err)
	}
	records, err := store.Discover(nil)
	if err != nil {
		t.Fatal(err)
	}

	fresh := testModel(t)
	model := NewModelWithRecovery(*fresh.game, "", RecoveryOptions{Store: &store, Choices: []RecoveryChoice{{Record: records[0], Game: *recovered.game}}})
	model.width, model.height = 80, 42
	if model.modal != recoveryModal {
		t.Fatal("recovery chooser did not open")
	}
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.recoveryID != id || model.snapshot.Values[0][0] != 4 {
		t.Fatal("selected recovery was not restored")
	}

	model.dirty, model.modal, model.savePath = true, saveModal, filepath.Join(t.TempDir(), "saved.json")
	model = sendKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.recoveryID != "" {
		t.Fatal("explicit save retained recovery ownership")
	}
	remaining, err := store.Discover(nil)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("recovery record remains: %v err=%v", remaining, err)
	}
}

func TestAutosaveFailureWarnsAndRetriesAfterNextMutation(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "recovery")
	if err := os.Symlink(target, directory); err != nil {
		t.Fatal(err)
	}
	store := recovery.NewStore(directory)
	model := testModel(t)
	model = NewModelWithRecovery(*model.game, "", RecoveryOptions{Store: &store})
	model.width, model.height = 80, 42

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	model = updated.(Model)
	updated, command := model.Update(autosaveDueMsg{generation: model.recoveryGeneration})
	model = updated.(Model)
	updated, _ = model.Update(command())
	model = updated.(Model)
	if model.recoveryWarning == "" {
		t.Fatal("autosave failure was not surfaced")
	}

	if err := os.Remove(directory); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	model = updated.(Model)
	updated, command = model.Update(autosaveDueMsg{generation: model.recoveryGeneration})
	model = updated.(Model)
	updated, _ = model.Update(command())
	model = updated.(Model)
	if model.recoveryWarning != "" {
		t.Fatalf("successful retry retained warning: %q", model.recoveryWarning)
	}
	records, err := store.Discover(nil)
	if err != nil || len(records) != 1 {
		t.Fatalf("retry records=%v err=%v", records, err)
	}
}
