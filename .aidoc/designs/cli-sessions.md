---
domain: Designs
status: Active
entry_points:
  - cli/controller.go
  - cmd/play.go
  - sessionfile/session_file.go
dependencies:
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/e2e-session-scenarios.md
  - .aidoc/architecture/guidelines.md
---

# CLI Sessions

The CLI exposes manual notes and durable game sessions through the existing command-line frontend. The design keeps rules and validation in the game engine while the CLI owns command parsing, rendering, and filesystem transport.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/roadmap.md` | Current stabilization priorities and sequencing |
| `.aidoc/designs/game-engine.md` | Canonical note, history, snapshot, and serialization semantics |
| `.aidoc/designs/e2e-session-scenarios.md` | Black-box scenarios for notes and restored sessions |
| `.aidoc/architecture/guidelines.md` | Package boundaries that keep I/O out of the engine |

## Why CLI Sessions Exist

Longer and harder puzzles need candidate notes and often span more than one sitting. The engine already supports both concerns, but engine-only capabilities do not make the CLI more playable and do not prove that a frontend can render snapshots or transport complete sessions correctly.

Explicit persistence is the smallest reliable first workflow. A player chooses when and where to save, which avoids hidden lifecycle policy, surprise files, and autosave recovery rules while the serialization boundary is still being validated through a real frontend.

## Command Model

Interactive commands extend the current vocabulary without changing existing aliases or bare three-digit value input:

- `note`, `n <row><column><value>` computes the cell’s next complete note set and submits `game.SetNotes`.
- `notes-clear`, `x <row><column>` submits an empty `game.SetNotes` value for the cell.
- `save <path>` serializes the current game and atomically replaces the destination.
- `quit`, `q` continues to exit without an implicit save.

The root command accepts `--resume <path>` as an alternative to puzzle generation and `--input`. Resume constructs a game with the host's current solver options, validates the entire document through `game.Restore`, and starts the existing interactive loop only after restoration succeeds. `--resume` conflicts with `--input` and `--level` because a serialized session already defines its puzzle.

Command parsing reports missing, extra, malformed, and out-of-range arguments consistently. Engine errors remain typed internally and are translated into concise terminal messages by `cli.Controller` rather than exposed as raw implementation details.

## Note Rendering

`cli.Controller.PrintBoard` derives all visible values and notes from one detached `game.Snapshot`. Filled cells retain the compact value display. Empty cells with notes use a deterministic three-by-three candidate layout so digits keep fixed positions and peer-note cleanup is visible without a separate inspection command.

The note-aware board may occupy more terminal rows than the compact board, but rendering must remain plain text, stable under redirected output, and free of terminal capability dependencies. A board without notes should retain the current compact shape to preserve the familiar default experience and existing black-box compatibility.

## Persistence Transport

The frontend writes serialized bytes to a temporary file in the destination directory, flushes and closes the file, preserves user-only permissions, and renames it over the destination. Keeping the temporary file beside the destination gives the rename the same-filesystem atomicity expected by the save contract.

Save failures leave any existing destination unchanged and report the failed path. Restore reads have a bounded size before calling `game.Restore`; malformed, unsupported, invalid, and oversized files fail before interactive play begins. The CLI never edits or deletes a rejected source file.

Session JSON remains engine-owned and opaque to `cmd` and `cli`. Frontend code calls `game.Game.Serialize` and `game.Restore` without decoding fields, synthesizing history, or depending on schema details beyond typed restoration errors.

## Ownership Boundaries

- `game.Game` owns note rules, peer cleanup, complete history, serialization, and semantic validation.
- `cli.Controller` owns interactive commands, note rendering, help, and user-facing error translation.
- `cmd` owns root flags, file selection, restore startup, and construction of solver options.
- A small persistence helper owns bounded reads and atomic writes so filesystem mechanics do not become controller behavior.

Persistence helpers accept explicit paths and byte slices and return errors. Persistence helpers do not choose default locations, print messages, or depend on Cobra, which keeps the transport independently testable and reusable by a later frontend.

## Failure and Compatibility Rules

- A failed note or save command leaves the in-memory session unchanged.
- A failed restore never falls back to a generated puzzle.
- Save and resume preserve current values, invalid entries, notes, history records, and the undo/redo cursor.
- Existing value, hint, repair, solve, reset, undo, redo, and quit behavior remains compatible.
- Session files contain gameplay state only; difficulty labels, database paths, and terminal preferences remain frontend configuration.

## Verification

Package tests cover command parsing, snapshot rendering, bounded reads, atomic replacement, and error translation. Cobra and built-binary scenarios cover option conflicts, manual notes, peer cleanup, mixed note/value undo and redo, save/resume round trips, redo preservation, corrupt input rejection, and protection of an existing destination when saving fails.
