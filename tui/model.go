// Package tui implements the opt-in Bubble Tea terminal frontend.
package tui

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
	usermessage "github.com/gnailuy/sudoku/message"
	"github.com/gnailuy/sudoku/playrun"
	"github.com/gnailuy/sudoku/recovery"
	"github.com/gnailuy/sudoku/sessionfile"
)

const (
	minWidth  = 72
	minHeight = 40
)

type inputMode uint8

const (
	valueMode inputMode = iota
	noteMode
)

type modalKind uint8

const (
	noModal modalKind = iota
	quitModal
	resetModal
	saveModal
	helpModal
	recoveryModal
)

// RecoveryChoice pairs validated recovery metadata with its restored game.
type RecoveryChoice struct {
	Record  recovery.Record
	Game    game.Game
	Tracker *playrun.Tracker
}

// RecoveryOptions configures autosave and optional startup recovery choices.
type RecoveryOptions struct {
	Store   *recovery.Store
	Choices []RecoveryChoice
	Warning string
}

type autosaveDueMsg struct{ generation uint64 }
type autosaveDoneMsg struct {
	generation uint64
	id         string
	err        error
}

// Model owns presentation state while delegating all puzzle transitions to game.Game.
type Model struct {
	game             *game.Game
	tracker          *playrun.Tracker
	snapshot         game.Snapshot
	row, column      int
	width, height    int
	mode             inputMode
	modal            modalKind
	theme            themeName
	message          string
	hint             *game.HintPlan
	hintStep         int
	dirty            bool
	autoCandidates   bool
	candidateVisible [9][9]bool
	savePath         string
	writeSession     func(string, []byte) error

	recoveryStore      *recovery.Store
	recoveryChoices    []RecoveryChoice
	recoverySelection  int
	recoveryID         string
	recoveryWarning    string
	recoveryGeneration uint64
	recoveryDue        uint64
	recoveryWriting    bool
	recoveryData       []byte
	quitAfterWrite     bool
}

// NewModel creates a TUI model without background recovery.
func NewModel(current game.Game, resumePath string) Model {
	return NewModelWithRecovery(current, resumePath, RecoveryOptions{})
}

// NewModelWithRecovery creates a TUI model with background recovery enabled.
func NewModelWithRecovery(current game.Game, resumePath string, options RecoveryOptions) Model {
	return NewTrackedModelWithRecovery(current, resumePath, options, nil)
}

// NewTrackedModelWithRecovery creates a TUI model with completion tracking.
func NewTrackedModelWithRecovery(current game.Game, resumePath string, options RecoveryOptions, tracker *playrun.Tracker) Model {
	model := Model{
		game:            &current,
		tracker:         tracker,
		snapshot:        current.Snapshot(),
		savePath:        resumePath,
		theme:           themeFromEnvironment(),
		writeSession:    sessionfile.Write,
		recoveryStore:   options.Store,
		recoveryChoices: options.Choices,
		recoveryWarning: options.Warning,
	}
	if len(model.recoveryChoices) > 0 {
		model.modal = recoveryModal
	}
	return model
}

func themeFromEnvironment() themeName {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return noColorTheme
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SUDOKU_THEME"))) {
	case "light":
		return lightTheme
	case "no-color", "nocolor", "none":
		return noColorTheme
	default:
		return darkTheme
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m *Model) setAllCandidateVisibility(visible bool) {
	m.autoCandidates = visible
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			m.candidateVisible[row][column] = visible && m.snapshot.Values[row][column] == 0
		}
	}
}

