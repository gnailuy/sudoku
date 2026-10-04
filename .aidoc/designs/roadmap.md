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

The next approved milestone adds optional Google accounts while preserving guest-first play. Delivery begins with the cross-repository ownership and security contract, then implements backend identity and game ownership before the browser adopts the explicit one-game claim flow.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/user-accounts.md` | Approved guest, identity, claim, and account contract |
| `.aidoc/designs/web-api.md` | Current client-neutral API and revision boundary |
| `.aidoc/designs/database-catalog.md` | Shared catalog and presentation-specific play runs |
| `.aidoc/designs/e2e-test-scenarios.md` | Maintained black-box verification baseline |
| `.aidoc/designs/deployment-hardening.md` | Portable artifact and replacement contract |

## Why Accounts Come Next

Guest-first play keeps Sudoku immediate while accounts add explicit cross-device continuity and a My games surface. One browser-held guest game avoids an anonymous game library and gives the claim boundary one understandable, auditable transition.

The account milestone replaces anonymous server recovery with sealed browser-held guest state, then adds Google identity, revocable web sessions, account-owned games, and one idempotent claim transaction. Schema version 3 now provides the provider-keyed identity, digest-only session, and owner-scoped account-game persistence foundation; runtime authentication, sealing, claim, and OpenAPI wiring come next. Existing catalog and game-engine boundaries remain authoritative.

## Approved Delivery Sequence

1. Keep the backend and frontend account design documents aligned on ownership states, sealed guest documents, OIDC and web sessions, claim, deletion, retention, and threat model.
2. Extend the backend persistence foundation with Google OIDC, guest sealed-document actions, account-game HTTP authorization, idempotent claim, logout/revocation, and the OpenAPI contract.
3. Implement the frontend's single-record IndexedDB repository, guest recovery, sign-in return, explicit claim, My games, and account controls.
4. Prove desktop and phone acceptance, stage the coordinated pair, then remove the anonymous server-session path; development sessions may be discarded rather than migrated.

Generator behavior, solver semantics, visible grades, catalog acquisition, and gameplay interactions remain unchanged. Google is the only version 1 identity provider; email links, shared games, collaboration, and broader social features remain separate decisions.

## Delivery Gates

- Unit and integration tests cover sealing failures, OIDC validation, session rotation and revocation, ownership, CSRF, rate limits, and transactional claim behavior.
- Applicable built-binary E2E scenarios prove zero durable guest rows, exact one-game claim, cross-user denial, cross-browser account resume, deletion, and unchanged CLI/TUI behavior.
- Pull-request CI keeps unit, race, vet, lint, API contract, API E2E, line-CLI E2E, and TUI PTY E2E independent and green.
- Product documentation describes only current behavior and approved direction; generated HTML, credentials, and environment-specific topology remain outside the repository.
- Repository files contain no private hostname, credential, operator path, live port assignment, release identifier, or neighboring-application topology.

No further generator work is scheduled. Strategy-aware construction, blind budget expansion, player-difficulty labels, shared games, collaboration, and production reliability ceremony remain outside maintained scope; each direction requires concrete product evidence and a separately approved design.
