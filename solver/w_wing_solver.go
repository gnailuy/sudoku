package solver

import (
	"fmt"
	"sort"

	"github.com/gnailuy/sudoku/core"
)

// WWingSolver finds W-Wing patterns.
//
// A W-Wing involves two bi-value cells with the same candidate pair {X,Y}
// that do NOT share a unit. If there exists a strong link on digit X that
// connects them (i.e., a conjugate pair in some unit where one end sees
// the first cell and the other sees the second cell), then Y can be
// eliminated from any cell that sees both bi-value cells.
//
// Reasoning: If cell A = {X,Y} and cell B = {X,Y}, and they are connected
// by a strong link on X (meaning at least one of the linked cells must be X),
// then:
//   - If the strong link forces X into a cell seen by A, then A = Y.
//   - If the strong link forces X into a cell seen by B, then B = Y.
//   - In either case, at least one of {A,B} must be Y.
//   - Therefore, any cell seeing both A and B cannot be Y.
//
// The solver applies eliminations to the board's elimination layer. It returns
// a placement move when an elimination creates a naked single, or an
// elimination-only move when candidates were reduced without creating a
// placement.
type WWingSolver struct {
	Base
}

// NewWWingSolver creates a WWingSolver and returns it.
func NewWWingSolver() *WWingSolver {
	return &WWingSolver{
		Base: Base{
			Key:         "w-wing",
			DisplayName: "W-Wing",
			Description: "Finds two bi-value cells {X,Y} connected by a strong link on X, enabling elimination of Y from cells that see both.",
			Weight:      WeightWWing,
		},
	}
}

// wwingConjugatePair holds the two ends of a conjugate pair for a digit.
type wwingConjugatePair struct {
	a, b core.Position
}

// Apply checks for W-Wing patterns across all empty cells.
func (s *WWingSolver) Apply(board *core.Board) *Move {
	// Step 1: Collect all bi-value cells grouped by their candidate pair.
	type biValue struct {
		pos  core.Position
		vals [2]int
	}
	type pairKey struct{ a, b int }
	groups := make(map[pairKey][]biValue)

	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			pos := core.NewPosition(row, col)
			if board.Get(pos) != 0 {
				continue
			}
			cands := board.Candidates(pos)
			if cands.Count() == 2 {
				v := cands.Values()
				key := pairKey{v[0], v[1]}
				groups[key] = append(groups[key], biValue{pos: pos, vals: [2]int{v[0], v[1]}})
			}
		}
	}

	// Step 2: Build conjugate pair map — for each digit, find all units
	// where that digit appears as a candidate in exactly 2 cells.
	conjugates := make(map[int][]wwingConjugatePair)
	units := allUnits()
	for _, u := range units {
		for digit := 1; digit <= 9; digit++ {
			var cells []core.Position
			for _, pos := range u.positions {
				if board.Get(pos) == 0 && board.Candidates(pos).Has(digit) {
					cells = append(cells, pos)
				}
			}
			if len(cells) == 2 {
				conjugates[digit] = append(conjugates[digit], wwingConjugatePair{a: cells[0], b: cells[1]})
			}
		}
	}

	// Step 3: For each pair of bi-value cells with the same candidates {X,Y},
	// check if they are connected by a strong link on X or Y.
	groupKeys := make([]pairKey, 0, len(groups))
	for pair := range groups {
		groupKeys = append(groupKeys, pair)
	}
	sort.Slice(groupKeys, func(i, j int) bool {
		if groupKeys[i].a != groupKeys[j].a {
			return groupKeys[i].a < groupKeys[j].a
		}
		return groupKeys[i].b < groupKeys[j].b
	})
	for _, pair := range groupKeys {
		cells := groups[pair]
		if len(cells) < 2 {
			continue
		}
		for i := 0; i < len(cells); i++ {
			for j := i + 1; j < len(cells); j++ {
				cellA := cells[i]
				cellB := cells[j]

				// The two cells must NOT share a unit (otherwise it's a
				// naked pair, not a W-Wing).
				if sharesUnit(cellA.pos, cellB.pos) {
					continue
				}

				// Try connecting via strong link on X (pair.a), eliminating Y (pair.b).
				if move := s.tryWWing(board, cellA.pos, cellB.pos, pair.a, pair.b, conjugates[pair.a]); move != nil {
					return move
				}
				// Try connecting via strong link on Y (pair.b), eliminating X (pair.a).
				if move := s.tryWWing(board, cellA.pos, cellB.pos, pair.b, pair.a, conjugates[pair.b]); move != nil {
					return move
				}
			}
		}
	}

	return nil
}

