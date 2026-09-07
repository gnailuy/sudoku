---
domain: Designs
status: Active
entry_points:
  - cmd/root.go
  - cmd/session.go
  - cli/controller.go
  - tui/model.go
  - webapi/server.go
  - db/db.go
  - db/puzzle.go
dependencies:
  - .aidoc/designs/database-puzzle-selection.md
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/e2e-database-scenarios.md
  - .aidoc/designs/database-concurrency.md
---

# Database Play Statistics and History Reset

The database keeps completion and acquisition histories as separate concepts, exposes both through `sudoku db stats`, and resets only an explicitly selected history dimension. Player actions can count one completion per play run; automatic solve, abandonment, and elapsed duration are outside the completion contract.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/database-puzzle-selection.md` | Acquisition counters and the selection policy that consumes them |
| `.aidoc/designs/game-engine.md` | Solved status and typed actions used to detect completion |
| `.aidoc/designs/e2e-database-scenarios.md` | Black-box statistics and reset acceptance scenarios |
| `.aidoc/designs/database-concurrency.md` | Mixed-workload snapshot/reset contention and lock bounds |
| `.aidoc/designs/roadmap.md` | Current and future database work |

## Why Completion Is Separate From Acquisition

`play_count` means that a stored puzzle was selected and exposed to a player. Acquisition increments before play begins so selection can prefer unseen puzzles and balance reuse even when a process later quits, crashes, or saves for another day.

Completion is the smallest reliable gameplay outcome. The engine reports when an accepted player action reaches `game.StatusSolved`, while process exit cannot distinguish abandonment from a crash or resumable session. Completion history therefore records a count and latest timestamp without inferring abandonment or solving duration.

## Completion Contract

A **play run** begins when a frontend creates or restores a playable `game.Game` and ends when that frontend exits or discards it.

- The first successful player action that changes an unsolved run to `game.StatusSolved` counts once.
- `game.ActionSolve`, invalid moves, persistence, recovery creation, exit, and loading an already solved session do not count.
- Hint-assisted completion counts because hints remain explicit player actions.
- Undoing and re-solving within one run does not count twice.
- Completing an unfinished restored session counts once for that new run.

The line CLI, TUI, and HTTP API use one frontend-neutral play-run tracker around `game.Game.Apply`. The engine remains storage-neutral, and a statistics failure never reverses an accepted game action.

## Storage and Identity

`db.DB.migrate` adds non-null `completion_count` with a zero default and nullable `last_completed_at` columns. Existing rows begin with no completion history, and `db.DB.RecordCompletion` atomically increments the count and assigns SQLite's current timestamp.

The existing normalized 81-character puzzle string remains the primary key. Imports, generation, and direct input retain digit-relabel normalization and `INSERT OR IGNORE`; equivalent digit labels share history, while rotations, reflections, transposition, and row or column symmetry do not.

`cmd.createSession` retains the normalized key and selected database path for the run tracker. Restored sessions derive the key from immutable givens. A missing row or failed write produces a concise warning without inserting another row or changing gameplay/session persistence.

## Statistics Snapshot

`sudoku db stats [--db <path>] [--level <easy|medium|hard|expert|evil>]` prints one row per included strategy grade and one overall row. Each row reports stored puzzles, never-selected and selected puzzles, total acquisitions, completed puzzles, total completions, and latest selection and completion times.

Statistics labels preserve the acquisition/completion distinction; no value represents abandonment, elapsed duration, or unique players. Empty timestamps render as `-`, and an unknown grade fails before database access. One SQLite read transaction provides a consistent per-grade and overall snapshot even when another client updates history concurrently.

## Explicit History Reset

`sudoku db reset-history --history <acquisition|completion|all> [--level <grade>] [--db <path>] [--yes]` requires an explicit history dimension. Acquisition reset clears `play_count` and `last_played_at`; completion reset clears `completion_count` and `last_completed_at`; `all` clears both in one transaction.

The preview names the database, grade filter, affected row count, and counters. Interactive use requires exact affirmative confirmation; non-interactive use requires `--yes`. Cancellation and empty selections do not mutate data.

Reset preserves puzzle rows, classification, source, normalized keys, explicit saves, recovery records, and any unselected history dimension. Concurrent history updates occur wholly before or after the reset transaction, so no partial count/timestamp pair is visible.

## Failure, Compatibility, and Privacy

- Existing databases and sessions remain readable after the additive migration.
- Statistics remain local to the selected SQLite file; no account identifier, telemetry, or network reporting is added.
- Completion updates use the bounded SQLite busy timeout and return promptly under contention.
- A completion-write failure leaves the game solved and visible; a reset failure rolls back its complete scope and exits non-zero.
- Once-per-run tracking is in memory. Replaying one saved unfinished session in another process creates another run.

## Verification

Package tests cover migration, atomic increments, filtered snapshots, reset scopes, rollback, missing normalized rows, automatic-solve exclusion, hints, undo/re-solve suppression, and concurrency. Built-binary scenarios cover separate history dimensions, frontend consistency, normalized identity, grade filtering, explicit confirmation, preserved data, and failure reporting; `.aidoc/designs/e2e-database-scenarios.md` is the canonical scenario list.

## Deferred Decisions

Abandonment, elapsed duration, durable attempt identities, player attribution, telemetry, minimum-clue policy, and full Sudoku-symmetry canonicalization remain separate decisions that require a concrete product need.
