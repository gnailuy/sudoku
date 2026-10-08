---
domain: Designs
status: Active
entry_points:
  - cmd/play.go
  - db/catalog.go
  - db/db.go
  - db/puzzle.go
dependencies:
  - .aidoc/designs/database-catalog.md
  - .aidoc/designs/difficulty-model.md
  - .aidoc/designs/database-play-statistics.md
  - .aidoc/designs/database-concurrency.md
  - .aidoc/designs/e2e-database-scenarios.md
---

# Database Puzzle Selection

Puzzle acquisition prefers an exact strategy grade, avoids immediate repeats, and remains useful after every stored puzzle has been played. A database acquisition atomically selects and marks one puzzle; never-played puzzles come first, then the least-played and least-recently-played puzzle. The disposable legacy table must be explicitly rebuilt before the versioned catalog can be used.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/database-catalog.md` | Defines base-puzzle identity, provenance, play-run state, and rebuild semantics |
| `.aidoc/designs/difficulty-model.md` | Defines the exact strategy-grade contract used by selection |
| `.aidoc/designs/database-play-statistics.md` | Keeps completion counters and history reset separate from acquisition semantics |
| `.aidoc/designs/database-concurrency.md` | Extends atomic acquisition into a mixed-handle and multi-process reliability contract |
| `.aidoc/designs/e2e-database-scenarios.md` | Owns black-box acceptance scenarios for acquisition and migration |
| `.aidoc/designs/roadmap.md` | Maintains the quality gates that protect this selection baseline |

## Why Track Acquisition

Random lookup can return the same puzzle repeatedly while other exact-grade puzzles remain unused. A permanent played/not-played filter avoids repeats only until the pool is exhausted, after which the database stops helping. Selection therefore needs durable history and an explicit recycling policy.

The base-puzzle catalog is a local puzzle pool, while presentation-specific identity belongs to separate play-run records. Puzzle classification is computed from the symmetry- and digit-canonical stored board, so equivalent presentations converge on one authoritative grade and history row. Acquisition records that a base puzzle was chosen; a linked play run preserves the shown board and completion status, while moves, notes, recovery payloads, and saved-session state remain outside the catalog schema.

## What Selection Guarantees

- Selection never crosses strategy grades. Only the requested `difficulty` is eligible; Evil additionally requires membership in the materialized `evil-v1` cohort.
- Rows with `play_count = 0` are selected before any previously played row.
- After every exact-grade row has been used, the lowest `play_count` wins; the oldest `last_played_at` breaks unequal recency, and randomness breaks remaining ties.
- Selection and the increment of `play_count`/`last_played_at` happen in one atomic SQLite statement. A row is returned only after its played state is durable.
- Imports and batch generation insert unplayed rows. Deduplication never clears existing play history.
- Explicit `--input` games and `--resume` sessions do not read or mutate puzzle acquisition history.

The acquisition policy uses the full pool before reuse, keeps reuse balanced over time, and avoids claiming that a puzzle was completed merely because it was selected.

## How Play Chooses a Source

Default play chooses source order from verified catalog supply:

1. Hard, Expert, and Evil atomically acquire an exact-grade catalog puzzle before attempting generation. These grades have maintained exact-grade supply, so default starts avoid an unnecessary bounded-generation delay and preserve requested/actual agreement.
2. Easy and Medium attempt bounded generation first because the maintained catalog has no exact-grade supply for those grades.
3. If Hard or Expert catalog-first acquisition is empty or unavailable, play reports the failed catalog boundary and falls back to bounded generation. Evil fails closed when its approved cohort cannot serve and never enters generation.
4. If generation misses, play stores that candidate unplayed and atomically acquires an exact-grade database puzzle when one is available.
5. If no exact-grade row exists or the database is unavailable, play may use the generated mismatch, marks it played when storage is available, and preserves the explicit actual-grade warning. If generation completes no puzzle, play reports an error.

`cmd.createSession` owns source ordering, while `cmd.generateWithFallbackTo` uses `db.DB.AcquireForPlay` for post-generation fallback selection. Keeping acquisition and mutation in `db` prevents callers from accidentally selecting without marking.

## Deterministic User and Test Boundary

Root play provides `--db <path>` and `--from-db`:

- `--db` selects the play database and defaults to the existing XDG path.
- `--from-db` skips generation and atomically acquires an exact-grade puzzle.
- `--from-db` returns a clear error when that grade has no stored puzzle.
- `--from-db` is mutually exclusive with `--input` and `--resume`.

The explicit database source is useful to players who want an offline stored puzzle and gives built-binary E2E a public, deterministic database boundary. No hidden seed or test-only switch is introduced.

## Schema and Indexing

`base_puzzles` stores the authoritative catalog row and acquisition aggregates. `serving_cohorts` binds a named rule to immutable evidence metadata, and `serving_cohort_members` links approved canonical IDs without duplicating puzzle content. `puzzle_provenance` records independently traceable sources, while `play_runs` owns exact presentation-specific state. `.aidoc/designs/database-catalog.md` defines stable identity and the destructive schema-version boundary.

The acquisition index orders base-puzzle rows by difficulty, count, and timestamp. Legacy `puzzles` tables are never backfilled because their content lacks symmetry-canonical provenance; operators use the explicit confirmed rebuild command and build a replacement catalog from the commit- and hash-pinned external source through `scripts/import_sudoku_exchange_diabolical.sh`.

## Failure and Concurrency Boundaries

- The atomic write statement serializes concurrent acquisitions so two callers cannot both return the same previously unplayed row.
- Configure a bounded SQLite busy timeout; do not wait indefinitely for a writer.
- If Hard or Expert acquisition fails, default play follows its existing generated fallback. Evil cohort failure and every `--from-db` failure return directly because neither boundary permits an alternate source.
- Statistics continue to report stored puzzle counts. `.aidoc/designs/database-play-statistics.md` defines the separately reviewed acquisition/completion statistics and reset increment.

Puzzle-admission changes, including minimum-clue and uniqueness policy, require a separately approved product need. The current identity and history semantics remain the maintained database baseline.
