package core

import "fmt"

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
		panic("Bug: Invalid Sudoku string")
	}

	for i := 0; i < 81; i++ {
		row := i / 9
		column := i % 9
		if s[i] == '.' {
			board.grid[row][column] = 0
		} else {
			board.grid[row][column] = int(s[i] - '0')
			board.filledCellsCount++
		}
	}
}
