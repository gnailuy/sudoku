package main

import (
	"fmt"
	"os"

	"github.com/gnailuy/sudoku/cli"
	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/generator"
	"github.com/gnailuy/sudoku/solver"
)

func main() {
	// Create and initialize the solver store.
	solverStore := solver.NewSudokuSolverStore()

	// Parse the command line options.
	options := cli.NewCommandLineOptions()
	options.Parse()

	if *options.Input != "" {
		// Read the input as a Sudoku string
		problem, err := generator.GenerateSudokuProblemFromString(*options.Input)

		if err != nil {
			fmt.Fprintf(os.Stderr, "The input is not a valid Sudoku problem: %s\n", *options.Input)
			os.Exit(1)
		}

		fmt.Println("Playing the game from input:")
		fmt.Println(*options.Input)
		playCli(*problem, solverStore)
	} else {
		// Generate a random problem.
		fmt.Printf("Generating a random %s Sudoku problem...\n", options.Level.String())
		problem := generator.GenerateSudokuProblem(generator.NewSudokuProblemOptions(solverStore, options.GetDifficultyOptions()))

		playCli(problem, solverStore)
	}
}

// Function to play a game in CLI.
func playCli(problem core.SudokuBoard, solverStore solver.SudokuSolverStore) {
	newGame := game.NewSudokuGame(problem, game.NewDefaultSudokuGameOptions(solverStore))
	newGame.PlayCli()
}
