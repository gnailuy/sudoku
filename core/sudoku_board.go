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

// Function to set the value of a cell
func (board *SudokuBoard) Set(cell Cell, number int) (err error) {
	if number < 1 || number > 9 {
		return errors.New("Cannot set invalid number: " + fmt.Sprint(number))
	}

	board.grid[cell.Row][cell.Column] = number
	board.filledCells++

	return nil
}

// Function to unset the value of a cell
func (board *SudokuBoard) Unset(cell Cell) {
	board.grid[cell.Row][cell.Column] = 0
	board.filledCells--
}

// Function to get the value of a cell
func (board *SudokuBoard) Get(cell Cell) int {
	return board.grid[cell.Row][cell.Column]
}

// Function to compare the board with another board
func (board *SudokuBoard) Compare(other SudokuBoard) bool {
	return board.grid == other.grid
}
