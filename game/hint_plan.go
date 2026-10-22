package game

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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
	HintReasonSubsetReservation HintReasonKind = "subset-reservation"
	HintReasonFishConfinement   HintReasonKind = "fish-confinement"
	HintReasonImplicationChain  HintReasonKind = "implication-chain"
)

type SubsetReservationReason struct {
	Unit           solver.UnitRef
	ReservedDigits []int
	PremiseCells   []core.Position
	ActionTargets  []solver.CandidateRef
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
	Kind              HintReasonKind
	SubsetReservation *SubsetReservationReason
	FishConfinement   *FishConfinementReason
	ImplicationChain  *ImplicationChainReason
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
	for _, present := range []bool{reason.SubsetReservation != nil, reason.FishConfinement != nil, reason.ImplicationChain != nil} {
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

func compactCellName(position core.Position) string {
	return fmt.Sprintf("r%dc%d", position.Row+1, position.Column+1)
}

func candidateTarget(ref solver.CandidateRef) HintTarget {
	position := ref.Position
	return HintTarget{Kind: HintTargetCandidate, Position: &position, Value: ref.Value}
}

func cellTarget(position core.Position) HintTarget {
	copy := position
	return HintTarget{Kind: HintTargetCell, Position: &copy}
}

func unitTarget(unit *solver.UnitRef) (HintTarget, bool) {
	if unit == nil {
		return HintTarget{}, false
	}
	kind := map[solver.UnitKind]HintTargetKind{
		solver.UnitRow: HintTargetRow, solver.UnitColumn: HintTargetColumn, solver.UnitBox: HintTargetBox,
	}[unit.Kind]
	if kind == "" {
		return HintTarget{}, false
	}
	return HintTarget{Kind: kind, Index: unit.Index}, true
}

func narrationMessage(descriptor message.Descriptor) HintNarrationChunk {
	copy := descriptor
	return HintNarrationChunk{Kind: HintNarrationMessage, Message: &copy}
}

func narrationTarget(target HintTarget) HintNarrationChunk {
	copy := target
	return HintNarrationChunk{Kind: HintNarrationTarget, Target: &copy}
}

func narrationDigit(value int) HintNarrationChunk {
	return HintNarrationChunk{Kind: HintNarrationDigit, Value: value}
}

var canonicalEnglishMessages = func() *message.Catalog {
	catalog, err := message.NewEnglishCatalog()
	if err != nil {
		panic(err)
	}
	return catalog
}()

func hintDescriptor(id string, arguments ...message.Argument) message.Descriptor {
	descriptor, err := canonicalEnglishMessages.Render(id, arguments...)
	if err != nil {
		panic(err)
	}
	return descriptor
}

func cellArgument(name string, position core.Position) message.Argument {
	cell := message.Cell{Row: position.Row + 1, Column: position.Column + 1}
	return message.Argument{Name: name, Kind: message.ArgumentCell, Cell: &cell}
}

func strategyArgument(name, strategy string) message.Argument {
	return message.Argument{Name: name, Kind: message.ArgumentStrategy, Strategy: &strategy}
}

func countArgument(name string, count int) message.Argument {
	return message.Argument{Name: name, Kind: message.ArgumentCount, Count: &count}
}

func digitArgument(name string, digit int) message.Argument {
	return message.Argument{Name: name, Kind: message.ArgumentDigit, Digit: &digit}
}

func candidateArgument(name string, ref solver.CandidateRef) message.Argument {
	candidate := message.Candidate{Cell: message.Cell{Row: ref.Position.Row + 1, Column: ref.Position.Column + 1}, Digit: ref.Value}
	return message.Argument{Name: name, Kind: message.ArgumentCandidate, Candidate: &candidate}
}

func digitsArgument(name string, values []int) message.Argument {
	return message.Argument{Name: name, Kind: message.ArgumentDigits, Digits: append([]int(nil), values...)}
}

func cellsArgument(name string, positions []core.Position) message.Argument {
	values := make([]message.Cell, len(positions))
	for i, position := range positions {
		values[i] = message.Cell{Row: position.Row + 1, Column: position.Column + 1}
	}
	return message.Argument{Name: name, Kind: message.ArgumentCells, Cells: values}
}

func candidatesArgument(name string, refs []solver.CandidateRef) message.Argument {
	values := make([]message.Candidate, len(refs))
	for i, ref := range refs {
		values[i] = message.Candidate{Cell: message.Cell{Row: ref.Position.Row + 1, Column: ref.Position.Column + 1}, Digit: ref.Value}
	}
	return message.Argument{Name: name, Kind: message.ArgumentCandidates, Candidates: values}
}

func unitsArgument(name string, refs []solver.UnitRef) message.Argument {
	values := make([]message.Unit, len(refs))
	for i, ref := range refs {
		values[i] = message.Unit{Kind: string(ref.Kind), Index: ref.Index + 1}
	}
	return message.Argument{Name: name, Kind: message.ArgumentUnits, Units: values}
}

func composeValueCorrectionPlan(snapshot Snapshot, position core.Position) *HintPlan {
	positionCopy := position
	conclusion := HintConclusion{ClearValue: &positionCopy}
	observe := hintDescriptor("hint.value-correction.observe", cellArgument("cell", position))
	conclude := hintDescriptor("hint.value-correction.conclude", cellArgument("cell", position))
	plan := &HintPlan{
		Kind:    HintKindValueCorrection,
		Title:   hintDescriptor("hint.value-correction.title"),
		Summary: hintDescriptor("hint.value-correction.summary", cellArgument("cell", position)),
		Steps: []HintStep{
			{ID: "find-wrong-value", Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationID("hint.narration.look-at"), narrationTarget(cellTarget(position)), narrationID("hint.narration.value-conflict")}, Marks: []HintMark{{Target: cellTarget(position), Role: HintRoleProblem}}, Nodes: []HintNode{}, Links: []HintLink{}},
			{ID: "clear-wrong-value", Kind: HintStepApply, Message: conclude, Narration: []HintNarrationChunk{narrationID("hint.narration.clear-value"), narrationTarget(cellTarget(position)), narrationID("hint.narration.clear-value-tail")}, Marks: []HintMark{{Target: cellTarget(position), Role: HintRoleConclusion}}, Nodes: []HintNode{}, Links: []HintLink{}, Effect: &HintEffect{ClearValue: &positionCopy}},
		},
		Conclusion: conclusion,
	}
	plan.PlanID = deterministicPlanID(snapshot, *plan)
	return plan
}

func composeCandidateCorrectionPlan(snapshot Snapshot, position core.Position, current, legal core.CandidateSet) *HintPlan {
	replacement := &CandidateReplacement{Position: position, Values: legal}
	var problemMarks, correctionMarks []HintMark
	for _, value := range (current &^ legal).Values() {
		problemMarks = append(problemMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: position, Value: value}), Role: HintRoleExtra})
	}
	for _, value := range legal.Values() {
		correctionMarks = append(correctionMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: position, Value: value}), Role: HintRoleConclusion})
	}
	observe := hintDescriptor("hint.candidate-correction.observe", cellArgument("cell", position))
	compare := hintDescriptor("hint.candidate-correction.compare")
	conclude := hintDescriptor("hint.candidate-correction.conclude", cellArgument("cell", position))
	plan := &HintPlan{
		Kind:    HintKindCandidateCorrection,
		Title:   hintDescriptor("hint.candidate-correction.title"),
		Summary: hintDescriptor("hint.candidate-correction.summary", cellArgument("cell", position)),
		Steps: []HintStep{
			{ID: "find-candidate-problem", Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationID("hint.narration.look-at"), narrationTarget(cellTarget(position)), narrationID("hint.narration.candidate-inconsistent")}, Marks: append([]HintMark{{Target: cellTarget(position), Role: HintRoleProblem}}, problemMarks...), Nodes: []HintNode{}, Links: []HintLink{}},
			{ID: "compare-candidate-cell", Kind: HintStepExplain, Message: compare, Narration: []HintNarrationChunk{narrationMessage(compare)}, Marks: problemMarks, Nodes: []HintNode{}, Links: []HintLink{}},
			{ID: "set-candidate-cell", Kind: HintStepApply, Message: conclude, Narration: []HintNarrationChunk{narrationID("hint.narration.reset-candidates"), narrationTarget(cellTarget(position)), narrationID("hint.narration.reset-candidates-tail")}, Marks: correctionMarks, Nodes: []HintNode{}, Links: []HintLink{}, Effect: &HintEffect{SetCandidates: replacement}},
		},
		Conclusion: HintConclusion{SetCandidates: replacement},
	}
	plan.PlanID = deterministicPlanID(snapshot, *plan)
	return plan
}

