package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/playrun"
	"github.com/gnailuy/sudoku/sessionfile"
)

// Controller owns all terminal I/O for the Sudoku game.
// It holds a Game and translates user commands into Game API calls.
type Controller struct {
	game         *game.Game
	closeChannel CloseChannel
	tracker      *playrun.Tracker
	hintPlan     *game.HintPlan
}

// NewController creates a CLI controller for the given game.
func NewController(g *game.Game) *Controller { return NewTrackedController(g, nil) }

// NewTrackedController creates a controller that records play-run completion.
func NewTrackedController(g *game.Game, tracker *playrun.Tracker) *Controller {
	return &Controller{game: g, closeChannel: NewCloseChannel(), tracker: tracker}
}

func (ctrl *Controller) apply(action game.Action) (game.Result, error) {
	before := ctrl.game.Snapshot()
	var result game.Result
	var err error
	if ctrl.tracker == nil {
		result, err = ctrl.game.Apply(action)
	} else {
		result, err = ctrl.tracker.Apply(ctrl.game, action)
		if warning := ctrl.tracker.TakeWarning(); warning != nil {
			printError("Warning:", warning)
		}
	}
	if err == nil {
		ctrl.discardHintAfterMutation(action, before)
	}
	return result, err
}

func (ctrl *Controller) discardHintAfterMutation(action game.Action, before game.Snapshot) {
	if ctrl.hintPlan == nil {
		return
	}
	if _, applyingHint := action.(game.ApplyHint); applyingHint {
		return
	}
	if ctrl.game.Snapshot() == before {
		return
	}
	ctrl.hintPlan = nil
	fmt.Println("Hint preview discarded because the board changed.")
}

// printError prints an error message with a prefix [ERROR].
func printError(message ...any) {
	fmt.Fprintln(os.Stderr, append([]any{"[ERROR]"}, message...)...)
}

func writeColumnNumbers(result *strings.Builder) {
	result.WriteString("    ")
	for i := 0; i < 9; i++ {
		if i%3 == 0 && i != 0 {
			result.WriteString("  ")
		}
		fmt.Fprintf(result, " %d", i+1)
	}
	result.WriteByte('\n')
}

// PrintBoard renders the 9×9 Sudoku grid with row/column numbers.
func (ctrl *Controller) PrintBoard() { fmt.Print(renderBoard(ctrl.game.Snapshot())) }

func renderBoard(snapshot game.Snapshot) string {
	if snapshotHasNotes(snapshot) {
		return renderNoteBoard(snapshot)
	}
	var result strings.Builder
	result.WriteByte('\n')
	writeColumnNumbers(&result)
	for row := 0; row < 9; row++ {
		if row%3 == 0 {
			result.WriteString("    -------+-------+-------\n")
		}
		fmt.Fprintf(&result, " %d ", row+1)
		for column := 0; column < 9; column++ {
			if column%3 == 0 {
				result.WriteString("| ")
			}
			value := snapshot.Values[row][column]
			if value == 0 {
				result.WriteString(". ")
			} else {
				fmt.Fprintf(&result, "%d ", value)
			}
		}
		fmt.Fprintf(&result, "| %d\n", row+1)
	}
	result.WriteString("    -------+-------+-------\n")
	writeColumnNumbers(&result)
	result.WriteByte('\n')
	return result.String()
}

func snapshotHasNotes(snapshot game.Snapshot) bool {
	for row := range snapshot.Notes {
		for column := range snapshot.Notes[row] {
			if !snapshot.Notes[row][column].IsEmpty() {
				return true
			}
		}
	}
	return false
}

func renderNoteBoard(snapshot game.Snapshot) string {
	var result strings.Builder
	result.WriteString("\n        1   2   3     4   5   6     7   8   9\n")
	separator := "    +-----------+-----------+-----------+\n"
	for row := 0; row < 9; row++ {
		if row%3 == 0 {
			result.WriteString(separator)
		}
		for candidateRow := 0; candidateRow < 3; candidateRow++ {
			if candidateRow == 1 {
				fmt.Fprintf(&result, " %d  |", row+1)
			} else {
				result.WriteString("    |")
			}
			for column := 0; column < 9; column++ {
				if column > 0 && column%3 == 0 {
					result.WriteByte('|')
				}
				result.WriteString(noteCellLine(snapshot, row, column, candidateRow))
				if column%3 != 2 {
					result.WriteByte(' ')
				}
			}
			if candidateRow == 1 {
				fmt.Fprintf(&result, "|  %d\n", row+1)
			} else {
				result.WriteString("|\n")
			}
		}
	}
	result.WriteString(separator)
	result.WriteString("        1   2   3     4   5   6     7   8   9\n\n")
	return result.String()
}

