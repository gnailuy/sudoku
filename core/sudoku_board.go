package core

import "fmt"

// Define the SudokuBoard struct
type SudokuBoard struct {
	grid              [9][9]int
	filledCells       int
	numberOfSolutions int
}

func (board *SudokuBoard) Set(row, col, num int) {
	if num < 1 || num > 9 {
		panic("Cannot set invalid number: " + fmt.Sprint(num))
	}

	board.grid[row][col] = num
	board.filledCells++
}

func (board *SudokuBoard) Unset(row, col int) {
	board.grid[row][col] = 0
	board.filledCells--
}

func (board *SudokuBoard) Get(row, col int) int {
	return board.grid[row][col]
}
