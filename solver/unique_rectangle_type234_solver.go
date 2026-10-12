package solver

import (
	"fmt"
	"sort"

	"github.com/gnailuy/sudoku/core"
)

// UniqueRectangleType2Solver detects Unique Rectangle Type 2 patterns.
//
// Type 2: Two opposite corners of the rectangle have the UR pair {A,B} plus
// the same extra candidate X (i.e., both have {A,B,X}). The other two corners
// are bi-value with {A,B}. Since placing A,B in all four corners creates a
// deadly pattern, one of the two extra-candidate corners must be X. Therefore,
// X can be eliminated from any cell that sees both extra-candidate corners.
type UniqueRectangleType2Solver struct {
	Base
}

// NewUniqueRectangleType2Solver creates a UR Type 2 solver.
func NewUniqueRectangleType2Solver() *UniqueRectangleType2Solver {
	return &UniqueRectangleType2Solver{
		Base: Base{
			Key:         "unique-rectangle-2",
			DisplayName: "Unique Rectangle Type 2",
			Description: "Two UR corners share one extra candidate X — eliminates X from cells seeing both.",
			Weight:      WeightUniqueRectangle2,
		},
	}
}

// Apply scans for UR Type 2 patterns.
func (s *UniqueRectangleType2Solver) Apply(board *core.Board) *Move {
	return scanURPatterns(board, s.Key, func(board *core.Board, corners [4]core.Position, a, b int) *Move {
		return s.checkType2(board, corners, a, b)
	})
}

// checkType2 checks if two opposite corners have {A,B}+X and eliminates X.
func (s *UniqueRectangleType2Solver) checkType2(board *core.Board, corners [4]core.Position, a, b int) *Move {
	// Try all 3 pairs of "two opposite corners" as the extra-candidate pair.
	// In a rectangle with corners 0,1,2,3 arranged as:
	//   0--1
	//   2--3
	// The opposite pairs are: (0,3) and (1,2); or (0,1) and (2,3); or (0,2) and (1,3).
	// Actually, for Type 2 we need two corners that have extras, two that are bivalue.
	// All combinations of 2 extra + 2 bivalue:
	combos := [][2][2]int{
		{{0, 1}, {2, 3}},
		{{0, 2}, {1, 3}},
		{{0, 3}, {1, 2}},
		{{1, 2}, {0, 3}},
		{{1, 3}, {0, 2}},
		{{2, 3}, {0, 1}},
	}

	for _, combo := range combos {
		extraIdxs := combo[0]
		biIdxs := combo[1]

		// The bivalue corners must have exactly {A,B}.
		allBi := true
		for _, idx := range biIdxs {
			cands := board.Candidates(corners[idx])
			if cands.Count() != 2 || !cands.Has(a) || !cands.Has(b) {
				allBi = false
				break
			}
		}
		if !allBi {
			continue
		}

		// Both extra corners must have {A,B} plus the same single extra candidate X.
		candsE0 := board.Candidates(corners[extraIdxs[0]])
		candsE1 := board.Candidates(corners[extraIdxs[1]])
		if candsE0.Count() != 3 || candsE1.Count() != 3 {
			continue
		}
		if !candsE0.Has(a) || !candsE0.Has(b) || !candsE1.Has(a) || !candsE1.Has(b) {
			continue
		}

		// Find the extra candidate X in each.
		var x0, x1 int
		for _, v := range candsE0.Values() {
			if v != a && v != b {
				x0 = v
			}
		}
		for _, v := range candsE1.Values() {
			if v != a && v != b {
				x1 = v
			}
		}
		if x0 != x1 || x0 == 0 {
			continue
		}
		x := x0

		// The two extra-candidate corners must share a unit (for the
		// elimination to be useful — cells seeing both must exist).
		posE0 := corners[extraIdxs[0]]
		posE1 := corners[extraIdxs[1]]

		premises := rectanglePremises(board, corners[biIdxs[0]], corners[biIdxs[1]], posE0, posE1)

		// Eliminate X from cells seeing both extra corners.
		var move *Move
		eliminated := false
		var eliminations []CandidateRef
		for row := 0; row < 9; row++ {
			for col := 0; col < 9; col++ {
				pos := core.NewPosition(row, col)
				if pos == posE0 || pos == posE1 {
					continue
				}
				// Skip the bivalue corners too.
				if pos == corners[biIdxs[0]] || pos == corners[biIdxs[1]] {
					continue
				}
				if !sharesUnit(pos, posE0) || !sharesUnit(pos, posE1) {
					continue
				}
				if board.EliminateCandidate(pos, x) {
					eliminated = true
					eliminations = append(eliminations, CandidateRef{Position: pos, Value: x})
					cands := board.Candidates(pos)
					if cands.Count() == 1 && move == nil {
						value := cands.Values()[0]
						move = &Move{
							Cell:      core.NewCell(pos, value),
							Technique: s.Key,
							Reason: fmt.Sprintf(
								"Unique Rectangle Type 2: corners %s,%s have {%d,%d}, corners %s,%s have {%d,%d,%d} — %d eliminated from %s, leaving %d",
								corners[biIdxs[0]].ToString(), corners[biIdxs[1]].ToString(), a, b,
								posE0.ToString(), posE1.ToString(), a, b, x,
								x, pos.ToString(), value,
							),
						}
					}
				}
			}
		}

		if eliminated {
			evidence := &Evidence{Premises: premises, Eliminations: eliminations}
			if move != nil {
				move.Evidence = evidence
				return move
			}
			return &Move{
				EliminationOnly: true,
				Technique:       s.Key,
				Evidence:        evidence,
				Reason: fmt.Sprintf(
					"Unique Rectangle Type 2: corners %s,%s have {%d,%d,%d} — %d eliminated from cells seeing both",
					posE0.ToString(), posE1.ToString(), a, b, x, x,
				),
			}
		}
	}

	return nil
}

