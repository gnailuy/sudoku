---
domain: Designs
status: Active
entry_points:
  - difficultyaudit/audit.go
  - cmd/difficulty_audit.go
  - solver/classify.go
dependencies:
  - .aidoc/designs/difficulty-calibration.md
  - .aidoc/designs/difficulty-model.md
  - .aidoc/designs/database-catalog.md
  - .aidoc/designs/e2e-calibration-scenarios.md
---

# Roadmap

The next approved milestone uses reproducible full-catalog evidence to make Evil consistently demanding without adding a visible level or depending on human ratings. Delivery preserves the canonical five strategy grades and current serving behavior until an independent-rating and solver-trace audit supports a stable cohort rule.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/difficulty-calibration.md` | Immutable corpus, measurement, and full-catalog audit contract |
| `.aidoc/designs/difficulty-model.md` | Authoritative strategy grades and within-grade score |
| `.aidoc/designs/database-catalog.md` | Pinned Sudoku Exchange provenance and playable-catalog boundary |
| `.aidoc/designs/e2e-calibration-scenarios.md` | Built-binary evidence-artifact acceptance |
| `.aidoc/designs/future-directions.md` | Deferred player-difficulty and solver-expansion directions |

## Why Evidence Comes First

The exact-Evil catalog is large enough for curation, but `db.DB.AcquireForPlay` balances exact-grade puzzles without considering their within-grade intensity. A puzzle can therefore qualify as Evil after one Evil-tier deduction while remaining mild through most of its trace.

The pinned Sudoku Exchange bank carries a finite Sukaku Explainer rating for each published record. The independent source rating and the canonical solver trace provide two reproducible models that can identify robust upper-tail candidates without claiming universal human difficulty.

A cohort formula chosen after inspecting final membership would overfit the catalog. The audit therefore publishes complete distributions and cross-model agreement before any exploratory/held-out split, percentile gate, diversity minimum, sensitivity range, or target cohort size becomes serving policy.

## Approved Delivery Sequence

1. Preserve every finite Sudoku Exchange rating in an immutable version 2 analysis manifest while keeping strategy-unsolved records outside the playable catalog.
2. Produce a commit- and solver-configuration-bound full-catalog artifact with per-puzzle source rating, strategy outcome and grade, score, technique counts, Evil-move count, advanced-technique diversity and density, trace length, clue count, and trace digest. Publish deterministic distributions, correlations, missing-data counts, and cross-model agreement without changing acquisition.
3. Predeclare an exploratory/held-out split, independent high-percentile gates, minimum Evil-move and diversity requirements, duplicate and symmetry policy, cohort-size target, and nearby-threshold sensitivity tests. Review the frozen rule and stability report before implementation.
4. If the cohort is sufficiently large and stable, materialize deterministic membership and preserve never-played-first plus balanced reuse inside the eligible Evil cohort. An unavailable cohort must not silently fall back to weaker Evil puzzles.

## Delivery Gates

- Full-catalog inputs bind the exact bank SHA-256, source-record hashes, repository commit, solver configuration digest, output schema, and command boundary.
- Repeated classification of every puzzle produces the same outcome, grade, score, maximum technique, and trace digest.
- Per-puzzle evidence remains in source-manifest order even when classification uses parallel workers.
- Reports distinguish solved and strategy-unsolved outcomes, preserve every independent rating without normalization, and expose complete metric distributions and correlation counts.
- Unit and built-binary E2E coverage prove deterministic artifacts, immutable output paths, source-commit binding, and representative report fields.
- Player-facing grade names, solver order, weights, catalog admission, acquisition, and API behavior remain unchanged until a later evidence review approves the cohort rule.

Solver-inventory expansion remains a separate evidence-gated track because a new strategy can reclassify puzzles downward as well as explain current stalls. New puzzle sources are unnecessary unless they add compatible independent evidence or materially different solving patterns.
