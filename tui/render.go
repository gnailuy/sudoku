package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gnailuy/sudoku/game"
	"github.com/muesli/termenv"
)

const (
	boardWidth  = 55
	boardHeight = 37
)

type themeName uint8

const (
	darkTheme themeName = iota
	lightTheme
	noColorTheme
)

type palette struct {
	name                               themeName
	background, text, muted, accent    lipgloss.Color
	given, player, invalid, digitMatch lipgloss.Color
	focusBackground, peerBackground    lipgloss.Color
	border, strongBorder, panelBorder  lipgloss.Color
}

type renderStyles struct {
	canvas                                         lipgloss.Style
	title, status, message, guide, modal           lipgloss.Style
	given, player, invalid, empty, note, candidate lipgloss.Style
	digitValue, digitCandidate, digitComplete      lipgloss.Style
	focus, peer, border, strongBorder              lipgloss.Style
	hintFocus, hintPremise, hintEliminated         lipgloss.Style
	hintConclusion                                 lipgloss.Style
}

func render(m Model) string {
	styles := stylesFor(m.theme)
	mode := "VALUE"
	if m.mode == noteMode {
		mode = "NOTES"
	}

	title := center(m.width, styles.title.Render("SUDOKU"), styles.canvas)
	status := fmt.Sprintf("%s  •  %s  •  Mistakes: %d  •  r%dc%d", mode, statusLabel(m.snapshot.Status), m.snapshot.Mistakes, m.row+1, m.column+1)
	if digit := activeDigit(m); digit != 0 {
		status += fmt.Sprintf("  •  Digit: %d", digit)
		if digitIsComplete(m, digit) {
			status += " all placed"
		}
	}
	if m.autoCandidates {
		status = "CANDIDATES ON  •  " + status
	} else {
		status = "CANDIDATES OFF  •  " + status
	}
	if m.dirty {
		status += "  •  unsaved"
	}

	mainContent := renderBoard(m, styles)
	if m.modal == helpModal {
		mainContent = place(boardWidth, boardHeight, renderHelp(styles), styles.canvas)
	} else if m.modal == recoveryModal {
		mainContent = place(boardWidth, boardHeight, renderRecovery(m, styles), styles.canvas)
	}
	parts := []string{
		title,
		center(m.width, styles.status.Render(status), styles.canvas),
		center(m.width, mainContent, styles.canvas),
	}
	displayMessage := m.message
	if m.recoveryWarning != "" {
		if displayMessage != "" {
			displayMessage += "  •  "
		}
		displayMessage += m.recoveryWarning
	}
	if displayMessage != "" {
		parts = append(parts, center(m.width, styles.message.Render(displayMessage), styles.canvas))
	} else {
		parts = append(parts, styles.canvas.Render(strings.Repeat(" ", m.width)))
	}

	switch m.modal {
	case quitModal:
		parts = append(parts, center(m.width, styles.modal.Render("Unsaved changes. Quit without saving?  y / N"), styles.canvas))
	case resetModal:
		parts = append(parts, center(m.width, styles.modal.Render("Reset the puzzle and clear history?  y / N"), styles.canvas))
	case saveModal:
		parts = append(parts, center(m.width, styles.modal.Render(fmt.Sprintf("Save session path: %s_\nEnter saves  •  Esc cancels", m.savePath)), styles.canvas))
	case helpModal, recoveryModal:
		parts = append(parts, styles.canvas.Render(strings.Repeat(" ", m.width)))
	default:
		guide := "arrows  •  1–9  •  n notes  •  a notes  •  i hint  •  ? help  •  S save  •  q quit"
		if m.hint != nil {
			guide = "←/h Back  •  →/l Next  •  Enter applies exact plan  •  Esc cancels"
		}
		parts = append(parts, center(m.width, styles.guide.Render(guide), styles.canvas))
	}
	return strings.Join(parts, "\n") + "\n"
}

