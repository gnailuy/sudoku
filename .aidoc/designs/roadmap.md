---
domain: Designs
status: Active
entry_points:
  - generator/generator.go
  - solver/classify.go
  - core/canonical.go
dependencies:
  - .aidoc/designs/exact-grade-generation-experiment.md
  - .aidoc/designs/difficulty-calibration.md
  - .aidoc/designs/deployment-hardening.md
  - .aidoc/designs/e2e-test-scenarios.md
---

# Roadmap

The active milestone is the isolated exact-grade generation experiment for Hard, Expert, and Evil. The resumable harness, seeded baseline, trace-guided candidate, and explicit CLI are in place; exploratory policy measurement and one final held-out evaluation remain before any grade-specific implementation proposal.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/exact-grade-generation-experiment.md` | Canonical experiment boundary, measurements, and advancement gate |
| `.aidoc/designs/difficulty-calibration.md` | Existing baseline evidence and reproducibility model |
| `.aidoc/designs/difficulty-model.md` | Fixed deterministic grade and within-grade score contract |
| `.aidoc/designs/deployment-hardening.md` | Maintained portable deployment and replacement contract |
| `.aidoc/designs/e2e-test-scenarios.md` | Maintained black-box verification baseline |
| `.aidoc/designs/future-directions.md` | Other deferred product, hosting, and technical directions |

## Why Exact-Grade Generation Comes Next

The catalog-first milestone made exact Hard, Expert, and Evil play reliable by importing a pinned public bank and separating canonical puzzle identity from presentation-specific runs. The remaining generator question is not required for current play; it is whether classifier feedback can produce new exact-grade puzzles more efficiently than independent generate-and-classify rounds.

The current baseline has weak exact-hit evidence for Hard and Expert and only partial Evil alignment. Trace-guided mutation is the smallest candidate that uses existing solve traces and catalog seeds without changing the five grades, strategy inventory, solver order, or `strategy-unsolved` semantics.

## Approved Experiment Sequence

1. Materialize immutable exploratory and held-out seed manifests for Hard, Expert, and Evil using distinct canonical base-puzzle IDs and fixed equal budgets.
2. Run `sudoku experiment generation` on the exploratory split and preserve every failed, duplicate, wrong-grade, strategy-unsolved, and timed-out observation.
3. Tune only the bounded clue-mutation policy against exploratory evidence, keeping deterministic grades, solver configuration, and baseline budgets fixed.
4. Freeze the selected policy, run it once on held-out seeds, and publish raw counts, uncertainty, exact-hit yield, cost, failure shape, diversity, and replay evidence.
5. Open a separate grade-specific implementation proposal only where held-out evidence passes the advancement gate.

Medium is excluded from the first experiment because the current pinned bank does not provide Medium seeds. A Medium arm requires a separately reviewed seed-independent method or a new traceable seed source.

## Advancement Gate

A candidate advances only when held-out evidence shows higher unique exact-grade yield than the baseline for at least one target grade, without worse reproducibility or any deterministic-classification violation. An improvement must remain meaningful after raw sample counts and uncertainty are considered.

A successful experiment authorizes only a separate implementation proposal for the successful grade. A failed or inconclusive experiment leaves `generator.GenerateBestEffort`, API behavior, interactive deadlines, catalog acquisition, fallback semantics, and visible labels unchanged.

## Maintained Quality and Delivery Gates

- Pull-request CI keeps unit, race, vet, lint, API contract, API E2E, line-CLI E2E, and TUI PTY E2E independent and green.
- Every experiment observation binds the repository identity, solver configuration, seed manifest, budget, and policy version.
- Canonical equivalence controls uniqueness so digit relabelling or Sudoku-preserving symmetry cannot inflate yield.
- `strategy-unsolved` remains separate from Evil, and score remains an ordering signal only within a completed strategy grade.
- Repository files contain no private hostname, credential, operator path, live port assignment, release identifier, or neighboring-application topology.
- Trusted deployment artifacts and the existing host-neutral replacement contract remain maintained; the experiment does not alter deployment.

Strategy-aware construction, blind budget expansion, player-difficulty labels, accounts, multi-user hosting, large-import optimization, and production reliability ceremony remain outside this milestone. Concrete evidence and separately approved designs are required before any of those directions enter the roadmap.
