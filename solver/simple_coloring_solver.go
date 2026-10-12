package solver

import (
	"fmt"
	"sort"

	"github.com/gnailuy/sudoku/core"
)

// SimpleColoringSolver finds eliminations using conjugate pair chains.
//
// A conjugate pair for a digit in a unit occurs when that digit appears as
// a candidate in exactly two cells of the unit. These pairs form chains:
// if cell A and B are conjugate, and B and C are conjugate (in a different
// unit), then A-B-C forms a chain. By assigning alternating colors (A=blue,
// B=green, C=blue, ...), we know that one complete color group is true and
// the other is false.
//
// Two elimination rules apply:
//
// Rule 1 (Color Twice in Unit): If two cells of the same color share a
// unit, that color is impossible — all cells of that color can have the
// digit eliminated. Cells of the opposite color are then placed.
//
// Rule 2 (Color Sees Both): If an uncolored cell can see cells of both
// colors, the digit can be eliminated from that cell — regardless of
// which color is true, one of the two colored cells it sees will contain
// the digit.
//
// The solver applies eliminations to the board's elimination layer. It returns
// a placement move when an elimination creates a naked single, or an
// elimination-only move when candidates were reduced without creating a
// placement.
type SimpleColoringSolver struct {
	Base
}

// NewSimpleColoringSolver creates a SimpleColoringSolver and returns it.
func NewSimpleColoringSolver() *SimpleColoringSolver {
	return &SimpleColoringSolver{
		Base: Base{
			Key:         "simple-coloring",
			DisplayName: "Simple Coloring",
			Description: "Tracks conjugate pair chains for a digit, assigning alternating colors. Eliminates the digit from cells that see both colors or from an invalid color group.",
			Weight:      WeightSimpleColoring,
		},
	}
}

// Apply checks for simple coloring patterns on all digits.
func (s *SimpleColoringSolver) Apply(board *core.Board) *Move {
	for digit := 1; digit <= 9; digit++ {
		if move := s.colorDigit(board, digit); move != nil {
			return move
		}
	}
	return nil
}

// colorDigit builds conjugate pair chains for a single digit and applies
// elimination rules.
func (s *SimpleColoringSolver) colorDigit(board *core.Board, digit int) *Move {
	// Build conjugate pairs: for each unit, find pairs of cells where the
	// digit appears as a candidate in exactly two cells.
	type pair struct {
		a, b core.Position
	}
	var pairs []pair

	units := allUnits()
	for _, u := range units {
		var cells []core.Position
		for _, pos := range u.positions {
			if board.Get(pos) == 0 && board.Candidates(pos).Has(digit) {
				cells = append(cells, pos)
			}
		}
		if len(cells) == 2 {
			pairs = append(pairs, pair{a: cells[0], b: cells[1]})
		}
	}

	if len(pairs) == 0 {
		return nil
	}

	// Build adjacency graph from conjugate pairs.
	adj := make(map[core.Position][]core.Position)
	for _, p := range pairs {
		adj[p.a] = append(adj[p.a], p.b)
		adj[p.b] = append(adj[p.b], p.a)
	}

	// Color connected components using BFS with alternating colors (0 and 1).
	colored := make(map[core.Position]int) // position → color (0 or 1)

	for _, startPos := range sortedPositions(adj) {
		if _, ok := colored[startPos]; ok {
			continue
		}

		// BFS to color this connected component.
		var color0, color1 []core.Position
		queue := []core.Position{startPos}
		colored[startPos] = 0
		color0 = append(color0, startPos)

		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			currColor := colored[curr]

			for _, neighbor := range adj[curr] {
				if _, ok := colored[neighbor]; ok {
					continue
				}
				nextColor := 1 - currColor
				colored[neighbor] = nextColor
				if nextColor == 0 {
					color0 = append(color0, neighbor)
				} else {
					color1 = append(color1, neighbor)
				}
				queue = append(queue, neighbor)
			}
		}

		// Skip trivial chains (single pair — no useful coloring deductions).
		if len(color0)+len(color1) <= 2 {
			continue
		}

		// Rule 1: If two same-colored cells share a unit, that color is invalid.
		// Eliminate the digit from all cells of that color.
		if invalidColor := s.findColorConflict(color0, color1); invalidColor >= 0 {
			evidence := coloringEvidence(digit, color0, color1, adj)
			validCells := color0
			invalidCells := color1
			if invalidColor == 0 {
				validCells = color1
				invalidCells = color0
			}

			// Eliminate digit from all invalid-color cells.
			var move *Move
			eliminated := false
			for _, pos := range invalidCells {
				if board.EliminateCandidate(pos, digit) {
					eliminated = true
					cands := board.Candidates(pos)
					if cands.Count() == 1 && move == nil {
						value := cands.Values()[0]
						move = &Move{
							Cell:      core.NewCell(pos, value),
							Technique: s.Key,
							Reason: fmt.Sprintf(
								"Simple Coloring (Rule 1): %d — two same-colored cells conflict, leaving %d as the only candidate for %s",
								digit, value, pos.ToString(),
							),
							Evidence: evidence,
						}
					}
				}
			}

			// Place digit in all valid-color cells (they must be true).
			for _, pos := range validCells {
				if board.Get(pos) == 0 && board.Candidates(pos).Has(digit) {
					// This is a valid placement — the valid color must be true.
					if move == nil {
						move = &Move{
							Cell:      core.NewCell(pos, digit),
							Technique: s.Key,
							Reason: fmt.Sprintf(
								"Simple Coloring (Rule 1): %d — opposite color confirmed true, placing %d at %s",
								digit, digit, pos.ToString(),
							),
							Evidence: evidence,
						}
					}
				}
			}

			if eliminated || move != nil {
				if move != nil {
					return move
				}
				return &Move{
					EliminationOnly: true,
					Technique:       s.Key,
					Reason: fmt.Sprintf(
						"Simple Coloring (Rule 1): %d — two same-colored cells share a unit, color eliminated",
						digit,
					),
					Evidence: evidence,
				}
			}
		}

		// Rule 2: If an uncolored cell sees cells of both colors, eliminate
		// the digit from that cell.
		move := s.applyRule2(board, digit, color0, color1, colored, adj)
		if move != nil {
			return move
		}
	}

	return nil
}

