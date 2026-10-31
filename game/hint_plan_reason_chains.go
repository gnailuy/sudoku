package game

import (
	"fmt"
	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/message"
	"github.com/gnailuy/sudoku/solver"
)

func coloringSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	conclusion := HintConclusion{Placement: placement(move)}
	if evidence != nil {
		conclusion.Eliminations = evidence.Eliminations
	}
	if evidence == nil || len(evidence.Partitions) != 2 || len(evidence.Premises) == 0 || len(evidence.Premises[0].Values) != 1 || len(evidence.Links) == 0 {
		return evidenceSteps(move, conclusion)
	}
	digit := evidence.Premises[0].Values[0]
	groupCells := func(partition solver.CandidatePartition) []core.Position {
		cells := make([]core.Position, len(partition.Candidates))
		for index, candidate := range partition.Candidates {
			if candidate.Value != digit {
				return nil
			}
			cells[index] = candidate.Position
		}
		return cells
	}
	groupA := groupCells(evidence.Partitions[0])
	groupB := groupCells(evidence.Partitions[1])
	if len(groupA) == 0 || len(groupB) == 0 {
		return evidenceSteps(move, conclusion)
	}
	marks := premiseCandidateMarks(evidence)
	nodes := visualNodes(evidence, false)
	links := visualLinks(evidence)
	firstNodes, firstLinks := visualPrefix(nodes, links)
	if len(firstNodes) != 2 {
		return evidenceSteps(move, conclusion)
	}
	unit, ok := sharedHintUnit(*firstNodes[0].Target.Position, *firstNodes[1].Target.Position)
	if !ok {
		return evidenceSteps(move, conclusion)
	}
	pair := hintDescriptor("hint.coloring.pair", unitsArgument("unit", []solver.UnitRef{unit}), digitArgument("digit", digit), cellsArgument("cells", []core.Position{*firstNodes[0].Target.Position, *firstNodes[1].Target.Position}))
	chain := hintDescriptor("hint.coloring.chain", digitArgument("digit", digit), cellsArgument("group_a", groupA), cellsArgument("group_b", groupB))
	alternatives := hintDescriptor("hint.coloring.alternatives", digitArgument("digit", digit))
	steps := []HintStep{
		{ID: "find-simple-coloring-pair", Kind: HintStepObserve, Message: pair, Narration: []HintNarrationChunk{narrationMessage(pair)}, Marks: marksForNodes(firstNodes), Nodes: firstNodes, Links: firstLinks},
		{ID: "extend-simple-coloring-chain", Kind: HintStepExplain, Message: chain, Narration: []HintNarrationChunk{narrationMessage(chain)}, Marks: marks, Nodes: nodes, Links: links},
		{ID: "compare-simple-coloring-groups", Kind: HintStepExplain, Message: alternatives, Narration: []HintNarrationChunk{narrationMessage(alternatives)}, Marks: marks, Nodes: nodes, Links: links},
	}
	for index, eliminated := range conclusion.Eliminations {
		groupACell, groupBCell, found := coloringWitnesses(eliminated.Position, groupA, groupB)
		if !found {
			return evidenceSteps(move, conclusion)
		}
		explain := hintDescriptor("hint.coloring.elimination", digitArgument("digit", digit), cellArgument("cell", eliminated.Position), cellArgument("group_a_cell", groupACell), cellArgument("group_b_cell", groupBCell))
		stepMarks := append([]HintMark(nil), marks...)
		stepMarks = append(stepMarks, HintMark{Target: candidateTarget(eliminated), Role: HintRoleFocus})
		steps = append(steps, HintStep{ID: fmt.Sprintf("explain-simple-coloring-elimination-%d", index+1), Kind: HintStepExplain, Message: explain, Narration: []HintNarrationChunk{narrationMessage(explain)}, Marks: stepMarks, Nodes: nodes, Links: links})
	}
	apply := coloringActionDescriptor(move, conclusion, digit)
	steps = append(steps, HintStep{ID: "apply-simple-coloring", Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(marks, conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)})
	return steps
}

func xCycleSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	conclusion := HintConclusion{Placement: placement(move)}
	if evidence != nil {
		conclusion.Eliminations = evidence.Eliminations
	}
	if evidence == nil || len(evidence.Chain) < 3 || len(evidence.Links) != len(evidence.Chain) {
		return evidenceSteps(move, conclusion)
	}
	digit := evidence.Links[0].From.Value
	positions := make([]core.Position, len(evidence.Chain))
	for index, group := range evidence.Chain {
		if len(group.Values) == 0 {
			return evidenceSteps(move, conclusion)
		}
		positions[index] = group.Position
	}
	for index, link := range evidence.Links {
		if link.From.Position != positions[index] || link.To.Position != positions[(index+1)%len(positions)] || link.From.Value != digit || link.To.Value != digit {
			return evidenceSteps(move, conclusion)
		}
	}

	var target core.Position
	wantKind := solver.LinkWeak
	if conclusion.Placement != nil {
		target = conclusion.Placement.Position
		wantKind = solver.LinkStrong
		for _, elimination := range conclusion.Eliminations {
			if elimination.Position == target {
				wantKind = solver.LinkWeak
				break
			}
		}
	} else if len(conclusion.Eliminations) > 0 {
		target = conclusion.Eliminations[0].Position
	} else {
		return evidenceSteps(move, conclusion)
	}
	start := -1
	for index, position := range positions {
		previous := (index + len(positions) - 1) % len(positions)
		if position == target && evidence.Links[index].Kind == wantKind && evidence.Links[previous].Kind == wantKind {
			start = index
			break
		}
	}
	if start < 0 {
		return evidenceSteps(move, conclusion)
	}
	rotatePositions := append(append([]core.Position(nil), positions[start:]...), positions[:start]...)
	rotateLinks := append(append([]solver.CandidateLink(nil), evidence.Links[start:]...), evidence.Links[:start]...)
	for index, link := range rotateLinks {
		want := wantKind
		if index > 0 && index < len(rotateLinks)-1 {
			if wantKind == solver.LinkStrong {
				if index%2 == 1 {
					want = solver.LinkWeak
				}
			} else if index%2 == 1 {
				want = solver.LinkStrong
			}
		}
		if link.Kind != want {
			return evidenceSteps(move, conclusion)
		}
	}
	allNodes := visualNodes(evidence, true)
	hintLinks := func(links []solver.CandidateLink) []HintLink {
		result := make([]HintLink, len(links))
		for index, link := range links {
			result[index] = HintLink{From: candidateNodeID(link.From), To: candidateNodeID(link.To), Kind: link.Kind}
		}
		return result
	}
	nodesFor := func(positions []core.Position) []HintNode {
		seen := make(map[core.Position]struct{}, len(positions))
		for _, position := range positions {
			seen[position] = struct{}{}
		}
		result := make([]HintNode, 0, len(positions))
		for _, node := range allNodes {
			if node.Target.Position != nil {
				if _, ok := seen[*node.Target.Position]; ok {
					result = append(result, node)
				}
			}
		}
		return result
	}

	steps := make([]HintStep, 0, len(rotateLinks)/2+2)
	if wantKind == solver.LinkStrong {
		unit, ok := sharedHintUnit(rotatePositions[0], rotatePositions[1])
		if !ok {
			return evidenceSteps(move, conclusion)
		}
		opening := hintDescriptor("hint.x-cycle.opening", unitsArgument("unit", []solver.UnitRef{unit}), digitArgument("digit", digit), cellsArgument("cells", rotatePositions[:2]))
		openingNodes := nodesFor(rotatePositions[:2])
		steps = append(steps, HintStep{ID: "find-x-cycle-opening-pair", Kind: HintStepObserve, Message: opening, Narration: []HintNarrationChunk{narrationMessage(opening)}, Marks: marksForNodes(openingNodes), Nodes: openingNodes, Links: hintLinks(rotateLinks[:1])})
		for weak := 1; weak+1 < len(rotateLinks); weak += 2 {
			blocked := rotatePositions[weak+1]
			result := rotatePositions[(weak+2)%len(rotatePositions)]
			unit, ok := sharedHintUnit(blocked, result)
			if !ok {
				return evidenceSteps(move, conclusion)
			}
			var descriptor message.Descriptor
			if weak == 1 {
				descriptor = hintDescriptor("hint.x-cycle.assume", cellArgument("start", rotatePositions[0]), cellArgument("forced", rotatePositions[1]), cellArgument("blocked", blocked), unitsArgument("unit", []solver.UnitRef{unit}), cellArgument("result", result), digitArgument("digit", digit))
			} else if result == rotatePositions[0] {
				descriptor = hintDescriptor("hint.x-cycle.close", cellArgument("true_cell", rotatePositions[weak]), cellArgument("blocked", blocked), unitsArgument("unit", []solver.UnitRef{unit}), cellArgument("result", result), digitArgument("digit", digit))
			} else {
				descriptor = hintDescriptor("hint.x-cycle.follow", cellArgument("true_cell", rotatePositions[weak]), cellArgument("blocked", blocked), unitsArgument("unit", []solver.UnitRef{unit}), cellArgument("result", result), digitArgument("digit", digit))
			}
			count := weak + 3
			if count > len(rotatePositions) {
				count = len(rotatePositions)
			}
			stepNodes := nodesFor(rotatePositions[:count])
			steps = append(steps, HintStep{ID: fmt.Sprintf("follow-x-cycle-%d", weak), Kind: HintStepExplain, Message: descriptor, Narration: []HintNarrationChunk{narrationMessage(descriptor)}, Marks: marksForNodes(stepNodes), Nodes: stepNodes, Links: hintLinks(rotateLinks[:weak+2])})
		}
		apply := hintDescriptor("hint.x-cycle.place", cellArgument("assumption", rotatePositions[0]), digitArgument("digit", digit), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
		steps = append(steps, HintStep{ID: "apply-x-cycle", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: append(marksForNodes(allNodes), conclusionMarks(conclusion)...), Nodes: allNodes, Links: hintLinks(rotateLinks), Effect: conclusionEffect(conclusion)})
		return steps
	}

	unit, ok := sharedHintUnit(rotatePositions[1], rotatePositions[2])
	if !ok {
		return evidenceSteps(move, conclusion)
	}
	opening := hintDescriptor("hint.x-cycle.opening", unitsArgument("unit", []solver.UnitRef{unit}), digitArgument("digit", digit), cellsArgument("cells", rotatePositions[1:3]))
	openingNodes := nodesFor(rotatePositions[1:3])
	steps = append(steps, HintStep{ID: "find-x-cycle-opening-pair", Kind: HintStepObserve, Message: opening, Narration: []HintNarrationChunk{narrationMessage(opening)}, Marks: marksForNodes(openingNodes), Nodes: openingNodes, Links: hintLinks(rotateLinks[1:2])})
	for weak := 2; weak+1 < len(rotateLinks)-1; weak += 2 {
		blocked := rotatePositions[weak+1]
		result := rotatePositions[weak+2]
		unit, ok := sharedHintUnit(blocked, result)
		if !ok {
			return evidenceSteps(move, conclusion)
		}
		descriptor := hintDescriptor("hint.x-cycle.follow", cellArgument("true_cell", rotatePositions[weak]), cellArgument("blocked", blocked), unitsArgument("unit", []solver.UnitRef{unit}), cellArgument("result", result), digitArgument("digit", digit))
		stepNodes := nodesFor(rotatePositions[1 : weak+3])
		steps = append(steps, HintStep{ID: fmt.Sprintf("follow-x-cycle-%d", weak), Kind: HintStepExplain, Message: descriptor, Narration: []HintNarrationChunk{narrationMessage(descriptor)}, Marks: marksForNodes(stepNodes), Nodes: stepNodes, Links: hintLinks(rotateLinks[1 : weak+2])})
	}
	cells := make([]core.Position, len(conclusion.Eliminations))
	for index, elimination := range conclusion.Eliminations {
		cells[index] = elimination.Position
	}
	var apply message.Descriptor
	if conclusion.Placement != nil {
		apply = hintDescriptor("hint.relational-wing.place", digitArgument("removed_digit", digit), digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
	} else {
		apply = hintDescriptor("hint.implication-chain.apply", digitArgument("digit", digit), cellsArgument("cells", cells), strategyArgument("strategy", move.Technique))
	}
	steps = append(steps, HintStep{ID: "apply-x-cycle", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: append(marksForNodes(allNodes), conclusionMarks(conclusion)...), Nodes: allNodes, Links: hintLinks(rotateLinks), Effect: conclusionEffect(conclusion)})
	return steps
}

func coloringActionDescriptor(move *solver.Move, conclusion HintConclusion, digit int) message.Descriptor {
	if conclusion.Placement != nil {
		return hintDescriptor("hint.coloring.place", digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), digitArgument("colored_digit", digit), strategyArgument("strategy", move.Technique))
	}
	return hintDescriptor("hint.coloring.apply", candidatesArgument("candidates", conclusion.Eliminations), digitArgument("colored_digit", digit), strategyArgument("strategy", move.Technique))
}

