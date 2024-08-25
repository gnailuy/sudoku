package game

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gnailuy/sudoku/core"
)

// Function to print the Sudoku game
func (board *SudokuGame) print() {
	for i := 0; i < 9; i++ {
		if i%3 == 0 {
			fmt.Println("--------+-------+--------")
		}

		for j := 0; j < 9; j++ {
			cell := core.NewCell(i, j)
			number := board.Problem.Get(cell)
			if number == 0 {
				number = board.invalidInput.Get(cell)
			}

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

// Print help function
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

// Function for the add and clear commands
func (game *SudokuGame) runAddCommand(row, column, number int) bool {
	err := game.AddInputAndRecordHistory(CellInput{
		Cell:   core.NewCell(row-1, column-1),
		Number: number,
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, "[ERROR] Error when adding the input:", err)
		return false
	}

	return true
}

// Function to run a command
func (game *SudokuGame) runCommand(command string) bool {
	commandFields := strings.SplitN(command, " ", 2)

	if len(commandFields) == 0 || len(commandFields[0]) == 0 {
		fmt.Fprintln(os.Stderr, "[ERROR] No command entered.")
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
			_, err := fmt.Sscanf(commandFields[1], "%d %d %d", &row, &column, &number)
			if err != nil {
				fmt.Fprintln(os.Stderr, "[ERROR] Error when reading the input:", err)
			} else {
				return game.runAddCommand(row, column, number)
			}
		}
	case "clear":
		if len(commandFields) != 2 {
			fmt.Fprintln(os.Stderr, "[ERROR] No argument specified for the clear command.")
		} else {
			var row, column int
			_, err := fmt.Sscanf(commandFields[1], "%d %d", &row, &column)
			if err != nil {
				fmt.Fprintln(os.Stderr, "[ERROR] Error when reading the input:", err)
			} else {
				return game.runAddCommand(row, column, 0)
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
			fmt.Println("You have entered an incorrect number.")
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
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "[ERROR] Unknown command.")
	}

	return false
}

// Function to ask the user for input
func (game *SudokuGame) askUserInput(scanner *bufio.Scanner) bool {
	// Print the problem
	game.print()

	// Ask for user input
	fmt.Println("Enter a command (Enter 'help' for help):")
	fmt.Print("> ")

	scanner.Scan()
	command := strings.TrimSpace(scanner.Text())

	return game.runCommand(command)
}

// Function to start the game
func (game *SudokuGame) Play() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		game.askUserInput(scanner)

		if game.IsSolved() {
			game.print()
			break
		}
	}

	fmt.Println("Congratulations! You have solved the problem.")
}
