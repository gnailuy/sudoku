package solver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gnailuy/sudoku/core"
)

// HiddenPairSolver finds hidden pairs in rows, columns, and boxes.
//
// A hidden pair occurs when two candidates appear in only two cells within a
// unit. All other candidates can be eliminated from those two cells.
//
// Hidden subsets are the complement of naked subsets: naked subsets find cells
// whose candidates are restricted to a set, while hidden subsets find
// candidates whose positions are restricted to a set of cells.
//
// The solver applies eliminations to the board's elimination layer. It returns
// a placement move when an elimination creates a naked single, or an
// elimination-only move when candidates were reduced without creating a
// placement.
type HiddenPairSolver struct {
	Base
}

// NewHiddenPairSolver creates a HiddenPairSolver and returns it.
func NewHiddenPairSolver() *HiddenPairSolver {
	return &HiddenPairSolver{
		Base: Base{
			Key:         "hidden-pair",
			DisplayName: "Hidden Pair",
			Description: "Finds two candidates confined to the same two cells in a unit, enabling eliminations that reveal a single candidate.",
			Weight:      WeightHiddenPair,
		},
	}
}

// Apply scans all units for hidden pairs.
func (s *HiddenPairSolver) Apply(board *core.Board) *Move {
	units := allUnits()

	for _, u := range units {
		if move := findHiddenSubsetOfSize(board, u.positions, u.name, 2, s.Key, "Hidden pair"); move != nil {
			return move
		}
	}

	return nil
}

// HiddenTripleSolver finds hidden triples in rows, columns, and boxes.
//
// A hidden triple occurs when three candidates appear in only three cells
// within a unit. All other candidates can be eliminated from those three
// cells.
type HiddenTripleSolver struct {
	Base
}

// NewHiddenTripleSolver creates a HiddenTripleSolver and returns it.
func NewHiddenTripleSolver() *HiddenTripleSolver {
	return &HiddenTripleSolver{
		Base: Base{
			Key:         "hidden-triple",
			DisplayName: "Hidden Triple",
			Description: "Finds three candidates confined to the same three cells in a unit, enabling eliminations that reveal a single candidate.",
			Weight:      WeightHiddenTriple,
		},
	}
}

// Apply scans all units for hidden triples.
func (s *HiddenTripleSolver) Apply(board *core.Board) *Move {
	units := allUnits()

	for _, u := range units {
		if move := findHiddenSubsetOfSize(board, u.positions, u.name, 3, s.Key, "Hidden triple"); move != nil {
			return move
		}
	}

	return nil
}

// HiddenQuadSolver finds hidden quads in rows, columns, and boxes.
//
// A hidden quad occurs when four candidates appear in only four cells within
// a unit. All other candidates can be eliminated from those four cells.
type HiddenQuadSolver struct {
	Base
}

// NewHiddenQuadSolver creates a HiddenQuadSolver and returns it.
func NewHiddenQuadSolver() *HiddenQuadSolver {
	return &HiddenQuadSolver{
		Base: Base{
			Key:         "hidden-quad",
			DisplayName: "Hidden Quad",
			Description: "Finds four candidates confined to the same four cells in a unit, enabling eliminations that reveal a single candidate.",
			Weight:      WeightHiddenQuad,
		},
	}
}

// Apply scans all units for hidden quads.
func (s *HiddenQuadSolver) Apply(board *core.Board) *Move {
	units := allUnits()

	for _, u := range units {
		if move := findHiddenSubsetOfSize(board, u.positions, u.name, 4, s.Key, "Hidden quad"); move != nil {
			return move
		}
	}

	return nil
}

// hiddenDigitPositions maps a digit to the positions of empty cells that
// contain it as a candidate within a unit.
type hiddenDigitPositions struct {
	digit     int
	positions []core.Position
}

// findHiddenSubsetOfSize is the shared logic for hidden pair/triple/quad solvers.
// It searches a single unit for hidden subsets of the given size.
func findHiddenSubsetOfSize(board *core.Board, positions []core.Position, unitName string, size int, technique string, displayName string) *Move {
	// Build a map of digit → positions of empty cells containing that digit.
	var digits []hiddenDigitPositions
	for d := 1; d <= 9; d++ {
		var cells []core.Position
		for _, pos := range positions {
			if board.Get(pos) == 0 && board.Candidates(pos).Has(d) {
				cells = append(cells, pos)
			}
		}
		// Digits that appear in 2–size cells are candidates for hidden subsets.
		if len(cells) >= 2 && len(cells) <= size {
			digits = append(digits, hiddenDigitPositions{digit: d, positions: cells})
		}
	}

	if len(digits) < size {
		return nil
	}

	// Generate combinations of 'size' digits from the eligible digits.
	indices := make([]int, size)
	return hiddenSubsetCombinationsSearch(board, digits, unitName, size, indices, 0, 0, technique, displayName)
}

