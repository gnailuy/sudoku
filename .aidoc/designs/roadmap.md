---
domain: Designs
status: Active
entry_points:
  - .github/workflows/ci.yml
  - scripts/e2e_api.py
  - scripts/e2e_cli.py
  - scripts/e2e_tui.py
  - scripts/coverage_report.py
  - solver/config.go
dependencies:
  - .aidoc/designs/e2e-test-scenarios.md
  - .aidoc/designs/difficulty-model.md
  - .aidoc/designs/difficulty-calibration.md
  - .aidoc/designs/future-directions.md
  - .aidoc/designs/database-puzzle-selection.md
  - .aidoc/designs/database-play-statistics.md
  - .aidoc/designs/database-concurrency.md
---

# Roadmap

The roadmap contains only ongoing maintenance and approved future work. Current product contracts live in their dedicated design documents; Git records completed delivery history.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/e2e-test-scenarios.md` | Canonical black-box behavior catalog and current automation pointers |
| `.aidoc/designs/difficulty-model.md` | Strategy-grade invariants and the calibration boundary |
| `.aidoc/designs/difficulty-calibration.md` | Strategy measurement methodology, report artifacts, and review decisions |
| `.aidoc/designs/future-directions.md` | Deliberately non-priority product and production directions |
| `.aidoc/designs/web-api.md` | Current HTTP contract, security boundary, and deployment defaults |
| `.aidoc/designs/database-puzzle-selection.md` | Current played-state selection and migration boundary |
| `.aidoc/designs/database-play-statistics.md` | Current completion counters, statistics, and explicit history reset |
| `.aidoc/designs/database-concurrency.md` | Current database reliability design: connection policy and deterministic mixed-workload stress |

## Why Quality Gates Remain

The current product spans terminal interaction, durable local state, generation, SQLite, and a network API. Independent CI and black-box lanes keep changes to any one boundary from weakening the established baseline.

The quality policy prioritizes repeatable evidence over speculative behavior. Reliable CI and deterministic black-box tests make calibration results meaningful and reduce the risk of changing database selection or import policy.

## Current and Future Work

### Maintain CI and Black-Box E2E

Pull-request CI separates unit tests, race detection, vet, lint, API contract checks, API E2E, and TUI E2E into independent jobs. Independent jobs keep a slow boundary from hiding fast failures and allow every gate to report its own timeout and diagnostics.

Storage and command-wiring tests use fixed classified puzzles through the `cmd.batchGenerateWith` generation seam. Real randomized generation remains covered in `generator`, while `cmd` tests prove reporting and SQLite composition without waiting for a target difficulty. Solver fallback fixtures use `solver.Backtracker.SolveDeterministic`; randomized `solver.Backtracker.Solve` remains available for diverse full-board generation without making race-test duration depend on a lucky search path. These boundaries keep `go test -race -count=1 ./...` viable as a mandatory gate without weakening generation or fallback coverage.

The API, TUI, and line-CLI harnesses build and execute the real binary with isolated temporary state. The line-CLI lane covers parsing, gameplay/history, durable sessions, import normalization and deduplication, bounded generation, and SQLite-visible composition. The public `--from-db` boundary deterministically covers exact-grade acquisition, migration, and balanced reuse; generated-fallback accounting uses focused package coverage.

### Maintain Boundary Unit and Integration Coverage

- `webapi` tests cover malformed input, lifecycle and persistence failures, exact authentication and CORS boundaries, revision conflicts, concurrent sessions, and process-lock exclusion.
- `cmd` tests cover deterministic generation/storage composition, session restoration and source rejection, API startup-policy validation, and process-lock ownership. CLI dispatch and Cobra workflows remain in built-binary E2E instead of being reimplemented in test-only controllers.
- The unit job records a cross-package Go coverage profile and publishes a package summary through `scripts/coverage_report.py`. Coverage is review evidence rather than a pass/fail threshold.
- Review prioritizes meaningful branches by risk: `webapi`, `cmd`, `recovery`, and `sessionfile` failure/lifecycle paths; `game` state invariants; and generator/solver correctness. Low-risk wrappers and generated boundary code do not justify artificial tests solely to raise a percentage.
- Every fixed defect gains a regression test at the narrowest layer that proves the behavior.

### Preserve the Calibration Contract

Calibration runs from the stable CI baseline with deterministic classifier semantics and versioned mixed corpora. Easy through Evil are canonical strategy grades rather than predictions of player experience; score orders puzzles within a grade, clue count guides generation, and strategy-unsolved remains separate. `.aidoc/designs/difficulty-calibration.md` owns the current evidence, corpus contract, reproducibility metadata, measurements, and remaining decision gates.

Calibration output remains local and telemetry-free. The 101-record corpus separates target-alignment failures from strategy-inventory stalls. Batch generation remains best-effort and stores each completed puzzle under its actual grade; per-puzzle wall-clock budgets are hard deadlines. Interactive play first uses an exact requested-grade result or database puzzle, then explicitly reports any actual-grade fallback. Technique-inventory changes remain separate; human data may support a later empirical player-difficulty layer but is not a prerequisite for strategy calibration.

### Strengthen Concurrent SQLite Reliability

`.aidoc/designs/database-concurrency.md` defines the active database increment: apply SQLite connection policy predictably across pooled connections and prove deterministic mixed access from goroutines, independent handles, and built-binary processes. The work preserves the schema and commands.

### Keep Later Database Decisions Separate

Measured large-import behavior and minimum-clue or uniqueness policy remain independently reviewed future decisions. Neither concern expands the concurrent SQLite reliability scope.

## Maintained Stabilization Gates

- Pull-request CI runs race detection, API E2E, TUI PTY E2E, and automated line-CLI/command E2E.
- Boundary failures and lifecycles in `webapi` and `cmd` have focused regression coverage without duplicating black-box command tests.
- CI publishes reproducible package coverage for risk-based review without a global threshold.
- All black-box lanes run against built artifacts with isolated state and deterministic fixtures.
- Calibration evidence and policy proposals begin from a green, stable baseline.
