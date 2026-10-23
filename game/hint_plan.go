package game

import (
	"fmt"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/message"
	"github.com/gnailuy/sudoku/solver"
)

type HintStepKind string

const (
	HintStepObserve HintStepKind = "observe"
	HintStepExplain HintStepKind = "explain"
	HintStepApply   HintStepKind = "apply"
)

type HintRole string

const (
	HintRoleFocus      HintRole = "focus"
	HintRolePremise    HintRole = "premise"
	HintRoleEliminated HintRole = "eliminated"
	HintRoleConclusion HintRole = "conclusion"
	HintRoleProblem    HintRole = "problem"
	HintRoleExtra      HintRole = "extra"
)

type HintTargetKind string

const (
	HintTargetCell      HintTargetKind = "cell"
	HintTargetCandidate HintTargetKind = "candidate"
	HintTargetRow       HintTargetKind = "row"
	HintTargetColumn    HintTargetKind = "column"
	HintTargetBox       HintTargetKind = "box"
)

type HintTarget struct {
	Kind     HintTargetKind
	Position *core.Position
	Value    int
	Index    int
}

type HintNarrationKind string

const (
	HintNarrationMessage HintNarrationKind = "message"
	HintNarrationTarget  HintNarrationKind = "target"
	HintNarrationDigit   HintNarrationKind = "digit"
)

// HintNarrationChunk carries renderer-neutral language without requiring a
// client to parse fallback prose. Exactly one payload matches Kind.
type HintNarrationChunk struct {
	Kind    HintNarrationKind
	Message *message.Descriptor
	Target  *HintTarget
	Value   int
}

