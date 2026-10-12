// Command export_hint_plans writes deterministic API-shaped hint fixtures for renderer acceptance tests.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/solver"
)

type manifest struct {
	SchemaVersion int `json:"schema_version"`
	Cases         []struct {
		Strategy    string `json:"strategy"`
		DisplayName string `json:"display_name"`
		Grade       string `json:"grade"`
		Puzzle      string `json:"puzzle"`
	} `json:"cases"`
}

type output struct {
	SchemaVersion int              `json:"schema_version"`
	Cases         []map[string]any `json:"cases"`
}

func main() {
	contents, err := os.ReadFile("game/testdata/hint_strategy_games.json")
	must(err)
	var source manifest
	must(json.Unmarshal(contents, &source))
	if source.SchemaVersion != 1 || len(source.Cases) != 23 {
		panic("hint strategy manifest must contain schema version 1 and 23 cases")
	}

	store := solver.NewStore()
	keys := store.GetAllStrategySolverKeys()
	result := output{SchemaVersion: 1, Cases: make([]map[string]any, 0, len(source.Cases))}
	for _, entry := range source.Cases {
		board := core.NewEmptyBoard()
		board.FromString(entry.Puzzle)
		options := game.NewDefaultOptions(store)
		options.StrategySolverKeys = keys
		current := game.NewGame(board, options)
		for number := 1; number <= 128; number++ {
			plan := current.Hint()
			if plan == nil {
				panic(fmt.Sprintf("%s exhausted hints", entry.Strategy))
			}
			if plan.Strategy.ID == entry.Strategy {
				result.Cases = append(result.Cases, map[string]any{
					"strategy":      entry.Strategy,
					"display_name":  entry.DisplayName,
					"grade":         entry.Grade,
					"puzzle":        entry.Puzzle,
					"reached_after": number,
					"hint":          wirePlan(plan),
				})
				break
			}
			_, err = current.Apply(game.ApplyHint{PlanID: plan.PlanID})
			must(err)
		}
	}
	if len(result.Cases) != 23 {
		panic(fmt.Sprintf("exported %d of 23 strategies", len(result.Cases)))
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	must(encoder.Encode(result))
}

func wirePlan(plan *game.HintPlan) map[string]any {
	steps := make([]map[string]any, len(plan.Steps))
	for i, step := range plan.Steps {
		marks := make([]map[string]any, len(step.Marks))
		for j, mark := range step.Marks {
			marks[j] = map[string]any{"target": wireTarget(mark.Target), "role": mark.Role}
		}
		steps[i] = map[string]any{"id": step.ID, "kind": step.Kind, "message": step.Message, "marks": marks}
		if step.Effect != nil {
			steps[i]["effect"] = wireEffect(step.Effect.Placement, step.Effect.Eliminations)
		}
	}
	return map[string]any{
		"protocol_version": plan.ProtocolVersion,
		"plan_id":          plan.PlanID,
		"strategy":         map[string]any{"id": plan.Strategy.ID, "display_name": plan.Strategy.DisplayName, "grade": plan.Strategy.Grade},
		"summary":          plan.Summary,
		"steps":            steps,
		"conclusion":       wireEffect(plan.Conclusion.Placement, plan.Conclusion.Eliminations),
	}
}

func wireTarget(target game.HintTarget) map[string]any {
	out := map[string]any{"kind": target.Kind}
	if target.Position != nil {
		out["row"], out["column"] = target.Position.Row+1, target.Position.Column+1
	}
	if target.Value != 0 {
		out["value"] = target.Value
	}
	if target.Kind == game.HintTargetRow || target.Kind == game.HintTargetColumn || target.Kind == game.HintTargetBox {
		out["index"] = target.Index + 1
	}
	return out
}

func wireEffect(placement *core.Cell, eliminations []solver.CandidateRef) map[string]any {
	out := map[string]any{"eliminations": make([]map[string]int, len(eliminations))}
	if placement != nil {
		out["placement"] = map[string]int{"row": placement.Position.Row + 1, "column": placement.Position.Column + 1, "value": placement.Value}
	}
	items := out["eliminations"].([]map[string]int)
	for i, ref := range eliminations {
		items[i] = map[string]int{"row": ref.Position.Row + 1, "column": ref.Position.Column + 1, "value": ref.Value}
	}
	return out
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
