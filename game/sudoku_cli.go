package game

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gnailuy/sudoku/cli"
	"github.com/gnailuy/sudoku/core"
)

// Function to print the Sudoku game
func (game *SudokuGame) print() {
	for i := 0; i < 9; i++ {
		if i%3 == 0 {
			fmt.Println("--------+-------+--------")
		}

		for j := 0; j < 9; j++ {
			cell := core.NewCell(i, j)
			number := game.Get(cell)

			if j%3 == 0 {
				fmt.Print("| ")
			}
			if number == 0 {
				fmt.Print(". ")
			} else {
				fmt.Printf("%d ", number)
			}
		}
		fmt.Println("|")
	}
	fmt.Println("--------+-------+--------")
}

// Function to print the help message
func (game *SudokuGame) printHelp() {
	fmt.Println("Supported commands:")
	fmt.Println("  - help                        : Print this help message.")
	fmt.Println("  - add `row` `column` `number` : Input a number to a call at (row, column).")
	fmt.Println("  - clear `row` `column`        : Clear the number in a call at (row, column).")
	fmt.Println("  - undo                        : Undo last move.")
	fmt.Println("  - redo                        : Redo last undo.")
	fmt.Println("  - reset                       : Reset the problem.")
	fmt.Println("  - check                       : Check if the current board is correct.")
	fmt.Println("  - solve                       : Solve the problem for me.")
	fmt.Println("  - quit                        : Quit the game.")
}

// Function to set a cell for the add and clear commands
func (game *SudokuGame) setNumber(row, column, number int) bool {
	// Check user input validity
	cellPointer, err := core.NewCellFromInput(row-1, column-1)

	if err != nil {
		fmt.Fprintln(os.Stderr, "[ERROR] Error in the input:", err)
		return false
	}

	// Skip adding if the input is the same as the current number
	if game.Get(*cellPointer) == number {
		return false
	}

	// Add the number to the cell
	err = game.AddInputAndRecordHistory(CellInput{
		Cell:   *cellPointer,
		Number: number,
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, "[ERROR] Error when adding a number:", err)
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
			var row, column, number int
			_, err := fmt.Sscanf(commandFields[1], "%1d%1d%1d", &row, &column, &number)
			if err != nil {
				fmt.Fprintln(os.Stderr, "[ERROR] Error when reading the input command:", err)
			} else {
				return game.setNumber(row, column, number)
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
				return game.setNumber(row, column, 0)
			}
		}
	case "undo":
		err := game.Undo()
		return err == nil
	case "redo":
		err := game.Redo()
		return err == nil
	case "check":
		if game.IsInvalid() {
			fmt.Println("You have entered incorrect number(s).")
		} else {
			fmt.Println("The current board is correct.")
		}
	case "reset":
		game.Reset()
		return true
	case "solve":
		game.Problem.Solve()
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
