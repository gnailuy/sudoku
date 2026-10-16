package game

import (
	"fmt"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

// sessionState is the complete player-controlled state restored by undo and
// redo. Boards and candidate arrays are values, so history entries are detached.
type sessionState struct {
	playBoard    core.Board
	invalidInput core.Board
	candidates   [9][9]core.CandidateSet
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
	candidates      [9][9]core.CandidateSet // The one authoritative candidate grid.
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

	game := Game{
		problemBoard:    problem,
		playBoard:       problem.Copy(),
		invalidInput:    core.NewEmptyBoard(),
		inputSequence:   []historyRecord{},
		inputCursor:     -1,
		completeSolver:  options.solverStore.GetDefaultSolver(),
		strategySolvers: options.GetStrategySolvers(),
	}
	game.candidates = legalCandidateGrid(&game.playBoard)
	return game
}

// countSolutions uses a detached board because complete solvers may mutate their input while searching.
func (game *Game) countSolutions(board core.Board) int {
	search := board.Copy()
	return game.completeSolver.CountSolutions(&search)
}

// Function to add a non-zero cell input.
func (game *Game) addNonZeroInput(input core.Cell) {
	if input.Value == 0 {
		panic("Bug: Cannot add a zero input with this function")
	}

	nextBoard := game.playBoard.Copy()
	_ = nextBoard.SetCell(input) // cell validated by caller
	game.invalidInput.Unset(input.Position)
	if game.countSolutions(nextBoard) <= 0 {
		_ = game.invalidInput.SetCell(input) // cell validated by caller
		return
	}

	game.playBoard = nextBoard
}

// Function to add a zero.
func (game *Game) addZeroInput(input core.Cell) {
	if input.Value != 0 {
		panic("Bug: Cannot add a non-zero input with this function")
	}

	game.playBoard.Unset(input.Position)
	game.invalidInput.Unset(input.Position) // Reset the invalid input state when adding a new input.

	// If the board has multiple solutions, we need to check if any previously invalid input is now valid.
	if !game.invalidInput.IsEmpty() && game.countSolutions(game.playBoard) > 1 {
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
	game.applyValueCandidateCleanup(input)
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
	game.candidates = legalCandidateGrid(&game.playBoard)
	game.inputSequence = []historyRecord{}
	game.inputCursor = -1
}

// restart begins a clean attempt on the same immutable puzzle.
func (game *Game) restart() {
	game.reset()
	game.mistakes = 0
}

func (game *Game) captureState() sessionState {
	return sessionState{
		playBoard:    game.playBoard.Copy(),
		invalidInput: game.invalidInput.Copy(),
		candidates:   game.candidates,
	}
}

func (game *Game) restoreState(state sessionState) {
	game.playBoard = state.playBoard.Copy()
	game.invalidInput = state.invalidInput.Copy()
	game.candidates = state.candidates
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

func (game *Game) applyValueCandidateCleanup(input core.Cell) {
	row, column := input.Position.Row, input.Position.Column
	if input.Value == 0 {
		game.candidates[row][column] = game.playBoard.Candidates(input.Position)
		return
	}
	game.candidates[row][column] = 0
	if game.playBoard.Get(input.Position) != input.Value {
		return
	}
	for index := 0; index < 9; index++ {
		game.candidates[row][index].Remove(input.Value)
		game.candidates[index][column].Remove(input.Value)
	}
	boxRow, boxColumn := row-row%3, column-column%3
	for r := boxRow; r < boxRow+3; r++ {
		for c := boxColumn; c < boxColumn+3; c++ {
			game.candidates[r][c].Remove(input.Value)
		}
	}
}

// solve replaces the visible state with a complete solution.
func (game *Game) solve() {
	game.completeSolver.Solve(&game.playBoard)
	game.invalidInput = core.NewEmptyBoard()
	game.candidates = [9][9]core.CandidateSet{}
}

// Hint returns one complete renderer-neutral teaching plan without mutation.
// Interactive hint selection stops at the first meaningful deduction,
// including elimination-only progress.
func (game *Game) Hint() *HintPlan {
	if plan := game.valueCorrectionHint(); plan != nil {
		return plan
	}
	if plan := game.candidateCorrectionHint(); plan != nil {
		return plan
	}

	hintBoard := game.playBoard.Copy()
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			position := core.NewPosition(row, column)
			legal := hintBoard.Candidates(position)
			for _, value := range legal.Values() {
				if !game.candidates[row][column].Has(value) {
					hintBoard.EliminateCandidate(position, value)
				}
			}
		}
	}
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
		completeTeachingEvidence(&hintBoard, before, move)
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
		completeTeachingEvidence(&hintBoard, candidateGrid(&hintBoard), move)
		return composeHintPlan(game.Snapshot(), game.completeSolver, move)
	}
	return nil
}

