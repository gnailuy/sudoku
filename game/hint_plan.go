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
		plan.Steps = []HintStep{{ID: "conclusion", Kind: HintStepConclude, Message: move.Reason, Marks: conclusionMarks(plan.Conclusion), Effect: conclusionEffect(plan.Conclusion)}}
	}
	plan.PlanID = deterministicPlanID(snapshot, *plan)
	return plan
}

func nakedSingleSteps(move *solver.Move) []HintStep {
	ref := solver.CandidateRef{Position: move.Cell.Position, Value: move.Cell.Value}
	return []HintStep{
		{ID: "inspect-" + cellName(move.Cell.Position), Kind: HintStepObserve, Message: fmt.Sprintf("Cell %s has only one candidate: %d.", cellName(move.Cell.Position), move.Cell.Value), Marks: []HintMark{{Target: cellTarget(move.Cell.Position), Role: HintRoleFocus}, {Target: candidateTarget(ref), Role: HintRolePremise}}},
		{ID: fmt.Sprintf("place-%s-%d", cellName(move.Cell.Position), move.Cell.Value), Kind: HintStepConclude, Message: fmt.Sprintf("Therefore %s must be %d.", cellName(move.Cell.Position), move.Cell.Value), Marks: []HintMark{{Target: cellTarget(move.Cell.Position), Role: HintRoleFocus}, {Target: candidateTarget(ref), Role: HintRoleConclusion}}, Effect: &HintEffect{Placement: &move.Cell}},
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
		{ID: fmt.Sprintf("compare-%s-%d", strings.ReplaceAll(unitName, " ", ""), move.Cell.Value), Kind: HintStepCompare, Message: fmt.Sprintf("Every other empty cell in %s is ruled out for %d.", unitName, move.Cell.Value), Marks: compare},
		{ID: fmt.Sprintf("place-%s-%d", cellName(move.Cell.Position), move.Cell.Value), Kind: HintStepConclude, Message: fmt.Sprintf("Therefore %s must be %d.", cellName(move.Cell.Position), move.Cell.Value), Marks: []HintMark{{Target: unit, Role: HintRoleFocus}, {Target: cellTarget(move.Cell.Position), Role: HintRoleFocus}, {Target: candidateTarget(ref), Role: HintRoleConclusion}}, Effect: &HintEffect{Placement: &move.Cell}},
	}
}

func nakedPairSteps(move *solver.Move) []HintStep {
	evidence := move.Evidence
	unit, _ := unitTarget(evidence.Unit)
	premiseMarks := []HintMark{{Target: unit, Role: HintRoleFocus}}
	var cells []string
	var values []int
	for _, premise := range evidence.Premises {
		cells = append(cells, cellName(premise.Position))
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
		{ID: "find-" + strings.Join(cells, "-"), Kind: HintStepObserve, Message: fmt.Sprintf("In %s, %s contain the same candidates: %s.", unitName, strings.Join(cells, " and "), valueText), Marks: premiseMarks},
		{ID: "reserve-" + strings.Join(cells, "-"), Kind: HintStepCompare, Message: fmt.Sprintf("Those digits must occupy the pair cells, so they cannot appear elsewhere in %s.", unitName), Marks: premiseMarks},
		{ID: "remove-pair-candidates", Kind: HintStepEliminate, Message: eliminationMessage(evidence.Eliminations), Marks: append(premiseMarks[1:], conclusion...), Effect: &HintEffect{Eliminations: append([]solver.CandidateRef(nil), evidence.Eliminations...)}},
	}
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
