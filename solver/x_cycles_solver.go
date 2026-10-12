package solver

import (
	"fmt"

	"github.com/gnailuy/sudoku/core"
)

// XCyclesSolver finds eliminations using X-Cycles (alternating inference chains
// for a single digit).
//
// X-Cycles extend Simple Coloring by using both strong links (conjugate pairs)
// and weak links (two cells with the same candidate in the same unit, but not
// a conjugate pair — there may be other cells with that candidate).
//
// The solver builds a graph where edges are either strong or weak, then
// searches for:
//
// Rule 1 (Nice Loop — even length, all links alternate):
// If a continuous loop alternates strong-weak-strong-weak-..., then:
//   - In every weak link, if a cell outside the chain sees both endpoints,
//     the digit can be eliminated from that cell.
//
// Rule 2 (Discontinuous Loop — strong link at both ends of the chain into one cell):
// If a chain from cell A to cell B (both weakly linked to cell X) has
// alternating links, and A and B both have strong connections that converge:
//   - The digit can be placed in X (if both ends are strong into X).
//   - The digit can be eliminated from X (if both ends are weak into X).
//
// For simplicity and correctness, we implement the two most common X-Cycle
// deduction rules:
//
// Type 1 (Discontinuous Nice Loop — off-chain elimination):
// A chain starts and ends at the same cell with two strong links. The
// digit must be in that cell (it's true in both colorings).
//
// Type 2 (Discontinuous Nice Loop — off-chain elimination):
// A chain starts and ends at the same cell with two weak links. The
// digit can be eliminated from that cell.
//
// Type 3 (Continuous Nice Loop):
// Same as Simple Coloring Rule 2 — cells seeing both ends of any weak
// link in the loop can eliminate the digit.
type XCyclesSolver struct {
	Base
}

// NewXCyclesSolver creates an XCyclesSolver.
func NewXCyclesSolver() *XCyclesSolver {
	return &XCyclesSolver{
		Base: Base{
			Key:         "x-cycles",
			DisplayName: "X-Cycles",
			Description: "Single-digit alternating inference chains using strong and weak links for eliminations.",
			Weight:      WeightXCycles,
		},
	}
}

// xcEdge represents an edge in the X-Cycles graph.
type xcEdge struct {
	to     core.Position
	strong bool // true = strong link (conjugate pair), false = weak link
}

// Apply checks for X-Cycle patterns on all digits.
func (s *XCyclesSolver) Apply(board *core.Board) *Move {
	for digit := 1; digit <= 9; digit++ {
		if move := s.findXCycles(board, digit); move != nil {
			return move
		}
	}
	return nil
}

// findXCycles builds the link graph for a digit and searches for useful cycles.
func (s *XCyclesSolver) findXCycles(board *core.Board, digit int) *Move {
	// Build adjacency graph with strong/weak classification.
	// A strong link exists between two cells in a unit if the digit appears
	// in exactly those two cells. A weak link exists if it appears in both
	// cells but also in other cells in the same unit.
	adj := make(map[core.Position][]xcEdge)
	units := allUnits()

	// For each unit, find cells with the digit.
	for _, u := range units {
		var cells []core.Position
		for _, pos := range u.positions {
			if board.Get(pos) == 0 && board.Candidates(pos).Has(digit) {
				cells = append(cells, pos)
			}
		}
		if len(cells) < 2 {
			continue
		}
		isStrong := (len(cells) == 2)

		for i := 0; i < len(cells); i++ {
			for j := i + 1; j < len(cells); j++ {
				// Add edge in both directions.
				s.addEdge(adj, cells[i], cells[j], isStrong)
			}
		}
	}

	// Collect all nodes (cells with the digit as candidate).
	nodes := sortedPositions(adj)

	// Search for useful alternating chains using DFS.
	// We look for chains that start with a strong link and end at the
	// starting cell, creating a cycle.

	// Type 2: Discontinuous nice loop with two weak links at the start cell.
	// If we find a chain: start -weak-> A -strong-> B -weak-> ... -strong-> C -weak-> start
	// then the digit is eliminated from start.
	for _, startPos := range nodes {
		if move := s.dfsXCycle(board, digit, adj, startPos); move != nil {
			return move
		}
	}

	return nil
}

// addEdge adds a bidirectional edge, upgrading to strong if a strong edge exists.
func (s *XCyclesSolver) addEdge(adj map[core.Position][]xcEdge, a, b core.Position, strong bool) {
	// Check if edge already exists; upgrade to strong if needed.
	s.addOneWay(adj, a, b, strong)
	s.addOneWay(adj, b, a, strong)
}

func (s *XCyclesSolver) addOneWay(adj map[core.Position][]xcEdge, from, to core.Position, strong bool) {
	for i, e := range adj[from] {
		if e.to == to {
			if strong {
				adj[from][i].strong = true
			}
			return
		}
	}
	adj[from] = append(adj[from], xcEdge{to: to, strong: strong})
}

