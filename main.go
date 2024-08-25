package main

import (
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/solver"
)

func main() {
	problem := solver.GenerateSudokuProblem(solver.NewDefaultSudokuProblemOptions())
	newGame := game.NewSudokuGame(problem)
	newGame.Play()
}
