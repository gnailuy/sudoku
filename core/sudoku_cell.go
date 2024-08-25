package core

import (
	"errors"
	"fmt"
)

// Define the Cell struct for a Sudoku board position
type Cell struct {
	Row    int
	Column int
}

// Constructor like function to create a new Sudoku cell
// Use this when you are sure the cell is valid, will panic if the cell is invalid
func NewCell(row, column int) (cell Cell) {
	cell = Cell{Row: row, Column: column}

	if !cell.IsValid() {
		panic("Bug: Invalid cell: " + cell.ToString())
	}

	return
}

// Constructor like function to create a new Sudoku cell from user input
// Use this to deal with user input, will return an error if the cell is invalid
func NewCellFromInput(row, column int) (cell *Cell, err error) {
	cell = &Cell{Row: row, Column: column}

	if !cell.IsValid() {
		return nil, errors.New("Invalid cell : " + cell.ToString())
	}

	return cell, nil
}

// Function to check if a cell is valid
func (cell *Cell) IsValid() bool {
	return cell.Row >= 0 && cell.Row < 9 && cell.Column >= 0 && cell.Column < 9
}

// Function to print the cell as a coordinate
func (cell *Cell) ToString() string {
	return fmt.Sprintf("(%d, %d)", cell.Row+1, cell.Column+1)
}