func composeHintPlan(snapshot Snapshot, strategy solver.Solver, move *solver.Move) *HintPlan {
	if move == nil {
		return nil
	}
	grade, _, _ := solver.StrategyTierForTechnique(move.Technique)
	if grade == "" {
		grade = "evil"
	}
	display := move.Technique
	if strategy != nil {
		display = strategy.GetDisplayName()
	}
	name := hintDescriptor("strategy." + move.Technique + ".name")
	gradeName := hintDescriptor("difficulty." + grade + ".name")
	plan := &HintPlan{
		Kind:     HintKindStrategy,
		Title:    name,
		Strategy: HintStrategy{ID: move.Technique, Name: name, Grade: grade, GradeName: gradeName},
	}
	if move.IsPlacement() {
		cell := move.Cell
		plan.Conclusion.Placement = &cell
	} else if move.Evidence != nil {
		plan.Conclusion.Eliminations = append([]solver.CandidateRef(nil), move.Evidence.Eliminations...)
	}

	switch move.Technique {
	case "naked-single":
		plan.Steps = nakedSingleSteps(move)
	case "hidden-single":
		plan.Steps = hiddenSingleSteps(move)
	case "naked-pair", "naked-triple", "naked-quad":
		plan.Steps = nakedSubsetSteps(move, display)
	case "hidden-pair", "hidden-triple", "hidden-quad":
		plan.Steps = hiddenSubsetSteps(move, display)
	case "pointing-pair":
		plan.Steps = intersectionSteps(move)
	case "x-wing", "swordfish", "jellyfish":
		plan.Steps = fishSteps(move, display)
	case "xy-wing", "w-wing", "xyz-wing":
		plan.Steps = relationalSteps(move, display)
	case "bug-plus-one":
		plan.Steps = bugPlusOneSteps(move, display)
	case "unique-rectangle", "unique-rectangle-2", "unique-rectangle-3", "unique-rectangle-4":
		plan.Steps = uniquenessSteps(move, display)
	case "simple-coloring":
		plan.Steps = coloringSteps(move, display)
	case "x-cycles":
		plan.Steps = xCycleSteps(move, display)
	case "xy-chain":
		plan.Steps = xyChainSteps(move, display)
	default:
		plan.Steps = evidenceSteps(move, plan.Conclusion)
	}
	plan.Reason = familyHintReason(move)
	for index := range plan.Steps {
		populateStrategyStep(&plan.Steps[index], index, len(plan.Steps), move.Technique)
		if plan.Steps[index].Marks == nil {
			plan.Steps[index].Marks = []HintMark{}
		}
		if plan.Steps[index].Nodes == nil {
			plan.Steps[index].Nodes = []HintNode{}
		}
		if plan.Steps[index].Links == nil {
			plan.Steps[index].Links = []HintLink{}
		}
	}
	plan.Summary = hintDescriptor("hint.strategy.summary", countArgument("steps", len(plan.Steps)), strategyArgument("strategy", move.Technique))
	plan.PlanID = deterministicPlanID(snapshot, *plan)
	return plan
}

