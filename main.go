package main

import "github.com/gnailuy/sudoku/core"

func main() {
	board := core.GenerateSudokuProblem(42)
	board.Print()

	board.Solve()
	board.Print()
}
