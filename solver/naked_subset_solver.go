package solver

import (
	"fmt"
	"strings"

	"github.com/gnailuy/sudoku/core"
)

// NakedPairSolver finds naked pairs in rows, columns, and boxes.
//
// A naked pair occurs when two cells in a unit share the same two candidates —
// those candidates can be eliminated from all other cells in the unit.
//
// This solver does not place values directly. Instead, it finds eliminations
// and then checks if any elimination creates a naked single (a cell with one
// candidate remaining), returning that as the move.
type NakedPairSolver struct {
	Base
}

// NewNakedPairSolver creates a NakedPairSolver and returns it.
func NewNakedPairSolver() *NakedPairSolver {
	return &NakedPairSolver{
		Base: Base{
			Key:         "naked-pair",
			DisplayName: "Naked Pair",
			Description: "Finds two cells in a unit sharing the same two candidates, enabling eliminations that reveal a single candidate.",
			Weight:      WeightNakedPair,
		},
	}
}

// Apply scans all units for naked pairs.
func (s *NakedPairSolver) Apply(board *core.Board) *Move {
	units := allUnits()

	for _, u := range units {
		if move := findNakedSubsetOfSize(board, u.positions, u.name, 2, s.Key, "Naked pair"); move != nil {
			return move
		}
	}

	return nil
}

// NakedTripleSolver finds naked triples in rows, columns, and boxes.
//
// A naked triple occurs when three cells in a unit collectively have exactly
// three candidates — those candidates can be eliminated from all other cells
// in the unit.
type NakedTripleSolver struct {
	Base
}

// NewNakedTripleSolver creates a NakedTripleSolver and returns it.
func NewNakedTripleSolver() *NakedTripleSolver {
	return &NakedTripleSolver{
		Base: Base{
			Key:         "naked-triple",
			DisplayName: "Naked Triple",
			Description: "Finds three cells in a unit sharing the same three candidates, enabling eliminations that reveal a single candidate.",
			Weight:      WeightNakedTriple,
		},
	}
}

// Apply scans all units for naked triples.
func (s *NakedTripleSolver) Apply(board *core.Board) *Move {
	units := allUnits()

	for _, u := range units {
		if move := findNakedSubsetOfSize(board, u.positions, u.name, 3, s.Key, "Naked triple"); move != nil {
			return move
		}
	}

	return nil
}

// NakedQuadSolver finds naked quads in rows, columns, and boxes.
//
// A naked quad occurs when four cells in a unit collectively have exactly
// four candidates — those candidates can be eliminated from all other cells
// in the unit.
type NakedQuadSolver struct {
	Base
}

// NewNakedQuadSolver creates a NakedQuadSolver and returns it.
func NewNakedQuadSolver() *NakedQuadSolver {
	return &NakedQuadSolver{
		Base: Base{
			Key:         "naked-quad",
			DisplayName: "Naked Quad",
			Description: "Finds four cells in a unit sharing the same four candidates, enabling eliminations that reveal a single candidate.",
			Weight:      WeightNakedQuad,
		},
	}
}

// Apply scans all units for naked quads.
func (s *NakedQuadSolver) Apply(board *core.Board) *Move {
	units := allUnits()

	for _, u := range units {
		if move := findNakedSubsetOfSize(board, u.positions, u.name, 4, s.Key, "Naked quad"); move != nil {
			return move
		}
	}

	return nil
}

// nakedSubsetCell holds an empty cell's position and its candidate set.
type nakedSubsetCell struct {
	pos        core.Position
	candidates core.CandidateSet
}

// findNakedSubsetOfSize is the shared logic for naked pair/triple/quad solvers.
// It searches a single unit for naked subsets of the given size.
func findNakedSubsetOfSize(board *core.Board, positions []core.Position, unitName string, size int, technique string, displayName string) *Move {
	// Collect empty cells and their candidates.
	var emptyCells []nakedSubsetCell
	for _, pos := range positions {
		cands := board.Candidates(pos)
		if !cands.IsEmpty() {
			emptyCells = append(emptyCells, nakedSubsetCell{pos: pos, candidates: cands})
		}
	}

	// Filter cells with at most 'size' candidates.
	var eligible []nakedSubsetCell
	for _, c := range emptyCells {
		if c.candidates.Count() <= size {
			eligible = append(eligible, c)
		}
	}

	if len(eligible) < size {
		return nil
	}

	// Generate combinations of 'size' cells from eligible.
	indices := make([]int, size)
	return nakedSubsetCombinationsSearch(eligible, positions, unitName, size, indices, 0, 0, board, technique, displayName)
}