func renderBoard(m Model, styles renderStyles) string {
	var out strings.Builder
	out.WriteString(boardRule(styles, '┏', '┯', '┳', '┓', '━'))
	out.WriteByte('\n')
	for row := 0; row < 9; row++ {
		for noteRow := 0; noteRow < 3; noteRow++ {
			out.WriteString(styles.strongBorder.Render("┃"))
			for column := 0; column < 9; column++ {
				out.WriteString(renderCell(m, styles, row, column, noteRow))
				if column == 8 {
					out.WriteString(styles.strongBorder.Render("┃"))
				} else if (column+1)%3 == 0 {
					out.WriteString(styles.strongBorder.Render("┃"))
				} else {
					out.WriteString(styles.border.Render("│"))
				}
			}
			out.WriteByte('\n')
		}
		if row == 8 {
			out.WriteString(boardRule(styles, '┗', '┷', '┻', '┛', '━'))
		} else if (row+1)%3 == 0 {
			out.WriteString(boardRule(styles, '┣', '┿', '╋', '┫', '━'))
		} else {
			out.WriteString(boardRule(styles, '┠', '┼', '╂', '┨', '─'))
		}
		if row != 8 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

func boardRule(styles renderStyles, left, minor, major, right, fill rune) string {
	var out strings.Builder
	out.WriteRune(left)
	for column := 0; column < 9; column++ {
		out.WriteString(strings.Repeat(string(fill), 5))
		if column == 8 {
			out.WriteRune(right)
		} else if (column+1)%3 == 0 {
			out.WriteRune(major)
		} else {
			out.WriteRune(minor)
		}
	}
	if fill == '━' {
		return styles.strongBorder.Render(out.String())
	}
	return styles.border.Render(out.String())
}

func renderCell(m Model, styles renderStyles, row, column, noteRow int) string {
	value := m.snapshot.Values[row][column]
	active := activeDigit(m)
	if value != 0 {
		var style lipgloss.Style
		switch {
		case m.snapshot.Invalid[row][column]:
			style = styles.invalid
		case m.snapshot.Givens[row][column] != 0:
			style = styles.given
		default:
			style = styles.player
		}
		if value == active && !m.snapshot.Invalid[row][column] {
			if digitIsComplete(m, value) {
				style = styles.digitComplete.Inherit(style)
			} else {
				style = styles.digitValue.Inherit(style)
			}
		}
		style = hintCellStyle(m, styles, row, column, style)
		return contextualStyle(m, styles, row, column, style).Render(cellContent(m, row, column, noteRow))
	}

	candidates := m.snapshot.Candidates[row][column]
	showCandidates := m.candidateVisible[row][column]
	if (!showCandidates || candidates.IsEmpty()) && !hasHintCandidate(m, row, column) {
		return contextualStyle(m, styles, row, column, hintCellStyle(m, styles, row, column, styles.empty)).Render("     ")
	}

	var out strings.Builder
	for offset := -1; offset <= 3; offset++ {
		style := styles.empty
		content := " "
		if offset >= 0 && offset < 3 {
			digit := noteRow*3 + offset + 1
			role, marked := hintCandidateRole(m, row, column, digit)
			switch {
			case marked && candidates.Has(digit):
				style = hintRoleStyle(styles, role).Inherit(style)
				content = string(rune('0' + digit))
			case showCandidates && candidates.Has(digit):
				if m.mode == noteMode {
					style = styles.note
				} else {
					style = styles.candidate
				}
				if digit == active {
					style = styles.digitCandidate.Inherit(style)
				}
				content = string(rune('0' + digit))
			}
		}
		out.WriteString(contextualStyle(m, styles, row, column, style).Render(content))
	}
	return out.String()
}

func activeDigit(m Model) int {
	if m.row < 0 || m.row >= 9 || m.column < 0 || m.column >= 9 {
		return 0
	}
	return m.snapshot.Values[m.row][m.column]
}

func digitIsComplete(m Model, digit int) bool {
	if digit < 1 || digit > 9 {
		return false
	}
	count := 0
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			if m.snapshot.Values[row][column] == digit && !m.snapshot.Invalid[row][column] {
				count++
			}
		}
	}
	return count == 9
}

func contextualStyle(m Model, styles renderStyles, row, column int, style lipgloss.Style) lipgloss.Style {
	if row == m.row && column == m.column {
		return styles.focus.Inherit(style)
	}
	if row == m.row || column == m.column || (row/3 == m.row/3 && column/3 == m.column/3) {
		return styles.peer.Inherit(style)
	}
	return style
}

