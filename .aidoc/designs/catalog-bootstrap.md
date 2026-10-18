---
domain: Designs
status: Active
entry_points:
  - catalogbundle/bundle.go
  - db/catalog_bundle.go
  - cmd/catalog.go
  - tools/build-catalog-bundle/main.go
dependencies:
  - .aidoc/designs/database-catalog.md
  - .aidoc/designs/database-puzzle-selection.md
  - .aidoc/designs/e2e-database-scenarios.md
---

# Embedded Catalog Bootstrap

The bootstrap boundary treats starter puzzles as versioned application reference data rather than a copied live SQLite database. Managed per-user databases receive validated additive releases automatically, while explicit database paths remain under caller control.

## Related Docs

| Document | Relationship |
|---|---|
| [Database catalog](database-catalog.md) | Owns canonical puzzle identity, provenance, cohorts, and live history |
| [Puzzle selection](database-puzzle-selection.md) | Defines how catalog supply becomes playable |
| [E2E database scenarios](e2e-database-scenarios.md) | Proves managed and explicit initialization through the built binary |
| [Difficulty model](difficulty-model.md) | Owns the strategy classification stored in each bundle row |

## Why the Bundle Is Not SQLite

A database file would mix schema state with application data and could accidentally carry counters, play runs, accounts, sessions, or saved games. The embedded JSON artifact contains only immutable catalog facts and separately approved cohort definitions. A companion SHA-256 file binds the exact artifact bytes, while the solver-configuration digest prevents classifications from being applied under a different strategy registry, tier assignment, or weight configuration.

Bundle release 2 ships 256 deterministic public-domain puzzles for each of Easy, Medium, Hard, and Expert plus all 2,023 members of the approved `evil-v1` cohort. The 3,047 canonical puzzles are reproduced from the hash-pinned Sudoku Exchange bank and frozen full-catalog audit rather than copied from a live database. The bounded non-Evil supply gives each grade substantial never-played rotation while keeping the embedded artifact small; Evil preserves the complete audited serving population instead of sampling it.

`tools/build-catalog-bundle` validates the three source-bank SHA-256 values, frozen report and evidence hashes, canonical puzzle identities, solver digest, per-grade counts, and complete Evil membership before replacing the reviewable JSON artifact. Release metadata retains the public-domain license and immutable source-record identifier for every puzzle.

The release-2 JSON is 1,707,150 bytes and compresses to 338,898 bytes with gzip level 9. A clean Linux amd64 Go build measured a 1,703,496-byte raw binary increase and a 341,046-byte compressed release increase over the empty release-1 bundle. This keeps the distributed cost near one third of a megabyte while preserving a directly reviewable artifact.

## Managed and Explicit Locations

The managed database is per-user application data: Linux uses `$XDG_DATA_HOME/sudoku/puzzles.db` or `~/.local/share/sudoku/puzzles.db`; macOS uses `~/Library/Application Support/Sudoku/puzzles.db`; Windows uses `%LOCALAPPDATA%\Sudoku\puzzles.db`. Missing platform location information fails closed instead of creating a database in the working directory.

A command without `--db` creates the managed parent directory, opens the schema, and applies the embedded release before catalog work. An explicit `--db` path never seeds implicitly. `sudoku db bootstrap --db <path>` applies the same release deliberately, and `--check` runs the conflict and addition analysis without committing catalog rows or a release record.

## Additive Release Contract

`catalog_bundle_releases` records the monotonic version, artifact digest, solver digest, puzzle count, cohort count, and application time. Reapplying the same version and digest is a no-op; the same version with different bytes is a conflict; an older binary leaves a newer recorded release untouched.

A newer release runs in one transaction. Missing canonical puzzles and official provenance are added, while existing grade, score, maximum technique, acquisition/completion history, user imports, play runs, accounts, sessions, and saved games remain unchanged. Conflicting immutable puzzle metadata aborts the release. Cohort definitions and exact membership are likewise immutable; bundle inclusion never weakens Evil's separately approved serving policy.

## User Imports

`sudoku import` remains the public extension boundary for user-provided records. Imports classify and canonicalize through the existing pipeline, add independent provenance for duplicates, and never join an official serving cohort merely because a grade matches. Bootstrap, import, and replenishment share catalog identity but retain distinct trust and publication rules.

## Verification

Package tests cover exact five-grade counts, complete Evil membership, artifact and solver-digest drift, dry-run rollback, idempotent replay, additive provenance, preserved acquisition history, conflict rollback, and managed-versus-explicit initialization. The built-binary database scenario verifies automatic first use, exact grade/cohort materialization, explicit check/apply, ledger persistence, and schema rebuild composition without network access.
