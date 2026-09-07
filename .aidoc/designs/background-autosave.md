---
domain: Designs
status: Active
entry_points:
  - cmd/tui.go
  - tui/model.go
  - sessionfile/session_file.go
dependencies:
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/tui-frontend.md
  - .aidoc/designs/cli-sessions.md
  - .aidoc/designs/e2e-tui-scenarios.md
---

# Background Autosave and Crash Recovery

Background autosave protects active TUI games from process, terminal, and host failures without changing the game-engine serialization contract. Recovery stays local, private, bounded, and distinct from the player-owned explicit save workflow.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/roadmap.md` | Current stabilization priorities and sequencing |
| `.aidoc/designs/game-engine.md` | Canonical serialized gameplay state restored from recovery data |
| `.aidoc/designs/tui-frontend.md` | Event-loop, dirty-state, modal, and explicit-save behavior |
| `.aidoc/designs/cli-sessions.md` | Existing atomic transport and player-owned save contract |
| `.aidoc/designs/e2e-tui-scenarios.md` | Black-box recovery, cleanup, failure, and compatibility scenarios |

## Why Local Recovery Exists

The TUI supports longer games, notes, history, candidates, and explicit saves, but an interrupted process can lose every change since the last manual save. Local recovery protects that state without introducing the conflict model required by cloud sync or multi-device storage.

Background recovery deliberately changes the TUI's no-surprise-file policy. The change is acceptable only with a documented private location, clear startup and discard behavior, bounded retention, and an opt-out. The line-oriented CLI remains script-friendly and keeps explicit persistence only.

## Recovery Scope

TUI autosave is enabled by default and can be disabled with `--no-autosave`. The TUI creates recovery state only after the first successful gameplay mutation; candidate visibility, focus, theme, help, failed actions, and other presentation state never create or update recovery files.

A recovery record wraps opaque bytes from `game.Game.Serialize` with a small frontend-owned format version, random record identifier, creation and update timestamps, and an optional display-safe source label. The wrapper never duplicates puzzle fields or history semantics, and `game.Restore` remains the only semantic validator.

Plain `sudoku tui` startup discovers valid recovery records before generating a puzzle. The player can resume the newest record, choose another record when several exist, discard records explicitly, or continue to a new game. Explicit `--input`, `--level`, and `--resume` requests bypass discovery so automation and intentional startup remain deterministic.

## Storage, Privacy, and Retention

Recovery records live under `$XDG_STATE_HOME/sudoku/recovery`, falling back to `$HOME/.local/state/sudoku/recovery`. The directory uses mode `0700`, records use mode `0600`, filenames contain random identifiers rather than puzzle content, and writes use the bounded atomic transport guarantees in `sessionfile`.

Each running TUI instance has one randomly identified recovery record, so concurrent games never overwrite one another. Recovery records are never keyed or selected by an operating-system process ID: after a restart, plain `sudoku tui` scans the private recovery directory, validates eligible records, and lets the player select one. The resumed instance adopts the selected durable record identifier for later autosaves. Recovery discovery ignores symlinks, non-regular files, oversized records, malformed wrappers, unsupported versions, and sessions rejected by `game.Restore`.

A successful explicit save removes the current recovery record. Further gameplay mutations create a fresh record. A clean unmodified exit or confirmed dirty-session discard removes the current record; interruption, `Ctrl-C`, write failure, or host loss leaves the last successful record available. Startup prunes invalid records and records older than 30 days, while valid recent records require an explicit resume or discard decision.

## Write and Conflict Model

A successful engine action increments a TUI recovery generation and schedules a one-second debounced write. Serialization occurs on the event loop; filesystem work runs as a Bubble Tea command carrying immutable bytes and the generation number. At most one write is active per model, and a newer generation coalesces into one follow-up write after the active write completes.

Only completion for the latest persisted generation can report recovery as current. Out-of-order completion cannot erase newer work or clear gameplay dirty state. Autosave failure keeps the game playable, shows a concise persistent warning, and retries after the next mutation; explicit save errors continue to use the existing save modal behavior.

Recovery files are never merged. Separate randomly identified records avoid writer conflicts, and recovery followed by explicit save uses the existing destination replacement contract. If the explicit destination changed outside Sudoku, explicit save reports the transport result rather than silently selecting or merging another file.

## Ownership Boundaries

- `game.Game` owns serialization, restoration, history, and semantic validation without autosave concepts.
- A presentation-neutral recovery package owns wrapper validation, secure location selection, discovery, pruning, atomic writes, and deletion.
- `tui.Model` owns mutation generations, debounce and single-flight coordination, recovery prompts, warnings, and cleanup decisions.
- `cmd/tui.go` owns flags and decides whether startup discovery applies before constructing a new session.
- `sessionfile` retains bounded read and atomic mode-`0600` replacement primitives shared by explicit saves and recovery storage.

## Verification

Recovery and model tests cover secure path selection, wrapper bounds and versions, symlink rejection, retention pruning, concurrent record isolation, debounce coalescing, stale completion, retry behavior, selection, and cleanup decisions. The pseudo-terminal harness covers gameplay compatibility, explicit save/resume, crash recovery, durable-record selection, private modes, cleanup, and opt-out behavior.

Applicable root CLI black-box scenarios verify unchanged output, flags, save bytes, and no recovery-file creation. Repository verification also covers build, package tests, vet, lint, documentation structure, pseudo-terminal behavior, and black-box compatibility.

## Deferred Work

Recovery does not add cloud sync, cross-device merge, account identity, encryption beyond operating-system file permissions, periodic snapshots, recovery history, or line-oriented CLI autosave. Those capabilities require separate product and threat-model decisions.