func activeHintMarks(m Model) []game.HintMark {
	if m.hint == nil || m.hintStep < 0 || m.hintStep >= len(m.hint.Steps) {
		return nil
	}
	return m.hint.Steps[m.hintStep].Marks
}

func hasHintCandidate(m Model, row, column int) bool {
	for digit := 1; digit <= 9; digit++ {
		if _, ok := hintCandidateRole(m, row, column, digit); ok {
			return true
		}
	}
	return false
}

func hintCandidateRole(m Model, row, column, digit int) (game.HintRole, bool) {
	var selected game.HintRole
	for _, mark := range activeHintMarks(m) {
		target := mark.Target
		if target.Kind != game.HintTargetCandidate || target.Position == nil || target.Position.Row != row || target.Position.Column != column || target.Value != digit {
			continue
		}
		if hintRolePriority(mark.Role) >= hintRolePriority(selected) {
			selected = mark.Role
		}
	}
	return selected, selected != ""
}

func hintCellStyle(m Model, styles renderStyles, row, column int, base lipgloss.Style) lipgloss.Style {
	role := game.HintRole("")
	for _, mark := range activeHintMarks(m) {
		target := mark.Target
		matches := target.Kind == game.HintTargetCell && target.Position != nil && target.Position.Row == row && target.Position.Column == column
		matches = matches || target.Kind == game.HintTargetRow && target.Index == row
		matches = matches || target.Kind == game.HintTargetColumn && target.Index == column
		matches = matches || target.Kind == game.HintTargetBox && target.Index == (row/3)*3+column/3
		if matches && hintRolePriority(mark.Role) >= hintRolePriority(role) {
			role = mark.Role
		}
	}
	if role == "" {
		return base
	}
	return hintRoleStyle(styles, role).Inherit(base)
}

func hintRolePriority(role game.HintRole) int {
	switch role {
	case game.HintRoleConclusion:
		return 5
	case game.HintRoleProblem, game.HintRoleExtra:
		return 4
	case game.HintRoleEliminated:
		return 3
	case game.HintRolePremise:
		return 2
	case game.HintRoleFocus:
		return 1
	default:
		return 0
	}
}

func hintRoleStyle(styles renderStyles, role game.HintRole) lipgloss.Style {
	switch role {
	case game.HintRoleConclusion:
		return styles.hintConclusion
	case game.HintRoleEliminated:
		return styles.hintEliminated
	case game.HintRolePremise:
		return styles.hintPremise
	default:
		return styles.hintFocus
	}
}

func cellContent(m Model, row, column, noteRow int) string {
	value := m.snapshot.Values[row][column]
	if value != 0 {
		if noteRow == 1 {
			return fmt.Sprintf("  %d  ", value)
		}
		return "     "
	}

	candidates := m.snapshot.Candidates[row][column]
	showCandidates := m.candidateVisible[row][column]
	if !showCandidates || candidates.IsEmpty() {
		return "     "
	}
	content := [5]byte{' ', ' ', ' ', ' ', ' '}
	for offset := 0; offset < 3; offset++ {
		candidate := noteRow*3 + offset + 1
		if candidates.Has(candidate) {
			content[offset+1] = byte('0' + candidate)
		}
	}
	return string(content[:])
}

func renderRecovery(m Model, styles renderStyles) string {
	lines := []string{"RECOVER A GAME", "Enter resumes  •  ↑/↓ chooses  •  d discards  •  n starts new"}
	for index, choice := range m.recoveryChoices {
		marker := "  "
		if index == m.recoverySelection {
			marker = "> "
		}
		label := choice.Record.Source
		if label == "" {
			label = "Sudoku game"
		}
		lines = append(lines, fmt.Sprintf("%s%s — %s", marker, choice.Record.UpdatedAt.Local().Format("2006-01-02 15:04"), label))
	}
	return styles.modal.Render(strings.Join(lines, "\n"))
}

func renderHelp(styles renderStyles) string {
	return styles.modal.Render(strings.Join([]string{
		"KEYBOARD HELP  •  ? or Esc closes",
		"Move     arrows / h j k l",
		"Edit     1–9 set or note  •  0 clears  •  n toggles mode",
		"Display  a shows or hides every candidate cell",
		"Context  focus a value to highlight matches and visible candidates",
		"Game     i previews hint  •  ←/→ steps  •  Enter applies  •  Esc cancels",
		"Hint     bold focus  •  underlined premise  •  struck elimination  •  reversed conclusion",
		"Session  S saves  •  R resets  •  q quits",
	}, "\n"))
}

