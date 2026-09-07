---
domain: Designs
status: Active
entry_points:
  - db/db.go
  - db/puzzle.go
  - scripts/e2e_cli.py
dependencies:
  - .aidoc/designs/database-puzzle-selection.md
  - .aidoc/designs/database-play-statistics.md
  - .aidoc/designs/e2e-database-scenarios.md
---

# Concurrent SQLite Reliability

Sudoku treats one SQLite file as a safe local coordination boundary for concurrent generation, import, acquisition, completion, statistics, and history reset. The concurrency contract requires explicit connection configuration, deterministic mixed-workload stress tests, and a built-binary multi-process acceptance case. It does not add a daemon, distributed lock, retry queue, or new user-facing flag.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/database-puzzle-selection.md` | Defines atomic exact-grade acquisition and balanced reuse |
| `.aidoc/designs/database-play-statistics.md` | Defines completion, snapshot, and atomic reset semantics |
| `.aidoc/designs/e2e-database-scenarios.md` | Owns black-box multi-process acceptance scenarios |
| `.aidoc/designs/roadmap.md` | Sequences this reliability increment before import-policy changes |

## Why Concurrency Needs an Explicit Contract

The database uses WAL mode, a five-second busy timeout, atomic acquisition statements, transactions for snapshots and resets, and focused concurrent acquisition and completion tests. SQLite pragmas can be connection-scoped, so configuring one pooled connection does not guarantee that every operation receives the same lock policy. Individual-operation coverage also does not establish sustained mixed-access behavior across independent handles and processes.

An explicit connection and contention contract keeps local database behavior predictable without introducing application-level coordination or distributed infrastructure.

## Runtime Contract

- A `db.DB` is safe to share across goroutines. Independent `db.Open` calls and independent Sudoku processes may target the same file.
- Every connection used by Sudoku has WAL mode and a bounded five-second busy timeout. Configuration must not depend on whichever pooled connection executed `Open`.
- Each `db.DB` serializes its own SQLite operations through one pooled connection. Concurrency across handles/processes remains coordinated by SQLite WAL and the busy timeout. This favors a simple, auditable local-store contract over speculative in-process write throughput.
- Writers wait only within the busy-timeout bound. Lock exhaustion returns a contextual error; command layers do not spin or retry indefinitely.
- Reads may complete while another handle has an uncommitted write and must observe a valid committed snapshot.
- Atomic operations retain their existing meaning under contention: acquisition selects and increments one row, completion increments are not lost, statistics rows agree with their overall snapshot, and reset exposes either the state before or after its full transaction.
- Closing and reopening the database preserves committed rows and counters. SQLite integrity checks must report `ok` after the workload.

`db.Open` limits each handle to one pooled connection before migration and configures the busy timeout in the driver data source so a replacement connection retains the policy. No application-level mutex coordinates separate handles or processes.

## Deterministic Stress Model

Package tests use fixed puzzle identifiers and bounded worker counts. They separate phases so every result has an exact oracle rather than accepting “no error” as sufficient evidence:

1. **Open and migration contention:** open several handles together against a new file and against an existing schema. All bounded test opens complete successfully; a separate deliberately held-lock case proves a contextual error returns within the configured bound. A successful reopen produces exactly one valid additive schema.
2. **Insert and deduplication:** concurrent handles insert disjoint rows plus deliberate duplicates. The final row count equals the unique input set, and existing history is never reset by duplicate insertion.
3. **Acquisition and completion:** workers acquire from a fixed exact-grade pool and increment completion on fixed rows. Total counter deltas equal successful operations; never-played-first and balanced reuse invariants remain true.
4. **Snapshot readers:** statistics readers run alongside writers. Every returned per-grade/overall result is internally consistent, even though separate snapshots may represent different committed moments.
5. **Reset contention:** a reset racing another counter update commits as one transaction or fails without partial counter/timestamp changes. The test does not assume which valid serialization wins.
6. **Busy timeout and recovery:** one handle holds a write lock while another attempts a write. The second operation returns within a narrow bound around five seconds, then succeeds after the lock is released.
7. **Durability and integrity:** close all handles, reopen the file, validate exact rows/counters, run `PRAGMA quick_check`, and require `ok`.

Tests use barriers and held transactions to create contention deliberately. They do not depend on scheduler luck, random generation, unbounded loops, sleeps as synchronization, or machine-specific throughput thresholds. The normal suite stays short; a bounded higher-iteration stress case may run under the existing race job but must complete within the job timeout.

## Built-Binary Acceptance

`scripts/e2e_cli.py` runs a deterministic multi-process case against one temporary database:

- start several `sudoku import` processes with overlapping fixed fixture files;
- run `sudoku db stats` readers while imports are active;
- acquire fixed exact-grade rows through `sudoku --from-db` after imports complete;
- verify process exit codes, concise stderr on failure, unique row totals, acquisition totals, and final `PRAGMA quick_check` through Python's standard-library SQLite client.

The harness uses isolated XDG roots, a fixed process count, subprocess deadlines, and guaranteed cleanup. It does not use real puzzle generation as the contention driver because generation duration and produced grade are intentionally variable. Parallel generation remains covered at the deterministic `cmd.batchGenerateWith` seam while its shared-database writes exercise the hardened `db.DB` contract.

## Failure and Compatibility Boundaries

- Existing database files migrate in place; no schema column, file format, or command syntax changes.
- Existing five-second lock patience remains the public behavior. Errors gain operation context but do not expose host-sensitive paths beyond the explicit database path already selected by the user.
- A busy or failed insert is not reported as a duplicate. Successful operation counts remain distinguishable from lock failures.
- No corruption repair, backup system, network filesystem guarantee, distributed coordination, or multi-host support is introduced.
- WAL sidecar files remain SQLite-owned and may exist while the database is open. Tests inspect the database only after responsible processes have closed it.

## Acceptance Criteria

- Connection configuration applies predictably to every operation; one handle cannot escape the busy-timeout policy through pool expansion.
- Deterministic package tests prove mixed inserts, acquisitions, completions, snapshots, reset serialization, bounded lock failure, reopen durability, and `quick_check` integrity.
- The built-binary multi-process database scenario passes three consecutive local runs with no leaked process or state outside its temporary directory.
- `go test -race -count=1 ./...`, vet, lint, API contract, and all built-binary E2E lanes pass.
- Database docs and the E2E matrix describe the maintained concurrency boundary and its explicit exclusions.

## Deferred Decisions

- Large-import progress and batching policy.
- Minimum-clue, uniqueness, and solver-cost admission policy.
- Backup/restore commands, corruption recovery, and operational maintenance.
- Network filesystems, distributed databases, accounts, and multi-tenant storage.
