package game

import (
	"fmt"
	"strings"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/message"
	"github.com/gnailuy/sudoku/solver"
)

func familyHintReason(move *solver.Move) *HintReason {
	if move == nil || move.Evidence == nil {
		return nil
	}
	evidence := move.Evidence
	switch move.Technique {
	case "naked-pair", "naked-triple", "naked-quad":
		if evidence.Unit == nil || len(evidence.Premises) < 2 || (len(evidence.Eliminations) == 0 && !move.IsPlacement()) {
			return nil
		}
		cells := make([]core.Position, len(evidence.Premises))
		for index, premise := range evidence.Premises {
			cells[index] = premise.Position
		}
		digits := combinedValues(evidence.Premises...)
		targets := append([]solver.CandidateRef(nil), evidence.Eliminations...)
		if len(targets) == 0 && move.IsPlacement() {
			targets = append(targets, solver.CandidateRef{Position: move.Cell.Position, Value: move.Cell.Value})
		}
		return &HintReason{Kind: HintReasonSubsetReservation, SubsetReservation: &SubsetReservationReason{
			Unit: *evidence.Unit, ReservedDigits: digits, PremiseCells: cells, ActionTargets: targets,
		}}
	case "hidden-pair", "hidden-triple", "hidden-quad":
		if evidence.Unit == nil || len(evidence.Premises) < 2 || (len(evidence.Eliminations) == 0 && !move.IsPlacement()) {
			return nil
		}
		cells := make([]core.Position, len(evidence.Premises))
		for index, premise := range evidence.Premises {
			cells[index] = premise.Position
		}
		digits := combinedValues(evidence.Premises...)
		targets := append([]solver.CandidateRef(nil), evidence.Eliminations...)
		if len(targets) == 0 && move.IsPlacement() {
			targets = append(targets, solver.CandidateRef{Position: move.Cell.Position, Value: move.Cell.Value})
		}
		return &HintReason{Kind: HintReasonHiddenSubsetConfinement, HiddenSubsetConfinement: &HiddenSubsetConfinementReason{
			Unit: *evidence.Unit, ConfinedDigits: digits, PremiseCells: cells, ActionTargets: targets,
		}}
	case "pointing-pair":
		if len(evidence.Units) != 2 || len(evidence.Premises) < 2 || (len(evidence.Eliminations) == 0 && !move.IsPlacement()) {
			return nil
		}
		digit := evidence.Premises[0].Values[0]
		premises := make([]solver.CandidateRef, 0, len(evidence.Premises))
		for _, premise := range evidence.Premises {
			if len(premise.Values) != 1 || premise.Values[0] != digit {
				return nil
			}
			premises = append(premises, solver.CandidateRef{Position: premise.Position, Value: digit})
		}
		targets := append([]solver.CandidateRef(nil), evidence.Eliminations...)
		if len(targets) == 0 && move.IsPlacement() {
			targets = append(targets, solver.CandidateRef{Position: move.Cell.Position, Value: move.Cell.Value})
		}
		return &HintReason{Kind: HintReasonIntersectionConfinement, IntersectionConfinement: &IntersectionConfinementReason{
			Digit: digit, SourceUnit: evidence.Units[0], IntersectingUnit: evidence.Units[1], PremiseCandidates: premises, ActionTargets: targets,
		}}
	case "x-wing", "swordfish", "jellyfish":
		size := map[string]int{"x-wing": 2, "swordfish": 3, "jellyfish": 4}[move.Technique]
		if len(evidence.Units) != size*2 || len(evidence.Premises) == 0 || len(evidence.Eliminations) == 0 {
			return nil
		}
		if len(evidence.Premises[0].Values) != 1 {
			return nil
		}
		digit := evidence.Premises[0].Values[0]
		patterns := make([]solver.CandidateRef, 0, len(evidence.Premises))
		for _, premise := range evidence.Premises {
			if len(premise.Values) != 1 || premise.Values[0] != digit {
				return nil
			}
			patterns = append(patterns, solver.CandidateRef{Position: premise.Position, Value: digit})
		}
		return &HintReason{Kind: HintReasonFishConfinement, FishConfinement: &FishConfinementReason{
			Digit: digit, BaseUnits: append([]solver.UnitRef(nil), evidence.Units[:size]...), CoverUnits: append([]solver.UnitRef(nil), evidence.Units[size:]...),
			PatternCandidates: patterns, ActionTargets: append([]solver.CandidateRef(nil), evidence.Eliminations...),
		}}
	case "xy-chain":
		if len(evidence.Chain) < 2 || len(evidence.Links) == 0 || len(evidence.Eliminations) == 0 {
			return nil
		}
		endpoint := sharedValues(evidence.Chain[0], evidence.Chain[len(evidence.Chain)-1])
		if len(endpoint) != 1 {
			return nil
		}
		return &HintReason{Kind: HintReasonImplicationChain, ImplicationChain: &ImplicationChainReason{
			BivalueGroups: append([]solver.CandidateGroup(nil), evidence.Chain...), Links: append([]solver.CandidateLink(nil), evidence.Links...), EndpointDigit: endpoint[0],
			ActionTargets: append([]solver.CandidateRef(nil), evidence.Eliminations...),
		}}
	default:
		return nil
	}
}

