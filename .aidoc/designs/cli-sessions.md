---
domain: Designs
status: Active
entry_points:
  - cli/controller.go
  - cmd/play.go
  - session/service.go
  - sessionfile/session_file.go
dependencies:
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/e2e-session-scenarios.md
  - .aidoc/designs/localization-contract.md
  - .aidoc/architecture/guidelines.md
---

# CLI Sessions

The line CLI keeps play deliberately compact while preserving complete durable game sessions. Candidate editing and candidate-focused teaching belong to the TUI; the engine remains frontend-neutral.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/game-engine.md` | Canonical candidate, history, snapshot, and serialization semantics |
| `.aidoc/designs/e2e-session-scenarios.md` | Black-box save and restore scenarios |
| `.aidoc/designs/hint-presentation-protocol.md` | Shared hint-plan contract and frontend degradation rules |
| `.aidoc/designs/localization-contract.md` | Shared canonical English rendering and descriptor ownership |
| `.aidoc/architecture/guidelines.md` | Package boundaries that keep I/O out of the engine |

## Why the Line CLI Is Compact

The line CLI is valuable for portable value entry, automation-friendly output, and explicit session transport. Rendering candidate grids and advanced teaching scenes made this frontend much taller and duplicated the TUI's stronger interaction model. The line CLI therefore always renders one value-only grid and leaves candidate entry, visibility, and candidate-focused hints to `sudoku tui`.

The engine still owns one authoritative candidate grid. A session created by another frontend may be resumed and saved through the line CLI without dropping hidden candidate state; the line CLI simply does not expose commands that mutate it.

## Command and Hint Model

Interactive commands retain bare three-digit value input, value add/clear, history, check/repair, reset/solve, explicit save, and quit. The root command accepts `--resume <path>` as an alternative to puzzle generation and `--input`.

`hint` or `i` queries one shared engine plan, renders its descriptor-backed explanation through the canonical English catalog, and immediately applies that exact `plan_id` as one atomic action. A placement plan renders its concluding semantic narration plus the exact row, column, and value. A candidate correction or elimination prints one client-specific limitation without rendering candidate details, applies the hidden candidate change, and lets the next `hint` continue from the resulting authoritative state. `sudoku tui` remains the complete preview and teaching surface.

The line CLI uses direct player language for its own help, prompts, board checks, errors, and completion message. It does not expose engine enum names or implementation-oriented descriptions as presentation copy.

Each command stops after one authoritative deduction, never skips candidate work or silently chains to a placement, and does not mutate when plan generation or exact-plan application fails. Successful application creates one ordinary Undo entry; stale-plan validation remains aligned with the TUI, API, and browser.

## Persistence Transport

`save <path>` serializes the complete engine state and atomically replaces the destination. The frontend writes beside the destination, flushes and closes the private mode-`0600` file, then renames it so a failed save leaves any existing destination unchanged.

Restore reads are bounded before `game.Restore` validates the complete document. Malformed, unsupported, invalid, and oversized files fail before interactive play; the CLI never edits or deletes a rejected source. Session JSON remains engine-owned and opaque to `cmd` and `cli`.

## Ownership and Verification

`game.Game` owns values, candidates, history, serialization, and semantic validation. `session.Service` owns frontend-neutral restore and puzzle acquisition with explicit solver/catalog dependencies and structured outcomes. `cli.Controller` owns compact rendering, commands, hint degradation, help, and error translation. `cmd` owns root flags, process signal registration, managed catalog composition, and startup wording; `sessionfile` owns bounded reads and atomic writes.

Package tests cover canonical descriptor and narration rendering, compact rendering despite hidden candidates, atomic placement and candidate hints, one-step Undo, continuation from candidate work, rejected legacy hint arguments, complete session round trips, bounded reads, atomic replacement, and error translation. Built-binary scenarios cover command help, compact placement output, hidden candidate-plan progression, save/resume with redo state, corrupt input rejection, flag conflicts, and failed-save preservation.
