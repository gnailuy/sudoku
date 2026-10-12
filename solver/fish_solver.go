package solver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gnailuy/sudoku/core"
)

// FishSolver finds fish patterns of a parameterized size.
//
// Fish patterns are generalizations of the X-Wing technique:
//   - Size 2: X-Wing — a candidate in exactly 2 cells in each of 2 rows,
//     sharing the same 2 columns (or transpose).
//   - Size 3: Swordfish — a candidate in 2–3 cells in each of 3 rows,
//     confined to the same 3 columns (or transpose).
//   - Size 4: Jellyfish — a candidate in 2–4 cells in each of 4 rows,
//     confined to the same 4 columns (or transpose).
//
// The solver applies eliminations to the board's elimination layer. It returns
// a placement move when an elimination creates a naked single, or an
// elimination-only move when candidates were reduced without creating a
// placement.
type FishSolver struct {
	Base
	size int // 2=X-Wing, 3=Swordfish, 4=Jellyfish
}

// NewXWingSolver creates a FishSolver configured for X-Wing (size 2).
func NewXWingSolver() *FishSolver {
	return &FishSolver{
		Base: Base{
			Key:         "x-wing",
			DisplayName: "X-Wing",
			Description: "Finds a candidate confined to the same two columns in two rows (or same two rows in two columns), enabling eliminations that reveal a single candidate.",
			Weight:      WeightXWing,
		},
		size: 2,
	}
}

// NewSwordfishSolver creates a FishSolver configured for Swordfish (size 3).
func NewSwordfishSolver() *FishSolver {
	return &FishSolver{
		Base: Base{
			Key:         "swordfish",
			DisplayName: "Swordfish",
			Description: "Finds a candidate confined to the same three columns in three rows (or same three rows in three columns), enabling eliminations that reveal a single candidate.",
			Weight:      WeightSwordfish,
		},
		size: 3,
	}
}

// NewJellyfishSolver creates a FishSolver configured for Jellyfish (size 4).
func NewJellyfishSolver() *FishSolver {
	return &FishSolver{
		Base: Base{
			Key:         "jellyfish",
			DisplayName: "Jellyfish",
			Description: "Finds a candidate confined to the same four columns in four rows (or same four rows in four columns), enabling eliminations that reveal a single candidate.",
			Weight:      WeightJellyfish,
		},
		size: 4,
	}
}

// fishLineInfo holds a line index (row or column) and the set of positions
// on the cross-axis where a digit appears as a candidate.
type fishLineInfo struct {
	lineIndex      int
	crossPositions []int
}

// Apply checks for fish patterns on all digits, trying both row-based and
// column-based orientations.
func (s *FishSolver) Apply(board *core.Board) *Move {
	// Row-based: digit in 2–size cells in each of 'size' rows, confined to
	// 'size' columns → eliminate from those columns.
	if move := s.findFish(board, true); move != nil {
		return move
	}

	// Column-based: digit in 2–size cells in each of 'size' columns,
	// confined to 'size' rows → eliminate from those rows.
	if move := s.findFish(board, false); move != nil {
		return move
	}

	return nil
}

// findFish searches for fish patterns in one orientation.
// If rowBased is true, base lines are rows and cross-axis is columns.
// If rowBased is false, base lines are columns and cross-axis is rows.
func (s *FishSolver) findFish(board *core.Board, rowBased bool) *Move {
	for digit := 1; digit <= 9; digit++ {
		// Collect lines where the digit appears in 2–size cells.
		var eligible []fishLineInfo

		for line := 0; line < 9; line++ {
			var crossPositions []int
			for cross := 0; cross < 9; cross++ {
				var pos core.Position
				if rowBased {
					pos = core.NewPosition(line, cross)
				} else {
					pos = core.NewPosition(cross, line)
				}
				if board.Get(pos) == 0 && board.Candidates(pos).Has(digit) {
					crossPositions = append(crossPositions, cross)
				}
			}
			if len(crossPositions) >= 2 && len(crossPositions) <= s.size {
				eligible = append(eligible, fishLineInfo{lineIndex: line, crossPositions: crossPositions})
			}
		}

		if len(eligible) < s.size {
			continue
		}

		// Try all combinations of 'size' eligible lines.
		indices := make([]int, s.size)
		if move := s.fishCombinationsSearch(board, eligible, indices, 0, 0, digit, rowBased); move != nil {
			return move
		}
	}

	return nil
}

