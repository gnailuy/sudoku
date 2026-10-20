---
domain: Designs
status: Active
entry_points:
  - cmd/tui.go
  - tui/model.go
  - recovery/recovery.go
  - scripts/e2e_tui.py
  - scripts/e2e_tui_hint_strategies.py
  - scripts/e2e_tui_full_game_hints.py
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

### 9.2 Values, Candidates, and History
**Action:** Move with arrows or `h`/`j`/`k`/`l`, focus a placed digit, enter a value, toggle note input with `n`, edit a hidden candidate cell, reveal all candidates with `a`, edit a visible candidate cell, then use `u` and `r`.
**Expected:** Focus stops at board edges. Focusing a placed digit names it in the status, strongly emphasizes equal placed values, and emphasizes only that digit inside visible candidate cells; a digit with all nine valid placements is dimmed. Invalid values and hint scenes remain higher priority. The hidden-cell edit reveals only that cell and replaces its stored set from an empty baseline; the visible-cell edit starts from its complete displayed set. Keys submit complete candidate actions against one grid, peer cleanup follows accepted values, and Undo/Redo restore whole game transitions without changing the presentation mask.

### 9.3 Help, Every-Strategy Hint Navigation, and Apply
**Action:** Start with candidates hidden, press `?`, inspect and close the keyboard-help overlay, press `i`, navigate Back and Next with Left/Right or `h`/`l`, then press Enter; separately cancel a preview with Esc. Run `scripts/e2e_tui_hint_strategies.py` against the built binary to repeat preview, local step navigation, and apply over the maintained complete-game manifest until each of the 23 registered strategies is reached.
**Expected:** A successful preview changes the presentation to `CANDIDATES ON` before Step 1 without changing puzzle state, history, or serialized session, and candidates remain visible after cancel or apply until explicitly hidden. The strategy name, difficulty name, and active narration render from shared canonical English descriptors while targets use compact TUI coordinates and digits remain typed; complete step-message and summary descriptors remain defensive fallbacks when narration or steps are unavailable. Help and local step navigation do not mutate the game. Wrong values and inconsistent candidate cells appear as correction plans before any strategy; a candidate correction points to the exact cell, may mark locally impossible extras, and resets that cell to its complete locally legal set without identifying the solved digit. Enter submits the retained `plan_id` once and confirms “Hint applied.” without exposing the engine action value; Esc discards it without mutation. The strategy matrix still observes and applies all 23 public strategy names and grades after validation passes.

### 9.4 Explicit Save and Safe Quit
**Action:** Change a cell, press `q` and decline, press `S`, enter a path, then quit. Resume with `sudoku tui --resume <path>`.
**Expected:** Dirty quit asks for confirmation. Save atomically creates a mode-0600 session, clears the dirty marker, and resumed values, notes, and history match.

### 9.5 Restore Rejection and CLI Compatibility
**Action:** Run TUI restore against corrupt, unsupported, and oversized files; then run scenario 1.1 from `.aidoc/designs/e2e-play-scenarios.md` and scenarios 8.3 and 8.4 from `.aidoc/designs/e2e-session-scenarios.md`.
**Expected:** Invalid restores fail before entering full-screen mode and preserve source files. The line-oriented CLI output and persistence behavior remain compatible.

### 9.6 Themes and No-Color Accessibility
**Action:** Start the known puzzle with the default environment, `SUDOKU_THEME=light`, and `NO_COLOR=1`.
**Expected:** Dark and light runs use their deterministic palettes. The no-color run emits no color codes while bold givens and matching values, underlined invalid values and matching visible candidate digits, reverse-video focus, faint peers, completed-digit context and globally revealed candidates, visible cell-level notes, clean digits, and heavy box boundaries preserve semantic distinctions.

### 9.7 Candidate Visibility Toggle
**Action:** Start TUI, press `a`, and press `a` again.
**Expected:** The one stored candidate grid appears only while globally visible, board geometry remains stable, the status changes between `CANDIDATES ON` and `CANDIDATES OFF`, and no dirty marker or history entry is created.

### 9.8 Candidate Transition Refresh
**Action:** With candidates enabled, set and clear a value, undo and redo, apply a hint, repair an invalid entry, solve, and reset.
**Expected:** Every displayed candidate set follows the accepted solver-safe board without an independent candidate action or history entry.

### 9.9 Candidate Editing and Accessibility
**Action:** Preserve a valid reduced set, then separately create reduced-plus-impossible, missing-solution-supporting, and empty candidate sets under dark, light, and `NO_COLOR` themes and request a hint.
**Expected:** One grid is rendered in fixed positions. A valid reduced set proceeds to strategy selection. Each inconsistent case identifies the problem cell, may mark locally impossible extras, and replaces the complete set with every locally legal digit without identifying the solved digit or relying on color.

### 9.10 Candidate Save and Resume
**Action:** Edit candidates, save, then resume the session.
**Expected:** Session bytes preserve the authoritative grid but not the visibility preference; the resumed TUI starts with `CANDIDATES OFF` and reveals the saved grid when requested.

### 9.11 Invalid-Entry Handling
**Action:** Enter an invalid value with candidates enabled, then clear or repair it.
**Expected:** The invalid cell shows its visible value, its peers ignore that value during candidate calculation, and candidates reappear in the cell after clear or repair.

### 9.12 Correction Priority
**Action:** Enter a wrong value, request and apply a hint, corrupt one candidate cell, request and apply another hint, then request a third hint.
**Expected:** The first plan clears the exact wrong value, the second replaces the exact inconsistent candidate set, and only the third plan names and teaches a Sudoku strategy. Each application is one Undo step.

### 9.13 Mistake Count
**Action:** Start a new TUI game, enter a confirmed invalid value, then use Undo.
**Expected:** The status starts at `Mistakes: 0`, changes to `Mistakes: 1` after the invalid value, and remains at one after Undo because the authoritative count is cumulative rather than part of history.

### 9.14 Complete Hint-Driven Games and Exact Strategy Union
**Action:** Execute `scripts/e2e_tui_full_game_hints.py` against the built binary. The harness solves the maintained minimum nine-game corpus entirely through TUI preview and apply controls.
**Expected:** Every game reaches the solved state, exits cleanly, and the combined trace equals the exact 23 public strategy names. This preserves complete-game acceptance after the compact line CLI stops at candidate deductions.

---

## 10. TUI Autosave and Recovery

Use an isolated state root: export `XDG_STATE_HOME=$SUDOKU_E2E_DIR/state` on Unix, or set `LOCALAPPDATA` to an isolated directory on Windows.

### 10.1 Crash Recovery and Durable Record Discovery
**Action:** Start `sudoku tui --input <known puzzle>`, make a gameplay change, wait past the one-second debounce, and terminate the process abnormally. Then run plain `sudoku tui`.
**Expected:** One platform-private record exists under a platform-private recovery directory. Unix uses modes `0600` and `0700`; Windows uses protected current-user/System/Administrators DACLs. Plain startup offers the changed game; selecting it restores the value without relying on the old process ID.

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