// UniqueRectangleType3Solver detects Unique Rectangle Type 3 patterns.
//
// Type 3: Two corners are bi-value {A,B}. The other two corners contain
// {A,B} plus extra candidates. The extra candidates from the two corners,
// when combined, form a naked subset with nearby cells in a shared unit.
// This enables elimination of those subset digits from other cells in that unit.
type UniqueRectangleType3Solver struct {
	Base
}

// NewUniqueRectangleType3Solver creates a UR Type 3 solver.
func NewUniqueRectangleType3Solver() *UniqueRectangleType3Solver {
	return &UniqueRectangleType3Solver{
		Base: Base{
			Key:         "unique-rectangle-3",
			DisplayName: "Unique Rectangle Type 3",
			Description: "Two UR corners with extras form a naked subset with nearby cells, enabling eliminations.",
			Weight:      WeightUniqueRectangle3,
		},
	}
}

// Apply scans for UR Type 3 patterns.
func (s *UniqueRectangleType3Solver) Apply(board *core.Board) *Move {
	return scanURPatterns(board, s.Key, func(board *core.Board, corners [4]core.Position, a, b int) *Move {
		return s.checkType3(board, corners, a, b)
	})
}

// checkType3 checks if two corners' extra candidates form a naked subset.
func (s *UniqueRectangleType3Solver) checkType3(board *core.Board, corners [4]core.Position, a, b int) *Move {
	// Try all combinations: 2 bivalue + 2 extra corners.
	combos := [][2][2]int{
		{{0, 1}, {2, 3}},
		{{0, 2}, {1, 3}},
		{{0, 3}, {1, 2}},
		{{1, 2}, {0, 3}},
		{{1, 3}, {0, 2}},
		{{2, 3}, {0, 1}},
	}

	for _, combo := range combos {
		extraIdxs := combo[0]
		biIdxs := combo[1]

		// The bivalue corners must have exactly {A,B}.
		allBi := true
		for _, idx := range biIdxs {
			cands := board.Candidates(corners[idx])
			if cands.Count() != 2 || !cands.Has(a) || !cands.Has(b) {
				allBi = false
				break
			}
		}
		if !allBi {
			continue
		}

		// Both extra corners must have {A,B} + extras and share at least one unit.
		posE0 := corners[extraIdxs[0]]
		posE1 := corners[extraIdxs[1]]
		candsE0 := board.Candidates(posE0)
		candsE1 := board.Candidates(posE1)
		if !candsE0.Has(a) || !candsE0.Has(b) || candsE0.Count() <= 2 {
			continue
		}
		if !candsE1.Has(a) || !candsE1.Has(b) || candsE1.Count() <= 2 {
			continue
		}

		// Collect the extra candidates (non-A, non-B) from both corners.
		var extras core.CandidateSet
		for _, v := range candsE0.Values() {
			if v != a && v != b {
				extras.Add(v)
			}
		}
		for _, v := range candsE1.Values() {
			if v != a && v != b {
				extras.Add(v)
			}
		}

		if extras.Count() == 0 {
			continue
		}

		// For each unit shared by both extra corners, check if the extras
		// form a naked subset with other cells in that unit.
		// The naked subset size = extras.Count(), and we need exactly that
		// many other cells in the unit whose candidates are a subset of extras.
		units := allUnits()
		for _, u := range units {
			if !u.contains(posE0) || !u.contains(posE1) {
				continue
			}

			move := s.tryNakedSubset(board, u, posE0, posE1, corners, extras)
			if move != nil {
				return move
			}
		}
	}

	return nil
}

