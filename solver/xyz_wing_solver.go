package solver

import (
	"fmt"

	"github.com/gnailuy/sudoku/core"
)

// XYZWingSolver finds XYZ-Wing patterns.
//
// An XYZ-Wing is a three-cell pattern:
//   - Pivot cell has exactly three candidates {X, Y, Z}.
//   - Wing 1 shares a unit with the pivot and has exactly two candidates {X, Z}.
//   - Wing 2 shares a unit with the pivot and has exactly two candidates {Y, Z}.
//   - (Wings may or may not share a unit with each other.)
//
// The shared candidate Z appears in the pivot and both wings. Since one of
// the three cells must contain Z, any cell that can see all three cells
// (pivot + both wings) can eliminate Z.
//
// Note: Unlike XY-Wing where eliminations occur in cells seeing both wings,
// XYZ-Wing eliminations require seeing ALL THREE cells (pivot + both wings).
// This is because the pivot itself contains Z as a candidate.
//
// The solver applies eliminations to the board's elimination layer. It returns
// a placement move when an elimination creates a naked single, or an
// elimination-only move when candidates were reduced without creating a
// placement.
type XYZWingSolver struct {
	Base
}

// NewXYZWingSolver creates an XYZWingSolver and returns it.
func NewXYZWingSolver() *XYZWingSolver {
	return &XYZWingSolver{
		Base: Base{
			Key:         "xyz-wing",
			DisplayName: "XYZ-Wing",
			Description: "Finds a pivot with {X,Y,Z} connected to wings with {X,Z} and {Y,Z}, enabling elimination of Z from cells that see all three.",
			Weight:      WeightXYZWing,
		},
	}
}

// xyzBiCell holds a bi-value cell position and its two candidates.
type xyzBiCell struct {
	pos  core.Position
	vals [2]int
}

// Apply checks for XYZ-Wing patterns across all empty cells.
func (s *XYZWingSolver) Apply(board *core.Board) *Move {
	// Collect bi-value cells (potential wings).
	var biCells []xyzBiCell

	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			pos := core.NewPosition(row, col)
			if board.Get(pos) != 0 {
				continue
			}
			cands := board.Candidates(pos)
			if cands.Count() == 2 {
				v := cands.Values()
				biCells = append(biCells, xyzBiCell{pos: pos, vals: [2]int{v[0], v[1]}})
			}
		}
	}

	// For each empty cell with exactly 3 candidates as potential pivot:
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			pivotPos := core.NewPosition(row, col)
			if board.Get(pivotPos) != 0 {
				continue
			}
			cands := board.Candidates(pivotPos)
			if cands.Count() != 3 {
				continue
			}
			vals := cands.Values()
			x, y, z := vals[0], vals[1], vals[2]

			// Try all 3 possible assignments of which candidate is the shared Z.
			for _, combo := range [][3]int{
				{x, y, z}, // shared = z, wing1 = {x,z}, wing2 = {y,z}
				{x, z, y}, // shared = y, wing1 = {x,y}, wing2 = {z,y}
				{y, z, x}, // shared = x, wing1 = {y,x}, wing2 = {z,x}
			} {
				ca, cb, shared := combo[0], combo[1], combo[2]
				if move := s.tryXYZWing(board, pivotPos, ca, cb, shared, biCells); move != nil {
					return move
				}
			}
		}
	}

	return nil
}

// tryXYZWing searches for wings having {ca, shared} and {cb, shared} that
// share a unit with the pivot, and eliminates shared from cells seeing all three.
func (s *XYZWingSolver) tryXYZWing(board *core.Board, pivotPos core.Position, ca, cb, shared int, biCells []xyzBiCell) *Move {
	// Find wing1 candidates: bi-value cells sharing a unit with pivot, having {ca, shared}.
	var wing1s []core.Position
	for _, bc := range biCells {
		if bc.pos == pivotPos || !sharesUnit(pivotPos, bc.pos) {
			continue
		}
		if (bc.vals[0] == ca && bc.vals[1] == shared) || (bc.vals[0] == shared && bc.vals[1] == ca) {
			wing1s = append(wing1s, bc.pos)
		}
	}

	// Find wing2 candidates: bi-value cells sharing a unit with pivot, having {cb, shared}.
	var wing2s []core.Position
	for _, bc := range biCells {
		if bc.pos == pivotPos || !sharesUnit(pivotPos, bc.pos) {
			continue
		}
		if (bc.vals[0] == cb && bc.vals[1] == shared) || (bc.vals[0] == shared && bc.vals[1] == cb) {
			wing2s = append(wing2s, bc.pos)
		}
	}

	// Try all combinations of wing1 and wing2.
	for _, w1 := range wing1s {
		for _, w2 := range wing2s {
			if w1 == w2 {
				continue
			}

			// Eliminate shared from cells that see all three: pivot, w1, w2.
			var move *Move
			eliminated := false
			for row := 0; row < 9; row++ {
				for col := 0; col < 9; col++ {
					pos := core.NewPosition(row, col)
					if pos == pivotPos || pos == w1 || pos == w2 {
						continue
					}
					if !sharesUnit(pos, pivotPos) || !sharesUnit(pos, w1) || !sharesUnit(pos, w2) {
						continue
					}
					if board.EliminateCandidate(pos, shared) {
						eliminated = true
						cands := board.Candidates(pos)
						if cands.Count() == 1 && move == nil {
							value := cands.Values()[0]
							move = &Move{
								Cell:      core.NewCell(pos, value),
								Technique: s.Key,
								Reason: fmt.Sprintf(
									"XYZ-Wing: pivot %s {%d,%d,%d}, wings %s {%d,%d} and %s {%d,%d} — %d eliminated from %s, leaving %d",
									pivotPos.ToString(), ca, cb, shared,
									w1.ToString(), ca, shared,
									w2.ToString(), cb, shared,
									shared, pos.ToString(), value,
								),
							}
						}
					}
				}
			}

			if eliminated {
				evidence := &Evidence{Premises: []CandidateGroup{
					{Position: pivotPos, Values: []int{ca, cb, shared}},
					{Position: w1, Values: []int{ca, shared}},
					{Position: w2, Values: []int{cb, shared}},
				}}
				if move != nil {
					move.Evidence = evidence
					return move
				}
				return &Move{
					EliminationOnly: true,
					Technique:       s.Key,
					Reason: fmt.Sprintf(
						"XYZ-Wing: pivot %s {%d,%d,%d}, wings %s {%d,%d} and %s {%d,%d} — %d eliminated from cells seeing all three",
						pivotPos.ToString(), ca, cb, shared,
						w1.ToString(), ca, shared,
						w2.ToString(), cb, shared,
						shared,
					),
					Evidence: evidence,
				}
			}
		}
	}

	return nil
}
