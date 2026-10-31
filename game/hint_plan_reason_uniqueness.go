package game

import (
	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/message"
	"github.com/gnailuy/sudoku/solver"
)

func sharedHintUnit(a, b core.Position) (solver.UnitRef, bool) {
	if a.Row == b.Row {
		return solver.UnitRef{Kind: solver.UnitRow, Index: a.Row}, true
	}
	if a.Column == b.Column {
		return solver.UnitRef{Kind: solver.UnitColumn, Index: a.Column}, true
	}
	if a.Row/3 == b.Row/3 && a.Column/3 == b.Column/3 {
		return solver.UnitRef{Kind: solver.UnitBox, Index: (a.Row/3)*3 + b.Column/3}, true
	}
	return solver.UnitRef{}, false
}

func bugPlusOneSteps(move *solver.Move, _ string) []HintStep {
	if move.Evidence == nil || len(move.Evidence.Premises) < 2 || !move.IsPlacement() {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	trivalue := move.Evidence.Premises[0]
	bivalue := move.Evidence.Premises[1:]
	if len(trivalue.Values) != 3 || len(move.Evidence.Units) != 3 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	for _, group := range bivalue {
		if len(group.Values) != 2 {
			return evidenceSteps(move, HintConclusion{Placement: placement(move)})
		}
	}

	extraDigit := move.Cell.Value
	otherDigits := make([]int, 0, 2)
	for _, digit := range trivalue.Values {
		if digit != extraDigit {
			otherDigits = append(otherDigits, digit)
		}
	}
	if len(otherDigits) != 2 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}

	bivalueMarks := make([]HintMark, 0, len(bivalue))
	for _, group := range bivalue {
		bivalueMarks = append(bivalueMarks, HintMark{Target: cellTarget(group.Position), Role: HintRolePremise})
	}
	trivalueMarks := append([]HintMark(nil), bivalueMarks...)
	trivalueMarks = append(trivalueMarks, HintMark{Target: cellTarget(trivalue.Position), Role: HintRoleFocus})
	for _, digit := range trivalue.Values {
		trivalueMarks = append(trivalueMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: trivalue.Position, Value: digit}), Role: HintRolePremise})
	}

	rowMarks := append([]HintMark(nil), trivalueMarks...)
	rowTarget, ok := unitTarget(&move.Evidence.Units[0])
	if !ok {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	rowMarks = append(rowMarks, HintMark{Target: rowTarget, Role: HintRoleFocus})
	columnMarks := append([]HintMark(nil), rowMarks...)
	columnTarget, ok := unitTarget(&move.Evidence.Units[1])
	if !ok {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	columnMarks = append(columnMarks, HintMark{Target: columnTarget, Role: HintRoleFocus})
	allUnitMarks := append([]HintMark(nil), columnMarks...)
	boxTarget, ok := unitTarget(&move.Evidence.Units[2])
	if !ok {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	allUnitMarks = append(allUnitMarks, HintMark{Target: boxTarget, Role: HintRoleFocus})

	conclusion := HintConclusion{Placement: placement(move)}
	bivalueMessage := hintDescriptor("hint.bug-plus-one.bivalue", cellArgument("cell", trivalue.Position), countArgument("count", len(bivalue)))
	trivalueMessage := hintDescriptor("hint.bug-plus-one.trivalue", cellArgument("cell", trivalue.Position), digitsArgument("digits", trivalue.Values))
	rowMessage := hintDescriptor("hint.bug-plus-one.unit-count", digitArgument("digit", extraDigit), digitsArgument("other_digits", otherDigits), unitsArgument("unit", move.Evidence.Units[:1]))
	columnMessage := hintDescriptor("hint.bug-plus-one.unit-count", digitArgument("digit", extraDigit), digitsArgument("other_digits", otherDigits), unitsArgument("unit", move.Evidence.Units[1:2]))
	explain := hintDescriptor("hint.bug-plus-one.explain", digitArgument("digit", extraDigit), digitsArgument("other_digits", otherDigits), cellArgument("cell", trivalue.Position), unitsArgument("units", move.Evidence.Units))
	apply := hintDescriptor("hint.bug-plus-one.place", digitArgument("digit", move.Cell.Value), cellArgument("cell", move.Cell.Position), strategyArgument("strategy", move.Technique))
	return []HintStep{
		{ID: "find-bivalue-cells", Kind: HintStepObserve, Message: bivalueMessage, Narration: []HintNarrationChunk{narrationMessage(bivalueMessage)}, Marks: bivalueMarks},
		{ID: "find-trivalue-cell", Kind: HintStepObserve, Message: trivalueMessage, Narration: []HintNarrationChunk{narrationMessage(trivalueMessage)}, Marks: trivalueMarks},
		{ID: "count-extra-candidate-in-row", Kind: HintStepExplain, Message: rowMessage, Narration: []HintNarrationChunk{narrationMessage(rowMessage)}, Marks: rowMarks},
		{ID: "count-extra-candidate-in-column", Kind: HintStepExplain, Message: columnMessage, Narration: []HintNarrationChunk{narrationMessage(columnMessage)}, Marks: columnMarks},
		{ID: "explain-extra-candidate", Kind: HintStepExplain, Message: explain, Narration: []HintNarrationChunk{narrationMessage(explain)}, Marks: allUnitMarks},
		{ID: "apply-bug-plus-one", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: append(allUnitMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func compactUniquenessSteps(move *solver.Move, conclusion HintConclusion) []HintStep {
	evidence := move.Evidence
	if evidence == nil || len(evidence.Premises) < 4 {
		return evidenceSteps(move, conclusion)
	}
	premises := evidence.Premises
	pair := sharedValues(premises[:4]...)
	if len(pair) != 2 {
		return evidenceSteps(move, conclusion)
	}
	corners := make([]core.Position, 4)
	expanded := make([]solver.CandidateGroup, 0, 2)
	for index, premise := range premises[:4] {
		corners[index] = premise.Position
		if len(premise.Values) > 2 {
			expanded = append(expanded, premise)
		}
	}
	marks := unitAndPremiseMarks(evidence)
	observe := hintDescriptor("hint.unique-rectangle.observe", cellsArgument("cells", corners), digitsArgument("digits", pair))
	var explain message.Descriptor
	var nodes []HintNode
	var links []HintLink
	switch move.Technique {
	case "unique-rectangle":
		if len(expanded) != 1 {
			return evidenceSteps(move, conclusion)
		}
		explain = hintDescriptor("hint.unique-rectangle.type1", cellArgument("cell", expanded[0].Position), digitsArgument("digits", pair))
	case "unique-rectangle-2":
		if len(expanded) != 2 {
			return evidenceSteps(move, conclusion)
		}
		extra := removeValues(sharedValues(expanded...), pair)
		if len(extra) != 1 {
			return evidenceSteps(move, conclusion)
		}
		explain = hintDescriptor("hint.unique-rectangle.type2", cellsArgument("cells", []core.Position{expanded[0].Position, expanded[1].Position}), digitArgument("digit", extra[0]))
	case "unique-rectangle-3":
		if evidence.Unit == nil || len(expanded) != 2 || len(premises) < 5 {
			return evidenceSteps(move, conclusion)
		}
		extras := removeValues(combinedValues(expanded...), pair)
		if len(extras) == 0 {
			return evidenceSteps(move, conclusion)
		}
		subsetCells := []core.Position{expanded[0].Position, expanded[1].Position}
		for _, premise := range premises[4:] {
			subsetCells = append(subsetCells, premise.Position)
		}
		explain = hintDescriptor("hint.unique-rectangle.type3", digitsArgument("digits", extras), cellsArgument("cells", subsetCells), unitsArgument("unit", []solver.UnitRef{*evidence.Unit}))
	case "unique-rectangle-4":
		if evidence.Unit == nil || len(expanded) != 2 || len(conclusion.Eliminations) == 0 {
			return evidenceSteps(move, conclusion)
		}
		removed := conclusion.Eliminations[0].Value
		conjugate := 0
		for _, digit := range pair {
			if digit != removed {
				conjugate = digit
			}
		}
		if conjugate == 0 {
			return evidenceSteps(move, conclusion)
		}
		cells := []core.Position{expanded[0].Position, expanded[1].Position}
		explain = hintDescriptor("hint.unique-rectangle.type4", digitArgument("digit", conjugate), cellsArgument("cells", cells), unitsArgument("unit", []solver.UnitRef{*evidence.Unit}), digitArgument("removed_digit", removed))
		refs := []solver.CandidateRef{{Position: cells[0], Value: conjugate}, {Position: cells[1], Value: conjugate}}
		nodes = []HintNode{{ID: candidateNodeID(refs[0]), Target: candidateTarget(refs[0])}, {ID: candidateNodeID(refs[1]), Target: candidateTarget(refs[1])}}
		links = []HintLink{{From: nodes[0].ID, To: nodes[1].ID, Kind: solver.LinkStrong}}
	default:
		return evidenceSteps(move, conclusion)
	}
	var apply message.Descriptor
	if conclusion.Placement != nil {
		apply = hintDescriptor("hint.unique-rectangle.place", digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
	} else if len(conclusion.Eliminations) > 0 {
		apply = hintDescriptor("hint.unique-rectangle.apply", candidatesArgument("candidates", conclusion.Eliminations), strategyArgument("strategy", move.Technique))
	} else {
		return evidenceSteps(move, conclusion)
	}
	return []HintStep{
		{ID: "find-unique-rectangle", Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationMessage(observe)}, Marks: marks},
		{ID: "explain-" + move.Technique, Kind: HintStepExplain, Message: explain, Narration: []HintNarrationChunk{narrationMessage(explain)}, Marks: marks, Nodes: nodes, Links: links},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(marks, conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)},
	}
}

func uniquenessSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	conclusion := HintConclusion{Placement: placement(move)}
	if evidence != nil {
		conclusion.Eliminations = evidence.Eliminations
	}
	if evidence == nil || len(evidence.Premises) < 4 {
		return compactUniquenessSteps(move, conclusion)
	}
	premises := evidence.Premises
	pair := sharedValues(premises[:4]...)
	if len(pair) != 2 || len(evidence.Eliminations) == 0 {
		return compactUniquenessSteps(move, conclusion)
	}
	corners := make([]core.Position, 4)
	var expanded []solver.CandidateGroup
	for index, premise := range premises[:4] {
		corners[index] = premise.Position
		if len(premise.Values) > 2 {
			expanded = append(expanded, premise)
		}
	}
	switch move.Technique {
	case "unique-rectangle":
		if len(expanded) != 1 {
			return compactUniquenessSteps(move, conclusion)
		}
		exact := make([]solver.CandidateGroup, 0, 3)
		for _, premise := range premises[:4] {
			if len(premise.Values) == 2 {
				exact = append(exact, premise)
			}
		}
		if len(exact) != 3 || len(expanded[0].Values) != 3 {
			return compactUniquenessSteps(move, conclusion)
		}
		extra := removeValues(expanded[0].Values, pair)
		if len(extra) != 1 || conclusion.Placement == nil || conclusion.Placement.Value != extra[0] {
			return compactUniquenessSteps(move, conclusion)
		}
		unitMarks := make([]HintMark, 0, 4)
		seenRows, seenColumns := map[int]bool{}, map[int]bool{}
		for _, corner := range corners {
			if !seenRows[corner.Row] {
				seenRows[corner.Row] = true
				target, _ := unitTarget(&solver.UnitRef{Kind: solver.UnitRow, Index: corner.Row})
				unitMarks = append(unitMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
			if !seenColumns[corner.Column] {
				seenColumns[corner.Column] = true
				target, _ := unitTarget(&solver.UnitRef{Kind: solver.UnitColumn, Index: corner.Column})
				unitMarks = append(unitMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
		}
		rectangleMarks := append([]HintMark(nil), unitMarks...)
		for _, corner := range corners {
			rectangleMarks = append(rectangleMarks, HintMark{Target: cellTarget(corner), Role: HintRoleFocus})
		}
		withCandidates := func(groups ...solver.CandidateGroup) []HintMark {
			result := append([]HintMark(nil), rectangleMarks...)
			for _, group := range groups {
				for _, value := range group.Values {
					result = append(result, HintMark{Target: candidateTarget(solver.CandidateRef{Position: group.Position, Value: value}), Role: HintRolePremise})
				}
			}
			return result
		}
		firstPairMarks := withCandidates(exact[:2]...)
		threeCornerMarks := withCandidates(exact...)
		allGroups := append(append([]solver.CandidateGroup(nil), exact...), expanded[0])
		allCandidateMarks := withCandidates(allGroups...)
		uniquenessMarks := append([]HintMark(nil), allCandidateMarks...)
		for index := range uniquenessMarks {
			target := uniquenessMarks[index].Target
			if target.Kind == HintTargetCandidate && target.Position != nil && *target.Position == expanded[0].Position && containsValue(pair, target.Value) {
				uniquenessMarks[index].Role = HintRoleEliminated
			}
		}
		findRectangle := hintDescriptor("hint.unique-rectangle.rectangle", cellsArgument("cells", corners))
		findPair := hintDescriptor("hint.unique-rectangle.pair-corners", cellsArgument("cells", []core.Position{exact[0].Position, exact[1].Position}), digitsArgument("digits", pair))
		findThird := hintDescriptor("hint.unique-rectangle.third-corner", cellArgument("cell", exact[2].Position), digitsArgument("digits", pair))
		findExpanded := hintDescriptor("hint.unique-rectangle.expanded-corner", cellArgument("cell", expanded[0].Position), digitsArgument("digits", expanded[0].Values), digitArgument("extra_digit", extra[0]))
		protectUniqueness := hintDescriptor("hint.unique-rectangle.protect-uniqueness", cellArgument("cell", expanded[0].Position), digitsArgument("digits", pair), digitArgument("extra_digit", extra[0]))
		apply := hintDescriptor("hint.unique-rectangle.type1-place", digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
		return []HintStep{
			{ID: "find-unique-rectangle-corners", Kind: HintStepObserve, Message: findRectangle, Narration: []HintNarrationChunk{narrationMessage(findRectangle)}, Marks: rectangleMarks},
			{ID: "find-unique-rectangle-pair", Kind: HintStepObserve, Message: findPair, Narration: []HintNarrationChunk{narrationMessage(findPair)}, Marks: firstPairMarks},
			{ID: "find-unique-rectangle-third-corner", Kind: HintStepObserve, Message: findThird, Narration: []HintNarrationChunk{narrationMessage(findThird)}, Marks: threeCornerMarks},
			{ID: "find-unique-rectangle-expanded-corner", Kind: HintStepObserve, Message: findExpanded, Narration: []HintNarrationChunk{narrationMessage(findExpanded)}, Marks: allCandidateMarks},
			{ID: "protect-unique-rectangle-uniqueness", Kind: HintStepExplain, Message: protectUniqueness, Narration: []HintNarrationChunk{narrationMessage(protectUniqueness)}, Marks: uniquenessMarks},
			{ID: "apply-unique-rectangle", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: append(uniquenessMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
		}
	case "unique-rectangle-2":
		exact := make([]solver.CandidateGroup, 0, 2)
		for _, premise := range premises[:4] {
			if len(premise.Values) == 2 {
				exact = append(exact, premise)
			}
		}
		if len(exact) != 2 || len(expanded) != 2 || len(expanded[0].Values) != 3 || len(expanded[1].Values) != 3 {
			return compactUniquenessSteps(move, conclusion)
		}
		extra := removeValues(sharedValues(expanded...), pair)
		if len(extra) != 1 || conclusion.Placement == nil || conclusion.Placement.Value == extra[0] {
			return compactUniquenessSteps(move, conclusion)
		}
		unitMarks := make([]HintMark, 0, 4)
		seenRows, seenColumns := map[int]bool{}, map[int]bool{}
		for _, corner := range corners {
			if !seenRows[corner.Row] {
				seenRows[corner.Row] = true
				target, _ := unitTarget(&solver.UnitRef{Kind: solver.UnitRow, Index: corner.Row})
				unitMarks = append(unitMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
			if !seenColumns[corner.Column] {
				seenColumns[corner.Column] = true
				target, _ := unitTarget(&solver.UnitRef{Kind: solver.UnitColumn, Index: corner.Column})
				unitMarks = append(unitMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
		}
		rectangleMarks := append([]HintMark(nil), unitMarks...)
		for _, corner := range corners {
			rectangleMarks = append(rectangleMarks, HintMark{Target: cellTarget(corner), Role: HintRoleFocus})
		}
		withCandidates := func(groups ...solver.CandidateGroup) []HintMark {
			result := append([]HintMark(nil), rectangleMarks...)
			for _, group := range groups {
				for _, value := range group.Values {
					result = append(result, HintMark{Target: candidateTarget(solver.CandidateRef{Position: group.Position, Value: value}), Role: HintRolePremise})
				}
			}
			return result
		}
		pairMarks := withCandidates(exact...)
		firstExpandedMarks := withCandidates(exact[0], exact[1], expanded[0])
		allCornerMarks := withCandidates(exact[0], exact[1], expanded[0], expanded[1])
		uniquenessMarks := append([]HintMark(nil), allCornerMarks...)
		uniquenessMarks = append(uniquenessMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: conclusion.Placement.Position, Value: extra[0]}), Role: HintRoleEliminated})
		findRectangle := hintDescriptor("hint.unique-rectangle.rectangle", cellsArgument("cells", corners))
		findPair := hintDescriptor("hint.unique-rectangle.pair-corners", cellsArgument("cells", []core.Position{exact[0].Position, exact[1].Position}), digitsArgument("digits", pair))
		findFirstExpanded := hintDescriptor("hint.unique-rectangle.type2-first-expanded", cellArgument("cell", expanded[0].Position), digitsArgument("pair", pair), digitArgument("extra_digit", extra[0]))
		findSharedExtra := hintDescriptor("hint.unique-rectangle.type2-shared-extra", cellArgument("cell", expanded[1].Position), digitsArgument("pair", pair), digitArgument("extra_digit", extra[0]))
		protectUniqueness := hintDescriptor("hint.unique-rectangle.type2-protect-uniqueness", cellsArgument("cells", []core.Position{expanded[0].Position, expanded[1].Position}), digitArgument("extra_digit", extra[0]), cellArgument("target", conclusion.Placement.Position))
		apply := hintDescriptor("hint.unique-rectangle.type2-place", digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
		return []HintStep{
			{ID: "find-unique-rectangle-corners", Kind: HintStepObserve, Message: findRectangle, Narration: []HintNarrationChunk{narrationMessage(findRectangle)}, Marks: rectangleMarks},
			{ID: "find-unique-rectangle-pair", Kind: HintStepObserve, Message: findPair, Narration: []HintNarrationChunk{narrationMessage(findPair)}, Marks: pairMarks},
			{ID: "find-unique-rectangle-first-expanded-corner", Kind: HintStepObserve, Message: findFirstExpanded, Narration: []HintNarrationChunk{narrationMessage(findFirstExpanded)}, Marks: firstExpandedMarks},
			{ID: "find-unique-rectangle-shared-extra", Kind: HintStepObserve, Message: findSharedExtra, Narration: []HintNarrationChunk{narrationMessage(findSharedExtra)}, Marks: allCornerMarks},
			{ID: "protect-unique-rectangle-uniqueness", Kind: HintStepExplain, Message: protectUniqueness, Narration: []HintNarrationChunk{narrationMessage(protectUniqueness)}, Marks: uniquenessMarks},
			{ID: "apply-unique-rectangle", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: append(uniquenessMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
		}
	case "unique-rectangle-3":
		exact := make([]solver.CandidateGroup, 0, 2)
		for _, premise := range premises[:4] {
			if len(premise.Values) == 2 {
				exact = append(exact, premise)
			}
		}
		if evidence.Unit == nil || len(exact) != 2 || len(expanded) != 2 || len(premises) != 5 || conclusion.Placement == nil || len(evidence.Eliminations) != 1 {
			return compactUniquenessSteps(move, conclusion)
		}
		extras := removeValues(combinedValues(expanded...), pair)
		partner := premises[4]
		removed := evidence.Eliminations[0]
		if len(extras) != 2 || !equalValues(partner.Values, extras) || !containsValue(extras, removed.Value) || removed.Position != conclusion.Placement.Position {
			return compactUniquenessSteps(move, conclusion)
		}
		unitMarks := make([]HintMark, 0, 4)
		seenRows, seenColumns := map[int]bool{}, map[int]bool{}
		for _, corner := range corners {
			if !seenRows[corner.Row] {
				seenRows[corner.Row] = true
				target, _ := unitTarget(&solver.UnitRef{Kind: solver.UnitRow, Index: corner.Row})
				unitMarks = append(unitMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
			if !seenColumns[corner.Column] {
				seenColumns[corner.Column] = true
				target, _ := unitTarget(&solver.UnitRef{Kind: solver.UnitColumn, Index: corner.Column})
				unitMarks = append(unitMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
		}
		rectangleMarks := append([]HintMark(nil), unitMarks...)
		for _, corner := range corners {
			rectangleMarks = append(rectangleMarks, HintMark{Target: cellTarget(corner), Role: HintRoleFocus})
		}
		withCandidates := func(groups ...solver.CandidateGroup) []HintMark {
			result := append([]HintMark(nil), rectangleMarks...)
			for _, group := range groups {
				for _, value := range group.Values {
					result = append(result, HintMark{Target: candidateTarget(solver.CandidateRef{Position: group.Position, Value: value}), Role: HintRolePremise})
				}
			}
			return result
		}
		pairMarks := withCandidates(exact...)
		firstExpandedMarks := withCandidates(exact[0], exact[1], expanded[0])
		allCornerMarks := withCandidates(exact[0], exact[1], expanded[0], expanded[1])
		subsetMarks := withCandidates(exact[0], exact[1], expanded[0], expanded[1], partner)
		eliminationMarks := append([]HintMark(nil), subsetMarks...)
		eliminationMarks = append(eliminationMarks, HintMark{Target: candidateTarget(removed), Role: HintRoleEliminated})
		firstExtra := removeValues(expanded[0].Values, pair)
		secondExtra := removeValues(expanded[1].Values, pair)
		if len(firstExtra) != 1 || len(secondExtra) != 1 || firstExtra[0] == secondExtra[0] {
			return compactUniquenessSteps(move, conclusion)
		}
		findRectangle := hintDescriptor("hint.unique-rectangle.rectangle", cellsArgument("cells", corners))
		findPair := hintDescriptor("hint.unique-rectangle.pair-corners", cellsArgument("cells", []core.Position{exact[0].Position, exact[1].Position}), digitsArgument("digits", pair))
		findFirstExpanded := hintDescriptor("hint.unique-rectangle.type3-first-expanded", cellArgument("cell", expanded[0].Position), digitsArgument("pair", pair), digitArgument("extra_digit", firstExtra[0]))
		findSecondExpanded := hintDescriptor("hint.unique-rectangle.type3-second-expanded", cellArgument("cell", expanded[1].Position), digitsArgument("pair", pair), digitArgument("extra_digit", secondExtra[0]))
		findPartner := hintDescriptor("hint.unique-rectangle.type3-partner", cellArgument("cell", partner.Position), digitsArgument("digits", extras))
		reserveExtras := hintDescriptor("hint.unique-rectangle.type3-reserve", cellsArgument("corners", []core.Position{expanded[0].Position, expanded[1].Position}), digitsArgument("digits", extras), cellArgument("partner", partner.Position), unitsArgument("unit", []solver.UnitRef{*evidence.Unit}), cellArgument("target", removed.Position), digitArgument("removed_digit", removed.Value))
		apply := hintDescriptor("hint.unique-rectangle.type3-place", digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
		return []HintStep{
			{ID: "find-unique-rectangle-corners", Kind: HintStepObserve, Message: findRectangle, Narration: []HintNarrationChunk{narrationMessage(findRectangle)}, Marks: rectangleMarks},
			{ID: "find-unique-rectangle-pair", Kind: HintStepObserve, Message: findPair, Narration: []HintNarrationChunk{narrationMessage(findPair)}, Marks: pairMarks},
			{ID: "find-unique-rectangle-first-expanded-corner", Kind: HintStepObserve, Message: findFirstExpanded, Narration: []HintNarrationChunk{narrationMessage(findFirstExpanded)}, Marks: firstExpandedMarks},
			{ID: "find-unique-rectangle-second-expanded-corner", Kind: HintStepObserve, Message: findSecondExpanded, Narration: []HintNarrationChunk{narrationMessage(findSecondExpanded)}, Marks: allCornerMarks},
			{ID: "find-unique-rectangle-subset-partner", Kind: HintStepObserve, Message: findPartner, Narration: []HintNarrationChunk{narrationMessage(findPartner)}, Marks: subsetMarks},
			{ID: "reserve-unique-rectangle-extra-digits", Kind: HintStepExplain, Message: reserveExtras, Narration: []HintNarrationChunk{narrationMessage(reserveExtras)}, Marks: eliminationMarks},
			{ID: "apply-unique-rectangle", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: append(eliminationMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
		}
	case "unique-rectangle-4":
		exact := make([]solver.CandidateGroup, 0, 2)
		for _, premise := range premises[:4] {
			if len(premise.Values) == 2 {
				exact = append(exact, premise)
			}
		}
		if evidence.Unit == nil || len(exact) != 2 || len(expanded) != 2 || len(expanded[0].Values) != 3 || len(expanded[1].Values) != 3 || len(evidence.Eliminations) != 2 || conclusion.Placement != nil {
			return compactUniquenessSteps(move, conclusion)
		}
		removed := evidence.Eliminations[0].Value
		for _, elimination := range evidence.Eliminations {
			if elimination.Value != removed || (elimination.Position != expanded[0].Position && elimination.Position != expanded[1].Position) {
				return compactUniquenessSteps(move, conclusion)
			}
		}
		conjugate := 0
		for _, digit := range pair {
			if digit != removed {
				conjugate = digit
			}
		}
		firstExtra := removeValues(expanded[0].Values, pair)
		secondExtra := removeValues(expanded[1].Values, pair)
		if conjugate == 0 || len(firstExtra) != 1 || len(secondExtra) != 1 {
			return compactUniquenessSteps(move, conclusion)
		}
		unitMarks := make([]HintMark, 0, 4)
		seenRows, seenColumns := map[int]bool{}, map[int]bool{}
		for _, corner := range corners {
			if !seenRows[corner.Row] {
				seenRows[corner.Row] = true
				target, _ := unitTarget(&solver.UnitRef{Kind: solver.UnitRow, Index: corner.Row})
				unitMarks = append(unitMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
			if !seenColumns[corner.Column] {
				seenColumns[corner.Column] = true
				target, _ := unitTarget(&solver.UnitRef{Kind: solver.UnitColumn, Index: corner.Column})
				unitMarks = append(unitMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
		}
		rectangleMarks := append([]HintMark(nil), unitMarks...)
		for _, corner := range corners {
			rectangleMarks = append(rectangleMarks, HintMark{Target: cellTarget(corner), Role: HintRoleFocus})
		}
		withCandidates := func(groups ...solver.CandidateGroup) []HintMark {
			result := append([]HintMark(nil), rectangleMarks...)
			for _, group := range groups {
				for _, value := range group.Values {
					result = append(result, HintMark{Target: candidateTarget(solver.CandidateRef{Position: group.Position, Value: value}), Role: HintRolePremise})
				}
			}
			return result
		}
		pairMarks := withCandidates(exact...)
		firstExpandedMarks := withCandidates(exact[0], exact[1], expanded[0])
		allCornerMarks := withCandidates(exact[0], exact[1], expanded[0], expanded[1])
		eliminationMarks := append([]HintMark(nil), allCornerMarks...)
		for _, elimination := range evidence.Eliminations {
			eliminationMarks = append(eliminationMarks, HintMark{Target: candidateTarget(elimination), Role: HintRoleEliminated})
		}
		conjugateRefs := []solver.CandidateRef{{Position: expanded[0].Position, Value: conjugate}, {Position: expanded[1].Position, Value: conjugate}}
		conjugateNodes := []HintNode{
			{ID: candidateNodeID(conjugateRefs[0]), Target: candidateTarget(conjugateRefs[0])},
			{ID: candidateNodeID(conjugateRefs[1]), Target: candidateTarget(conjugateRefs[1])},
		}
		conjugateLinks := []HintLink{{From: conjugateNodes[0].ID, To: conjugateNodes[1].ID, Kind: solver.LinkStrong}}
		findRectangle := hintDescriptor("hint.unique-rectangle.rectangle", cellsArgument("cells", corners))
		findPair := hintDescriptor("hint.unique-rectangle.pair-corners", cellsArgument("cells", []core.Position{exact[0].Position, exact[1].Position}), digitsArgument("digits", pair))
		findFirstExpanded := hintDescriptor("hint.unique-rectangle.type4-first-expanded", cellArgument("cell", expanded[0].Position), digitsArgument("pair", pair), digitArgument("extra_digit", firstExtra[0]))
		findSecondExpanded := hintDescriptor("hint.unique-rectangle.type4-second-expanded", cellArgument("cell", expanded[1].Position), digitsArgument("pair", pair), digitArgument("extra_digit", secondExtra[0]))
		explain := hintDescriptor("hint.unique-rectangle.type4", digitArgument("digit", conjugate), cellsArgument("cells", []core.Position{expanded[0].Position, expanded[1].Position}), unitsArgument("unit", []solver.UnitRef{*evidence.Unit}), digitArgument("removed_digit", removed))
		apply := hintDescriptor("hint.unique-rectangle.type4-apply", digitArgument("digit", removed), cellsArgument("cells", []core.Position{expanded[0].Position, expanded[1].Position}), strategyArgument("strategy", move.Technique))
		return []HintStep{
			{ID: "find-unique-rectangle-corners", Kind: HintStepObserve, Message: findRectangle, Narration: []HintNarrationChunk{narrationMessage(findRectangle)}, Marks: rectangleMarks},
			{ID: "find-unique-rectangle-pair", Kind: HintStepObserve, Message: findPair, Narration: []HintNarrationChunk{narrationMessage(findPair)}, Marks: pairMarks},
			{ID: "find-unique-rectangle-first-expanded-corner", Kind: HintStepObserve, Message: findFirstExpanded, Narration: []HintNarrationChunk{narrationMessage(findFirstExpanded)}, Marks: firstExpandedMarks},
			{ID: "find-unique-rectangle-second-expanded-corner", Kind: HintStepObserve, Message: findSecondExpanded, Narration: []HintNarrationChunk{narrationMessage(findSecondExpanded)}, Marks: allCornerMarks},
			{ID: "protect-unique-rectangle-with-conjugate-pair", Kind: HintStepExplain, Message: explain, Narration: []HintNarrationChunk{narrationMessage(explain)}, Marks: eliminationMarks, Nodes: conjugateNodes, Links: conjugateLinks},
			{ID: "apply-unique-rectangle", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: append(allCornerMarks, conclusionMarks(conclusion)...), Nodes: conjugateNodes, Links: conjugateLinks, Effect: conclusionEffect(conclusion)},
		}
	default:
		return compactUniquenessSteps(move, conclusion)
	}
}

func equalValues(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func removeValues(values, removed []int) []int {
	blocked := make(map[int]struct{}, len(removed))
	for _, value := range removed {
		blocked[value] = struct{}{}
	}
	result := make([]int, 0, len(values))
	for _, value := range values {
		if _, ok := blocked[value]; !ok {
			result = append(result, value)
		}
	}
	return result
}
