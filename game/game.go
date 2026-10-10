package game

import (
	"fmt"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

// sessionState is the complete player-controlled state restored by undo and
// redo. Boards and note arrays are values, so history entries are detached.
type sessionState struct {
	playBoard    core.Board
	invalidInput core.Board
	notes        [9][9]core.CandidateSet
}

type historyRecord struct {
	before sessionState
	after  sessionState
}

// Game holds the state for an interactive Sudoku session.
type Game struct {
	problemBoard    core.Board
	playBoard       core.Board
	invalidInput    core.Board              // Put the invalid input in another board to keep the play board solvable.
	notes           [9][9]core.CandidateSet // Manual player notes, independent of solver candidates.
	mistakes        int                     // Confirmed invalid value submissions; deliberately outside undo/redo history.
	inputSequence   []historyRecord         // Atomic value and note transitions.
	inputCursor     int                     // The cursor of the current transition.
	completeSolver  solver.CompleteSolver   // The complete solver for judging input and solving, must be reliable.
	strategySolvers []solver.StrategySolver // An optional list of strategy solvers to give hints.
}

// NewGame creates a new game from a problem board and options.
func NewGame(problem core.Board, options Options) Game {
	if !problem.IsValid() {
		panic("Bug: Invalid problem board when creating a new Sudoku game")
	}

	return Game{
		problemBoard:    problem,
		playBoard:       problem.Copy(),
		invalidInput:    core.NewEmptyBoard(),
		inputSequence:   []historyRecord{},
		inputCursor:     -1,
		completeSolver:  options.solverStore.GetDefaultSolver(),
		strategySolvers: options.GetStrategySolvers(),
	}
}

// Function to count the solutions of the current play board using the complete solver.
func (game *Game) countSolutions() int {
	return game.completeSolver.CountSolutions(&game.playBoard)
}

// Function to add a non-zero cell input.
func (game *Game) addNonZeroInput(input core.Cell) {
	if input.Value == 0 {
		panic("Bug: Cannot add a zero input with this function")
	}

	_ = game.playBoard.SetCell(input)       // cell validated by caller
	game.invalidInput.Unset(input.Position) // Reset the invalid input state when adding a new input.

	if game.countSolutions() <= 0 {
		// Store the invalid input in the invalidInput board and unset the cell in the play board.
		game.playBoard.Unset(input.Position)
		_ = game.invalidInput.SetCell(input) // cell validated by caller
	}
}

// Function to add a zero.
func (game *Game) addZeroInput(input core.Cell) {
	if input.Value != 0 {
		panic("Bug: Cannot add a non-zero input with this function")
	}

	game.playBoard.Unset(input.Position)
	game.invalidInput.Unset(input.Position) // Reset the invalid input state when adding a new input.

	// If the board has multiple solutions, we need to check if any previously invalid input is now valid.
	if !game.invalidInput.IsEmpty() && game.countSolutions() > 1 {
		for i := 0; i < 9; i++ {
			for j := 0; j < 9; j++ {
				value := game.invalidInput.Get(core.NewPosition(i, j))
				if value != 0 {
					// Try to add the previously invalid input to the play board.
					game.addNonZeroInput(core.NewCell(core.NewPosition(i, j), value))
				}
			}
		}
	}
}

// Function to get the cell value of the game boards.
func (game *Game) Get(position core.Position) int {
	if game.playBoard.Get(position) != 0 {
		return game.playBoard.Get(position)
	} else {
		return game.invalidInput.Get(position)
	}
}

// addInput applies a cell value to the visible state.
func (game *Game) addInput(input core.Cell) (err error) {
	if !input.IsValid() {
		return invalidCellError(input.Position, input.Value)
	}

	if game.problemBoard.Get(input.Position) != 0 {
		position := input.Position
		return &EngineError{
			Code:     ErrorImmutableCell,
			Position: &position,
			Detail:   "cannot change the value of a problem cell",
		}
	}

	if input.Value == 0 {
		game.addZeroInput(input)
	} else {
		game.addNonZeroInput(input)
	}

	return
}

// addInputAndRecordHistory applies a value transition and records it.
func (game *Game) addInputAndRecordHistory(input core.Cell) (err error) {
	if !input.IsValid() {
		return invalidCellError(input.Position, input.Value)
	}

	before := game.captureState()
	err = game.addInput(input)
	if err != nil {
		return
	}
	game.applyValueNoteCleanup(input)
	game.recordTransition(before)

	return
}

// undo restores the state before the current history record.
func (game *Game) undo() (err error) {
	if game.inputCursor < 0 {
		return &EngineError{Code: ErrorNoUndo, Detail: "no input to undo"}
	}

	record := game.inputSequence[game.inputCursor]
	game.inputCursor--
	game.restoreState(record.before)

	return
}

// redo restores the next state in history.
func (game *Game) redo() (err error) {
	if game.inputCursor >= len(game.inputSequence)-1 {
		return &EngineError{Code: ErrorNoRedo, Detail: "no input to redo"}
	}

	game.inputCursor++
	record := game.inputSequence[game.inputCursor]
	game.restoreState(record.after)

	return
}

// repair undoes transitions until the visible state is valid.
func (game *Game) repair() (undoSteps int) {
	for !game.IsValid() && game.inputCursor >= 0 {
		undoSteps++
		_ = game.undo()
	}

	return undoSteps
}

// reset restores the original puzzle and clears transition history.
func (game *Game) reset() {
	game.playBoard = game.problemBoard.Copy()
	game.invalidInput = core.NewEmptyBoard()
	game.notes = [9][9]core.CandidateSet{}
	game.inputSequence = []historyRecord{}
	game.inputCursor = -1
}

func (game *Game) captureState() sessionState {
	return sessionState{
		playBoard:    game.playBoard.Copy(),
		invalidInput: game.invalidInput.Copy(),
		notes:        game.notes,
	}
}

func (game *Game) restoreState(state sessionState) {
	game.playBoard = state.playBoard.Copy()
	game.invalidInput = state.invalidInput.Copy()
	game.notes = state.notes
}

func (game *Game) recordTransition(before sessionState) {
	if len(game.inputSequence) > game.inputCursor+1 {
		game.inputSequence = game.inputSequence[:game.inputCursor+1]
	}
	game.inputSequence = append(game.inputSequence, historyRecord{
		before: before,
		after:  game.captureState(),
	})
	game.inputCursor++
}

func (game *Game) applyValueNoteCleanup(input core.Cell) {
	row, column := input.Position.Row, input.Position.Column
	game.notes[row][column] = 0
	if input.Value == 0 || game.playBoard.Get(input.Position) != input.Value {
		return
	}
	for index := 0; index < 9; index++ {
		game.notes[row][index].Remove(input.Value)
		game.notes[index][column].Remove(input.Value)
	}
	boxRow, boxColumn := row-row%3, column-column%3
	for r := boxRow; r < boxRow+3; r++ {
		for c := boxColumn; c < boxColumn+3; c++ {
			game.notes[r][c].Remove(input.Value)
		}
	}
}

// solve replaces the visible state with a complete solution.
func (game *Game) solve() {
	game.completeSolver.Solve(&game.playBoard)
	game.invalidInput = core.NewEmptyBoard()
	game.notes = [9][9]core.CandidateSet{}
}

// Hint returns one complete renderer-neutral teaching plan without mutation.
// Interactive hint selection stops at the first meaningful deduction,
// including elimination-only progress.
func (game *Game) Hint() *HintPlan {
	hintBoard := game.playBoard.Copy()
	for _, strategy := range game.strategySolvers {
		before := candidateGrid(&hintBoard)
		var move *solver.Move
		if teaching, ok := strategy.(solver.TeachingStrategy); ok {
			move = teaching.Hint(&hintBoard)
		} else {
			move = strategy.Apply(&hintBoard)
		}
		if move == nil {
			continue
		}
		if move.Evidence == nil {
			move.Evidence = &solver.Evidence{}
		}
		if move.EliminationOnly && len(move.Evidence.Eliminations) == 0 {
			move.Evidence.Eliminations = candidateEliminations(before, candidateGrid(&hintBoard))
		}
		return composeHintPlan(game.Snapshot(), strategy, move)
	}

	solved := hintBoard.Copy()
	if !game.completeSolver.Solve(&solved) {
		return nil
	}
	for _, position := range hintBoard.EmptyPositions() {
		value := solved.Get(position)
		move := &solver.Move{
			Cell:      core.NewCell(position, value),
			Technique: "backtracker",
			Reason:    fmt.Sprintf("backtracking finds %d at %s", value, position.ToString()),
		}
		return composeHintPlan(game.Snapshot(), game.completeSolver, move)
	}
	return nil
}

func candidateGrid(board *core.Board) [9][9]core.CandidateSet {
	var candidates [9][9]core.CandidateSet
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			candidates[row][column] = board.Candidates(core.NewPosition(row, column))
		}
	}
	return candidates
}

func candidateEliminations(before, after [9][9]core.CandidateSet) []solver.CandidateRef {
	var refs []solver.CandidateRef
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			removed := before[row][column] &^ after[row][column]
			for _, value := range removed.Values() {
				refs = append(refs, solver.CandidateRef{Position: core.NewPosition(row, column), Value: value})
			}
		}
	}
	return refs
}

// Function to check if the game is solved.
func (game *Game) IsSolved() bool {
	return game.playBoard.IsSolved()
}

// Function to check if the game is in a valid state.
func (game *Game) IsValid() bool {
	return game.invalidInput.IsEmpty()
}

// Function to print the Sudoku game to string.
func (game *Game) ToString() string {
	result := "Problem:\n"
	result += game.problemBoard.ToString()
	result += "\n"

	playBoardCopy := game.playBoard.Copy()
	playBoardCopy.Merge(game.invalidInput)

	status := "Valid"
	if game.IsSolved() {
		status = "Solved"
	} else if !game.IsValid() {
		status = "Invalid"
	}

	if playBoardCopy != game.problemBoard {
		result += "Current board (" + status + "):\n"
		result += playBoardCopy.ToString()
		result += "\n"
	}

	return result
}
