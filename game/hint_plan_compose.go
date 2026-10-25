package game

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/message"
	"github.com/gnailuy/sudoku/solver"
)

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
