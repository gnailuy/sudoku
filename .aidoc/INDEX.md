---
domain: Conventions
status: Active
entry_points: []
dependencies: []
---

# .aidoc/INDEX.md — Discovery Index

The project index provides reading chains for common starting points and a complete document map.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `AGENT.md` | Active repository rules and operator entry point |
| `.aidoc/designs/roadmap.md` | Approved portable build and deployment sequence |
| `.aidoc/designs/deployment-hardening.md` | Canonical artifact, service, isolation, and replacement contract |
| `.aidoc/architecture/guidelines.md` | Package boundaries and cross-cutting design constraints |

## Reading Chains

### Understanding the Architecture
1. `AGENT.md` — global rules, repo layout
2. `.aidoc/architecture/guidelines.md` — design constraints, layer boundaries, solver interface contract
3. `core/candidates.go` — `CandidateSet` bitfield type
4. `core/board.go` — `Board` struct with compute-on-fly `Candidates()` method
5. `solver/solver.go` — `Solver`, `StrategySolver`, `CompleteSolver` interfaces and `Base`
6. `solver/move.go` — `Move` struct (cell + technique + reason)
16. `solver/store.go` — solver registry with typed access
8. `game/game.go` — private session state and compatibility adapters
9. `game/contract.go` — typed actions, detached snapshots, results, and engine errors
10. `game/serialization.go` — versioned complete-session persistence and atomic restoration
11. `cli/controller.go` — line-oriented CLI controller (terminal I/O, commands, display)
12. `sessionfile/session_file.go` — presentation-neutral bounded and atomic session transport
13. `tui/model.go` — Bubble Tea event model and action translation
14. `tui/render.go` — deterministic full-screen renderer
15. `recovery/recovery.go` — private XDG recovery records, validation, retention, and atomic transport

### Understanding Puzzle Generation
1. `.aidoc/designs/difficulty-model.md` — strategy-grade contract, tier invariants, and configuration boundary
2. `.aidoc/designs/difficulty-calibration.md` — strategy measurement contract, evidence, reports, and decision gates
3. `.aidoc/designs/exact-grade-generation-experiment.md` — maintained offline replenishment contract and experiment boundary
4. `replenisher/replenisher.go` — resumable exact-grade batch construction and publication
5. `cmd/replenish.go` — explicit offline replenishment CLI
6. `generationexperiment/runner.go` — immutable jobs, resumable observations, canonical duplicate accounting, and raw-count reports
7. `generationexperiment/executors.go` — seeded baseline and bounded trace-guided mutation adapters
8. `cmd/experiment.go` — explicit isolated experiment CLI
9. `calibration/runner.go` — immutable manifests, reproducibility checks, observations, checkpoints, and reports
10. `calibration/baselines/mixed-generator-alignment-v6/report.md` — current 101-record measurement report
11. `calibration/baselines/mixed-generator-alignment-v6/analysis.md` — generator alignment, trace, budget, and coverage interpretation
12. `calibration/baselines/mixed-imported-expansion-v5/report.md` — preserved imported-stratum expansion results
13. `calibration/baselines/mixed-generated-expansion-v4/report.md` — preserved generated-stratum expansion results
14. `calibration/baselines/mixed-external-expansion-v3/report.md` — preserved expanded external baseline
15. `calibration/baselines/mixed-pilot-v2/report.md` — preserved initial pilot baseline
16. `cmd/calibrate.go` — resumable local measurement command
17. `generator/difficulty.go` — difficulty levels and `StrategySolverKeys`
18. `generator/generator.go` — board generation, cell removal, best-effort generation with limits
19. `generator/options.go` — `Options` and `BestEffortOptions` (time/round limits)
20. `solver/classify.go` — puzzle classification (difficulty tier, score, max technique)
21. `.aidoc/designs/database-catalog.md` — base-puzzle identity, provenance, play-run separation, and rebuild boundary
22. `core/canonical.go` — shared digit and Sudoku-symmetry canonicalization boundary
23. `.aidoc/designs/database-puzzle-selection.md` — current acquisition, recycling, and rebuild contract
24. `.aidoc/designs/database-play-statistics.md` — current completion, statistics, and history-reset contract
25. `.aidoc/designs/database-concurrency.md` — connection policy and deterministic mixed-workload reliability contract
26. `db/db.go` — SQLite database open, schema, and rebuild boundary
27. `db/catalog.go` — provenance and presentation-specific play-run contracts
28. `db/puzzle.go` — catalog CRUD, acquisition, and statistics
29. `cmd/play.go` — fallback flow and auto-store
30. `cmd/generate.go` — batch generation CLI
31. `cmd/import.go` — plain and hash-pinned external import boundary
32. `scripts/import_sudoku_exchange_diabolical.sh` — exact public-domain source pin and import composition
### Understanding the Roadmap
1. `.aidoc/designs/roadmap.md` — maintained replenishment boundary and delivery gates
2. `.aidoc/designs/exact-grade-generation-experiment.md` — offline replenishment, resume, validation, and publication contract
3. `.aidoc/designs/difficulty-model.md` — calibration boundary and strategy-grade invariants
4. `.aidoc/designs/difficulty-calibration.md` — strategy measurement methodology, report contract, and product decisions
5. `.aidoc/designs/deployment-hardening.md` — maintained artifact, service, isolation, access-policy, and replacement contract
6. `.aidoc/designs/e2e-test-scenarios.md` — compatibility and black-box acceptance scenarios
7. `.aidoc/designs/database-catalog.md` — stable catalog identity, provenance, play-run state, and destructive rebuild
8. `.aidoc/designs/database-puzzle-selection.md` — current database acquisition and rebuild boundary
9. `.aidoc/designs/database-play-statistics.md` — current completion, statistics, and explicit reset behavior
10. `.aidoc/designs/database-concurrency.md` — connection policy and mixed-workload reliability contract
11. `.aidoc/designs/future-directions.md` — deferred evidence-gated database, product, hosting, and rating directions
12. `.aidoc/designs/web-api.md` — client-neutral HTTP resources, revisions, recovery, client access, and security boundary
13. `api/openapi.yaml` — canonical OpenAPI 3.1.1 wire contract, schemas, errors, and examples
14. `.aidoc/designs/game-engine.md` — stable engine API, notes, history, and serialization design
15. `.aidoc/designs/background-autosave.md` — recovery lifecycle, privacy, storage, retention, and conflict policy
16. `.aidoc/designs/tui-frontend.md` — current full-screen interaction and rendering semantics
17. `.aidoc/architecture/guidelines.md` — current architecture and solver contract