// populateStrategyStep attaches catalog-backed wording after the composer has
// built the renderer-neutral scene and effect.
func populateStrategyStep(step *HintStep, index, total int, strategy string) {
	if step.Message.ID != "" {
		if !step.HasValidNarration() {
			step.Narration = []HintNarrationChunk{narrationMessage(step.Message)}
		}
		return
	}
	id := "hint.strategy.compare"
	if index == 0 {
		id = "hint.strategy.observe"
	}
	if index == total-1 {
		id = "hint.strategy.conclude"
		step.Message = hintDescriptor(id, strategyArgument("strategy", strategy))
		if !step.HasValidNarration() {
			conclusion := HintConclusion{}
			if step.Effect != nil {
				conclusion.Placement = step.Effect.Placement
				conclusion.Eliminations = step.Effect.Eliminations
			}
			step.Narration = []HintNarrationChunk{narrationID("hint.narration.apply")}
			step.Narration = append(step.Narration, conclusionNarration(conclusion)...)
			step.Narration = append(step.Narration, narrationMessage(hintDescriptor("hint.narration.strategy-result", strategyArgument("strategy", strategy))))
		}
		return
	}
	step.Message = hintDescriptor(id)
	if !step.HasValidNarration() {
		step.Narration = []HintNarrationChunk{narrationMessage(step.Message)}
	}
}

