---
domain: Designs
status: Active
entry_points:
  - cmd/play.go
  - db/db.go
  - db/puzzle.go
dependencies:
  - .aidoc/designs/difficulty-model.md
  - .aidoc/designs/database-play-statistics.md
  - .aidoc/designs/database-concurrency.md
  - .aidoc/designs/e2e-database-scenarios.md
---

# Database Puzzle Selection

Puzzle acquisition prefers an exact strategy grade, avoids immediate repeats, and remains useful after every stored puzzle has been played. A database acquisition atomically selects and marks one puzzle; never-played puzzles come first, then the least-played and least-recently-played puzzle. Existing databases migrate in place with all rows initially unplayed.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/difficulty-model.md` | Defines the exact strategy-grade contract used by selection |
| `.aidoc/designs/database-play-statistics.md` | Keeps completion counters and history reset separate from acquisition semantics |
| `.aidoc/designs/database-concurrency.md` | Extends atomic acquisition into a mixed-handle and multi-process reliability contract |
| `.aidoc/designs/e2e-database-scenarios.md` | Owns black-box acceptance scenarios for acquisition and migration |
| `.aidoc/designs/roadmap.md` | Sequences this behavior before other database enhancements |

## Why Track Acquisition

Random lookup can return the same puzzle repeatedly while other exact-grade puzzles remain unused. A permanent played/not-played filter avoids repeats only until the pool is exhausted, after which the database stops helping. Selection therefore needs durable history and an explicit recycling policy.

The database is a local puzzle pool, not a game-session ledger. It records that a puzzle was chosen for play, but completion, abandonment, moves, notes, recovery, and saved-session state remain outside this schema.

## What Selection Guarantees

- Selection never crosses strategy grades. Only the requested `difficulty` is eligible.
- Rows with `play_count = 0` are selected before any previously played row.
- After every exact-grade row has been used, the lowest `play_count` wins; the oldest `last_played_at` breaks unequal recency, and randomness breaks remaining ties.
- Selection and the increment of `play_count`/`last_played_at` happen in one atomic SQLite statement. A row is returned only after its played state is durable.
- Imports and batch generation insert unplayed rows. Deduplication never clears existing play history.
- Explicit `--input` games and `--resume` sessions do not read or mutate puzzle acquisition history.

The acquisition policy uses the full pool before reuse, keeps reuse balanced over time, and avoids claiming that a puzzle was completed merely because it was selected.

## How Play Chooses a Source

Default play keeps the current source order:

1. Attempt bounded generation for the requested grade.
2. If generation matches, store the generated puzzle and mark it played because it is the selected game.
3. If generation misses, store that candidate unplayed and atomically acquire an exact-grade database puzzle.
4. If no exact-grade row exists or the database is unavailable, use the generated mismatch and mark it played when storage is available. Preserve the existing explicit actual-grade warning.
5. If generation completes no puzzle and no database puzzle is available, preserve the current error.

`cmd.generateWithFallbackTo` uses `db.DB.AcquireForPlay` for fallback selection. Keeping acquisition and mutation in `db` prevents callers from accidentally selecting without marking.

## Deterministic User and Test Boundary

Root play provides `--db <path>` and `--from-db`:

- `--db` selects the play database and defaults to the existing XDG path.
- `--from-db` skips generation and atomically acquires an exact-grade puzzle.
- `--from-db` returns a clear error when that grade has no stored puzzle.
- `--from-db` is mutually exclusive with `--input` and `--resume`.

The explicit database source is useful to players who want an offline stored puzzle and gives built-binary E2E a public, deterministic database boundary. No hidden seed or test-only switch is introduced.

## Migration and Indexing

`db.DB.migrate` maintains a non-null `play_count` column with a zero default, a nullable `last_played_at` timestamp, and an acquisition index ordered by difficulty, count, and timestamp. Migration inspects the existing table before each additive change because SQLite lacks a portable conditional column-addition form.

Existing rows receive a zero count and no timestamp, so the first post-upgrade cycle uses every existing exact-grade puzzle before reuse. No schema-version table, destructive rebuild, backfill timestamp, or normalization change is required for this increment.

## Failure and Concurrency Boundaries

- The atomic write statement serializes concurrent acquisitions so two callers cannot both return the same previously unplayed row.
- Configure a bounded SQLite busy timeout; do not wait indefinitely for a writer.
- If acquisition or played-state persistence fails, default play follows its existing generated fallback. `--from-db` reports the database error because it has no permitted alternate source.
- Statistics continue to report stored puzzle counts. `.aidoc/designs/database-play-statistics.md` defines the separately reviewed acquisition/completion statistics and reset increment.

Minimum-clue policy, large-import progress behavior, and broader concurrent SQLite stress remain separate follow-ups.