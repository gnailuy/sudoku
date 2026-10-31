package game

import (
	"github.com/gnailuy/sudoku/core"
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
