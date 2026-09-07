---
domain: Designs
status: Active
entry_points:
  - cmd/root.go
  - db/db.go
dependencies:
  - .aidoc/designs/e2e-test-scenarios.md
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/database-puzzle-selection.md
  - .aidoc/designs/database-play-statistics.md
  - .aidoc/designs/database-concurrency.md
---

# E2E Database Scenarios

The database scenario catalog protects root-command database composition, played-state acquisition, acquisition/completion statistics, and concurrent SQLite reliability through public commands and focused deterministic seams.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/e2e-test-scenarios.md` | E2E discovery, isolation, and automation entry points |
| `.aidoc/designs/database-puzzle-selection.md` | Exact-grade acquisition and recycling contract |
| `.aidoc/designs/database-play-statistics.md` | Completion, statistics, and history-reset contract |
| `.aidoc/designs/database-concurrency.md` | Mixed-workload, lock-bound, and multi-process contract |
| `AGENT.md` | Contributor verification requirements |

## Why This Boundary

Database behavior crosses generation, classification, persistence, and startup. Deterministic cases use the public `--from-db` boundary; generated fallback accounting and deliberate lock exhaustion use the narrowest deterministic package seam.

## Database and Fallback

### Auto-Store and Fallback
**Action:** Run the matching `scripts/e2e_cli.py` cases for automatic play storage, exact-grade database fallback, and an empty requested grade.
**Expected:** The selected puzzle is stored under the isolated XDG database. Exact-grade fallback avoids a mismatch warning; an unavailable grade reports the actual generated grade without mutating another database.

### Input and Command Boundaries
**Action:** Run the multiple-solution input and root-help cases in `scripts/e2e_cli.py`.
**Expected:** Multiple-solution input warns and starts with the first solution. Root help exposes the `generate`, `import`, and `tui` commands.

## Played-State Acquisition

### Never-Played First and Balanced Reuse
**Setup:** Import two distinct puzzles with one exact strategy grade.
**Action:** Acquire three puzzles through `sudoku --from-db --level <grade> --db <path>`.
**Expected:** Each row is selected before either repeats; later selections keep acquisition counts within one.

### In-Place Migration
**Setup:** Create a pre-played-state database and open it with the current binary.
**Expected:** Migration preserves puzzles and classifications, initializes history, and permits exact-grade acquisition.

### Source and Failure Boundaries
**Action:** Exercise an empty grade, custom database path, conflicting `--input` or `--resume`, and deterministic generated-fallback accounting.
**Expected:** Stable errors do not generate substitutes or mutate unrelated databases. Only the puzzle selected for play gains acquisition history; explicit input and restored sessions leave acquisition history unchanged.

## Acquisition and Completion Statistics

### Separate History Dimensions
**Setup:** Acquire one normalized fixture twice, quit one run unfinished, and complete the other with player actions.
**Action:** Run `sudoku db stats --db <path>`.
**Expected:** The grade and overall rows report two acquisitions and one completion without labeling either as abandonment.

### Completion Boundaries
**Action:** Exercise quit, save/recovery, invalid moves, automatic solve, player completion, hint-assisted completion, undo/re-solve, and an already solved restored session.
**Expected:** Only player and hint-assisted completion increment once per run; automatic solve and non-completion actions do not.

### Identity, Migration, and Snapshot
**Action:** Open a pre-completion-schema database, submit digit-relabelled equivalent fixtures, request all-grade and filtered statistics, and update counters concurrently in a focused package test.
**Expected:** Migration preserves existing history and initializes completion history. Equivalent fixtures share one row. Each snapshot is internally consistent, empty timestamps render as `-`, and invalid grades fail before database work.

### Explicit Reset Scope
**Action:** Preview acquisition, completion, and all-history resets; cancel once; then confirm with `--yes`, with and without a grade filter.
**Expected:** Preview identifies database, scope, filter, rows, and counters. Reset changes only the selected count/timestamp pairs while preserving puzzle data, saves, recovery records, and the other history dimension.

### Frontend and Failure Consistency
**Action:** Complete a puzzle through the line CLI, TUI, and HTTP API where applicable; force completion-write and reset failures separately.
**Expected:** Every frontend applies one completion rule. Completion-write failure leaves gameplay solved with a warning; reset failure exits non-zero without partial mutation.

## Concurrent SQLite Reliability

### Multi-Process Import and Read
**Action:** Run overlapping fixed-fixture imports and bounded `sudoku db stats` readers against one temporary database.
**Expected:** Processes finish without hangs, imports produce exactly the unique normalized rows, and every statistics response is internally consistent.

### Post-Contention Acquisition and Integrity
**Action:** Acquire fixed-grade rows after writers close, inspect counters, reopen the database, and run SQLite `PRAGMA quick_check`.
**Expected:** Acquisition totals match successful selections, reuse remains balanced, committed counters survive reopen, and integrity returns `ok`.

### Deterministic Lock Bound
**Action:** Hold a write transaction in a package test and write through another handle before and after releasing the lock.
**Expected:** The blocked write returns within the configured five-second bound without partial mutation; the later write succeeds.

## Deferred Database Scenarios

Measured large-import behavior and minimum-clue or uniqueness policy remain separate decisions with their own future acceptance scenarios.
