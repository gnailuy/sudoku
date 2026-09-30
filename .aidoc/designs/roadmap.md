---
domain: Designs
status: Active
entry_points:
  - generator/generator.go
  - solver/classify.go
  - core/canonical.go
dependencies:
  - .aidoc/designs/exact-grade-generation-experiment.md
  - .aidoc/designs/difficulty-model.md
  - .aidoc/designs/deployment-hardening.md
  - .aidoc/designs/e2e-test-scenarios.md
---

# Roadmap

The next generator milestone is a separately reviewed offline Expert/Evil catalog replenisher. The milestone productizes the approved exact-seed clue-addition decision without retaining experiment datasets or changing interactive generation.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/exact-grade-generation-experiment.md` | Canonical replenishment decision and product invariants |
| `.aidoc/designs/difficulty-model.md` | Fixed deterministic grade and within-grade score contract |
| `.aidoc/designs/database-catalog.md` | Catalog identity, provenance, and storage boundaries |
| `.aidoc/designs/database-puzzle-selection.md` | Current acquisition, recycling, and fallback contract |
| `.aidoc/designs/deployment-hardening.md` | Maintained portable deployment and replacement contract |
| `.aidoc/designs/e2e-test-scenarios.md` | Maintained black-box verification baseline |

## Why Replenishment Comes Next

Catalog-first play made exact high-grade selection reliable. The remaining product opportunity is controlled offline supply growth, not a new interactive generator or a repository of experiment outputs.

Exact-seed clue addition is the smallest approved direction that uses existing classification and canonicalization boundaries. A replenisher can validate a complete candidate batch before storage and keep player requests independent of search cost.

## Approved Next Change

The next proposal may implement offline Expert/Evil replenishment around the approved exact-seed clue-addition policy. The implementation must keep deterministic classification authoritative, reject canonical duplicates, record derivation provenance, and remain separate from normal play until independently reviewed.

Hard stays outside the change because its decision evidence was inconclusive. Medium stays outside the change because the pinned catalog has no approved Medium seed source. Interactive deadlines, fallback behavior, solver semantics, visible grades, and current catalog acquisition remain unchanged.

## Delivery Gates

- Unit and integration tests cover exact-grade acceptance, canonical duplicates, provenance, interruption, and atomic batch publication.
- Applicable built-binary E2E scenarios prove that normal CLI and API play remain unchanged.
- Pull-request CI keeps unit, race, vet, lint, API contract, API E2E, line-CLI E2E, and TUI PTY E2E independent and green.
- Product documentation describes only current behavior and approved direction; transient measurements remain outside the repository.
- Repository files contain no private hostname, credential, operator path, live port assignment, release identifier, or neighboring-application topology.

Strategy-aware construction, blind budget expansion, player-difficulty labels, accounts, multi-user hosting, and production reliability ceremony remain outside this milestone. Each direction requires concrete product evidence and a separately approved design.
