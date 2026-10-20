---
domain: Designs
status: Active
entry_points:
  - cmd/tui.go
  - tui/model.go
dependencies:
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/cli-sessions.md
  - .aidoc/designs/localization-contract.md
  - .aidoc/designs/e2e-tui-scenarios.md
---

# TUI Frontend

The optional full-screen terminal interface is the smallest second frontend for the stable game engine. The TUI improves keyboard play and proves that engine state, history, hints, notes, and serialization can support multiple presentations without changing the existing line-oriented CLI.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/game-engine.md` | Canonical actions, snapshots, hints, notes, and serialization |
| `.aidoc/designs/cli-sessions.md` | Existing CLI behavior and explicit persistence policy |
| `.aidoc/designs/localization-contract.md` | Shared canonical English rendering and descriptor ownership |
| `.aidoc/designs/e2e-tui-scenarios.md` | Black-box compatibility and TUI scenarios |

## Why the TUI Exists

The TUI reuses the Go engine directly without adding a network, browser, service, or cross-language boundary. The terminal frontend validates the presentation contract; the browser client is a separate repository, and this repository contains no mobile client.

The existing CLI remains valuable for scripts, redirected input, and minimal terminals. The `sudoku tui` command complements rather than replaces the root play command, so established commands and output remain compatible.

## Interaction Model

The TUI presents one focused cell, the board, game status, the authoritative cumulative Mistakes count, available history, a concise key guide, and a message area. When focus rests on a placed digit, the status names that active digit, every equal placed value receives a strong match treatment, and that digit is emphasized inside visible candidate cells without coloring the whole cell. A digit with all nine valid placements uses a quieter completed treatment. Arrow keys and `h`/`j`/`k`/`l` move focus within the board without wrapping. Digit keys set values in value mode and toggle the authoritative candidate set in candidate mode; `0`, Backspace, or Delete clears the focused value or candidate set according to the active mode.

The following global actions remain available when no modal is open:

- `n` toggles value and candidate input without changing candidate visibility; candidate input edits the one authoritative set;
- `a` shows every editable candidate cell or hides every candidate cell without mutating or dirtying the session;
- the first candidate digit in a hidden cell reveals only that cell and replaces its stored set from an empty visible baseline, while candidate input in a visible cell edits the complete displayed set;
- `u` and `r` submit undo and redo;
- `i` requests a hint preview and reveals candidates before its first step; Left/Right or `h`/`l` navigate the retained steps locally, Enter applies the exact plan, and Esc cancels it while leaving candidates visible;
- `?` opens a compact keyboard-help overlay;
- `c` checks the current status without mutating the game;
- `R` resets after confirmation;
- `S` opens an explicit save-path prompt;
- `q` opens a quit confirmation when the session has unsaved changes and otherwise exits.

The TUI translates keys into `game.Action` values and renders only `game.Snapshot`. Shared hint titles, strategy and difficulty names, summaries, steps, and narration use descriptor fallbacks from the canonical English catalog; TUI-only controls, status framing, persistence failures, and engine-error presentation remain at the terminal boundary. Status and action feedback use direct player-facing labels rather than exposing engine values such as `in-progress`, `set-value`, or `apply-hint`. The TUI never reproduces Sudoku validation rules.

## Startup and Persistence

`sudoku tui` accepts the same puzzle sources as root play: `--input`, `--level`, and `--resume`, with the same mutual exclusions and default difficulty. Shared session construction lives behind a command-level helper so the CLI and TUI do not duplicate generation, classification, solver options, or restore validation.

Player-owned persistence remains explicit: a resumed session remembers its source path for the running instance, while `S` confirms the destination for an intentional save. Independently, successful TUI gameplay mutations create one-second debounced private recovery records unless `--no-autosave` is set. Plain startup discovers recent records by random durable identifier rather than process ID; explicit puzzle sources bypass discovery. `.aidoc/designs/background-autosave.md` owns the recovery lifecycle and cleanup policy.

The filesystem transport lives in the presentation-neutral `sessionfile` package so both frontends consume the same guarantees. The transport retains the current bounded-read, user-only permission, atomic-replacement, and source-preservation guarantees.

## Rendering and Terminal Boundaries

The TUI supports terminals large enough to show a 9×9 board, status, messages, and help. A too-small terminal renders a resize instruction and accepts only resize and quit events until the minimum layout fits.

Given cells, player values, invalid entries, the focused cell, peer cells, and focused-digit context remain semantically distinct without punctuation around digits. Focus uses a strong background, peers use a quieter background, matching placed values use a strong foreground, matching visible candidate digits add underline emphasis, and 3×3 boundaries are heavier than cell boundaries. Invalid-value and hint-scene treatments take priority over persistent digit context. Visible candidates use fixed positions without placeholder dots; hidden candidates leave no digit or matching mark. A successful hint preview reveals every candidate cell as presentation state and leaves them visible after close or apply. Bold focus, underlined premises, struck eliminations, and reversed conclusions preserve hint meaning without color. The title, status, board, messages, and one-line key guide center within the available width.

Dark and light palettes are deterministic and selected with `SUDOKU_THEME`; `NO_COLOR` or the `no-color` theme retains bold matching values, underlined matching candidates, reverse-video focus, and faint completed/peer context while removing color distinctions. The renderer must produce deterministic output from model state so package tests can verify layouts without a live terminal.

The TUI uses a single event loop to serialize engine actions. Background puzzle generation and filesystem operations may return messages to the event loop, but no goroutine mutates `game.Game` directly.

## Dependency Decision

The implementation uses Bubble Tea v1.3.4 for terminal lifecycle, keyboard events, resize events, and deterministic model updates. Lip Gloss v1.0.0 provides styling and width-aware centering, while semantic state, palettes, layout, and accessibility policy remain in project-owned TUI code. Bubbles is not required.

The terminal dependencies are confined to the `tui` package and `cmd/tui.go`. The `game`, `core`, `solver`, and persistence packages remain terminal-library independent.

## Verification

Package tests cover key-to-action translation, canonical descriptor and typed-narration rendering, focus boundaries, focused-digit value/candidate context, completed-digit treatment, independent note mode and cell visibility, hidden-cell empty-baseline edits, visible-cell complete-set edits, authoritative mistake-count rendering, modal confirmations and help, dirty-state tracking, save transport, read-only hint step navigation/cancel/apply, small-terminal fallback, clean cell rendering, theme selection, no-color accessibility, and deterministic rendering. The model injects its persistence function for isolated save tests.

`scripts/e2e_tui.py` is a standard-library pseudo-terminal harness that starts the built binary, sends keys, resizes the terminal, and inspects stable screen text. Black-box scenarios cover startup from input and saved state, value and note entry, undo/redo, hint step navigation and exact apply, explicit save, invalid restore rejection, quit confirmation, and CLI backward compatibility.