func coloringWitnesses(target core.Position, groupA, groupB []core.Position) (core.Position, core.Position, bool) {
	var a, b core.Position
	foundA, foundB := false, false
	for _, position := range groupA {
		if _, ok := sharedHintUnit(target, position); ok {
			a, foundA = position, true
			break
		}
	}
	for _, position := range groupB {
		if _, ok := sharedHintUnit(target, position); ok {
			b, foundB = position, true
			break
		}
	}
	return a, b, foundA && foundB
}

func xyChainDetailedSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	conclusion := HintConclusion{Placement: placement(move)}
	if evidence != nil {
		conclusion.Eliminations = evidence.Eliminations
	}
	if evidence == nil || conclusion.Placement == nil || len(evidence.Chain) != 8 || len(evidence.Links) != 7 {
		return evidenceSteps(move, conclusion)
	}
	endpointValues := sharedValues(evidence.Chain[0], evidence.Chain[len(evidence.Chain)-1])
	if len(endpointValues) != 1 {
		return evidenceSteps(move, conclusion)
	}
	endpoint := endpointValues[0]
	positions := make([]core.Position, len(evidence.Chain))
	forced := make([]int, len(evidence.Chain))
	for index, group := range evidence.Chain {
		if len(group.Values) != 2 {
			return evidenceSteps(move, conclusion)
		}
		positions[index] = group.Position
	}
	forced[0] = otherHintDigit(evidence.Chain[0].Values, endpoint)
	if forced[0] == 0 {
		return evidenceSteps(move, conclusion)
	}
	for index, relation := range evidence.Links {
		if relation.Kind != solver.LinkImplication || relation.From.Position != positions[index] || relation.To.Position != positions[index+1] || relation.From.Value != relation.To.Value {
			return evidenceSteps(move, conclusion)
		}
		forced[index+1] = otherHintDigit(evidence.Chain[index+1].Values, relation.To.Value)
		if forced[index+1] == 0 {
			return evidenceSteps(move, conclusion)
		}
	}
	if forced[len(forced)-1] != endpoint {
		return evidenceSteps(move, conclusion)
	}
	units := make([]solver.UnitRef, len(evidence.Links))
	for index, relation := range evidence.Links {
		unit, ok := sharedHintUnit(relation.From.Position, relation.To.Position)
		if !ok {
			return evidenceSteps(move, conclusion)
		}
		units[index] = unit
	}

	nodes := visualNodes(evidence, true)
	links := visualLinks(evidence)
	scene := func(cellCount, linkCount int) ([]HintNode, []HintLink) {
		visiblePositions := make(map[core.Position]struct{}, cellCount)
		for _, position := range positions[:cellCount] {
			visiblePositions[position] = struct{}{}
		}
		visibleNodes := make([]HintNode, 0, cellCount*2)
		for _, node := range nodes {
			if node.Target.Position != nil {
				if _, ok := visiblePositions[*node.Target.Position]; ok {
					visibleNodes = append(visibleNodes, node)
				}
			}
		}
		return visibleNodes, append([]HintLink(nil), links[:linkCount]...)
	}
	step1Nodes, step1Links := scene(1, 0)
	step2Nodes, step2Links := scene(3, 2)
	step3Nodes, step3Links := scene(5, 4)
	step4Nodes, step4Links := scene(7, 6)

	start := hintDescriptor("hint.xy-chain.start", cellArgument("cell", positions[0]), digitArgument("first_digit", evidence.Chain[0].Values[0]), digitArgument("second_digit", evidence.Chain[0].Values[1]), digitArgument("endpoint_digit", endpoint), digitArgument("other_digit", forced[0]))
	assume := hintDescriptor("hint.xy-chain.assume", cellArgument("start", positions[0]), digitArgument("endpoint_digit", endpoint), digitArgument("start_value", forced[0]), cellArgument("middle", positions[1]), digitArgument("middle_value", forced[1]), cellArgument("result", positions[2]), digitArgument("result_value", forced[2]), unitsArgument("first_unit", units[:1]), unitsArgument("second_unit", units[1:2]))
	follow1 := hintDescriptor("hint.xy-chain.follow", cellArgument("start", positions[2]), digitArgument("start_value", forced[2]), cellArgument("middle", positions[3]), digitArgument("middle_value", forced[3]), cellArgument("result", positions[4]), digitArgument("result_value", forced[4]), unitsArgument("first_unit", units[2:3]), unitsArgument("second_unit", units[3:4]))
	follow2 := hintDescriptor("hint.xy-chain.follow", cellArgument("start", positions[4]), digitArgument("start_value", forced[4]), cellArgument("middle", positions[5]), digitArgument("middle_value", forced[5]), cellArgument("result", positions[6]), digitArgument("result_value", forced[6]), unitsArgument("first_unit", units[4:5]), unitsArgument("second_unit", units[5:6]))
	reach := hintDescriptor("hint.xy-chain.reach", cellArgument("start", positions[6]), digitArgument("start_value", forced[6]), cellArgument("endpoint", positions[7]), digitArgument("shared_digit", evidence.Links[6].To.Value), digitArgument("endpoint_digit", endpoint))
	apply := hintDescriptor("hint.xy-chain.place", cellArgument("first_endpoint", positions[0]), cellArgument("second_endpoint", positions[7]), digitArgument("endpoint_digit", endpoint), cellArgument("cell", conclusion.Placement.Position), digitArgument("digit", conclusion.Placement.Value), strategyArgument("strategy", move.Technique))
	return []HintStep{
		{ID: "start-xy-chain", Kind: HintStepObserve, Message: start, Narration: []HintNarrationChunk{narrationMessage(start)}, Marks: marksForNodes(step1Nodes), Nodes: step1Nodes, Links: step1Links},
		{ID: "carry-xy-chain-across-row-one", Kind: HintStepExplain, Message: assume, Narration: []HintNarrationChunk{narrationMessage(assume)}, Marks: marksForNodes(step2Nodes), Nodes: step2Nodes, Links: step2Links},
		{ID: "continue-xy-chain-into-row-two", Kind: HintStepExplain, Message: follow1, Narration: []HintNarrationChunk{narrationMessage(follow1)}, Marks: marksForNodes(step3Nodes), Nodes: step3Nodes, Links: step3Links},
		{ID: "follow-xy-chain-across-row-two", Kind: HintStepExplain, Message: follow2, Narration: []HintNarrationChunk{narrationMessage(follow2)}, Marks: marksForNodes(step4Nodes), Nodes: step4Nodes, Links: step4Links},
		{ID: "reach-xy-chain-endpoint", Kind: HintStepExplain, Message: reach, Narration: []HintNarrationChunk{narrationMessage(reach)}, Marks: marksForNodes(nodes), Nodes: nodes, Links: links},
		{ID: "apply-xy-chain", Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(marksForNodes(nodes), conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)},
	}
}

