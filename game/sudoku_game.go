package game

import (
	"errors"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

// Define the cell state enum of the problem board.
type CellState int

const (
	ProblemCell CellState = iota
	UserCell
)

// Define the user input sequence struct with the previous value of the cell.
type CellInputHistory struct {
	Input         core.Cell
	PreviousValue int
}

// Define the Sudoku game struct.
type SudokuGame struct {
	// Public fields.
	Problem core.SudokuBoard

	// Private fields.
	boardState      [9][9]CellState        // The state of the problem board.
	invalidInput    core.SudokuBoard       // Put the invalid input in another board to keep the original problem board solvable.
	inputSequence   []CellInputHistory     // User input sequence.
	inputCursor     int                    // The cursor of the current user input.
	defaultSolver   solver.ISudokuSolver   // The default solver to judge the input, must be reliable.
	strategySolvers []solver.ISudokuSolver // An optional list of strategy solvers to give hints, may be unreliable.
}

// Function to create a new Sudoku game.
func NewSudokuGame(problem core.SudokuBoard, options SudokuGameOptions) SudokuGame {
	if !problem.IsValid() {
		panic("Bug: Invalid problem board when creating a new Sudoku game")
	}

	boardState := [9][9]CellState{}

	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if problem.Get(core.NewPosition(i, j)) == 0 {
				boardState[i][j] = UserCell
			} else {
				boardState[i][j] = ProblemCell
			}
		}
	}

	return SudokuGame{
		Problem:         problem,
		boardState:      boardState,
		invalidInput:    core.NewEmptySudokuBoard(),
		inputSequence:   []CellInputHistory{},
		inputCursor:     -1,
		defaultSolver:   options.solverStore.GetDefaultSolver(),
		strategySolvers: options.GetStrategySolvers(),
	}
}

// Function to count the solutions of the problem board using the default solver.
func (game *SudokuGame) countProblemSolutions() int {
	return game.defaultSolver.CountSolutions(&game.Problem)
}

// Function to add a non-zero cell input.
func (game *SudokuGame) addNonZeroInput(input core.Cell) {
	if input.Value == 0 {
		panic("Bug: Cannot add a zero input with this function")
	}

	game.Problem.SetCell(input)
	game.invalidInput.Unset(input.Position) // Reset the invalid input state when adding a new input.

	if game.countProblemSolutions() <= 0 {
		// Store the invalid input in the invalidInput board and unset the cell in the problem board.
		game.Problem.Unset(input.Position)
		game.invalidInput.SetCell(input)
	}
}

// Function to add a zero.
func (game *SudokuGame) addZeroInput(input core.Cell) {
	if input.Value != 0 {
		panic("Bug: Cannot add a non-zero input with this function")
	}

	game.Problem.Unset(input.Position)
	game.invalidInput.Unset(input.Position) // Reset the invalid input state when adding a new input.

	// If the board has multiple solutions, we need to check if any previously invalid input is now valid.
	if !game.invalidInput.IsEmpty() && game.countProblemSolutions() > 1 {
		for i := 0; i < 9; i++ {
			for j := 0; j < 9; j++ {
				value := game.invalidInput.Get(core.NewPosition(i, j))
				if value != 0 {
					// Try to add the previously invalid input to the problem board.
					game.addNonZeroInput(core.NewCell(core.NewPosition(i, j), value))
				}
			}
		}
	}
}

// Function to get the cell value of the game boards.
func (game *SudokuGame) Get(position core.Position) int {
	if game.Problem.Get(position) != 0 {
		return game.Problem.Get(position)
	} else {
		return game.invalidInput.Get(position)
	}
}

// Function to add a cell input.
func (game *SudokuGame) AddInput(input core.Cell) (err error) {
	if !input.IsValid() {
		panic("Bug: Invalid input when adding input. Check user input before calling this function")
	}

	if game.boardState[input.Position.Row][input.Position.Column] == ProblemCell {
		err = errors.New("cannot change the value of a problem cell")
		return
	}

	if input.Value == 0 {
		game.addZeroInput(input)
	} else {
		game.addNonZeroInput(input)
	}

	return
}

// Function to add a cell input and record the history.
func (game *SudokuGame) AddInputAndRecordHistory(input core.Cell) (err error) {
	previousValue := game.Get(input.Position)

	err = game.AddInput(input)
	if err != nil {
		return
	}

	// On new input, we remove all the input after the cursor.
	if len(game.inputSequence) > game.inputCursor+1 {
		game.inputSequence = game.inputSequence[:game.inputCursor+1]
	}

	// Then append the new input to the input sequence.
	game.inputSequence = append(game.inputSequence, CellInputHistory{
		Input:         input,
		PreviousValue: previousValue,
	})
	game.inputCursor++

	return
}

// Function to undo the last cell input.
func (game *SudokuGame) Undo() (err error) {
	if game.inputCursor < 0 {
		err = errors.New("no input to undo")
		return
	}

	lastInput := game.inputSequence[game.inputCursor]
	game.inputCursor--

	game.AddInput(core.Cell{
		Position: lastInput.Input.Position,
		Value:    lastInput.PreviousValue,
	})

	return
}

// Function to redo the last undone cell input.
func (game *SudokuGame) Redo() (err error) {
	if game.inputCursor >= len(game.inputSequence)-1 {
		err = errors.New("no input to redo")
		return
	}

	game.inputCursor++
	nextInput := game.inputSequence[game.inputCursor]

	game.AddInput(nextInput.Input)

	return
}

// Function to repair the game to the last valid state.
func (game *SudokuGame) Repair() (undoSteps int) {
	for !game.IsValid() && game.inputCursor >= 0 {
		undoSteps++
		game.Undo()
	}

	return undoSteps
}

// Function to reset the game to the initial state.
func (game *SudokuGame) Reset() {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if game.boardState[i][j] == UserCell {
				game.Problem.Unset(core.NewPosition(i, j))
			}
		}
	}

	game.invalidInput = core.NewEmptySudokuBoard()
	game.inputSequence = []CellInputHistory{}
	game.inputCursor = -1
}

// Function to solve the game.
func (game *SudokuGame) Solve() {
	game.defaultSolver.Solve(&game.Problem)
}

// Function to get a hint of the game.
func (game *SudokuGame) Hint() *core.Cell {
	// If there is any invalid input, randomly remove one of them.
	if !game.invalidInput.IsEmpty() {
		positionPointer := game.invalidInput.GetRandomPositionWith(func(value int) bool {
			return value != 0
		})

		if positionPointer == nil {
			panic("Bug: Invalid input board is not empty but cannot find a valid position")
		}

		return &core.Cell{
			Position: *positionPointer,
			Value:    0,
		}
	}

	// If any of the strategy solvers can give a hint, use it.
	for _, solver := range game.strategySolvers {
		hint := solver.Hint(&game.Problem)
		if hint != nil {
			return hint
		}
	}

	// Otherwise, get a hint from the default solver.
	return game.defaultSolver.Hint(&game.Problem)
}

// Function to check if the game is solved.
func (game *SudokuGame) IsSolved() bool {
	return game.Problem.IsSolved()
}

// Function to check if the game is in a valid state.
func (game *SudokuGame) IsValid() bool {
	return game.invalidInput.IsEmpty()
}