// tryNakedSubset checks if the extras from the two UR corners plus other cells
// in the unit form a naked subset, enabling eliminations.
func (s *UniqueRectangleType3Solver) tryNakedSubset(board *core.Board, u unit, posE0, posE1 core.Position, corners [4]core.Position, extras core.CandidateSet) *Move {
	subsetSize := extras.Count()

	// Find cells in this unit (excluding the two extra UR corners) whose
	// candidates are a subset of the extras.
	var partners []core.Position
	for _, pos := range u.positions {
		if pos == posE0 || pos == posE1 {
			continue
		}
		if board.Get(pos) != 0 {
			continue
		}
		cands := board.Candidates(pos)
		if cands.Count() == 0 {
			continue
		}
		// Check if all candidates of this cell are within extras.
		isSubset := true
		for _, v := range cands.Values() {
			if !extras.Has(v) {
				isSubset = false
				break
			}
		}
		if isSubset {
			partners = append(partners, pos)
		}
	}

	// We need exactly (subsetSize - 1) partners to complete the naked subset.
	// The "virtual" cell contribution from the UR corners counts as 1 cell
	// with candidates = extras.
	needed := subsetSize - 1
	if len(partners) < needed {
		return nil
	}

	// Try all combinations of 'needed' partners.
	if needed == 0 {
		// No partners needed — extras themselves complete the subset.
		// Eliminate extras from other cells in the unit.
		return s.eliminateFromUnit(board, u, []core.Position{posE0, posE1}, corners, extras)
	}

	// Generate combinations of 'needed' from partners.
	indices := make([]int, needed)
	return s.combinePartners(board, u, posE0, posE1, corners, extras, partners, indices, 0, 0)
}

func (s *UniqueRectangleType3Solver) combinePartners(board *core.Board, u unit, posE0, posE1 core.Position, corners [4]core.Position, extras core.CandidateSet, partners []core.Position, indices []int, start, depth int) *Move {
	if depth == len(indices) {
		// Verify the combined candidates of the chosen partners match extras.
		var combined core.CandidateSet
		for i := 0; i < depth; i++ {
			cands := board.Candidates(partners[indices[i]])
			for _, v := range cands.Values() {
				combined.Add(v)
			}
		}
		// Combined must exactly equal extras.
		if combined != extras {
			return nil
		}

		// Build the exclusion set (UR corners + chosen partners).
		exclude := make(map[core.Position]bool)
		exclude[posE0] = true
		exclude[posE1] = true
		for i := 0; i < depth; i++ {
			exclude[partners[indices[i]]] = true
		}

		return s.eliminateFromUnit(board, u, mapKeys(exclude), corners, extras)
	}

	for i := start; i < len(partners); i++ {
		indices[depth] = i
		if move := s.combinePartners(board, u, posE0, posE1, corners, extras, partners, indices, i+1, depth+1); move != nil {
			return move
		}
	}
	return nil
}