// tryWWing checks if cellA and cellB (both bi-value with {linkDigit, elimDigit})
// are connected by a strong link on linkDigit, and if so, eliminates elimDigit
// from cells that see both.
func (s *WWingSolver) tryWWing(board *core.Board, cellA, cellB core.Position, linkDigit, elimDigit int, pairs []wwingConjugatePair) *Move {
	for _, cp := range pairs {
		// One end of the conjugate pair must see cellA, the other must see cellB.
		// The conjugate pair cells themselves must not be cellA or cellB.
		var aEnd, bEnd core.Position
		var found bool
		if cp.a != cellA && cp.b != cellB && cp.a != cellB && cp.b != cellA &&
			sharesUnit(cp.a, cellA) && sharesUnit(cp.b, cellB) {
			aEnd = cp.a
			bEnd = cp.b
			found = true
		} else if cp.b != cellA && cp.a != cellB && cp.b != cellB && cp.a != cellA &&
			sharesUnit(cp.b, cellA) && sharesUnit(cp.a, cellB) {
			aEnd = cp.b
			bEnd = cp.a
			found = true
		}
		if !found {
			continue
		}

		// Found a valid W-Wing! Eliminate elimDigit from cells that see both cellA and cellB.
		var move *Move
		eliminated := false
		for row := 0; row < 9; row++ {
			for col := 0; col < 9; col++ {
				pos := core.NewPosition(row, col)
				if pos == cellA || pos == cellB || pos == aEnd || pos == bEnd {
					continue
				}
				if !sharesUnit(pos, cellA) || !sharesUnit(pos, cellB) {
					continue
				}
				if board.EliminateCandidate(pos, elimDigit) {
					eliminated = true
					cands := board.Candidates(pos)
					if cands.Count() == 1 && move == nil {
						value := cands.Values()[0]
						move = &Move{
							Cell:      core.NewCell(pos, value),
							Technique: s.Key,
							Reason: fmt.Sprintf(
								"W-Wing: %s and %s both have {%d,%d}, connected by strong link on %d via %s-%s — %d eliminated from %s, leaving %d",
								cellA.ToString(), cellB.ToString(),
								linkDigit, elimDigit, linkDigit,
								aEnd.ToString(), bEnd.ToString(),
								elimDigit, pos.ToString(), value,
							),
						}
					}
				}
			}
		}

		if eliminated {
			pairValues := []int{linkDigit, elimDigit}
			sort.Ints(pairValues)
			evidence := &Evidence{Premises: []CandidateGroup{
				{Position: cellA, Values: append([]int(nil), pairValues...)},
				{Position: cellB, Values: append([]int(nil), pairValues...)},
				{Position: aEnd, Values: []int{linkDigit}},
				{Position: bEnd, Values: []int{linkDigit}},
			}, Links: []CandidateLink{{
				From: CandidateRef{Position: aEnd, Value: linkDigit},
				To:   CandidateRef{Position: bEnd, Value: linkDigit},
				Kind: LinkStrong,
			}}}
			if move != nil {
				move.Evidence = evidence
				return move
			}
			return &Move{
				EliminationOnly: true,
				Technique:       s.Key,
				Reason: fmt.Sprintf(
					"W-Wing: %s and %s both have {%d,%d}, connected by strong link on %d via %s-%s — %d eliminated from cells seeing both",
					cellA.ToString(), cellB.ToString(),
					linkDigit, elimDigit, linkDigit,
					aEnd.ToString(), bEnd.ToString(),
					elimDigit,
				),
				Evidence: evidence,
			}
		}
	}

	return nil
}
