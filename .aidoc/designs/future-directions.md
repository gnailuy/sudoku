---
domain: Designs
status: Active
entry_points:
  - cmd/api.go
  - cmd/import.go
  - db/puzzle.go
  - solver/classify.go
dependencies:
  - .aidoc/designs/roadmap.md
  - .aidoc/designs/database-concurrency.md
  - .aidoc/designs/difficulty-calibration.md
  - .aidoc/designs/web-api.md
---

# Future Directions

This document is the only backend location for deliberately deferred product and technical directions. None of these items is committed work; each requires concrete evidence and a separately approved design before implementation.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/roadmap.md` | Approved portable build and deployment sequence |
| `.aidoc/designs/web-api.md` | Current client-neutral API and trust boundary |
| `.aidoc/designs/database-concurrency.md` | Maintained local SQLite reliability baseline |
| `.aidoc/designs/difficulty-calibration.md` | Current deterministic strategy-grade evidence |

## Product Expansion

Possible product directions include single-user saved-game portability, accounts, account-scoped authorization, public multi-user hosting, cloud synchronization, shared games, collaboration, localization, and native mobile clients.

Accounts and multi-user hosting form one product and security boundary. Any proposal must define identity, ownership, tenant isolation, authorization for every session operation, anonymous-player behavior, retention, abuse handling, and migration from the current single-operator model. A shared deployment credential is operational protection, not user identity.

The TypeScript browser client remains a separate repository. This Go repository stays client-neutral unless a reviewed capability is shared by multiple clients and belongs in the engine or API.

## Deployment Beyond the Current Milestone

The portable deployment milestone deliberately excludes availability objectives, zero-downtime replacement, monitoring platforms, scheduled browser checks, backup drills, and release-management frameworks. A concrete operational failure or recovery requirement must justify any addition and define a proportionate acceptance test.

Public multi-user hosting requires account-scoped authorization, rate limits, abuse controls, tenant-safe observability, privacy and retention policy, capacity objectives, and multi-tenant recovery. Network filesystems, distributed databases, active-active replicas, and multi-region recovery likewise need measured availability or scale requirements; the current local SQLite and single-host contract does not imply support.

## Evidence-Gated Database Work

Large-import batching, resumable partial batches, expanded progress reporting, and throughput optimization require a reproducible user workload that demonstrates unacceptable latency or lock behavior on supported hardware. Any proposal must separate classification cost from SQLite write cost, preserve puzzle identity and history semantics, and define a bounded black-box acceptance case.

Minimum-clue and uniqueness admission require a concrete product need and explicit semantics. Full Sudoku-symmetry canonicalization, durable attempt identities, abandonment tracking, elapsed-duration statistics, player attribution, and telemetry likewise remain outside the current database contract.

## Evidence-Gated Solver and Rating Work

Technique-tier changes, strategy expansion, weight changes, generation-budget changes, and storage treatment of `strategy-unsolved` puzzles require representative fixtures and before-and-after evidence on exploratory and held-out corpora. Target-hit, reproducibility, latency, and coverage thresholds must be stated before tuning begins.

Human observations may support a separately named player-difficulty model. Such a model needs a defined population, rating method, privacy boundary, sample-quality controls, and an explanation of how it coexists with deterministic strategy grades; it must not silently redefine Easy through Evil.

## Decision Gate

Deferred work becomes a roadmap candidate only when a concrete user or workload establishes ownership, threat model, data lifecycle, compatibility expectations, measurable acceptance criteria, and test strategy. Until then, the maintained product and portable deployment roadmap remains unchanged.
