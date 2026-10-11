---
domain: Designs
status: Active
entry_points:
  - api/openapi.yaml
  - cmd/api.go
  - webapi/server.go
  - webapi/account_routes.go
dependencies:
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/e2e-api-scenarios.md
  - .aidoc/designs/user-accounts.md
---

# HTTP API Backend

The HTTP backend exposes the authoritative game engine only through sealed guest documents and owner-scoped account games. The removed anonymous session registry, transfer endpoints, recovery namespace, and bearer-authenticated `/api/v1/sessions*` surface are not compatibility boundaries.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/game-engine.md` | Canonical state, action, hint, and serialization contract |
| `.aidoc/designs/hint-presentation-protocol.md` | Semantic teaching plan and exact application contract |
| `.aidoc/designs/user-accounts.md` | Guest documents, identity, ownership, and claim boundary |
| `.aidoc/designs/e2e-api-scenarios.md` | Built-binary and handler acceptance coverage |
| `.aidoc/architecture/guidelines.md` | Package dependency boundaries |

## Why the API Is Client-Neutral

The HTTP boundary lets separately deployed clients use one Go-owned gameplay contract without sharing implementation types. The OpenAPI contract owns exact wire shapes, while `webapi` translates requests to `game.Game` and keeps browser presentation state outside the backend.

Guest and account ownership modes replace anonymous durable server sessions. Guest play creates no server-side player or recovery row; signed-in play uses durable owner predicates and optimistic revisions. Removing the overlapping anonymous model prevents three persistence and authorization systems from representing the same browser game.

## Process and Contract Boundaries

`cmd.runAPI` owns loopback listener policy, account runtime loading, exact browser origins, timeouts, and graceful shutdown. `webapi.NewHandler` owns strict generated routing, body limits, JSON validation, exact-origin CORS, and stable errors. `webapi.AccountRuntime` supplies sealed guest documents, OIDC, application-session authentication, owner-scoped game services, and explicit guest claim.

`api/openapi.yaml` uses OpenAPI 3.1.1 and defines health, guest, authentication, account, and account-game resources. Generated transport models stay inside `webapi/generated.go`; `scripts/check-api-contract.sh` rejects stale generation. Deliberate contract breaks remain acceptable during development but must update the client and both repositories' acceptance catalogs together.

## Resource and Revision Model

Guest create returns one authenticated encrypted document, authoritative revision, actual difficulty, and snapshot. Guest actions submit the latest document plus expected revision and return a replacement document only after the engine accepts the transition. Guest hint preview is read-only and returns no replacement document.

Account-game create, list, read, action, hint, presentation-time synchronization, and deletion derive ownership from the authenticated application session. Mutations require the session-bound request proof, include the expected gameplay revision when applicable, and never accept a user identifier from the client.

Action requests use the existing typed engine actions. `set-notes` replaces one complete note set, `adopt-candidates-as-notes` performs one whole-board transition, and `apply-hint` accepts only the deterministic `plan_id` from the current read-only preview. Presentation preferences remain client-owned.

## Security and Failure Policy

`sudoku api` accepts loopback listeners only. A trusted reverse proxy or equivalent host boundary owns public TLS and routing; exact browser origins remain explicit. Guest documents provide authenticated state, while account routes use a secure HttpOnly application-session cookie plus HMAC request proof for mutations.

Malformed JSON, unsupported media, oversized bodies, unavailable private configuration, stale revisions, engine rejection, missing ownership, and internal failures use bounded stable errors. Logs and errors never include sealed documents, cookies, OAuth material, puzzle history, or profile secrets.

## Verification

Handler tests prove strict decoding, body limits, exact origins, route absence for `/api/v1/sessions*`, sealed guest replacement, login rotation, session-bound request proof, idempotent claim, owner-scoped game access, elapsed-time synchronization, and account deletion. Package tests prove engine translation and persistence semantics.

`scripts/e2e_api.py` starts the built binary, proves health, exact-origin policy, loopback-only startup, graceful shutdown, fail-closed unconfigured guest/account routes, and the permanent absence of the anonymous session namespace. `scripts/check-api-contract.sh` validates and regenerates the OpenAPI boundary.
