---
domain: Designs
status: Active
entry_points:
  - main.go
  - cmd/root.go
  - cli/controller.go
dependencies:
  - .aidoc/designs/e2e-test-scenarios.md
  - .aidoc/designs/game-engine.md
---

# E2E Root Play Scenarios

The root-play scenario catalog protects root startup, puzzle input, game commands, aliases, and redo semantics through the built line-oriented CLI.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/e2e-test-scenarios.md` | E2E discovery map, isolation rules, and automation entry points |
| `AGENT.md` | Required black-box verification discipline |

## Why This Boundary

Root play is the oldest compatibility surface and composes Cobra parsing, database selection, the controller, and the game engine. Testing the binary catches regressions that package tests cannot see.

## 1. Interactive Play

### 1.1 Input a Known Puzzle
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Board is displayed. Game starts. Exits on "quit".

### 1.2 Input a Puzzle Using Dots Notation
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Dots (`.`) are treated as empty cells. Board displays correctly.

### 1.3 Input a Puzzle Using Zeros Notation
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Zeros are converted to empty cells. Same puzzle as 1.2.

### 1.4 Generate a Puzzle by Difficulty Level
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** A puzzle is generated (may show mismatch warning if best-effort misses). Game starts.

### 1.5 Invalid Input (Too Short)
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Error message about invalid puzzle string. Non-zero exit or error output.

### 1.6 Invalid Level Flag
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Usage error shown with valid difficulty levels listed.

### 1.7 Help Flag
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Usage printed with subcommands (`calibrate`, `generate`, `import`, `tui`) and flags listed.

### 1.8 Backward Compatibility: `--input` Flag
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Works the same as before the cobra migration.

### 1.9 Backward Compatibility: `--level` Flag
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Works the same as before the cobra migration.

## 2. Game Commands

The game-command scenarios verify the stable engine boundary through the real terminal frontend: `cli.Controller` renders detached snapshots and submits typed actions while preserving established behavior. Scenarios start from the known puzzle fixture owned by `scripts/e2e_cli.py`.

### 2.1 Add a Value (`add` / `a` / bare digits)
**Input:** `add 1 1 4` or `a 1 1 4` or `1 1 4`
**Expected:** Cell (1,1) is set to 4. Board updates.

### 2.2 Clear a Value (`clear` / `d`)
**Input:** After adding a value at (1,1): `clear 1 1` or `d 1 1`
**Expected:** Cell (1,1) is reset to empty.

### 2.3 Undo (`undo` / `u`)
**Input:** After adding a value: `undo` or `u`
**Expected:** Last move is reversed. Cell returns to previous state.

### 2.4 Redo (`redo` / `r`)
**Input:** After undo: `redo` or `r`
**Expected:** Undone move is re-applied, including whether the value is valid or invalid.

### 2.5 Check Board (`check` / `c`)
**Input:** `check` or `c`
**Expected (correct board):** "The current board is correct."
**Expected (incorrect board):** "You have entered incorrect value(s)."

### 2.6 Repair Board (`repair` / `f`)
**Input:** After entering wrong values: `repair` or `f`
**Expected:** All incorrect inputs are removed. Board returns to a valid state.

### 2.7 Reset Board (`reset` / `e`)
**Input:** After making several moves: `reset` or `e`
**Expected:** Board returns to the original problem state. All user inputs cleared.

### 2.8 Hint (`hint` / `i`)
**Input:** `hint` or `i`
**Expected:** A correct value is filled into a cell. The technique and reason are displayed.

### 2.9 Solve (`solve` / `s`)
**Input:** `solve` or `s`
**Expected:** The complete solution is displayed. "Congratulations" message shown.

### 2.10 Quit (`quit` / `q`)
**Input:** `quit` or `q`
**Expected:** Game exits cleanly.

### 2.11 Shorthand Digit Input
**Input:** `1 2 3` (no `add` prefix)
**Expected:** Treated as `add 1 2 3`. Cell (1,2) is set to 3.

### 2.12 Divergent Input Truncates Redo History
**Input:** Add a value, undo it, add a different value, then run `redo`.
**Expected:** The different value remains in the cell and the abandoned value does not reappear because the new action replaced the old redo branch.

---
