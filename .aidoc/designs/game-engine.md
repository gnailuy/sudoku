---
domain: Designs
status: Active
entry_points:
  - game/game.go
  - game/contract.go
  - game/serialization.go
dependencies:
  - .aidoc/architecture/guidelines.md
  - .aidoc/designs/roadmap.md
  - .aidoc/designs/e2e-play-scenarios.md
---

# Game Engine

`game.Game` is the stable, reusable engine for the CLI, TUI, and HTTP API. The engine owns game rules and state transitions; frontends own input, rendering, navigation, and persistence transport.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/architecture/guidelines.md` | Current package boundaries that the engine must preserve |
| `.aidoc/designs/roadmap.md` | Current frontend delivery order |
| `.aidoc/designs/e2e-play-scenarios.md` | Black-box CLI behavior that must remain compatible |

## Why the Engine Boundary Exists

`game.Game` already avoids terminal I/O, but public mutable boards, command-shaped methods, and unexported history state prevent a frontend from treating it as a durable engine contract. A stable boundary lets every frontend observe the same state, submit the same actions, and receive the same validation results without duplicating Sudoku rules.

The engine extends the original `game` package rather than introducing a second game model. The CLI, TUI, and HTTP API share this contract; graphical frontends remain outside this repository.

## Engine Responsibilities

The engine owns:

- immutable puzzle givens and the current player state;
- accepted values, invalid player entries, and player notes;
- action validation and deterministic state transitions;
- unified undo/redo history for value and note actions;
- solved, valid, and conflict status;
- hint calculation through the configured solver store;
- versioned state serialization and restoration.

Frontends own presentation, command parsing, keyboard or pointer gestures, accessibility, localization, file/network transport, and autosave timing. The engine returns structured state and errors; it does not print, read input, exit a process, or choose UI wording.

## Public Contract

The stable contract is organized around four concepts:

- `Game` is the authoritative mutable session owned by one caller at a time.
- `Snapshot` is a detached read model containing givens, visible values, invalid markers, manual notes, derived legal candidates, status, and undo/redo availability. Mutating a snapshot cannot mutate the game.
- `Action` is a typed player intent: set or clear a value, replace a cell’s complete note set, reset, repair, solve, undo, redo, or apply a hint. An empty, one-digit, or multi-digit `SetNotes` value is the single engine mutation for manual notes; frontends translate toggle-style input into a complete set instead of expanding the engine contract.
- `Result` describes the accepted transition, changed cells, current status, undo/redo availability, and the recommendation used by an applied hint. Invalid actions return typed errors and leave state unchanged.

`Hint` remains a query: it returns a structured recommendation with position, value, technique, and explanation. Applying the recommendation is a separate action so hints participate in history exactly like player moves.

`cli.Controller` renders only `Game.Snapshot` values and performs game operations only through `Game.Apply`. Command-shaped mutation helpers are private engine implementation details; `Game.Apply` is the public mutation boundary. Public callers must not receive mutable references to the engine's internal boards or history.

## State and Validation Invariants

- Puzzle givens never change after game creation or restoration.
- Every accepted action is atomic: all value, note, status, and history changes succeed together or not at all.
- New actions after an undo discard the redo tail.
- Undo and redo restore the complete prior state, including invalid markers and all note changes caused by an action.
- User mistakes remain observable for rendering but never contaminate the solver's valid board.
- Public input errors return typed errors; panics remain reserved for violated internal invariants.
- Snapshots are self-consistent and safe for asynchronous rendering after the engine advances.
- A `Game` is not concurrently mutable. Frontends serialize actions and may pass snapshots between goroutines.

## Note-Taking Semantics

Notes are player annotations, distinct from `core.Board.Candidates`, which computes legal solver candidates. Notes use `core.CandidateSet` as a value representation but are stored by the game engine.

Notes are allowed only on editable empty cells and contain digits 1–9. Setting a value clears notes in that cell and removes that value from notes in peer cells. Clearing a value does not recreate notes. Every automatic note cleanup is part of the same action delta, so undo restores the exact previous notes.

`Game.Snapshot` derives legal candidates from the solver-safe play board through `core.Board.Candidates`. Derived candidates remain separate from manual notes, actions, history, dirty state, and serialization; frontends decide whether to display them.

## Serialization Contract

`Game.Serialize()` returns the complete session as versioned JSON. `game.Restore(data, options)` validates that JSON and returns a newly constructed game. The host supplies `Options` during restoration because solver configuration is executable host policy, not persisted player state.

Version 1 has these fields:

- `version`: schema version, currently `1`;
- `puzzle`: the original 81-character puzzle string;
- `current`: the current player-controlled state;
- `history`: ordered before/after state records;
- `cursor`: the applied history record, or `-1` when no record is applied.

Each player-controlled state stores separate 81-character `values` and `invalid` grids plus sparse `notes` records containing a zero-based row, zero-based column, and note digits. `values` excludes puzzle givens. Separating invalid entries keeps player mistakes visible without contaminating the solver board. Before/after history records use the same complete state shape, so restoration preserves mixed value/note undo and redo exactly.

Restoration rejects malformed JSON, unknown fields, unsupported versions, invalid or unsolvable puzzles, edits to givens, overlapping valid/invalid entries, invalid note records, unsolvable accepted values, entries incorrectly marked invalid, disconnected history, and out-of-range cursors. Current state is stored independently from the cursor because temporary compatibility adapters may change state outside the stable action history; restoration preserves that state and the adapter's existing undo/redo behavior. All validation completes before a `Game` is returned, so corrupt input cannot produce a partially initialized session. Failures use `StateError` and its stable code: `malformed-state`, `unsupported-version`, `invalid-puzzle`, `invalid-session`, or `invalid-history`.

Unknown future schema versions fail explicitly; migrations can be added per version. `Game.ToString` remains diagnostic text and is not a persistence format.

## Compatibility and Failure Boundaries

The engine preserves existing CLI commands and black-box behavior unless a separately reviewed UX change says otherwise. The CLI renders snapshots and translates commands into actions; it must not inspect engine internals.

Serialization failures, invalid actions, unavailable undo/redo, and attempts to edit givens are expected errors. Solver inability to produce a hint is a valid no-hint result, not an engine failure. Corrupt restored state must never produce a partially initialized game.

## Verification

Engine tests cover every action, typed error, atomic rollback, note cleanup, mixed value/note undo-redo sequences, redo truncation, immutable snapshots, and serialization round trips. Restoration tests include malformed and unsupported versions.

The CLI boundary is verified against `.aidoc/designs/e2e-play-scenarios.md` by building the binary and exercising it as a black box. Package tests cover repair, solve, applied-hint metadata, typed errors, and snapshot isolation; package tests, `go vet`, golangci-lint, and CI must remain green.