func noteCellLine(snapshot game.Snapshot, row, column, candidateRow int) string {
	if value := snapshot.Values[row][column]; value != 0 {
		if candidateRow == 1 {
			return fmt.Sprintf(" %d ", value)
		}
		return "   "
	}
	result := [3]byte{' ', ' ', ' '}
	for candidateColumn := 0; candidateColumn < 3; candidateColumn++ {
		candidate := candidateRow*3 + candidateColumn + 1
		if snapshot.Notes[row][column].Has(candidate) {
			result[candidateColumn] = byte('0' + candidate)
		}
	}
	return string(result[:])
}

// PrintHelp displays the available commands.
func (ctrl *Controller) PrintHelp() {
	fmt.Println("Supported commands:")
	fmt.Println("  - help, h                        : Print this help message.")
	fmt.Println("  - add, a <row> <column> <value> : Add the value to the cell at (row, column).")
	fmt.Println("  - clear, d <row> <column>       : Clear the value in a cell at (row, column).")
	fmt.Println("  - note, n <row> <column> <value>: Toggle a note in an empty cell.")
	fmt.Println("  - notes-clear, x <row> <column> : Clear all notes in an empty cell.")
	fmt.Println("  - save <path>                   : Save the complete session atomically.")
	fmt.Println("  - check, c                      : Check if the current board is correct.")
	fmt.Println("  - undo, u                       : Undo last move.")
	fmt.Println("  - redo, r                       : Redo last undo.")
	fmt.Println("  - repair, f                     : Undo all invalid inputs.")
	fmt.Println("  - hint, i                       : Preview a teaching hint without changing the game.")
	fmt.Println("  - hint apply                   : Apply the exact previewed hint as one action.")
	fmt.Println("  - hint cancel                  : Discard the previewed hint.")
	fmt.Println("  - solve, s                      : Solve the problem for me.")
	fmt.Println("  - reset, e                      : Reset the game and start over.")
	fmt.Println("  - quit, q                       : Quit the game.")
}

func (ctrl *Controller) setValue(rowInput, columnInput, valueInput int) (bool, error) {
	position, err := core.NewPositionFromInput(rowInput, columnInput)
	if err != nil {
		return false, fmt.Errorf("error in the input position: %w", err)
	}
	if valueInput < 0 || valueInput > 9 {
		return false, errors.New("error in the input value: expected a digit from 0 through 9")
	}
	if ctrl.game.Snapshot().Values[position.Row][position.Column] == valueInput {
		return false, nil
	}
	var action game.Action = game.ClearValue{Position: *position}
	if valueInput != 0 {
		action = game.SetValue{Position: *position, Value: valueInput}
	}
	_, err = ctrl.apply(action)
	return err == nil, err
}

func parseDigitArguments(input string, count int) ([]int, error) {
	fields := strings.Fields(input)
	if len(fields) == 1 && len(fields[0]) == count {
		fields = strings.Split(fields[0], "")
	}
	if len(fields) != count {
		return nil, fmt.Errorf("expected %d single-digit arguments", count)
	}
	values := make([]int, count)
	for index, field := range fields {
		if len(field) != 1 {
			return nil, fmt.Errorf("argument %d must be a single digit", index+1)
		}
		value, err := strconv.Atoi(field)
		if err != nil {
			return nil, fmt.Errorf("argument %d must be a digit", index+1)
		}
		values[index] = value
	}
	return values, nil
}

func (ctrl *Controller) runAddCommand(arguments string) (bool, error) {
	values, err := parseDigitArguments(arguments, 3)
	if err != nil {
		return false, err
	}
	return ctrl.setValue(values[0], values[1], values[2])
}

func (ctrl *Controller) runClearCommand(arguments string) (bool, error) {
	values, err := parseDigitArguments(arguments, 2)
	if err != nil {
		return false, err
	}
	return ctrl.setValue(values[0], values[1], 0)
}

func (ctrl *Controller) runNoteCommand(arguments string) (bool, error) {
	values, err := parseDigitArguments(arguments, 3)
	if err != nil {
		return false, err
	}
	position, err := core.NewPositionFromInput(values[0], values[1])
	if err != nil {
		return false, fmt.Errorf("error in the input position: %w", err)
	}
	notes := ctrl.game.Snapshot().Notes[position.Row][position.Column]
	if notes.Has(values[2]) {
		notes.Remove(values[2])
	} else {
		notes.Add(values[2])
	}
	_, err = ctrl.apply(game.SetNotes{Position: *position, Values: notes.Values()})
	return err == nil, err
}

