package core

import (
	"errors"
	"fmt"

	"github.com/gnailuy/sudoku/util"
)

// Board represents a 9x9 Sudoku grid.
type Board struct {
	grid             [9][9]int
	filledCellsCount int

	// eliminations tracks additional candidate eliminations beyond what is
	// derived from placed values. Each entry is a bitmask of eliminated
	// digits (bit i set means digit i has been eliminated by a strategy
	// solver). This allows solvers like Swordfish and Hidden Subsets to
	// remove candidates without immediately placing a value.
	eliminations [9][9]CandidateSet
}

// NewEmptyBoard creates an empty Sudoku board with all cells set to zero.
func NewEmptyBoard() Board {
	return Board{}
}

// Function to set the value to a position.
func (board *Board) Set(position Position, value int) (err error) {
	if value < 1 || value > 9 {
		return errors.New("cannot set invalid number: " + fmt.Sprint(value))
	}

	if board.grid[position.Row][position.Column] == 0 {
		board.filledCellsCount++
	}

	board.grid[position.Row][position.Column] = value
	// Clear any eliminations for this cell since it's now filled.
	board.ClearEliminations(position)

	return nil
}

// Function to set the value of a cell.
func (board *Board) SetCell(cell Cell) (err error) {
	if !cell.IsValid() {
		return errors.New("cannot set invalid cell: " + cell.ToString())
	}

	_ = board.Set(cell.Position, cell.Value)

	return nil
}

// Function to unset the value of a position.
func (board *Board) Unset(position Position) {
	if board.grid[position.Row][position.Column] > 0 {
		board.filledCellsCount--
		board.grid[position.Row][position.Column] = 0
	}
}

// Function to get the value of a position.
func (board *Board) Get(position Position) int {
	return board.grid[position.Row][position.Column]
}

// Candidates computes and returns the candidate set for the cell at position.
// For filled cells, the set is empty. The result combines peer-based
// elimination (from placed values) with any additional eliminations tracked
// in the eliminations layer.
func (board *Board) Candidates(position Position) CandidateSet {
	cs := board.candidatesFromValues(position)
	cs &^= board.eliminations[position.Row][position.Column]
	return cs
}

func (board *Board) candidatesFromValues(position Position) CandidateSet {
	if board.grid[position.Row][position.Column] != 0 {
		return 0
	}
	cs := allCandidates
	for c := 0; c < 9; c++ {
		if v := board.grid[position.Row][c]; v != 0 {
			cs.Remove(v)
		}
	}
	for r := 0; r < 9; r++ {
		if v := board.grid[r][position.Column]; v != 0 {
			cs.Remove(v)
		}
	}
	startRow, startCol := position.Row/3*3, position.Column/3*3
	for r := startRow; r < startRow+3; r++ {
		for c := startCol; c < startCol+3; c++ {
			if v := board.grid[r][c]; v != 0 {
				cs.Remove(v)
			}
		}
	}
	return cs
}

// PruneRedundantEliminations drops strategy eliminations that placed values
// now exclude naturally while preserving every still-meaningful deduction.
func (board *Board) PruneRedundantEliminations() {
	for row := 0; row < 9; row++ {
		for column := 0; column < 9; column++ {
			position := NewPosition(row, column)
			board.eliminations[row][column] &= board.candidatesFromValues(position)
		}
	}
}

// EliminateCandidate removes a candidate digit from the given position's
// elimination set. Returns true if the digit was actually a candidate
// (i.e., it was present before elimination).
func (board *Board) EliminateCandidate(position Position, digit int) bool {
	if board.grid[position.Row][position.Column] != 0 {
		return false
	}
	cands := board.Candidates(position)
	if !cands.Has(digit) {
		return false
	}
	board.eliminations[position.Row][position.Column].Add(digit)
	return true
}

// ClearEliminations resets the elimination layer for the given position.
// Called when a cell is set (placed) to avoid stale elimination data.
func (board *Board) ClearEliminations(position Position) {
	board.eliminations[position.Row][position.Column] = 0
}

// Eliminations returns the strategy-derived candidate eliminations stored for
// an empty cell. The returned value is detached from the board.
func (board *Board) Eliminations(position Position) CandidateSet {
	return board.eliminations[position.Row][position.Column]
}

// EmptyPositions returns all positions on the board that are empty (value 0).
func (board *Board) EmptyPositions() []Position {
	positions := make([]Position, 0, 81-board.filledCellsCount)
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if board.grid[r][c] == 0 {
				positions = append(positions, NewPosition(r, c))
			}
		}
	}
	return positions
}

// ForEachCell calls fn for every cell on the board with its position and value.
// If fn returns false, iteration stops early.
func (board *Board) ForEachCell(fn func(position Position, value int) bool) {
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if !fn(NewPosition(r, c), board.grid[r][c]) {
				return
			}
		}
	}
}

// Function to get a random position satisfying the value validator.
func (board *Board) GetRandomPositionWith(validator func(int) bool) *Position {
	rowOrder := util.GenerateNumberArray(0, 9, true)
	columnOrder := util.GenerateNumberArray(0, 9, true)
	for _, row := range rowOrder {
		for _, column := range columnOrder {
			position := NewPosition(row, column)
			value := board.Get(position)
			if validator(value) {
				return &position
			}
		}
	}

	return nil
}

// Function to get the number of filled cells.
func (board *Board) GetFilledCellsCount() int {
	return board.filledCellsCount
}

// Function to return a copy of the board.
func (board *Board) Copy() Board {
	return Board{
		grid:             board.grid,
		filledCellsCount: board.filledCellsCount,
		eliminations:     board.eliminations,
	}
}

// Function to merge the board with another board.
func (board *Board) Merge(otherBoard Board) {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if board.grid[i][j] == 0 && otherBoard.grid[i][j] != 0 {
				_ = board.Set(NewPosition(i, j), otherBoard.grid[i][j])
			}
		}
	}
}
