package solver

import (
	"fmt"

	"github.com/gnailuy/sudoku/core"
)

// HiddenSingleSolver finds candidates that appear in only one cell within a unit
// (row, column, or box). Even if that cell has multiple candidates, this value
// must go there because no other cell in the unit can hold it.
type HiddenSingleSolver struct {
	Base
}

// NewHiddenSingleSolver creates a HiddenSingleSolver and returns it.
func NewHiddenSingleSolver() *HiddenSingleSolver {
	return &HiddenSingleSolver{
		Base: Base{
			Key:         "hidden-single",
			DisplayName: "Hidden Single",
			Description: "Finds a candidate that appears in only one cell within a row, column, or box.",
			Weight:      WeightHiddenSingle,
		},
	}
}

// Apply checks all units (rows, columns, boxes) for hidden singles.
func (s *HiddenSingleSolver) Apply(board *core.Board) *Move {
	// Check rows.
	for row := 0; row < 9; row++ {
		positions := make([]core.Position, 0, 9)
		for col := 0; col < 9; col++ {
			positions = append(positions, core.NewPosition(row, col))
		}
		if move := s.findHiddenSingle(board, positions, UnitRef{Kind: UnitRow, Index: row}); move != nil {
			return move
		}
	}

	// Check columns.
	for col := 0; col < 9; col++ {
		positions := make([]core.Position, 0, 9)
		for row := 0; row < 9; row++ {
			positions = append(positions, core.NewPosition(row, col))
		}
		if move := s.findHiddenSingle(board, positions, UnitRef{Kind: UnitColumn, Index: col}); move != nil {
			return move
		}
	}

	// Check boxes.
	for boxRow := 0; boxRow < 3; boxRow++ {
		for boxCol := 0; boxCol < 3; boxCol++ {
			positions := make([]core.Position, 0, 9)
			startRow, startCol := boxRow*3, boxCol*3
			for r := startRow; r < startRow+3; r++ {
				for c := startCol; c < startCol+3; c++ {
					positions = append(positions, core.NewPosition(r, c))
				}
			}
			if move := s.findHiddenSingle(board, positions, UnitRef{Kind: UnitBox, Index: boxRow*3 + boxCol}); move != nil {
				return move
			}
		}
	}

	return nil
}

// findHiddenSingle checks a single unit (9 positions) for a candidate value
// that appears in exactly one empty cell.
func (s *HiddenSingleSolver) findHiddenSingle(board *core.Board, positions []core.Position, unit UnitRef) *Move {
	// For each digit 1-9, track which empty cells in this unit can hold it.
	for digit := 1; digit <= 9; digit++ {
		var found core.Position
		count := 0

		for _, pos := range positions {
			if board.Get(pos) != 0 {
				continue
			}

			candidates := board.Candidates(pos)
			if candidates.Has(digit) {
				found = pos
				count++
				if count > 1 {
					break // More than one cell can hold this digit; not a hidden single.
				}
			}
		}

		if count == 1 {
			unitName := fmt.Sprintf("%s %d", unit.Kind, unit.Index+1)
			ruledOut := make([]CandidateRef, 0, 8)
			witnesses := make([]core.Cell, 0, 8)
			for _, pos := range positions {
				if pos != found && board.Get(pos) == 0 {
					ruledOut = append(ruledOut, CandidateRef{Position: pos, Value: digit})
					if witness, ok := hiddenSingleWitness(board, pos, digit, unit); ok {
						witnesses = append(witnesses, witness)
					}
				}
			}
			return &Move{
				Cell:      core.NewCell(found, digit),
				Technique: s.Key,
				Reason:    fmt.Sprintf("%d can only go in %s within %s", digit, found.ToString(), unitName),
				Evidence: &Evidence{
					Unit:      &unit,
					Premises:  []CandidateGroup{{Position: found, Values: []int{digit}}},
					Witnesses: witnesses,
					RuledOut:  ruledOut,
				},
			}
		}
	}

	return nil
}

func hiddenSingleWitness(board *core.Board, target core.Position, digit int, source UnitRef) (core.Cell, bool) {
	row := make([]core.Position, 0, 9)
	column := make([]core.Position, 0, 9)
	box := make([]core.Position, 0, 9)
	for index := 0; index < 9; index++ {
		row = append(row, core.NewPosition(target.Row, index))
		column = append(column, core.NewPosition(index, target.Column))
	}
	startRow, startColumn := target.Row/3*3, target.Column/3*3
	for rowIndex := startRow; rowIndex < startRow+3; rowIndex++ {
		for columnIndex := startColumn; columnIndex < startColumn+3; columnIndex++ {
			box = append(box, core.NewPosition(rowIndex, columnIndex))
		}
	}

	search := func(positions []core.Position) (core.Cell, bool) {
		for _, position := range positions {
			if board.Get(position) == digit {
				return core.NewCell(position, digit), true
			}
		}
		return core.Cell{}, false
	}

	var peers [][]core.Position
	switch source.Kind {
	case UnitRow:
		peers = [][]core.Position{column, box}
	case UnitColumn:
		peers = [][]core.Position{row, box}
	default:
		peers = [][]core.Position{row, column}
	}
	for _, positions := range peers {
		if witness, ok := search(positions); ok {
			return witness, true
		}
	}
	return core.Cell{}, false
}
