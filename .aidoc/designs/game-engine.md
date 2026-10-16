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
| `.aidoc/designs/roadmap.md` | Approved project priorities and sequencing |
| `.aidoc/designs/hint-presentation-protocol.md` | Approved renderer-neutral replacement for the current hint shape |
| `.aidoc/designs/e2e-play-scenarios.md` | Black-box CLI behavior that must remain compatible |

## Why the Engine Boundary Exists

`game.Game` already avoids terminal I/O, but public mutable boards, command-shaped methods, and unexported history state prevent a frontend from treating it as a durable engine contract. A stable boundary lets every frontend observe the same state, submit the same actions, and receive the same validation results without duplicating Sudoku rules.

The engine extends the original `game` package rather than introducing a second game model. The CLI, TUI, and HTTP API share this contract; graphical frontends remain outside this repository.

## Engine Responsibilities

The engine owns:

- immutable puzzle givens and the current player state;
- accepted values, invalid player entries, and one authoritative candidate grid;
- action validation and deterministic state transitions;
- unified undo/redo history for value and candidate actions;
- solved, valid, and conflict status;
- hint calculation through the configured solver store;
- complete current-state serialization and restoration.

Frontends own presentation, command parsing, keyboard or pointer gestures, accessibility, localization, file/network transport, and autosave timing. The engine returns structured state and errors; it does not print, read input, exit a process, or choose UI wording.

## Public Contract

The stable contract is organized around four concepts:

- `Game` is the authoritative mutable session owned by one caller at a time.
- `Snapshot` is a detached read model containing givens, visible values, invalid markers, the authoritative candidate grid, a cumulative mistake count, status, and undo/redo availability. Mutating a snapshot cannot mutate the game.
- `Action` is a typed player intent: set or clear a value, replace one cell’s complete candidate set, reset, restart, repair, solve, undo, redo, or apply a hint. `SetCandidates` accepts an empty, one-digit, or multi-digit complete replacement.
- `Result` describes the accepted transition, changed cells, current status, undo/redo availability, and the recommendation used by an applied hint. Invalid actions return typed errors and leave state unchanged.

The hint query returns one complete renderer-neutral `HintPlan` without mutation. It first returns an exact value or candidate correction when player state is inconsistent, and only then selects a strategy against a detached board constrained by the authoritative candidate grid. Step navigation remains client-local; accepting the preview submits `ApplyHint` with its deterministic `plan_id`, verifies the conclusion, and records one value or candidate transition.

`cli.Controller` renders only `Game.Snapshot` values and performs game operations only through `Game.Apply`. Command-shaped mutation helpers are private engine implementation details; `Game.Apply` is the public mutation boundary. Public callers must not receive mutable references to the engine's internal boards or history.

`Reset` preserves the cumulative mistake count and restores the initial legal candidate grid. `Restart` also clears mistakes while preserving puzzle identity and givens. Both clear values, invalid entries, history, and solved state; Undo and Redo cannot cross either boundary.

## State and Validation Invariants

- Puzzle givens never change after game creation or restoration.
- Every accepted action is atomic: all value, candidate, status, and history changes succeed together or not at all.
- New actions after an undo discard the redo tail.
- Undo and redo restore the complete prior state, including invalid markers and every candidate change caused by an action.
- User mistakes remain observable for rendering but never contaminate the solver's valid board.
- Each accepted `SetValue` that the engine confirms as invalid increments the session's cumulative mistake count. Candidate edits, hints, erase, Undo, and Redo do not change it; undo history therefore cannot reduce or replay mistakes.
- Public input errors return typed errors; panics remain reserved for violated internal invariants.
- Snapshots are self-consistent and safe for asynchronous rendering after the engine advances.
- A `Game` is not concurrently mutable. Frontends serialize actions and may pass snapshots between goroutines.

## Candidate Semantics

The candidate grid is player-controlled engine state represented by `core.CandidateSet`. New and restarted games initialize editable empty cells from accepted-value legality. `SetCandidates` replaces one complete cell set and enters unified history.

Setting any visible value clears the target candidates. Only a value accepted into the solver-safe board removes that digit from candidate peers; a visible invalid entry does not constrain peers. Clearing a value restores the target’s current legal set. Every cleanup belongs to the same action delta, so Undo restores the exact previous grid.

A hint validates wrong values first, then removes candidates ruled out by accepted values and restores a missing digit required by the solved continuation. Strategy selection starts only after validation and applies the authoritative grid as constraints on a detached solver board. Elimination conclusions mutate the same grid directly, so later hints, serialization, Undo, Redo, CLI, TUI, and HTTP clients all continue from one state.

## Serialization Contract

`Game.Serialize()` returns the complete session in the one current JSON shape. `game.Restore(data, options)` validates that exact shape and returns a newly constructed game. The host supplies `Options` during restoration because solver configuration is executable host policy, not persisted player state. Format changes replace the shape across coordinated callers and stored development data instead of adding version branches or compatibility adapters.

The current shape has these fields:
- `puzzle`: the original 81-character puzzle string;
- `mistakes`: the cumulative count of confirmed invalid value submissions;
- `current`: the current player-controlled state;
- `history`: ordered before/after state records;
- `cursor`: the applied history record, or `-1` when no record is applied.

Each player-controlled state stores separate 81-character `values` and `invalid` grids plus sparse `candidates` records containing a zero-based row, zero-based column, and digits. `values` excludes puzzle givens. Separating invalid entries keeps player mistakes visible without contaminating the solver board. Before/after history records use the same complete state shape, so restoration preserves mixed value/candidate Undo and Redo exactly.

Restoration rejects malformed JSON, unknown fields, invalid or unsolvable puzzles, edits to givens, overlapping valid/invalid entries, invalid candidate records, candidates on occupied cells, unsolvable accepted values, entries incorrectly marked invalid, disconnected history, and out-of-range cursors. Current state is stored explicitly alongside the validated history and cursor. All validation completes before a `Game` is returned, so corrupt input cannot produce a partially initialized session. Failures use `StateError` and its stable code: `malformed-state`, `invalid-puzzle`, `invalid-session`, or `invalid-history`.

Serialized game state has no version discriminator or migration path. An obsolete or future field is an unknown field and fails as malformed input. `Game.ToString` remains diagnostic text and is not a persistence format.

## Compatibility and Failure Boundaries

The engine preserves existing CLI commands and black-box behavior unless a separately reviewed UX change says otherwise. The CLI renders snapshots and translates commands into actions; it must not inspect engine internals.

Serialization failures, invalid actions, unavailable undo/redo, and attempts to edit givens are expected errors. Solver inability to produce a hint is a valid no-hint result, not an engine failure. Corrupt restored state must never produce a partially initialized game.

## Verification

Engine tests cover every action, typed error, atomic rollback, candidate cleanup, correction-first hint priority, deterministic hint identity, placement and elimination plans, exact-plan rejection, atomic elimination Undo, mixed value/candidate history, redo truncation, immutable snapshots, and serialization round trips. Restoration tests include malformed input and rejection of obsolete fields.

The CLI boundary is verified against `.aidoc/designs/e2e-play-scenarios.md` by building the binary and exercising it as a black box. Package tests cover repair, solve, applied-hint metadata, typed errors, and snapshot isolation; package tests, `go vet`, golangci-lint, and CI must remain green.