func (m *Model) revealCandidateCell(row, column int) {
	m.candidateVisible[row][column] = true
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := message.(type) {
	case autosaveDueMsg:
		if typed.generation != m.recoveryGeneration {
			return m, nil
		}
		m.recoveryDue = typed.generation
		if m.recoveryWriting {
			return m, nil
		}
		return m.startRecoveryWrite(typed.generation)
	case autosaveDoneMsg:
		m.recoveryWriting = false
		if typed.id != m.recoveryID && m.recoveryStore != nil {
			_ = m.recoveryStore.Delete(typed.id)
		}
		if m.quitAfterWrite {
			if m.recoveryStore != nil {
				_ = m.recoveryStore.Delete(typed.id)
			}
			return m, tea.Quit
		}
		if typed.err != nil {
			m.recoveryWarning = "Autosave failed: " + typed.err.Error()
			return m, nil
		}
		m.recoveryWarning = ""
		if m.recoveryDue > typed.generation {
			return m.startRecoveryWrite(m.recoveryDue)
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = typed.Width, typed.Height
		return m, nil
	case tea.KeyMsg:
		if m.width > 0 && (m.width < minWidth || m.height < minHeight) {
			if typed.String() == "q" || typed.String() == "esc" {
				return m, m.cleanupAndQuit()
			}
			if typed.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}
		if typed.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.modal != noModal {
			return m.updateModal(typed)
		}
		return m.updateBoard(typed)
	}
	return m, nil
}

func (m Model) updateBoard(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.message = ""
	var command tea.Cmd
	if m.hint != nil {
		switch key.String() {
		case "left", "h":
			if m.hintStep > 0 {
				m.hintStep--
			}
			m.message = hintStepMessage(*m.hint, m.hintStep)
			return m, nil
		case "right", "l":
			if m.hintStep+1 < len(m.hint.Steps) {
				m.hintStep++
			}
			m.message = hintStepMessage(*m.hint, m.hintStep)
			return m, nil
		case "enter":
			return m, m.apply(game.ApplyHint{PlanID: m.hint.PlanID})
		case "esc":
			m.hint = nil
			m.hintStep = 0
			m.message = "Hint preview canceled."
			return m, nil
		default:
			m.message = hintStepMessage(*m.hint, m.hintStep)
			return m, nil
		}
	}
	switch key.String() {
	case "up", "k":
		if m.row > 0 {
			m.row--
		}
	case "down", "j":
		if m.row < 8 {
			m.row++
		}
	case "left", "h":
		if m.column > 0 {
			m.column--
		}
	case "right", "l":
		if m.column < 8 {
			m.column++
		}
	case "n":
		if m.mode == valueMode {
			m.mode = noteMode
		} else {
			m.mode = valueMode
		}
	case "a":
		m.setAllCandidateVisibility(!m.autoCandidates)
		if m.autoCandidates {
			m.message = "Candidates shown."
		} else {
			m.message = "Candidates hidden."
		}
	case "u":
		command = m.apply(game.Undo{})
	case "r":
		command = m.apply(game.Redo{})
	case "?":
		m.modal = helpModal
	case "i":
		m.hint = m.game.Hint()
		m.hintStep = 0
		if m.hint == nil {
			m.message = "No hint is available."
		} else {
			m.setAllCandidateVisibility(true)
			m.message = hintStepMessage(*m.hint, m.hintStep)
		}
	case "enter":
		m.message = "Press i to preview a hint first."
	case "c":
		m.snapshot = m.game.Snapshot()
		m.message = "Board is " + string(m.snapshot.Status) + "."
	case "R":
		m.modal = resetModal
	case "S":
		m.modal = saveModal
	case "q", "esc":
		if m.dirty {
			m.modal = quitModal
		} else {
			return m, m.cleanupAndQuit()
		}
	case "0", "backspace", "delete":
		position := core.NewPosition(m.row, m.column)
		if m.mode == noteMode {
			m.revealCandidateCell(m.row, m.column)
			command = m.apply(game.SetCandidates{Position: position})
		} else {
			command = m.apply(game.ClearValue{Position: position})
		}
	default:
		if len(key.Runes) == 1 && key.Runes[0] >= '1' && key.Runes[0] <= '9' {
			value := int(key.Runes[0] - '0')
			position := core.NewPosition(m.row, m.column)
			if m.mode == noteMode {
				var candidates core.CandidateSet
				if m.candidateVisible[m.row][m.column] {
					candidates = m.snapshot.Candidates[m.row][m.column]
				} else {
					m.revealCandidateCell(m.row, m.column)
				}
				if candidates.Has(value) {
					candidates.Remove(value)
				} else {
					candidates.Add(value)
				}
				command = m.apply(game.SetCandidates{Position: position, Values: candidates.Values()})
			} else {
				command = m.apply(game.SetValue{Position: position, Value: value})
			}
		}
	}
	return m, command
}

func (m Model) updateModal(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.modal == recoveryModal {
		switch key.String() {
		case "up", "k":
			if m.recoverySelection > 0 {
				m.recoverySelection--
			}
		case "down", "j":
			if m.recoverySelection+1 < len(m.recoveryChoices) {
				m.recoverySelection++
			}
		case "enter":
			choice := m.recoveryChoices[m.recoverySelection]
			m.game = &choice.Game
			m.tracker = choice.Tracker
			m.snapshot = choice.Game.Snapshot()
			m.recoveryID = choice.Record.ID
			m.recoveryChoices = nil
			m.modal = noModal
			m.message = "Recovered game from " + choice.Record.UpdatedAt.Local().Format(time.RFC822)
		case "d":
			choice := m.recoveryChoices[m.recoverySelection]
			if err := m.recoveryStore.Delete(choice.Record.ID); err != nil {
				m.recoveryWarning = "Discard failed: " + err.Error()
				return m, nil
			}
			m.recoveryChoices = append(m.recoveryChoices[:m.recoverySelection], m.recoveryChoices[m.recoverySelection+1:]...)
			if len(m.recoveryChoices) == 0 {
				m.modal = noModal
			} else if m.recoverySelection == len(m.recoveryChoices) {
				m.recoverySelection--
			}
		case "n", "esc":
			m.recoveryChoices = nil
			m.modal = noModal
		case "q":
			return m, tea.Quit
		}
		return m, nil
	}
	if m.modal == helpModal {
		if key.String() == "?" || key.String() == "esc" {
			m.modal = noModal
		}
		return m, nil
	}
	if m.modal == saveModal {
		switch key.String() {
		case "esc":
			m.modal = noModal
		case "enter":
			if strings.TrimSpace(m.savePath) == "" {
				m.message = "Save path cannot be empty."
				return m, nil
			}
			data, err := m.game.Serialize()
			if err == nil {
				err = m.writeSession(m.savePath, data)
			}
			if err != nil {
				m.message = "Save failed: " + err.Error()
			} else {
				m.dirty = false
				m.message = "Saved to " + m.savePath
				if m.recoveryID != "" && m.recoveryStore != nil {
					if err := m.recoveryStore.Delete(m.recoveryID); err != nil {
						m.recoveryWarning = "Recovery cleanup failed: " + err.Error()
					} else {
						m.recoveryID = ""
					}
				}
			}
			m.modal = noModal
		case "backspace":
			if m.savePath != "" {
				_, size := utf8.DecodeLastRuneInString(m.savePath)
				m.savePath = m.savePath[:len(m.savePath)-size]
			}
		default:
			if len(key.Runes) > 0 {
				m.savePath += string(key.Runes)
			}
		}
		return m, nil
	}
	switch key.String() {
	case "y", "Y":
		if m.modal == quitModal {
			return m, m.cleanupAndQuit()
		}
		m.modal = noModal
		return m, m.apply(game.Reset{})
	case "n", "N", "esc":
		m.modal = noModal
	}
	return m, nil
}

func (m *Model) apply(action game.Action) tea.Cmd {
	var result game.Result
	var err error
	if m.tracker == nil {
		result, err = m.game.Apply(action)
	} else {
		result, err = m.tracker.Apply(m.game, action)
	}
	if err != nil {
		m.message = err.Error()
		return nil
	}
	m.snapshot = m.game.Snapshot()
	if _, reset := action.(game.Reset); reset {
		m.setAllCandidateVisibility(false)
		m.mode = valueMode
	}
	m.dirty = true
	m.hint = nil
	m.hintStep = 0
	m.message = fmt.Sprintf("Applied %s; board is %s.", result.Action, result.Status)
	if m.tracker != nil {
		if warning := m.tracker.TakeWarning(); warning != nil {
			m.recoveryWarning = "Warning: " + warning.Error()
		}
	}
	return m.scheduleRecovery()
}

func formatTUIHintNarration(step game.HintStep) string {
	if !step.HasValidNarration() {
		return step.Message
	}
	var result strings.Builder
	for _, chunk := range step.Narration {
		switch chunk.Kind {
		case game.HintNarrationText:
			result.WriteString(chunk.Text)
		case game.HintNarrationTarget:
			if chunk.Target == nil {
				continue
			}
			if chunk.Target.Position != nil {
				fmt.Fprintf(&result, "r%dc%d", chunk.Target.Position.Row+1, chunk.Target.Position.Column+1)
			} else {
				fmt.Fprintf(&result, "%s %d", chunk.Target.Kind, chunk.Target.Index+1)
			}
		case game.HintNarrationDigit:
			fmt.Fprint(&result, chunk.Value)
		}
	}
	return result.String()
}

func hintStepMessage(plan game.HintPlan, index int) string {
	if len(plan.Steps) == 0 {
		return "Hint preview: " + plan.Summary
	}
	step := plan.Steps[index]
	strategyName, gradeName := plan.Strategy.DisplayName, plan.Strategy.Grade
	if catalog, err := usermessage.NewEnglishCatalog(); err == nil {
		if descriptor, renderErr := usermessage.RenderEnglishName(catalog, "strategy", plan.Strategy.ID); renderErr == nil {
			strategyName = descriptor.Text
		}
		if descriptor, renderErr := usermessage.RenderEnglishName(catalog, "difficulty", plan.Strategy.Grade); renderErr == nil {
			gradeName = descriptor.Text
		}
	}
	return fmt.Sprintf("Hint — %s (%s)  •  Step %d of %d [%s]: %s", strategyName, gradeName, index+1, len(plan.Steps), step.Kind, formatTUIHintNarration(step))
}

func (m *Model) scheduleRecovery() tea.Cmd {
	if m.recoveryStore == nil {
		return nil
	}
	data, err := m.game.Serialize()
	if err != nil {
		m.recoveryWarning = "Autosave failed: " + err.Error()
		return nil
	}
	if m.recoveryID == "" {
		m.recoveryID, err = recovery.NewID()
		if err != nil {
			m.recoveryWarning = "Autosave failed: " + err.Error()
			return nil
		}
	}
	m.recoveryGeneration++
	generation := m.recoveryGeneration
	m.recoveryData = append(m.recoveryData[:0], data...)
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return autosaveDueMsg{generation: generation} })
}

func (m *Model) startRecoveryWrite(generation uint64) (tea.Model, tea.Cmd) {
	if m.recoveryStore == nil || m.recoveryID == "" {
		return *m, nil
	}
	m.recoveryWriting = true
	store, id := m.recoveryStore, m.recoveryID
	data := append([]byte(nil), m.recoveryData...)
	return *m, func() tea.Msg {
		return autosaveDoneMsg{generation: generation, id: id, err: store.Write(id, "TUI game", data)}
	}
}

func (m *Model) cleanupAndQuit() tea.Cmd {
	if m.recoveryWriting {
		m.quitAfterWrite = true
		return nil
	}
	if m.recoveryStore == nil || m.recoveryID == "" {
		return tea.Quit
	}
	store, id := m.recoveryStore, m.recoveryID
	return tea.Sequence(func() tea.Msg { _ = store.Delete(id); return nil }, tea.Quit)
}

func (m Model) View() string {
	if m.width > 0 && (m.width < minWidth || m.height < minHeight) {
		return fmt.Sprintf("Sudoku TUI\n\nTerminal too small: %dx%d. Resize to at least %dx%d.\n", m.width, m.height, minWidth, minHeight)
	}
	return render(m)
}
