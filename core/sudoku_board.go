package core

import (
	"errors"
	"fmt"

	"github.com/gnailuy/sudoku/util"
)

// Define the SudokuBoard struct.
type SudokuBoard struct {
	grid             [9][9]int
	filledCellsCount int
}

// Constructor like function to create a empty Sudoku board.
func NewEmptySudokuBoard() SudokuBoard {
	return *new(SudokuBoard)
}

// Function to set the value to a position.
func (board *SudokuBoard) Set(position Position, value int) (err error) {
	if value < 1 || value > 9 {
		return errors.New("cannot set invalid number: " + fmt.Sprint(value))
	}

	if board.grid[position.Row][position.Column] == 0 {
		board.filledCellsCount++
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
		board.filledCellsCount--
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
func (board *SudokuBoard) GetFilledCellsCount() int {
	return board.filledCellsCount
}

// Function to return a copy of the board.
func (board *SudokuBoard) Copy() SudokuBoard {
	return SudokuBoard{
		grid:             board.grid,
		filledCellsCount: board.filledCellsCount,
	}
}
