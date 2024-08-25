package game

import (
	"errors"

	"github.com/gnailuy/sudoku/core"
)

// Define the cell state enum of the problem board
type CellState int

const (
	ProblemCell CellState = iota
	UserCell
)

// Define the user input struct
type CellInput struct {
	Cell   core.Cell
	Number int
}

// Check if the cell input is valid
func (input CellInput) isValid() bool {
	return input.Cell.IsValid() && input.Number >= 0 && input.Number <= 9
}

// Define the input sequence struct with the old value of the cell
type CellInputHistory struct {
	CellInput
	OldNumber int
}

// Define the Sudoku game struct
type SudokuGame struct {
	// Public fields
	Problem core.SudokuBoard

	// Private fields
	boardState    [9][9]CellState    // The state of the problem board
	invalidInput  core.SudokuBoard   // We put the invalid input in another board to keep the original problem board solvable
	inputSequence []CellInputHistory // User input sequence
	inputCursor   int                // The cursor of the current user input
}

// Function to create a new Sudoku game
func NewSudokuGame(options core.SudokuProblemOptions) SudokuGame {
	solvedBoard := core.GenerateSolvedBoard()
	problem := solvedBoard.GenerateSudokuProblem(options)
	boardState := [9][9]CellState{}

	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if problem.Get(core.NewCell(i, j)) == 0 {
				boardState[i][j] = UserCell
			} else {
				boardState[i][j] = ProblemCell
			}
		}
	}

	return SudokuGame{
		Problem:       problem,
		boardState:    boardState,
		invalidInput:  core.NewEmptySudokuBoard(),
		inputSequence: []CellInputHistory{},
		inputCursor:   -1,
	}
}

// Function to add a non-zero cell input
func (game *SudokuGame) addNonZeroInput(input CellInput) {
	if input.Number == 0 {
		panic("Bug: Cannot add a zero input with this function.")
	}

	game.Problem.Set(input.Cell, input.Number)
	game.invalidInput.Unset(input.Cell) // Reset the invalid input state when adding a new input

	if !game.Problem.IsSolvable() {
		// We store the invalid input in the invalidInput array and unset the cell in the problem board
		game.Problem.Unset(input.Cell)
		game.invalidInput.Set(input.Cell, input.Number)
	}
}

// Function to add a zero
func (game *SudokuGame) addZeroInput(input CellInput) {
	if input.Number != 0 {
		panic("Bug: Cannot add a non-zero input with this function.")
	}

	game.Problem.Unset(input.Cell)
	game.invalidInput.Unset(input.Cell) // Reset the invalid input state when adding a new input

	// If the board has multiple solutions, we need to check if any previously invalid input is now valid
	if !game.invalidInput.IsEmpty() && game.Problem.CountSolutions() > 1 {
		for i := 0; i < 9; i++ {
			for j := 0; j < 9; j++ {
				cell := core.NewCell(i, j)
				number := game.invalidInput.Get(cell)
				if number != 0 {
					game.addNonZeroInput(CellInput{
						Cell:   cell,
						Number: number,
					})
				}
			}
		}
	}
}

// Function to add a cell input
func (game *SudokuGame) AddInput(input CellInput) (err error) {
	if !input.isValid() {
		panic("Bug: Invalid input when adding input. Check user input before calling this function.")
	}

	if game.boardState[input.Cell.Row][input.Cell.Column] == ProblemCell {
		err = errors.New("cannot change the value of a problem cell")
		return
	}

	if input.Number == 0 {
		game.addZeroInput(input)
	} else {
		game.addNonZeroInput(input)
	}

	return
}

// Function to add a cell input and record the history
func (game *SudokuGame) AddInputAndRecordHistory(input CellInput) (err error) {
	oldNumber := max(game.Problem.Get(input.Cell), game.invalidInput.Get(input.Cell))

	err = game.AddInput(input)
	if err != nil {
		return
	}

	// On new input, we remove all the input after the cursor
	if len(game.inputSequence) > game.inputCursor+1 {
		game.inputSequence = game.inputSequence[:game.inputCursor+1]
	}

	game.inputSequence = append(game.inputSequence, CellInputHistory{
		CellInput: input,
		OldNumber: oldNumber,
	})
	game.inputCursor++

	return
}

// Undo the last cell input
func (game *SudokuGame) Undo() (err error) {
	if game.inputCursor < 0 {
		err = errors.New("no input to undo")
		return
	}

	lastInput := game.inputSequence[game.inputCursor]
	game.inputCursor--

	game.AddInput(CellInput{
		Cell:   lastInput.Cell,
		Number: lastInput.OldNumber,
	})

	return
}

// Redo the next cell input
func (game *SudokuGame) Redo() (err error) {
	if game.inputCursor >= len(game.inputSequence)-1 {
		err = errors.New("no input to redo")
		return
	}

	game.inputCursor++
	nextInput := game.inputSequence[game.inputCursor]

	game.AddInput(CellInput{
		Cell:   nextInput.Cell,
		Number: nextInput.Number,
	})

	return
}

// Function to reset the game to the initial state
func (game *SudokuGame) Reset() {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if game.boardState[i][j] == UserCell {
				game.Problem.Unset(core.NewCell(i, j))
			}
		}
	}

	game.invalidInput = core.NewEmptySudokuBoard()
	game.inputSequence = []CellInputHistory{}
	game.inputCursor = -1
}

// Function to check if the game is solved
func (game *SudokuGame) IsSolved() bool {
	return game.Problem.IsSolved()
}

// Function to check if the game is in an invalid state
func (game *SudokuGame) IsInvalid() bool {
	return !game.invalidInput.IsEmpty()
}
