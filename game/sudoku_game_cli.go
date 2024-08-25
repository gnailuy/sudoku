package game

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gnailuy/sudoku/cli"
	"github.com/gnailuy/sudoku/core"
)

// Function to print an error message with a prefix [ERROR].
func printError(message ...any) {
	fmt.Fprintln(os.Stderr, "[ERROR]", message)
}

// Function to print the column numbers.
func printColumnNumbers() {
	fmt.Print("    ")
	for i := 0; i < 9; i++ {
		if i%3 == 0 && i != 0 {
			fmt.Print("  ")
		}
		fmt.Printf(" %d", i+1)
	}
	fmt.Println()
}

// Function to print the Sudoku game.
func (game *SudokuGame) print() {
	// Header column numbers.
	fmt.Println()
	printColumnNumbers()

	// Board and row numbers.
	for i := 0; i < 9; i++ {
		if i%3 == 0 {
			fmt.Println("    -------+-------+-------")
		}

		fmt.Printf(" %d ", i+1)
		for j := 0; j < 9; j++ {
			position := core.NewPosition(i, j)
			value := game.Get(position)

			if j%3 == 0 {
				fmt.Print("| ")
			}
			if value == 0 {
				fmt.Print(". ")
			} else {
				fmt.Printf("%d ", value)
			}
		}
		fmt.Println("|", i+1)
	}
	fmt.Println("    -------+-------+-------")

	// Footer column numbers.
	printColumnNumbers()
	fmt.Println()
}

// Function to print the help message.
func (game *SudokuGame) printHelp() {
	fmt.Println("Supported commands:")
	fmt.Println("  - help, h                       : Print this help message.")
	fmt.Println("  - add, a <row> <column> <value> : Add the value to the cell at (row, column).")
	fmt.Println("  - clear, d <row> <column>       : Clear the value in a cell at (row, column).")
	fmt.Println("  - check, c                      : Check if the current board is correct.")
	fmt.Println("  - undo, u                       : Undo last move.")
	fmt.Println("  - redo, r                       : Redo last undo.")
	fmt.Println("  - repair, f                     : Undo all invalid inputs.")
	fmt.Println("  - hint, i                       : Apply a hint for the next move.")
	fmt.Println("  - solve, s                      : Solve the problem for me.")
	fmt.Println("  - reset, e                      : Reset the game and start over.")
	fmt.Println("  - quit, q                       : Quit the game.")
}

// Function to set a cell for the add and clear commands.
func (game *SudokuGame) setValue(rowInput, columnInput, valueInput int) (success bool, err error) {
	// Check user input validity.
	positionPointer, err := core.NewPositionFromInput(rowInput, columnInput)
	if err != nil {
		return false, fmt.Errorf("error in the input position: %w", err)
	}

	cellPointer, err := core.NewCellFromInput(*positionPointer, valueInput)
	if err != nil {
		return false, fmt.Errorf("error in the input value: %w", err)
	}

	// Skip adding if the input is the same as the current value.
	if game.Get(*positionPointer) == valueInput {
		return false, nil
	}

	// Add the value to the cell.
	err = game.AddInputAndRecordHistory(*cellPointer)
	success = err == nil
	return
}

// Function to handle the add command.
func (game *SudokuGame) runAddCommand(commandArguments string) (added bool, err error) {
	var row, column, value int
	_, err = fmt.Sscanf(commandArguments, "%1d%1d%1d", &row, &column, &value)
	if err != nil {
		return false, err
	} else {
		added, err = game.setValue(row, column, value)
		return
	}
}

// Function to handle the clear command.
func (game *SudokuGame) runClearCommand(commandArguments string) (cleared bool, err error) {
	var row, column int
	_, err = fmt.Sscanf(commandArguments, "%1d%1d", &row, &column)
	if err != nil {
		return false, err
	} else {
		cleared, err = game.setValue(row, column, 0)
		return
	}
}

// Function to handle the command with arguments.
func (game *SudokuGame) runCommandWithArguments(commandFields []string) (success bool, err error) {
	if len(commandFields) != 2 {
		return false, errors.New("no argument specified for the command")
	}

	switch commandFields[0] {
	case "add", "a":
		return game.runAddCommand(commandFields[1])
	case "clear", "d":
		return game.runClearCommand(commandFields[1])
	default:
		return false, fmt.Errorf("unsupported command: %s", commandFields[0])
	}
}

// Function to run a command.
func (game *SudokuGame) runCommand(command string, closeChannel cli.CloseChannel) bool {
	commandFields := strings.SplitN(command, " ", 2)

	// Empty command, return directly.
	if len(commandFields) == 0 || len(commandFields[0]) == 0 {
		return false
	}

	switch commandFields[0] {
	case "help", "h":
		game.printHelp()
		return false
	case "add", "a":
	case "clear", "d":
		success, err := game.runCommandWithArguments(commandFields)
		if err != nil {
			printError("Failed to run the", commandFields[0], "command:", err)
		}
		return success
	case "check", "c":
		if game.IsValid() {
			fmt.Println("The current board is correct.")
		} else {
			fmt.Println("You have entered incorrect values(s).")
		}
	case "undo", "u":
		err := game.Undo()
		return err == nil
	case "redo", "r":
		err := game.Redo()
		return err == nil
	case "repair", "f":
		return game.Repair() > 0
	case "hint", "i":
		hint := game.Hint()
		if hint != nil {
			added, err := game.setValue(hint.Position.Row+1, hint.Position.Column+1, hint.Value)
			if err != nil {
				printError("Failed to apply hint:", err)
			}
			if added {
				if hint.Value != 0 {
					fmt.Printf("Hint: Added %d to cell (%d, %d)\n", hint.Value, hint.Position.Row+1, hint.Position.Column+1)
				} else {
					fmt.Printf("Hint: Cleared cell (%d, %d)\n", hint.Position.Row+1, hint.Position.Column+1)
				}
			}
			return added
		}
		return false
	case "solve", "s":
		game.Solve()
		return true
	case "reset", "e":
		game.Reset()
		return true
	case "quit", "q":
		closeChannel.Close()
	default:
		// I find myself often forgetting to use the 'add' command and just typing the numbers directly.
		added, err := game.runAddCommand(command)
		if err != nil {
			printError("Failed to run the command:", err)
		}
		return added
	}

	return false
}

// Function to ask the user for input.
func (game *SudokuGame) askUserInput(scanner *bufio.Scanner, inputChannel chan string, closeChannel cli.CloseChannel) {
	// Check if the close channel is closed.
	if closeChannel.IsClosed() {
		return
	}

	// Print the problem.
	game.print()

	// Ask for user input.
	fmt.Println("Enter a command (Enter 'help' or 'h for help):")
	fmt.Print("> ")

	// Block until the user enters a command.
	if scanner.Scan() {
		inputChannel <- strings.TrimSpace(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		printError("Failed to read the input command:", err)
	}
}

// Function to start the game.
func (game *SudokuGame) PlayCli() {
	inputChannel := make(chan string)
	closeChannel := cli.NewCloseChannel()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		// Ask for user input in a goroutine, it will block until the user enters a command.
		go game.askUserInput(scanner, inputChannel, closeChannel)

		// Block until we receive a command or the close channel is closed.
		select {
		case command := <-inputChannel:
			game.runCommand(command, closeChannel)
		case <-closeChannel:
			fmt.Println("\nExiting the game:")
			fmt.Println(game.Problem.ToString())
			os.Exit(0)
		}

		if game.IsSolved() {
			game.print()
			break
		}
	}

	fmt.Println("Congratulations! You have solved the problem.")
}
