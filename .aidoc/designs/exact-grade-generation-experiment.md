---
domain: Designs
status: Active
entry_points:
  - generator/generator.go
  - solver/classify.go
  - core/canonical.go
dependencies:
  - .aidoc/designs/difficulty-model.md
  - .aidoc/designs/database-catalog.md
  - .aidoc/designs/database-puzzle-selection.md
---

# Exact-Grade Catalog Replenishment Decision

Exact-seed clue addition is the approved direction for an offline Expert/Evil catalog replenisher. The product keeps deterministic grades, canonical identity, provenance, and catalog-first play authoritative; experiment inputs, observations, and reports are not product artifacts and do not belong in the repository.

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

## Approved Product Scope

A separately reviewed product change may add an offline replenisher for Expert and Evil. Every accepted puzzle must pass `solver.ClassifyPuzzle`, remain in its requested exact grade, receive a unique `core.CanonicalPuzzle` identity, preserve source and derivation provenance, and enter the catalog through the existing storage boundary.

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

`generator.GenerateBestEffort`, `solver.ClassifyPuzzle`, `core.CanonicalPuzzle`, and the catalog persistence APIs are the implementation boundaries for the separately reviewed replenisher. The existing `generationexperiment` command remains isolated from interactive and API paths and is not a product dependency.
