package game

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/eiannone/keyboard"
	"github.com/gnailuy/sudoku/core"
)

// Print help function
func (game *SudokuGame) printHelp() {
	fmt.Println("Supported commands:")
	fmt.Println("  - help: Print this help message.")
	fmt.Println("  - add `row` `column` `number`: Input a number to a call at (row, column).")
	fmt.Println("  - solve: Solve the problem for me.")
	fmt.Println("  - quit: Quit the game.")
}

// Function to run a command
func (game *SudokuGame) runCommand(command string) bool {
	command = strings.TrimSpace(command)
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
				return false
			} else {
				err := game.AddInput(CellInput{
					Cell:   core.NewCell(row-1, column-1),
					Number: number,
				})

				if err != nil {
					fmt.Fprintln(os.Stderr, "[ERROR] Error when adding the input:", err)
					return false
				}
				return true
			}
		}
	case "solve":
		game.Problem.Solve()
	case "quit":
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "[ERROR] Unknown command.")
	}

	return false
}

// Function to ask the user for input
func (game *SudokuGame) askUserInput() bool {
	// Print the problem
	game.Problem.Print()

	// Ask for user input
	fmt.Println("Enter a command (Enter 'help' for help):")
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	command, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return game.runCommand(command)
}

// Function to start the game
func (game *SudokuGame) Play() {
	err := keyboard.Open()
	if err != nil {
		panic(err)
	}
	defer keyboard.Close()

	for {
		added := game.askUserInput()

		if added && game.HasInvalidCells() {
			fmt.Fprintln(os.Stderr, "[ERROR] Your input is invalid.")
			game.Problem.Print()

			fmt.Println("Press any key to undo the last move...")
			_, _, err := keyboard.GetKey()
			if err != nil {
				panic(err)
			}

			game.Undo()
		}

		if game.IsSolved() {
			game.Problem.Print()
			break
		}
	}
}
