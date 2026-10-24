---
domain: Designs
status: Active
entry_points:
  - scripts/e2e_api.py
  - webapi/server_test.go
  - webapi/account_routes_test.go
  - webapi/hint_strategy_routes_test.go
  - tools/e2e-web-server/main.go
dependencies:
  - .aidoc/designs/web-api.md
  - .aidoc/designs/localization-contract.md
  - .aidoc/designs/user-accounts.md
---

# HTTP API E2E Scenarios

The API acceptance catalog protects the composed guest/account boundary against the running binary and the configured handler. Anonymous server sessions and API recovery are intentionally absent.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/web-api.md` | Canonical resource, revision, and security boundary |
| `.aidoc/designs/localization-contract.md` | Descriptor inventory, typed arguments, and canonical English fallback |
| `.aidoc/designs/user-accounts.md` | Guest, login, ownership, and claim invariants |
| `.aidoc/designs/e2e-test-scenarios.md` | Repository-wide black-box discovery map |
| `api/openapi.yaml` | Exact wire contract |

## Why These Scenarios Exist

OpenAPI validation cannot prove listener safety, generated routing, middleware order, process shutdown, sealed-document behavior, or database ownership. The built-binary lane proves deployment-visible behavior; handler and package integration provide deterministic configured account coverage without external identity traffic.

## Built-Binary Clean Boundary

**Action:** Start `sudoku api` on an isolated loopback port and database, probe health, then call `/api/v1/sessions`, a child session path, and the former import path.

**Expected:** Health returns 200 and every removed anonymous-session path returns 404. No recovery directory, anonymous gameplay row, or compatibility redirect is created.

**Action:** Start without a web configuration, create and mutate a guest game, restart against the same managed application-data directory, mutate the original document again, and probe the current-account route. Repeat startup with a private guest-only configuration. Then attempt account-only configuration and a managed key with group-readable permissions.

**Expected:** Zero-config and configured guest-only modes create sealed guest games without Google setup. The generated 32-byte key is platform-private and remains stable across restart, so the original guest document stays valid. Unix rejects group/world-readable mode bits; Windows rejects reparse points and DACLs that allow principals beyond the current user, Local System, and Administrators. Account routes return the bounded `account-unavailable` response. Account-only configuration fails before startup because account mode depends on a complete guest section.

**Action:** Send one request from the configured exact origin and one preflight from another origin.

**Expected:** The configured origin receives its exact CORS response; the unconfigured origin receives 403. Wildcard and path-bearing origins remain invalid startup configuration.

**Action:** Attempt a non-loopback listener, then terminate a healthy loopback process with `SIGTERM`.

**Expected:** Non-loopback startup fails before listening. The healthy process completes bounded graceful shutdown with exit status zero.

## Configured Guest and Account Integration

**Action:** Create a sealed guest game, preview a hint, apply revisioned candidate/value/hint actions, and replace the document after each accepted mutation. Repeat preview and exact apply over the maintained complete-game manifest until each of the 23 registered strategies crosses the HTTP handler boundary.

**Expected:** Guest state creates no user, application-session, account-game, claim, or recovery row. Revisions advance once, stale or wrong hint plans fail, and every replacement document opens to the authoritative snapshot. Every strategy exposes its public identifier, name, grade, ordered teaching steps, and typed semantic marks before the exact `plan_id` advances the sealed document by one revision. XY-Wing transports one then two implication links. W-Wing first shows only its matching bivalue cells, then introduces and preserves one strong link while the third and final steps add two implication arrows that carry both bridge cases to the exact elimination. Simple Coloring and X-Cycles use family-specific observe, explain, and action descriptors backed by their typed partitions or ordered links; neither falls back to the generic comparison or conclusion message.

**Action:** Complete deterministic local OIDC login, use the application-session cookie and request proof, create an account game, claim a guest game twice, mutate/list/delete owned games, revoke sessions, and delete the account.

**Expected:** Login rotates digest-only application sessions. Cookie mutations reject missing request proof. Same-owner claim retries return one game; cross-owner attempts reveal no owner. Owner predicates hide foreign IDs, and account deletion preserves catalog/cohort evidence.

**Action:** Exercise hint plans, complete candidate replacement, invalid-value mistake counting, repeated Undo/Redo, a manual mutation from a historical point, elapsed-time synchronization, and revision conflict in guest and account modes. Before requesting a strategy, enter one wrong value and corrupt one candidate cell.

**Expected:** Both ownership modes expose identical game-engine semantics and the same descriptor wire shape. Engine catalog coverage proves every emitted plan title, summary, step, narration message, strategy name, and difficulty name has a known ID, valid typed arguments, and bounded canonical English fallback; the guest HTTP strategy matrix proves those descriptors cross the handler with one-based coordinates. Built-binary acceptance proves `Accept-Language` does not vary the bounded unavailable-error descriptor. A restart preserves puzzle identity, givens, and actual difficulty while clearing values, invalid entries, history, mistakes, solved status, and account elapsed time while restoring the initial legal candidate grid. Read-only hints do not mutate state. The wrong value produces a typed value-correction plan before strategy selection; after application, the corrupted candidate cell produces a typed candidate-correction plan that restores the complete locally legal set without identifying the solved digit. Each exact correction advances one revision and is undoable. A later strategy elimination mutates the same grid, persists in the replacement document, and cannot reappear after restore or a later placement. Each persisted Undo restores the exact prior board and candidate snapshot, Redo follows the original path, and a manual mutation from that historical point truncates the abandoned Redo branch. Accepted actions advance once; stale writes never overwrite authoritative state.

## Cross-Repository Browser Harness

**Action:** Start `go run ./tools/e2e-web-server` from an isolated checkout, then run the web client's dedicated full-stack Playwright corpus through its Vite proxy.

**Expected:** The harness exposes only temporary guest/account state, cycles through the maintained minimal nine-game fixture corpus, and uses the production HTTP handler, sealed-document transport, revisions, engine, and strategy registry. The web client must solve every game and observe the exact 23-strategy union. The harness is test infrastructure rather than a production server mode.

## Database Cutover

**Action:** Run `sudoku db rebuild-web --db <path> --yes` against schema version 5, then open the database with schema-6 code.

**Expected:** Users, external identities, application sessions, account games, and claims are empty and use the new schema. Base puzzles, provenance, serving cohorts, cohort membership, and historical play runs are unchanged. The operation is transactional and refuses any source schema other than version 5.
