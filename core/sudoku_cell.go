package core

import (
	"errors"
	"fmt"
)

// Define the struct for a Sudoku board cell
type Cell struct {
	Position Position
	Value    int
}

// Constructor like function to create a new Sudoku cell
// Use this when you are sure the cell is valid, will panic otherwise
func NewCell(position Position, value int) (cell Cell) {
	if !position.IsValid() {
		panic("Bug: Invalid cell position: " + position.ToString())
	}

	if value < 0 || value > 9 {
		panic("Bug: Invalid cell value: " + fmt.Sprint(value))
	}

	cell = Cell{Position: position, Value: value}

	return
}

// Constructor like function to create a new Sudoku cell from user input
// Use this to deal with user input, will return an error if the cell is invalid
func NewCellFromInput(position Position, value int) (cell *Cell, err error) {
	if !position.IsValid() {
		return nil, errors.New("invalid cell position: " + position.ToString())
	}

	if value < 0 || value > 9 {
		return nil, errors.New("invalid cell value: " + fmt.Sprint(value))
	}

	cell = &Cell{Position: position, Value: value}

	return cell, nil
}

// Function to check if a cell is valid
func (cell *Cell) IsValid() bool {
	return cell.Position.IsValid() && cell.Value >= 0 && cell.Value <= 9
}

// Function to print the cell as a user facing coordinate, 1-indexed
func (cell *Cell) ToString() string {
	return fmt.Sprintf("%s: %d", cell.Position.ToString(), cell.Value)
}
