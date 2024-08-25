package core

import (
	"fmt"
)

// Function to print the Sudoku board
func (board *SudokuBoard) Print() {
	for i, row := range board.grid {
		if i%3 == 0 {
			fmt.Println("--------+-------+--------")
		}
		for j, cell := range row {
			if j%3 == 0 {
				fmt.Print("| ")
			}
			if cell == 0 {
				fmt.Print(". ")
			} else {
				fmt.Printf("%d ", cell)
			}
		}
		fmt.Println("|")
	}
	fmt.Println("--------+-------+--------")
}
