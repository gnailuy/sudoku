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

const HintProtocolVersion = 1

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

type HintMark struct {
	Target HintTarget
	Role   HintRole
}

type HintEffect struct {
	Placement    *core.Cell
	Eliminations []solver.CandidateRef
}

type HintStep struct {
	ID      string
	Kind    HintStepKind
	Message string
	Marks   []HintMark
	Effect  *HintEffect
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
	ProtocolVersion int
	PlanID          string
	Strategy        HintStrategy
	Summary         string
	Steps           []HintStep
	Conclusion      HintConclusion
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
		ProtocolVersion: HintProtocolVersion,
		Strategy:        HintStrategy{ID: move.Technique, DisplayName: display, Grade: grade},
		Summary:         move.Reason,
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
	default:
		plan.Steps = evidenceSteps(move, plan.Conclusion)
	}
	plan.Summary = hintSummary(plan.Steps)
	plan.PlanID = deterministicPlanID(snapshot, *plan)
	return plan
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
		{ID: "inspect-" + compactCellName(move.Cell.Position), Kind: HintStepObserve, Message: fmt.Sprintf("Look at %s. It has only one candidate: %d.", cellName(move.Cell.Position), move.Cell.Value), Marks: observeMarks},
		{ID: fmt.Sprintf("place-%s-%d", compactCellName(move.Cell.Position), move.Cell.Value), Kind: HintStepConclude, Message: fmt.Sprintf("Because no other digit can go in %s, place %d there. This is called a Naked Single.", cellName(move.Cell.Position), move.Cell.Value), Marks: []HintMark{{Target: cellTarget(move.Cell.Position), Role: HintRoleFocus}, {Target: candidateTarget(ref), Role: HintRoleConclusion}}, Effect: &HintEffect{Placement: &move.Cell}},
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
		{ID: fmt.Sprintf("scan-%s-for-%d", strings.ReplaceAll(unitName, " ", ""), move.Cell.Value), Kind: HintStepObserve, Message: fmt.Sprintf("In %s, consider where %d can appear.", unitName, move.Cell.Value), Marks: marks},
		{ID: fmt.Sprintf("compare-%s-%d", strings.ReplaceAll(unitName, " ", ""), move.Cell.Value), Kind: HintStepCompare, Message: fmt.Sprintf("Every other empty cell in %s cannot contain %d, leaving only %s.", unitName, move.Cell.Value, cellName(move.Cell.Position)), Marks: compare},
		{ID: fmt.Sprintf("place-%s-%d", compactCellName(move.Cell.Position), move.Cell.Value), Kind: HintStepConclude, Message: fmt.Sprintf("Because %s still needs %d and %s is the only possible cell, place %d there. This is called a Hidden Single.", unitName, move.Cell.Value, cellName(move.Cell.Position), move.Cell.Value), Marks: []HintMark{{Target: unit, Role: HintRoleFocus}, {Target: cellTarget(move.Cell.Position), Role: HintRoleFocus}, {Target: candidateTarget(ref), Role: HintRoleConclusion}}, Effect: &HintEffect{Placement: &move.Cell}},
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
	switch move.Technique {
	case "xy-wing":
		shared := sharedCandidate(premises[1].Values, premises[2].Values)
		observe = fmt.Sprintf("Look at pivot %s with candidates %s and its two wings, %s with %s and %s with %s.", cellName(premises[0].Position), intsText(premises[0].Values), cellName(premises[1].Position), intsText(premises[1].Values), cellName(premises[2].Position), intsText(premises[2].Values))
		compare = fmt.Sprintf("Whichever value the pivot takes, one wing must contain %d, so any cell that sees both wings cannot contain %d.", shared, shared)
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
	return []HintStep{
		{ID: "find-" + move.Technique, Kind: HintStepObserve, Message: observe, Marks: marks},
		{ID: "compare-" + move.Technique, Kind: HintStepCompare, Message: compare, Marks: marks},
		{ID: "apply-" + move.Technique, Kind: conclusionKind(conclusion), Message: fmt.Sprintf("Because that candidate relationship holds, %s This pattern is called %s.", deductionPhrase(conclusion), display), Marks: append(marks, conclusionMarks(conclusion)...), Effect: conclusionEffect(conclusion)},
	}
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
