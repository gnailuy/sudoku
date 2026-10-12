package solver

import (
	"fmt"

	"github.com/gnailuy/sudoku/core"
)

// XYChainSolver finds XY-Chain patterns.
//
// An XY-Chain is a generalization of XY-Wing to chains of arbitrary length.
// It uses bi-value cells (exactly 2 candidates) connected end-to-end:
//
//	Cell 1: {A,B} — Cell 2: {B,C} — Cell 3: {C,D} — ... — Cell N: {?,Z}
//
// Each consecutive pair shares a unit and a candidate. The chain alternates:
// if we assume A in Cell 1, then Cell 1 ≠ B, so Cell 2 = B (strong inference
// within the bi-value cell), and if Cell 2 = B then Cell 2 ≠ C, so Cell 3
// must be C if they share a conjugate, etc.
//
// The key insight: the first candidate of the first cell and the last
// candidate of the last cell are the "endpoints." If the first cell starts
// with {A,B} and the last cell ends with {Y,Z}, then:
//   - Either Cell 1 = A, or the chain forces Cell N = Z.
//   - If A == Z (the endpoints share the same digit), then any cell that
//     sees BOTH Cell 1 and Cell N can eliminate that digit — at least one
//     of them must contain it.
//
// This is exactly the same logic as XY-Wing, but extended to chains of
// length > 3.
//
// The solver applies eliminations to the board's elimination layer. It returns
// a placement move when an elimination creates a naked single, or an
// elimination-only move when candidates were reduced without creating a
// placement.
type XYChainSolver struct {
	Base
}

// NewXYChainSolver creates an XYChainSolver.
func NewXYChainSolver() *XYChainSolver {
	return &XYChainSolver{
		Base: Base{
			Key:         "xy-chain",
			DisplayName: "XY-Chain",
			Description: "Chains of bi-value cells connected by shared candidates — eliminates the common endpoint digit from cells seeing both ends.",
			Weight:      WeightXYChain,
		},
	}
}

// xyChainCell holds a bi-value cell's position and candidates.
type xyChainCell struct {
	pos  core.Position
	vals [2]int
}

// xyLink represents a connection between two bi-value cells in the XY-Chain graph.
type xyLink struct {
	to        int // index into biCells
	enterExit [2]int
	// enterExit[0] = the shared digit (entering 'to')
	// enterExit[1] = the other digit of 'to' (exiting 'to')
}

// Apply searches for XY-Chain patterns.
func (s *XYChainSolver) Apply(board *core.Board) *Move {
	// Collect all bi-value cells.
	var biCells []xyChainCell
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			pos := core.NewPosition(row, col)
			if board.Get(pos) != 0 {
				continue
			}
			cands := board.Candidates(pos)
			if cands.Count() == 2 {
				v := cands.Values()
				biCells = append(biCells, xyChainCell{pos: pos, vals: [2]int{v[0], v[1]}})
			}
		}
	}

	if len(biCells) < 3 {
		return nil
	}

	// Build adjacency: two bi-value cells are linked if they share a unit
	// AND share exactly one candidate (the linking digit).
	adj := make([][]xyLink, len(biCells))
	for i := 0; i < len(biCells); i++ {
		for j := 0; j < len(biCells); j++ {
			if i == j {
				continue
			}
			if !sharesUnit(biCells[i].pos, biCells[j].pos) {
				continue
			}
			// Find shared candidates.
			for _, vi := range biCells[i].vals {
				for _, vj := range biCells[j].vals {
					if vi == vj {
						// The shared digit is vi.
						// The "exit" digit of j is the other candidate of j.
						otherJ := biCells[j].vals[0]
						if otherJ == vj {
							otherJ = biCells[j].vals[1]
						}
						adj[i] = append(adj[i], xyLink{
							to:        j,
							enterExit: [2]int{vj, otherJ},
						})
					}
				}
			}
		}
	}

	// DFS from each bi-value cell, trying each of its two candidates as
	// the "exit digit" (the one we assume is NOT in the cell, forcing
	// the chain). We track the chain and check if the endpoint digit
	// matches the start's other candidate.
	visited := make([]bool, len(biCells))

	for startIdx := 0; startIdx < len(biCells); startIdx++ {
		// Try each candidate of the start cell as the "free" endpoint digit.
		for startSide := 0; startSide < 2; startSide++ {
			startDigit := biCells[startIdx].vals[startSide]
			exitDigit := biCells[startIdx].vals[1-startSide]

			// Reset visited.
			for k := range visited {
				visited[k] = false
			}
			visited[startIdx] = true

			// DFS: follow links where the entering digit matches our
			// current exit digit.
			if move := s.dfsChainSearch(board, biCells, adj, visited, startIdx, startDigit, exitDigit, startIdx, 1, []int{startIdx}); move != nil {
				return move
			}
		}
	}

	return nil
}

