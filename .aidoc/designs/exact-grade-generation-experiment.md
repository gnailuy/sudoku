---
domain: Designs
status: Active
entry_points:
  - generator/generator.go
  - cmd/replenish.go
  - replenisher/replenisher.go
  - db/puzzle.go
  - solver/classify.go
  - core/canonical.go
dependencies:
  - .aidoc/designs/difficulty-model.md
  - .aidoc/designs/database-catalog.md
  - .aidoc/designs/database-puzzle-selection.md
---

# Exact-Grade Catalog Replenishment

The offline Expert/Evil catalog replenisher uses deterministic exact-seed clue addition while keeping strategy grades, canonical identity, provenance, and catalog-first play authoritative. Experiment inputs, observations, and reports are not product artifacts and do not belong in the repository.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/difficulty-model.md` | Canonical strategy grades and within-grade score contract |
| `.aidoc/designs/database-catalog.md` | Symmetry-canonical identity and source provenance |
| `.aidoc/designs/database-puzzle-selection.md` | Current catalog acquisition and fallback behavior |
| `.aidoc/designs/roadmap.md` | Approved product scope and delivery boundaries |

## Why Offline Replenishment Is the Product Boundary

Catalog-first play already provides reliable exact Hard, Expert, and Evil puzzles without making generation latency part of an interactive request. Offline replenishment can expand supply under controlled budgets while keeping player-facing creation deterministic and responsive.

The validated clue-addition direction starts from an exact-grade puzzle and restores a correct solution clue before authoritative reclassification. The direction uses existing solver feedback without changing strategy order, grade labels, score semantics, or the distinction between Evil and `strategy-unsolved`.

## Product Scope

`sudoku replenish` builds offline Expert or Evil batches from exact-grade catalog seeds. Every accepted puzzle must pass `solver.ClassifyPuzzle`, remain in its requested exact grade, receive a unique `core.CanonicalPuzzle` identity, preserve source and derivation provenance, and enter the catalog through the existing storage boundary.

Hard remains excluded because the decision evidence was inconclusive. Medium remains excluded because the current pinned catalog does not provide an approved exact-grade seed source. Neither grade may enter the replenisher through relabeling or an unreviewed seed source.

## Product Invariants

- Interactive CLI and API creation keep their current deadlines and fallback behavior.
- Catalog acquisition remains the default path for high-grade play.
- Deterministic classification remains authoritative; within-grade score never overrides a grade.
- Canonical duplicate rejection covers digit relabelling and Sudoku-preserving symmetry.
- Replenishment records enough provenance to identify the seed and policy that produced each accepted puzzle.
- Replenishment never mutates player sessions or the live catalog until a complete candidate batch passes validation.

## Repository Boundary

The repository records the product decision and implementation constraints, not one-off experiment manifests, observations, checkpoints, or reports. Version control preserves the decision's development history; current documentation describes only the approved product direction.

`replenisher.Run` owns deterministic search, durable state, duplicate rejection, and complete-batch publication. `solver.ClassifyPuzzle`, `core.CanonicalPuzzle`, and `db.PublishPuzzleBatch` remain the authoritative validation and storage boundaries. The existing `generationexperiment` command remains isolated from interactive and API paths and is not a product dependency.

## Resume and Publication Contract

The required state file records immutable command configuration, classifications consumed, accepted candidates, and publication status. Each classification replaces that file atomically with mode `0600`; rerunning the same command resumes it, while changed configuration is rejected. A signal stops after the most recent durable checkpoint.

The live catalog is read-only during search. Publication begins only after the requested count is complete, inserts every base puzzle and derivation provenance row in one transaction, and rejects partial collisions. A fully matching existing batch is an idempotent successful resume for the narrow crash window after the database commit.
