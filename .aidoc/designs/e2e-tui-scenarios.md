---
domain: Designs
status: Active
entry_points:
  - cmd/tui.go
  - tui/model.go
  - recovery/recovery.go
dependencies:
  - .aidoc/designs/e2e-test-scenarios.md
  - .aidoc/designs/tui-frontend.md
  - .aidoc/designs/background-autosave.md
---

# E2E TUI Scenarios

The TUI scenario catalog verifies terminal rendering, keyboard interaction, accessibility, candidates, persistence, autosave, and crash recovery through a real pseudo-terminal.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/e2e-test-scenarios.md` | E2E discovery map, isolation rules, and automation entry points |
| `AGENT.md` | Required black-box verification discipline |

## Why This Boundary

The full-screen frontend depends on terminal events and asynchronous recovery writes that unit tests cannot represent completely. The pseudo-terminal harness protects the composed user experience.

## 9. Full-Screen TUI

The TUI scenarios require a pseudo-terminal. The standard-library harness exercises the built binary rather than package internals:

**Action:** Execute the matching case in `scripts/e2e_tui.py`, which owns the canonical command sequence and fixture.

### 9.1 Startup and Resize
**Action:** Start `sudoku tui --input <known puzzle>` in a terminal of at least 72×40, then resize below the minimum and back.
**Expected:** The centered board uses clean digits, fixed-position notes, strong 3×3 boundaries, a strong focus background, and subtle peer backgrounds. A small terminal displays only the minimum-size instruction; resizing restores the unchanged game.

### 9.2 Values, Notes, and History
**Action:** Move with arrows or `h`/`j`/`k`/`l`, enter a value, toggle note mode with `n`, enter a note, then use `u` and `r`.
**Expected:** Focus stops at board edges. Keys submit engine actions, note cleanup follows peer rules, and undo/redo restore whole transitions.

### 9.3 Help, Hint Preview, and Apply
**Action:** Press `?`, inspect and close the keyboard-help overlay, press `i`, inspect the technique/reason, then press Enter.
**Expected:** Help does not mutate the board. Hint preview does not mutate the board; Enter applies that hint through `game.ApplyHint` and marks the session dirty.

### 9.4 Explicit Save and Safe Quit
**Action:** Change a cell, press `q` and decline, press `S`, enter a path, then quit. Resume with `sudoku tui --resume <path>`.
**Expected:** Dirty quit asks for confirmation. Save atomically creates a mode-0600 session, clears the dirty marker, and resumed values, notes, and history match.

### 9.5 Restore Rejection and CLI Compatibility
**Action:** Run TUI restore against corrupt, unsupported, and oversized files; then run scenario 1.1 from `.aidoc/designs/e2e-play-scenarios.md` and scenarios 8.3 and 8.4 from `.aidoc/designs/e2e-session-scenarios.md`.
**Expected:** Invalid restores fail before entering full-screen mode and preserve source files. The line-oriented CLI output and persistence behavior remain compatible.

### 9.6 Themes and No-Color Accessibility
**Action:** Start the known puzzle with the default environment, `SUDOKU_THEME=light`, and `NO_COLOR=1`.
**Expected:** Dark and light runs use their deterministic palettes. The no-color run emits no color codes while bold givens, underlined invalid values, reverse-video focus, faint peers/automatic candidates, bold manual notes, clean digits, and heavy box boundaries preserve semantic distinctions.

### 9.7 Automatic-Candidate Toggle
**Action:** Start TUI, press `a`, and press `a` again.
**Expected:** Legal candidates appear only while enabled, board geometry remains stable, the status changes between `AUTO ON` and `AUTO OFF`, and no dirty marker is created.

### 9.8 Candidate Transition Refresh
**Action:** With candidates enabled, set and clear a value, undo and redo, apply a hint, repair an invalid entry, solve, and reset.
**Expected:** Every displayed candidate set follows the accepted solver-safe board without an independent candidate action or history entry.

### 9.9 Automatic and Manual Coexistence
**Action:** Add legal and stale manual notes with candidates enabled under dark, light, and `NO_COLOR` themes.
**Expected:** Manual styling wins at overlapping positions, stale manual notes remain visible, and automatic candidates remain distinguishable without color alone.

### 9.10 Candidate Save and Resume
**Action:** Save with candidates enabled, then resume the session.
**Expected:** Session bytes contain no candidate preference or derived data, and the resumed TUI starts with `AUTO OFF`.

### 9.11 Invalid-Entry Handling
**Action:** Enter an invalid value with candidates enabled, then clear or repair it.
**Expected:** The invalid cell shows its visible value, its peers ignore that value during candidate calculation, and candidates reappear in the cell after clear or repair.

### 9.12 Candidate Adoption
**Action:** Enable automatic candidates and note mode, then enter a digit in an editable empty cell and use Undo followed by Redo.
**Expected:** The first edit copies the complete candidate grid, applies the initiating digit toggle, switches the display to `AUTO OFF`, and remains in note mode without confirmation. One Undo restores the complete prior note map, and one Redo reapplies the replacement.

### 9.13 Mistake Count
**Action:** Start a new TUI game, enter a confirmed invalid value, then use Undo.
**Expected:** The status starts at `Mistakes: 0`, changes to `Mistakes: 1` after the invalid value, and remains at one after Undo because the authoritative count is cumulative rather than part of history.

---

## 10. TUI Autosave and Recovery

Use an isolated state root with `export XDG_STATE_HOME=$SUDOKU_E2E_DIR/state`.

### 10.1 Crash Recovery and Durable Record Discovery
**Action:** Start `sudoku tui --input <known puzzle>`, make a gameplay change, wait past the one-second debounce, and terminate the process abnormally. Then run plain `sudoku tui`.
**Expected:** One mode-`0600` record exists under a mode-`0700` recovery directory. Plain startup offers the changed game; selecting it restores the value without relying on the old process ID.

### 10.2 Concurrent Recovery Selection
**Action:** Interrupt two changed TUI instances, then run plain `sudoku tui` and move through the recovery list.
**Expected:** Each instance has a different random record. The list is newest-first, either game can be selected, and `d` discards only the selected record.

### 10.3 Explicit Sources and Opt-Out
**Action:** Leave a valid recovery record, then start with each of `--input`, explicit `--level`, `--resume`, and `--no-autosave`.
**Expected:** Every intentional source bypasses recovery selection. `--no-autosave` also creates no recovery record after mutations.

### 10.4 Recovery Cleanup
**Action:** Recover a game and explicitly save it; separately confirm dirty quit/discard and cleanly quit an unmodified recovered game.
**Expected:** Each successful cleanup path removes the active recovery record. `Ctrl-C` and abnormal termination retain the latest successful record.

### 10.5 Recovery Failure Is Non-Fatal
**Action:** Use an inaccessible or unsafe state path and mutate the game.
**Expected:** Gameplay continues with a persistent concise autosave warning. A later mutation retries recovery after the path is fixed.