### Running Black-Box E2E Scenarios
1. `.aidoc/designs/e2e-test-scenarios.md` — discovery map, automation boundaries, and isolation rules
2. `.aidoc/designs/e2e-play-scenarios.md` — root play and game commands
3. `.aidoc/designs/e2e-calibration-scenarios.md` — immutable corpus measurement
4. `.aidoc/designs/e2e-generation-scenarios.md` — generation flags, workers, and storage
5. `.aidoc/designs/e2e-import-scenarios.md` — import parsing and deduplication
6. `.aidoc/designs/e2e-database-scenarios.md` — database composition and fallback
7. `.aidoc/designs/e2e-session-scenarios.md` — notes, explicit save, and restore
8. `.aidoc/designs/e2e-tui-scenarios.md` — pseudo-terminal frontend and recovery
9. `.aidoc/designs/e2e-api-scenarios.md` — HTTP lifecycle and security

### Adding a New Strategy Solver
1. `.aidoc/architecture/guidelines.md` — constraints, interface contract, step-by-step
2. `solver/solver.go` — implement `StrategySolver`
3. `solver/move.go` — return `*Move` from `Apply()`
4. `solver/store.go` — register with `RegisterStrategy()`
5. Write tests in `solver/<name>_solver_test.go`
6. Update `generator/difficulty.go` to reference the new solver key

## Document Map

