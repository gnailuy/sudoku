---
domain: Designs
status: Active
entry_points:
  - cmd/play.go
  - cli/controller.go
  - sessionfile/session_file.go
dependencies:
  - .aidoc/designs/e2e-test-scenarios.md
  - .aidoc/designs/cli-sessions.md
---

# E2E CLI Session Scenarios

The CLI-session scenario catalog verifies manual notes, unified history, bounded restore, atomic explicit save, and resume compatibility through the built line-oriented CLI.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/e2e-test-scenarios.md` | E2E discovery map, isolation rules, and automation entry points |
| `AGENT.md` | Required black-box verification discipline |

## Why This Boundary

Durable sessions compose engine serialization with frontend parsing, rendering, and filesystem transport. Black-box scenarios ensure rejected files and failed saves cannot corrupt user-owned data.

## 8. Manual Notes and Durable Sessions

Use the known puzzle from section 2 of `.aidoc/designs/e2e-play-scenarios.md` and paths under `$SUDOKU_E2E_DIR`.

### 8.1 Toggle and Clear Manual Notes
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Notes 1 and 9 appear in fixed candidate positions, then `x` returns the board to compact rendering.

### 8.2 Peer Cleanup and Unified History
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Setting (1,1) removes notes from the target and its row, column, and box peers but preserves the note at non-peer (4,5). Undo restores the value and all removed notes atomically; redo reapplies cleanup.

### 8.3 Save, Resume, and Preserve Redo
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** The first restored board includes note 5 at (1,1), `redo` restores value 4, and the session file has mode `0600`. The second restore retains the invalid entry and note, and `check` reports the invalid board.

### 8.4 Reject Corrupt, Obsolete-Shape, and Oversized Sessions
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** The saved session has no version discriminator. Every corrupt, obsolete-shape, or oversized restore exits non-zero before interactive play with a concise error, and source files remain unchanged.

### 8.5 Resume Flag Conflicts
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Cobra rejects both combinations before reading or generating a puzzle.

### 8.6 Failed Save Preserves the Destination
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Save reports an error, the existing destination remains a directory, and no `.sudoku-session-*` temporary file remains.

---