func narrationID(id string) HintNarrationChunk { return narrationMessage(hintDescriptor(id)) }

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
	case "x-wing":
		if len(evidence.Units) != 4 || len(evidence.Premises) == 0 || len(evidence.Eliminations) == 0 {
			return nil
		}
		digit := evidence.Premises[0].Values[0]
		patterns := make([]solver.CandidateRef, 0, len(evidence.Premises))
		for _, premise := range evidence.Premises {
			patterns = append(patterns, solver.CandidateRef{Position: premise.Position, Value: digit})
		}
		return &HintReason{Kind: HintReasonFishConfinement, FishConfinement: &FishConfinementReason{
			Digit: digit, BaseUnits: append([]solver.UnitRef(nil), evidence.Units[:2]...), CoverUnits: append([]solver.UnitRef(nil), evidence.Units[2:]...),
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
	for _, premise := range evidence.Premises {
		for _, value := range premise.Values {
			premiseMarks = append(premiseMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: value}), Role: HintRolePremise})
		}
	}
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	return []HintStep{
		{ID: "find-confined-" + move.Technique, Kind: HintStepObserve, Marks: premiseMarks},
		{ID: "compare-confined-" + move.Technique, Kind: HintStepExplain, Marks: premiseMarks},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Marks: append(premiseMarks[1:], conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func intersectionSteps(move *solver.Move) []HintStep {
	evidence := move.Evidence
	var premiseMarks []HintMark
	for _, unitRef := range evidence.Units {
		unit := unitRef
		target, _ := unitTarget(&unit)
		premiseMarks = append(premiseMarks, HintMark{Target: target, Role: HintRoleFocus})
	}
	for _, premise := range evidence.Premises {
		for _, candidate := range premise.Values {
			premiseMarks = append(premiseMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: candidate}), Role: HintRolePremise})
		}
	}
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	return []HintStep{
		{ID: "find-intersection", Kind: HintStepObserve, Marks: premiseMarks},
		{ID: "compare-intersection", Kind: HintStepExplain, Marks: premiseMarks},
		{ID: "apply-intersection", Kind: conclusionKind(conclusion), Marks: append(premiseMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func fishSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	marks := unitAndPremiseMarks(evidence)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	if move.Technique == "x-wing" && len(evidence.Units) == 4 && len(evidence.Premises) > 0 && len(evidence.Eliminations) > 0 {
		baseUnits, coverUnits := evidence.Units[:2], evidence.Units[2:]
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
			{ID: "find-x-wing", Kind: HintStepObserve, Message: observe, Narration: []HintNarrationChunk{narrationMessage(observe)}, Marks: baseMarks},
			{ID: "explain-x-wing", Kind: HintStepExplain, Message: explain, Narration: []HintNarrationChunk{narrationMessage(explain)}, Marks: explainMarks},
			{ID: "apply-x-wing", Kind: conclusionKind(conclusion), Message: apply, Narration: actionNarration(apply), Marks: append(explainMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
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
	marks := unitAndPremiseMarks(evidence)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	var firstNodes, nodes []HintNode
	var firstLinks, links []HintLink
	switch move.Technique {
	case "xy-wing":
		if len(evidence.Links) == 2 {
			nodes = visualNodes(evidence, false)
			links = visualLinks(evidence)
			firstNodes, firstLinks = visualPrefix(nodes, links)
		}
	case "w-wing":
		if len(evidence.Links) == 1 {
			nodes = visualNodes(evidence, false)
			links = visualLinks(evidence)
			firstNodes, firstLinks = nodes, links
		}
	case "xyz-wing":
	}
	observeMarks := marks
	if len(firstNodes) > 0 {
		observeMarks = marksForNodes(firstNodes)
	}
	return []HintStep{
		{ID: "find-" + move.Technique, Kind: HintStepObserve, Marks: observeMarks, Nodes: firstNodes, Links: firstLinks},
		{ID: "compare-" + move.Technique, Kind: HintStepExplain, Marks: marks, Nodes: nodes, Links: links},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Marks: append(marks, conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)},
	}
}

func bugPlusOneSteps(move *solver.Move, _ string) []HintStep {
	if move.Evidence == nil || len(move.Evidence.Premises) == 0 || !move.IsPlacement() {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	marks := unitAndPremiseMarks(move.Evidence)
	conclusion := HintConclusion{Placement: placement(move)}
	return []HintStep{
		{ID: "find-trivalue-cell", Kind: HintStepObserve, Marks: marks},
		{ID: "compare-extra-candidate", Kind: HintStepExplain, Marks: marks},
		{ID: "apply-bug-plus-one", Kind: HintStepApply, Marks: append(marks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func uniquenessSteps(move *solver.Move, _ string) []HintStep {
	evidence := move.Evidence
	if evidence == nil {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	if len(evidence.Premises) < 4 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations})
	}
	premises := evidence.Premises
	if len(sharedValues(premises[:4]...)) < 2 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations})
	}
	marks := unitAndPremiseMarks(evidence)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	return []HintStep{
		{ID: "find-" + move.Technique, Kind: HintStepObserve, Marks: marks},
		{ID: "compare-" + move.Technique, Kind: HintStepExplain, Marks: marks},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Marks: append(marks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
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

func deterministicPlanID(snapshot Snapshot, plan HintPlan) string {
	plan.PlanID = ""
	stripFallback := func(descriptor *message.Descriptor) { descriptor.Text = "" }
	stripFallback(&plan.Title)
	stripFallback(&plan.Summary)
	stripFallback(&plan.Strategy.Name)
	stripFallback(&plan.Strategy.GradeName)
	plan.Steps = append([]HintStep(nil), plan.Steps...)
	for index := range plan.Steps {
		stripFallback(&plan.Steps[index].Message)
		plan.Steps[index].Narration = append([]HintNarrationChunk(nil), plan.Steps[index].Narration...)
		for chunkIndex := range plan.Steps[index].Narration {
			if plan.Steps[index].Narration[chunkIndex].Message != nil {
				descriptor := *plan.Steps[index].Narration[chunkIndex].Message
				stripFallback(&descriptor)
				plan.Steps[index].Narration[chunkIndex].Message = &descriptor
			}
		}
	}
	payload, err := json.Marshal(struct {
		Snapshot Snapshot
		Plan     HintPlan
	}{snapshot, plan})
	if err != nil {
		panic(err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}
