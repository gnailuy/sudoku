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
}

// NewController creates a CLI controller for the given game.
func NewController(g *game.Game) *Controller { return NewTrackedController(g, nil) }

// NewTrackedController creates a controller that records play-run completion.
func NewTrackedController(g *game.Game, tracker *playrun.Tracker) *Controller {
	return &Controller{game: g, closeChannel: NewCloseChannel(), tracker: tracker}
}

func (ctrl *Controller) apply(action game.Action) (game.Result, error) {
	if ctrl.tracker == nil {
		return ctrl.game.Apply(action)
	}
	result, err := ctrl.tracker.Apply(ctrl.game, action)
	if warning := ctrl.tracker.TakeWarning(); warning != nil {
		printError("Warning:", warning)
	}
	return result, err
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
	fmt.Println("  - hint, i                       : Apply a hint for the next move.")
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
		result, err := ctrl.apply(game.ApplyHint{})
		if err != nil {
			printError("Failed to apply hint:", userFacingError(err))
			return false
		}
		if result.Hint != nil {
			fmt.Printf("Hint: %s\n", result.Hint.Reason)
		}
		return true
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
	case "help", "h", "check", "c", "undo", "u", "redo", "r", "repair", "f", "hint", "i", "solve", "s", "reset", "e", "quit", "q":
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
