---
domain: Designs
status: Active
entry_points:
  - cmd/api.go
  - webapi/server.go
  - api/openapi.yaml
dependencies:
  - .aidoc/designs/e2e-test-scenarios.md
  - .aidoc/designs/web-api.md
---

# E2E API Scenarios

The API scenario catalog verifies HTTP startup safety, lifecycle operations, recovery, concurrency, origin policy, authentication, and OpenAPI conformance against the running built server.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/e2e-test-scenarios.md` | E2E discovery map, isolation rules, and automation entry points |
| `AGENT.md` | Required black-box verification discipline |

## Why This Boundary

The API is an external security and compatibility boundary. Black-box requests must prove that runtime behavior, durable sessions, and the published contract agree without importing Go handlers.

## 11. HTTP API Backend

Run the automated black-box lifecycle against the built backend with isolated state roots:

**Action:** Execute the matching case in `scripts/e2e_api.py`, which owns the canonical command sequence and fixture.

The harness calls the running `sudoku api` process rather than importing Go handlers. It covers startup safety, process-lock exclusion, health, strict input, exact-origin CORS configuration, exact bearer authentication, lifecycle operations, revision conflict, export/import, restart recovery, and discard. Handler and command tests complement the black-box lane with malformed-input envelopes, preflight method/header rejection, concurrent session isolation, allowed-origin validation, and lock ownership. The scenarios below define the broader acceptance contract; frontend behavior remains in the separate client project.

### 11.1 Startup, Binding, and Health
**Action:** Start `sudoku api` with isolated XDG roots using the default listener and an explicit network listener, call `/healthz`, and request an unknown `/api/` path.
**Expected:** The default listener is loopback, the explicit listener requires authentication configuration, startup prints the listening address, health reports healthy, the unknown route returns a stable JSON `404`, and no frontend assets or SPA fallback are served.

### 11.2 Session Creation and Strict Input
**Action:** Create sessions by difficulty and puzzle string, then send conflicting sources, unknown fields, malformed JSON, wrong content types, and oversized bodies.
**Expected:** Valid requests return opaque IDs, revision zero, and authoritative snapshots. Invalid requests return bounded stable errors without creating sessions or leaking host details.

### 11.3 OpenAPI Contract and Runtime Conformance
**Action:** Validate and lint `api/openapi.yaml` with the pinned Redocly CLI, regenerate the strict Go boundary and confirm a clean diff, compare the contract with the target branch using `oasdiff`, then execute every declared operation and representative examples against the built server.
**Expected:** The OpenAPI 3.1.1 contract is valid and lint-clean, generation is reproducible, no unapproved breaking change is reported, documented examples match runtime responses, and no implemented route is absent from the contract.

### 11.4 Actions, Hints, and Revision Conflicts
**Action:** Enter values, atomically replace a cell's complete note set, preview/apply a hint, undo/redo, and submit two actions with the same expected revision.
**Expected:** Accepted mutations increment revisions once and match engine semantics. Empty, one-digit, and multi-digit note replacements use the same action and each accepted replacement produces one revision and one undo record. Legacy toggle and clear note kinds are rejected as unknown actions. Hint preview is read-only. The delayed mutation returns `409` with current state and never overwrites the accepted action.

### 11.5 Restart Recovery and Discard
**Action:** Mutate two API sessions, stop and restart the server, reconnect to both, then discard one.
**Expected:** Both sessions restore from separate private records with complete values, notes, and history. Discard removes only the selected record, and another restart retains the other session.

### 11.6 Concurrent Sessions and Process Lock
**Action:** Mutate separate sessions concurrently, submit concurrent actions to one session, and start a second `sudoku api` process against the same state root.
**Expected:** Different sessions proceed independently, one session remains revision-ordered, and the second process fails clearly without modifying recovery records.

### 11.7 Origin Policy
**Action:** Send browser-style preflight and mutation requests with no configured origin, exact allowed local and remote HTTP/HTTPS origins, a different port, `null`, a wildcard, and a path-bearing origin.
**Expected:** Cross-origin browser access is denied by default. Only exact configured origins succeed; responses never enable wildcard CORS, and authenticated preflight permits only the required authorization header.

### 11.8 Authentication and Remote Access
**Action:** Bind to a non-loopback address with no token, then with a configured token; call API resources with a missing, incorrect, and correct bearer credential.
**Expected:** Unsafe startup is rejected. Missing and incorrect credentials receive bounded unauthorized responses, the correct credential succeeds, and logs never contain the token.

### 11.9 Existing Frontend Compatibility
**Action:** Run all applicable root CLI, TUI, serialization, candidate, and recovery scenarios after API tests.
**Expected:** Existing output, actions, session bytes, recovery behavior, and terminal rendering remain compatible.
