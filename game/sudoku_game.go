package game

import (
	"errors"

	"github.com/gnailuy/sudoku/core"
)

// Define the cell state enum of the problem board
type CellState int

const (
	EmptyCell CellState = iota
	ProblemCell
	ValidInput
	InvalidInput
)

// Define the user input struct
type CellInput struct {
	Cell   core.Cell
	Number int
}

// Define the Sudoku game struct
type SudokuGame struct {
	// Public fields
	Problem core.SudokuBoard

	// Private fields
	boardState    [9][9]CellState
	solution      core.SudokuBoard
	inputSequence []CellInput
}

// Function to create a new Sudoku game
func NewSudokuGame(iteration int) SudokuGame {
	solvedBoard := core.GenerateSolvedBoard()
	problem := solvedBoard.GenerateSudokuProblem(iteration)
	boardState := [9][9]CellState{}

	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if problem.Get(core.NewCell(i, j)) == 0 {
				boardState[i][j] = EmptyCell
			} else {
				boardState[i][j] = ProblemCell
			}
		}
	}

	return SudokuGame{
		Problem:       problem,
		boardState:    boardState,
		solution:      solvedBoard,
		inputSequence: []CellInput{},
	}
}

// Function to check if a cell input is correct
func (game *SudokuGame) isValidInput(input CellInput) bool {
	return input.Cell.IsValid() && input.Number == game.solution.Get(input.Cell)
}

// Function to check if the game has invalid cells
func (game *SudokuGame) HasInvalidCells() bool {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if game.boardState[i][j] == InvalidInput {
				return true
			}
		}
	}

	return false
}

// Function to check if the game is solved
func (game *SudokuGame) IsSolved() bool {
	return game.Problem.Compare(game.solution)
}

// Function to add a cell input
func (game *SudokuGame) AddInput(input CellInput) (err error) {
	if !input.Cell.IsValid() {
		panic("Bug: Invalid cell input when adding input. Check user input before calling this function.")
	}

	if game.boardState[input.Cell.Row][input.Cell.Column] == ProblemCell {
		err = errors.New("cannot change the value of a problem cell")
		return
	}

	game.inputSequence = append(game.inputSequence, input)
	game.Problem.Set(input.Cell, input.Number)

	if game.isValidInput(input) {
		game.boardState[input.Cell.Row][input.Cell.Column] = ValidInput
	} else {
		game.boardState[input.Cell.Row][input.Cell.Column] = InvalidInput
	}

	return nil
}

// Undo the last cell input
func (game *SudokuGame) Undo() {
	if len(game.inputSequence) == 0 {
		return
	}

	lastInput := game.inputSequence[len(game.inputSequence)-1]
	game.inputSequence = game.inputSequence[:len(game.inputSequence)-1]
	game.Problem.Unset(lastInput.Cell)
	game.boardState[lastInput.Cell.Row][lastInput.Cell.Column] = EmptyCell
}
