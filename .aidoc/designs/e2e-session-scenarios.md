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

The CLI-session catalog verifies bounded restore, atomic explicit save, redo preservation, and failure safety through the built line-oriented CLI.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/e2e-test-scenarios.md` | E2E discovery map, isolation rules, and automation entry points |
| `.aidoc/designs/cli-sessions.md` | Compact rendering and complete-session transport contract |
| `AGENT.md` | Required black-box verification discipline |

## Why This Boundary

Durable sessions compose engine serialization with frontend parsing and filesystem transport. Black-box scenarios ensure rejected files and failed saves cannot corrupt user-owned data, while package tests prove hidden candidate state survives the engine-owned round trip.

## 8. Durable Sessions

Use the known puzzle from section 2 of `.aidoc/designs/e2e-play-scenarios.md` and paths under `$SUDOKU_E2E_DIR`.

### 8.1 Save, Resume, and Preserve Redo
**Action:** Execute the matching case in `scripts/e2e_cli.py`, which owns the canonical command sequence and fixture.
**Expected:** Save produces a platform-private session after a value change and Undo: mode `0600` on Unix or a protected current-user/System/Administrators DACL on Windows. The restored game reports the pre-redo board, `redo` reapplies the value, and `check` reports the resulting invalid state.

### 8.2 Reject Corrupt, Obsolete-Shape, and Oversized Sessions
**Action:** Execute the matching cases in `scripts/e2e_cli.py`.
**Expected:** The saved session has no obsolete version discriminator. Every rejected restore exits non-zero before interactive play with a concise error, and source files remain unchanged.

### 8.3 Reject Resume Flag Conflicts
**Action:** Combine `--resume` with `--input` and with `--level` through `scripts/e2e_cli.py`.
**Expected:** Cobra rejects both combinations before reading or generating a puzzle.

### 8.4 Preserve a Failed Save Destination
**Action:** Save to an existing directory through `scripts/e2e_cli.py`.
**Expected:** Save reports an error, the directory remains intact, and no `.sudoku-session-*` temporary file remains.
