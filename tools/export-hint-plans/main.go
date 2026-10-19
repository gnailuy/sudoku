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
		nodes := make([]map[string]any, len(step.Nodes))
		for j, node := range step.Nodes {
			nodes[j] = map[string]any{"id": node.ID, "target": wireTarget(node.Target)}
			if node.Group != "" {
				nodes[j]["group"] = node.Group
			}
			if node.Order != 0 {
				nodes[j]["order"] = node.Order
			}
		}
		links := make([]map[string]any, len(step.Links))
		for j, link := range step.Links {
			links[j] = map[string]any{"from": link.From, "to": link.To, "kind": link.Kind}
		}
		steps[i] = map[string]any{"id": step.ID, "kind": step.Kind, "message": step.Message, "marks": marks, "nodes": nodes, "links": links}
		if len(step.Narration) > 0 {
			narration := make([]map[string]any, len(step.Narration))
			for j, chunk := range step.Narration {
				narration[j] = map[string]any{"kind": chunk.Kind}
				if chunk.Message != nil {
					narration[j]["message"] = chunk.Message
				}
				if chunk.Target != nil {
					narration[j]["target"] = wireTarget(*chunk.Target)
				}
				if chunk.Value != 0 {
					narration[j]["value"] = chunk.Value
				}
			}
			steps[i]["narration"] = narration
		}
		if step.Effect != nil {
			steps[i]["effect"] = wireEffect(*step.Effect)
		}
	}
	return map[string]any{
		"plan_id":  plan.PlanID,
		"kind":     plan.Kind,
		"title":    plan.Title,
		"strategy": map[string]any{"id": plan.Strategy.ID, "name": plan.Strategy.Name, "grade": plan.Strategy.Grade, "grade_name": plan.Strategy.GradeName},
		"summary":  plan.Summary,
		"steps":    steps,
		"conclusion": wireEffect(game.HintEffect{
			Placement:     plan.Conclusion.Placement,
			Eliminations:  plan.Conclusion.Eliminations,
			ClearValue:    plan.Conclusion.ClearValue,
			SetCandidates: plan.Conclusion.SetCandidates,
		}),
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

func wireEffect(effect game.HintEffect) map[string]any {
	switch {
	case effect.ClearValue != nil:
		return map[string]any{"kind": "clear-value", "clear_value": map[string]int{"row": effect.ClearValue.Row + 1, "column": effect.ClearValue.Column + 1}}
	case effect.SetCandidates != nil:
		return map[string]any{"kind": "set-candidates", "set_candidates": map[string]any{"row": effect.SetCandidates.Position.Row + 1, "column": effect.SetCandidates.Position.Column + 1, "values": effect.SetCandidates.Values.Values()}}
	case effect.Placement != nil:
		return map[string]any{"kind": "place-value", "placement": map[string]int{"row": effect.Placement.Position.Row + 1, "column": effect.Placement.Position.Column + 1, "value": effect.Placement.Value}}
	default:
		items := make([]map[string]int, len(effect.Eliminations))
		for i, ref := range effect.Eliminations {
			items[i] = map[string]int{"row": ref.Position.Row + 1, "column": ref.Position.Column + 1, "value": ref.Value}
		}
		return map[string]any{"kind": "eliminate-candidates", "eliminations": items}
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
