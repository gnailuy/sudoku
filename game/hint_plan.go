package game

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

type HintStepKind string

const (
	HintStepObserve   HintStepKind = "observe"
	HintStepCompare   HintStepKind = "compare"
	HintStepEliminate HintStepKind = "eliminate"
	HintStepConclude  HintStepKind = "conclude"
)

type HintRole string

const (
	HintRoleFocus      HintRole = "focus"
	HintRolePremise    HintRole = "premise"
	HintRoleEliminated HintRole = "eliminated"
	HintRoleConclusion HintRole = "conclusion"
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
	HintNarrationText   HintNarrationKind = "text"
	HintNarrationTarget HintNarrationKind = "target"
	HintNarrationDigit  HintNarrationKind = "digit"
)

// HintNarrationChunk carries renderer-neutral language without requiring a
// client to parse fallback prose. Exactly one payload matches Kind.
type HintNarrationChunk struct {
	Kind   HintNarrationKind
	Text   string
	Target *HintTarget
	Value  int
}

// HasValidNarration reports whether every chunk is one well-formed variant.
// Renderers use the complete fallback message when this returns false.
func (step HintStep) HasValidNarration() bool {
	if len(step.Narration) == 0 {
		return false
	}
	for _, chunk := range step.Narration {
		switch chunk.Kind {
		case HintNarrationText:
			if chunk.Text == "" || chunk.Target != nil || chunk.Value != 0 {
				return false
			}
		case HintNarrationTarget:
			if chunk.Text != "" || chunk.Target == nil || chunk.Value != 0 || !validHintNarrationTarget(*chunk.Target) {
				return false
			}
		case HintNarrationDigit:
			if chunk.Text != "" || chunk.Target != nil || chunk.Value < 1 || chunk.Value > 9 {
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
	Placement    *core.Cell
	Eliminations []solver.CandidateRef
}

type HintStep struct {
	ID        string
	Kind      HintStepKind
	Message   string
	Narration []HintNarrationChunk
	Marks     []HintMark
	Nodes     []HintNode
	Links     []HintLink
	Effect    *HintEffect
}

type HintStrategy struct {
	ID          string
	DisplayName string
	Grade       string
}

type HintConclusion struct {
	Placement    *core.Cell
	Eliminations []solver.CandidateRef
}

type HintPlan struct {
	PlanID     string
	Strategy   HintStrategy
	Summary    string
	Steps      []HintStep
	Conclusion HintConclusion
}

// String returns the complete plain-text fallback for limited renderers.
func (plan HintPlan) String() string { return plan.Summary }

func cellName(position core.Position) string {
	return fmt.Sprintf("row %d, column %d", position.Row+1, position.Column+1)
}

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

func narrationText(text string) HintNarrationChunk {
	return HintNarrationChunk{Kind: HintNarrationText, Text: text}
}

func narrationTarget(target HintTarget) HintNarrationChunk {
	copy := target
	return HintNarrationChunk{Kind: HintNarrationTarget, Target: &copy}
}

func narrationDigit(value int) HintNarrationChunk {
	return HintNarrationChunk{Kind: HintNarrationDigit, Value: value}
}

func composeHintPlan(snapshot Snapshot, strategy solver.Solver, move *solver.Move) *HintPlan {
	if move == nil {
		return nil
	}
	grade, _, _ := solver.StrategyTierForTechnique(move.Technique)
	display := move.Technique
	if strategy != nil {
		display = strategy.GetDisplayName()
	}
	plan := &HintPlan{
		Strategy: HintStrategy{ID: move.Technique, DisplayName: display, Grade: grade},
		Summary:  move.Reason,
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
	for index := range plan.Steps {
		if !plan.Steps[index].HasValidNarration() {
			plan.Steps[index].Narration = completeStepNarration(plan.Steps[index], display)
		}
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
	plan.Summary = hintSummary(plan.Steps)
	plan.PlanID = deterministicPlanID(snapshot, *plan)
	return plan
}

// completeStepNarration guarantees that every strategy exposes a closed,
// renderer-neutral narration stream. Teaching steps without specialized
// wording retain their complete authored message as one text token; final
// deductions additionally expose typed digits and targets so clients can
// render the result without parsing prose.
func completeStepNarration(step HintStep, display string) []HintNarrationChunk {
	if step.Effect == nil {
		return []HintNarrationChunk{narrationText(step.Message)}
	}
	conclusion := HintConclusion{
		Placement:    step.Effect.Placement,
		Eliminations: step.Effect.Eliminations,
	}
	chunks := []HintNarrationChunk{narrationText("Apply the deduction: ")}
	chunks = append(chunks, conclusionNarration(conclusion)...)
	chunks = append(chunks, narrationText(". This is the "+display+" result."))
	return chunks
}

func hintSummary(steps []HintStep) string {
	messages := make([]string, len(steps))
	for index, step := range steps {
		messages[index] = step.Message
	}
	return strings.Join(messages, " ")
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
		{ID: "observe-candidate-pattern", Kind: HintStepObserve, Message: "Look at the highlighted cells and candidates.", Marks: premiseMarks},
		{ID: "compare-candidate-constraints", Kind: HintStepCompare, Message: move.Reason, Marks: append(append([]HintMark(nil), premiseMarks...), comparisonMarks...)},
	}
	kind := HintStepConclude
	if conclusion.Placement == nil {
		kind = HintStepEliminate
	}
	steps = append(steps, HintStep{ID: "apply-deduction", Kind: kind, Message: conclusionMessage(conclusion), Marks: conclusionMarks(conclusion), Effect: conclusionEffect(conclusion)})
	return steps
}

func conclusionMessage(conclusion HintConclusion) string {
	if conclusion.Placement != nil {
		return fmt.Sprintf("Therefore %s must be %d.", cellName(conclusion.Placement.Position), conclusion.Placement.Value)
	}
	return eliminationMessage(conclusion.Eliminations)
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
		{ID: "inspect-" + compactCellName(move.Cell.Position), Kind: HintStepObserve, Message: fmt.Sprintf("Look at %s. It has only one candidate: %d.", cellName(move.Cell.Position), move.Cell.Value), Narration: []HintNarrationChunk{narrationText("Look at "), narrationTarget(cellTarget(move.Cell.Position)), narrationText(". It has only one candidate: "), narrationDigit(move.Cell.Value), narrationText(".")}, Marks: observeMarks},
		{ID: fmt.Sprintf("place-%s-%d", compactCellName(move.Cell.Position), move.Cell.Value), Kind: HintStepConclude, Message: fmt.Sprintf("Because no other digit can go in %s, place %d there. This is called a Naked Single.", cellName(move.Cell.Position), move.Cell.Value), Narration: []HintNarrationChunk{narrationText("Because no other digit can go in "), narrationTarget(cellTarget(move.Cell.Position)), narrationText(", place "), narrationDigit(move.Cell.Value), narrationText(" there. This is called a Naked Single.")}, Marks: []HintMark{{Target: cellTarget(move.Cell.Position), Role: HintRoleFocus}, {Target: candidateTarget(ref), Role: HintRoleConclusion}}, Effect: &HintEffect{Placement: &move.Cell}},
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
		{ID: fmt.Sprintf("scan-%s-for-%d", strings.ReplaceAll(unitName, " ", ""), move.Cell.Value), Kind: HintStepObserve, Message: fmt.Sprintf("In %s, consider where %d can appear.", unitName, move.Cell.Value), Narration: []HintNarrationChunk{narrationText("In "), narrationTarget(unit), narrationText(", consider where "), narrationDigit(move.Cell.Value), narrationText(" can appear.")}, Marks: marks},
		{ID: fmt.Sprintf("compare-%s-%d", strings.ReplaceAll(unitName, " ", ""), move.Cell.Value), Kind: HintStepCompare, Message: fmt.Sprintf("Every other empty cell in %s cannot contain %d, leaving only %s.", unitName, move.Cell.Value, cellName(move.Cell.Position)), Narration: []HintNarrationChunk{narrationText("Every other empty cell in "), narrationTarget(unit), narrationText(" cannot contain "), narrationDigit(move.Cell.Value), narrationText(", leaving only "), narrationTarget(cellTarget(move.Cell.Position)), narrationText(".")}, Marks: compare},
		{ID: fmt.Sprintf("place-%s-%d", compactCellName(move.Cell.Position), move.Cell.Value), Kind: HintStepConclude, Message: fmt.Sprintf("Because %s still needs %d and %s is the only possible cell, place %d there. This is called a Hidden Single.", unitName, move.Cell.Value, cellName(move.Cell.Position), move.Cell.Value), Narration: []HintNarrationChunk{narrationText("Because "), narrationTarget(unit), narrationText(" still needs "), narrationDigit(move.Cell.Value), narrationText(" and "), narrationTarget(cellTarget(move.Cell.Position)), narrationText(" is the only possible cell, place "), narrationDigit(move.Cell.Value), narrationText(" there. This is called a Hidden Single.")}, Marks: []HintMark{{Target: unit, Role: HintRoleFocus}, {Target: cellTarget(move.Cell.Position), Role: HintRoleFocus}, {Target: candidateTarget(ref), Role: HintRoleConclusion}}, Effect: &HintEffect{Placement: &move.Cell}},
	}
}

func nakedSubsetSteps(move *solver.Move, display string) []HintStep {
	evidence := move.Evidence
	unit, _ := unitTarget(evidence.Unit)
	premiseMarks := []HintMark{{Target: unit, Role: HintRoleFocus}}
	var cells []string
	var cellIDs []string
	valueSet := make(map[int]struct{})
	for _, premise := range evidence.Premises {
		cells = append(cells, cellName(premise.Position))
		cellIDs = append(cellIDs, compactCellName(premise.Position))
		for _, value := range premise.Values {
			valueSet[value] = struct{}{}
			premiseMarks = append(premiseMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: value}), Role: HintRolePremise})
		}
	}
	values := sortedValues(valueSet)
	valueText := intsText(values)
	unitName := namedUnit(evidence.Unit)
	count := len(evidence.Premises)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	finalMessage := fmt.Sprintf("Because those %d cells reserve %s, %s This pattern is called a %s.", count, valueText, deductionPhrase(conclusion), display)
	return []HintStep{
		{ID: "find-" + strings.Join(cellIDs, "-"), Kind: HintStepObserve, Message: fmt.Sprintf("In %s, look at %s. Together, those %d cells contain only %s.", unitName, joinNatural(cells), count, valueText), Marks: premiseMarks},
		{ID: "reserve-" + strings.Join(cellIDs, "-"), Kind: HintStepCompare, Message: fmt.Sprintf("Those %d cells must contain %s in some order, so no other cell in %s can use those digits.", count, valueText, unitName), Marks: premiseMarks},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Message: finalMessage, Marks: append(premiseMarks[1:], conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func hiddenSubsetSteps(move *solver.Move, display string) []HintStep {
	evidence := move.Evidence
	unit, _ := unitTarget(evidence.Unit)
	premiseMarks := []HintMark{{Target: unit, Role: HintRoleFocus}}
	var cells []string
	valueSet := make(map[int]struct{})
	for _, premise := range evidence.Premises {
		cells = append(cells, cellName(premise.Position))
		for _, value := range premise.Values {
			valueSet[value] = struct{}{}
			premiseMarks = append(premiseMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: value}), Role: HintRolePremise})
		}
	}
	values := sortedValues(valueSet)
	valueText := intsText(values)
	unitName := namedUnit(evidence.Unit)
	count := len(values)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	return []HintStep{
		{ID: "find-confined-" + move.Technique, Kind: HintStepObserve, Message: fmt.Sprintf("In %s, the digits %s can appear only in %s.", unitName, valueText, joinNatural(cells)), Marks: premiseMarks},
		{ID: "compare-confined-" + move.Technique, Kind: HintStepCompare, Message: fmt.Sprintf("Those %d digits must fill those %d cells in some order, so every other candidate in the cells can be removed.", count, len(cells)), Marks: premiseMarks},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Message: fmt.Sprintf("Because %s are confined to those cells, %s This pattern is called a %s.", valueText, deductionPhrase(conclusion), display), Marks: append(premiseMarks[1:], conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
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
	var cells []string
	value := 0
	for _, premise := range evidence.Premises {
		cells = append(cells, cellName(premise.Position))
		for _, candidate := range premise.Values {
			value = candidate
			premiseMarks = append(premiseMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: candidate}), Role: HintRolePremise})
		}
	}
	source := namedUnit(evidence.Unit)
	other := "the intersecting unit"
	if len(evidence.Units) > 1 {
		other = namedUnit(&evidence.Units[1])
	}
	name := "Pointing Pair"
	relation := fmt.Sprintf("Within %s, candidate %d appears only in %s, all on %s.", source, value, joinNatural(cells), other)
	if evidence.Unit != nil && evidence.Unit.Kind != solver.UnitBox {
		name = "Box-Line Reduction"
		relation = fmt.Sprintf("Within %s, candidate %d appears only in %s, all inside %s.", source, value, joinNatural(cells), other)
	}
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	return []HintStep{
		{ID: "find-intersection", Kind: HintStepObserve, Message: relation, Marks: premiseMarks},
		{ID: "compare-intersection", Kind: HintStepCompare, Message: fmt.Sprintf("Candidate %d is therefore unavailable in the rest of %s outside %s.", value, other, source), Marks: premiseMarks},
		{ID: "apply-intersection", Kind: conclusionKind(conclusion), Message: fmt.Sprintf("Because candidate %d is confined to that intersection, %s This is called %s.", value, deductionPhrase(conclusion), name), Marks: append(premiseMarks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func fishSteps(move *solver.Move, display string) []HintStep {
	evidence := move.Evidence
	size := map[string]int{"x-wing": 2, "swordfish": 3, "jellyfish": 4}[move.Technique]
	baseUnits, crossUnits := evidence.Units[:size], evidence.Units[size:size*2]
	digit := evidence.Premises[0].Values[0]
	marks := unitAndPremiseMarks(evidence)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	return []HintStep{
		{ID: "find-" + move.Technique, Kind: HintStepObserve, Message: fmt.Sprintf("For candidate %d, look at %s and %s. In each selected %s, %d appears only at the highlighted intersections.", digit, groupedUnits(baseUnits), groupedUnits(crossUnits), baseUnits[0].Kind, digit), Marks: marks},
		{ID: "compare-" + move.Technique, Kind: HintStepCompare, Message: fmt.Sprintf("Those %d %s confine candidate %d to the same %d %s, so other cells in those %s cannot contain %d.", size, pluralUnit(baseUnits[0].Kind), digit, size, pluralUnit(crossUnits[0].Kind), pluralUnit(crossUnits[0].Kind), digit), Marks: marks},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Message: fmt.Sprintf("Because candidate %d is locked into those intersections, %s This pattern is called %s.", digit, deductionPhrase(conclusion), display), Marks: append(marks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func relationalSteps(move *solver.Move, display string) []HintStep {
	evidence := move.Evidence
	premises := evidence.Premises
	marks := unitAndPremiseMarks(evidence)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	var observe, compare string
	var firstNodes, nodes []HintNode
	var firstLinks, links []HintLink
	switch move.Technique {
	case "xy-wing":
		shared := sharedCandidate(premises[1].Values, premises[2].Values)
		observe = fmt.Sprintf("Look at pivot %s with candidates %s and its two wings, %s with %s and %s with %s.", cellName(premises[0].Position), intsText(premises[0].Values), cellName(premises[1].Position), intsText(premises[1].Values), cellName(premises[2].Position), intsText(premises[2].Values))
		compare = fmt.Sprintf("Whichever value the pivot takes, one wing must contain %d, so any cell that sees both wings cannot contain %d.", shared, shared)
		if len(evidence.Links) == 2 {
			nodes = visualNodes(evidence, false)
			links = visualLinks(evidence)
			firstNodes, firstLinks = visualPrefix(nodes, links)
		}
	case "w-wing":
		link := premises[2].Values[0]
		pair := intsText(premises[0].Values)
		observe = fmt.Sprintf("Look at %s and %s. Both contain only %s, and candidate %d has a strong link between %s and %s.", cellName(premises[0].Position), cellName(premises[1].Position), pair, link, cellName(premises[2].Position), cellName(premises[3].Position))
		compare = "The strong link guarantees that one of the two outer cells takes the other candidate, so a cell that sees both outer cells cannot use it."
	case "xyz-wing":
		shared := sharedCandidate(premises[0].Values, premises[1].Values, premises[2].Values)
		observe = fmt.Sprintf("Look at pivot %s with candidates %s and its two wings, %s with %s and %s with %s.", cellName(premises[0].Position), intsText(premises[0].Values), cellName(premises[1].Position), intsText(premises[1].Values), cellName(premises[2].Position), intsText(premises[2].Values))
		compare = fmt.Sprintf("One of those three cells must contain %d, so any cell that sees the pivot and both wings cannot contain %d.", shared, shared)
	}
	observeMarks := marks
	if len(firstNodes) > 0 {
		observeMarks = marksForNodes(firstNodes)
	}
	return []HintStep{
		{ID: "find-" + move.Technique, Kind: HintStepObserve, Message: observe, Marks: observeMarks, Nodes: firstNodes, Links: firstLinks},
		{ID: "compare-" + move.Technique, Kind: HintStepCompare, Message: compare, Marks: marks, Nodes: nodes, Links: links},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Message: fmt.Sprintf("Because that candidate relationship holds, %s This pattern is called %s.", deductionPhrase(conclusion), display), Marks: append(marks, conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)},
	}
}

func bugPlusOneSteps(move *solver.Move, display string) []HintStep {
	if move.Evidence == nil || len(move.Evidence.Premises) == 0 || !move.IsPlacement() {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	premise := move.Evidence.Premises[0]
	marks := unitAndPremiseMarks(move.Evidence)
	conclusion := HintConclusion{Placement: placement(move)}
	return []HintStep{
		{ID: "find-trivalue-cell", Kind: HintStepObserve, Message: fmt.Sprintf("Look at %s. It is the only unsolved cell with three candidates: %s; every other unsolved cell has two.", cellName(premise.Position), intsText(premise.Values)), Marks: marks},
		{ID: "compare-extra-candidate", Kind: HintStepCompare, Message: fmt.Sprintf("Candidate %d is the extra candidate that keeps the row, column, and box candidate counts from forming a two-solution loop.", move.Cell.Value), Marks: marks},
		{ID: "apply-bug-plus-one", Kind: HintStepConclude, Message: fmt.Sprintf("Because removing that extra candidate would leave two interchangeable solutions, place %d at %s. This pattern is called %s.", move.Cell.Value, cellName(move.Cell.Position), display), Marks: append(marks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func uniquenessSteps(move *solver.Move, display string) []HintStep {
	evidence := move.Evidence
	if evidence == nil {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	if len(evidence.Premises) < 4 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations})
	}
	premises := evidence.Premises
	pair := sharedValues(premises[:4]...)
	if len(pair) < 2 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations})
	}
	pair = pair[:2]
	locations := make([]string, 4)
	for index := range locations {
		locations[index] = cellName(premises[index].Position)
	}
	marks := unitAndPremiseMarks(evidence)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	observe := fmt.Sprintf("Look at the rectangle corners %s. They span two rows, two columns, and two boxes, and all four contain candidates %s.", joinNatural(locations), intsText(pair))
	compare := ""
	switch move.Technique {
	case "unique-rectangle":
		compare = fmt.Sprintf("Three corners contain only %s. The fourth must use another candidate; otherwise the two digits could swap around the rectangle and create two solutions.", intsText(pair))
	case "unique-rectangle-2":
		extra := firstValueOutside(premises[2].Values, pair)
		compare = fmt.Sprintf("The other two corners contain only %s, while the two expanded corners both contain candidate %d. At least one expanded corner must use %d, so any cell that sees both cannot use it.", intsText(pair), extra, extra)
	case "unique-rectangle-3":
		unitName := namedUnit(evidence.Unit)
		compare = fmt.Sprintf("The extra candidates in the expanded corners join the highlighted cells in %s to reserve those digits, so the same candidates can be removed elsewhere in that unit.", unitName)
	case "unique-rectangle-4":
		removed := 0
		if len(evidence.Eliminations) > 0 {
			removed = evidence.Eliminations[0].Value
		}
		conjugate := pair[0]
		if conjugate == removed {
			conjugate = pair[1]
		}
		compare = fmt.Sprintf("In %s, candidate %d appears in exactly the two expanded corners. One of them must contain %d, so candidate %d can be removed from both.", namedUnit(evidence.Unit), conjugate, conjugate, removed)
	}
	return []HintStep{
		{ID: "find-" + move.Technique, Kind: HintStepObserve, Message: observe, Marks: marks},
		{ID: "compare-" + move.Technique, Kind: HintStepCompare, Message: compare, Marks: marks},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Message: fmt.Sprintf("Because the alternative would make the solution non-unique, %s This pattern is called %s.", deductionPhrase(conclusion), display), Marks: append(marks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
}

func coloringSteps(move *solver.Move, display string) []HintStep {
	evidence := move.Evidence
	if evidence == nil {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	if len(evidence.Partitions) != 2 || len(evidence.Premises) == 0 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations})
	}
	digit := evidence.Premises[0].Values[0]
	marks := premiseCandidateMarks(evidence)
	nodes := visualNodes(evidence, false)
	links := visualLinks(evidence)
	firstNodes, firstLinks := visualPrefix(nodes, links)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	compare := fmt.Sprintf("Alternating that either-or choice separates the highlighted candidates into two groups. Each highlighted target sees one cell from each group, so whichever group is true, the target cannot contain %d.", digit)
	if partitionConflict(evidence.Partitions[0]) || partitionConflict(evidence.Partitions[1]) {
		compare = "Alternating that either-or choice separates the highlighted candidates into two groups. Two cells in one group share a unit, so that group cannot be true and the other group must be true."
	}
	return []HintStep{
		{ID: "find-simple-coloring", Kind: HintStepObserve, Message: fmt.Sprintf("Start with one linked pair for candidate %d. Exactly one end can contain %d.", digit, digit), Marks: marksForNodes(firstNodes), Nodes: firstNodes, Links: firstLinks},
		{ID: "compare-simple-coloring", Kind: HintStepCompare, Message: compare, Marks: marks, Nodes: nodes, Links: links},
		{ID: "apply-simple-coloring", Kind: conclusionKind(conclusion), Message: fmt.Sprintf("Because the alternating groups settle the highlighted candidates, %s This method is called %s.", deductionPhrase(conclusion), display), Marks: append(marks, conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)},
	}
}

func partitionConflict(partition solver.CandidatePartition) bool {
	for left := 0; left < len(partition.Candidates); left++ {
		for right := left + 1; right < len(partition.Candidates); right++ {
			if sharesTeachingUnit(partition.Candidates[left].Position, partition.Candidates[right].Position) {
				return true
			}
		}
	}
	return false
}

func xCycleSteps(move *solver.Move, display string) []HintStep {
	evidence := move.Evidence
	if evidence == nil {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	if len(evidence.Chain) < 3 || len(evidence.Links) < 3 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations})
	}
	digit := evidence.Links[0].From.Value
	marks := premiseCandidateMarks(evidence)
	nodes := visualNodes(evidence, true)
	links := visualLinks(evidence)
	firstNodes, firstLinks := visualPrefix(nodes, links)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	compare := fmt.Sprintf("The links alternate around the loop. If the first cell contained %d, following the cannot-both and must-contain links returns requiring it not to contain %d, so %d must be removed there.", digit, digit, digit)
	if conclusion.Placement != nil && conclusion.Placement.Value == digit {
		compare = fmt.Sprintf("The links alternate around the loop. If the first cell did not contain %d, following the must-contain and cannot-both links returns requiring it to contain %d, so %d must go there.", digit, digit, digit)
	}
	return []HintStep{
		{ID: "find-x-cycle", Kind: HintStepObserve, Message: fmt.Sprintf("Start candidate %d at %s. The first solid or dashed segment shows whether one end must contain it or both ends cannot.", digit, cellName(evidence.Chain[0].Position)), Marks: marksForNodes(firstNodes), Nodes: firstNodes, Links: firstLinks},
		{ID: "compare-x-cycle", Kind: HintStepCompare, Message: compare, Marks: marks, Nodes: nodes, Links: links},
		{ID: "apply-x-cycle", Kind: conclusionKind(conclusion), Message: fmt.Sprintf("Because the alternating loop closes with that forced result, %s This pattern is called %s.", deductionPhrase(conclusion), display), Marks: append(marks, conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)},
	}
}

func conclusionNarration(conclusion HintConclusion) []HintNarrationChunk {
	if conclusion.Placement != nil {
		return []HintNarrationChunk{narrationText("place "), narrationDigit(conclusion.Placement.Value), narrationText(" in "), narrationTarget(cellTarget(conclusion.Placement.Position))}
	}
	chunks := []HintNarrationChunk{narrationText("remove candidate ")}
	for index, ref := range conclusion.Eliminations {
		if index > 0 {
			chunks = append(chunks, narrationText(", "))
		}
		chunks = append(chunks, narrationDigit(ref.Value), narrationText(" from "), narrationTarget(cellTarget(ref.Position)))
	}
	return chunks
}

func xyChainSteps(move *solver.Move, display string) []HintStep {
	evidence := move.Evidence
	if evidence == nil {
		return evidenceSteps(move, HintConclusion{Placement: placement(move)})
	}
	if len(evidence.Chain) < 4 {
		return evidenceSteps(move, HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations})
	}
	marks := premiseCandidateMarks(evidence)
	nodes := visualNodes(evidence, true)
	links := visualLinks(evidence)
	firstNodes, firstLinks := visualPrefix(nodes, links)
	conclusion := HintConclusion{Placement: placement(move), Eliminations: evidence.Eliminations}
	endpoint := sharedCandidate(evidence.Chain[0].Values, evidence.Chain[len(evidence.Chain)-1].Values)
	finalNarration := []HintNarrationChunk{narrationText("Because every highlighted target sees both endpoints, it cannot contain "), narrationDigit(endpoint), narrationText("; ")}
	finalNarration = append(finalNarration, conclusionNarration(conclusion)...)
	finalNarration = append(finalNarration, narrationText(". This pattern is called an "+display+"."))
	return []HintStep{
		{ID: "find-xy-chain", Kind: HintStepObserve, Message: "Start with the first two-candidate cells. The numbered arrow introduces the first implication.", Narration: []HintNarrationChunk{narrationText("Start with "), narrationTarget(cellTarget(evidence.Chain[0].Position)), narrationText(" and "), narrationTarget(cellTarget(evidence.Chain[1].Position)), narrationText(". The numbered arrow introduces the first implication.")}, Marks: marksForNodes(firstNodes), Nodes: firstNodes, Links: firstLinks},
		{ID: "compare-xy-chain", Kind: HintStepCompare, Message: fmt.Sprintf("If the first endpoint does not contain %d, its other candidate forces the shared candidate in the next cell. That implication continues through the numbered chain until the last endpoint must contain %d, so at least one endpoint contains %d.", endpoint, endpoint, endpoint), Narration: []HintNarrationChunk{narrationText("If the first endpoint does not contain "), narrationDigit(endpoint), narrationText(", its other candidate forces the shared candidate in the next highlighted cell. That implication continues through the numbered chain until the last endpoint must contain "), narrationDigit(endpoint), narrationText(", so at least one endpoint contains "), narrationDigit(endpoint), narrationText(".")}, Marks: marks, Nodes: nodes, Links: links},
		{ID: "apply-xy-chain", Kind: conclusionKind(conclusion), Message: fmt.Sprintf("Because every highlighted target sees both endpoints, it cannot contain %d; %s This pattern is called an %s.", endpoint, deductionPhrase(conclusion), display), Narration: finalNarration, Marks: append(marks, conclusionMarks(conclusion)...), Nodes: nodes, Links: links, Effect: conclusionEffect(conclusion)},
	}
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

func firstValueOutside(values, excluded []int) int {
	for _, value := range values {
		found := false
		for _, blocked := range excluded {
			if value == blocked {
				found = true
				break
			}
		}
		if !found {
			return value
		}
	}
	return 0
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

func groupedUnits(units []solver.UnitRef) string {
	indices := make([]int, len(units))
	for index, unit := range units {
		indices[index] = unit.Index + 1
	}
	return pluralUnit(units[0].Kind) + " " + intsText(indices)
}

func pluralUnit(kind solver.UnitKind) string {
	return string(kind) + "s"
}

func sharedCandidate(groups ...[]int) int {
	counts := make(map[int]int)
	for _, values := range groups {
		seen := make(map[int]struct{})
		for _, value := range values {
			seen[value] = struct{}{}
		}
		for value := range seen {
			counts[value]++
		}
	}
	for value, count := range counts {
		if count == len(groups) {
			return value
		}
	}
	return 0
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
		return HintStepConclude
	}
	return HintStepEliminate
}

func namedUnit(unit *solver.UnitRef) string {
	if unit == nil {
		return "the highlighted unit"
	}
	return fmt.Sprintf("%s %d", unit.Kind, unit.Index+1)
}

func sortedValues(set map[int]struct{}) []int {
	values := make([]int, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Ints(values)
	return values
}

func joinNatural(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

func lowerFirst(text string) string {
	if text == "" {
		return text
	}
	return strings.ToLower(text[:1]) + text[1:]
}

func intsText(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprint(value)
	}
	return joinNatural(parts)
}

func deductionPhrase(conclusion HintConclusion) string {
	if conclusion.Placement != nil {
		return fmt.Sprintf("%s is left with only %d, so place %d there.", cellName(conclusion.Placement.Position), conclusion.Placement.Value, conclusion.Placement.Value)
	}
	return lowerFirst(eliminationMessage(conclusion.Eliminations))
}

func eliminationMessage(refs []solver.CandidateRef) string {
	parts := make([]string, len(refs))
	for i, ref := range refs {
		parts[i] = fmt.Sprintf("%d from %s", ref.Value, cellName(ref.Position))
	}
	return "Remove candidate " + strings.Join(parts, ", ") + "; this deduction does not place a value."
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
