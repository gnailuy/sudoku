---
domain: Designs
status: Active
entry_points:
  - cmd/import.go
  - db/puzzle.go
  - scripts/e2e_cli.py
dependencies:
  - .aidoc/designs/database-concurrency.md
  - .aidoc/designs/e2e-import-scenarios.md
  - .aidoc/designs/database-puzzle-selection.md
---

# Large Puzzle Imports

Large imports remain streaming, resumable through safe reruns, and explicit about partial success. A measured internal transaction batch replaces one transaction per row without turning import into an all-or-nothing operation or changing puzzle validation, normalization, classification, deduplication, and history semantics.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/database-concurrency.md` | Defines lock patience, connection policy, and integrity requirements under concurrent access |
| `.aidoc/designs/database-puzzle-selection.md` | Defines normalized identity and the history that duplicate imports preserve |
| `.aidoc/designs/e2e-import-scenarios.md` | Owns built-binary acceptance for success, interruption, rerun, and failure |
| `.aidoc/designs/roadmap.md` | Sequences this database increment before puzzle-admission policy |

## Why Large Imports Need a Separate Contract

The importer currently scans incrementally but validates, classifies, and inserts one puzzle at a time. Per-row commits provide natural partial success, yet their throughput cost is unmeasured, database failures are reported as skipped lines, and interruption has no defined boundary. A whole-file transaction would improve write throughput at the cost of long locks, unbounded rollback scope, and lost work after interruption.

The large-import contract optimizes only after reproducible measurements. Puzzle classification may dominate database writes, so transaction batching must be justified by end-to-end evidence rather than SQLite assumptions.

## Measurement Contract

A repeatable local harness measures the built binary against immutable, hash-identified input fixtures. The fixture matrix includes valid unique puzzles, normalized duplicates, comments, and invalid records at representative small, medium, and large line counts.

Each candidate runs at least three times against a fresh temporary database on the same machine. The report records the repository revision, Go version, operating system, fixture hash and composition, transaction batch size, wall time, puzzles per second, peak resident memory, stored/duplicate/invalid totals, output database size, and `PRAGMA quick_check` result.

The comparison includes current per-row insertion and bounded candidate batch sizes. The implementation selects the smallest fixed batch whose median end-to-end throughput is within ten percent of the best measured candidate and whose write transaction remains comfortably below the existing five-second busy timeout. CI verifies semantics and bounded resources; it does not enforce machine-dependent throughput thresholds.

## Streaming and Resource Boundaries

- Import reads one line at a time and retains at most one classified transaction batch. Memory therefore grows with the fixed internal batch and maximum line length, not file length.
- An input line is limited to 4 KiB before normalization. An oversized line is invalid, produces its line number, and does not terminate later processing.
- A regular input file larger than 1 GiB is rejected before database mutation. The import command continues to require a seekable file path; stdin, directories, devices, and unbounded streams are outside this increment.
- Classification occurs before opening the write transaction. The transaction contains only normalized, classified puzzle records and remains short enough to coexist with database readers.
- Transaction batch size is an internal measured constant, not a public tuning flag. A command-line knob would expose storage internals without a stable user need.

## Commit and Partial-Success Semantics

Each batch is atomic. Successful batches remain committed after a later read error, database error, or interruption; the active uncommitted batch contributes no stored rows. Invalid or unsolvable lines remain non-fatal records and do not prevent valid records in the same pending batch from being committed.

Database insertion returns per-record inserted-or-duplicate outcomes so the final counters remain exact. A duplicate preserves its existing source, acquisition history, and completion history. A database error rolls back the active batch, stops the import, prints the partial report, and exits non-zero rather than misclassifying the failure as an invalid line or duplicate.

Ctrl-C requests cancellation through the command context. The importer stops accepting new lines, rolls back the active batch, prints the committed partial report, and exits with the conventional interrupted status. A database call may take up to the existing bounded lock wait before cancellation completes; the importer never retries or spins indefinitely.

Rerunning the same file is the supported resume mechanism. Existing normalization and primary-key deduplication skip every previously committed puzzle, while the interrupted or failed batch is reconsidered from the input file. No checkpoint, sidecar, schema column, or import-session table is introduced.

## Progress and Final Reporting

Progress is emitted after every committed batch and at a bounded time interval while classification is slower than commits. Each update reports processed, valid, stored, duplicate, invalid, elapsed-time, and average-rate fields. Progress goes to stderr so stdout retains the final report boundary; tests match counters and field presence without asserting unstable elapsed-time or rate values.

The final report is printed on success, interruption, read failure, and database failure. The report separates committed stored/duplicate totals from scanned valid/invalid totals and identifies an uncommitted pending count when work was rolled back. Success exits zero, malformed records alone still permit success, and operational failures exit non-zero.

## Database Boundary

`db.DB` exposes a bounded batch insertion operation that owns one SQLite transaction and returns ordered inserted-or-duplicate outcomes. The operation preserves `INSERT OR IGNORE` identity semantics and rolls back the complete batch on any operational error. `cmd.runImport` owns scanning, validation, classification, cancellation, progress, and aggregation.

The batch API does not accept raw lines, invoke solvers, print progress, retry locks, or choose its own batch size. Keeping classification outside `db` preserves package responsibilities and minimizes write-lock duration.

## Verification

Focused package tests cover ordered batch outcomes, duplicate history preservation, rollback after an injected failure, cancellation before commit, oversized lines, file-type and file-size rejection, exact partial counters, and bounded memory ownership. Existing import normalization and classification tests remain authoritative for puzzle semantics.

Built-binary scenarios cover a multi-batch success, visible progress, Ctrl-C after at least one committed batch, rerun completion without duplicate mutation, held-lock failure with a partial report, oversized input rejection, and final row/counter totals plus `PRAGMA quick_check`. The interruption harness synchronizes on a committed progress update instead of sleeping or depending on machine speed.

## Compatibility and Exclusions

Existing valid files, source labels, database paths, puzzle identity, classification, and final report fields remain compatible. Additional progress and partial-report fields are additive. The schema and five-second SQLite busy timeout do not change.

Archive formats, recursive directory import, network sources, stdin, background jobs, durable checkpoints, configurable batch sizes, corruption repair, minimum-clue policy, and uniqueness-admission policy remain outside this increment.
