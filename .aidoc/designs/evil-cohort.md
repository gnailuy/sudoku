---
domain: Designs
status: Active
entry_points:
  - evilcohort/cohort.go
  - cmd/database.go
  - db/puzzle.go
dependencies:
  - .aidoc/designs/difficulty-calibration.md
  - .aidoc/designs/difficulty-model.md
  - .aidoc/designs/database-puzzle-selection.md
  - .aidoc/designs/roadmap.md
---

# Evil Serving Cohort

The Evil serving cohort combines independent upper-tail evidence from Sukaku Explainer and the canonical strategy trace. Schema version 5 materializes the approved membership by canonical base-puzzle ID, and Evil acquisition now fails closed unless that bound cohort is available.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/difficulty-calibration.md` | Immutable evidence and full-catalog audit contract |
| `.aidoc/designs/difficulty-model.md` | Authoritative five-grade strategy model |
| `.aidoc/designs/database-puzzle-selection.md` | Current exact-grade acquisition and reuse behavior |
| `.aidoc/designs/roadmap.md` | Delivery order and implementation gate |

## Why a Cohort Exists

The playable catalog contains 43,017 exact-Evil puzzles, but exact grade alone requires only one Evil-tier deduction. The full-catalog audit shows that the upstream rating and canonical advanced-move density provide independent positive signals, while canonical weighted score and technique diversity correlate negatively with the upstream rating. A single score threshold would therefore select the wrong tail.

The cohort narrows Evil serving without changing public grade names, solver order, weights, catalog admission, or non-Evil acquisition. The cohort is a selection policy, not a new difficulty model and not a claim about universal human experience.

## Frozen Population and Partition

The eligible population contains only canonical base puzzles whose deterministic classification outcome is `solved` and whose exact strategy grade is `evil`. Strategy-unsolved records and lower grades are excluded before thresholds are calculated.

Canonical `base_puzzle_id` identity collapses digit relabelings and Sudoku-preserving symmetries. Duplicate source records map to one base puzzle and cannot receive extra weight. Presented transformations inherit their base puzzle's cohort membership rather than being scored separately.

The first 64 bits of the hexadecimal canonical hash in `base_puzzle_id`, interpreted as an unsigned big-endian integer, define the stable partition. Remainder zero modulo five is held out; the other four remainders are exploratory. The current catalog contains 34,389 exploratory and 8,628 held-out exact-Evil base puzzles.

The full audit exposed every record before this post-audit partition existed, so the held-out set is a locked validation partition rather than a historically blind sample. Future threshold changes may use exploratory evidence only; held-out results remain a release gate and must not become tuning input.

## Frozen Membership Rule

Exploratory exact-Evil records define deterministic nearest-rank percentiles. A base puzzle belongs to the cohort only when all conditions hold:

1. Sukaku Explainer rating is at least the exploratory 80th percentile: `7.0`.
2. Canonical advanced-move density is at least the exploratory 80th percentile: `1/6`.
3. The canonical trace contains at least two Evil-tier moves.
4. The canonical trace contains at least three distinct advanced techniques.

The independent rating and density gates prevent either model from dominating membership. The move and diversity floors reject one-off threshold crossings with too little explainable advanced evidence. Canonical traces make the rule invariant under digit relabeling and board symmetry.

## Stability Evidence

The full-catalog artifact is bound to manifest `65689133e5019f881923762ed74bbc66f682835ba89981ccb748454a6ed1b970`, repository commit `d2f4a3c585f5cc0aa50b6d0bf09ed7e4c73e71b6`, and solver configuration `40af00b6619ec7aa17baeb3b7ce741cf8a7b895a0bd6d0376bb357c462eedd26`.

| Joint percentile | Rating gate | Density gate | Exploratory | Held-out | Combined |
|---:|---:|---:|---:|---:|---:|
| 75th | 6.9 | 0.157895 | 2,839 (8.26%) | 732 (8.48%) | 3,571 |
| **80th** | **7.0** | **0.166667** | **1,604 (4.66%)** | **419 (4.86%)** | **2,023** |
| 85th | 7.1 | 0.179104 | 895 (2.60%) | 226 (2.62%) | 1,121 |

The frozen rule's exploratory rate has a Wilson 95% interval of 4.45%–4.89%; the held-out rate is 4.86% with a 4.42%–5.33% interval. The 0.19 percentage-point difference shows no observed partition cliff, and the selected 2,023-puzzle cohort falls inside the predeclared 1,500–3,000 target.

At the frozen percentile gates, lowering both evidence floors to one selects 2,183 records; requiring two Evil moves and two advanced techniques selects the same 2,023 records as the frozen rule; requiring three Evil moves selects 1,750. Nearby evidence floors therefore change size gradually rather than exposing a brittle single-value boundary.

## Materialization and Serving Contract

`sudoku db materialize-evil-cohort` validates the immutable full-catalog report and evidence hashes, reproduces the frozen population and membership counts, derives each `base_puzzle_id` from the canonical puzzle hash, and atomically replaces `evil-v1`. Materialization fails without changing prior membership if any selected exact-Evil base puzzle is absent from the catalog.

`db.DB.AcquireForPlay` restricts Evil selection to `evil-v1`, then preserves never-played-first, lowest-play-count, and oldest-recency ordering inside that set. An absent, empty, invalid, or inaccessible cohort fails immediately; root play and `--from-db` never enter bounded generation or serve a weaker fallback for that request. Non-Evil acquisition remains unchanged.

The stored definition binds the manifest, repository commit, solver configuration, evidence hash, rule version, and member count. Any solver or evidence-version change invalidates membership and requires a new exploratory analysis plus held-out release review.
