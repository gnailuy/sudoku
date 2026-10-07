---
domain: Designs
status: Active
entry_points:
  - cmd/import.go
  - db/db.go
dependencies:
  - .aidoc/designs/e2e-test-scenarios.md
  - .aidoc/designs/game-engine.md
---

# E2E Import Scenarios

The import scenario catalog verifies plain and hash-pinned external parsing, full Sudoku-symmetry canonicalization, source provenance, strategy reclassification, error handling, database storage, and deduplication through the built import command.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/e2e-test-scenarios.md` | E2E discovery map, isolation rules, and automation entry points |
| `AGENT.md` | Required black-box verification discipline |

## Why This Boundary

Import accepts mixed external text and writes persistent records. Black-box coverage protects user-visible partial-success behavior and normalization at the complete command boundary.

## 5. Import CLI

### 5.1 Import Help
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Shows file, source, format, SHA-256 pin, immutable analysis-manifest, worker, and database flags.

### 5.2 Import Puzzles from File
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Puzzles are classified, symmetry- and digit-canonicalized, and stored. The two lines are transposed and digit-relabeled presentations of the same puzzle, so one is stored and one is a duplicate.

### 5.3 Import with Invalid Lines
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Valid puzzle stored. Invalid lines (too short, wrong characters) are skipped with stderr messages. Valid lines are processed.

### 5.4 Import with Source Label
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Custom source label ("top1465") is stored with each puzzle.

### 5.5 Import Missing File
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Error: "no such file". Exit code 1.

### 5.6 Import Empty / Comments-Only File
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Report shows 0 total lines, 0 stored. No errors.

### 5.7 Import Dedup
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Second import reports all as duplicates.

### 5.8 Pinned Sudoku Exchange Record and Analysis Manifest
**Action:** Import a three-field Sudoku Exchange fixture with its exact complete-file SHA-256, an explicit commit-bound source label, and a new analysis-manifest path; then retry without the hash pin and with the existing manifest path.
**Expected:** The pinned import stores each playable canonical puzzle with the published 12-byte hash as provenance. The version 2 analysis manifest preserves every structurally valid canonical puzzle, source order, finite Sukaku Explainer rating, complete-file SHA-256, and published record hash, including records excluded from play as strategy-unsolved. An unpinned import and an attempted manifest overwrite both fail before opening or mutating the database.

### 5.9 Strategy-Unsolved Exclusion
**Action:** Import a structurally valid pinned-source puzzle beyond the canonical strategy inventory.
**Expected:** The report counts the record as strategy-unsolved, stores no playable row, and does not mislabel it Evil.

---