func (ctrl *Controller) runClearNotesCommand(arguments string) (bool, error) {
	values, err := parseDigitArguments(arguments, 2)
	if err != nil {
		return false, err
	}
	position, err := core.NewPositionFromInput(values[0], values[1])
	if err != nil {
		return false, fmt.Errorf("error in the input position: %w", err)
	}
	_, err = ctrl.apply(game.SetNotes{Position: *position})
	return err == nil, err
}

func (ctrl *Controller) runSaveCommand(path string) (bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return false, errors.New("expected a session file path")
	}
	data, err := ctrl.game.Serialize()
	if err != nil {
		return false, fmt.Errorf("serialize session: %w", err)
	}
	if err := sessionfile.Write(path, data); err != nil {
		return false, err
	}
	fmt.Printf("Session saved to %s.\n", path)
	return false, nil
}

func formatNarrationTarget(target game.HintTarget, compact bool) string {
	if target.Position != nil {
		if compact {
			return fmt.Sprintf("r%dc%d", target.Position.Row+1, target.Position.Column+1)
		}
		if target.Kind == game.HintTargetCandidate {
			return fmt.Sprintf("candidate %d at row %d, column %d", target.Value, target.Position.Row+1, target.Position.Column+1)
		}
		return fmt.Sprintf("row %d, column %d", target.Position.Row+1, target.Position.Column+1)
	}
	return fmt.Sprintf("%s %d", target.Kind, target.Index+1)
}

func formatHintNarration(step game.HintStep, compact bool) string {
	if !step.HasValidNarration() {
		return step.Message
	}
	var result strings.Builder
	for _, chunk := range step.Narration {
		switch chunk.Kind {
		case game.HintNarrationText:
			result.WriteString(chunk.Text)
		case game.HintNarrationTarget:
			if chunk.Target != nil {
				result.WriteString(formatNarrationTarget(*chunk.Target, compact))
			}
		case game.HintNarrationDigit:
			fmt.Fprint(&result, chunk.Value)
		}
	}
	return result.String()
}

func formatHintPlan(plan game.HintPlan) string {
	var result strings.Builder
	result.WriteString("Hint preview\n")
	fmt.Fprintf(&result, "Strategy: %s (%s)\n", plan.Strategy.DisplayName, plan.Strategy.Grade)
	fmt.Fprintf(&result, "Summary: %s\n", plan.Summary)
	result.WriteString("Steps:\n")
	for index, step := range plan.Steps {
		fmt.Fprintf(&result, "  %d. [%s] %s\n", index+1, step.Kind, formatHintNarration(step, false))
	}
	fmt.Fprintf(&result, "Conclusion: %s\n", hintConclusionText(plan))
	result.WriteString("Run `hint apply` to accept this plan or `hint cancel` to discard it.\n")
	return result.String()
}

func hintConclusionText(plan game.HintPlan) string {
	if placement := plan.Conclusion.Placement; placement != nil {
		return fmt.Sprintf("Place %d at r%dc%d.", placement.Value, placement.Position.Row+1, placement.Position.Column+1)
	}
	parts := make([]string, len(plan.Conclusion.Eliminations))
	for index, elimination := range plan.Conclusion.Eliminations {
		parts[index] = fmt.Sprintf("%d from r%dc%d", elimination.Value, elimination.Position.Row+1, elimination.Position.Column+1)
	}
	return "Remove candidate " + strings.Join(parts, ", ") + "."
}

func (ctrl *Controller) runHintCommand(arguments string) bool {
	switch strings.TrimSpace(arguments) {
	case "":
		ctrl.hintPlan = ctrl.game.Hint()
		if ctrl.hintPlan == nil {
			printError("Failed to preview hint:", "no hint is available")
			return false
		}
		fmt.Print(formatHintPlan(*ctrl.hintPlan))
		return false
	case "apply":
		if ctrl.hintPlan == nil {
			printError("Failed to apply hint:", "preview a hint first")
			return false
		}
		result, err := ctrl.apply(game.ApplyHint{PlanID: ctrl.hintPlan.PlanID})
		if err != nil {
			printError("Failed to apply hint:", userFacingError(err))
			return false
		}
		ctrl.hintPlan = nil
		if result.Hint != nil {
			fmt.Printf("Applied hint: %s\n", result.Hint.Summary)
		}
		return true
	case "cancel":
		ctrl.hintPlan = nil
		fmt.Println("Hint preview canceled.")
		return false
	default:
		printError("Failed to run the hint command: expected `hint`, `hint apply`, or `hint cancel`")
		return false
	}
}