// nakedSubsetCombinationsSearch recursively generates combinations and checks for naked subsets.
func nakedSubsetCombinationsSearch(eligible []nakedSubsetCell, allPositions []core.Position, unitName string, size int, indices []int, start int, depth int, board *core.Board, technique string, displayName string) *Move {
	if depth == size {
		// Compute the union of candidates in the selected cells.
		var union core.CandidateSet
		for i := 0; i < size; i++ {
			union |= eligible[indices[i]].candidates
		}

		// A naked subset: 'size' cells whose combined candidates have exactly 'size' values.
		if union.Count() != size {
			return nil
		}

		// Build a set of positions in the subset for quick lookup.
		subsetPositions := make(map[core.Position]bool, size)
		for i := 0; i < size; i++ {
			subsetPositions[eligible[indices[i]].pos] = true
		}

		// Check if any non-subset cell in the unit has candidates that can be eliminated.
		unionVals := union.Values()
		for _, pos := range allPositions {
			if subsetPositions[pos] || board.Get(pos) != 0 {
				continue
			}

			cands := board.Candidates(pos)
			// Check if this cell has any of the subset's candidates.
			hasOverlap := false
			for _, v := range unionVals {
				if cands.Has(v) {
					hasOverlap = true
					break
				}
			}
			if !hasOverlap {
				continue
			}

			// Elimination possible. Simulate the elimination to see if it
			// creates a naked single.
			reduced := cands
			for _, rv := range unionVals {
				reduced.Remove(rv)
			}

			if reduced.Count() == 1 {
				value := reduced.Values()[0]
				cellNames := nakedSubsetCellNames(eligible, indices, size)
				valNames := nakedSubsetValNames(unionVals)
				return &Move{
					Cell:      core.NewCell(pos, value),
					Technique: technique,
					Reason: fmt.Sprintf(
						"%s {%s} in %s at {%s} eliminates candidates, leaving %d as the only candidate for %s",
						displayName, valNames, unitName, cellNames, value, pos.ToString(),
					),
				}
			}
		}

		return nil
	}

	for i := start; i < len(eligible); i++ {
		indices[depth] = i
		if move := nakedSubsetCombinationsSearch(eligible, allPositions, unitName, size, indices, i+1, depth+1, board, technique, displayName); move != nil {
			return move
		}
	}

	return nil
}

// nakedSubsetCellNames formats the positions of the subset cells.
func nakedSubsetCellNames(eligible []nakedSubsetCell, indices []int, size int) string {
	names := make([]string, size)
	for i := 0; i < size; i++ {
		names[i] = eligible[indices[i]].pos.ToString()
	}
	return strings.Join(names, ", ")
}

// nakedSubsetValNames formats a slice of values as a comma-separated string.
func nakedSubsetValNames(vals []int) string {
	names := make([]string, len(vals))
	for i, v := range vals {
		names[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(names, ", ")
}

// Hint returns the first naked-pair elimination as a teaching deduction without
// requiring that the elimination immediately creates a placement.
func (s *NakedPairSolver) Hint(board *core.Board) *Move {
	for _, u := range allUnits() {
		if move := findNakedPairElimination(board, u.positions, u.name, s.Key); move != nil {
			return move
		}
	}
	return nil
}

func findNakedPairElimination(board *core.Board, positions []core.Position, unitName, technique string) *Move {
	var eligible []nakedSubsetCell
	for _, position := range positions {
		candidates := board.Candidates(position)
		if candidates.Count() == 2 {
			eligible = append(eligible, nakedSubsetCell{pos: position, candidates: candidates})
		}
	}
	for left := 0; left < len(eligible); left++ {
		for right := left + 1; right < len(eligible); right++ {
			if eligible[left].candidates != eligible[right].candidates {
				continue
			}
			pair := eligible[left].candidates.Values()
			var eliminations []CandidateRef
			for _, position := range positions {
				if position == eligible[left].pos || position == eligible[right].pos || board.Get(position) != 0 {
					continue
				}
				candidates := board.Candidates(position)
				for _, value := range pair {
					if candidates.Has(value) {
						eliminations = append(eliminations, CandidateRef{Position: position, Value: value})
					}
				}
			}
			if len(eliminations) == 0 {
				continue
			}
			return &Move{
				Technique: technique,
				Reason: fmt.Sprintf("Naked pair {%s} in %s at {%s, %s} eliminates candidates from the other cells",
					nakedSubsetValNames(pair), unitName, eligible[left].pos.ToString(), eligible[right].pos.ToString()),
				EliminationOnly: true,
				Evidence: &Evidence{
					Unit: parseNakedSubsetUnit(unitName),
					Premises: []CandidateGroup{
						{Position: eligible[left].pos, Values: pair},
						{Position: eligible[right].pos, Values: pair},
					},
					Eliminations: eliminations,
				},
			}
		}
	}
	return nil
}

func parseNakedSubsetUnit(name string) *UnitRef {
	var kind string
	var number int
	if _, err := fmt.Sscanf(name, "%s %d", &kind, &number); err != nil || number < 1 || number > 9 {
		return nil
	}
	unit := UnitRef{Kind: UnitKind(kind), Index: number - 1}
	return &unit
}
