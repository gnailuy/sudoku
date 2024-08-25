package main

import (
	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
)

func main() {
	newGame := game.NewSudokuGame(core.NewDefaultSudokuProblemOptions())
	newGame.Play()
}