func xyChainSteps(move *solver.Move, display string) []HintStep {
	evidence := move.Evidence
	if evidence != nil && move.IsPlacement() && len(evidence.Chain) == 8 && len(evidence.Links) == 7 && len(sharedValues(evidence.Chain[0], evidence.Chain[len(evidence.Chain)-1])) == 1 {
		if detailed := xyChainDetailedSteps(move, display); len(detailed) > 0 && detailed[0].Message.ID != "" {
			return detailed
		}
	}
	conclusion := HintConclusion{Placement: placement(move)}
	if evidence != nil {
		conclusion.Eliminations = evidence.Eliminations
	}
	if evidence == nil || len(evidence.Chain) < 2 || len(evidence.Links) != len(evidence.Chain)-1 {
		return evidenceSteps(move, conclusion)
	}
	endpoint := 0
	if len(evidence.Eliminations) > 0 {
		endpoint = evidence.Eliminations[0].Value
		for _, elimination := range evidence.Eliminations[1:] {
			if elimination.Value != endpoint {
				return evidenceSteps(move, conclusion)
			}
		}
	} else {
		endpointValues := sharedValues(evidence.Chain[0], evidence.Chain[len(evidence.Chain)-1])
		if len(endpointValues) == 1 {
			endpoint = endpointValues[0]
		}
	}
	if endpoint == 0 || !containsValue(evidence.Chain[0].Values, endpoint) || !containsValue(evidence.Chain[len(evidence.Chain)-1].Values, endpoint) {
		return evidenceSteps(move, conclusion)
	}
	positions := make([]core.Position, len(evidence.Chain))
	for index, group := range evidence.Chain {
		if len(group.Values) != 2 {
			return evidenceSteps(move, conclusion)
		}
		positions[index] = group.Position
	}
	for index, relation := range evidence.Links {
		if relation.Kind != solver.LinkImplication || relation.From.Position != positions[index] || relation.To.Position != positions[index+1] || relation.From.Value != relation.To.Value {
			return evidenceSteps(move, conclusion)
		}
	}
	if conclusion.Placement == nil && len(conclusion.Eliminations) == 0 {
		return evidenceSteps(move, conclusion)
	}

	nodes := visualNodes(evidence, true)
	links := visualLinks(evidence)
	scene := func(cellCount, linkCount int) ([]HintNode, []HintLink) {
		visiblePositions := make(map[core.Position]struct{}, cellCount)
		for _, position := range positions[:cellCount] {
			visiblePositions[position] = struct{}{}
		}
		visibleNodes := make([]HintNode, 0, cellCount*2)
		for _, node := range nodes {
			if node.Target.Position != nil {
				if _, ok := visiblePositions[*node.Target.Position]; ok {
					visibleNodes = append(visibleNodes, node)
				}
			}
		}
		return visibleNodes, append([]HintLink(nil), links[:linkCount]...)
	}
	start := hintDescriptor("hint.implication-chain.observe", cellsArgument("cell", positions[:1]), digitsArgument("digits", evidence.Chain[0].Values))
	startNodes, startLinks := scene(1, 0)
	steps := []HintStep{{ID: "start-xy-chain", Kind: HintStepObserve, Message: start, Narration: []HintNarrationChunk{narrationMessage(start)}, Marks: marksForNodes(startNodes), Nodes: startNodes, Links: startLinks}}
	for index, relation := range evidence.Links {
		explain := hintDescriptor("hint.implication-chain.explain", candidateArgument("from", relation.From), candidateArgument("to", relation.To))
		stepNodes, stepLinks := scene(index+2, index+1)
		steps = append(steps, HintStep{ID: fmt.Sprintf("follow-xy-chain-%d", index+1), Kind: HintStepExplain, Message: explain, Narration: []HintNarrationChunk{narrationMessage(explain)}, Marks: marksForNodes(stepNodes), Nodes: stepNodes, Links: stepLinks})
	}
	var apply message.Descriptor
	if conclusion.Placement != nil {
		apply = hintDescriptor("hint.xy-chain.place", cellArgument("first_endpoint", positions[0]), cellArgument("second_endpoint", positions[len(positions)-1]), digitArgument("endpoint_digit", endpoint), cellArgument("cell", conclusion.Placement.Position), digitArgument("digit", conclusion.Placement.Value), strategyArgument("strategy", move.Technique))
	} else {
		cells := make([]core.Position, len(conclusion.Eliminations))
		for index, elimination := range conclusion.Eliminations {
			cells[index] = elimination.Position
		}
		apply = hintDescriptor("hint.implication-chain.apply", digitArgument("digit", endpoint), cellsArgument("cells", cells), strategyArgument("strategy", move.Technique))
	}
	steps = append(steps, HintStep{ID: "apply-xy-chain", Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(marksForNodes(nodes), conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)})
	return steps
}

func otherHintDigit(values []int, excluded int) int {
	for _, value := range values {
		if value != excluded {
			return value
		}
	}
	return 0
}