// HasValidNarration reports whether every chunk is one well-formed variant.
// Renderers use the complete fallback message when this returns false.
func (step HintStep) HasValidNarration() bool {
	if len(step.Narration) == 0 {
		return false
	}
	for _, chunk := range step.Narration {
		switch chunk.Kind {
		case HintNarrationMessage:
			if chunk.Message == nil || chunk.Message.Validate() != nil || chunk.Target != nil || chunk.Value != 0 {
				return false
			}
		case HintNarrationTarget:
			if chunk.Message != nil || chunk.Target == nil || chunk.Value != 0 || !validHintNarrationTarget(*chunk.Target) {
				return false
			}
		case HintNarrationDigit:
			if chunk.Message != nil || chunk.Target != nil || chunk.Value < 1 || chunk.Value > 9 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func validHintNarrationTarget(target HintTarget) bool {
	switch target.Kind {
	case HintTargetCell:
		return target.Position != nil && target.Position.IsValid() && target.Value == 0 && target.Index == 0
	case HintTargetCandidate:
		return target.Position != nil && target.Position.IsValid() && target.Value >= 1 && target.Value <= 9 && target.Index == 0
	case HintTargetRow, HintTargetColumn, HintTargetBox:
		return target.Position == nil && target.Value == 0 && target.Index >= 0 && target.Index < 9
	default:
		return false
	}
}

type HintMark struct {
	Target HintTarget
	Role   HintRole
}

// HintNode gives a candidate a stable visual identity. Group and Order are
// optional semantic labels; renderers must not rely on color alone for either.
type HintNode struct {
	ID     string
	Target HintTarget
	Group  string
	Order  int
}

type HintLink struct {
	From string
	To   string
	Kind solver.LinkKind
}

type HintEffect struct {
	Placement     *core.Cell
	Eliminations  []solver.CandidateRef
	ClearValue    *core.Position
	SetCandidates *CandidateReplacement
}

type HintStep struct {
	ID        string
	Kind      HintStepKind
	Message   message.Descriptor
	Narration []HintNarrationChunk
	Marks     []HintMark
	Nodes     []HintNode
	Links     []HintLink
	Effect    *HintEffect
}

type HintStrategy struct {
	ID        string
	Name      message.Descriptor
	Grade     string
	GradeName message.Descriptor
}

type CandidateReplacement struct {
	Position core.Position
	Values   core.CandidateSet
}

type HintConclusion struct {
	Placement     *core.Cell
	Eliminations  []solver.CandidateRef
	ClearValue    *core.Position
	SetCandidates *CandidateReplacement
}

type HintReasonKind string

const (
	HintReasonSubsetReservation       HintReasonKind = "subset-reservation"
	HintReasonHiddenSubsetConfinement HintReasonKind = "hidden-subset-confinement"
	HintReasonIntersectionConfinement HintReasonKind = "intersection-confinement"
	HintReasonFishConfinement         HintReasonKind = "fish-confinement"
	HintReasonImplicationChain        HintReasonKind = "implication-chain"
)

type SubsetReservationReason struct {
	Unit           solver.UnitRef
	ReservedDigits []int
	PremiseCells   []core.Position
	ActionTargets  []solver.CandidateRef
}

type HiddenSubsetConfinementReason struct {
	Unit           solver.UnitRef
	ConfinedDigits []int
	PremiseCells   []core.Position
	ActionTargets  []solver.CandidateRef
}

type IntersectionConfinementReason struct {
	Digit             int
	SourceUnit        solver.UnitRef
	IntersectingUnit  solver.UnitRef
	PremiseCandidates []solver.CandidateRef
	ActionTargets     []solver.CandidateRef
}

type FishConfinementReason struct {
	Digit             int
	BaseUnits         []solver.UnitRef
	CoverUnits        []solver.UnitRef
	PatternCandidates []solver.CandidateRef
	ActionTargets     []solver.CandidateRef
}

type ImplicationChainReason struct {
	BivalueGroups []solver.CandidateGroup
	Links         []solver.CandidateLink
	EndpointDigit int
	ActionTargets []solver.CandidateRef
}

type HintReason struct {
	Kind                    HintReasonKind
	SubsetReservation       *SubsetReservationReason
	HiddenSubsetConfinement *HiddenSubsetConfinementReason
	IntersectionConfinement *IntersectionConfinementReason
	FishConfinement         *FishConfinementReason
	ImplicationChain        *ImplicationChainReason
}

// Validate rejects incomplete or mismatched family reasons before transport.
func (reason HintReason) Validate() error {
	validUnit := func(unit solver.UnitRef) bool {
		return (unit.Kind == solver.UnitRow || unit.Kind == solver.UnitColumn || unit.Kind == solver.UnitBox) && unit.Index >= 0 && unit.Index < 9
	}
	validRefs := func(refs []solver.CandidateRef) bool {
		if len(refs) == 0 || len(refs) > 64 {
			return false
		}
		for _, ref := range refs {
			if !ref.Position.IsValid() || ref.Value < 1 || ref.Value > 9 {
				return false
			}
		}
		return true
	}
	payloads := 0
	for _, present := range []bool{reason.SubsetReservation != nil, reason.HiddenSubsetConfinement != nil, reason.IntersectionConfinement != nil, reason.FishConfinement != nil, reason.ImplicationChain != nil} {
		if present {
			payloads++
		}
	}
	if payloads != 1 {
		return fmt.Errorf("hint reason must carry exactly one family payload")
	}
	switch reason.Kind {
	case HintReasonSubsetReservation:
		value := reason.SubsetReservation
		if value == nil || !validUnit(value.Unit) || len(value.ReservedDigits) < 2 || len(value.ReservedDigits) > 4 || len(value.PremiseCells) != len(value.ReservedDigits) || !validRefs(value.ActionTargets) {
			return fmt.Errorf("invalid subset-reservation reason")
		}
		seenDigits := make(map[int]struct{}, len(value.ReservedDigits))
		for _, digit := range value.ReservedDigits {
			if digit < 1 || digit > 9 {
				return fmt.Errorf("invalid subset reserved digit")
			}
			seenDigits[digit] = struct{}{}
		}
		if len(seenDigits) != len(value.ReservedDigits) {
			return fmt.Errorf("duplicate subset reserved digit")
		}
		seenCells := make(map[core.Position]struct{}, len(value.PremiseCells))
		for _, position := range value.PremiseCells {
			if !position.IsValid() {
				return fmt.Errorf("invalid subset premise cell")
			}
			seenCells[position] = struct{}{}
		}
		if len(seenCells) != len(value.PremiseCells) {
			return fmt.Errorf("duplicate subset premise cell")
		}
	case HintReasonHiddenSubsetConfinement:
		value := reason.HiddenSubsetConfinement
		if value == nil || !validUnit(value.Unit) || len(value.ConfinedDigits) < 2 || len(value.ConfinedDigits) > 4 || len(value.PremiseCells) != len(value.ConfinedDigits) || !validRefs(value.ActionTargets) {
			return fmt.Errorf("invalid hidden-subset-confinement reason")
		}
		seenDigits := make(map[int]struct{}, len(value.ConfinedDigits))
		for _, digit := range value.ConfinedDigits {
			if digit < 1 || digit > 9 {
				return fmt.Errorf("invalid hidden-subset confined digit")
			}
			seenDigits[digit] = struct{}{}
		}
		if len(seenDigits) != len(value.ConfinedDigits) {
			return fmt.Errorf("duplicate hidden-subset confined digit")
		}
		seenCells := make(map[core.Position]struct{}, len(value.PremiseCells))
		for _, position := range value.PremiseCells {
			if !position.IsValid() {
				return fmt.Errorf("invalid hidden-subset premise cell")
			}
			seenCells[position] = struct{}{}
		}
		if len(seenCells) != len(value.PremiseCells) {
			return fmt.Errorf("duplicate hidden-subset premise cell")
		}
	case HintReasonIntersectionConfinement:
		value := reason.IntersectionConfinement
		if value == nil || value.Digit < 1 || value.Digit > 9 || !validUnit(value.SourceUnit) || !validUnit(value.IntersectingUnit) || value.SourceUnit == value.IntersectingUnit || len(value.PremiseCandidates) < 2 || !validRefs(value.PremiseCandidates) || !validRefs(value.ActionTargets) {
			return fmt.Errorf("invalid intersection-confinement reason")
		}
		for _, ref := range value.PremiseCandidates {
			if ref.Value != value.Digit {
				return fmt.Errorf("intersection premise digit does not match reason")
			}
		}
	case HintReasonFishConfinement:
		value := reason.FishConfinement
		if value == nil || value.Digit < 1 || value.Digit > 9 || len(value.BaseUnits) < 2 || len(value.CoverUnits) != len(value.BaseUnits) || !validRefs(value.PatternCandidates) || !validRefs(value.ActionTargets) {
			return fmt.Errorf("invalid fish-confinement reason")
		}
		for _, unit := range append(append([]solver.UnitRef(nil), value.BaseUnits...), value.CoverUnits...) {
			if !validUnit(unit) {
				return fmt.Errorf("invalid fish unit")
			}
		}
	case HintReasonImplicationChain:
		value := reason.ImplicationChain
		if value == nil || len(value.BivalueGroups) < 2 || len(value.Links) != len(value.BivalueGroups)-1 || value.EndpointDigit < 1 || value.EndpointDigit > 9 || !validRefs(value.ActionTargets) {
			return fmt.Errorf("invalid implication-chain reason")
		}
		for _, group := range value.BivalueGroups {
			if !group.Position.IsValid() || len(group.Values) != 2 {
				return fmt.Errorf("invalid chain bivalue group")
			}
		}
	default:
		return fmt.Errorf("unknown hint reason kind %q", reason.Kind)
	}
	return nil
}

type HintKind string

const (
	HintKindValueCorrection     HintKind = "value-correction"
	HintKindCandidateCorrection HintKind = "candidate-correction"
	HintKindStrategy            HintKind = "strategy"
)

type HintPlan struct {
	PlanID     string
	Kind       HintKind
	Title      message.Descriptor
	Strategy   HintStrategy
	Summary    message.Descriptor
	Reason     *HintReason
	Steps      []HintStep
	Conclusion HintConclusion
}

// String returns the complete plain-text fallback for limited renderers.
func (plan HintPlan) String() string { return plan.Summary.Text }