func (ctrl *Controller) runCommandWithArguments(name, arguments string) (bool, error) {
	if arguments == "" {
		return false, errors.New("no argument specified for the command")
	}
	switch name {
	case "add", "a":
		return ctrl.runAddCommand(arguments)
	case "clear", "d":
		return ctrl.runClearCommand(arguments)
	case "note", "n":
		return ctrl.runNoteCommand(arguments)
	case "notes-clear", "x":
		return ctrl.runClearNotesCommand(arguments)
	case "save":
		return ctrl.runSaveCommand(arguments)
	default:
		return false, fmt.Errorf("unsupported command: %s", name)
	}
}

// RunCommand parses and dispatches a single command.
func (ctrl *Controller) RunCommand(command string) bool {
	command = strings.TrimSpace(command)
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	name := fields[0]
	arguments := strings.TrimSpace(strings.TrimPrefix(command, name))
	if arguments != "" && commandTakesNoArguments(name) {
		printError("Failed to run the", name, "command: unexpected arguments")
		return false
	}

	switch name {
	case "help", "h":
		ctrl.PrintHelp()
	case "add", "a", "clear", "d", "note", "n", "notes-clear", "x", "save":
		success, err := ctrl.runCommandWithArguments(name, arguments)
		if err != nil {
			printError("Failed to run the", name, "command:", userFacingError(err))
		}
		return success
	case "check", "c":
		if ctrl.game.Snapshot().Status != game.StatusInvalid {
			fmt.Println("The current board is correct.")
		} else {
			fmt.Println("You have entered incorrect value(s).")
		}
	case "undo", "u":
		_, err := ctrl.apply(game.Undo{})
		return err == nil
	case "redo", "r":
		_, err := ctrl.apply(game.Redo{})
		return err == nil
	case "repair", "f":
		_, err := ctrl.apply(game.Repair{})
		return err == nil
	case "hint", "i":
		return ctrl.runHintCommand(arguments)
	case "solve", "s":
		_, err := ctrl.apply(game.Solve{})
		return err == nil
	case "reset", "e":
		_, err := ctrl.apply(game.Reset{})
		return err == nil
	case "quit", "q":
		ctrl.closeChannel.Close()
	default:
		added, err := ctrl.runAddCommand(command)
		if err != nil {
			printError("Failed to run the command:", userFacingError(err))
		}
		return added
	}
	return false
}

func commandTakesNoArguments(name string) bool {
	switch name {
	case "help", "h", "check", "c", "undo", "u", "redo", "r", "repair", "f", "solve", "s", "reset", "e", "quit", "q":
		return true
	default:
		return false
	}
}

func userFacingError(err error) string {
	var engineError *game.EngineError
	if errors.As(err, &engineError) {
		switch engineError.Code {
		case game.ErrorInvalidCell:
			return "row, column, and value must be digits from 1 through 9"
		case game.ErrorImmutableCell:
			return "that cell is part of the puzzle"
		case game.ErrorNoteNotAllowed:
			return "notes are allowed only in empty cells"
		case game.ErrorNoUndo:
			return "there is no action to undo"
		case game.ErrorNoRedo:
			return "there is no action to redo"
		case game.ErrorNoHint:
			return "no hint is available"
		case game.ErrorStaleHint:
			return "the previewed hint is stale or already consumed; preview a new hint"
		}
	}
	return err.Error()
}

func (ctrl *Controller) askUserInput(scanner *bufio.Scanner, inputChannel chan string) {
	if ctrl.closeChannel.IsClosed() {
		return
	}
	ctrl.PrintBoard()
	fmt.Println("Enter a command (Enter 'help' or 'h' for help):")
	fmt.Print("> ")
	if scanner.Scan() {
		inputChannel <- strings.TrimSpace(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		printError("Failed to read the input command:", err)
	}
}

// Play runs the main interactive game loop.
func (ctrl *Controller) Play() {
	inputChannel := make(chan string)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		go ctrl.askUserInput(scanner, inputChannel)
		select {
		case command := <-inputChannel:
			ctrl.RunCommand(command)
		case <-ctrl.closeChannel:
			fmt.Println("\nExiting the game.")
			fmt.Println(ctrl.game.Snapshot().String())
			return
		}
		if ctrl.game.Snapshot().Status == game.StatusSolved {
			ctrl.PrintBoard()
			break
		}
	}
	fmt.Println("Congratulations! You have solved the problem.")
	fmt.Println(ctrl.game.Snapshot().String())
}
