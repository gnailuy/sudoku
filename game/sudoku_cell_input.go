package game

import "github.com/gnailuy/sudoku/core"

// Define the user input struct
type CellInput struct {
	Cell   core.Cell
	Number int
}

// Define the user input sequence struct with the previous value of the cell
type CellInputHistory struct {
	CellInput
	PreviousNumber int
}

// Function to check if the cell input is valid, allowing the number to be zero
func (input CellInput) IsValid() bool {
	return input.Cell.IsValid() && input.Number >= 0 && input.Number <= 9
}
