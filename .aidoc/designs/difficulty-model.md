---
domain: Designs
status: Active
entry_points:
  - generator/difficulty.go
  - solver/classify.go
  - solver/config.go
  - solver/scoring.go
dependencies:
  - .aidoc/architecture/guidelines.md
  - .aidoc/designs/difficulty-calibration.md
---

# Strategy Rating Model

Easy through Evil are deterministic strategy grades assigned from the highest technique tier required by the canonical solver. The grades do not predict player experience; weighted score orders puzzles within a grade, while clue count guides generation.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/architecture/guidelines.md` | Solver interface contract and layer boundaries |
| `.aidoc/designs/difficulty-calibration.md` | Measurement contract and evidence required for policy changes |
| `.aidoc/INDEX.md` | Discovery index |

## Why Strategy Grades Exist

Human difficulty depends on experience, recognition speed, interface, and play conditions that this repository does not observe. A canonical strategy trace is objective, reproducible, explainable, and under product control, so the trace provides the authoritative rating contract.

The familiar Easy through Evil names remain public, but their meaning is strictly solver-relative. Human ratings and telemetry are outside the current model; player evidence is neither a prerequisite for strategy grades nor grounds for silently changing them. `.aidoc/designs/future-directions.md` owns any player-difficulty proposal.

## Strategy Grade Contract

The canonical hierarchy groups the solver inventory into five ordered capability tiers. A completed puzzle receives the grade of the highest tier represented in its trace; a higher numeric score cannot promote or demote that grade. `solver.StrategyTierNames`, `solver.StrategySolverKeysForTier`, and `solver.StrategyTierForTechnique` define the hierarchy and inventory.

Published Sudoku technique references informed the original grouping, but the product contract does not claim that the hierarchy predicts human effort. Moving a technique between tiers changes product semantics and requires a separately reviewed proposal with reproducible before-and-after evidence.

`solver.ClassifyPuzzle` applies strategies in stable registration order and records the highest explicit technique tier. A board that is already solved requires no moves and belongs to Easy; a trace that stalls before completion has no completed strategy grade.

## Generation Guidance

`generator.Difficulty` combines clue guidance with the strategy keys introduced at a requested tier. `generator.Difficulty.AllowedSolverKeys` includes lower-tier techniques, while `generator.Difficulty.LowerTierSolverKeys` supports the check that the generated puzzle genuinely requires the requested tier.

Clue ranges constrain the generator's search space but do not assign or override a completed puzzle's grade. `solver/config.go` is the canonical source for current clue ranges and weights; the design document intentionally does not duplicate those values.

Generation and classification share the canonical hierarchy through `solver.StrategyTierNames` and `solver.StrategySolverKeysForTier`, preventing their tier order and membership from drifting. Existing Go and wire fields retain the name `Difficulty` for compatibility, but their values carry strategy grades.

## Classification Outcome

`solver.Classification.Outcome` distinguishes `solved` from `strategy-unsolved`. A stalled trace is never promoted to Evil and never represented as a backtracking result; generator target matching accepts only completed strategy solves.

`solver.Classification.Difficulty` retains its legacy field name for compatibility and stores the highest required strategy tier. `solver.Classification.MaxTechnique` and `solver.Classification.Moves` preserve the explainable trace evidence behind that grade.

## Within-Grade Scoring

`solver.ScorePuzzle` sums the configured weight of every recognized strategy move. Unknown techniques contribute zero, keeping backtracking and other non-strategy operations outside the rating contract.

Score ranks completed puzzles only within the same strategy grade. Numeric weights are subordinate to the explicit tier hierarchy, so cross-grade score overlap is expected and does not invalidate a grade.

## Calibration Boundary

Solver weights, technique tiers, clue guidance, generation budgets, and success-rate policy must not change from intuition alone. `.aidoc/designs/difficulty-calibration.md` defines the corpus, deterministic classification semantics, measurements, report artifacts, and decision gates required before policy changes.

`solver/config.go` remains the auditable configuration boundary. A proposal to add score bands or alter classification requires baseline distributions, pathological-input analysis, rejected alternatives, and before-and-after validation.
