---
domain: Designs
status: Active
entry_points:
  - cmd/api.go
  - webapi/server.go
  - .github/workflows/ci.yml
dependencies:
  - .aidoc/designs/deployment-hardening.md
  - .aidoc/designs/web-api.md
  - .aidoc/designs/e2e-test-scenarios.md
  - .aidoc/designs/future-directions.md
---

# Roadmap

The next approved milestone adds portable build and deployment support for a development-stage Sudoku service. The milestone favors simple replacement, verification, and isolation over production reliability ceremony.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/deployment-hardening.md` | Canonical portable deployment and replacement contract |
| `.aidoc/designs/web-api.md` | Current HTTP contract and loopback-safe defaults |
| `.aidoc/designs/e2e-test-scenarios.md` | Maintained black-box verification baseline |
| `.aidoc/designs/future-directions.md` | Deferred product, hosting, and technical directions |
| [Sudoku UI roadmap](https://github.com/gnailuy/sudoku-ui/blob/master/.aidoc/designs/roadmap.md) | Coordinated browser build and deployment responsibilities |

## Why Portable Deployment Comes Next

Sudoku remains a development project: refactoring, downtime, and replacement of active game sessions are acceptable. Deployment work exists to make each installation understandable and repeatable, not to imply production availability or compatibility guarantees.

A hosted Sudoku installation must remain isolated from unrelated applications on the same machine. The backend therefore keeps its own loopback listener, service, configuration, state, logs, and release location while a reverse proxy owns the bounded public route.

## Approved Delivery Sequence

1. Keep the backend build, tests, contract checks, and built-binary E2E lanes green.
2. Keep the trusted-branch backend artifact contract verifiable: all CI gates complete before the workflow publishes the executable and its commit-bound manifest.
3. Provide a host-neutral service example whose listener, release location, state roots, and allowed browser origins are operator inputs.
4. Define one serialized replacement flow: stage a frontend/backend pair, verify checksums, start and health-check the backend, verify the browser journey, then select the pair.
5. Leave or restore the previous working pair when staging, startup, health, asset, or browser verification fails.
6. Add automatic default-branch deployment only after the artifact and host-side replacement flow pass end to end.

Branch previews are deployment consumers rather than a backend automation requirement. An operator may install selected development-branch artifacts manually or with private host tooling without committing the preview hostname, port, branch selection, or host layout.

## Maintained Delivery Gates

- Pull-request CI keeps unit, race, vet, lint, API contract, API E2E, line-CLI E2E, and TUI PTY E2E independent and green.
- Deployment consumes only artifacts from trusted workflows; untrusted pull-request artifacts cannot replace a hosted installation.
- The backend binds to loopback behind a reverse proxy unless an operator explicitly chooses and secures another network boundary.
- A replacement verifies artifact identity, backend health, API session creation, and the companion desktop/mobile browser journey before completion.
- Repository files contain no private hostname, credential, operator path, live port assignment, release identifier, or neighboring-application topology.
- Authentication is an optional host policy until the application gains an account system; browser assets never contain host credentials or backend secrets.

Monitoring platforms, scheduled browser checks, backup drills, immutable-release frameworks, availability objectives, and zero-downtime promotion are not part of this development milestone. Concrete operational problems may justify separately reviewed additions later.
