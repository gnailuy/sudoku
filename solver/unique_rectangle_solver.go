package solver

import (
	"fmt"
	"sort"

	"github.com/gnailuy/sudoku/core"
)

// UniqueRectangleSolver detects Unique Rectangle Type 1 patterns.
//
// A Unique Rectangle (also called a Deadly Pattern) arises when four cells
// form a rectangle across exactly two rows, two columns, and two boxes.
// If all four cells contained the same two candidates {A,B}, the puzzle
// would have two solutions (swapping A and B) — violating uniqueness.
//
// Type 1: Three of the four rectangle cells have exactly the candidates
// {A,B} (bivalue with the same pair). The fourth cell has {A,B} plus
// additional candidates. Since {A,B} alone in all four cells would
// create a deadly pattern, the fourth cell cannot be A or B — its value
// must be one of the extra candidates. We eliminate A and B from it.
//
// If the elimination leaves the cell with a single candidate, we return
// a placement move. Otherwise, we return an elimination-only move.
type UniqueRectangleSolver struct {
	Base
}

// NewUniqueRectangleSolver creates a UniqueRectangleSolver.
func NewUniqueRectangleSolver() *UniqueRectangleSolver {
	return &UniqueRectangleSolver{
		Base: Base{
			Key:         "unique-rectangle",
			DisplayName: "Unique Rectangle Type 1",
			Description: "Eliminates candidates that would create a deadly pattern (two-solution rectangle) in valid puzzles.",
			Weight:      WeightUniqueRectangle,
		},
	}
}

// Apply scans for Unique Rectangle Type 1 patterns and returns a move if found.
func (s *UniqueRectangleSolver) Apply(board *core.Board) *Move {
	// Collect all empty cells with exactly 2 candidates, grouped by their candidate pair.
	type pairKey struct{ a, b int }
	bivalueCells := make(map[pairKey][]core.Position)

	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			pos := core.NewPosition(r, c)
			if board.Get(pos) != 0 {
				continue
			}
			cands := board.Candidates(pos)
			if cands.Count() == 2 {
				vals := cands.Values()
				key := pairKey{vals[0], vals[1]}
				bivalueCells[key] = append(bivalueCells[key], pos)
			}
		}
	}

	// For each pair, check if 3 bivalue cells can form a rectangle with a 4th cell.
	pairKeys := make([]pairKey, 0, len(bivalueCells))
	for pair := range bivalueCells {
		pairKeys = append(pairKeys, pair)
	}
	sort.Slice(pairKeys, func(i, j int) bool {
		if pairKeys[i].a != pairKeys[j].a {
			return pairKeys[i].a < pairKeys[j].a
		}
		return pairKeys[i].b < pairKeys[j].b
	})
	for _, pair := range pairKeys {
		cells := bivalueCells[pair]
		if len(cells) < 3 {
			continue
		}

		// Try all combinations of 3 cells with this pair.
		for i := 0; i < len(cells); i++ {
			for j := i + 1; j < len(cells); j++ {
				for k := j + 1; k < len(cells); k++ {
					three := [3]core.Position{cells[i], cells[j], cells[k]}
					move := s.checkTriple(board, three, pair.a, pair.b)
					if move != nil {
						return move
					}
				}
			}
		}
	}

	return nil
}