func (game *Game) valueCorrectionHint() *HintPlan {
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			position := core.NewPosition(row, column)
			if game.invalidInput.Get(position) != 0 {
				return composeValueCorrectionPlan(game.Snapshot(), position)
			}
		}
	}
	return nil
}

func (game *Game) candidateCorrectionHint() *HintPlan {
	solution := game.playBoard.Copy()
	if !game.completeSolver.Solve(&solution) {
		return nil
	}
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			position := core.NewPosition(row, column)
			if game.problemBoard.Get(position) != 0 || game.Get(position) != 0 {
				continue
			}
			current := game.candidates[row][column]
			legal := game.playBoard.Candidates(position)
			corrected := current & legal
			corrected.Add(solution.Get(position))
			if corrected != current {
				return composeCandidateCorrectionPlan(game.Snapshot(), position, current, corrected, legal)
			}
		}
	}
	return nil
}

// completeTeachingEvidence turns every registered solver result into the same
// typed teaching boundary. Strategies may provide narrower evidence directly;
// the fallback derives exact effects and the candidate neighborhood used to
// explain them without requiring a client to understand a strategy name.
func completeTeachingEvidence(board *core.Board, before [9][9]core.CandidateSet, move *solver.Move) {
	if move.Evidence == nil {
		move.Evidence = &solver.Evidence{}
	}
	after := candidateGrid(board)
	if len(move.Evidence.Eliminations) == 0 {
		move.Evidence.Eliminations = candidateEliminations(before, after)
	}

	// Solvers that identify a placement without mutating their working board
	// still prove that every other candidate at the target is ruled out.
	if move.IsPlacement() && len(move.Evidence.RuledOut) == 0 {
		candidates := before[move.Cell.Position.Row][move.Cell.Position.Column]
		for _, value := range candidates.Values() {
			if value != move.Cell.Value {
				move.Evidence.RuledOut = append(move.Evidence.RuledOut, solver.CandidateRef{Position: move.Cell.Position, Value: value})
			}
		}
	}

	if len(move.Evidence.Premises) == 0 {
		move.Evidence.Premises = teachingPremises(before, move)
	}
	if len(move.Evidence.Units) == 0 {
		move.Evidence.Units = premiseUnits(move.Evidence.Premises)
	}
	if move.Evidence.Unit != nil && !containsUnit(move.Evidence.Units, *move.Evidence.Unit) {
		move.Evidence.Units = append(move.Evidence.Units, *move.Evidence.Unit)
	}
}

func teachingPremises(before [9][9]core.CandidateSet, move *solver.Move) []solver.CandidateGroup {
	targets := append([]solver.CandidateRef(nil), move.Evidence.Eliminations...)
	targets = append(targets, move.Evidence.RuledOut...)
	if move.IsPlacement() {
		targets = append(targets, solver.CandidateRef{Position: move.Cell.Position, Value: move.Cell.Value})
	}
	positions := make(map[core.Position]struct{})
	for _, target := range targets {
		positions[target.Position] = struct{}{}
		for row := 0; row < 9; row++ {
			for column := 0; column < 9; column++ {
				position := core.NewPosition(row, column)
				if position != target.Position && sharesTeachingUnit(position, target.Position) && before[row][column].Has(target.Value) {
					positions[position] = struct{}{}
				}
			}
		}
	}
	var groups []solver.CandidateGroup
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			position := core.NewPosition(row, column)
			if _, ok := positions[position]; !ok || before[row][column].IsEmpty() {
				continue
			}
			groups = append(groups, solver.CandidateGroup{Position: position, Values: before[row][column].Values()})
		}
	}
	return groups
}

func sharesTeachingUnit(a, b core.Position) bool {
	return a.Row == b.Row || a.Column == b.Column || (a.Row/3 == b.Row/3 && a.Column/3 == b.Column/3)
}

func premiseUnits(groups []solver.CandidateGroup) []solver.UnitRef {
	seen := make(map[solver.UnitRef]struct{})
	for _, group := range groups {
		refs := []solver.UnitRef{
			{Kind: solver.UnitRow, Index: group.Position.Row},
			{Kind: solver.UnitColumn, Index: group.Position.Column},
			{Kind: solver.UnitBox, Index: (group.Position.Row/3)*3 + group.Position.Column/3},
		}
		for _, ref := range refs {
			seen[ref] = struct{}{}
		}
	}
	var units []solver.UnitRef
	for _, kind := range []solver.UnitKind{solver.UnitRow, solver.UnitColumn, solver.UnitBox} {
		for index := 0; index < 9; index++ {
			ref := solver.UnitRef{Kind: kind, Index: index}
			if _, ok := seen[ref]; ok {
				units = append(units, ref)
			}
		}
	}
	return units
}

func containsUnit(units []solver.UnitRef, target solver.UnitRef) bool {
	for _, unit := range units {
		if unit == target {
			return true
		}
	}
	return false
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

func legalCandidateGrid(board *core.Board) [9][9]core.CandidateSet {
	return candidateGrid(board)
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
