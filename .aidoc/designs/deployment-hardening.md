---
domain: Designs
status: Active
entry_points:
  - cmd/api.go
  - webapi/server.go
  - api/openapi.yaml
dependencies:
  - .aidoc/designs/web-api.md
  - .aidoc/designs/e2e-api-scenarios.md
---

# Portable Deployment Contract

The deployment contract packages the Go API for simple, verifiable installation behind a reverse proxy. It keeps service state and host policy outside build artifacts and supports safe replacement of one frontend/backend pair without prescribing an operator's domain, port, or filesystem layout.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/web-api.md` | Current guest/account API and network boundary |
| `.aidoc/designs/e2e-api-scenarios.md` | Built-binary and service-example verification |
| [Sudoku UI deployment design](https://github.com/gnailuy/sudoku-ui/blob/master/.aidoc/designs/deployment-hardening.md) | Static artifact, mount, and browser contract |

## Why the Deployment Boundary Exists

A development installation should be easy to reproduce without coupling Sudoku to one host. The deployment boundary separates repository-owned artifacts and checks from operator-owned routing, credentials, service configuration, state, and release selection.

Sudoku may share a machine and reverse proxy with unrelated applications. Sudoku operations must affect only its bounded route, service, listener, state, and files; a Sudoku replacement must not rebuild, restart, roll back, or rewrite a neighboring application.

## Network and Host Policy

The API binds to loopback by default. A reverse proxy may expose the static client and API at an origin root or normalized path prefix, forwarding only the selected API namespace and removing the mount prefix when required.

Guest and account routes fail closed unless `--accounts-config` names a private regular file that supplies guest-document keys, cookie and lifetime policy, request-proof key, and Google OIDC configuration. Account sessions use opaque `Secure`, `HttpOnly`, `SameSite=Lax` cookies; state-changing account requests require same-origin proof. The backend remains loopback-only, exact browser origins are explicit, and a trusted reverse proxy or equivalent host boundary owns public routing. A reverse-proxy access gate, VPN, or allowlist may add host policy but never substitutes for the application ownership boundary.

Repository examples use generic values only. Public files contain no operator hostname, external URL, private address assignment, live port, absolute host layout, credential, or neighboring-site route.

## Artifact Contract

A trusted `main` push or explicitly dispatched `main` workflow waits for every maintained CI lane, then publishes `sudoku-backend-<commit>`. The artifact contains only the Linux executable and `manifest.json`; the manifest binds the repository, workflow and run identity, full commit, executable path, and SHA-256 checksum. `scripts/package_release.py` creates and verifies the same contract locally. Pull requests run its contract tests but cannot publish a deployable artifact.

The frontend repository owns its static artifact and mount input; private host tooling may pair one successful backend artifact with one successful frontend artifact.

A pair record needs only the selected backend commit, frontend commit, checksums, and frontend mount input. The record may live in private host state. The repositories do not require a shared release framework or know which environment consumes the pair.

Pull-request artifacts are verification inputs only. Automatic deployment, when enabled, consumes successful trusted default-branch artifacts rather than artifacts produced by untrusted pull-request code.

## Service and State Contract

One unprivileged service runs the selected backend binary from an operator-chosen release directory. The mode-restricted account configuration, puzzle and account database, and logs remain outside that directory so replacing application files does not replace state or expose credentials.

`deploy/sudoku-api.service.example` demonstrates a home-relative installation, private XDG roots, a loopback listener, bounded restart backoff, and the API's ten-second graceful shutdown budget. Operators may adapt the release and state paths while preserving those boundaries.

The service starts after the user's service manager starts and can be enabled for ordinary host startup. Repository CI validates the example and proves graceful and forced process restart against the built binary; target-host startup remains an operator verification because CI does not control the host.

## Replacement and Failure Handling

A serialized host-side flow downloads or receives the selected artifacts, verifies their checksums, stages the pair away from the active files, and starts the backend on its private listener. The flow verifies payload-free backend health, fail-closed behavior without account configuration, and representative configured guest/account creation, mutation, ownership, cookie, exact-origin, and request-proof boundaries before selecting the new pair.

The companion browser verification checks the shell, referenced assets, health route, session creation, and one desktop/mobile gameplay journey with no unexpected request, page, or console errors. A shared host additionally checks that representative neighboring routes remain unchanged before and after Sudoku replacement.

A failed checksum, startup, health, asset, API, or browser check leaves the working pair selected or restores it. Development downtime and loss of active Sudoku sessions are acceptable; changing or destabilizing a neighboring application is not.

## Scope Boundary

Useful service logs and a payload-free health endpoint are part of the maintained contract. Elaborate monitoring, scheduled browser probes, backup/restore drills, immutable-release ceremony, multi-host coordination, availability objectives, and zero-downtime replacement are deferred until evidence makes them worthwhile.

The implemented account boundary supplies identity, ownership, session revocation, guest claim, and account deletion. Rate limits, abuse controls, tenant-safe observability, backup drills, and availability objectives remain evidence-gated product or operational work rather than deployment substitutes.
