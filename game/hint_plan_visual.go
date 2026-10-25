package game

import (
	"fmt"
	"sort"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

func candidateNodeID(ref solver.CandidateRef) string {
	return fmt.Sprintf("r%dc%d-v%d", ref.Position.Row+1, ref.Position.Column+1, ref.Value)
}

func visualNodes(evidence *solver.Evidence, ordered bool) []HintNode {
	groups := make(map[string]string)
	for index, partition := range evidence.Partitions {
		for _, ref := range partition.Candidates {
			groups[candidateNodeID(ref)] = string(rune('A' + index))
		}
	}
	order := make(map[core.Position]int)
	if ordered {
		for index, group := range evidence.Chain {
			order[group.Position] = index + 1
		}
	}
	seen := make(map[string]struct{})
	var nodes []HintNode
	for _, premise := range evidence.Premises {
		for _, value := range premise.Values {
			ref := solver.CandidateRef{Position: premise.Position, Value: value}
			id := candidateNodeID(ref)
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			nodeOrder := order[ref.Position]
			if len(premise.Values) > 0 && premise.Values[0] != value {
				nodeOrder = 0
			}
			nodes = append(nodes, HintNode{ID: id, Target: candidateTarget(ref), Group: groups[id], Order: nodeOrder})
		}
	}
	return nodes
}

func visualLinks(evidence *solver.Evidence) []HintLink {
	links := make([]HintLink, len(evidence.Links))
	for index, link := range evidence.Links {
		links[index] = HintLink{From: candidateNodeID(link.From), To: candidateNodeID(link.To), Kind: link.Kind}
	}
	return links
}

func visualPrefix(nodes []HintNode, links []HintLink) ([]HintNode, []HintLink) {
	if len(links) == 0 {
		return nodes, links
	}
	first := links[:1]
	ids := map[string]struct{}{first[0].From: {}, first[0].To: {}}
	var selected []HintNode
	for _, node := range nodes {
		if _, ok := ids[node.ID]; ok {
			selected = append(selected, node)
		}
	}
	return selected, first
}

func premiseCandidateMarks(evidence *solver.Evidence) []HintMark {
	var marks []HintMark
	for _, premise := range evidence.Premises {
		for _, value := range premise.Values {
			marks = append(marks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: value}), Role: HintRolePremise})
		}
	}
	return marks
}

func marksForNodes(nodes []HintNode) []HintMark {
	marks := make([]HintMark, len(nodes))
	for index, node := range nodes {
		marks[index] = HintMark{Target: node.Target, Role: HintRolePremise}
	}
	return marks
}

func nodesForCandidateRefs(nodes []HintNode, refs ...solver.CandidateRef) []HintNode {
	wanted := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		wanted[candidateNodeID(ref)] = struct{}{}
	}
	selected := make([]HintNode, 0, len(wanted))
	for _, node := range nodes {
		if _, ok := wanted[node.ID]; ok {
			selected = append(selected, node)
		}
	}
	return selected
}

func containsValue(values []int, wanted int) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func sharedValues(groups ...solver.CandidateGroup) []int {
	if len(groups) == 0 {
		return nil
	}
	counts := make(map[int]int)
	for _, group := range groups {
		seen := make(map[int]struct{})
		for _, value := range group.Values {
			seen[value] = struct{}{}
		}
		for value := range seen {
			counts[value]++
		}
	}
	var values []int
	for value, count := range counts {
		if count == len(groups) {
			values = append(values, value)
		}
	}
	sort.Ints(values)
	return values
}

func combinedValues(groups ...solver.CandidateGroup) []int {
	seen := make(map[int]struct{})
	for _, group := range groups {
		for _, value := range group.Values {
			seen[value] = struct{}{}
		}
	}
	values := make([]int, 0, len(seen))
	for value := range seen {
		values = append(values, value)
	}
	sort.Ints(values)
	return values
}

func unitAndPremiseMarks(evidence *solver.Evidence) []HintMark {
	var marks []HintMark
	for _, unitRef := range evidence.Units {
		unit := unitRef
		if target, ok := unitTarget(&unit); ok {
			marks = append(marks, HintMark{Target: target, Role: HintRoleFocus})
		}
	}
	for _, premise := range evidence.Premises {
		marks = append(marks, HintMark{Target: cellTarget(premise.Position), Role: HintRoleFocus})
		for _, value := range premise.Values {
			marks = append(marks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: value}), Role: HintRolePremise})
		}
	}
	return marks
}

func placement(move *solver.Move) *core.Cell {
	if move.IsPlacement() {
		cell := move.Cell
		return &cell
	}
	return nil
}

func conclusionKind(conclusion HintConclusion) HintStepKind {
	if conclusion.Placement != nil {
		return HintStepApply
	}
	return HintStepApply
}

func conclusionMarks(conclusion HintConclusion) []HintMark {
	if conclusion.Placement != nil {
		return []HintMark{{Target: candidateTarget(solver.CandidateRef{Position: conclusion.Placement.Position, Value: conclusion.Placement.Value}), Role: HintRoleConclusion}}
	}
	marks := make([]HintMark, len(conclusion.Eliminations))
	for i, ref := range conclusion.Eliminations {
		marks[i] = HintMark{Target: candidateTarget(ref), Role: HintRoleConclusion}
	}
	return marks
}

func conclusionEffect(conclusion HintConclusion) *HintEffect {
	return &HintEffect{Placement: conclusion.Placement, Eliminations: append([]solver.CandidateRef(nil), conclusion.Eliminations...)}
}
