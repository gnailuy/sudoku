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

The import scenario catalog verifies file parsing, normalization, source labels, error handling, database storage, and deduplication through the built import command.

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
**Expected:** Shows flags: `-f`, `--source`, `--db`.

### 5.2 Import Puzzles from File
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Puzzles are classified, normalized, and stored. The two lines represent the same puzzle (different notation), so one is stored and one is a duplicate.

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

---
