package game

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gnailuy/sudoku/cli"
	"github.com/gnailuy/sudoku/core"
)

// Function to print the column numbers
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

// Function to print the Sudoku game
func (game *SudokuGame) print() {
	// Header column numbers
	fmt.Println()
	printColumnNumbers()

	// Board and row numbers
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

	// Footer column numbers
	printColumnNumbers()
	fmt.Println()
}

// Function to print the help message
func (game *SudokuGame) printHelp() {
	fmt.Println("Supported commands:")
	fmt.Println("  - help                        : Print this help message.")
	fmt.Println("  - add `row` `column` `value`  : Input a value to a cell at (row, column).")
	fmt.Println("  - clear `row` `column`        : Clear the value in a cell at (row, column).")
	fmt.Println("  - undo                        : Undo last move.")
	fmt.Println("  - redo                        : Redo last undo.")
	fmt.Println("  - repair                      : Undo all invalid inputs.")
	fmt.Println("  - reset                       : Reset the problem.")
	fmt.Println("  - check                       : Check if the current board is correct.")
	fmt.Println("  - solve                       : Solve the problem for me.")
	fmt.Println("  - quit                        : Quit the game.")
}

// Function to set a cell for the add and clear commands
func (game *SudokuGame) setValue(row, column, value int) bool {
	// Check user input validity
	positionPointer, err := core.NewPositionFromInput(row-1, column-1)

	if err != nil {
		fmt.Fprintln(os.Stderr, "[ERROR] Error in the input:", err)
		return false
	}

	// Skip adding if the input is the same as the current value
	if game.Get(*positionPointer) == value {
		return false
	}

	// Add the value to the cell
	err = game.AddInputAndRecordHistory(core.Cell{
		Position: *positionPointer,
		Value:    value,
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, "[ERROR] Error when adding an input value:", err)
		return false
	}

	return true
}

// Function to run a command
func (game *SudokuGame) runCommand(command string, closeChannel cli.CloseChannel) bool {
	commandFields := strings.SplitN(command, " ", 2)

	// Empty command, return directly
	if len(commandFields) == 0 || len(commandFields[0]) == 0 {
		return false
	}

	switch commandFields[0] {
	case "help":
		game.printHelp()
		return false
	case "add":
		if len(commandFields) != 2 {
			fmt.Fprintln(os.Stderr, "[ERROR] No argument specified for the add command.")
		} else {
			var row, column, value int
			_, err := fmt.Sscanf(commandFields[1], "%1d%1d%1d", &row, &column, &value)
			if err != nil {
				fmt.Fprintln(os.Stderr, "[ERROR] Error when reading the input command:", err)
			} else {
				return game.setValue(row, column, value)
			}
		}
	case "clear":
		if len(commandFields) != 2 {
			fmt.Fprintln(os.Stderr, "[ERROR] No argument specified for the clear command.")
		} else {
			var row, column int
			_, err := fmt.Sscanf(commandFields[1], "%1d%1d", &row, &column)
			if err != nil {
				fmt.Fprintln(os.Stderr, "[ERROR] Error when reading the input command:", err)
			} else {
				return game.setValue(row, column, 0)
			}
		}
	case "undo":
		err := game.Undo()
		return err == nil
	case "redo":
		err := game.Redo()
		return err == nil
	case "repair":
		return game.Repair() > 0
	case "check":
		if game.Invalid() {
			fmt.Println("The current board is correct.")
		} else {
			fmt.Println("You have entered incorrect values(s).")
		}
	case "reset":
		game.Reset()
		return true
	case "solve":
		game.Solve()
		return true
	case "quit":
		closeChannel.Close()
	default:
		fmt.Fprintln(os.Stderr, "[ERROR] Unknown command.")
	}

	return false
}

// Function to ask the user for input
func (game *SudokuGame) askUserInput(scanner *bufio.Scanner, inputChannel chan string, closeChannel cli.CloseChannel) {
	// Check if the close channel is closed
	if closeChannel.IsClosed() {
		return
	}

	// Print the problem
	game.print()

	// Ask for user input
	fmt.Println("Enter a command (Enter 'help' for help):")
	fmt.Print("> ")

	// Block until the user enters a command
	if scanner.Scan() {
		inputChannel <- strings.TrimSpace(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "[ERROR] Error when reading the input command:", err)
	}
}

// Function to start the game
func (game *SudokuGame) Play() {
	inputChannel := make(chan string)
	closeChannel := cli.NewCloseChannel()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		// Ask for user input in a goroutine, it will block until the user enters a command
		go game.askUserInput(scanner, inputChannel, closeChannel)

		// Block until we receive a command or the close channel is closed
		select {
		case command := <-inputChannel:
			game.runCommand(command, closeChannel)
		case <-closeChannel:
			fmt.Println("\nExiting the game. Bye!")
			os.Exit(0)
		}

		if game.IsSolved() {
			game.print()
			break
		}
	}

	fmt.Println("Congratulations! You have solved the problem.")
}
