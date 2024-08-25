package core

import (
	"errors"
	"fmt"
)

// Define the SudokuBoard struct
type SudokuBoard struct {
	grid        [9][9]int
	filledCells int
}

// Constructor like function to create a empty Sudoku board
func NewEmptySudokuBoard() SudokuBoard {
	var board SudokuBoard
	return board
}

// Function to set the value to a position
func (board *SudokuBoard) Set(position Position, value int) (err error) {
	if value < 1 || value > 9 {
		return errors.New("cannot set invalid number: " + fmt.Sprint(value))
	}

	if board.grid[position.Row][position.Column] == 0 {
		board.filledCells++
	}
	board.grid[position.Row][position.Column] = value

	return nil
}

// Function to set the value of a cell
func (board *SudokuBoard) SetCell(cell Cell) (err error) {
	if !cell.IsValid() {
		return errors.New("cannot set invalid cell: " + cell.ToString())
	}

	board.Set(cell.Position, cell.Value)

	return nil
}

// Function to unset the value of a position
func (board *SudokuBoard) Unset(position Position) {
	if board.grid[position.Row][position.Column] > 0 {
		board.filledCells--
	}
	board.grid[position.Row][position.Column] = 0
}

// Function to get the value of a position
func (board *SudokuBoard) Get(position Position) int {
	return board.grid[position.Row][position.Column]
}

// Function to get the number of filled cells
func (board *SudokuBoard) FilledCells() int {
	return board.filledCells
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
