---
domain: Designs
status: Active
entry_points:
  - scripts/e2e_api.py
  - webapi/server_test.go
  - webapi/account_routes_test.go
dependencies:
  - .aidoc/designs/web-api.md
  - .aidoc/designs/user-accounts.md
---

# HTTP API E2E Scenarios

The API acceptance catalog protects the composed guest/account boundary against the running binary and the configured handler. Anonymous server sessions and API recovery are intentionally absent.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/web-api.md` | Canonical resource, revision, and security boundary |
| `.aidoc/designs/user-accounts.md` | Guest, login, ownership, and claim invariants |
| `.aidoc/designs/e2e-test-scenarios.md` | Repository-wide black-box discovery map |
| `api/openapi.yaml` | Exact wire contract |

## Why These Scenarios Exist

OpenAPI validation cannot prove listener safety, generated routing, middleware order, process shutdown, sealed-document behavior, or database ownership. The built-binary lane proves deployment-visible behavior; handler and package integration provide deterministic configured account coverage without external identity traffic.

## Built-Binary Clean Boundary

**Action:** Start `sudoku api` on an isolated loopback port and database, probe health, then call `/api/v1/sessions`, a child session path, and the former import path.

**Expected:** Health returns 200 and every removed anonymous-session path returns 404. No recovery directory, anonymous gameplay row, or compatibility redirect is created.

**Action:** Start without private account configuration and call guest create plus the current-account probe.

**Expected:** Both routes fail closed with the bounded `account-unavailable` response. Startup remains healthy so operators can distinguish missing private configuration from process failure.

**Action:** Send one request from the configured exact origin and one preflight from another origin.

**Expected:** The configured origin receives its exact CORS response; the unconfigured origin receives 403. Wildcard and path-bearing origins remain invalid startup configuration.

**Action:** Attempt a non-loopback listener, then terminate a healthy loopback process with `SIGTERM`.

**Expected:** Non-loopback startup fails before listening. The healthy process completes bounded graceful shutdown with exit status zero.

## Configured Guest and Account Integration

**Action:** Create a sealed guest game, preview a hint, apply revisioned note/value/hint actions, and replace the document after each accepted mutation.

**Expected:** Guest state creates no user, application-session, account-game, claim, or recovery row. Revisions advance once, stale or wrong hint plans fail, and every replacement document opens to the authoritative snapshot.

**Action:** Complete deterministic local OIDC login, use the application-session cookie and request proof, create an account game, claim a guest game twice, mutate/list/delete owned games, revoke sessions, and delete the account.

**Expected:** Login rotates digest-only application sessions. Cookie mutations reject missing request proof. Same-owner claim retries return one game; cross-owner attempts reveal no owner. Owner predicates hide foreign IDs, and account deletion preserves catalog/cohort evidence.

**Action:** Exercise hint plans, complete note replacement, candidate adoption, invalid-value mistake counting, Undo/Redo, elapsed-time synchronization, and revision conflict in guest and account modes.

**Expected:** Both ownership modes expose identical game-engine semantics. Read-only hints do not mutate state; an elimination hint applied to an empty note map materializes the legal-candidate grid, removes the taught candidates, and remains one undoable revision; accepted actions advance once; stale writes never overwrite authoritative state.

## Database Cutover

**Action:** Run `sudoku db rebuild-web --db <path> --yes` against schema version 5, then open the database with schema-6 code.

**Expected:** Users, external identities, application sessions, account games, and claims are empty and use the new schema. Base puzzles, provenance, serving cohorts, cohort membership, and historical play runs are unchanged. The operation is transactional and refuses any source schema other than version 5.
