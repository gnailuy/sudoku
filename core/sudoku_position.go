package core

import (
	"errors"
	"fmt"
)

// Define the struct for a Sudoku board position.
type Position struct {
	Row    int
	Column int
}

// Constructor like function to create a new Sudoku position.
// Use this when you are sure the position is valid, will panic otherwise.
// Note that the internal representation is 0-indexed.
func NewPosition(row, column int) (position Position) {
	position = Position{Row: row, Column: column}

	if !position.IsValid() {
		panic("Bug: Invalid board position: " + position.ToString())
	}

	return
}

// Constructor like function to create a new Sudoku position from user input.
// Use this to deal with user input, will return an error if the position is invalid.
// Note that the user input is 1-indexed.
func NewPositionFromInput(rowInput, columnInput int) (position *Position, err error) {
	position = &Position{Row: rowInput - 1, Column: columnInput - 1}

	if !position.IsValid() {
		return nil, errors.New("invalid board position: " + position.ToString())
	}

	return position, nil
}

// Function to check if a position is valid.
func (position *Position) IsValid() bool {
	return position.Row >= 0 && position.Row < 9 && position.Column >= 0 && position.Column < 9
}

// Function to print the position as a user facing coordinate, 1-indexed.
func (position *Position) ToString() string {
	return fmt.Sprintf("(%d, %d)", position.Row+1, position.Column+1)
}
