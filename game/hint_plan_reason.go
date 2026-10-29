package game

import (
	"fmt"
	"sort"
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
	constraintMarks := append([]HintMark(nil), units...)
	constraintMarks = append(constraintMarks, HintMark{Target: cellTarget(move.Cell.Position), Role: HintRoleFocus})
	excludedDigits := make([]int, 0, len(move.Evidence.Witnesses))
	for _, witness := range move.Evidence.Witnesses {
		excludedDigits = append(excludedDigits, witness.Value)
		constraintMarks = append(constraintMarks, HintMark{Target: cellTarget(witness.Position), Role: HintRolePremise})
	}
	onlyMarks := append([]HintMark(nil), constraintMarks...)
	onlyMarks = append(onlyMarks, HintMark{Target: candidateTarget(ref), Role: HintRolePremise})
	applyMarks := append([]HintMark(nil), constraintMarks...)
	applyMarks = append(applyMarks, HintMark{Target: candidateTarget(ref), Role: HintRoleConclusion})
	constraints := hintDescriptor("hint.naked-single.constraints", cellArgument("cell", move.Cell.Position), digitsArgument("excluded_digits", excludedDigits))
	only := hintDescriptor("hint.naked-single.only", cellArgument("cell", move.Cell.Position), digitArgument("digit", move.Cell.Value))
	apply := hintDescriptor("hint.naked-single.apply", cellArgument("cell", move.Cell.Position), digitArgument("digit", move.Cell.Value), strategyArgument("strategy", move.Technique))
	return []HintStep{
		{ID: "inspect-" + compactCellName(move.Cell.Position) + "-constraints", Kind: HintStepObserve, Message: constraints, Narration: []HintNarrationChunk{narrationMessage(constraints)}, Marks: constraintMarks},
		{ID: "identify-" + compactCellName(move.Cell.Position) + "-candidate", Kind: HintStepExplain, Message: only, Narration: []HintNarrationChunk{narrationMessage(only)}, Marks: onlyMarks},
		{ID: fmt.Sprintf("place-%s-%d", compactCellName(move.Cell.Position), move.Cell.Value), Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: applyMarks, Effect: &HintEffect{Placement: &move.Cell}},
	}
}