// dfsXCycle performs a DFS from startPos to find alternating chains that
// return to startPos, producing useful eliminations.
func (s *XCyclesSolver) dfsXCycle(board *core.Board, digit int, adj map[core.Position][]xcEdge, startPos core.Position) *Move {
	// We need chains of length >= 4 (at least 2 links after start).
	// Chain alternates: the link from the previous cell determines the next
	// required link type (strong after weak, weak after strong).

	// Try starting with each edge type from startPos.
	for _, firstEdge := range adj[startPos] {
		visited := map[core.Position]bool{startPos: true}

		visited[firstEdge.to] = true

		path := []core.Position{startPos, firstEdge.to}
		linkKinds := []bool{firstEdge.strong}
		if move := s.dfsChain(board, digit, adj, startPos, firstEdge.to, !firstEdge.strong, 1, visited, firstEdge.strong, path, linkKinds); move != nil {
			return move
		}
	}

	return nil
}

// dfsChain recursively searches for an alternating chain back to startPos.
func (s *XCyclesSolver) dfsChain(board *core.Board, digit int, adj map[core.Position][]xcEdge, startPos, currPos core.Position, nextMustStrong bool, depth int, visited map[core.Position]bool, firstLinkStrong bool, path []core.Position, linkKinds []bool) *Move {
	// Limit chain length to prevent excessive computation.
	if depth > 20 {
		return nil
	}

	for _, edge := range adj[currPos] {
		if edge.strong != nextMustStrong {
			continue
		}

		// Check if we've returned to startPos (closing the loop).
		if edge.to == startPos && depth >= 3 {
			// We have a cycle: startPos -> ... -> currPos -> startPos.
			cycle := append(append([]core.Position(nil), path...), startPos)
			cycleLinks := append(append([]bool(nil), linkKinds...), edge.strong)
			evidence := xCycleEvidence(board, digit, cycle, cycleLinks)
			// The first link was firstLinkStrong, the last link is edge.strong.
			// For a proper alternating nice loop, both links at startPos
			// should have compatible types.

			// Type 1: Both links at start are strong (first and last).
			// The digit MUST be in startPos.
			if firstLinkStrong && edge.strong {
				// Place the digit at startPos.
				if board.Get(startPos) == 0 && board.Candidates(startPos).Has(digit) {
					return &Move{
						Cell:      core.NewCell(startPos, digit),
						Technique: s.Key,
						Reason: fmt.Sprintf(
							"X-Cycles (Type 1): alternating chain proves %d must be at %s",
							digit, startPos.ToString(),
						),
						Evidence: evidence,
					}
				}
			}

			// Type 2: Both links at start are weak.
			// The digit can be eliminated from startPos.
			if !firstLinkStrong && !edge.strong {
				if board.EliminateCandidate(startPos, digit) {
					evidence.Eliminations = []CandidateRef{{Position: startPos, Value: digit}}
					cands := board.Candidates(startPos)
					if cands.Count() == 1 {
						value := cands.Values()[0]
						return &Move{
							Cell:      core.NewCell(startPos, value),
							Technique: s.Key,
							Reason: fmt.Sprintf(
								"X-Cycles (Type 2): alternating chain eliminates %d from %s, leaving %d",
								digit, startPos.ToString(), value,
							),
							Evidence: evidence,
						}
					}
					return &Move{
						EliminationOnly: true,
						Technique:       s.Key,
						Reason: fmt.Sprintf(
							"X-Cycles (Type 2): alternating chain eliminates %d from %s",
							digit, startPos.ToString(),
						),
						Evidence: evidence,
					}
				}
			}

			// Type 3: Continuous nice loop (one strong, one weak at start).
			// Eliminate digit from cells that see both endpoints of any
			// weak link in the loop. This is complex to track without
			// storing the full path, so we skip it for now — Types 1 and 2
			// cover the most common cases.
			continue
		}

		if visited[edge.to] {
			continue
		}

		// Continue the chain.
		visited[edge.to] = true
		nextPath := append(append([]core.Position(nil), path...), edge.to)
		nextLinks := append(append([]bool(nil), linkKinds...), edge.strong)
		if move := s.dfsChain(board, digit, adj, startPos, edge.to, !nextMustStrong, depth+1, visited, firstLinkStrong, nextPath, nextLinks); move != nil {
			return move
		}
		delete(visited, edge.to)
	}

	return nil
}

func xCycleEvidence(board *core.Board, digit int, path []core.Position, strong []bool) *Evidence {
	evidence := &Evidence{}
	for index, position := range path {
		if index == len(path)-1 && position == path[0] {
			break
		}
		values := board.Candidates(position).Values()
		evidence.Chain = append(evidence.Chain, CandidateGroup{Position: position, Values: values})
		evidence.Premises = append(evidence.Premises, CandidateGroup{Position: position, Values: []int{digit}})
	}
	for index := 0; index+1 < len(path) && index < len(strong); index++ {
		kind := LinkWeak
		if strong[index] {
			kind = LinkStrong
		}
		evidence.Links = append(evidence.Links, CandidateLink{
			From: CandidateRef{Position: path[index], Value: digit},
			To:   CandidateRef{Position: path[index+1], Value: digit},
			Kind: kind,
		})
	}
	return evidence
}
