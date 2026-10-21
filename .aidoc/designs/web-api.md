---
domain: Designs
status: Active
entry_points:
  - api/openapi.yaml
  - cmd/api.go
  - cmd/web_runtime.go
  - webapi/server.go
  - webapi/account_routes.go
dependencies:
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/e2e-api-scenarios.md
  - .aidoc/designs/localization-contract.md
  - .aidoc/designs/user-accounts.md
---

# HTTP API Backend

The HTTP backend exposes the authoritative game engine only through revisioned sealed guest documents and owner-scoped account games. The current API has no anonymous server-session, transfer, recovery, or application-bearer namespace; acceptance tests keep those unsupported surfaces absent.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/game-engine.md` | Canonical state, action, hint, and serialization contract |
| `.aidoc/designs/hint-presentation-protocol.md` | Semantic teaching plan and exact application contract |
| `.aidoc/designs/localization-contract.md` | Message descriptors, canonical English fallback, and client localization boundary |
| `.aidoc/designs/user-accounts.md` | Guest documents, identity, ownership, and claim boundary |
| `.aidoc/designs/e2e-api-scenarios.md` | Built-binary and handler acceptance coverage |
| `.aidoc/architecture/guidelines.md` | Package dependency boundaries |

## Why the API Is Client-Neutral

The HTTP boundary lets separately deployed clients use one Go-owned gameplay contract without sharing implementation types. The OpenAPI contract owns exact wire shapes, while `webapi` translates requests to `game.Game` and keeps browser presentation state outside the backend.

Guest and account ownership modes replace anonymous durable server sessions. Guest play creates no server-side player or recovery row; signed-in play uses durable owner predicates and optimistic revisions. Removing the overlapping anonymous model prevents three persistence and authorization systems from representing the same browser game.

## Process and Contract Boundaries

`cmd.runAPI` owns loopback listener policy, independent guest and account runtime loading, exact browser origins, timeouts, and graceful shutdown. `webapi.NewHandler` owns strict generated routing, body limits, JSON validation, exact-origin CORS, and stable errors. Every OpenAPI operation with a required JSON body—including guest hint preview and account presentation updates—passes through the same content-type and 1 MiB ingress policy before generated decoding. `webapi.GuestRuntime` supplies sealed guest documents and game creation. `webapi.AccountRuntime` depends on that guest capability and adds OIDC, application-session authentication, owner-scoped game services, and explicit guest claim. Guest endpoints validate only the guest capability; login, claim, and account endpoints validate the complete account capability.

`api/openapi.yaml` uses OpenAPI 3.1.1 and defines health, guest, authentication, account, and account-game resources. Generated transport models stay inside `webapi/generated.go`; `scripts/check-api-contract.sh` rejects stale generation. Deliberate contract breaks remain acceptable during development but must update the client and both repositories' acceptance catalogs together.

## Resource and Revision Model

Guest create returns one authenticated encrypted document, authoritative revision, actual difficulty, and snapshot. Guest actions submit the latest document plus expected revision and return a replacement document only after the engine accepts the transition. Guest hint preview is read-only and returns no replacement document.

Account-game create, list, read, action, hint, presentation-time synchronization, and deletion derive ownership from the authenticated application session. Creation prepares the reusable catalog puzzle first, then atomically inserts the linked play run and owner-scoped game so an account persistence failure cannot leave a half-created presentation. Mutations require the session-bound request proof, include the expected gameplay revision when applicable, and never accept a user identifier from the client.

Action requests use the typed engine boundary. `set-candidates` replaces one complete cell set, and `apply-hint` accepts only the deterministic `plan_id` from the current read-only preview. Snapshots expose one candidate grid and no parallel notes field. Hint plans identify correction or strategy kind; expose descriptor-valued title, summary, step messages, narration messages, strategy names, and difficulty names; and carry a typed clear, candidate replacement, placement, or elimination conclusion. Every descriptor includes its deterministic canonical English fallback, and digit, coordinate, candidate, unit, strategy, difficulty, and count semantics remain typed. `restart` begins a revision-checked fresh attempt on the same puzzle; account games persist the clean engine state and zero elapsed presentation time atomically. Guest presentation time remains browser-owned and resets only after the replacement sealed document is accepted. Presentation preferences remain client-owned.

## Security and Failure Policy

`sudoku api` accepts loopback listeners only. A trusted reverse proxy or equivalent host boundary owns public TLS and routing; exact browser origins remain explicit. Zero-config startup creates or reuses one private 32-byte sealing key in the managed application-data directory, so guest documents survive process restarts without sharing a compiled key. Private storage means a regular non-link file with owner-only mode bits on Unix, or a non-reparse-point file with a protected DACL limited to the current user, Local System, and Administrators on Windows. `--web-config` must satisfy the same platform-native privacy boundary and may provide a named active guest key plus decrypt-only rotation keys. Google discovery and account storage initialize only when the same private configuration contains complete guest and account sections; partial account configuration fails startup. Account routes use a secure HttpOnly application-session cookie plus HMAC request proof for mutations.

Malformed JSON, unsupported media, oversized bodies, unavailable private configuration, stale revisions, engine rejection, missing ownership, and internal failures use stable error codes plus bounded semantic message descriptors. The API ignores `Accept-Language`; identical state and input produce identical descriptors, fallback text, revisions, and plan IDs. Logs, descriptor arguments, and fallbacks never include sealed documents, cookies, OAuth material, puzzle history, raw internal errors, or profile secrets.

## Verification

Handler tests prove the complete required-body operation matrix uses shared media-type and size enforcement, then cover strict decoding, exact origins, route absence for `/api/v1/sessions*`, sealed guest replacement, login rotation, session-bound request proof, idempotent claim, owner-scoped game access, elapsed-time synchronization, and account deletion. Package tests prove engine translation and persistence semantics.

`scripts/e2e_api.py` starts the built binary and proves zero-config guest creation, account-route failure, managed-key persistence across restart, Unix owner-only key enforcement, configured guest-only startup, account-without-guest rejection, the direct `--web-config` cutover, health, exact-origin policy, loopback-only startup, graceful shutdown, locale-invariant descriptor fallbacks, and permanent absence of the anonymous session namespace. The permanent Windows unit lane runs the complete Go package suite, including native `privatefs` DACL, principal, object-type, and reparse-point tests plus managed-key reuse and configured-startup coverage. Non-Windows hosts still cross-compile the Windows binary as a separate portability check. `scripts/check-api-contract.sh` validates and regenerates the OpenAPI descriptor boundary.
