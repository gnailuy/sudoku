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

The next approved milestone hardens the current single-operator deployment before any broader product expansion. Delivery proceeds in independently reversible slices while the existing engine, clients, and quality gates remain stable.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/deployment-hardening.md` | Canonical operating, paired-release, recovery, and acceptance contract |
| `.aidoc/designs/web-api.md` | Current HTTP contract, security boundary, and safe defaults |
| `.aidoc/designs/e2e-test-scenarios.md` | Maintained black-box verification baseline |
| `.aidoc/designs/future-directions.md` | Deferred product, hosting, and technical directions |
| [Sudoku UI roadmap](https://github.com/gnailuy/sudoku-ui/blob/master/.aidoc/designs/roadmap.md) | Coordinated browser-client and static-release responsibilities |

## Why Deployment Hardening Comes Next

The current private preview proves the browser and backend can run together, but an ad hoc preview is not an operational baseline. A dependable single-operator deployment needs an explicit access boundary, durable service lifecycle, actionable health signals, recoverable state, and a verified release rollback path before the project considers accounts or public multi-user hosting.

The deployment milestone changes operations rather than gameplay. Accounts, account-scoped authorization, public multi-tenancy, collaboration, and shared games remain non-goals for this milestone.

## Milestone 1: Deployment-Hardening Design

The approved [deployment-hardening design](deployment-hardening.md) is the canonical operating contract for exposure, authentication, paired immutable releases, durable service ownership, monitoring, backup consistency, isolated restore, rollback, and the full recovery exercise. The companion browser design owns mount-aware static artifacts, cache behavior, and desktop/phone evidence.

The design supports a dedicated origin or a path-prefixed Sudoku installation on a host shared with an unrelated site. Shared Caddy infrastructure does not couple application assets, processes, state, releases, checks, backups, or rollback.

The design milestone changes documentation only. Caddy, services, credentials, firewall rules, public routes, and persistent state remain unchanged until a separately reviewed implementation slice is approved for the target host.

## Milestone 2: Reversible Implementation Slices

Implementation follows the reviewed design in this order:

1. maintain the implemented exposure/authentication boundary across the backend and browser repositories;
2. maintain the portable durable API service contract and built-binary graceful/forced-restart proof, then install it and prove unattended startup on an approved target host;
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