func hiddenSingleSteps(move *solver.Move) []HintStep {
	ref := solver.CandidateRef{Position: move.Cell.Position, Value: move.Cell.Value}
	unit, _ := unitTarget(move.Evidence.Unit)
	baseMarks := []HintMark{{Target: unit, Role: HintRoleFocus}, {Target: candidateTarget(ref), Role: HintRolePremise}}
	blockerMarks := append([]HintMark(nil), baseMarks...)
	blockerCells := make([]core.Position, 0, len(move.Evidence.Witnesses))
	for _, witness := range move.Evidence.Witnesses {
		blockerCells = append(blockerCells, witness.Position)
		blockerMarks = append(blockerMarks, HintMark{Target: cellTarget(witness.Position), Role: HintRolePremise})
	}
	for _, ruledOut := range move.Evidence.RuledOut {
		blockerMarks = append(blockerMarks, HintMark{Target: candidateTarget(ruledOut), Role: HintRoleEliminated})
	}
	applyMarks := append([]HintMark(nil), blockerMarks...)
	applyMarks[1] = HintMark{Target: candidateTarget(ref), Role: HintRoleConclusion}
	unitName := fmt.Sprintf("%s %d", move.Evidence.Unit.Kind, move.Evidence.Unit.Index+1)
	inspect := hintDescriptor("hint.hidden-single.inspect", unitsArgument("unit", []solver.UnitRef{*move.Evidence.Unit}), digitArgument("digit", move.Cell.Value))
	blockers := hintDescriptor("hint.hidden-single.blockers", cellsArgument("blockers", blockerCells), digitArgument("digit", move.Cell.Value), unitsArgument("unit", []solver.UnitRef{*move.Evidence.Unit}))
	only := hintDescriptor("hint.hidden-single.only", cellArgument("cell", move.Cell.Position), digitArgument("digit", move.Cell.Value))
	apply := hintDescriptor("hint.hidden-single.apply", digitArgument("digit", move.Cell.Value), cellArgument("cell", move.Cell.Position), strategyArgument("strategy", move.Technique))
	return []HintStep{
		{ID: fmt.Sprintf("scan-%s-for-%d", strings.ReplaceAll(unitName, " ", ""), move.Cell.Value), Kind: HintStepObserve, Message: inspect, Narration: []HintNarrationChunk{narrationMessage(inspect)}, Marks: baseMarks},
		{ID: fmt.Sprintf("rule-out-%s-%d", strings.ReplaceAll(unitName, " ", ""), move.Cell.Value), Kind: HintStepExplain, Message: blockers, Narration: []HintNarrationChunk{narrationMessage(blockers)}, Marks: blockerMarks},
		{ID: fmt.Sprintf("identify-%s-%d", compactCellName(move.Cell.Position), move.Cell.Value), Kind: HintStepExplain, Message: only, Narration: []HintNarrationChunk{narrationMessage(only)}, Marks: blockerMarks},
		{ID: fmt.Sprintf("place-%s-%d", compactCellName(move.Cell.Position), move.Cell.Value), Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: applyMarks, Effect: &HintEffect{Placement: &move.Cell}},
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
	if move.Technique == "naked-pair" && len(evidence.Premises) == 2 && len(evidence.Eliminations) > 0 {
		digits := combinedValues(evidence.Premises...)
		units := []solver.UnitRef{*evidence.Unit}
		first := evidence.Premises[0]
		second := evidence.Premises[1]
		firstMarks := []HintMark{{Target: unit, Role: HintRoleFocus}}
		for _, value := range first.Values {
			firstMarks = append(firstMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: first.Position, Value: value}), Role: HintRolePremise})
		}
		pairMarks := append([]HintMark(nil), firstMarks...)
		for _, value := range second.Values {
			pairMarks = append(pairMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: second.Position, Value: value}), Role: HintRolePremise})
		}
		consequenceMarks := append([]HintMark(nil), pairMarks...)
		consequenceMarks = append(consequenceMarks, conclusionMarks(conclusion)...)
		inspect := hintDescriptor("hint.naked-pair.inspect", unitsArgument("unit", units), cellArgument("cell", first.Position), digitArgument("first_digit", digits[0]), digitArgument("second_digit", digits[1]))
		match := hintDescriptor("hint.naked-pair.match", cellArgument("cell", second.Position), digitArgument("first_digit", digits[0]), digitArgument("second_digit", digits[1]))
		reserve := hintDescriptor("hint.naked-pair.reserve", digitsArgument("digits", digits), digitArgument("digit", evidence.Eliminations[0].Value), unitsArgument("unit", units))
		apply := hintDescriptor("hint.naked-pair.apply", candidatesArgument("candidates", evidence.Eliminations), strategyArgument("strategy", move.Technique))
		return []HintStep{
			{ID: "inspect-" + compactCellName(first.Position), Kind: HintStepObserve, Message: inspect, Narration: []HintNarrationChunk{narrationMessage(inspect)}, Marks: firstMarks},
			{ID: "match-" + compactCellName(second.Position), Kind: HintStepExplain, Message: match, Narration: []HintNarrationChunk{narrationMessage(match)}, Marks: pairMarks},
			{ID: "reserve-" + strings.Join(cellIDs, "-"), Kind: HintStepExplain, Message: reserve, Narration: []HintNarrationChunk{narrationMessage(reserve)}, Marks: consequenceMarks},
			{ID: "apply-naked-pair", Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: consequenceMarks, Effect: conclusionEffect(conclusion)},
		}
	}
	if move.Technique == "naked-triple" && len(evidence.Premises) == 3 && conclusion.Placement != nil {
		digits := combinedValues(evidence.Premises...)
		units := []solver.UnitRef{*evidence.Unit}
		first := evidence.Premises[0]
		second := evidence.Premises[1]
		third := evidence.Premises[2]
		marksFor := func(groups ...solver.CandidateGroup) []HintMark {
			marks := []HintMark{{Target: unit, Role: HintRoleFocus}}
			for _, group := range groups {
				for _, value := range group.Values {
					marks = append(marks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: group.Position, Value: value}), Role: HintRolePremise})
				}
			}
			return marks
		}
		firstMarks := marksFor(first)
		secondMarks := marksFor(first, second)
		tripleMarks := marksFor(first, second, third)
		consequenceMarks := append([]HintMark(nil), tripleMarks...)
		removedDigits := make([]int, 0, len(evidence.RuledOut))
		for _, ref := range evidence.RuledOut {
			if ref.Position == conclusion.Placement.Position {
				removedDigits = append(removedDigits, ref.Value)
				consequenceMarks = append(consequenceMarks, HintMark{Target: candidateTarget(ref), Role: HintRoleEliminated})
			}
		}
		consequenceMarks = append(consequenceMarks, conclusionMarks(conclusion)...)
		firstDescriptor := hintDescriptor("hint.naked-triple.first", unitsArgument("unit", units), cellArgument("cell", first.Position), digitArgument("first_digit", first.Values[0]), digitArgument("second_digit", first.Values[1]))
		secondDescriptor := hintDescriptor("hint.naked-triple.second", cellArgument("cell", second.Position), digitArgument("first_digit", second.Values[0]), digitArgument("second_digit", second.Values[1]))
		thirdDescriptor := hintDescriptor("hint.naked-triple.third", cellArgument("cell", third.Position), digitArgument("first_digit", third.Values[0]), digitArgument("second_digit", third.Values[1]), digitsArgument("reserved_digits", digits))
		consequenceDescriptor := hintDescriptor("hint.naked-triple.consequence", digitsArgument("reserved_digits", digits), digitsArgument("removed_digits", removedDigits), cellArgument("cell", conclusion.Placement.Position), digitArgument("remaining_digit", conclusion.Placement.Value))
		applyDescriptor := hintDescriptor("hint.naked-triple.apply", digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
		return []HintStep{
			{ID: "inspect-" + compactCellName(first.Position), Kind: HintStepObserve, Message: firstDescriptor, Narration: []HintNarrationChunk{narrationMessage(firstDescriptor)}, Marks: firstMarks},
			{ID: "add-" + compactCellName(second.Position), Kind: HintStepExplain, Message: secondDescriptor, Narration: []HintNarrationChunk{narrationMessage(secondDescriptor)}, Marks: secondMarks},
			{ID: "complete-" + strings.Join(cellIDs, "-"), Kind: HintStepExplain, Message: thirdDescriptor, Narration: []HintNarrationChunk{narrationMessage(thirdDescriptor)}, Marks: tripleMarks},
			{ID: "reserve-" + strings.Join(cellIDs, "-"), Kind: HintStepExplain, Message: consequenceDescriptor, Narration: []HintNarrationChunk{narrationMessage(consequenceDescriptor)}, Marks: consequenceMarks},
			{ID: "apply-naked-triple", Kind: HintStepApply, Message: applyDescriptor, Narration: actionNarration(applyDescriptor), Marks: consequenceMarks, Effect: conclusionEffect(conclusion)},
		}
	}
	if move.Technique == "naked-quad" && len(evidence.Premises) == 4 && conclusion.Placement != nil {
		digits := combinedValues(evidence.Premises...)
		units := []solver.UnitRef{*evidence.Unit}
		ordered := append([]solver.CandidateGroup(nil), evidence.Premises...)
		sort.SliceStable(ordered, func(i, j int) bool {
			if len(ordered[i].Values) != len(ordered[j].Values) {
				return len(ordered[i].Values) < len(ordered[j].Values)
			}
			if ordered[i].Position.Row != ordered[j].Position.Row {
				return ordered[i].Position.Row < ordered[j].Position.Row
			}
			return ordered[i].Position.Column < ordered[j].Position.Column
		})
		marksFor := func(groups ...solver.CandidateGroup) []HintMark {
			var marks []HintMark
			for _, group := range groups {
				for _, value := range group.Values {
					marks = append(marks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: group.Position, Value: value}), Role: HintRolePremise})
				}
			}
			return marks
		}
		progressiveMarks := make([][]HintMark, len(ordered))
		for index := range ordered {
			progressiveMarks[index] = marksFor(ordered[:index+1]...)
		}
		steps := make([]HintStep, 0, 6)
		for index, group := range ordered {
			descriptor := hintDescriptor("hint.naked-quad.cell", unitsArgument("unit", units), cellArgument("cell", group.Position), digitsArgument("digits", group.Values))
			kind := HintStepExplain
			if index == 0 {
				kind = HintStepObserve
			}
			steps = append(steps, HintStep{ID: fmt.Sprintf("naked-quad-cell-%d-%s", index+1, compactCellName(group.Position)), Kind: kind, Message: descriptor, Narration: []HintNarrationChunk{narrationMessage(descriptor)}, Marks: progressiveMarks[index]})
		}
		reserveMarks := append([]HintMark(nil), progressiveMarks[len(progressiveMarks)-1]...)
		reserveMarks = append(reserveMarks, HintMark{Target: unit, Role: HintRoleFocus})
		reserve := hintDescriptor("hint.subset-reservation.explain", unitsArgument("unit", units), cellsArgument("cells", cells), digitsArgument("digits", digits))
		apply := hintDescriptor("hint.subset-reservation.place", digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
		applyMarks := append([]HintMark(nil), progressiveMarks[len(progressiveMarks)-1]...)
		applyMarks = append(applyMarks, conclusionMarks(conclusion)...)
		steps = append(steps,
			HintStep{ID: "reserve-" + strings.Join(cellIDs, "-"), Kind: HintStepExplain, Message: reserve, Narration: []HintNarrationChunk{narrationMessage(reserve)}, Marks: reserveMarks},
			HintStep{ID: "apply-naked-quad", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: applyMarks, Effect: conclusionEffect(conclusion)},
		)
		return steps
	}
	if move.Technique == "naked-triple" || move.Technique == "naked-quad" {
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
	if move.Technique == "hidden-pair" && len(evidence.Premises) == 2 && len(digits) == 2 && len(evidence.Eliminations) > 0 {
		firstDigit, secondDigit := digits[0], digits[1]
		firstMarks := make([]HintMark, 0, 2)
		secondMarks := []HintMark{{Target: unit, Role: HintRoleFocus}}
		for _, premise := range evidence.Premises {
			firstMarks = append(firstMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: firstDigit}), Role: HintRolePremise})
			secondMarks = append(secondMarks,
				HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: firstDigit}), Role: HintRolePremise},
				HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: secondDigit}), Role: HintRolePremise},
			)
		}
		consequenceMarks := append([]HintMark(nil), secondMarks...)
		applyMarks := append([]HintMark(nil), secondMarks...)
		for _, ref := range evidence.Eliminations {
			consequenceMarks = append(consequenceMarks, HintMark{Target: candidateTarget(ref), Role: HintRoleEliminated})
			applyMarks = append(applyMarks, HintMark{Target: candidateTarget(ref), Role: HintRoleConclusion})
		}
		first := hintDescriptor("hint.hidden-pair.first", unitsArgument("unit", units), digitArgument("digit", firstDigit), cellsArgument("cells", cells))
		second := hintDescriptor("hint.hidden-pair.second", digitArgument("digit", secondDigit), cellsArgument("cells", cells), digitsArgument("digits", digits))
		consequence := hintDescriptor("hint.hidden-pair.consequence", digitsArgument("digits", digits), candidatesArgument("candidates", evidence.Eliminations))
		apply := hintDescriptor("hint.hidden-pair.apply", candidatesArgument("candidates", evidence.Eliminations), strategyArgument("strategy", move.Technique))
		return []HintStep{
			{ID: "find-first-hidden-pair-digit", Kind: HintStepObserve, Message: first, Narration: []HintNarrationChunk{narrationMessage(first)}, Marks: firstMarks},
			{ID: "complete-hidden-pair", Kind: HintStepExplain, Message: second, Narration: []HintNarrationChunk{narrationMessage(second)}, Marks: secondMarks},
			{ID: "identify-hidden-pair-removals", Kind: HintStepExplain, Message: consequence, Narration: []HintNarrationChunk{narrationMessage(consequence)}, Marks: consequenceMarks},
			{ID: "apply-hidden-pair", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: applyMarks, Effect: conclusionEffect(conclusion)},
		}
	}
	completeHiddenTriple := len(evidence.Premises) == 3 && len(digits) == 3
	for _, premise := range evidence.Premises {
		for _, digit := range digits {
			found := false
			for _, value := range premise.Values {
				if value == digit {
					found = true
					break
				}
			}
			completeHiddenTriple = completeHiddenTriple && found
		}
	}
	if move.Technique == "hidden-triple" && completeHiddenTriple && len(evidence.Eliminations) > 0 {
		firstMarks := make([]HintMark, 0, 3)
		secondMarks := []HintMark{{Target: unit, Role: HintRoleFocus}}
		thirdMarks := []HintMark{{Target: unit, Role: HintRoleFocus}}
		for _, premise := range evidence.Premises {
			firstMarks = append(firstMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: digits[0]}), Role: HintRolePremise})
			for _, digit := range digits[:2] {
				secondMarks = append(secondMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: digit}), Role: HintRolePremise})
			}
			for _, digit := range digits {
				thirdMarks = append(thirdMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: digit}), Role: HintRolePremise})
			}
		}
		consequenceMarks := append([]HintMark(nil), thirdMarks...)
		applyMarks := append([]HintMark(nil), thirdMarks...)
		for _, ref := range evidence.Eliminations {
			consequenceMarks = append(consequenceMarks, HintMark{Target: candidateTarget(ref), Role: HintRoleEliminated})
			applyMarks = append(applyMarks, HintMark{Target: candidateTarget(ref), Role: HintRoleConclusion})
		}
		first := hintDescriptor("hint.hidden-triple.first", unitsArgument("unit", units), digitArgument("digit", digits[0]), cellsArgument("cells", cells))
		second := hintDescriptor("hint.hidden-triple.second", unitsArgument("unit", units), digitsArgument("digits", digits[:2]), cellsArgument("cells", cells))
		third := hintDescriptor("hint.hidden-triple.third", unitsArgument("unit", units), digitsArgument("digits", digits), cellsArgument("cells", cells))
		consequence := hintDescriptor("hint.hidden-triple.consequence", digitsArgument("digits", digits), unitsArgument("unit", units))
		apply := hintDescriptor("hint.hidden-triple.apply", candidatesArgument("candidates", evidence.Eliminations), strategyArgument("strategy", move.Technique))
		return []HintStep{
			{ID: "find-first-hidden-triple-digit", Kind: HintStepObserve, Message: first, Narration: []HintNarrationChunk{narrationMessage(first)}, Marks: firstMarks},
			{ID: "add-second-hidden-triple-digit", Kind: HintStepExplain, Message: second, Narration: []HintNarrationChunk{narrationMessage(second)}, Marks: secondMarks},
			{ID: "complete-hidden-triple", Kind: HintStepExplain, Message: third, Narration: []HintNarrationChunk{narrationMessage(third)}, Marks: thirdMarks},
			{ID: "identify-hidden-triple-removals", Kind: HintStepExplain, Message: consequence, Narration: []HintNarrationChunk{narrationMessage(consequence)}, Marks: consequenceMarks},
			{ID: "apply-hidden-triple", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: applyMarks, Effect: conclusionEffect(conclusion)},
		}
	}
	completeHiddenQuad := len(evidence.Premises) == 4 && len(digits) == 4
	for _, premise := range evidence.Premises {
		for _, digit := range digits {
			found := false
			for _, value := range premise.Values {
				if value == digit {
					found = true
					break
				}
			}
			completeHiddenQuad = completeHiddenQuad && found
		}
	}
	if move.Technique == "hidden-quad" && completeHiddenQuad && len(evidence.Eliminations) > 0 {
		firstMarks := make([]HintMark, 0, 4)
		secondMarks := []HintMark{{Target: unit, Role: HintRoleFocus}}
		thirdMarks := []HintMark{{Target: unit, Role: HintRoleFocus}}
		fourthMarks := []HintMark{{Target: unit, Role: HintRoleFocus}}
		for _, premise := range evidence.Premises {
			firstMarks = append(firstMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: digits[0]}), Role: HintRolePremise})
			for _, digit := range digits[:2] {
				secondMarks = append(secondMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: digit}), Role: HintRolePremise})
			}
			for _, digit := range digits[:3] {
				thirdMarks = append(thirdMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: digit}), Role: HintRolePremise})
			}
			for _, digit := range digits {
				fourthMarks = append(fourthMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: digit}), Role: HintRolePremise})
			}
		}
		consequenceMarks := append([]HintMark(nil), fourthMarks...)
		applyMarks := append([]HintMark(nil), fourthMarks...)
		for _, ref := range evidence.Eliminations {
			consequenceMarks = append(consequenceMarks, HintMark{Target: candidateTarget(ref), Role: HintRoleEliminated})
			applyMarks = append(applyMarks, HintMark{Target: candidateTarget(ref), Role: HintRoleConclusion})
		}
		first := hintDescriptor("hint.hidden-quad.first", unitsArgument("unit", units), digitArgument("digit", digits[0]), cellsArgument("cells", cells))
		second := hintDescriptor("hint.hidden-quad.second", unitsArgument("unit", units), digitsArgument("digits", digits[:2]), cellsArgument("cells", cells))
		third := hintDescriptor("hint.hidden-quad.third", unitsArgument("unit", units), digitsArgument("digits", digits[:3]), cellsArgument("cells", cells))
		fourth := hintDescriptor("hint.hidden-quad.fourth", unitsArgument("unit", units), digitsArgument("digits", digits), cellsArgument("cells", cells))
		consequence := hintDescriptor("hint.hidden-quad.consequence", digitsArgument("digits", digits), unitsArgument("unit", units))
		apply := hintDescriptor("hint.hidden-quad.apply", candidatesArgument("candidates", evidence.Eliminations), strategyArgument("strategy", move.Technique))
		return []HintStep{
			{ID: "find-first-hidden-quad-digit", Kind: HintStepObserve, Message: first, Narration: []HintNarrationChunk{narrationMessage(first)}, Marks: firstMarks},
			{ID: "add-second-hidden-quad-digit", Kind: HintStepExplain, Message: second, Narration: []HintNarrationChunk{narrationMessage(second)}, Marks: secondMarks},
			{ID: "add-third-hidden-quad-digit", Kind: HintStepExplain, Message: third, Narration: []HintNarrationChunk{narrationMessage(third)}, Marks: thirdMarks},
			{ID: "complete-hidden-quad", Kind: HintStepExplain, Message: fourth, Narration: []HintNarrationChunk{narrationMessage(fourth)}, Marks: fourthMarks},
			{ID: "identify-hidden-quad-removals", Kind: HintStepExplain, Message: consequence, Narration: []HintNarrationChunk{narrationMessage(consequence)}, Marks: consequenceMarks},
			{ID: "apply-hidden-quad", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: applyMarks, Effect: conclusionEffect(conclusion)},
		}
	}
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
	if move.Technique == "pointing-pair" && source.Kind == solver.UnitBox && len(evidence.Premises) == 2 && conclusion.Placement != nil {
		first := solver.CandidateRef{Position: evidence.Premises[0].Position, Value: digit}
		second := solver.CandidateRef{Position: evidence.Premises[1].Position, Value: digit}
		removed := solver.CandidateRef{Position: conclusion.Placement.Position, Value: digit}
		for _, ref := range append(append([]solver.CandidateRef(nil), evidence.RuledOut...), evidence.Eliminations...) {
			if ref.Position == conclusion.Placement.Position && ref.Value == digit {
				removed = ref
				break
			}
		}
		boxTarget, _ := unitTarget(&source)
		columnTarget, _ := unitTarget(&intersection)
		firstMarks := []HintMark{
			{Target: boxTarget, Role: HintRoleFocus},
			{Target: candidateTarget(first), Role: HintRolePremise},
		}
		pairMarks := append(append([]HintMark(nil), firstMarks...),
			HintMark{Target: candidateTarget(second), Role: HintRolePremise},
			HintMark{Target: columnTarget, Role: HintRoleFocus},
		)
		consequenceMarks := append(append([]HintMark(nil), pairMarks...), HintMark{Target: candidateTarget(removed), Role: HintRoleEliminated})
		applyMarks := append(append([]HintMark(nil), consequenceMarks...), HintMark{Target: candidateTarget(solver.CandidateRef{Position: conclusion.Placement.Position, Value: conclusion.Placement.Value}), Role: HintRoleConclusion})
		firstMessage := hintDescriptor("hint.pointing-pair.first", unitsArgument("source_unit", []solver.UnitRef{source}), digitArgument("digit", digit), cellArgument("cell", first.Position))
		secondMessage := hintDescriptor("hint.pointing-pair.second", digitArgument("digit", digit), unitsArgument("source_unit", []solver.UnitRef{source}), cellArgument("cell", second.Position), unitsArgument("intersection_unit", []solver.UnitRef{intersection}))
		consequenceMessage := hintDescriptor("hint.pointing-pair.consequence", digitArgument("digit", digit), unitsArgument("source_unit", []solver.UnitRef{source}), unitsArgument("intersection_unit", []solver.UnitRef{intersection}), message.Argument{Name: "candidate", Kind: message.ArgumentCandidate, Candidate: &message.Candidate{Cell: message.Cell{Row: removed.Position.Row + 1, Column: removed.Position.Column + 1}, Digit: removed.Value}})
		applyMessage := hintDescriptor("hint.pointing-pair.apply", cellArgument("cell", conclusion.Placement.Position), digitArgument("digit", conclusion.Placement.Value), strategyArgument("strategy", move.Technique))
		return []HintStep{
			{ID: "inspect-" + compactCellName(first.Position) + "-candidate", Kind: HintStepObserve, Message: firstMessage, Narration: []HintNarrationChunk{narrationMessage(firstMessage)}, Marks: firstMarks},
			{ID: "complete-pointing-pair", Kind: HintStepExplain, Message: secondMessage, Narration: []HintNarrationChunk{narrationMessage(secondMessage)}, Marks: pairMarks},
			{ID: "remove-" + compactCellName(removed.Position) + fmt.Sprintf("-%d", removed.Value), Kind: HintStepExplain, Message: consequenceMessage, Narration: []HintNarrationChunk{narrationMessage(consequenceMessage)}, Marks: consequenceMarks},
			{ID: "place-" + compactCellName(conclusion.Placement.Position) + fmt.Sprintf("-%d", conclusion.Placement.Value), Kind: HintStepApply, Message: applyMessage, Narration: actionNarration(applyMessage), Marks: applyMarks, Effect: conclusionEffect(conclusion)},
		}
	}
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
		if move.Technique == "x-wing" && len(evidence.Premises) == 4 && baseUnits[0].Kind == solver.UnitRow && baseUnits[1].Kind == solver.UnitRow && coverUnits[0].Kind == solver.UnitColumn && coverUnits[1].Kind == solver.UnitColumn {
			firstBase, secondBase := baseUnits[0], baseUnits[1]
			firstMarks := make([]HintMark, 0, 2)
			patternMarks := make([]HintMark, 0, 6)
			for _, premise := range evidence.Premises[:2] {
				firstMarks = append(firstMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: digit}), Role: HintRolePremise})
			}
			for _, premise := range evidence.Premises {
				patternMarks = append(patternMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: digit}), Role: HintRolePremise})
			}
			for _, ref := range baseUnits {
				target, _ := unitTarget(&ref)
				patternMarks = append(patternMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
			consequenceMarks := append([]HintMark(nil), patternMarks...)
			for _, ref := range coverUnits {
				target, _ := unitTarget(&ref)
				consequenceMarks = append(consequenceMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
			first := hintDescriptor("hint.x-wing.first", unitsArgument("base_unit", []solver.UnitRef{firstBase}), digitArgument("digit", digit), unitsArgument("cover_units", coverUnits))
			second := hintDescriptor("hint.x-wing.second", unitsArgument("base_unit", []solver.UnitRef{secondBase}), digitArgument("digit", digit), unitsArgument("cover_units", coverUnits))
			consequence := hintDescriptor("hint.x-wing.consequence", unitsArgument("base_units", baseUnits), unitsArgument("cover_units", coverUnits), digitArgument("digit", digit))
			apply := hintDescriptor("hint.x-wing.apply", digitArgument("digit", digit), cellsArgument("cells", positions), strategyArgument("strategy", move.Technique))
			return []HintStep{
				{ID: "find-first-x-wing-line", Kind: HintStepObserve, Message: first, Narration: []HintNarrationChunk{narrationMessage(first)}, Marks: firstMarks},
				{ID: "find-matching-x-wing-line", Kind: HintStepExplain, Message: second, Narration: []HintNarrationChunk{narrationMessage(second)}, Marks: patternMarks},
				{ID: "explain-x-wing-columns", Kind: HintStepExplain, Message: consequence, Narration: []HintNarrationChunk{narrationMessage(consequence)}, Marks: consequenceMarks},
				{ID: "apply-x-wing", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: append(consequenceMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
			}
		}
		if move.Technique == "swordfish" && len(evidence.Premises) == 6 && baseUnits[0].Kind == solver.UnitRow && baseUnits[1].Kind == solver.UnitRow && baseUnits[2].Kind == solver.UnitRow && coverUnits[0].Kind == solver.UnitColumn && coverUnits[1].Kind == solver.UnitColumn && coverUnits[2].Kind == solver.UnitColumn {
			premiseMarks := premiseCandidateMarks(evidence)
			firstMarks := append([]HintMark(nil), premiseMarks[:2]...)
			secondMarks := append([]HintMark(nil), premiseMarks[:4]...)
			for _, ref := range baseUnits[:2] {
				target, _ := unitTarget(&ref)
				secondMarks = append(secondMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
			thirdMarks := append([]HintMark(nil), premiseMarks...)
			for _, ref := range baseUnits {
				target, _ := unitTarget(&ref)
				thirdMarks = append(thirdMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
			consequenceMarks := append([]HintMark(nil), thirdMarks...)
			for _, ref := range coverUnits {
				target, _ := unitTarget(&ref)
				consequenceMarks = append(consequenceMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
			lineCoverUnits := func(premises []solver.CandidateGroup) []solver.UnitRef {
				refs := make([]solver.UnitRef, len(premises))
				for index, premise := range premises {
					refs[index] = solver.UnitRef{Kind: solver.UnitColumn, Index: premise.Position.Column}
				}
				return refs
			}
			first := hintDescriptor("hint.fish-confinement.observe", unitsArgument("base_units", baseUnits[:1]), digitArgument("digit", digit), unitsArgument("cover_units", lineCoverUnits(evidence.Premises[:2])))
			second := hintDescriptor("hint.fish-confinement.observe", unitsArgument("base_units", baseUnits[1:2]), digitArgument("digit", digit), unitsArgument("cover_units", lineCoverUnits(evidence.Premises[2:4])))
			third := hintDescriptor("hint.fish-confinement.observe", unitsArgument("base_units", baseUnits[2:3]), digitArgument("digit", digit), unitsArgument("cover_units", lineCoverUnits(evidence.Premises[4:6])))
			consequence := hintDescriptor("hint.swordfish.consequence", unitsArgument("base_units", baseUnits), unitsArgument("cover_units", coverUnits), digitArgument("digit", digit))
			apply := hintDescriptor("hint.fish-confinement.apply", digitArgument("digit", digit), cellsArgument("cells", positions), strategyArgument("strategy", move.Technique))
			return []HintStep{
				{ID: "find-first-swordfish-row", Kind: HintStepObserve, Message: first, Narration: []HintNarrationChunk{narrationMessage(first)}, Marks: firstMarks},
				{ID: "add-second-swordfish-row", Kind: HintStepExplain, Message: second, Narration: []HintNarrationChunk{narrationMessage(second)}, Marks: secondMarks},
				{ID: "complete-swordfish-rows", Kind: HintStepExplain, Message: third, Narration: []HintNarrationChunk{narrationMessage(third)}, Marks: thirdMarks},
				{ID: "explain-swordfish-columns", Kind: HintStepExplain, Message: consequence, Narration: []HintNarrationChunk{narrationMessage(consequence)}, Marks: consequenceMarks},
				{ID: "apply-swordfish", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: append(consequenceMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
			}
		}
		if move.Technique == "jellyfish" && len(evidence.Premises) == 8 && baseUnits[0].Kind == solver.UnitRow && baseUnits[1].Kind == solver.UnitRow && baseUnits[2].Kind == solver.UnitRow && baseUnits[3].Kind == solver.UnitRow && coverUnits[0].Kind == solver.UnitColumn && coverUnits[1].Kind == solver.UnitColumn && coverUnits[2].Kind == solver.UnitColumn && coverUnits[3].Kind == solver.UnitColumn {
			premiseMarks := premiseCandidateMarks(evidence)
			lineCoverUnits := func(premises []solver.CandidateGroup) []solver.UnitRef {
				refs := make([]solver.UnitRef, len(premises))
				for index, premise := range premises {
					refs[index] = solver.UnitRef{Kind: solver.UnitColumn, Index: premise.Position.Column}
				}
				return refs
			}
			lineMarks := func(lineCount int) []HintMark {
				marks := append([]HintMark(nil), premiseMarks[:lineCount*2]...)
				if lineCount > 1 {
					for _, ref := range baseUnits[:lineCount] {
						target, _ := unitTarget(&ref)
						marks = append(marks, HintMark{Target: target, Role: HintRoleFocus})
					}
				}
				return marks
			}
			firstMarks := lineMarks(1)
			secondMarks := lineMarks(2)
			thirdMarks := lineMarks(3)
			fourthMarks := lineMarks(4)
			consequenceMarks := append([]HintMark(nil), fourthMarks...)
			for _, ref := range coverUnits {
				target, _ := unitTarget(&ref)
				consequenceMarks = append(consequenceMarks, HintMark{Target: target, Role: HintRoleFocus})
			}
			first := hintDescriptor("hint.fish-confinement.observe", unitsArgument("base_units", baseUnits[:1]), digitArgument("digit", digit), unitsArgument("cover_units", lineCoverUnits(evidence.Premises[:2])))
			second := hintDescriptor("hint.fish-confinement.observe", unitsArgument("base_units", baseUnits[1:2]), digitArgument("digit", digit), unitsArgument("cover_units", lineCoverUnits(evidence.Premises[2:4])))
			third := hintDescriptor("hint.fish-confinement.observe", unitsArgument("base_units", baseUnits[2:3]), digitArgument("digit", digit), unitsArgument("cover_units", lineCoverUnits(evidence.Premises[4:6])))
			fourth := hintDescriptor("hint.fish-confinement.observe", unitsArgument("base_units", baseUnits[3:4]), digitArgument("digit", digit), unitsArgument("cover_units", lineCoverUnits(evidence.Premises[6:8])))
			consequence := hintDescriptor("hint.jellyfish.consequence", unitsArgument("base_units", baseUnits), unitsArgument("cover_units", coverUnits), digitArgument("digit", digit))
			apply := hintDescriptor("hint.fish-confinement.apply", digitArgument("digit", digit), cellsArgument("cells", positions), strategyArgument("strategy", move.Technique))
			return []HintStep{
				{ID: "find-first-jellyfish-row", Kind: HintStepObserve, Message: first, Narration: []HintNarrationChunk{narrationMessage(first)}, Marks: firstMarks},
				{ID: "add-second-jellyfish-row", Kind: HintStepExplain, Message: second, Narration: []HintNarrationChunk{narrationMessage(second)}, Marks: secondMarks},
				{ID: "add-third-jellyfish-row", Kind: HintStepExplain, Message: third, Narration: []HintNarrationChunk{narrationMessage(third)}, Marks: thirdMarks},
				{ID: "complete-jellyfish-rows", Kind: HintStepExplain, Message: fourth, Narration: []HintNarrationChunk{narrationMessage(fourth)}, Marks: fourthMarks},
				{ID: "explain-jellyfish-columns", Kind: HintStepExplain, Message: consequence, Narration: []HintNarrationChunk{narrationMessage(consequence)}, Marks: consequenceMarks},
				{ID: "apply-jellyfish", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: append(consequenceMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
			}
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
		if move.Technique == "xy-wing" {
			if len(pivot.Values) != 2 || len(wings[0].Values) != 2 || len(wings[1].Values) != 2 {
				return evidenceSteps(move, conclusion)
			}
			wingForPivot := func(value int) (solver.CandidateGroup, bool) {
				for _, wing := range wings {
					if containsValue(wing.Values, value) {
						return wing, true
					}
				}
				return solver.CandidateGroup{}, false
			}
			firstWing, firstOK := wingForPivot(pivot.Values[0])
			secondWing, secondOK := wingForPivot(pivot.Values[1])
			if !firstOK || !secondOK || firstWing.Position == secondWing.Position {
				return evidenceSteps(move, conclusion)
			}
			pivotNodes := nodesForCandidateRefs(nodes,
				solver.CandidateRef{Position: pivot.Position, Value: pivot.Values[0]},
				solver.CandidateRef{Position: pivot.Position, Value: pivot.Values[1]},
			)
			firstCaseNodes := nodesForCandidateRefs(nodes,
				solver.CandidateRef{Position: pivot.Position, Value: pivot.Values[0]},
				solver.CandidateRef{Position: pivot.Position, Value: pivot.Values[1]},
				solver.CandidateRef{Position: firstWing.Position, Value: pivot.Values[0]},
				solver.CandidateRef{Position: firstWing.Position, Value: actionDigit},
			)
			firstCaseLink := HintLink{From: candidateNodeID(solver.CandidateRef{Position: pivot.Position, Value: pivot.Values[0]}), To: candidateNodeID(solver.CandidateRef{Position: firstWing.Position, Value: actionDigit}), Kind: solver.LinkImplication}
			secondCaseLink := HintLink{From: candidateNodeID(solver.CandidateRef{Position: pivot.Position, Value: pivot.Values[1]}), To: candidateNodeID(solver.CandidateRef{Position: secondWing.Position, Value: actionDigit}), Kind: solver.LinkImplication}
			caseLinks := []HintLink{firstCaseLink, secondCaseLink}
			pivotMessage := hintDescriptor("hint.xy-wing.pivot", cellArgument("pivot", pivot.Position), digitsArgument("digits", pivot.Values))
			firstCaseMessage := hintDescriptor("hint.xy-wing.case", cellArgument("pivot", pivot.Position), digitArgument("pivot_digit", pivot.Values[0]), cellArgument("wing", firstWing.Position), digitArgument("shared_digit", actionDigit))
			secondCaseMessage := hintDescriptor("hint.xy-wing.case", cellArgument("pivot", pivot.Position), digitArgument("pivot_digit", pivot.Values[1]), cellArgument("wing", secondWing.Position), digitArgument("shared_digit", actionDigit))
			guarantee := hintDescriptor("hint.xy-wing.guarantee", cellArgument("pivot", pivot.Position), cellsArgument("wing_cells", wingCells), digitArgument("shared_digit", actionDigit))
			var target core.Position
			if conclusion.Placement != nil {
				target = conclusion.Placement.Position
			} else if len(evidence.Eliminations) > 0 {
				target = evidence.Eliminations[0].Position
			} else {
				return evidenceSteps(move, conclusion)
			}
			targetRef := solver.CandidateRef{Position: target, Value: actionDigit}
			finalNodes := append([]HintNode(nil), nodes...)
			finalNodes = append(finalNodes, HintNode{ID: candidateNodeID(targetRef), Target: candidateTarget(targetRef)})
			finalLinks := append([]HintLink(nil), caseLinks...)
			for _, wing := range wings {
				finalLinks = append(finalLinks, HintLink{From: candidateNodeID(solver.CandidateRef{Position: wing.Position, Value: actionDigit}), To: candidateNodeID(targetRef), Kind: solver.LinkWeak})
			}
			var apply message.Descriptor
			if conclusion.Placement != nil {
				apply = hintDescriptor("hint.xy-wing.place", cellArgument("cell", target), cellsArgument("wing_cells", wingCells), digitArgument("shared_digit", actionDigit), digitArgument("digit", conclusion.Placement.Value), strategyArgument("strategy", move.Technique))
			} else {
				positions := make([]core.Position, len(evidence.Eliminations))
				for index, ref := range evidence.Eliminations {
					positions[index] = ref.Position
				}
				apply = hintDescriptor("hint.xy-wing.apply", cellsArgument("cells", positions), cellsArgument("wing_cells", wingCells), digitArgument("shared_digit", actionDigit), strategyArgument("strategy", move.Technique))
			}
			return []HintStep{
				{ID: "find-xy-wing-pivot", Kind: HintStepObserve, Message: pivotMessage, Narration: []HintNarrationChunk{narrationMessage(pivotMessage)}, Marks: marksForNodes(pivotNodes), Nodes: pivotNodes, Links: []HintLink{}},
				{ID: "follow-xy-wing-first-case", Kind: HintStepExplain, Message: firstCaseMessage, Narration: []HintNarrationChunk{narrationMessage(firstCaseMessage)}, Marks: marksForNodes(firstCaseNodes), Nodes: firstCaseNodes, Links: []HintLink{firstCaseLink}},
				{ID: "follow-xy-wing-second-case", Kind: HintStepExplain, Message: secondCaseMessage, Narration: []HintNarrationChunk{narrationMessage(secondCaseMessage)}, Marks: marksForNodes(nodes), Nodes: nodes, Links: caseLinks},
				{ID: "combine-xy-wing-cases", Kind: HintStepExplain, Message: guarantee, Narration: []HintNarrationChunk{narrationMessage(guarantee)}, Marks: baseMarks, Nodes: nodes, Links: caseLinks},
				{ID: "apply-xy-wing", Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(baseMarks, conclusionMarks(conclusion)...), Nodes: finalNodes, Links: finalLinks, Effect: conclusionEffect(conclusion)},
			}
		}
		if move.Technique == "xyz-wing" {
			if len(pivot.Values) != 3 || len(wings[0].Values) != 2 || len(wings[1].Values) != 2 || conclusion.Placement != nil {
				return evidenceSteps(move, conclusion)
			}
			nonShared := make([]int, 0, 2)
			for _, value := range pivot.Values {
				if value != actionDigit {
					nonShared = append(nonShared, value)
				}
			}
			if len(nonShared) != 2 {
				return evidenceSteps(move, conclusion)
			}
			wingForPivot := func(value int) (solver.CandidateGroup, bool) {
				for _, wing := range wings {
					if containsValue(wing.Values, value) && containsValue(wing.Values, actionDigit) {
						return wing, true
					}
				}
				return solver.CandidateGroup{}, false
			}
			firstWing, firstOK := wingForPivot(nonShared[0])
			secondWing, secondOK := wingForPivot(nonShared[1])
			if !firstOK || !secondOK || firstWing.Position == secondWing.Position {
				return evidenceSteps(move, conclusion)
			}
			firstUnit, firstUnitOK := sharedHintUnit(pivot.Position, firstWing.Position)
			secondUnit, secondUnitOK := sharedHintUnit(pivot.Position, secondWing.Position)
			if !firstUnitOK || !secondUnitOK {
				return evidenceSteps(move, conclusion)
			}
			pivotNodes := nodesForCandidateRefs(nodes,
				solver.CandidateRef{Position: pivot.Position, Value: pivot.Values[0]},
				solver.CandidateRef{Position: pivot.Position, Value: pivot.Values[1]},
				solver.CandidateRef{Position: pivot.Position, Value: pivot.Values[2]},
			)
			firstCaseNodes := nodesForCandidateRefs(nodes,
				solver.CandidateRef{Position: pivot.Position, Value: pivot.Values[0]},
				solver.CandidateRef{Position: pivot.Position, Value: pivot.Values[1]},
				solver.CandidateRef{Position: pivot.Position, Value: pivot.Values[2]},
				solver.CandidateRef{Position: firstWing.Position, Value: nonShared[0]},
				solver.CandidateRef{Position: firstWing.Position, Value: actionDigit},
			)
			firstCaseLink := HintLink{From: candidateNodeID(solver.CandidateRef{Position: pivot.Position, Value: nonShared[0]}), To: candidateNodeID(solver.CandidateRef{Position: firstWing.Position, Value: actionDigit}), Kind: solver.LinkImplication}
			secondCaseLink := HintLink{From: candidateNodeID(solver.CandidateRef{Position: pivot.Position, Value: nonShared[1]}), To: candidateNodeID(solver.CandidateRef{Position: secondWing.Position, Value: actionDigit}), Kind: solver.LinkImplication}
			caseLinks := []HintLink{firstCaseLink, secondCaseLink}
			pivotMessage := hintDescriptor("hint.xyz-wing.pivot", cellArgument("pivot", pivot.Position), digitsArgument("digits", pivot.Values))
			firstCaseMessage := hintDescriptor("hint.xyz-wing.case", cellArgument("pivot", pivot.Position), digitArgument("pivot_digit", nonShared[0]), cellArgument("wing", firstWing.Position), unitsArgument("unit", []solver.UnitRef{firstUnit}), digitArgument("shared_digit", actionDigit))
			secondCaseMessage := hintDescriptor("hint.xyz-wing.case", cellArgument("pivot", pivot.Position), digitArgument("pivot_digit", nonShared[1]), cellArgument("wing", secondWing.Position), unitsArgument("unit", []solver.UnitRef{secondUnit}), digitArgument("shared_digit", actionDigit))
			pivotCaseMessage := hintDescriptor("hint.xyz-wing.pivot-case", cellArgument("pivot", pivot.Position), digitArgument("shared_digit", actionDigit))
			patternCells := []core.Position{pivot.Position, firstWing.Position, secondWing.Position}
			guarantee := hintDescriptor("hint.xyz-wing.guarantee", cellArgument("pivot", pivot.Position), cellsArgument("pattern_cells", patternCells), digitArgument("shared_digit", actionDigit))
			target := evidence.Eliminations[0].Position
			targetRef := solver.CandidateRef{Position: target, Value: actionDigit}
			finalNodes := append([]HintNode(nil), nodes...)
			finalNodes = append(finalNodes, HintNode{ID: candidateNodeID(targetRef), Target: candidateTarget(targetRef)})
			finalLinks := append([]HintLink(nil), caseLinks...)
			witnessUnits := make([]solver.UnitRef, 0, len(patternCells))
			for _, patternCell := range patternCells {
				unit, ok := sharedHintUnit(target, patternCell)
				if !ok {
					return evidenceSteps(move, conclusion)
				}
				witnessUnits = append(witnessUnits, unit)
				finalLinks = append(finalLinks, HintLink{From: candidateNodeID(solver.CandidateRef{Position: patternCell, Value: actionDigit}), To: candidateNodeID(targetRef), Kind: solver.LinkWeak})
			}
			messagePatternCells := make([]core.Position, 0, len(patternCells))
			messageWitnessUnits := make([]solver.UnitRef, 0, len(witnessUnits))
			for _, preferredKind := range []solver.UnitKind{solver.UnitRow, solver.UnitColumn, solver.UnitBox} {
				for index, unit := range witnessUnits {
					if unit.Kind == preferredKind {
						messagePatternCells = append(messagePatternCells, patternCells[index])
						messageWitnessUnits = append(messageWitnessUnits, unit)
					}
				}
			}
			if len(messagePatternCells) != 3 {
				return evidenceSteps(move, conclusion)
			}
			apply := hintDescriptor("hint.xyz-wing.apply",
				cellArgument("target", target),
				cellArgument("first_pattern_cell", messagePatternCells[0]), unitsArgument("first_unit", []solver.UnitRef{messageWitnessUnits[0]}),
				cellArgument("second_pattern_cell", messagePatternCells[1]), unitsArgument("second_unit", []solver.UnitRef{messageWitnessUnits[1]}),
				cellArgument("third_pattern_cell", messagePatternCells[2]), unitsArgument("third_unit", []solver.UnitRef{messageWitnessUnits[2]}),
				digitArgument("shared_digit", actionDigit), strategyArgument("strategy", move.Technique))
			return []HintStep{
				{ID: "find-xyz-wing-pivot", Kind: HintStepObserve, Message: pivotMessage, Narration: []HintNarrationChunk{narrationMessage(pivotMessage)}, Marks: marksForNodes(pivotNodes), Nodes: pivotNodes, Links: []HintLink{}},
				{ID: "follow-xyz-wing-first-case", Kind: HintStepExplain, Message: firstCaseMessage, Narration: []HintNarrationChunk{narrationMessage(firstCaseMessage)}, Marks: marksForNodes(firstCaseNodes), Nodes: firstCaseNodes, Links: []HintLink{firstCaseLink}},
				{ID: "follow-xyz-wing-second-case", Kind: HintStepExplain, Message: secondCaseMessage, Narration: []HintNarrationChunk{narrationMessage(secondCaseMessage)}, Marks: marksForNodes(nodes), Nodes: nodes, Links: caseLinks},
				{ID: "cover-xyz-wing-shared-case", Kind: HintStepExplain, Message: pivotCaseMessage, Narration: []HintNarrationChunk{narrationMessage(pivotCaseMessage)}, Marks: marksForNodes(nodes), Nodes: nodes, Links: caseLinks},
				{ID: "combine-xyz-wing-cases", Kind: HintStepExplain, Message: guarantee, Narration: []HintNarrationChunk{narrationMessage(guarantee)}, Marks: baseMarks, Nodes: nodes, Links: caseLinks},
				{ID: "apply-xyz-wing", Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(baseMarks, conclusionMarks(conclusion)...), Nodes: finalNodes, Links: finalLinks, Effect: conclusionEffect(conclusion)},
			}
		}
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
		linkUnit, ok := sharedHintUnit(linkCells[0], linkCells[1])
		if !ok {
			return evidenceSteps(move, conclusion)
		}
		observe = hintDescriptor("hint.strong-link-wing.observe", cellsArgument("wing_cells", wingCells), digitsArgument("digits", pair))
		bridge := hintDescriptor("hint.strong-link-wing.bridge", unitsArgument("unit", []solver.UnitRef{linkUnit}), digitArgument("link_digit", linkDigit), cellsArgument("link_cells", linkCells))
		force := hintDescriptor("hint.strong-link-wing.force",
			digitArgument("link_digit", linkDigit),
			cellArgument("first_link_cell", linkCells[0]),
			cellArgument("first_wing_cell", wingCells[0]),
			cellArgument("second_link_cell", linkCells[1]),
			cellArgument("second_wing_cell", wingCells[1]),
			digitArgument("forced_digit", actionDigit),
		)
		links = visualLinks(evidence)
		caseLinks := append([]HintLink(nil), links...)
		caseLinks = append(caseLinks,
			HintLink{From: candidateNodeID(solver.CandidateRef{Position: linkCells[0], Value: linkDigit}), To: candidateNodeID(solver.CandidateRef{Position: wingCells[0], Value: actionDigit}), Kind: solver.LinkImplication},
			HintLink{From: candidateNodeID(solver.CandidateRef{Position: linkCells[1], Value: linkDigit}), To: candidateNodeID(solver.CandidateRef{Position: wingCells[1], Value: actionDigit}), Kind: solver.LinkImplication},
		)
		bridgeMarks := marksForNodes(nodes)
		if target, valid := unitTarget(&linkUnit); valid {
			bridgeMarks = append(bridgeMarks, HintMark{Target: target, Role: HintRoleFocus})
		}
		for _, node := range nodes {
			if node.Target.Position != nil && (*node.Target.Position == wingCells[0] || *node.Target.Position == wingCells[1]) {
				firstNodes = append(firstNodes, node)
			}
		}
		var apply message.Descriptor
		if conclusion.Placement != nil {
			apply = hintDescriptor("hint.strong-link-wing.place", digitArgument("removed_digit", actionDigit), digitArgument("digit", conclusion.Placement.Value), cellArgument("cell", conclusion.Placement.Position), cellsArgument("wing_cells", wingCells), strategyArgument("strategy", move.Technique))
		} else {
			positions := make([]core.Position, len(evidence.Eliminations))
			for i, ref := range evidence.Eliminations {
				positions[i] = ref.Position
			}
			apply = hintDescriptor("hint.strong-link-wing.apply", digitArgument("digit", actionDigit), cellsArgument("cells", positions), cellsArgument("wing_cells", wingCells), strategyArgument("strategy", move.Technique))
		}
		return []HintStep{
			{ID: "find-w-wing-pair", Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationMessage(observe)}, Marks: marksForNodes(firstNodes), Nodes: firstNodes, Links: []HintLink{}},
			{ID: "find-w-wing-bridge", Kind: HintStepObserve, Message: bridge, Narration: []HintNarrationChunk{narrationMessage(bridge)}, Marks: bridgeMarks, Nodes: nodes, Links: links},
			{ID: "explain-w-wing-cases", Kind: HintStepExplain, Message: force, Narration: []HintNarrationChunk{narrationMessage(force)}, Marks: bridgeMarks, Nodes: nodes, Links: caseLinks},
			{ID: "apply-w-wing", Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(bridgeMarks, conclusionMarks(conclusion)...), Nodes: nodes, Links: caseLinks, Effect: conclusionEffect(conclusion)},
		}
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
	if evidence == nil || conclusion.Placement == nil || len(evidence.Chain) != 7 || len(evidence.Links) != 7 {
		return evidenceSteps(move, conclusion)
	}
	digit := conclusion.Placement.Value
	positions := make([]core.Position, len(evidence.Chain))
	for index, group := range evidence.Chain {
		if len(group.Values) == 0 {
			return evidenceSteps(move, conclusion)
		}
		positions[index] = group.Position
	}
	if positions[0] != conclusion.Placement.Position {
		return evidenceSteps(move, conclusion)
	}
	wantKinds := []solver.LinkKind{solver.LinkStrong, solver.LinkWeak, solver.LinkStrong, solver.LinkWeak, solver.LinkStrong, solver.LinkWeak, solver.LinkStrong}
	for index, link := range evidence.Links {
		wantFrom := positions[index]
		wantTo := positions[(index+1)%len(positions)]
		if link.Kind != wantKinds[index] || link.From.Position != wantFrom || link.To.Position != wantTo || link.From.Value != digit || link.To.Value != digit {
			return evidenceSteps(move, conclusion)
		}
	}
	strongUnits := make([]solver.UnitRef, 4)
	for index, linkIndex := range []int{0, 2, 4, 6} {
		unit, ok := sharedHintUnit(evidence.Links[linkIndex].From.Position, evidence.Links[linkIndex].To.Position)
		if !ok {
			return evidenceSteps(move, conclusion)
		}
		strongUnits[index] = unit
	}
	marks := premiseCandidateMarks(evidence)
	nodes := visualNodes(evidence, true)
	links := visualLinks(evidence)
	if len(nodes) != 7 {
		return evidenceSteps(move, conclusion)
	}
	scene := func(linkCount int) ([]HintNode, []HintLink) {
		selectedLinks := append([]HintLink(nil), links[:linkCount]...)
		ids := make(map[string]struct{}, linkCount+1)
		for _, link := range selectedLinks {
			ids[link.From] = struct{}{}
			ids[link.To] = struct{}{}
		}
		selectedNodes := make([]HintNode, 0, len(ids))
		for _, node := range nodes {
			if _, ok := ids[node.ID]; ok {
				selectedNodes = append(selectedNodes, node)
			}
		}
		return selectedNodes, selectedLinks
	}
	step1Nodes, step1Links := scene(1)
	step2Nodes, step2Links := scene(3)
	step3Nodes, step3Links := scene(5)
	opening := hintDescriptor("hint.x-cycle.opening", unitsArgument("unit", strongUnits[:1]), digitArgument("digit", digit), cellsArgument("cells", positions[:2]))
	assume := hintDescriptor("hint.x-cycle.assume", cellArgument("start", positions[0]), cellArgument("forced", positions[1]), cellArgument("blocked", positions[2]), unitsArgument("unit", strongUnits[1:2]), cellArgument("result", positions[3]), digitArgument("digit", digit))
	follow := hintDescriptor("hint.x-cycle.follow", cellArgument("true_cell", positions[3]), cellArgument("blocked", positions[4]), unitsArgument("unit", strongUnits[2:3]), cellArgument("result", positions[5]), digitArgument("digit", digit))
	closeLoop := hintDescriptor("hint.x-cycle.close", cellArgument("true_cell", positions[5]), cellArgument("blocked", positions[6]), unitsArgument("unit", strongUnits[3:4]), cellArgument("result", positions[0]), digitArgument("digit", digit))
	apply := hintDescriptor("hint.x-cycle.place", cellArgument("assumption", positions[0]), digitArgument("digit", digit), cellArgument("cell", conclusion.Placement.Position), strategyArgument("strategy", move.Technique))
	return []HintStep{
		{ID: "find-x-cycle-opening-pair", Kind: HintStepObserve, Message: opening, Narration: []HintNarrationChunk{narrationMessage(opening)}, Marks: marksForNodes(step1Nodes), Nodes: step1Nodes, Links: step1Links},
		{ID: "follow-x-cycle-to-row-eight", Kind: HintStepExplain, Message: assume, Narration: []HintNarrationChunk{narrationMessage(assume)}, Marks: marksForNodes(step2Nodes), Nodes: step2Nodes, Links: step2Links},
		{ID: "follow-x-cycle-to-row-two", Kind: HintStepExplain, Message: follow, Narration: []HintNarrationChunk{narrationMessage(follow)}, Marks: marksForNodes(step3Nodes), Nodes: step3Nodes, Links: step3Links},
		{ID: "close-x-cycle-loop", Kind: HintStepExplain, Message: closeLoop, Narration: []HintNarrationChunk{narrationMessage(closeLoop)}, Marks: marks, Nodes: nodes, Links: links},
		{ID: "apply-x-cycle", Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(marks, conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)},
	}
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

func otherHintDigit(values []int, excluded int) int {
	for _, value := range values {
		if value != excluded {
			return value
		}
	}
	return 0
}
