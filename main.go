package main

import "github.com/gnailuy/sudoku/game"

func main() {
	newGame := game.NewSudokuGame(42)
	newGame.Play()
}