func mapKeys(m map[core.Position]bool) []core.Position {
	keys := make([]core.Position, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Row != keys[j].Row {
			return keys[i].Row < keys[j].Row
		}
		return keys[i].Column < keys[j].Column
	})
	return keys
}

func (s *UniqueRectangleType3Solver) eliminateFromUnit(board *core.Board, u unit, excludeList []core.Position, corners [4]core.Position, extras core.CandidateSet) *Move {
	unitRef := unitReference(u)
	premisePositions := append([]core.Position(nil), corners[:]...)
	premisePositions = append(premisePositions, excludeList...)
	premises := uniquePositionPremises(board, premisePositions)

	exclude := make(map[core.Position]bool)
	for _, p := range excludeList {
		exclude[p] = true
	}

	var move *Move
	eliminated := false
	var eliminations []CandidateRef
	for _, pos := range u.positions {
		if exclude[pos] {
			continue
		}
		if board.Get(pos) != 0 {
			continue
		}
		for _, v := range extras.Values() {
			if board.EliminateCandidate(pos, v) {
				eliminated = true
				eliminations = append(eliminations, CandidateRef{Position: pos, Value: v})
				cands := board.Candidates(pos)
				if cands.Count() == 1 && move == nil {
					value := cands.Values()[0]
					move = &Move{
						Cell:      core.NewCell(pos, value),
						Technique: s.Key,
						Reason: fmt.Sprintf(
							"Unique Rectangle Type 3: UR extras form naked subset {%s} — %d eliminated from %s, leaving %d",
							formatValues(extras.Values()), v, pos.ToString(), value,
						),
					}
				}
			}
		}
	}

	if eliminated {
		evidence := &Evidence{
			Unit:         &unitRef,
			Units:        []UnitRef{unitRef},
			Premises:     premises,
			Eliminations: eliminations,
		}
		if move != nil {
			move.Evidence = evidence
			return move
		}
		return &Move{
			EliminationOnly: true,
			Technique:       s.Key,
			Evidence:        evidence,
			Reason: fmt.Sprintf(
				"Unique Rectangle Type 3: UR extras form naked subset {%s} — candidates eliminated",
				formatValues(extras.Values()),
			),
		}
	}

	return nil
}

// UniqueRectangleType4Solver detects Unique Rectangle Type 4 patterns.
//
// Type 4: Two corners are bi-value {A,B}. The other two corners contain
// {A,B} plus extras and share a unit. If one of {A,B} forms a conjugate
// pair in that shared unit (appears in exactly those two cells), then the
// other of {A,B} can be eliminated from both extra corners — it can't be
// in both without creating a deadly pattern, and the conjugate constraint
// forces the first digit into one of them.
type UniqueRectangleType4Solver struct {
	Base
}

// NewUniqueRectangleType4Solver creates a UR Type 4 solver.
func NewUniqueRectangleType4Solver() *UniqueRectangleType4Solver {
	return &UniqueRectangleType4Solver{
		Base: Base{
			Key:         "unique-rectangle-4",
			DisplayName: "Unique Rectangle Type 4",
			Description: "Two UR corners with extras share a conjugate pair on one UR digit — the other UR digit is eliminated from both.",
			Weight:      WeightUniqueRectangle4,
		},
	}
}

// Apply scans for UR Type 4 patterns.
func (s *UniqueRectangleType4Solver) Apply(board *core.Board) *Move {
	return scanURPatterns(board, s.Key, func(board *core.Board, corners [4]core.Position, a, b int) *Move {
		return s.checkType4(board, corners, a, b)
	})
}