// checkTriple tests whether three bivalue cells {a,b} can form three corners
// of a rectangle, and whether the fourth corner is a valid Type 1 target.
func (s *UniqueRectangleSolver) checkTriple(board *core.Board, three [3]core.Position, a, b int) *Move {
	// Collect the row/column pairs.
	rows := make(map[int][]int) // row -> list of column indices in 'three'
	cols := make(map[int][]int) // col -> list of row indices in 'three'
	for _, pos := range three {
		rows[pos.Row] = append(rows[pos.Row], pos.Column)
		cols[pos.Column] = append(cols[pos.Column], pos.Row)
	}

	// A rectangle requires exactly 2 rows and 2 columns.
	if len(rows) != 2 || len(cols) != 2 {
		return nil
	}

	// Extract the two rows and two columns.
	rowKeys := make([]int, 0, 2)
	for r := range rows {
		rowKeys = append(rowKeys, r)
	}
	colKeys := make([]int, 0, 2)
	for c := range cols {
		colKeys = append(colKeys, c)
	}
	sort.Ints(rowKeys)
	sort.Ints(colKeys)

	// Find the missing corner (the intersection not in 'three').
	threeSet := make(map[[2]int]bool)
	for _, pos := range three {
		threeSet[[2]int{pos.Row, pos.Column}] = true
	}

	var fourthPos core.Position
	found := false
	for _, r := range rowKeys {
		for _, c := range colKeys {
			if !threeSet[[2]int{r, c}] {
				fourthPos = core.NewPosition(r, c)
				found = true
			}
		}
	}
	if !found {
		return nil
	}

	// The fourth cell must be empty and contain both a and b as candidates.
	if board.Get(fourthPos) != 0 {
		return nil
	}
	fourthCands := board.Candidates(fourthPos)
	if !fourthCands.Has(a) || !fourthCands.Has(b) {
		return nil
	}

	// Must have extra candidates beyond {a,b} — otherwise it's already a deadly pattern.
	if fourthCands.Count() <= 2 {
		return nil
	}

	// Verify the rectangle spans exactly 2 boxes. All four corners must
	// cover exactly 2 boxes for the Unique Rectangle to be valid.
	boxSet := make(map[int]bool)
	for _, pos := range three {
		boxSet[boxIndex(pos)] = true
	}
	boxSet[boxIndex(fourthPos)] = true
	if len(boxSet) != 2 {
		return nil
	}

	// Preserve the four-corner facts before applying the conclusion.
	premises := rectanglePremises(board, append(three[:], fourthPos)...)

	// Eliminate a and b from the fourth cell.
	elimA := board.EliminateCandidate(fourthPos, a)
	elimB := board.EliminateCandidate(fourthPos, b)
	if !elimA && !elimB {
		return nil
	}

	// Check if the fourth cell now has a single candidate.
	newCands := board.Candidates(fourthPos)
	evidence := &Evidence{
		Premises: premises,
		Eliminations: []CandidateRef{
			{Position: fourthPos, Value: a},
			{Position: fourthPos, Value: b},
		},
	}
	if newCands.Count() == 1 {
		value := newCands.Values()[0]
		return &Move{
			Cell:      core.NewCell(fourthPos, value),
			Technique: "unique-rectangle",
			Evidence:  evidence,
			Reason: fmt.Sprintf(
				"Unique Rectangle Type 1: cells %s, %s, %s all have {%d,%d} — "+
					"%s can't be %d or %d (deadly pattern), leaving %d",
				three[0].ToString(), three[1].ToString(), three[2].ToString(),
				a, b, fourthPos.ToString(), a, b, value,
			),
		}
	}

	return &Move{
		EliminationOnly: true,
		Technique:       "unique-rectangle",
		Evidence:        evidence,
		Reason: fmt.Sprintf(
			"Unique Rectangle Type 1: cells %s, %s, %s all have {%d,%d} — "+
				"eliminated %d,%d from %s (would create deadly pattern)",
			three[0].ToString(), three[1].ToString(), three[2].ToString(),
			a, b, a, b, fourthPos.ToString(),
		),
	}
}

func rectanglePremises(board *core.Board, positions ...core.Position) []CandidateGroup {
	premises := make([]CandidateGroup, 0, len(positions))
	for _, position := range positions {
		premises = append(premises, CandidateGroup{Position: position, Values: board.Candidates(position).Values()})
	}
	return premises
}

// boxIndex returns the box number (0-8) for a given position.
func boxIndex(pos core.Position) int {
	return (pos.Row/3)*3 + pos.Column/3
}
