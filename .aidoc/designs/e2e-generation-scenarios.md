---
domain: Designs
status: Active
entry_points:
  - cmd/generate.go
  - cmd/experiment.go
  - generator/generator.go
  - generationexperiment/runner.go
dependencies:
  - .aidoc/designs/e2e-test-scenarios.md
  - .aidoc/designs/game-engine.md
---

# E2E Generation Scenarios

The generation scenario catalog verifies generation flags, worker composition, hard deadlines, actual-grade reporting, database storage, validation, deduplication, and isolated resumable experiment execution through the built command.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/e2e-test-scenarios.md` | E2E discovery map, isolation rules, and automation entry points |
| `AGENT.md` | Required black-box verification discipline |

## Why This Boundary

Generation combines probabilistic puzzle construction with deterministic command and storage contracts. Assertions therefore protect composition and bounded outcomes rather than requiring a random run to hit an exact grade. Per-puzzle timeouts are hard caller deadlines; only completed puzzles are classified and stored, always under their actual strategy grade.

## 4. Puzzle Generation CLI

### 4.1 Generate Help
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Shows all flags: `-n`, `-d`, `-w`, `-t`, `--rounds`, `--db`.

### 4.2 Generate Easy Puzzles
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Generates 5 puzzles. Reports: generated count, stored count, duplicates, by-level breakdown. Some may be classified at a different difficulty than requested (mismatch is expected for easy/medium targets).

### 4.3 Generate Hard Puzzles
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Attempts 3 puzzles. Every completed puzzle is classified and stored under its actual grade. The report separates completed generation, target matches, and hard-deadline timeouts.

### 4.4 Generate with Parallel Workers
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Uses up to 4 parallel workers. Every completed puzzle is stored under its actual grade; deadline expirations are reported separately.

### 4.5 Generate with Invalid Difficulty
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Error: "Invalid difficulty level". Exit code 1.

### 4.6 Generate with Count 0
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Error: "count must be positive". Exit code 1.

### 4.7 Generate with Custom DB Path
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** DB file created at `/tmp/custom-test.db`. Puzzles stored there.

### 4.8 Generate Dedup
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Second run may report duplicates if the same normalized puzzles are generated.

### 4.9 Generate with Hard Deadline
**Action:** Execute the bounded generation case in `scripts/e2e_cli.py` with a one-millisecond per-puzzle timeout and an isolated database.
**Expected:** The command returns within a bounded margin of the configured deadline, reports the timeout explicitly, and does not store an incomplete or unclassified puzzle.

### 4.10 Run and Resume an Exact-Grade Experiment
**Action:** Execute the fixed one-sample manifest in `scripts/e2e_cli.py` through `sudoku experiment generation`, then execute the same command again against the same output directory.
**Expected:** The first run records one baseline and one candidate observation under equal budgets. The second run appends zero observations, preserves the complete two-arm report, and never creates or opens a puzzle database.

---