// checkType4 checks if a conjugate pair on one UR digit enables elimination of the other.
func (s *UniqueRectangleType4Solver) checkType4(board *core.Board, corners [4]core.Position, a, b int) *Move {
	combos := [][2][2]int{
		{{0, 1}, {2, 3}},
		{{0, 2}, {1, 3}},
		{{0, 3}, {1, 2}},
		{{1, 2}, {0, 3}},
		{{1, 3}, {0, 2}},
		{{2, 3}, {0, 1}},
	}

	for _, combo := range combos {
		extraIdxs := combo[0]
		biIdxs := combo[1]

		// The bivalue corners must have exactly {A,B}.
		allBi := true
		for _, idx := range biIdxs {
			cands := board.Candidates(corners[idx])
			if cands.Count() != 2 || !cands.Has(a) || !cands.Has(b) {
				allBi = false
				break
			}
		}
		if !allBi {
			continue
		}

		// Both extra corners must have {A,B} + extras.
		posE0 := corners[extraIdxs[0]]
		posE1 := corners[extraIdxs[1]]
		candsE0 := board.Candidates(posE0)
		candsE1 := board.Candidates(posE1)
		if !candsE0.Has(a) || !candsE0.Has(b) || candsE0.Count() <= 2 {
			continue
		}
		if !candsE1.Has(a) || !candsE1.Has(b) || candsE1.Count() <= 2 {
			continue
		}

		// Check each shared unit for a conjugate pair on A or B.
		units := allUnits()
		for _, u := range units {
			if !u.contains(posE0) || !u.contains(posE1) {
				continue
			}

			// Check if digit A forms a conjugate pair in this unit
			// (appears in exactly the two extra corners).
			for _, conjugateDigit := range []int{a, b} {
				elimDigit := a
				if conjugateDigit == a {
					elimDigit = b
				}

				// Count cells in this unit where conjugateDigit is a candidate.
				var digitCells []core.Position
				for _, pos := range u.positions {
					if board.Get(pos) == 0 && board.Candidates(pos).Has(conjugateDigit) {
						digitCells = append(digitCells, pos)
					}
				}

				// Must be exactly the two extra corners.
				if len(digitCells) != 2 {
					continue
				}
				if !((digitCells[0] == posE0 && digitCells[1] == posE1) ||
					(digitCells[0] == posE1 && digitCells[1] == posE0)) {
					continue
				}

				// Conjugate pair on conjugateDigit! Eliminate elimDigit from both extra corners.
				premises := rectanglePremises(board, corners[biIdxs[0]], corners[biIdxs[1]], posE0, posE1)
				var move *Move
				eliminated := false
				var eliminations []CandidateRef
				for _, pos := range []core.Position{posE0, posE1} {
					if board.EliminateCandidate(pos, elimDigit) {
						eliminated = true
						eliminations = append(eliminations, CandidateRef{Position: pos, Value: elimDigit})
						cands := board.Candidates(pos)
						if cands.Count() == 1 && move == nil {
							value := cands.Values()[0]
							move = &Move{
								Cell:      core.NewCell(pos, value),
								Technique: s.Key,
								Reason: fmt.Sprintf(
									"Unique Rectangle Type 4: %d is conjugate in %s,%s — %d eliminated from %s, leaving %d",
									conjugateDigit, posE0.ToString(), posE1.ToString(),
									elimDigit, pos.ToString(), value,
								),
							}
						}
					}
				}

				if eliminated {
					unitRef := unitReference(u)
					evidence := &Evidence{
						Unit:         &unitRef,
						Units:        []UnitRef{unitRef},
						Premises:     premises,
						Eliminations: eliminations,
					}
					if move != nil {
						move.Evidence = evidence
						return move
					}
					return &Move{
						EliminationOnly: true,
						Technique:       s.Key,
						Evidence:        evidence,
						Reason: fmt.Sprintf(
							"Unique Rectangle Type 4: %d is conjugate in %s,%s — %d eliminated from both",
							conjugateDigit, posE0.ToString(), posE1.ToString(), elimDigit,
						),
					}
				}
			}
		}
	}

	return nil
}

func uniquePositionPremises(board *core.Board, positions []core.Position) []CandidateGroup {
	seen := make(map[core.Position]struct{})
	unique := make([]core.Position, 0, len(positions))
	for _, position := range positions {
		if _, ok := seen[position]; ok {
			continue
		}
		seen[position] = struct{}{}
		unique = append(unique, position)
	}
	return rectanglePremises(board, unique...)
}

func unitReference(u unit) UnitRef {
	first := u.positions[0]
	sameRow, sameColumn := true, true
	for _, position := range u.positions[1:] {
		sameRow = sameRow && position.Row == first.Row
		sameColumn = sameColumn && position.Column == first.Column
	}
	if sameRow {
		return UnitRef{Kind: UnitRow, Index: first.Row}
	}
	if sameColumn {
		return UnitRef{Kind: UnitColumn, Index: first.Column}
	}
	return UnitRef{Kind: UnitBox, Index: boxIndex(first)}
}

