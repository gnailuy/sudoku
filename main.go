package main

import (
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/generator"
	"github.com/gnailuy/sudoku/solver"
)

func main() {
	// Create and initialize the solver store.
	solverStore := solver.NewSudokuSolverStore()

	// Generate a problem.
	problem := generator.GenerateSudokuProblem(generator.NewDefaultSudokuProblemOptions(solverStore))

	// Play the game.
	newGame := game.NewSudokuGame(problem, game.NewDefaultSudokuGameOptions(solverStore))
	newGame.PlayCli()
}
