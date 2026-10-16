---
domain: Designs
status: Active
entry_points:
  - game/contract.go
  - tui/model.go
  - tui/render.go
dependencies:
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/hint-presentation-protocol.md
  - .aidoc/designs/e2e-tui-scenarios.md
---

# Authoritative Candidates

The game owns one candidate grid. Players, strategies, persistence, and every frontend read and mutate that same state; there is no separate manual-note map, derived automatic grid, or adoption transition.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/game-engine.md` | Defines candidate state, actions, history, and validation |
| `.aidoc/designs/hint-presentation-protocol.md` | Defines correction-first hint plans and strategy plans |
| `.aidoc/designs/tui-frontend.md` | Defines candidate visibility and editing gestures |
| `.aidoc/designs/e2e-tui-scenarios.md` | Protects the composed terminal behavior |

## Why One Grid Exists

Separate notes and automatic candidates create two truths about the same empty cell. Hints then need adoption rules, applied eliminations must be copied between models, and a renderer can show a grid different from the state the solver uses. One authoritative grid removes that ambiguity.

A new or restarted game initializes every editable empty cell to its legal candidates under the givens. `SetCandidates` replaces one complete cell set as an atomic player action. Accepted value placement clears the target and removes that digit from candidate peers; invalid visible input clears only its occupied target and never constrains peers. Undo, Redo, reset, restart, save, and restore preserve the exact grid.

## Engine Contract

`game.Snapshot.Candidates` is detached value data. Given and filled cells have empty sets. Candidate edits are allowed only on editable empty cells and contain distinct digits 1–9.

Candidate state is serialized directly with the session and history. Schema version 3 stores sparse `candidates` records beside values and invalid entries. Restoring validates positions, digits, occupancy, history connectivity, and the accepted board before returning a game.

All clients consume `Candidates` and `SetCandidates` directly. The candidate grid is the only persisted pencilmark state.

## Validation Before Strategy Hints

A hint request validates player state before selecting a Sudoku strategy:

1. A visible invalid value produces a `value-correction` plan pointing to that exact cell and an undoable `clear-value` conclusion.
2. For each editable empty cell, the engine compares the candidate set with accepted-value legality and a solved continuation. Impossible candidates are marked extra, and a missing candidate required by the solution is marked missing. The first inconsistency produces a `candidate-correction` plan with one complete replacement set.
3. Only a consistent board and candidate grid proceed to strategy selection. Strategy solvers receive a detached board constrained by the authoritative candidate grid, so the explanation and application use the same state the player sees.

Correction plans use the same deterministic `plan_id`, local step navigation, stale-plan rejection, application boundary, revision increment, and one-step Undo as strategy plans. Correction titles are not presented as Sudoku technique names.

## Presentation

Frontends may hide or show the grid without changing game state. The TUI and web client start with candidates hidden, reveal them for candidate-edit mode or a successful hint preview, and retain the visibility preference until the player changes it. Visibility is presentation-only and is not serialized.

A fixed 3×3 mini-grid renders each candidate. Correction scenes distinguish the problematic cell, impossible digits, missing required digits, and the corrected conclusion without relying on color alone. Filled, given, and invalid-visible cells render their value instead of candidates.

The line CLI uses the same authoritative candidate state and dense candidate renderer. Its existing `note` aliases remain command vocabulary for candidate toggles, but they submit `SetCandidates`; no second note model exists.

## Verification

Engine tests prove initialization, direct edits, peer cleanup, invalid-value isolation, exact undo/redo, version-3 round trips, correction priority, deterministic correction identity, stale rejection, and strategy selection against the candidate grid. TUI, CLI, API, and browser tests prove the single wire/state shape, visibility-only toggles, correction rendering, exact application, and persistence across ownership modes.
