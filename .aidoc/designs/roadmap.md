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

The isolated exact-grade generation experiment is complete. Held-out evidence authorizes a separately reviewed Expert/Evil implementation proposal; Hard remains evidence-gated because its confidence interval still overlaps the baseline.

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

## Approved Next Proposal

The next proposal may design an offline Expert/Evil replenishment path around the validated exact-seed clue-addition policy. The proposal must keep deterministic classification authoritative, preserve canonical duplicate rejection and provenance, and remain separate from normal catalog acquisition until independently reviewed.

Hard stays out of the proposal because 10/10 candidate hits versus 5/10 baseline hits still produced narrowly overlapping Wilson intervals. Medium stays out because the pinned bank does not provide Medium seeds. Neither grade may be included through relabeling or an unreviewed seed source.

## Advancement Gate

The confirmatory held-out cohort produced 10/10 unique exact-grade candidate outputs for each of Hard, Expert, and Evil. The equal-budget baseline produced 5/10 Hard and 0/10 Expert or Evil outputs; an independent replay reproduced all 60 semantic observations exactly.

Expert and Evil pass because their candidate and baseline Wilson intervals do not overlap. Hard remains inconclusive because its intervals overlap narrowly. The current result leaves `generator.GenerateBestEffort`, API behavior, interactive deadlines, catalog acquisition, fallback semantics, and visible labels unchanged.

## Maintained Quality and Delivery Gates

- Pull-request CI keeps unit, race, vet, lint, API contract, API E2E, line-CLI E2E, and TUI PTY E2E independent and green.
- Every experiment observation binds the repository identity, solver configuration, seed manifest, budget, and policy version.
- Canonical equivalence controls uniqueness so digit relabelling or Sudoku-preserving symmetry cannot inflate yield.
- `strategy-unsolved` remains separate from Evil, and score remains an ordering signal only within a completed strategy grade.
- Repository files contain no private hostname, credential, operator path, live port assignment, release identifier, or neighboring-application topology.
- Trusted deployment artifacts and the existing host-neutral replacement contract remain maintained; the experiment does not alter deployment.

Strategy-aware construction, blind budget expansion, player-difficulty labels, accounts, multi-user hosting, large-import optimization, and production reliability ceremony remain outside this milestone. Concrete evidence and separately approved designs are required before any of those directions enter the roadmap.
