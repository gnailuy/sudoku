package core

import (
	"errors"
	"fmt"
)

// Define the SudokuBoard struct
type SudokuBoard struct {
	grid              [9][9]int
	filledCells       int
	numberOfSolutions int
}

// Constructor like function to create a empty Sudoku board
func NewEmptySudokuBoard() SudokuBoard {
	var board SudokuBoard
	return board
}

// Function to check if the board is empty
func (board *SudokuBoard) IsEmpty() bool {
	return board.filledCells == 0
}

// Function to set the value of a cell
func (board *SudokuBoard) Set(cell Cell, number int) (err error) {
	if number < 1 || number > 9 {
		return errors.New("Cannot set invalid number: " + fmt.Sprint(number))
	}

	if board.grid[cell.Row][cell.Column] == 0 {
		board.filledCells++
	}
	board.grid[cell.Row][cell.Column] = number

	return nil
}

// Function to unset the value of a cell
func (board *SudokuBoard) Unset(cell Cell) {
	if board.grid[cell.Row][cell.Column] > 0 {
		board.filledCells--
	}
	board.grid[cell.Row][cell.Column] = 0
}

// Function to get the value of a cell
func (board *SudokuBoard) Get(cell Cell) int {
	return board.grid[cell.Row][cell.Column]
}

// Function to print the board as a single string
func (board *SudokuBoard) ToString() string {
	result := ""

	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			result += fmt.Sprint(board.grid[i][j])
		}
	}

	return result
}
