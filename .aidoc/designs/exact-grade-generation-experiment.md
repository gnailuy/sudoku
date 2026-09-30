---
domain: Designs
status: Active
entry_points:
  - generationexperiment/runner.go
  - generationexperiment/executors.go
  - cmd/experiment.go
  - generator/generator.go
  - solver/classify.go
  - core/canonical.go
dependencies:
  - .aidoc/designs/difficulty-model.md
  - .aidoc/designs/difficulty-calibration.md
  - .aidoc/designs/database-catalog.md
---

# Exact-Grade Generation Experiment

The completed generator experiment compares trace-guided mutation with the current generate-from-scratch baseline for exact Hard, Expert, and Evil strategy grades. The frozen policy passed the held-out advancement gate for Expert and Evil without changing product behavior; Hard improved but remains independently inconclusive.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/difficulty-model.md` | Canonical strategy grades and within-grade score contract |
| `.aidoc/designs/difficulty-calibration.md` | Reproducible corpus, measurement, and held-out evaluation rules |
| `.aidoc/designs/database-catalog.md` | Symmetry-canonical identity and source provenance |
| `.aidoc/designs/roadmap.md` | Approved experiment sequence and delivery gate |

## Why Trace-Guided Mutation Is the Candidate

The current `generator.GenerateBestEffort` repeatedly creates a solved board, removes clues under tier guidance, and classifies each completed result. Existing calibration found exact target hits in 10/10 Easy calls, 1/10 Hard calls, 0/10 Medium and Expert calls, and 4/10 Evil calls. Increasing blind rounds would spend more of the same search budget without using the classifier's explanation of why a near miss landed in another grade.

The canonical catalog provides many exact Hard, Expert, and Evil starting points with stable symmetry-normalized identity. Trace-guided mutation can therefore test a narrow question: whether clue edits informed by the deterministic solve trace find new exact-grade puzzles more efficiently than independent generation, without changing grades, strategy order, or labels.

Strategy-aware construction remains a larger algorithmic commitment, and catalog-only offline replenishment does not improve generation. Both alternatives stay deferred until the bounded mutation experiment shows whether classifier feedback is useful.

## Experiment Boundary

The experiment has two arms under identical per-sample budgets:

1. The baseline arm calls the existing generate-from-scratch path and records every completed classification.
2. The candidate arm starts from an exact-grade catalog seed, applies one bounded clue mutation at a time, preserves validity and unique solvability, then classifies the candidate with `solver.ClassifyPuzzle`.

A candidate mutation may use the ordered trace, highest technique, stall point, clue count, and within-grade score to choose its next edit. A candidate mutation must not alter solver registration order, strategy implementations, tier assignment, weights, or the classification result.

The first experiment covers Hard, Expert, and Evil because the pinned catalog supplies exact-grade seeds for those strata. Medium requires a separately approved seed-independent arm or a new traceable seed source; the experiment must not relabel Hard puzzles or tune against generated Medium near misses to manufacture a seed set.

## Reproducibility Contract

Each arm uses a fixed manifest of canonical seed IDs, deterministic random seeds, identical wall-clock and classification-count budgets, and explicit repository and solver-configuration identities. Exploratory and held-out seed sets are disjoint by canonical base-puzzle ID.

The exploratory split may select mutation policies and bounded parameters. The held-out split runs each final candidate once and cannot feed further tuning. Reports preserve unsuccessful, wrong-grade, duplicate, invalid, non-unique, and `strategy-unsolved` outcomes rather than replacing them with successful samples.

Every accepted result passes `core.CanonicalPuzzle` before uniqueness accounting. A result equal to any experiment seed or earlier result under Sudoku-preserving symmetry and digit relabelling counts as a duplicate, not a new exact-grade hit.

## Held-Out Result and Decision

The frozen `trace-mutation-v2-exact-seed-addition` policy restored one correct clue when starting from an exact-grade seed. The confirmatory cohort used 30 disjoint held-out seeds from the hash-pinned Sudoku Exchange bank prefix, with ten seeds per grade and equal 2,000 ms and 12-classification budgets.

| Grade | Baseline exact hits | Candidate exact hits | Candidate decision |
|----------|-------------:|-------------:|---|
| Hard | 5/10 | 10/10 | Promising; Wilson intervals still overlap narrowly |
| Expert | 0/10 | 10/10 | Pass |
| Evil | 0/10 | 10/10 | Pass |

The candidate produced 30 distinct canonical exact-grade outputs and reproduced every outcome, puzzle, grade, score, maximum technique, trace digest, and classification count on an independent replay. The baseline produced five exact-grade outputs. `generationexperiment/baselines/exact-grade-held-out-confirmatory-v2/analysis.md` preserves raw counts, Wilson intervals, cost, failure shape, and the decision.

The result authorizes a separate implementation proposal for Expert and Evil only. Hard needs more independent evidence before its own proposal. The experiment does not authorize changing interactive deadlines, catalog acquisition, fallback behavior, solver semantics, or the five visible grade names.

## Safety and Isolation

Experiment commands write only to an explicit output directory and never mutate the live puzzle database. Catalog inputs are opened read-only or exported into an immutable manifest before measurement. Interruptions preserve completed observations, and resume rejects a changed manifest, repository identity, solver configuration, or experiment policy.

The experiment stores provenance and aggregate measurements but no player data. Published fixtures must respect their source license and redistribution terms; restricted puzzle strings remain local while hashes and aggregate results may be reported.

## Harness Boundary

`generationexperiment.Run` owns immutable manifest validation, stable baseline/candidate job ordering, append-only observations, resumable checkpoints, canonical duplicate accounting, and deterministic raw-count reports. Both arms receive the same manifest budget and sample seed through the `generationexperiment.Executor` boundary; execution adapters cannot write harness state or classify duplicates themselves.

`generationexperiment.NewExecutor` connects the current generate-from-scratch baseline and bounded trace-guided clue mutation. Both adapters use the manifest random seed, classification limit, and wall-clock budget; the candidate moves toward the target grade by removing clues after lower or exact grades and restoring solution clues after higher or stalled classifications. `cmd.newGenerationExperimentCommand` exposes only explicit manifest and output paths and never opens the live catalog.

The harness rejects seed reuse under Sudoku symmetry and digit relabelling, changed manifests, observations bound to another job, checkpoints ahead of durable observations, and execution metrics beyond either shared budget limit. A failed executor leaves prior observations resumable without manufacturing an outcome.

## Implementation Pointers

`generator.GenerateBestEffort` is the baseline boundary. `solver.ClassifyPuzzle` provides the authoritative outcome and trace, and `core.CanonicalPuzzle` provides equivalence identity. `generationexperiment.executeCandidate` owns the frozen bounded mutation policy; the manifests and durable evidence live under `generationexperiment/testdata/` and `generationexperiment/baselines/` without entering an interactive or API path.
