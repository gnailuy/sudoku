package main

import (
	"fmt"
	"os"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/generator"
	"github.com/gnailuy/sudoku/solver"
	"github.com/spf13/pflag"
)

func main() {
	// Create and initialize the solver store.
	solverStore := solver.NewSudokuSolverStore()

	// Accept an optional argument to play a specific game given by an input string.
	// If the argument is not provided, a random game will be generated.
	input := pflag.StringP("input", "i", "", "A Sudoku problem string to play.")
	pflag.Parse()

	if *input != "" {
		// Read the input as a Sudoku string
		problem, err := generator.GenerateSudokuProblemFromString(*input)

		if err != nil {
			fmt.Fprintf(os.Stderr, "The input is not a valid Sudoku problem: %s\n", *input)
			os.Exit(1)
		}

		fmt.Println("Playing the game from input:")
		fmt.Println(*input)
		playCli(*problem, solverStore)
	} else {
		// Generate a random problem.
		fmt.Println("Generating a random Sudoku problem...")
		problem := generator.GenerateSudokuProblem(generator.NewDefaultSudokuProblemOptions(solverStore))

		playCli(problem, solverStore)
	}
}

// Function to play a game in CLI.
func playCli(problem core.SudokuBoard, solverStore solver.SudokuSolverStore) {
	newGame := game.NewSudokuGame(problem, game.NewDefaultSudokuGameOptions(solverStore))
	newGame.PlayCli()
}
