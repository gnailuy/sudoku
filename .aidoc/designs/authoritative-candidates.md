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

Separate annotation and solver-candidate models create two truths about the same empty cell. Hints then need adoption rules, applied eliminations must be copied between models, and a renderer can show a grid different from the state the solver uses. One authoritative grid removes that ambiguity.

A new or restarted game initializes every editable empty cell to its legal candidates under the givens. `SetCandidates` replaces one complete cell set as an atomic player action. Accepted value placement clears the target and removes that digit from candidate peers; invalid visible input clears only its occupied target and never constrains peers. Undo, Redo, reset, restart, save, and restore preserve the exact grid.

## Engine Contract

`game.Snapshot.Candidates` is detached value data. Given and filled cells have empty sets. Candidate edits are allowed only on editable empty cells and contain distinct digits 1–9.

Candidate state is serialized directly with the session and history. The one current state shape stores sparse `candidates` records beside values and invalid entries. Restoring validates positions, digits, occupancy, history connectivity, and the accepted board before returning a game.

All clients consume `Candidates` and `SetCandidates` directly. The candidate grid is the only persisted pencilmark state.

## Validation Before Strategy Hints

A hint request validates player state before selecting a Sudoku strategy:

1. A visible invalid value produces a `value-correction` plan pointing to that exact cell and an undoable `clear-value` conclusion.
2. For each editable empty cell, the engine checks accepted-value legality and verifies that the candidate set still supports the puzzle's unique continuation. The first set with an impossible digit, an empty state, or a missing solution-supporting digit produces a `candidate-correction` plan that resets the whole cell to every digit allowed directly by its accepted row, column, and box values. The plan may mark locally impossible extras, but it never identifies which restored digit belongs to the solved continuation.
3. Only a consistent board and candidate grid proceed to strategy selection. Strategy solvers receive a detached board constrained by the authoritative candidate grid, so the explanation and application use the same state the player sees.

Correction plans use the same deterministic `plan_id`, local step navigation, stale-plan rejection, application boundary, revision increment, and one-step Undo as strategy plans. Correction titles are not presented as Sudoku technique names.

## Presentation

Frontends own an 81-cell visibility mask without changing game state. The TUI and web client start with every cell hidden. Candidates on reveals every editable empty cell, Candidates off hides every cell, and note input on a hidden cell reveals only that cell. The first edit of a hidden cell replaces its stored set from an empty visible baseline; later edits use the now-visible authoritative set. Editing a visible cell starts from its complete displayed set. A successful hint preview reveals every candidate cell and leaves them visible after close or apply.

Visibility is renderer state rather than game data. The web may persist the mask by game identity in browser storage, while the TUI keeps it in its presentation model; neither engine serialization nor Undo/Redo contains visibility. A future cross-device preference may use a separate non-history record rather than expanding the game state.

A fixed 3×3 mini-grid renders each candidate. Correction scenes distinguish the problematic cell, any locally impossible digits, and the complete locally legal replacement without relying on color alone. The scene never singles out a solution-supporting digit. Filled, given, and invalid-visible cells render their value instead of candidates.

The line CLI preserves the same authoritative candidate state but deliberately renders only values and exposes no candidate-editing commands. `hint` prints and immediately applies one exact plan; candidate corrections and eliminations use a compact capability message, mutate the hidden shared grid, and stop so the next `hint` continues from that state. The TUI and HTTP clients expose the complete candidate surface.

## Verification

Engine tests prove initialization, direct edits, peer cleanup, invalid-value isolation, exact undo/redo, current-shape round trips, full-cell candidate reset, preservation of valid reduced sets, correction priority, deterministic correction identity, stale rejection, and strategy selection against the candidate grid. TUI, CLI, API, and browser tests prove the single wire/state shape, visibility-only toggles, compact CLI degradation, correction rendering, exact application, and persistence across ownership modes.
