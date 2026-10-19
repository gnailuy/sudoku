---
domain: Conventions
status: Active
entry_points: []
dependencies: []
---

# Sudoku Documentation Index

This index is the complete registry for durable project knowledge and a set of short routes to the code that proves each contract. Start with `AGENT.md` for repository rules, then choose only the route that matches the task.

## Related Docs

| Document | Relationship |
|---|---|
| `AGENT.md` | Mandatory contributor rules, commands, and review gates |
| `README.md` | Human quick start and supported user workflows |
| `.aidoc/architecture/guidelines.md` | Package boundaries and cross-cutting design constraints |
| `.aidoc/designs/future-directions.md` | Evidence gates for work that is not current behavior |

## Draft Work

| Topic | Owner | Status | Document |
|---|---|---|---|
| Locale-neutral API messages and shared English terminal rendering | Product owner | Draft | `.aidoc/designs/localization-contract.md` |

Draft work is discoverable for planning but is not current-state authority and does not appear in current-state reading chains.

## Reading Chains

### Change Engine, Candidates, or Hints

1. `.aidoc/designs/game-engine.md` — engine state, actions, history, and serialization.
2. `.aidoc/designs/authoritative-candidates.md` — one candidate grid across clients.
3. `.aidoc/designs/hint-presentation-protocol.md` — correction, strategy, preview, and application semantics.
4. `game/contract.go`, `game/game.go`, and `game/hint_plan.go` — authoritative action, orchestration, and hint-plan implementation.
5. Choose the affected client: `.aidoc/designs/cli-sessions.md`, `.aidoc/designs/tui-frontend.md`, or `.aidoc/designs/web-api.md`.

### Change Rating, Generation, or Catalog Acquisition

1. `.aidoc/designs/difficulty-model.md` — canonical strategy-grade meaning.
2. `.aidoc/designs/difficulty-calibration.md` — evidence and measurement limits.
3. `.aidoc/designs/database-puzzle-selection.md` — runtime source order and exact-grade acquisition.
4. `.aidoc/designs/catalog-bootstrap.md` and `.aidoc/designs/evil-cohort.md` — embedded supply and fail-closed Evil membership.
5. `cmd/play.go`, `db/puzzle.go`, and `catalogbundle/bundle.go` — authoritative runtime boundaries.

### Change Guest, Account, or Deployment Behavior

1. `.aidoc/designs/web-api.md` — client-neutral guest and account resources.
2. `.aidoc/designs/user-accounts.md` — identity, ownership, sessions, and claim semantics.
3. `.aidoc/designs/deployment-hardening.md` — artifact, loopback, origin, service, and replacement boundaries.
4. `api/openapi.yaml`, `cmd/api.go`, `cmd/accounts.go`, and `webapi/server.go` — wire, account-runtime composition, and server implementation.
5. `.aidoc/designs/e2e-api-scenarios.md` — black-box acceptance.

### Run Black-Box Acceptance

1. `.aidoc/designs/e2e-test-scenarios.md` — isolation and suite map.
2. Select the affected scenario catalog: play, session, TUI, API, generation, import, database, or calibration.
3. Build the binary and run the named harness under `scripts/`.

## Document Registry

### Architecture and Product Contracts

| Document | Purpose |
|---|---|
| `.aidoc/architecture/guidelines.md` | Package boundaries and dependency direction |
| `.aidoc/designs/game-engine.md` | Authoritative game actions, snapshots, history, and serialization |
| `.aidoc/designs/authoritative-candidates.md` | Single candidate-state and client-presentation contract |
| `.aidoc/designs/hint-presentation-protocol.md` | Renderer-neutral teaching plans and exact application |
| `.aidoc/designs/hint-reference-plans.md` | Canonical examples for placement and elimination plans |
| `.aidoc/designs/cli-sessions.md` | Compact line CLI, hint degradation, save, and resume |
| `.aidoc/designs/tui-frontend.md` | Full-screen interaction and rendering semantics |
| `.aidoc/designs/background-autosave.md` | TUI recovery lifecycle, privacy, and retention |
| `.aidoc/designs/web-api.md` | Guest/account HTTP resources and revision boundaries |
| `.aidoc/designs/user-accounts.md` | Identity, ownership, claim, session, and account security |
| `.aidoc/designs/deployment-hardening.md` | Portable artifact, service, isolation, and replacement contract |
| `.aidoc/designs/localization-contract.md` | Draft locale-neutral messages with deterministic English fallback |
| `.aidoc/designs/future-directions.md` | Evidence gates for unapproved product expansion |

### Rating, Generation, and Persistence

| Document | Purpose |
|---|---|
| `.aidoc/designs/difficulty-model.md` | Solver-relative grades and within-grade score |
| `.aidoc/designs/difficulty-calibration.md` | Reproducible strategy measurement and current evidence |
| `.aidoc/designs/exact-grade-generation-experiment.md` | Offline replenishment experiment and publication boundary |
| `.aidoc/designs/evil-cohort.md` | Frozen exact-Evil membership and fail-closed serving |
| `.aidoc/designs/database-catalog.md` | Stable puzzle identity, provenance, schema, and play-run separation |
| `.aidoc/designs/catalog-bootstrap.md` | Embedded additive catalog release |
| `.aidoc/designs/database-puzzle-selection.md` | Runtime acquisition, source order, and balanced reuse |
| `.aidoc/designs/database-play-statistics.md` | Acquisition/completion counters and explicit reset |
| `.aidoc/designs/database-concurrency.md` | SQLite handle and multi-process reliability contract |

### Black-Box Scenario Catalogs

| Document | Purpose |
|---|---|
| `.aidoc/designs/e2e-test-scenarios.md` | E2E suite ownership, discovery, and isolation |
| `.aidoc/designs/e2e-play-scenarios.md` | Root play and line-command acceptance |
| `.aidoc/designs/e2e-session-scenarios.md` | Explicit save and validated restore |
| `.aidoc/designs/e2e-tui-scenarios.md` | Full-screen TUI and recovery |
| `.aidoc/designs/e2e-api-scenarios.md` | HTTP lifecycle, security, and contract |
| `.aidoc/designs/e2e-generation-scenarios.md` | Generation, experiment, and replenishment |
| `.aidoc/designs/e2e-import-scenarios.md` | Import parsing, provenance, and deduplication |
| `.aidoc/designs/e2e-database-scenarios.md` | Catalog, acquisition, statistics, and concurrency |
| `.aidoc/designs/e2e-calibration-scenarios.md` | Immutable corpus measurement and audit |

## Validation

Run `python3 scripts/validate_aidoc.py` for required frontmatter, allowed status values, list types, portable metadata paths, and metadata targets. Run `python3 -m unittest scripts/test_validate_aidoc.py` to prove that valid metadata passes, invalid metadata fails, and natural-language prose remains outside the script boundary. Mechanical success does not replace code-grounded semantic review or fresh-context navigation evidence.