// dfsChainSearch recursively follows the chain, looking for endpoints where
// the exit digit matches startDigit.
func (s *XYChainSolver) dfsChainSearch(board *core.Board, biCells []xyChainCell, adj [][]xyLink, visited []bool, startIdx int, startDigit int, currentExitDigit int, currentIdx int, depth int, chain []int) *Move {
	// Limit chain depth to prevent excessive computation.
	if depth > 15 {
		return nil
	}

	for _, lk := range adj[currentIdx] {
		nextIdx := lk.to
		if visited[nextIdx] {
			continue
		}

		// The entering digit of the next cell must match our current exit digit.
		if lk.enterExit[0] != currentExitDigit {
			continue
		}

		nextExitDigit := lk.enterExit[1]

		// Check if this creates a useful chain:
		// The chain starts at biCells[startIdx] with startDigit as one endpoint.
		// The chain ends at biCells[nextIdx] with nextExitDigit as the other endpoint.
		// If startDigit == nextExitDigit, we can eliminate that digit from cells
		// seeing both the start and end.
		if depth >= 2 && startDigit == nextExitDigit {
			startPos := biCells[startIdx].pos
			endPos := biCells[nextIdx].pos

			// Don't process if start and end share a unit (XY-Wing already handles that).
			// Actually, XY-Wing handles length-2 chains. We handle length >= 2 here,
			// but skip length 2 (which is exactly XY-Wing).
			if depth == 2 {
				// This is equivalent to XY-Wing — skip to avoid duplicates.
				// Only process chains of length >= 3.
			} else {
				// Eliminate startDigit from cells seeing both start and end.
				completeChain := append(append([]int(nil), chain...), nextIdx)
				if move := s.eliminateChainEndpoints(board, biCells, completeChain, startPos, endPos, startDigit); move != nil {
					return move
				}
			}
		}

		// Continue the chain.
		visited[nextIdx] = true
		nextChain := append(append([]int(nil), chain...), nextIdx)
		if move := s.dfsChainSearch(board, biCells, adj, visited, startIdx, startDigit, nextExitDigit, nextIdx, depth+1, nextChain); move != nil {
			return move
		}
		visited[nextIdx] = false
	}

	return nil
}

// eliminateChainEndpoints eliminates digit from cells seeing both chain endpoints.
func (s *XYChainSolver) eliminateChainEndpoints(board *core.Board, biCells []xyChainCell, chain []int, startPos, endPos core.Position, digit int) *Move {
	var move *Move
	eliminated := false
	evidence := xyChainEvidence(biCells, chain)

	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			pos := core.NewPosition(row, col)
			if pos == startPos || pos == endPos {
				continue
			}
			if !sharesUnit(pos, startPos) || !sharesUnit(pos, endPos) {
				continue
			}
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
							"XY-Chain: chain from %s to %s proves %d in one endpoint — %d eliminated from %s, leaving %d",
							startPos.ToString(), endPos.ToString(), digit,
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
				"XY-Chain: chain from %s to %s — %d eliminated from cells seeing both ends",
				startPos.ToString(), endPos.ToString(), digit,
			),
			Evidence: evidence,
		}
	}

	return nil
}

func xyChainEvidence(cells []xyChainCell, chain []int) *Evidence {
	evidence := &Evidence{}
	for _, index := range chain {
		cell := cells[index]
		values := []int{cell.vals[0], cell.vals[1]}
		evidence.Chain = append(evidence.Chain, CandidateGroup{Position: cell.pos, Values: values})
		evidence.Premises = append(evidence.Premises, CandidateGroup{Position: cell.pos, Values: values})
	}
	for index := 0; index+1 < len(chain); index++ {
		left, right := cells[chain[index]], cells[chain[index+1]]
		shared := 0
		for _, first := range left.vals {
			for _, second := range right.vals {
				if first == second {
					shared = first
				}
			}
		}
		evidence.Links = append(evidence.Links, CandidateLink{
			From: CandidateRef{Position: left.pos, Value: shared},
			To:   CandidateRef{Position: right.pos, Value: shared},
			Kind: LinkImplication,
		})
	}
	return evidence
}
