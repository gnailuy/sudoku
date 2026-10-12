package solver

import (
	"fmt"

	"github.com/gnailuy/sudoku/core"
)

// XYWingSolver finds XY-Wing patterns.
//
// An XY-Wing is a three-cell pattern:
//   - Pivot cell has exactly two candidates {X, Y}.
//   - Wing 1 shares a unit with the pivot and has exactly two candidates {X, Z}.
//   - Wing 2 shares a unit with the pivot and has exactly two candidates {Y, Z}.
//   - The two wings do NOT need to share a unit with each other.
//
// Any cell that can see both wings can eliminate Z, because exactly one wing
// must contain Z — the pivot forces one wing to lose its shared candidate,
// making the other wing's Z the only option. So Z cannot appear in cells
// that see both wings.
//
// The solver applies eliminations to the board's elimination layer. It returns
// a placement move when an elimination creates a naked single, or an
// elimination-only move when candidates were reduced without creating a
// placement.
type XYWingSolver struct {
	Base
}

// NewXYWingSolver creates an XYWingSolver and returns it.
func NewXYWingSolver() *XYWingSolver {
	return &XYWingSolver{
		Base: Base{
			Key:         "xy-wing",
			DisplayName: "XY-Wing",
			Description: "Finds a pivot-wing pattern where a pivot with {X,Y} connects to wings with {X,Z} and {Y,Z}, enabling elimination of Z from cells that see both wings.",
			Weight:      WeightXYWing,
		},
	}
}

// Apply checks for XY-Wing patterns across all empty cells.
func (s *XYWingSolver) Apply(board *core.Board) *Move {
	// Collect all bi-value cells (cells with exactly 2 candidates).
	type biValue struct {
		pos  core.Position
		vals [2]int
	}
	var biCells []biValue

	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			pos := core.NewPosition(row, col)
			if board.Get(pos) != 0 {
				continue
			}
			cands := board.Candidates(pos)
			if cands.Count() == 2 {
				v := cands.Values()
				biCells = append(biCells, biValue{pos: pos, vals: [2]int{v[0], v[1]}})
			}
		}
	}

	// For each bi-value cell as pivot {X, Y}:
	for _, pivot := range biCells {
		x, y := pivot.vals[0], pivot.vals[1]

		// Find wing candidates: bi-value cells that share a unit with pivot.
		var xWings, yWings []biValue // xWings have {X, Z}, yWings have {Y, Z}
		for _, cell := range biCells {
			if cell.pos == pivot.pos {
				continue
			}
			if !sharesUnit(pivot.pos, cell.pos) {
				continue
			}
			a, b := cell.vals[0], cell.vals[1]
			// Wing 1 pattern: has X but not Y → {X, Z} where Z = the other value.
			if (a == x || b == x) && a != y && b != y {
				xWings = append(xWings, cell)
			}
			// Wing 2 pattern: has Y but not X → {Y, Z} where Z = the other value.
			if (a == y || b == y) && a != x && b != x {
				yWings = append(yWings, cell)
			}
		}

		// Try all combinations of xWing and yWing.
		for _, w1 := range xWings {
			// Z from wing1: the non-X value.
			z1 := w1.vals[0]
			if z1 == x {
				z1 = w1.vals[1]
			}

			for _, w2 := range yWings {
				// Z from wing2: the non-Y value.
				z2 := w2.vals[0]
				if z2 == y {
					z2 = w2.vals[1]
				}

				// Both wings must eliminate the same digit Z.
				if z1 != z2 {
					continue
				}
				z := z1

				// The two wings must be different cells.
				if w1.pos == w2.pos {
					continue
				}

				// Eliminate Z from all cells that can see both wings.
				var move *Move
				eliminated := false
				for row := 0; row < 9; row++ {
					for col := 0; col < 9; col++ {
						pos := core.NewPosition(row, col)
						if pos == pivot.pos || pos == w1.pos || pos == w2.pos {
							continue
						}
						if !sharesUnit(pos, w1.pos) || !sharesUnit(pos, w2.pos) {
							continue
						}
						if board.EliminateCandidate(pos, z) {
							eliminated = true
							cands := board.Candidates(pos)
							if cands.Count() == 1 && move == nil {
								value := cands.Values()[0]
								move = &Move{
									Cell:      core.NewCell(pos, value),
									Technique: s.Key,
									Reason: fmt.Sprintf(
										"XY-Wing: pivot %s {%d,%d}, wings %s {%d,%d} and %s {%d,%d} — %d eliminated from %s, leaving %d",
										pivot.pos.ToString(), x, y,
										w1.pos.ToString(), w1.vals[0], w1.vals[1],
										w2.pos.ToString(), w2.vals[0], w2.vals[1],
										z, pos.ToString(), value,
									),
								}
							}
						}
					}
				}

				if eliminated {
					evidence := &Evidence{Premises: []CandidateGroup{
						{Position: pivot.pos, Values: []int{x, y}},
						{Position: w1.pos, Values: []int{x, z}},
						{Position: w2.pos, Values: []int{y, z}},
					}}
					if move != nil {
						move.Evidence = evidence
						return move
					}
					return &Move{
						EliminationOnly: true,
						Technique:       s.Key,
						Reason: fmt.Sprintf(
							"XY-Wing: pivot %s {%d,%d}, wings %s {%d,%d} and %s {%d,%d} — %d eliminated from cells seeing both wings",
							pivot.pos.ToString(), x, y,
							w1.pos.ToString(), w1.vals[0], w1.vals[1],
							w2.pos.ToString(), w2.vals[0], w2.vals[1],
							z,
						),
						Evidence: evidence,
					}
				}
			}
		}
	}

	return nil
}

// sharesUnit reports whether two positions share a row, column, or box.
func sharesUnit(a, b core.Position) bool {
	if a.Row == b.Row {
		return true
	}
	if a.Column == b.Column {
		return true
	}
	// Same box: both in the same 3×3 region.
	if a.Row/3 == b.Row/3 && a.Column/3 == b.Column/3 {
		return true
	}
	return false
}
