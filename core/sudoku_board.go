package core

import (
	"errors"
	"fmt"

	"github.com/gnailuy/sudoku/util"
)

// Define the SudokuBoard struct.
type SudokuBoard struct {
	grid        [9][9]int
	filledCells int
}

// Constructor like function to create a empty Sudoku board.
func NewEmptySudokuBoard() SudokuBoard {
	var board SudokuBoard
	return board
}

// Function to set the value to a position.
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

// Function to set the value of a cell.
func (board *SudokuBoard) SetCell(cell Cell) (err error) {
	if !cell.IsValid() {
		return errors.New("cannot set invalid cell: " + cell.ToString())
	}

	board.Set(cell.Position, cell.Value)

	return nil
}

// Function to unset the value of a position.
func (board *SudokuBoard) Unset(position Position) {
	if board.grid[position.Row][position.Column] > 0 {
		board.filledCells--
	}
	board.grid[position.Row][position.Column] = 0
}

// Function to get the value of a position.
func (board *SudokuBoard) Get(position Position) int {
	return board.grid[position.Row][position.Column]
}

// Function to get a random position satisfying the value validator.
func (board *SudokuBoard) GetRandomPositionWith(validator func(int) bool) *Position {
	rowOrder := util.GenerateNumberArray(0, 9, true)
	columnOrder := util.GenerateNumberArray(0, 9, true)
	for _, row := range rowOrder {
		for _, column := range columnOrder {
			position := NewPosition(row, column)
			value := board.Get(position)
			if validator(value) {
				return &position
			}
		}
	}

	return nil
}

// Function to get the number of filled cells.
func (board *SudokuBoard) FilledCells() int {
	return board.filledCells
}

// Function to print the board as a single string.
func (board *SudokuBoard) ToString() string {
	result := ""

	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if board.grid[i][j] == 0 {
				result += "."
			} else {
				result += fmt.Sprint(board.grid[i][j])
			}
		}
	}

	return result
}

// Function to build a Sudoku board from a string.
func (board *SudokuBoard) FromString(s string) {
	if !IsValidSudokuString(s) {
		panic("Invalid Sudoku string")
	}

	for i := 0; i < 81; i++ {
		row := i / 9
		column := i % 9
		if s[i] == '.' {
			board.grid[row][column] = 0
		} else {
			board.grid[row][column] = int(s[i] - '0')
			board.filledCells++
		}
	}
}

// Function to check if a Sudoku string is valid.
func IsValidSudokuString(s string) bool {
	if len(s) != 81 {
		return false
	}

	for i := 0; i < 81; i++ {
		if s[i] != '.' && (s[i] < '1' || s[i] > '9') {
			return false
		}
	}

	return true
}