// hiddenSubsetCombinationsSearch recursively generates combinations and checks for hidden subsets.
func hiddenSubsetCombinationsSearch(board *core.Board, digits []hiddenDigitPositions, unitName string, size int, indices []int, start int, depth int, technique string, displayName string) *Move {
	if depth == size {
		// Compute the union of positions for the selected digits.
		posSet := make(map[core.Position]bool)
		for i := 0; i < size; i++ {
			for _, pos := range digits[indices[i]].positions {
				posSet[pos] = true
			}
		}

		// A hidden subset: 'size' digits whose combined positions have exactly 'size' cells.
		if len(posSet) != size {
			return nil
		}

		// Build the set of digits in the hidden subset and preserve the exact
		// cells/digits as renderer-neutral teaching evidence.
		hiddenDigits := make(map[int]bool, size)
		hiddenValues := make([]int, size)
		for i := 0; i < size; i++ {
			value := digits[indices[i]].digit
			hiddenDigits[value] = true
			hiddenValues[i] = value
		}
		sort.Ints(hiddenValues)
		positions := make([]core.Position, 0, len(posSet))
		for position := range posSet {
			positions = append(positions, position)
		}
		sort.Slice(positions, func(i, j int) bool {
			if positions[i].Row != positions[j].Row {
				return positions[i].Row < positions[j].Row
			}
			return positions[i].Column < positions[j].Column
		})
		premises := make([]CandidateGroup, len(positions))
		for i, position := range positions {
			premises[i] = CandidateGroup{Position: position, Values: append([]int(nil), hiddenValues...)}
		}
		evidence := &Evidence{Unit: parseNakedSubsetUnit(unitName), Premises: premises}

		// For each cell in the subset, eliminate non-hidden candidates.
		var move *Move
		eliminated := false
		for pos := range posSet {
			cands := board.Candidates(pos)
			for _, v := range cands.Values() {
				if !hiddenDigits[v] {
					if board.EliminateCandidate(pos, v) {
						eliminated = true
					}
				}
			}
			// Check if the cell is now a naked single after eliminations.
			newCands := board.Candidates(pos)
			if newCands.Count() == 1 && move == nil {
				value := newCands.Values()[0]
				digitNames := hiddenSubsetDigitNames(digits, indices, size)
				cellNames := hiddenSubsetCellNames(posSet)
				move = &Move{
					Cell:      core.NewCell(pos, value),
					Technique: technique,
					Reason: fmt.Sprintf(
						"%s {%s} in %s confined to {%s}, leaving %d as the only candidate for %s",
						displayName, digitNames, unitName, cellNames, value, pos.ToString(),
					),
				}
			}
		}

		if eliminated {
			if move != nil {
				move.Evidence = evidence
				return move
			}
			// Eliminations applied but no naked single yet.
			digitNames := hiddenSubsetDigitNames(digits, indices, size)
			cellNames := hiddenSubsetCellNames(posSet)
			return &Move{
				EliminationOnly: true,
				Technique:       technique,
				Reason: fmt.Sprintf(
					"%s {%s} in %s confined to {%s} — candidates eliminated",
					displayName, digitNames, unitName, cellNames,
				),
				Evidence: evidence,
			}
		}

		return nil
	}

	for i := start; i < len(digits); i++ {
		indices[depth] = i
		if move := hiddenSubsetCombinationsSearch(board, digits, unitName, size, indices, i+1, depth+1, technique, displayName); move != nil {
			return move
		}
	}

	return nil
}

// hiddenSubsetDigitNames formats the digits of the hidden subset.
func hiddenSubsetDigitNames(digits []hiddenDigitPositions, indices []int, size int) string {
	names := make([]string, size)
	for i := 0; i < size; i++ {
		names[i] = fmt.Sprintf("%d", digits[indices[i]].digit)
	}
	return strings.Join(names, ", ")
}

// hiddenSubsetCellNames formats the positions of the subset cells (sorted for deterministic output).
func hiddenSubsetCellNames(posSet map[core.Position]bool) string {
	names := make([]string, 0, len(posSet))
	for pos := range posSet {
		names = append(names, pos.ToString())
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
