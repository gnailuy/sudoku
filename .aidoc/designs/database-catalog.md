---
domain: Designs
status: Active
entry_points:
  - db/db.go
  - db/catalog.go
  - db/puzzle.go
dependencies:
  - .aidoc/designs/difficulty-model.md
  - .aidoc/designs/database-puzzle-selection.md
  - .aidoc/designs/database-play-statistics.md
  - .aidoc/designs/e2e-database-scenarios.md
---

# Base-Puzzle Catalog, Provenance, and Play Runs

The SQLite schema separates stable base-puzzle identity, independently traceable provenance, and presentation-specific play runs. The separation allows later imports and transformed presentations to converge on one authoritative strategy classification without erasing where a puzzle came from or which board a player saw.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/difficulty-model.md` | Defines the authoritative strategy classification stored on a base puzzle |
| `.aidoc/designs/database-puzzle-selection.md` | Defines acquisition and balanced reuse of base puzzles |
| `.aidoc/designs/database-play-statistics.md` | Defines acquisition and completion counters |
| `.aidoc/designs/e2e-database-scenarios.md` | Owns destructive rebuild and schema-boundary acceptance |

## Why Identity Is Separate

A published puzzle can appear with relabelled digits, transposed bands, reflected stacks, or another valid Sudoku presentation. Source text is therefore not durable identity. Catalog ingestion must first produce one symmetry- and digit-canonical puzzle string; `db.BasePuzzleID` then derives a deterministic `bp_` identifier from that content.

`core.CanonicalPuzzle` selects one dot-notation representative across digit relabelling, band and stack permutations, row and column permutations within those groups, and transposition. Import, generation, automatic storage, and completion tracking all use that same canonical boundary before classification, deduplication, or history lookup; `db.InsertPuzzle` still rejects a caller-supplied ID that disagrees with the canonical content.

The pinned external corpus is `grantm/sudoku-exchange-puzzle-bank`'s public-domain `diabolical.txt` at commit `d8c8ebaee0c08c412cfba96af1923dfa61c83317`. The import boundary verifies the complete file SHA-256 `08553d0c1145ea4d7c13008040f47ea8205d21fe1eaf8f4ab17a1a6981928b35`, preserves each published record hash as `source_ref`, reclassifies canonical content, and excludes `strategy-unsolved` records from the playable catalog.

## Catalog Contract

`base_puzzles` stores one row per canonical puzzle. `base_puzzle_id`, `canonical_puzzle`, authoritative strategy grade, score, and maximum technique describe the puzzle itself. Acquisition and completion aggregates remain attached to the base puzzle so existing selection and statistics behavior stays stable during the schema split.

`puzzle_provenance` stores any number of unique `(base_puzzle_id, source, source_ref)` records. Duplicate catalog insertion can add new provenance without replacing classification or history. A source label is descriptive; `source_ref` carries the source's immutable record identifier when one exists.

`play_runs` stores caller-owned run identity, the linked base-puzzle ID, the exact presented puzzle string, and `active`, `completed`, or `abandoned` status. Every tracked CLI, TUI, and HTTP session starts a run after presentation transformation; API recovery preserves the run ID instead of creating a second record. Presentation state never changes canonical catalog content.

A player-driven completion atomically changes the active run to `completed` and increments the linked base puzzle's completion count. Repeated completion observation cannot increment either record twice, while automatic solve remains excluded by the shared `playrun.Tracker` policy.

## Destructive Schema Boundary

Schema version 2 intentionally does not migrate the legacy `puzzles` table. Legacy rows lack enough canonicalization and provenance evidence to choose stable base-puzzle identities without guessing. `db.Open` returns `db.ErrRebuildRequired` and names the supported command instead of silently rewriting data.

`sudoku db rebuild --db <path> --yes` atomically drops the disposable catalog, provenance, play-run, and legacy tables, then creates schema version 2. Interactive use requires the exact word `rebuild`; non-interactive use requires `--yes`. The database file remains in place, and any failed transaction leaves the prior schema intact.

## Failure and Integrity Boundaries

- Foreign keys prevent provenance and play runs from referring to absent base puzzles.
- A canonical puzzle and its base-puzzle ID are independently unique.
- A play-run ID is caller-supplied and globally unique within one database.
- New play runs begin as `active`; status updates reject unknown states.
- Rebuild is the only supported transition from a legacy development schema.
- Catalog import never infers provenance from a filename, grade, or puzzle content; the pinned helper supplies an explicit commit-bound source label and each parsed record supplies its published source hash.

## Verification

Package tests prove full symmetry collapse, deterministic base-puzzle IDs, additive provenance on duplicate catalog content, independent transformed presentation storage, atomic run completion, constrained status, explicit legacy rejection, atomic rebuild, and existing acquisition/statistics behavior. Built-binary scenarios prove hash-pinned import and rebuild behavior plus exact presented-board linkage, stable base identity, completion status, and run-ID preservation across API restart recovery.
