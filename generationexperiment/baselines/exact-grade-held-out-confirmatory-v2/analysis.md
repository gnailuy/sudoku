# Confirmatory Exact-Grade Experiment

The frozen `trace-mutation-v2-exact-seed-addition` policy was evaluated once on 30 disjoint held-out seeds selected deterministically from the first 2,000 source-order records of Sudoku Exchange commit `d8c8ebaee0c08c412cfba96af1923dfa61c83317`; the complete source hash is `08553d0c1145ea4d7c13008040f47ea8205d21fe1eaf8f4ab17a1a6981928b35`, and the prefix hash is `34b0ff803aab4ba7764ecea6e0a83e7f1f64e59ea73ec3a007759d223ffa607c`. Each arm received 2,000 ms and 12 classifications per sample. The player-facing generator and catalog were not changed.

| Grade | Arm | Exact-grade hits | Wilson 95% CI | Median ms | p95 ms | Classifications | Unique exact outputs |
|---|---|---:|---:|---:|---:|---:|---:|
| Hard | baseline | 5/10 | 23.7–76.3% | 1606 | 2000 | 29 | 5 |
| Hard | candidate | 10/10 | 72.2–100.0% | 18 | 240 | 33 | 10 |
| Expert | baseline | 0/10 | 0.0–27.8% | 2000 | 2000 | 1 | 0 |
| Expert | candidate | 10/10 | 72.2–100.0% | 58 | 549 | 33 | 10 |
| Evil | baseline | 0/10 | 0.0–27.8% | 2000 | 2000 | 1 | 0 |
| Evil | candidate | 10/10 | 72.2–100.0% | 76.5 | 492 | 35 | 10 |

## Failure Shape

- **Hard baseline:** exact-grade=5, timed-out=5.
- **Hard candidate:** exact-grade=10.
- **Expert baseline:** timed-out=10.
- **Expert candidate:** exact-grade=10.
- **Evil baseline:** timed-out=10.
- **Evil candidate:** exact-grade=10.

## Reproducibility and Decision

- An independent replay reproduced all 60 outcome, puzzle, canonical identity, grade, score, maximum-technique, trace-digest, and classification-count fields exactly; duration was excluded from equality.
- The candidate produced 30/30 unique exact-grade outputs. The baseline produced 5/30 exact-grade outputs: 5/10 Hard and 0/10 Expert or Evil.
- Expert and Evil pass the advancement gate: the candidate Wilson intervals do not overlap the baseline intervals, reproducibility is equal, and no invalid, non-unique, duplicate, wrong-grade, strategy-unsolved, or timed-out candidate outcome occurred.
- Hard improved from 5/10 to 10/10, but its 95% intervals still overlap narrowly. Treat Hard as promising rather than independently conclusive.
- The next change must be a separately reviewed Expert/Evil implementation proposal. This report does not authorize changes to interactive deadlines, catalog acquisition, fallback behavior, solver semantics, or visible grades.
