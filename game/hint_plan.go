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
	case "naked-pair":
		plan.Steps = nakedPairSteps(move)
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

func nakedPairSteps(move *solver.Move) []HintStep {
	evidence := move.Evidence
	unit, _ := unitTarget(evidence.Unit)
	premiseMarks := []HintMark{{Target: unit, Role: HintRoleFocus}}
	var cells []string
	var cellIDs []string
	var values []int
	for _, premise := range evidence.Premises {
		cells = append(cells, cellName(premise.Position))
		cellIDs = append(cellIDs, compactCellName(premise.Position))
		if len(values) == 0 {
			values = append(values, premise.Values...)
		}
		for _, value := range premise.Values {
			premiseMarks = append(premiseMarks, HintMark{Target: candidateTarget(solver.CandidateRef{Position: premise.Position, Value: value}), Role: HintRolePremise})
		}
	}
	sort.Ints(values)
	valueText := intsText(values)
	unitName := fmt.Sprintf("%s %d", evidence.Unit.Kind, evidence.Unit.Index+1)
	conclusion := conclusionMarks(HintConclusion{Eliminations: evidence.Eliminations})
	return []HintStep{
		{ID: "find-" + strings.Join(cellIDs, "-"), Kind: HintStepObserve, Message: fmt.Sprintf("In %s, the cells at %s each have exactly the same two candidates: %s.", unitName, strings.Join(cells, " and "), valueText), Marks: premiseMarks},
		{ID: "reserve-" + strings.Join(cellIDs, "-"), Kind: HintStepCompare, Message: fmt.Sprintf("Those two cells must contain %s in some order, so no other cell in %s can contain either digit.", valueText, unitName), Marks: premiseMarks},
		{ID: "remove-pair-candidates", Kind: HintStepEliminate, Message: fmt.Sprintf("Because those two cells reserve %s, %s This pattern is called a Naked Pair.", valueText, lowerFirst(eliminationMessage(evidence.Eliminations))), Marks: append(premiseMarks[1:], conclusion...), Effect: &HintEffect{Eliminations: append([]solver.CandidateRef(nil), evidence.Eliminations...)}},
	}
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
	return strings.Join(parts, " and ")
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
