---
domain: Architecture
status: Active
entry_points:
  - solver/solver.go
dependencies:
  - .aidoc/INDEX.md
  - .aidoc/designs/difficulty-model.md
---

# Architecture Guidelines

The Sudoku project follows a layered architecture where each package owns a single concern.
The architecture guide captures the design constraints, layer boundaries, and solver contract that code alone doesn't express.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/INDEX.md` | Discovery index with reading chains |
| `.aidoc/designs/difficulty-model.md` | Difficulty model design and target state |
| `.aidoc/designs/game-engine.md` | Canonical game-engine boundary and state invariants |
| `.aidoc/designs/cli-sessions.md` | Current CLI note and persistence transport boundaries |
| `AGENT.md` | Operational rules for AI agents working on this repo |

## Why This Structure

The package layout ensures strategy solvers can be added independently without touching the generator, game, or CLI.
Each solver is self-contained: implement the interface, register in the store, and add the key to difficulty definitions.
The generator already has plumbing for strategy-based validation — it calls `StrategySolver.Apply()` during cell removal — so new solvers plug in without generator changes.

## Layer Boundaries

Requests flow from `cmd` through frontend controllers into the pure game and generator boundaries. Persistent paths enter `db`, solving paths enter `solver`, and all domain paths terminate in `core`; package imports must follow the constraints below rather than bypassing an owner.

**Dependency rules:**
- `core` has zero dependencies on other packages (leaf layer).
- `solver` depends only on `core` and `util`.
- `db` depends on `core` and `solver` (for normalization and classification). Does **not** depend on `generator`, `game`, or `cli`.
- `generator` depends on `core`, `solver`, and `util`.
- `game` depends on `core` and `solver`. Does **not** depend on `cli` or `generator`. Contains pure game logic — no I/O imports (`fmt` for string formatting only; no `os`, `bufio`, or terminal I/O). `game.Game.Apply` is the stable mutation boundary; `game.Game.Snapshot` is the detached rendering boundary.
- `cli` depends on `game` (for the controller). Owns interactive terminal I/O: board display, input handling, signal handling.
- `cmd` depends on `cli`, `core`, `db`, `game`, `generator`, and `solver`. Owns all CLI command definitions (cobra), flag parsing, fallback flow, batch generation, and import logic.
- `main.go` delegates to `cmd.Execute()` — minimal entry point.
- `util` has no internal dependencies (pure helpers).

Violations of these boundaries indicate a design problem.

## Game Engine Boundary

`game.Game` owns mutable session state and never exposes its internal boards. `game.Game.ProblemBoard`, `game.Game.PlayBoard`, and `game.Game.Snapshot` return detached values so frontend rendering cannot mutate the session accidentally.

Player transitions enter through typed actions in `game/contract.go`. `game.Game.Apply` returns structured value and candidate changes, applied-hint metadata, and typed `game.EngineError` values; rejected actions leave the complete visible state unchanged. `cli.Controller` uses this boundary for set, clear, reset, repair, solve, hint, undo, and redo operations rather than calling command-shaped helpers.

The game owns one authoritative candidate grid. Value actions clear candidates on the changed cell and remove an accepted value from peer candidates in the same transition. The unified history restores boards, invalid markers, and candidate sets together, which prevents frontends from reconstructing cleanup during Undo or Redo.

`game.Game.Hint` is a query even when a strategy solver eliminates candidates internally. Hint search operates on a detached board constrained by the authoritative grid. `game.ApplyHint` records the exact value, complete candidate replacement, or candidate eliminations from the previewed plan as one history transition, so later hints consume the same state every client observes.

## Solver Interface Contract

`solver.Solver` defines stable metadata, while `solver.StrategySolver` represents one human-style technique and `solver.CompleteSolver` represents a full solving boundary. Strategy solvers may report no progress without implying that a puzzle is invalid; complete solvers own full-solve, hint, and solution-query behavior. Production callers use bounded existence or uniqueness queries and stop after one or two solutions; exhaustive counting remains available only when the exact total is genuinely required. See `solver/solver.go` for the interfaces and `solver/move.go` for the detached move result.

`solver.Store` is the registration and lookup boundary for both solver classes. New techniques register through `solver.Store.RegisterStrategy`, join the appropriate strategy grade in `solver/config.go`, and keep unit traversal within the shared helpers in `solver/units.go`. The implementation and package tests are the canonical source for method signatures and registration mechanics.

## Design Constraints

- **Interface naming:** Types follow Go conventions. Examples: `Solver` (base interface), `StrategySolver`, `CompleteSolver`, `Base`, `Backtracker`, `Store`, `Move`, `Board`, `Game`, `Difficulty`, `Options`, `CandidateSet`.
- **Candidate boundaries:** `Board.Candidates(pos)` computes solver-local candidates from accepted values and the board's elimination layer. `game.Game` stores the player-visible authoritative `CandidateSet` grid, constrains a detached solver board from it for hints, and persists it with history. `core.Board.EliminateCandidate` remains the solver-board mutation primitive; frontends mutate game candidates only through typed engine actions.
- **Error vs panic:** Methods called with invalid state from within the system `panic` (bug detection). Methods processing user input return errors. This split is intentional.
- **Geometric distribution stop:** The generator uses `util.RandomBool(0.125)` to probabilistically stop cell removal after reaching the target clue range. This produces natural variation within a difficulty band.
