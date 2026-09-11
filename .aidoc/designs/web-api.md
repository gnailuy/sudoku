---
domain: Designs
status: Active
entry_points:
  - api/openapi.yaml
  - cmd/root.go
  - cmd/session.go
  - game/contract.go
  - recovery/recovery.go
dependencies:
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/background-autosave.md
  - .aidoc/designs/e2e-api-scenarios.md
---

# HTTP API Backend

The versioned, general-purpose HTTP backend exposes the existing game engine without adding frontend code. A `sudoku api` process remains authoritative for gameplay, recovery, validation, and concurrency while separately deployed clients own presentation and may connect from any explicitly permitted origin.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/game-engine.md` | Canonical state, action, hint, and serialization contract |
| `.aidoc/designs/background-autosave.md` | Private recovery storage available to API sessions |
| `.aidoc/designs/e2e-api-scenarios.md` | Black-box HTTP acceptance coverage |
| `.aidoc/architecture/guidelines.md` | Package dependency boundaries |

## Why the API Is Client-Neutral

The HTTP boundary makes `game.Game` available to local tools, separately hosted browser applications, and other network clients through one stable contract. Keeping frontend source and build tooling in a separate TypeScript project lets backend and UX contracts evolve and deploy independently. A contract-first OpenAPI document lets those projects agree on HTTP behavior without sharing Go implementation types.

The backend is safe by default rather than local-only: it binds to loopback unless an operator explicitly selects another address, and non-loopback operation requires authentication. Transport models remain independent from Go engine structs, errors and revisions are machine-readable, and handlers are tested through HTTP.

## Process and Package Boundaries

`cmd/api.go` owns command flags, dependency construction, startup output, signal shutdown, and listener lifecycle. `webapi` owns HTTP routing, request limits, JSON translation, session lookup, revision checks, origin policy, and error envelopes. The Sudoku repository contains no frontend asset package, browser launcher, TypeScript toolchain, or SPA routing.

Each active API session owns one `game.Game`, one opaque random session ID, one monotonically increasing revision, and one recovery record. A registry serializes access per session; different sessions may proceed independently. `game.Game` remains non-concurrent and unaware of HTTP.

Session creation reuses existing puzzle input, difficulty generation, solver configuration, and database fallback policies through dependencies wired by `cmd`. The API must not import `cmd` or duplicate generation rules.

## Contract-First OpenAPI Workflow

`api/openapi.yaml` is the canonical external contract and uses OpenAPI 3.1.1. The contract defines paths, operation identifiers, strict request and response schemas, optional bearer authentication, stable error codes, one-mebibyte session transfer limits, examples, and revision semantics before handlers are implemented. Prose in this design explains intent and cross-cutting constraints; the OpenAPI document owns exact wire names and shapes.

The implementation uses `oapi-codegen` to generate transport models and a strict Go server interface from the pinned contract. Generated code remains confined to the HTTP boundary: handwritten adapters perform dependency wiring, authentication, origin checks, session coordination, and translation to `game.Game`; generated code never owns Sudoku rules, persistence policy, or application implementation. Generated files are committed and CI rejects stale generated output.

CI validates and lints the OpenAPI document with a pinned Redocly CLI, verifies generated-code freshness, and uses `oasdiff` against the target branch to report breaking contract changes. Deliberate breaking changes require a new URL namespace rather than silently changing `/api/v1`. Representative examples and every declared operation are exercised through the built server so schema validity does not substitute for runtime conformance.

Static API reference documentation is rendered from `api/openapi.yaml` by the pinned Redocly CLI and published as a CI artifact or site. `sudoku api` does not embed or serve Swagger UI, Scalar, Redoc, or other documentation assets. Released or tagged contracts are the source for separately versioned client generation; a TypeScript client may use `openapi-typescript`, while other client repositories may select an appropriate generator without changing this backend.

## Version 1 Resource Model

All JSON endpoints live below `/api/v1`; `/healthz` is outside the API namespace. Version 1 exposes:

- `GET /api/v1/sessions` lists active and recovered sessions using bounded non-payload summaries;
- `POST /api/v1/sessions` creates a game from a difficulty or puzzle string;
- `POST /api/v1/sessions/import` restores bounded versioned session bytes supplied by a client;
- `GET /api/v1/sessions/{id}` returns the current detached snapshot and revision;
- `GET /api/v1/sessions/{id}/hint` previews a structured hint without mutation;
- `GET /api/v1/sessions/{id}/export` returns versioned session bytes;
- `POST /api/v1/sessions/{id}/actions` submits one typed engine action;
- `DELETE /api/v1/sessions/{id}` explicitly discards the session and recovery record.

Create requests use a tagged source object so difficulty and puzzle input cannot conflict. Import and export transfer bounded validated bytes; no endpoint accepts or returns an arbitrary host filesystem path.

API snapshots use explicit JSON fields for givens, visible values, invalid markers, manual notes, legal candidates, status, and undo/redo availability. Rows and columns are numbered 1 through 9 at the transport boundary. API models do not expose Go type names, internal history records, frontend view models, or the version 1 persistence document.

## Actions, Revisions, and Errors

Action requests contain `expected_revision`, an action kind, and only the fields required by that action. The API translates accepted action kinds to `game.Action`; unknown kinds or extra fields fail before the engine is called. The `set-notes` action atomically replaces one cell's full manual-note set, so clients may debounce rapid local note edits into one revision-checked mutation and reconcile from the authoritative response. Presentation preferences such as theme, focus, or candidate visibility remain client-owned and have no API action.

Every accepted mutation increments the session revision exactly once and returns the resulting snapshot, action result, and new revision. Rejected actions leave both game state and revision unchanged. A stale `expected_revision` returns HTTP `409 Conflict` with the current revision and snapshot so delayed clients cannot silently overwrite newer state.

Expected failures use a stable envelope with a code, human-readable message, and optional field or cell context. Invalid JSON and transport validation use `400`; absent sessions use `404`; stale revisions use `409`; oversized bodies use `413`; engine rejections use `422`; unexpected failures use `500` without leaking paths or internal details.

## Recovery and Lifecycle

Accepted mutations persist the newest serialized session through the private recovery store before the success response is committed. A persistence failure returns an error and must not claim durable success; the in-memory engine is reconstructed from the last durable bytes when necessary to preserve the contract.

Server startup discovers valid records from a dedicated API recovery namespace and exposes bounded summaries through the session collection. The TUI recovery namespace remains separate, so simultaneous frontends cannot adopt the same record; explicit import/export is the transfer boundary. Explicit discard deletes one API record, while graceful server shutdown retains active records for restart.

Only one `sudoku api` process may own the API recovery namespace at a time. Startup acquires an exclusive process-lifetime lock and fails clearly rather than allowing two registries to write the same records.

## Network Security and Client Access

`sudoku api` listens on `127.0.0.1` by default but accepts an explicit `--listen` address for LAN, container, or hosted deployment. Non-loopback binding requires an operator-supplied authentication token; every `/api/v1` request must then present that token as a bearer credential. `/healthz` remains payload-free and may be used by deployment health checks. The server validates request authority, requires JSON content types for mutations, limits request/header/body sizes, applies read/write/idle timeouts, logs metadata without credentials, puzzle data, or session payloads, and shuts down cleanly.

Browser cross-origin access is disabled by default. A repeatable `--allowed-origin` option may enable any exact `http` or `https` origin needed by a separately deployed frontend; wildcards, `null`, path-bearing origins, and malformed origins are rejected. Preflight responses advertise only required methods and headers, including `Authorization` when authentication is active, and browser requests must match a configured origin exactly.

Opaque session IDs prevent accidental collisions and never substitute for authentication. Internet-facing deployments terminate TLS at a trusted reverse proxy or platform boundary; `sudoku api` does not infer identity from forwarded headers. Account-specific authorization, multi-tenant isolation, distributed rate limiting, and shared-game collaboration remain separate product work.

## Verification

Handler tests exercise real HTTP requests, strict decoding, bounded error envelopes, body limits, action translation, typed errors, revision conflicts, recovery failure, concurrent access and isolation, exact bearer authentication, bounded preflight policy, default CORS denial, and exact-origin allowlisting. Command tests validate configured origins and exclusive recovery lock ownership. Contract verification validates and lints `api/openapi.yaml`, confirms generated Go code is current, checks compatibility with `oasdiff`, and executes representative OpenAPI examples against the built server. Black-box tests start the built binary with isolated XDG roots on both default and explicit listen addresses, verify process-lock exclusion and security startup failures, call every versioned operation, restart the process, and confirm recovery without importing Go packages or relying on a frontend.
