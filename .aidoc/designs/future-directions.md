---
domain: Designs
status: Active
entry_points:
  - cmd/api.go
  - cmd/import.go
  - db/puzzle.go
  - webapi/server.go
dependencies:
  - .aidoc/designs/roadmap.md
  - .aidoc/designs/database-concurrency.md
  - .aidoc/designs/database-puzzle-selection.md
  - .aidoc/designs/web-api.md
---

# Future Directions

Sudoku may eventually revisit evidence-gated database work or support hosted and multi-user use cases, but those capabilities are not current priorities. Each direction below changes product behavior, ownership, persistence, or security assumptions and therefore requires a separately approved design before implementation.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/roadmap.md` | Current stabilization priorities and sequencing |
| `.aidoc/designs/database-concurrency.md` | Maintained SQLite reliability and lock-behavior baseline |
| `.aidoc/designs/database-puzzle-selection.md` | Maintained puzzle identity, acquisition, and history semantics |
| `.aidoc/designs/web-api.md` | Existing API trust boundary and safe defaults |
| `api/openapi.yaml` | Canonical external contract that future clients consume |

## Product Directions

Possible later product work includes accounts and account-scoped authorization, multi-tenancy, cloud synchronization, shared games, cross-device merge, localization, mobile clients, and collaboration.

A TypeScript client and browser UI belong to a separate project. This Go repository remains client-neutral unless a future design establishes a backend capability that multiple clients need.

## Production Exposure

The current API is suitable for deliberate deployment behind an operator-controlled boundary, but a public multi-user service needs additional controls. Before such exposure, a design should cover rate limiting, account-scoped authorization, operational metrics, backup and restore guidance, and a documented TLS reverse-proxy profile.

Production design must also define tenant isolation, credential lifecycle, abuse handling, retention, upgrades, and recovery objectives. Loopback defaults, mandatory authentication for non-loopback binding, bounded requests, and exact-origin CORS remain baseline protections rather than substitutes for those controls.

## Evidence-Gated Local Database Work

The current per-puzzle import transaction is appropriate for a small local SQLite application. Large-import batching, resumable partial batches, expanded progress reporting, and synthetic throughput machinery are not active roadmap work because no real workload has demonstrated unacceptable import latency or lock behavior.

Import-performance work requires a reproducible user workload that shows a material problem on supported hardware. Any proposal must separate puzzle classification cost from SQLite write cost, preserve the existing identity and history semantics, and define a bounded black-box acceptance case before changing the importer.

Minimum-clue and uniqueness admission also require a concrete product need and separately approved semantics. Neither admission policy is necessary to complete or maintain the current database baseline.

## Decision Gate

Future work starts only when a concrete user need establishes ownership, threat model, data lifecycle, compatibility expectations, and test strategy. Until then, these directions remain context for design decisions, not committed roadmap items.