func place(width, height int, content string, canvas lipgloss.Style) string {
	centered := strings.Split(center(width, content, canvas), "\n")
	if len(centered) >= height {
		return strings.Join(centered, "\n")
	}
	blank := canvas.Render(strings.Repeat(" ", width))
	top := (height - len(centered)) / 2
	bottom := height - len(centered) - top
	lines := make([]string, 0, height)
	for range top {
		lines = append(lines, blank)
	}
	lines = append(lines, centered...)
	for range bottom {
		lines = append(lines, blank)
	}
	return strings.Join(lines, "\n")
}

func center(width int, content string, canvas lipgloss.Style) string {
	lines := strings.Split(content, "\n")
	for index, line := range lines {
		lineWidth := lipgloss.Width(line)
		if width <= lineWidth {
			continue
		}
		left := (width - lineWidth) / 2
		right := width - lineWidth - left
		lines[index] = canvas.Render(strings.Repeat(" ", left)) + line + canvas.Render(strings.Repeat(" ", right))
	}
	return strings.Join(lines, "\n")
}

func stylesFor(name themeName) renderStyles {
	p := paletteFor(name)
	renderer := lipgloss.NewRenderer(io.Discard)
	if name == noColorTheme {
		renderer.SetColorProfile(termenv.ANSI)
	} else {
		renderer.SetColorProfile(termenv.TrueColor)
	}
	base := renderer.NewStyle()
	if name != noColorTheme {
		base = base.Foreground(p.text).Background(p.background)
	}
	return renderStyles{
		canvas:         base,
		title:          base.Foreground(p.accent).Bold(true),
		status:         base.Foreground(p.muted),
		message:        base.Foreground(p.accent),
		guide:          base.Foreground(p.muted).Faint(name == noColorTheme),
		modal:          base.Border(lipgloss.RoundedBorder()).BorderForeground(p.panelBorder).Padding(0, 2),
		given:          base.Foreground(p.given).Bold(true),
		player:         base.Foreground(p.player),
		invalid:        base.Foreground(p.invalid).Bold(true).Underline(true),
		empty:          base,
		note:           base.Foreground(p.player).Bold(name == noColorTheme),
		candidate:      base.Foreground(p.muted).Faint(true),
		digitValue:     renderer.NewStyle().Foreground(p.digitMatch).Bold(true),
		digitCandidate: renderer.NewStyle().Foreground(p.digitMatch).Bold(true).Underline(true),
		digitComplete:  renderer.NewStyle().Foreground(p.digitMatch).Faint(true),
		focus:          renderer.NewStyle().Background(p.focusBackground).Bold(true).Reverse(name == noColorTheme),
		peer:           renderer.NewStyle().Background(p.peerBackground).Faint(name == noColorTheme),
		hintFocus:      renderer.NewStyle().Bold(true),
		hintPremise:    renderer.NewStyle().Underline(true),
		hintEliminated: renderer.NewStyle().Strikethrough(true),
		hintConclusion: renderer.NewStyle().Reverse(true).Bold(true),
		border:         base.Foreground(p.border),
		strongBorder:   base.Foreground(p.strongBorder).Bold(true),
	}
}

func paletteFor(name themeName) palette {
	switch name {
	case lightTheme:
		return palette{name: name, background: "#F7F4ED", text: "#25313C", muted: "#52606B", accent: "#087F8C", given: "#17202A", player: "#096B72", invalid: "#B42318", digitMatch: "#7A3E00", focusBackground: "#F6C453", peerBackground: "#E8F2F3", border: "#9AA6AF", strongBorder: "#40515F", panelBorder: "#087F8C"}
	case noColorTheme:
		return palette{name: name}
	default:
		return palette{name: name, background: "#101820", text: "#E7EDF2", muted: "#81909D", accent: "#63D2D5", given: "#F4F7FA", player: "#63D2D5", invalid: "#FF6B6B", digitMatch: "#FFD166", focusBackground: "#755B18", peerBackground: "#1B3038", border: "#40515F", strongBorder: "#AAB7C2", panelBorder: "#63D2D5"}
	}
}
