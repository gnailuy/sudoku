---
domain: Designs
status: Active
entry_points:
  - scripts/e2e_cli.py
  - scripts/e2e_tui.py
  - scripts/e2e_api.py
dependencies:
  - .aidoc/INDEX.md
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/web-api.md
---

# E2E Test Scenarios

The E2E index maps every black-box acceptance boundary to a focused scenario catalog and executable harness. `AGENT.md` owns the contributor verification requirements.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/INDEX.md` | Project discovery index and reading chains |
| `.aidoc/designs/game-engine.md` | Engine actions and state exposed through frontends |
| `.aidoc/designs/cli-sessions.md` | Manual-note rendering and persistence transport contract |
| `.aidoc/designs/web-api.md` | HTTP lifecycle, security, and contract boundary |
| `AGENT.md` | Required verification discipline for feature and bug-fix PRs |

## Why the Catalog Is Split

End-to-end tests treat the built Sudoku binary as a user would, outside Go package boundaries. Focused catalogs keep each user-facing contract reviewable and allow contributors to find applicable scenarios without scanning one monolithic checklist. Package tests support these scenarios but never replace them.

## Scenario Catalog

| Boundary | Scenario document | Automated entry point |
|----------|-------------------|-----------------------|
| Root play and game commands | `.aidoc/designs/e2e-play-scenarios.md` | `scripts/e2e_cli.py` |
| Calibration | `.aidoc/designs/e2e-calibration-scenarios.md` | `scripts/e2e_cli.py` |
| Puzzle generation | `.aidoc/designs/e2e-generation-scenarios.md` | `scripts/e2e_cli.py` |
| Puzzle import | `.aidoc/designs/e2e-import-scenarios.md` | `scripts/e2e_cli.py` |
| Database composition, acquisition, and play statistics | `.aidoc/designs/e2e-database-scenarios.md` | `scripts/e2e_cli.py`, `scripts/e2e_tui.py`, and `scripts/e2e_api.py` |
| Notes and durable CLI sessions | `.aidoc/designs/e2e-session-scenarios.md` | `scripts/e2e_cli.py` |
| Full-screen TUI and recovery | `.aidoc/designs/e2e-tui-scenarios.md` | `scripts/e2e_tui.py` |
| HTTP API lifecycle | `.aidoc/designs/e2e-api-scenarios.md` | `scripts/e2e_api.py` and `scripts/check-api-contract.sh` |

## Automation and Isolation

Every automated black-box entry point builds or receives the repository binary and owns isolated temporary XDG roots. `scripts/e2e_cli.py` covers root play, sessions, calibration, generation, import, and SQLite composition. `scripts/e2e_tui.py` owns pseudo-terminal behavior and recovery. `scripts/e2e_api.py` owns the running HTTP lifecycle, while `scripts/check-api-contract.sh` validates OpenAPI compatibility.

The harnesses use fixed puzzle fixtures whenever deterministic assertions matter. Generation smoke tests bound real generation and verify command, worker, and database composition without asserting a random difficulty result. The public `--from-db` boundary provides deterministic exact-grade acquisition and migration coverage; focused package tests cover generated-fallback accounting. Play-statistics coverage spans the line CLI, TUI, and API so completion semantics cannot drift between frontends.

## Running the Catalogs

Build once, isolate persistent paths, then run the applicable harnesses:

Run `go build -o sudoku .`, create a temporary `SUDOKU_E2E_DIR`, point `XDG_DATA_HOME` and `SUDOKU_DB` into that directory, then invoke each applicable Python harness with `./sudoku`.

Root play commands auto-store under `XDG_DATA_HOME`; generate and import commands use `--db $SUDOKU_DB`. Read the linked catalog before manual execution because some boundaries add setup such as `XDG_STATE_HOME`, a pseudo-terminal, or API listener configuration.
