package solver

import (
	"fmt"

	"github.com/gnailuy/sudoku/core"
)

// BUGPlusOneSolver detects the Bivalue Universal Grave + 1 pattern.
//
// A Bivalue Universal Grave (BUG) is a state where every unsolved cell has
// exactly 2 candidates. If a valid Sudoku were to reach this state, it would
// have multiple solutions — which violates the uniqueness constraint. The
// BUG+1 pattern occurs when exactly one unsolved cell has 3 candidates and
// all other unsolved cells have exactly 2. The extra (third) candidate in
// that cell must be the correct value, because removing it would create a
// BUG, implying multiple solutions.
//
// To identify the extra candidate: for each candidate in the trivalue cell,
// count how many times it appears as a candidate in the cell's row, column,
// and box. In a BUG state, every digit appears exactly twice per unit among
// unsolved cells. The candidate that appears an odd number of times (3 times)
// in any of its units is the extra one — and therefore the correct value.
type BUGPlusOneSolver struct {
	Base
}

// NewBUGPlusOneSolver creates a BUGPlusOneSolver.
func NewBUGPlusOneSolver() *BUGPlusOneSolver {
	return &BUGPlusOneSolver{
		Base: Base{
			Key:         "bug-plus-one",
			DisplayName: "BUG+1",
			Description: "Bivalue Universal Grave + 1: when all unsolved cells have 2 candidates except one with 3, the extra candidate is the answer.",
			Weight:      WeightBUGPlusOne,
		},
	}
}

// Apply checks for the BUG+1 pattern and returns a placement move if found.
func (s *BUGPlusOneSolver) Apply(board *core.Board) *Move {
	var triCell *core.Position
	var triCands core.CandidateSet
	var bivalueCells []CandidateGroup

	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			pos := core.NewPosition(r, c)
			if board.Get(pos) != 0 {
				continue
			}

			cands := board.Candidates(pos)
			count := cands.Count()

			switch {
			case count == 2:
				bivalueCells = append(bivalueCells, CandidateGroup{Position: pos, Values: cands.Values()})
			case count == 3:
				if triCell != nil {
					// More than one trivalue cell — not BUG+1.
					return nil
				}
				p := pos
				triCell = &p
				triCands = cands
			default:
				// Any cell with 0, 1, or 4+ candidates breaks the BUG pattern.
				return nil
			}
		}
	}

	if triCell == nil {
		// All unsolved cells are bivalue — pure BUG (shouldn't happen in valid puzzles).
		return nil
	}

	// Find the extra candidate: the one that appears an odd number of times
	// in the trivalue cell's row, column, or box.
	value := findBUGExtraCandidate(board, *triCell, triCands)
	if value == 0 {
		return nil
	}

	premises := make([]CandidateGroup, 0, len(bivalueCells)+1)
	premises = append(premises, CandidateGroup{Position: *triCell, Values: triCands.Values()})
	premises = append(premises, bivalueCells...)

	return &Move{
		Cell:      core.NewCell(*triCell, value),
		Technique: "bug-plus-one",
		Evidence: &Evidence{
			Premises: premises,
			Units: []UnitRef{
				{Kind: UnitRow, Index: triCell.Row},
				{Kind: UnitColumn, Index: triCell.Column},
				{Kind: UnitBox, Index: boxIndex(*triCell)},
			},
		},
		Reason: fmt.Sprintf(
			"BUG+1: all unsolved cells have 2 candidates except %s with {%s} — %d is the extra candidate and must be placed",
			triCell.ToString(), candidateString(triCands), value,
		),
	}
}

// findBUGExtraCandidate identifies which of the 3 candidates in the trivalue
// cell is the "extra" one by checking candidate frequency in its units.
// In a BUG, every digit appears exactly twice per unit; the extra digit
// appears 3 times in at least one unit.
func findBUGExtraCandidate(board *core.Board, pos core.Position, cands core.CandidateSet) int {
	vals := cands.Values()

	for _, v := range vals {
		// Count appearances of v as a candidate in the row.
		rowCount := 0
		for c := 0; c < 9; c++ {
			p := core.NewPosition(pos.Row, c)
			if board.Get(p) == 0 && board.Candidates(p).Has(v) {
				rowCount++
			}
		}
		if rowCount%2 == 1 {
			return v
		}

		// Count in column.
		colCount := 0
		for r := 0; r < 9; r++ {
			p := core.NewPosition(r, pos.Column)
			if board.Get(p) == 0 && board.Candidates(p).Has(v) {
				colCount++
			}
		}
		if colCount%2 == 1 {
			return v
		}

		// Count in box.
		boxStartRow := (pos.Row / 3) * 3
		boxStartCol := (pos.Column / 3) * 3
		boxCount := 0
		for r := boxStartRow; r < boxStartRow+3; r++ {
			for c := boxStartCol; c < boxStartCol+3; c++ {
				p := core.NewPosition(r, c)
				if board.Get(p) == 0 && board.Candidates(p).Has(v) {
					boxCount++
				}
			}
		}
		if boxCount%2 == 1 {
			return v
		}
	}

	return 0
}

// candidateString formats a CandidateSet as a comma-separated string.
func candidateString(cs core.CandidateSet) string {
	vals := cs.Values()
	s := ""
	for i, v := range vals {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("%d", v)
	}
	return s
}
