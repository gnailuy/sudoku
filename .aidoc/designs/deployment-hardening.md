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

# Portable Deployment Contract

The deployment contract packages the Go API for simple, verifiable installation behind a reverse proxy. It keeps service state and host policy outside build artifacts and supports safe replacement of one frontend/backend pair without prescribing an operator's domain, port, or filesystem layout.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/roadmap.md` | Approved delivery order and scope |
| `.aidoc/designs/web-api.md` | Current API, recovery, and network boundary |
| `.aidoc/designs/e2e-api-scenarios.md` | Built-binary and service-example verification |
| [Sudoku UI deployment design](https://github.com/gnailuy/sudoku-ui/blob/master/.aidoc/designs/deployment-hardening.md) | Static artifact, mount, and browser contract |

## Why the Deployment Boundary Exists

A development installation should be easy to reproduce without coupling Sudoku to one host. The deployment boundary separates repository-owned artifacts and checks from operator-owned routing, credentials, service configuration, state, and release selection.

Sudoku may share a machine and reverse proxy with unrelated applications. Sudoku operations must affect only its bounded route, service, listener, state, and files; a Sudoku replacement must not rebuild, restart, roll back, or rewrite a neighboring application.

## Network and Host Policy

The API binds to loopback by default. A reverse proxy may expose the static client and API at an origin root or normalized path prefix, forwarding only the selected API namespace and removing the mount prefix when required.

Authentication is an optional host policy until the application has accounts and user authorization. Operators may add a reverse-proxy access gate, VPN, allowlist, or no gate according to the installation's exposure; credentials and policy remain outside repositories and browser assets. Direct non-loopback API binding retains the bearer-token requirement defined by `.aidoc/designs/web-api.md`.

Repository examples use generic values only. Public files contain no operator hostname, external URL, private address assignment, live port, absolute host layout, credential, or neighboring-site route.

## Artifact Contract

A trusted branch workflow builds and tests the backend, then makes the binary, Git commit, and SHA-256 checksum available to the deployment boundary. The frontend repository owns its static artifact and mount input; private host tooling may pair one successful backend artifact with one successful frontend artifact.

A pair record needs only the selected backend commit, frontend commit, checksums, and frontend mount input. The record may live in private host state. The repositories do not require a shared release framework or know which environment consumes the pair.

Pull-request artifacts are verification inputs only. Automatic deployment, when enabled, consumes successful trusted default-branch artifacts rather than artifacts produced by untrusted pull-request code.

## Service and State Contract

One unprivileged service runs the selected backend binary from an operator-chosen release directory. Configuration, puzzle data, API recovery records, and logs remain outside that directory so replacing application files does not replace state or expose host configuration.

`deploy/sudoku-api.service.example` demonstrates a home-relative installation, private XDG roots, a loopback listener, bounded restart backoff, and the API's ten-second graceful shutdown budget. Operators may adapt the release and state paths while preserving those boundaries.

The service starts after the user's service manager starts and can be enabled for ordinary host startup. Repository CI validates the example and proves graceful and forced process restart against the built binary; target-host startup remains an operator verification because CI does not control the host.

## Replacement and Failure Handling

A serialized host-side flow downloads or receives the selected artifacts, verifies their checksums, stages the pair away from the active files, and starts the backend on its private listener. The flow verifies backend health and a representative API session before selecting the new pair.

The companion browser verification checks the shell, referenced assets, health route, session creation, and one desktop/mobile gameplay journey with no unexpected request, page, or console errors. A shared host additionally checks that representative neighboring routes remain unchanged before and after Sudoku replacement.

A failed checksum, startup, health, asset, API, or browser check leaves the working pair selected or restores it. Development downtime and loss of active Sudoku sessions are acceptable; changing or destabilizing a neighboring application is not.

## Scope Boundary

Useful service logs and a payload-free health endpoint are part of the maintained contract. Elaborate monitoring, scheduled browser probes, backup/restore drills, immutable-release ceremony, multi-host coordination, availability objectives, and zero-downtime replacement are deferred until evidence makes them worthwhile.

Accounts, player identity, public multi-user authorization, rate limits, and abuse controls remain product work rather than deployment substitutes.