// fishCombinationsSearch generates combinations of base lines and checks for fish patterns.
func (s *FishSolver) fishCombinationsSearch(board *core.Board, eligible []fishLineInfo, indices []int, start int, depth int, digit int, rowBased bool) *Move {
	if depth == s.size {
		// Compute the union of cross positions.
		crossSet := make(map[int]bool)
		for i := 0; i < s.size; i++ {
			for _, c := range eligible[indices[i]].crossPositions {
				crossSet[c] = true
			}
		}

		// Fish pattern: the union must be exactly 'size' cross positions.
		if len(crossSet) != s.size {
			return nil
		}

		// Collect the base lines and cross positions.
		crossLines := make([]int, 0, s.size)
		for c := range crossSet {
			crossLines = append(crossLines, c)
		}
		sort.Ints(crossLines)

		baseLines := make([]int, s.size)
		baseSet := make(map[int]bool, s.size)
		for i := 0; i < s.size; i++ {
			baseLines[i] = eligible[indices[i]].lineIndex
			baseSet[eligible[indices[i]].lineIndex] = true
		}

		// Apply eliminations: eliminate digit from cross lines, excluding base lines.
		var move *Move
		eliminated := false
		for _, cross := range crossLines {
			for line := 0; line < 9; line++ {
				if baseSet[line] {
					continue
				}
				var pos core.Position
				if rowBased {
					pos = core.NewPosition(line, cross)
				} else {
					pos = core.NewPosition(cross, line)
				}
				if board.EliminateCandidate(pos, digit) {
					eliminated = true
					cands := board.Candidates(pos)
					if cands.Count() == 1 && move == nil {
						value := cands.Values()[0]
						move = &Move{
							Cell:      core.NewCell(pos, value),
							Technique: s.Key,
							Reason:    s.formatReason(digit, baseLines, crossLines, rowBased, value, pos),
						}
					}
				}
			}
		}

		if eliminated {
			evidence := fishEvidence(eligible, indices, crossLines, digit, rowBased, s.size)
			if move != nil {
				move.Evidence = evidence
				return move
			}
			return &Move{
				EliminationOnly: true,
				Technique:       s.Key,
				Reason:          s.formatEliminationReason(digit, baseLines, crossLines, rowBased),
				Evidence:        evidence,
			}
		}

		return nil
	}

	for i := start; i < len(eligible); i++ {
		indices[depth] = i
		if move := s.fishCombinationsSearch(board, eligible, indices, i+1, depth+1, digit, rowBased); move != nil {
			return move
		}
	}

	return nil
}

func fishEvidence(eligible []fishLineInfo, indices, crossLines []int, digit int, rowBased bool, size int) *Evidence {
	baseKind, crossKind := UnitRow, UnitColumn
	if !rowBased {
		baseKind, crossKind = UnitColumn, UnitRow
	}
	units := make([]UnitRef, 0, size*2)
	premises := make([]CandidateGroup, 0, size*size)
	for _, selected := range indices[:size] {
		line := eligible[selected]
		units = append(units, UnitRef{Kind: baseKind, Index: line.lineIndex})
		for _, cross := range line.crossPositions {
			position := core.NewPosition(line.lineIndex, cross)
			if !rowBased {
				position = core.NewPosition(cross, line.lineIndex)
			}
			premises = append(premises, CandidateGroup{Position: position, Values: []int{digit}})
		}
	}
	for _, cross := range crossLines {
		units = append(units, UnitRef{Kind: crossKind, Index: cross})
	}
	primary := units[0]
	return &Evidence{Unit: &primary, Units: units, Premises: premises}
}

// formatReason creates a human-readable explanation for a fish placement move.
func (s *FishSolver) formatReason(digit int, baseLines, crossLines []int, rowBased bool, value int, pos core.Position) string {
	var baseName, crossName string
	if rowBased {
		baseName = "rows"
		crossName = "columns"
	} else {
		baseName = "columns"
		crossName = "rows"
	}

	return fmt.Sprintf(
		"%s: %d in %s %s is confined to %s %s, leaving %d as the only candidate for %s",
		s.DisplayName, digit, baseName, formatLineNumbers(baseLines),
		crossName, formatLineNumbers(crossLines), value, pos.ToString(),
	)
}

// formatEliminationReason creates a human-readable explanation for a fish elimination-only move.
func (s *FishSolver) formatEliminationReason(digit int, baseLines, crossLines []int, rowBased bool) string {
	var baseName, crossName string
	if rowBased {
		baseName = "rows"
		crossName = "columns"
	} else {
		baseName = "columns"
		crossName = "rows"
	}

	return fmt.Sprintf(
		"%s: %d in %s %s is confined to %s %s — candidates eliminated",
		s.DisplayName, digit, baseName, formatLineNumbers(baseLines),
		crossName, formatLineNumbers(crossLines),
	)
}

// formatLineNumbers formats a slice of 0-based line indices as 1-based, comma-separated.
func formatLineNumbers(lines []int) string {
	parts := make([]string, len(lines))
	for i, l := range lines {
		parts[i] = fmt.Sprintf("%d", l+1)
	}
	return strings.Join(parts, ",")
}