| Path | Purpose |
|------|---------|
| `AGENT.md` | AI operator entry point — rules and repo layout |
| `.aidoc/INDEX.md` | This file — discovery index and reading chains |
| `.aidoc/architecture/guidelines.md` | Design constraints, layer boundaries, solver contract |
| `.aidoc/designs/difficulty-model.md` | Strategy-grade contract, within-grade scoring, clue guidance, and calibration boundary |
| `.aidoc/designs/difficulty-calibration.md` | Strategy calibration methodology, corpus contract, evidence, reports, and decision gates |
| `.aidoc/designs/exact-grade-generation-experiment.md` | Trace-guided mutation comparison, reproducibility contract, and advancement gate |
| `replenisher/replenisher.go` | Offline exact-grade seed mutation, durable resume, validation, and batch publication |
| `cmd/replenish.go` | Expert/Evil offline replenishment CLI boundary |
| `generationexperiment/runner.go` | Isolated manifest-bound experiment harness, durable observations, resume, duplicate accounting, and reports |
| `generationexperiment/executors.go` | Seeded baseline and trace-guided mutation execution adapters |
| `cmd/experiment.go` | Explicit output-only generation experiment command |
| `.aidoc/designs/roadmap.md` | Approved exact-grade generation experiment and maintained delivery gates |
| `.aidoc/designs/deployment-hardening.md` | Backend artifact, service, isolation, access-policy, and failure contract |
| `.aidoc/designs/database-catalog.md` | Stable base-puzzle identity, provenance, play-run separation, and destructive schema rebuild |
| `.aidoc/designs/database-puzzle-selection.md` | Current exact-grade acquisition, played-state recycling, migration, and acceptance contract |
| `.aidoc/designs/database-play-statistics.md` | Current completion counters, acquisition/completion statistics, and explicit history reset |
| `.aidoc/designs/database-concurrency.md` | SQLite connection policy, mixed-workload stress, and multi-process acceptance contract |
| `.aidoc/designs/future-directions.md` | Non-priority product and production directions with decision gates |
| `.aidoc/designs/web-api.md` | Contract-first OpenAPI workflow, resources, revisions, recovery, client access, and network security boundary |
| `.aidoc/designs/background-autosave.md` | Background autosave lifecycle, privacy, retention, and conflict design |
| `.aidoc/designs/automatic-candidates.md` | Automatic-candidate engine contract, TUI interaction, and rendering constraints |
| `.aidoc/designs/tui-frontend.md` | TUI interaction model, persistence policy, rendering, and dependency boundaries |
| `.aidoc/designs/cli-sessions.md` | CLI manual notes, rendering, save, and resume design |
| `.aidoc/designs/game-engine.md` | Engine API, notes, unified history, snapshots, and serialization design |
| `.aidoc/designs/e2e-test-scenarios.md` | E2E discovery map, automation boundaries, and isolation rules |
| `.aidoc/designs/e2e-play-scenarios.md` | Root play and game-command acceptance scenarios |
| `.aidoc/designs/e2e-calibration-scenarios.md` | Calibration acceptance scenarios |
| `.aidoc/designs/e2e-generation-scenarios.md` | Generation acceptance scenarios |
| `.aidoc/designs/e2e-import-scenarios.md` | Import acceptance scenarios |
| `.aidoc/designs/e2e-database-scenarios.md` | Database composition and fallback scenarios |
| `.aidoc/designs/e2e-session-scenarios.md` | Manual-note and durable-session scenarios |
| `.aidoc/designs/e2e-tui-scenarios.md` | Full-screen TUI and recovery scenarios |
| `.aidoc/designs/e2e-api-scenarios.md` | HTTP lifecycle, security, and contract scenarios |
| `api/openapi.yaml` | Canonical OpenAPI 3.1.1 HTTP wire contract |
| `README.md` | Human-facing project summary |
| `cmd/root.go` | Cobra root command and shared state |
| `cmd/play.go` | Interactive play mode, fallback flow, auto-store |
| `cmd/session.go` | Shared CLI/TUI session startup and restore validation |
| `cmd/tui.go` | Opt-in full-screen TUI command and terminal lifecycle |
| `cmd/api.go` | API flags, dependency wiring, and signal-aware server shutdown |
| `webapi/server.go` | HTTP security boundary, session registry, recovery, and engine translation |
| `webapi/generated.go` | Generated OpenAPI models and strict server interface |
| `tui/model.go` | TUI event model, focus, modes, confirmations, and persistence |
| `tui/render.go` | Deterministic color-independent board rendering |
| `recovery/recovery.go` | Private XDG recovery records, discovery, validation, retention, and deletion |
| `sessionfile/session_file.go` | Bounded reads and atomic mode-0600 session writes |
| `calibration/runner.go` | Immutable corpus manifests, append-only observations, resumable checkpoints, and derived reports |
| `calibration/testdata/mixed-pilot-v2.json` | Immutable traceable mixed-corpus pilot manifest |
| `calibration/baselines/mixed-pilot-v2/report.md` | Preserved initial pilot baseline and statistical limitations |
| `calibration/testdata/mixed-external-expansion-v3.json` | Immutable source-order external-stratum expansion manifest |
| `calibration/baselines/mixed-external-expansion-v3/report.md` | Preserved external expansion evidence and limitations |
| `calibration/testdata/mixed-generated-expansion-v4.json` | Immutable sequential generated-stratum expansion manifest |
| `calibration/baselines/mixed-generated-expansion-v4/report.md` | Preserved generated expansion evidence and limitations |
| `calibration/testdata/mixed-imported-expansion-v5.json` | Immutable source-order imported-stratum expansion manifest |
| `calibration/baselines/mixed-imported-expansion-v5/report.md` | Preserved imported expansion evidence and limitations |
| `calibration/testdata/mixed-generator-alignment-v6.json` | Current immutable generator-alignment and coverage manifest |
| `calibration/baselines/mixed-generator-alignment-v6/report.md` | Current deterministic 101-record measurement report |
| `calibration/baselines/mixed-generator-alignment-v6/analysis.md` | Generator target, trace, budget, and strategy-coverage interpretation |
| `cmd/calibrate.go` | Local difficulty measurement CLI boundary |
| `cmd/generate.go` | Batch generation CLI (parallel workers, progress, report) |
| `cmd/import.go` | Plain and hash-pinned external import, canonicalization, reclassification, provenance, and report boundary |
| `core/canonical.go` | Shared canonical representative across digit relabelling and Sudoku-preserving symmetries |
| `scripts/import_sudoku_exchange_diabolical.sh` | Exact public-domain source download pin and import composition |
| `scripts/e2e_cli.py` | Built-binary line CLI, session, calibration, import, generation, and SQLite E2E harness |
| `scripts/e2e_api.py` | Built-binary HTTP lifecycle E2E harness |
| `scripts/e2e_tui.py` | Built-binary PTY TUI and recovery E2E harness |
| `scripts/coverage_report.py` | Package-level Go coverage summary for risk-based CI review |
| `db/db.go` | SQLite puzzle database — open, close, schema migration |
| `db/puzzle.go` | Puzzle CRUD, random query by difficulty, statistics |
| `game/contract.go` | Stable engine actions, snapshots, results, and typed errors |
| `game/serialization.go` | Versioned JSON session serialization, validation, and restoration |
| `solver/classify.go` | Puzzle classification — difficulty tier, score, max technique |