func actionNarration(descriptor message.Descriptor) []HintNarrationChunk {
	return []HintNarrationChunk{narrationMessage(descriptor)}
}

func evidenceSteps(move *solver.Move, conclusion HintConclusion) []HintStep {
	evidence := move.Evidence
	var premiseMarks []HintMark
	for _, unitRef := range evidence.Units {
		unit := unitRef
		if target, ok := unitTarget(&unit); ok {
			premiseMarks = append(premiseMarks, HintMark{Target: target, Role: HintRoleFocus})
		}
	}
	for _, premise := range evidence.Premises {
		premiseMarks = append(premiseMarks, HintMark{Target: cellTarget(premise.Position), Role: HintRoleFocus})
		for _, value := range premise.Values {
			premiseMarks = append(premiseMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: value}), Role: HintRolePremise})
		}
	}
	var comparisonMarks []HintMark
	refs := append([]solver.CandidateRef(nil), evidence.RuledOut...)
	refs = append(refs, evidence.Eliminations...)
	for _, ref := range refs {
		comparisonMarks = append(comparisonMarks, HintMark{Target: candidateTarget(ref), Role: HintRoleEliminated})
	}
	steps := []HintStep{
		{ID: "observe-candidate-pattern", Kind: HintStepObserve, Marks: premiseMarks},
		{ID: "compare-candidate-constraints", Kind: HintStepExplain, Marks: append(append([]HintMark(nil), premiseMarks...), comparisonMarks...)},
	}
	kind := HintStepApply
	if conclusion.Placement == nil {
		kind = HintStepApply
	}
	steps = append(steps, HintStep{ID: "apply-deduction", Kind: kind, Marks: conclusionMarks(conclusion), Effect: conclusionEffect(conclusion)})
	return steps
}

func nakedSingleSteps(move *solver.Move) []HintStep {
	ref := solver.CandidateRef{Position: move.Cell.Position, Value: move.Cell.Value}
	units := []HintMark{
		{Target: HintTarget{Kind: HintTargetRow, Index: move.Cell.Position.Row}, Role: HintRoleFocus},
		{Target: HintTarget{Kind: HintTargetColumn, Index: move.Cell.Position.Column}, Role: HintRoleFocus},
		{Target: HintTarget{Kind: HintTargetBox, Index: (move.Cell.Position.Row/3)*3 + move.Cell.Position.Column/3}, Role: HintRoleFocus},
	}
	observeMarks := append(units, HintMark{Target: cellTarget(move.Cell.Position), Role: HintRoleFocus}, HintMark{Target: candidateTarget(ref), Role: HintRolePremise})
	return []HintStep{
		{ID: "inspect-" + compactCellName(move.Cell.Position), Kind: HintStepObserve, Narration: []HintNarrationChunk{narrationID("hint.narration.look-at"), narrationTarget(cellTarget(move.Cell.Position)), narrationID("hint.narration.only-candidate"), narrationDigit(move.Cell.Value), narrationID("hint.narration.period")}, Marks: observeMarks},
		{ID: fmt.Sprintf("place-%s-%d", compactCellName(move.Cell.Position), move.Cell.Value), Kind: HintStepApply, Narration: []HintNarrationChunk{narrationID("hint.narration.no-other-digit"), narrationTarget(cellTarget(move.Cell.Position)), narrationID("hint.narration.place"), narrationDigit(move.Cell.Value), narrationID("hint.narration.naked-single-tail")}, Marks: []HintMark{{Target: cellTarget(move.Cell.Position), Role: HintRoleFocus}, {Target: candidateTarget(ref), Role: HintRoleConclusion}}, Effect: &HintEffect{Placement: &move.Cell}},
	}
}

