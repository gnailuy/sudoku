package game

import (
	"fmt"
	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/message"
	"github.com/gnailuy/sudoku/solver"
	"sort"
	"strings"
)

func actionNarration(descriptor message.Descriptor) []HintNarrationChunk {
	return []HintNarrationChunk{narrationMessage(descriptor)}
}

func evidenceSteps(_ *solver.Move, _ HintConclusion) []HintStep {
	return nil
}

func backtrackerSteps(move *solver.Move) []HintStep {
	placement := solver.CandidateRef{Position: move.Cell.Position, Value: move.Cell.Value}
	observe := hintDescriptor("hint.backtracker.observe")
	apply := hintDescriptor("hint.backtracker.apply", cellArgument("cell", move.Cell.Position), digitArgument("digit", move.Cell.Value))
	return []HintStep{
		{ID: "explain-complete-search", Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationMessage(observe)}, Marks: []HintMark{}},
		{ID: "apply-complete-search", Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: []HintMark{{Target: candidateTarget(placement), Role: HintRoleConclusion}}, Effect: &HintEffect{Placement: &move.Cell}},
	}
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
	only := hintDescriptor("hint.hidden-single.only", cellArgument("cell", move.Cell.Position), digitArgument("digit", move.Cell.Value))
	apply := hintDescriptor("hint.hidden-single.apply", digitArgument("digit", move.Cell.Value), cellArgument("cell", move.Cell.Position), strategyArgument("strategy", move.Technique))
	steps := []HintStep{
		{ID: fmt.Sprintf("scan-%s-for-%d", strings.ReplaceAll(unitName, " ", ""), move.Cell.Value), Kind: HintStepObserve, Message: inspect, Narration: []HintNarrationChunk{narrationMessage(inspect)}, Marks: baseMarks},
	}
	if len(blockerCells) > 0 {
		blockers := hintDescriptor("hint.hidden-single.blockers", cellsArgument("blockers", blockerCells), digitArgument("digit", move.Cell.Value), unitsArgument("unit", []solver.UnitRef{*move.Evidence.Unit}))
		steps = append(steps, HintStep{ID: fmt.Sprintf("rule-out-%s-%d", strings.ReplaceAll(unitName, " ", ""), move.Cell.Value), Kind: HintStepExplain, Message: blockers, Narration: []HintNarrationChunk{narrationMessage(blockers)}, Marks: blockerMarks})
	}
	steps = append(steps,
		HintStep{ID: fmt.Sprintf("identify-%s-%d", compactCellName(move.Cell.Position), move.Cell.Value), Kind: HintStepExplain, Message: only, Narration: []HintNarrationChunk{narrationMessage(only)}, Marks: blockerMarks},
		HintStep{ID: fmt.Sprintf("place-%s-%d", compactCellName(move.Cell.Position), move.Cell.Value), Kind: HintStepApply, Message: apply, Narration: actionNarration(apply), Marks: applyMarks, Effect: &HintEffect{Placement: &move.Cell}},
	)
	return steps
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
