package core

import (
	"errors"
	"fmt"
)

// Placeholder for the zero values.
var defaultZeroPlaceholder = '.'
var allowedZeroPlaceholdersSet = map[byte]bool{
	'.': true,
	'0': true,
	'x': true,
	'X': true,
	'*': true,
	'_': true,
}

func isAllowedZeroPlaceholder(s byte) bool {
	return allowedZeroPlaceholdersSet[s]
}

// Function to check if a Sudoku string is valid.
func IsValidSudokuString(s string) bool {
	if len(s) != 81 {
		return false
	}

	for i := 0; i < 81; i++ {
		if !isAllowedZeroPlaceholder(s[i]) && (s[i] < '1' || s[i] > '9') {
			return false
		}
	}

	return true
}

// ParseBoard validates one serialized puzzle and returns its board value.
func ParseBoard(input string) (Board, error) {
	if !IsValidSudokuString(input) {
		return Board{}, errors.New("invalid Sudoku string: " + input)
	}
	board := NewEmptyBoard()
	board.FromString(input)
	if !board.IsValid() {
		return Board{}, errors.New("invalid Sudoku board: " + input)
	}
	return board, nil
}

// Function to print the board as a single string.
func (board *Board) ToString() string {
	result := ""

	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if board.grid[i][j] == 0 {
				result += string(defaultZeroPlaceholder)
			} else {
				result += fmt.Sprint(board.grid[i][j])
			}
		}
	}

	return result
}

// Function to build a Sudoku board from a string.
func (board *Board) FromString(s string) {
	if !IsValidSudokuString(s) {
		panic("Bug: Invalid Sudoku string")
	}

	for i := 0; i < 81; i++ {
		row := i / 9
		column := i % 9
		if !isAllowedZeroPlaceholder(s[i]) {
			_ = board.Set(NewPosition(row, column), int(s[i]-'0'))
		}
	}
}