// findColorConflict checks if any two cells of the same color share a unit.
// Returns the conflicting color (0 or 1), or -1 if no conflict.
func (s *SimpleColoringSolver) findColorConflict(color0, color1 []core.Position) int {
	if s.hasInternalConflict(color0) {
		return 0
	}
	if s.hasInternalConflict(color1) {
		return 1
	}
	return -1
}

// hasInternalConflict checks if any two positions in the group share a unit.
func (s *SimpleColoringSolver) hasInternalConflict(cells []core.Position) bool {
	for i := 0; i < len(cells); i++ {
		for j := i + 1; j < len(cells); j++ {
			if sharesUnit(cells[i], cells[j]) {
				return true
			}
		}
	}
	return false
}

// applyRule2 eliminates the digit from uncolored cells that see both colors.
func (s *SimpleColoringSolver) applyRule2(board *core.Board, digit int, color0, color1 []core.Position, colored map[core.Position]int, adj map[core.Position][]core.Position) *Move {
	var move *Move
	eliminated := false
	evidence := coloringEvidence(digit, color0, color1, adj)

	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			pos := core.NewPosition(row, col)
			if board.Get(pos) != 0 {
				continue
			}
			if _, ok := colored[pos]; ok {
				continue
			}
			if !board.Candidates(pos).Has(digit) {
				continue
			}

			// Check if this cell sees at least one cell of each color.
			seesColor0 := false
			for _, c := range color0 {
				if sharesUnit(pos, c) {
					seesColor0 = true
					break
				}
			}
			if !seesColor0 {
				continue
			}
			seesColor1 := false
			for _, c := range color1 {
				if sharesUnit(pos, c) {
					seesColor1 = true
					break
				}
			}
			if !seesColor1 {
				continue
			}

			// This cell sees both colors — eliminate the digit.
			if board.EliminateCandidate(pos, digit) {
				eliminated = true
				evidence.Eliminations = append(evidence.Eliminations, CandidateRef{Position: pos, Value: digit})
				cands := board.Candidates(pos)
				if cands.Count() == 1 && move == nil {
					value := cands.Values()[0]
					move = &Move{
						Cell:      core.NewCell(pos, value),
						Technique: s.Key,
						Reason: fmt.Sprintf(
							"Simple Coloring (Rule 2): %d — %s sees both colors, leaving %d as the only candidate",
							digit, pos.ToString(), value,
						),
						Evidence: evidence,
					}
				}
			}
		}
	}

	if eliminated {
		if move != nil {
			return move
		}
		return &Move{
			EliminationOnly: true,
			Technique:       s.Key,
			Reason: fmt.Sprintf(
				"Simple Coloring (Rule 2): %d — candidates eliminated from cells seeing both colors",
				digit,
			),
			Evidence: evidence,
		}
	}

	return nil
}

func coloringEvidence(digit int, color0, color1 []core.Position, adj map[core.Position][]core.Position) *Evidence {
	evidence := &Evidence{}
	for _, group := range [][]core.Position{color0, color1} {
		partition := CandidatePartition{}
		for _, position := range group {
			ref := CandidateRef{Position: position, Value: digit}
			partition.Candidates = append(partition.Candidates, ref)
			evidence.Premises = append(evidence.Premises, CandidateGroup{Position: position, Values: []int{digit}})
		}
		evidence.Partitions = append(evidence.Partitions, partition)
	}
	seen := make(map[[2]core.Position]struct{})
	for _, from := range sortedPositions(adj) {
		neighbors := append([]core.Position(nil), adj[from]...)
		sort.Slice(neighbors, func(left, right int) bool {
			if neighbors[left].Row != neighbors[right].Row {
				return neighbors[left].Row < neighbors[right].Row
			}
			return neighbors[left].Column < neighbors[right].Column
		})
		for _, to := range neighbors {
			if _, ok := seen[[2]core.Position{to, from}]; ok {
				continue
			}
			seen[[2]core.Position{from, to}] = struct{}{}
			evidence.Links = append(evidence.Links, CandidateLink{
				From: CandidateRef{Position: from, Value: digit},
				To:   CandidateRef{Position: to, Value: digit},
				Kind: LinkStrong,
			})
		}
	}
	return evidence
}