// ---- Shared UR scanning infrastructure ----

// urCheckFunc is the callback signature for UR type checks.
type urCheckFunc func(board *core.Board, corners [4]core.Position, a, b int) *Move

// scanURPatterns finds all potential UR rectangles and calls checkFn for each.
func scanURPatterns(board *core.Board, technique string, checkFn urCheckFunc) *Move {
	// Collect all empty cells that contain at least two candidates.
	type cellInfo struct {
		pos   core.Position
		cands core.CandidateSet
	}
	var cells []cellInfo
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			pos := core.NewPosition(r, c)
			if board.Get(pos) != 0 {
				continue
			}
			cands := board.Candidates(pos)
			if cands.Count() >= 2 {
				cells = append(cells, cellInfo{pos: pos, cands: cands})
			}
		}
	}

	// For each candidate pair {a, b}, find all cells containing both.
	type pairKey struct{ a, b int }
	pairCells := make(map[pairKey][]core.Position)
	for _, ci := range cells {
		vals := ci.cands.Values()
		for i := 0; i < len(vals); i++ {
			for j := i + 1; j < len(vals); j++ {
				key := pairKey{vals[i], vals[j]}
				pairCells[key] = append(pairCells[key], ci.pos)
			}
		}
	}

	// For each pair with >= 4 cells, try all rectangles.
	pairKeys := make([]pairKey, 0, len(pairCells))
	for pair := range pairCells {
		pairKeys = append(pairKeys, pair)
	}
	sort.Slice(pairKeys, func(i, j int) bool {
		if pairKeys[i].a != pairKeys[j].a {
			return pairKeys[i].a < pairKeys[j].a
		}
		return pairKeys[i].b < pairKeys[j].b
	})
	for _, pair := range pairKeys {
		positions := pairCells[pair]
		if len(positions) < 4 {
			continue
		}

		// Group by row.
		rowGroups := make(map[int][]core.Position)
		for _, pos := range positions {
			rowGroups[pos.Row] = append(rowGroups[pos.Row], pos)
		}

		// Need at least 2 rows.
		rows := make([]int, 0, len(rowGroups))
		for r := range rowGroups {
			rows = append(rows, r)
		}
		sort.Ints(rows)

		// Try all pairs of rows.
		for i := 0; i < len(rows); i++ {
			for j := i + 1; j < len(rows); j++ {
				r1, r2 := rows[i], rows[j]
				cols1 := rowGroups[r1]
				cols2 := rowGroups[r2]

				// Try all pairs of columns that appear in both rows.
				for _, p1 := range cols1 {
					for _, p2 := range cols1 {
						if p1.Column >= p2.Column {
							continue
						}
						// Find matching columns in row 2.
						var c1Match, c2Match bool
						for _, p := range cols2 {
							if p.Column == p1.Column {
								c1Match = true
							}
							if p.Column == p2.Column {
								c2Match = true
							}
						}
						if !c1Match || !c2Match {
							continue
						}

						corners := [4]core.Position{
							core.NewPosition(r1, p1.Column),
							core.NewPosition(r1, p2.Column),
							core.NewPosition(r2, p1.Column),
							core.NewPosition(r2, p2.Column),
						}

						// Must span exactly 2 boxes.
						boxSet := make(map[int]bool)
						for _, c := range corners {
							boxSet[boxIndex(c)] = true
						}
						if len(boxSet) != 2 {
							continue
						}

						if move := checkFn(board, corners, pair.a, pair.b); move != nil {
							return move
						}
					}
				}
			}
		}
	}

	return nil
}

// formatValues formats a sorted slice of ints as comma-separated.
func formatValues(vals []int) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return join(parts, ",")
}

func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += sep + p
	}
	return result
}

// unit.contains checks if a unit contains a position.
func (u unit) contains(pos core.Position) bool {
	for _, p := range u.positions {
		if p == pos {
			return true
		}
	}
	return false
}