func hiddenSingleSteps(move *solver.Move) []HintStep {
	ref := solver.CandidateRef{Position: move.Cell.Position, Value: move.Cell.Value}
	unit, _ := unitTarget(move.Evidence.Unit)
	marks := []HintMark{{Target: unit, Role: HintRoleFocus}, {Target: candidateTarget(ref), Role: HintRolePremise}}
	compare := append([]HintMark(nil), marks...)
	for _, ruledOut := range move.Evidence.RuledOut {
		compare = append(compare, HintMark{Target: candidateTarget(ruledOut), Role: HintRoleEliminated})
	}
	unitName := fmt.Sprintf("%s %d", move.Evidence.Unit.Kind, move.Evidence.Unit.Index+1)
	return []HintStep{
		{ID: fmt.Sprintf("scan-%s-for-%d", strings.ReplaceAll(unitName, " ", ""), move.Cell.Value), Kind: HintStepObserve, Narration: []HintNarrationChunk{narrationID("hint.narration.in-unit"), narrationTarget(unit), narrationID("hint.narration.consider-digit"), narrationDigit(move.Cell.Value), narrationID("hint.narration.can-appear")}, Marks: marks},
		{ID: fmt.Sprintf("compare-%s-%d", strings.ReplaceAll(unitName, " ", ""), move.Cell.Value), Kind: HintStepExplain, Narration: []HintNarrationChunk{narrationID("hint.narration.other-cells"), narrationTarget(unit), narrationID("hint.narration.cannot-contain"), narrationDigit(move.Cell.Value), narrationID("hint.narration.leaving-only"), narrationTarget(cellTarget(move.Cell.Position)), narrationID("hint.narration.period")}, Marks: compare},
		{ID: fmt.Sprintf("place-%s-%d", compactCellName(move.Cell.Position), move.Cell.Value), Kind: HintStepApply, Narration: []HintNarrationChunk{narrationID("hint.narration.because-unit"), narrationTarget(unit), narrationID("hint.narration.still-needs"), narrationDigit(move.Cell.Value), narrationID("hint.narration.only-cell"), narrationTarget(cellTarget(move.Cell.Position)), narrationID("hint.narration.hidden-single-tail"), narrationDigit(move.Cell.Value), narrationID("hint.narration.hidden-single-name")}, Marks: []HintMark{{Target: unit, Role: HintRoleFocus}, {Target: cellTarget(move.Cell.Position), Role: HintRoleFocus}, {Target: candidateTarget(ref), Role: HintRoleConclusion}}, Effect: &HintEffect{Placement: &move.Cell}},
	}
}

func nakedSubsetSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	unit, _ := unitTarget(evidence.Unit)
	premiseMarks := []HintMark{{Target: unit, Role: HintRoleFocus}}
	var cellIDs []string
	var cells []core.Position
	for _, premise := range evidence.Premises {
		cellIDs = append(cellIDs, compactCellName(premise.Position))
		cells = append(cells, premise.Position)
		for _, value := range premise.Values {
			premiseMarks = append(premiseMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: value}), Role: HintRolePremise})
		}
	}
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	if move.Technique == "naked-pair" || move.Technique == "naked-triple" || move.Technique == "naked-quad" {
		digits := combinedValues(evidence.Premises...)
		units := []solver.UnitRef{*evidence.Unit}
		observe := hintDescriptor("hint.subset-reservation.observe", unitsArgument("unit", units), cellsArgument("cells", cells), digitsArgument("digits", digits))
		explain := hintDescriptor("hint.subset-reservation.explain", unitsArgument("unit", units), cellsArgument("cells", cells), digitsArgument("digits", digits))
		var apply message.Descriptor
		if conclusion.Placement != nil {
			apply = hintDescriptor("hint.subset-reservation.place", digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
		} else {
			apply = hintDescriptor("hint.subset-reservation.apply", candidatesArgument("candidates", evidence.Eliminations), strategyArgument("strategy", move.Technique))
		}
		candidateMarks := append([]HintMark(nil), premiseMarks[1:]...)
		return []HintStep{
			{ID: "find-" + strings.Join(cellIDs, "-"), Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationMessage(observe)}, Marks: candidateMarks},
			{ID: "reserve-" + strings.Join(cellIDs, "-"), Kind: HintStepExplain, Message: explain, Narration: []HintNarrationChunk{narrationMessage(explain)}, Marks: premiseMarks},
			{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(candidateMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
		}
	}
	return []HintStep{
		{ID: "find-" + strings.Join(cellIDs, "-"), Kind: HintStepObserve, Marks: premiseMarks},
		{ID: "reserve-" + strings.Join(cellIDs, "-"), Kind: HintStepExplain, Marks: premiseMarks},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Marks: append(premiseMarks[1:], conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func hiddenSubsetSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	unit, _ := unitTarget(evidence.Unit)
	premiseMarks := []HintMark{{Target: unit, Role: HintRoleFocus}}
	var cells []core.Position
	for _, premise := range evidence.Premises {
		cells = append(cells, premise.Position)
		for _, value := range premise.Values {
			premiseMarks = append(premiseMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: value}), Role: HintRolePremise})
		}
	}
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	digits := combinedValues(evidence.Premises...)
	units := []solver.UnitRef{*evidence.Unit}
	observe := hintDescriptor("hint.hidden-subset-confinement.observe", unitsArgument("unit", units), digitsArgument("digits", digits), cellsArgument("cells", cells))
	explain := hintDescriptor("hint.hidden-subset-confinement.explain", digitsArgument("digits", digits), unitsArgument("unit", units), cellsArgument("cells", cells))
	var apply message.Descriptor
	if conclusion.Placement != nil {
		apply = hintDescriptor("hint.hidden-subset-confinement.place", digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
	} else {
		apply = hintDescriptor("hint.hidden-subset-confinement.apply", candidatesArgument("candidates", evidence.Eliminations), strategyArgument("strategy", move.Technique))
	}
	candidateMarks := append([]HintMark(nil), premiseMarks[1:]...)
	return []HintStep{
		{ID: "find-confined-" + move.Technique, Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationMessage(observe)}, Marks: candidateMarks},
		{ID: "compare-confined-" + move.Technique, Kind: HintStepExplain, Message: explain, Narration: []HintNarrationChunk{narrationMessage(explain)}, Marks: premiseMarks},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(candidateMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func intersectionSteps(move *solver.Move) []HintStep {
	evidence := move.Evidence
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	if len(evidence.Units) != 2 || len(evidence.Premises) < 2 {
		return evidenceSteps(move, conclusion)
	}
	digit := evidence.Premises[0].Values[0]
	source, intersection := evidence.Units[0], evidence.Units[1]
	premiseRefs := make([]solver.CandidateRef, 0, len(evidence.Premises))
	premiseCells := make([]core.Position, 0, len(evidence.Premises))
	for _, premise := range evidence.Premises {
		premiseRefs = append(premiseRefs, solver.CandidateRef{Position: premise.Position, Value: digit})
		premiseCells = append(premiseCells, premise.Position)
	}
	observe := hintDescriptor("hint.intersection-confinement.observe", unitsArgument("source_unit", []solver.UnitRef{source}), digitArgument("digit", digit), unitsArgument("intersection_unit", []solver.UnitRef{intersection}), cellsArgument("cells", premiseCells))
	explain := hintDescriptor("hint.intersection-confinement.explain", digitArgument("digit", digit), unitsArgument("source_unit", []solver.UnitRef{source}), unitsArgument("intersection_unit", []solver.UnitRef{intersection}))
	var apply message.Descriptor
	if conclusion.Placement != nil {
		apply = hintDescriptor("hint.intersection-confinement.place", digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
	} else {
		positions := make([]core.Position, len(evidence.Eliminations))
		for i, ref := range evidence.Eliminations {
			positions[i] = ref.Position
		}
		apply = hintDescriptor("hint.intersection-confinement.apply", digitArgument("digit", digit), cellsArgument("cells", positions), strategyArgument("strategy", move.Technique))
	}
	premiseMarks := []HintMark{}
	if target, ok := unitTarget(&source); ok {
		premiseMarks = append(premiseMarks, HintMark{Target: target, Role: HintRoleFocus})
	}
	for _, ref := range premiseRefs {
		premiseMarks = append(premiseMarks, HintMark{Target: candidateTarget(ref), Role: HintRolePremise})
	}
	explainMarks := append([]HintMark(nil), premiseMarks...)
	if target, ok := unitTarget(&intersection); ok {
		explainMarks = append(explainMarks, HintMark{Target: target, Role: HintRoleFocus})
	}
	return []HintStep{
		{ID: "find-intersection", Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationMessage(observe)}, Marks: premiseMarks},
		{ID: "explain-intersection", Kind: HintStepExplain, Message: explain, Narration: []HintNarrationChunk{narrationMessage(explain)}, Marks: explainMarks},
		{ID: "apply-intersection", Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(explainMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func fishSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	marks := unitAndPremiseMarks(evidence)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	size := map[string]int{"x-wing": 2, "swordfish": 3, "jellyfish": 4}[move.Technique]
	if size != 0 && len(evidence.Units) == size*2 && len(evidence.Premises) > 0 && len(evidence.Premises[0].Values) == 1 && len(evidence.Eliminations) > 0 {
		baseUnits, coverUnits := evidence.Units[:size], evidence.Units[size:]
		digit := evidence.Premises[0].Values[0]
		observe := hintDescriptor("hint.fish-confinement.observe", unitsArgument("base_units", baseUnits), digitArgument("digit", digit), unitsArgument("cover_units", coverUnits))
		explain := hintDescriptor("hint.fish-confinement.explain", unitsArgument("base_units", baseUnits), unitsArgument("cover_units", coverUnits), digitArgument("digit", digit))
		positions := make([]core.Position, len(evidence.Eliminations))
		for i, ref := range evidence.Eliminations {
			positions[i] = ref.Position
		}
		apply := hintDescriptor("hint.fish-confinement.apply", digitArgument("digit", digit), cellsArgument("cells", positions), strategyArgument("strategy", move.Technique))
		premiseMarks := premiseCandidateMarks(evidence)
		baseMarks := append([]HintMark(nil), premiseMarks...)
		for _, ref := range baseUnits {
			target, _ := unitTarget(&ref)
			baseMarks = append(baseMarks, HintMark{Target: target, Role: HintRoleFocus})
		}
		explainMarks := append([]HintMark(nil), baseMarks...)
		for _, ref := range coverUnits {
			target, _ := unitTarget(&ref)
			explainMarks = append(explainMarks, HintMark{Target: target, Role: HintRoleFocus})
		}
		return []HintStep{
			{ID: "find-" + move.Technique, Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationMessage(observe)}, Marks: baseMarks},
			{ID: "explain-" + move.Technique, Kind: HintStepExplain, Message: explain, Narration: []HintNarrationChunk{narrationMessage(explain)}, Marks: explainMarks},
			{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(explainMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
		}
	}
	return []HintStep{
		{ID: "find-" + move.Technique, Kind: HintStepObserve, Marks: marks},
		{ID: "compare-" + move.Technique, Kind: HintStepExplain, Marks: marks},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Marks: append(marks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func relationalSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	if evidence == nil || len(evidence.Premises) < 3 || len(evidence.Eliminations) == 0 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	baseMarks := unitAndPremiseMarks(evidence)
	nodes := visualNodes(evidence, false)
	var firstNodes []HintNode
	var firstLinks, links []HintLink
	var observe, explain message.Descriptor
	actionDigit := 0
	switch move.Technique {
	case "xy-wing", "xyz-wing":
		pivot, wings := evidence.Premises[0], evidence.Premises[1:3]
		shared := sharedValues(wings...)
		if len(shared) != 1 || (move.Technique == "xy-wing" && len(evidence.Links) != 2) {
			return evidenceSteps(move, conclusion)
		}
		actionDigit = shared[0]
		wingCells := []core.Position{wings[0].Position, wings[1].Position}
		observe = hintDescriptor("hint.pivot-wing.observe", cellArgument("pivot", pivot.Position), digitsArgument("pivot_digits", pivot.Values), cellsArgument("wing_cells", wingCells))
		explain = hintDescriptor("hint.pivot-wing.explain", digitArgument("shared_digit", actionDigit), cellArgument("pivot", pivot.Position), cellsArgument("wing_cells", wingCells))
		links = visualLinks(evidence)
		if len(links) > 0 {
			firstNodes, firstLinks = visualPrefix(nodes, links)
		} else {
			for _, node := range nodes {
				if node.Target.Position != nil && *node.Target.Position == pivot.Position {
					firstNodes = append(firstNodes, node)
				}
			}
		}
	case "w-wing":
		if len(evidence.Premises) < 4 || len(evidence.Links) != 1 || evidence.Links[0].Kind != solver.LinkStrong {
			return evidenceSteps(move, conclusion)
		}
		wings := evidence.Premises[:2]
		pair := sharedValues(wings...)
		linkDigit := evidence.Links[0].From.Value
		if len(pair) != 2 || evidence.Links[0].To.Value != linkDigit {
			return evidenceSteps(move, conclusion)
		}
		for _, digit := range pair {
			if digit != linkDigit {
				actionDigit = digit
			}
		}
		wingCells := []core.Position{wings[0].Position, wings[1].Position}
		linkCells := []core.Position{evidence.Links[0].From.Position, evidence.Links[0].To.Position}
		observe = hintDescriptor("hint.strong-link-wing.observe", cellsArgument("wing_cells", wingCells), digitsArgument("digits", pair))
		explain = hintDescriptor("hint.strong-link-wing.explain", digitArgument("link_digit", linkDigit), cellsArgument("link_cells", linkCells), cellsArgument("wing_cells", wingCells))
		links = visualLinks(evidence)
		firstNodes, firstLinks = nodes, links
	default:
		return evidenceSteps(move, conclusion)
	}
	if actionDigit == 0 {
		return evidenceSteps(move, conclusion)
	}
	var apply message.Descriptor
	if conclusion.Placement != nil {
		apply = hintDescriptor("hint.relational-wing.place", digitArgument("removed_digit", actionDigit), digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
	} else {
		positions := make([]core.Position, len(evidence.Eliminations))
		for i, ref := range evidence.Eliminations {
			positions[i] = ref.Position
		}
		apply = hintDescriptor("hint.relational-wing.apply", digitArgument("digit", actionDigit), cellsArgument("cells", positions), strategyArgument("strategy", move.Technique))
	}
	return []HintStep{
		{ID: "find-" + move.Technique, Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationMessage(observe)}, Marks: marksForNodes(firstNodes), Nodes: firstNodes, Links: firstLinks},
		{ID: "explain-" + move.Technique, Kind: HintStepExplain, Message: explain, Narration: []HintNarrationChunk{narrationMessage(explain)}, Marks: baseMarks, Nodes: nodes, Links: links},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(baseMarks, conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)},
	}
}

func bugPlusOneSteps(move *solver.Move, _ string) []HintStep {
	if move.Evidence == nil || len(move.Evidence.Premises) == 0 || !move.IsPlacement() {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	premise := move.Evidence.Premises[0]
	if len(premise.Values) != 3 || len(move.Evidence.Units) != 3 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	marks := unitAndPremiseMarks(move.Evidence)
	conclusion := HintConclusion{Placement: placement(move)}
	observe := hintDescriptor("hint.bug-plus-one.observe", cellArgument("cell", premise.Position), digitsArgument("digits", premise.Values))
	explain := hintDescriptor("hint.bug-plus-one.explain", digitArgument("digit", move.Cell.Value), cellArgument("cell", premise.Position), unitsArgument("units", move.Evidence.Units))
	apply := hintDescriptor("hint.bug-plus-one.place", digitArgument("digit", move.Cell.Value), cellArgument("cell", move.Cell.Position), strategyArgument("strategy", move.Technique))
	return []HintStep{
		{ID: "find-trivalue-cell", Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationMessage(observe)}, Marks: marks},
		{ID: "explain-extra-candidate", Kind: HintStepExplain, Message: explain, Narration: []HintNarrationChunk{narrationMessage(explain)}, Marks: marks},
		{ID: "apply-bug-plus-one", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: append(marks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func uniquenessSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	conclusion := HintConclusion{Placement: placement(move)}
	if evidence != nil {
		conclusion.Eliminations = evidence.Eliminations
	}
	if evidence == nil || len(evidence.Premises) < 4 {
		return evidenceSteps(move, conclusion)
	}
	premises := evidence.Premises
	pair := sharedValues(premises[:4]...)
	if len(pair) != 2 || len(evidence.Eliminations) == 0 {
		return evidenceSteps(move, conclusion)
	}
	marks := unitAndPremiseMarks(evidence)
	corners := make([]core.Position, 4)
	var expanded []solver.CandidateGroup
	for index, premise := range premises[:4] {
		corners[index] = premise.Position
		if len(premise.Values) > 2 {
			expanded = append(expanded, premise)
		}
	}
	observe := hintDescriptor("hint.unique-rectangle.observe", cellsArgument("cells", corners), digitsArgument("digits", pair))
	var explain message.Descriptor
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
		if evidence.Unit == nil || len(expanded) != 2 {
			return evidenceSteps(move, conclusion)
		}
		extras := removeValues(combinedValues(expanded...), pair)
		if len(extras) == 0 {
			return evidenceSteps(move, conclusion)
		}
		partnerCells := make([]core.Position, 0, len(premises)-4)
		for _, premise := range premises[4:] {
			partnerCells = append(partnerCells, premise.Position)
		}
		explain = hintDescriptor("hint.unique-rectangle.type3", digitsArgument("digits", extras), cellsArgument("cells", partnerCells), unitsArgument("unit", []solver.UnitRef{*evidence.Unit}))
	case "unique-rectangle-4":
		if evidence.Unit == nil || len(expanded) != 2 {
			return evidenceSteps(move, conclusion)
		}
		removed := evidence.Eliminations[0].Value
		conjugate := 0
		for _, digit := range pair {
			if digit != removed {
				conjugate = digit
			}
		}
		if conjugate == 0 {
			return evidenceSteps(move, conclusion)
		}
		explain = hintDescriptor("hint.unique-rectangle.type4", digitArgument("digit", conjugate), cellsArgument("cells", []core.Position{expanded[0].Position, expanded[1].Position}), unitsArgument("unit", []solver.UnitRef{*evidence.Unit}), digitArgument("removed_digit", removed))
	default:
		return evidenceSteps(move, conclusion)
	}
	var apply message.Descriptor
	if conclusion.Placement != nil {
		apply = hintDescriptor("hint.unique-rectangle.place", digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
	} else {
		apply = hintDescriptor("hint.unique-rectangle.apply", candidatesArgument("candidates", evidence.Eliminations), strategyArgument("strategy", move.Technique))
	}
	return []HintStep{
		{ID: "find-" + move.Technique, Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationMessage(observe)}, Marks: marks},
		{ID: "explain-" + move.Technique, Kind: HintStepExplain, Message: explain, Narration: []HintNarrationChunk{narrationMessage(explain)}, Marks: marks},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(marks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
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

func coloringSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	if evidence == nil {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	if len(evidence.Partitions) != 2 || len(evidence.Premises) == 0 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations})
	}
	marks := premiseCandidateMarks(evidence)
	nodes := visualNodes(evidence, false)
	links := visualLinks(evidence)
	firstNodes, firstLinks := visualPrefix(nodes, links)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	return []HintStep{
		{ID: "find-simple-coloring", Kind: HintStepObserve, Marks: marksForNodes(firstNodes), Nodes: firstNodes, Links: firstLinks},
		{ID: "compare-simple-coloring", Kind: HintStepExplain, Marks: marks, Nodes: nodes, Links: links},
		{ID: "apply-simple-coloring", Kind: conclusionKind(conclusion), Marks: append(marks, conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)},
	}
}

func xCycleSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	if evidence == nil {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	if len(evidence.Chain) < 3 || len(evidence.Links) < 3 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations})
	}
	marks := premiseCandidateMarks(evidence)
	nodes := visualNodes(evidence, true)
	links := visualLinks(evidence)
	firstNodes, firstLinks := visualPrefix(nodes, links)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	return []HintStep{
		{ID: "find-x-cycle", Kind: HintStepObserve, Marks: marksForNodes(firstNodes), Nodes: firstNodes, Links: firstLinks},
		{ID: "compare-x-cycle", Kind: HintStepExplain, Marks: marks, Nodes: nodes, Links: links},
		{ID: "apply-x-cycle", Kind: conclusionKind(conclusion), Marks: append(marks, conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)},
	}
}

func conclusionNarration(conclusion HintConclusion) []HintNarrationChunk {
	if conclusion.Placement != nil {
		return []HintNarrationChunk{narrationID("hint.narration.place-result"), narrationDigit(conclusion.Placement.Value), narrationMessage(hintDescriptor("hint.narration.in")), narrationTarget(cellTarget(conclusion.Placement.Position))}
	}
	chunks := []HintNarrationChunk{narrationMessage(hintDescriptor("hint.narration.remove"))}
	for index, ref := range conclusion.Eliminations {
		if index > 0 {
			chunks = append(chunks, narrationMessage(hintDescriptor("hint.narration.separator")))
		}
		chunks = append(chunks, narrationDigit(ref.Value), narrationMessage(hintDescriptor("hint.narration.from")), narrationTarget(cellTarget(ref.Position)))
	}
	return chunks
}

func xyChainSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	if evidence == nil {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	if len(evidence.Chain) < 4 || len(evidence.Links) == 0 || len(evidence.Eliminations) == 0 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations})
	}
	nodes := visualNodes(evidence, true)
	links := visualLinks(evidence)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	first := evidence.Chain[0]
	observe := hintDescriptor("hint.implication-chain.observe", cellsArgument("cell", []core.Position{first.Position}), digitsArgument("digits", first.Values))
	firstNodeIDs := map[string]struct{}{}
	for _, value := range first.Values {
		firstNodeIDs[candidateNodeID(solver.CandidateRef{Position: first.Position, Value: value})] = struct{}{}
	}
	var firstNodes []HintNode
	for _, node := range nodes {
		if _, ok := firstNodeIDs[node.ID]; ok {
			firstNodes = append(firstNodes, node)
		}
	}
	steps := []HintStep{{ID: "find-xy-chain", Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationMessage(observe)}, Marks: marksForNodes(firstNodes), Nodes: firstNodes, Links: []HintLink{}}}
	for index, relation := range evidence.Links {
		descriptor := hintDescriptor("hint.implication-chain.explain", candidateArgument("from", relation.From), candidateArgument("to", relation.To))
		visibleLinks := append([]HintLink(nil), links[:index+1]...)
		ids := make(map[string]struct{})
		for _, link := range visibleLinks {
			ids[link.From] = struct{}{}
			ids[link.To] = struct{}{}
		}
		var visibleNodes []HintNode
		for _, node := range nodes {
			if _, ok := ids[node.ID]; ok {
				visibleNodes = append(visibleNodes, node)
			}
		}
		steps = append(steps, HintStep{ID: fmt.Sprintf("explain-xy-chain-%d", index+1), Kind: HintStepExplain, Message: descriptor, Narration: []HintNarrationChunk{narrationMessage(descriptor), narrationTarget(candidateTarget(relation.From)), narrationTarget(candidateTarget(relation.To))}, Marks: marksForNodes(visibleNodes), Nodes: visibleNodes, Links: visibleLinks})
	}
	endpoint := sharedValues(evidence.Chain[0], evidence.Chain[len(evidence.Chain)-1])[0]
	positions := make([]core.Position, len(evidence.Eliminations))
	for i, ref := range evidence.Eliminations {
		positions[i] = ref.Position
	}
	apply := hintDescriptor("hint.implication-chain.apply", digitArgument("digit", endpoint), cellsArgument("cells", positions), strategyArgument("strategy", move.Technique))
	steps = append(steps, HintStep{ID: "apply-xy-chain", Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(marksForNodes(nodes), conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)})
	return steps
}
