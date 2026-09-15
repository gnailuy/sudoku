---
domain: Designs
status: Active
entry_points:
  - cmd/api.go
  - webapi/server.go
  - .github/workflows/ci.yml
dependencies:
  - .aidoc/designs/web-api.md
  - .aidoc/designs/e2e-test-scenarios.md
  - .aidoc/designs/future-directions.md
---

# Roadmap

The next approved milestone hardens the current single-operator deployment before any broader product expansion. Delivery proceeds in independently reversible slices while the existing engine, clients, and quality gates remain stable.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/web-api.md` | Current HTTP contract, security boundary, and safe defaults |
| `.aidoc/designs/e2e-test-scenarios.md` | Maintained black-box verification baseline |
| `.aidoc/designs/future-directions.md` | Deferred product, hosting, and technical directions |
| [Sudoku UI roadmap](https://github.com/gnailuy/sudoku-ui/blob/master/.aidoc/designs/roadmap.md) | Coordinated browser-client and static-release responsibilities |

## Why Deployment Hardening Comes Next

The current private preview proves the browser and backend can run together, but an ad hoc preview is not an operational baseline. A dependable single-operator deployment needs an explicit access boundary, durable service lifecycle, actionable health signals, recoverable state, and a verified release rollback path before the project considers accounts or public multi-user hosting.

The deployment milestone changes operations rather than gameplay. Accounts, account-scoped authorization, public multi-tenancy, collaboration, and shared games remain non-goals for this milestone.

## Milestone 1: Deployment-Hardening Design

The design defines one coordinated operating contract for the backend and browser client before host configuration changes begin:

1. **Exposure and authentication boundary:** state who may reach the single-operator deployment, where authentication is enforced, which routes are public for health checks, and how credentials are created, rotated, and revoked.
2. **Durable service lifecycle:** define installation paths, process ownership, startup ordering, restart policy, graceful shutdown, state directories, and behavior after host reboot.
3. **Health monitoring and alerts:** distinguish process liveness, API readiness, static-release availability, storage failures, and actionable operator alerts.
4. **Backup and restore:** cover the Sudoku database, service configuration, and versioned release artifacts; define retention, integrity checks, and a timed restore drill.
5. **Release, rollback, and recovery:** define immutable release identification, preflight checks, deployment order, rollback triggers, previous-release restoration, and final end-to-end recovery verification.

The design includes acceptance criteria, failure tests, rollback proof, ownership boundaries, and explicit non-goals. It does not modify Caddy, services, credentials, firewall rules, or public exposure.

## Milestone 2: Reversible Implementation Slices

Implementation follows the reviewed design in this order:

1. enforce and verify the exposure/authentication boundary;
2. install durable backend and static-web service lifecycles with restart and reboot proof;
3. add health monitoring and actionable failure alerts;
4. automate bounded backups and complete a restore drill;
5. publish a versioned release and prove rollback to the prior version;
6. run the complete recovery exercise and record the operating evidence.

Each slice must be independently reviewable, include its own failure test, and leave a verified rollback path. Shared-host changes require explicit operator approval at the point of application.

## Maintained Delivery Gates

- Pull-request CI keeps unit, race, vet, lint, API contract, API E2E, line-CLI E2E, and TUI PTY E2E independent and green.
- Black-box verification runs against built artifacts with isolated deterministic state.
- Deployment changes preserve loopback-safe backend defaults and the client-neutral API contract.
- Documentation and operating procedures remain portable; repository files contain no private hostnames, credentials, user paths, or environment-specific secrets.
- The milestone is complete only after access, restart, alert, restore, rollback, and full recovery evidence all pass.
