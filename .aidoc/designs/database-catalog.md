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

The catalog boundary accepts canonical content rather than guessing normalization inside storage. The pinned-bank import slice owns the canonicalization algorithm and duplicate evidence. Until that slice lands, existing producers retain their current digit-normalized input behavior while using the new identity and provenance schema.

## Catalog Contract

`base_puzzles` stores one row per canonical puzzle. `base_puzzle_id`, `canonical_puzzle`, authoritative strategy grade, score, and maximum technique describe the puzzle itself. Acquisition and completion aggregates remain attached to the base puzzle so existing selection and statistics behavior stays stable during the schema split.

`puzzle_provenance` stores any number of unique `(base_puzzle_id, source, source_ref)` records. Duplicate catalog insertion can add new provenance without replacing classification or history. A source label is descriptive; `source_ref` carries the source's immutable record identifier when one exists.

`play_runs` stores caller-owned run identity, the linked base-puzzle ID, the exact presented puzzle string, and `active`, `completed`, or `abandoned` status. Presentation state never changes canonical catalog content. Frontend integration and transformed-session lifecycle remain the later end-to-end slice; the schema and domain methods establish the boundary now.

## Destructive Schema Boundary

Schema version 2 intentionally does not migrate the legacy `puzzles` table. Legacy rows lack enough canonicalization and provenance evidence to choose stable base-puzzle identities without guessing. `db.Open` returns `db.ErrRebuildRequired` and names the supported command instead of silently rewriting data.

`sudoku db rebuild --db <path> --yes` atomically drops the disposable catalog, provenance, play-run, and legacy tables, then creates schema version 2. Interactive use requires the exact word `rebuild`; non-interactive use requires `--yes`. The database file remains in place, and any failed transaction leaves the prior schema intact.

## Failure and Integrity Boundaries

- Foreign keys prevent provenance and play runs from referring to absent base puzzles.
- A canonical puzzle and its base-puzzle ID are independently unique.
- A play-run ID is caller-supplied and globally unique within one database.
- New play runs begin as `active`; status updates reject unknown states.
- Rebuild is the only supported transition from a legacy development schema.
- Catalog import never infers provenance from a filename, grade, or puzzle content.

## Verification

Package tests prove deterministic base-puzzle IDs, additive provenance on duplicate catalog content, independent transformed presentation storage, constrained run status, explicit legacy rejection, atomic rebuild, and existing acquisition/statistics behavior. The built-binary CLI scenario proves that a legacy database fails closed, non-interactive rebuild requires `--yes`, and a confirmed rebuild produces only the versioned catalog tables.
