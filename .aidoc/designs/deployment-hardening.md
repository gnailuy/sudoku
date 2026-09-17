---
domain: Designs
status: Active
entry_points:
  - cmd/api.go
  - webapi/server.go
  - api/openapi.yaml
dependencies:
  - .aidoc/designs/roadmap.md
  - .aidoc/designs/web-api.md
  - .aidoc/designs/e2e-api-scenarios.md
---

# Single-Operator Deployment Hardening

The deployment contract promotes one immutable frontend/backend pair behind a reverse-proxy authentication boundary while keeping the Go API loopback-only. The contract supports either a dedicated origin or an operator-selected path prefix on a shared host without coupling Sudoku releases, state, or recovery to another site.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/roadmap.md` | Delivery order and milestone exit gate |
| `.aidoc/designs/web-api.md` | Current API, recovery, and network-security contract |
| `.aidoc/designs/e2e-api-scenarios.md` | Existing built-binary API verification |
| [Sudoku UI deployment design](https://github.com/gnailuy/sudoku-ui/blob/master/.aidoc/designs/deployment-hardening.md) | Static artifacts and browser verification |

## Why the Operating Contract Exists

The private preview proves connectivity but not safe promotion, durable restart, or recovery. A single operator needs reproducible releases and evidence that failure cannot expose the backend, lose active games, or leave frontend and backend versions mismatched.

Sudoku may share a host and Caddy process with an unrelated site, but shared infrastructure does not create a shared application lifecycle. Sudoku owns a route namespace, release roots, service, state, configuration, logs, checks, and backups that can be changed or restored without replacing another site's assets or process.

## Exposure and Authentication Boundary

Caddy is the only public listener. The API binds to loopback, and the complete Sudoku surface—static shell and API routes—sits behind one host-owned, single-operator authentication policy. A payload-free health route may remain unauthenticated for liveness only; readiness and state checks run locally.

The public mount is a deployment input: either an origin root or a normalized prefix such as `/sudoku/`. A prefixed deployment keeps the shell, API, and health route inside that namespace; the reverse proxy removes the mount prefix only when forwarding to the existing backend routes. Repository artifacts contain no domain, host path, credential, or dependency on a neighboring site's URL layout.

Authentication hashes and other secrets live outside release directories with restrictive permissions. Rotation installs a new Caddy-supported hash, validates configuration, reloads Caddy, proves the new credential, and proves the old credential is rejected. Browser assets never contain credentials, bearer tokens, private hostnames, or backend addresses.

## Paired Immutable Releases

One release ID identifies the tested frontend/backend pair. Its immutable manifest records both Git commits, backend binary SHA-256, frontend asset SHA-256 values, OpenAPI digest, build toolchains, build time, mount mode, and previous compatible release ID.

The host keeps immutable backend and frontend release directories plus atomic `current` and retained `previous` references. API recovery state, the puzzle database, host configuration, authentication material, and logs remain outside releases. Caddy reads only the active frontend reference, and the service starts only the active backend reference.

A candidate is staged with verified checksums and permissions. The backend starts on an alternate loopback port with isolated state; API smoke checks and the companion desktop/phone browser journey run against the staged pair before `current` changes.

Promotion records `previous`, switches the pair atomically, restarts the API gracefully, proves local readiness and release identity, then proves the authenticated public shell, expected assets, gameplay journey, and clean browser console. A frontend and backend are never promoted independently.

## Durable Service and Storage

One unprivileged service identity owns explicit working, data, recovery, and configuration locations. The service starts after networking, restarts unexpected failures with bounded backoff, honors the API's ten-second graceful shutdown budget, uses restrictive permissions, and starts after host reboot without an interactive login.

`deploy/sudoku-api.service.example` keeps the paired `current` backend separate from persistent XDG data and recovery roots. `scripts/check_deployment.py` enforces the portable service contract, while `scripts/e2e_api.py` proves that clean restart and forced termination preserve an accepted active game. Enabling the user manager without an interactive login, repeated-failure behavior, and reboot proof are target-host acceptance steps because repository CI does not control the host service manager.

Lifecycle acceptance also covers a missing release, invalid permissions, occupied port, recovery-lock conflict, and bounded restart failure. Every successful restart preserves an active game; no service operation changes a public route, credential, or neighboring application.

## Monitoring, Backup, and Restore

Independent checks cover service liveness and restart loops, local API readiness, authenticated static release identity, storage writability and capacity, backup age and integrity, and scheduled desktop/phone browser journeys. Local checks run every five minutes, alert after two consecutive failures with component and release ID, and emit one recovery notice; browser and backup-integrity checks may run daily.

A cold bounded snapshot gracefully stops the API, copies the SQLite database and API recovery namespace as one consistency unit, restarts immediately, and verifies readiness. Backups also retain non-secret deployment configuration and current/previous manifests, are checksummed and mode-restricted, publish atomically, and use configurable retention with capacity preflight.

Restore is rehearsed into isolated directories and an alternate port. The drill verifies checksums, SQLite `quick_check`, recovery parsing, session restoration, database statistics, and the companion browser journey before any separately approved live-state replacement.

## Rollback and Acceptance

Pre-promotion readiness, asset, contract, or browser failure returns to the untouched current release automatically. After promotion, the operator chooses rollback; rollback atomically restores the paired `previous` reference, restarts the API, verifies recovery and release identity, and repeats desktop and phone gameplay.

Milestone acceptance proves unauthenticated application requests are rejected, authenticated shell and API journeys succeed, health discloses no state, the backend port is externally unreachable, repository and built assets contain no environment secret, restart preserves a game, each component failure alerts, isolated restore succeeds, paired rollback succeeds, and the complete exercise still passes after host reboot.

Accounts, player identity, public multi-user service, authorization changes, active-active replicas, and changes to another hosted site's deployment remain explicit non-goals.
