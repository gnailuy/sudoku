---
domain: Designs
status: Active
entry_points:
  - game/contract.go
  - tui/model.go
  - tui/render.go
dependencies:
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/tui-frontend.md
  - .aidoc/designs/e2e-tui-scenarios.md
---

# Automatic Candidates

Automatic candidates provide opt-in legal-candidate assistance without changing manual notes or persisted game state. Candidate calculation belongs to the game-engine read boundary, while each frontend owns whether and how the derived data is displayed.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/game-engine.md` | Defines the detached snapshot and the distinction between engine state and derived queries |
| `.aidoc/designs/tui-frontend.md` | Defines the keyboard, rendering, theme, and accessibility boundaries used by the first consumer |
| `.aidoc/designs/e2e-tui-scenarios.md` | Defines the black-box behaviors that protect candidate assistance |
| `.aidoc/designs/roadmap.md` | Defines current stabilization priorities |

## Why Automatic Candidates Are Derived

Legal candidates answer which digits remain possible under the accepted Sudoku values. Manual notes record a player's own reasoning and may intentionally be incomplete or temporarily wrong. Treating automatic candidates as notes would erase that distinction, create surprising history changes, and require synchronization rules during every action.

The engine therefore exposes legal candidates as detached, read-only snapshot data computed from the solver-safe play board. Automatic candidates are never actions, never history entries, and never part of the dirty-session calculation. Existing version 1 session files remain compatible because no candidate data or frontend preference is serialized.

## Engine Read Contract

`game.Snapshot` gains a candidate set for every board position. The implementation derives each set through `core.Board.Candidates` while constructing the snapshot; the engine does not add a second candidate algorithm or maintain a cache.

A candidate set is non-empty only when the solver-safe board position is empty and editable. Given cells and accepted player values have empty sets. An invalid visible entry does not constrain peers because invalid entries are already excluded from the solver-safe board, but its occupied screen cell suppresses candidate rendering until the player clears or repairs it.

Snapshot candidate arrays are value data and remain detached from `game.Game`. Mutating a returned set cannot change the session. Repeated snapshots of unchanged state produce identical candidate sets.

## Update Semantics

Every new snapshot reflects the current accepted board. Candidate sets therefore update after value entry, clear, undo, redo, hint application, solve, repair, reset, and restore without a candidate-specific transition or event.

Manual-note actions do not alter legal candidates. Automatic candidate changes do not appear in `game.Result.Changes`, because results describe accepted mutable-state transitions rather than recomputed read data. Frontends already refresh from the post-action snapshot and must not incrementally reconstruct candidate rules.

## TUI Interaction

The full-screen TUI adds `a` as the automatic-candidate toggle when no modal or text prompt is active. Automatic candidates start off for every process, including resumed sessions, so the existing uncluttered board remains the default. Toggling the display does not mark the session dirty and does not create an undo entry.

The help overlay and one-line key guide expose the toggle. The line-oriented CLI has no candidate command: a dense candidate board would complicate its stable text format without improving scriptability, while the shared snapshot contract remains available to future frontends.

## Rendering and Manual Notes

Automatic candidates and manual notes share the existing fixed 3×3 digit positions. A legal candidate uses a dim neutral style; a manual note uses the existing note emphasis. When the same digit is both legal and manually noted, manual-note styling wins. A manual note that is no longer legal remains visible with manual-note styling because automatic assistance must never rewrite player reasoning.

Dark and light themes use palette roles rather than hard-coded colors. `NO_COLOR` distinguishes automatic candidates with faint text and manual notes with normal or bold text; focused-cell and peer attributes compose without removing that distinction. Filled, given, or invalid-visible cells render their value state instead of mini-grid candidates.

The renderer must keep the current cell dimensions, board alignment, resize threshold, and heavy 3×3 boundaries. Turning automatic candidates on or off changes only mini-grid content and the toggle status shown near the mode indicator.

## Failure and Compatibility Boundaries

Candidate derivation has no recoverable runtime failure path: the engine operates on a validated 9×9 board and `core.Board.Candidates` already defines legal-candidate behavior. An empty candidate set on an editable cell is valid when the accepted board leaves no legal digit; the TUI renders the empty mini-grid and existing board status remains authoritative.

Automatic candidates do not add candidate persistence, automatic note population, note cleanup beyond existing value actions, solver-technique explanations, background computation, a new dependency, or a serialization version. Automatic candidates do not change root CLI output, save bytes, restore validation, hint selection, or solve behavior.

## Delivery and Verification

Automatic candidates use two implementation layers:

1. Extend detached snapshots with computed legal candidates and contract tests for empty, filled, invalid-visible, changed, restored, and mutation-isolation cases.
2. Add the TUI toggle, combined note/candidate rendering, help text, theme tests, and pseudo-terminal black-box coverage.

Package tests verify deterministic candidate derivation and rendering in dark, light, and no-color modes. Black-box tests verify toggle default and keyboard behavior, value and clear updates, undo/redo, hint application, reset, save/resume, invalid-entry suppression, resize, dirty-state neutrality, and coexistence with legal and stale manual notes.
