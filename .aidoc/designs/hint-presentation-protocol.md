---
domain: Designs
status: Active
entry_points:
  - solver/move.go
  - game/contract.go
  - api/openapi.yaml
dependencies:
  - .aidoc/architecture/guidelines.md
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/web-api.md
  - .aidoc/designs/hint-reference-plans.md
---

# Hint Presentation Protocol

The hint presentation protocol turns one solver recommendation into a portable teaching plan that CLI, TUI, web, and future clients can render without reinterpreting Sudoku logic. The engine owns semantic evidence, ordered explanation, and the logical conclusion; each client owns visual style, interaction, and capability-aware rendering.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/architecture/guidelines.md` | Solver and package dependency boundaries |
| `.aidoc/designs/game-engine.md` | Authoritative hint query and action semantics |
| `.aidoc/designs/web-api.md` | Revisioned transport and client-neutral contract |
| `.aidoc/designs/hint-reference-plans.md` | Concrete placement and elimination teaching plans |
| `solver/move.go` | Current recommendation boundary to be replaced by typed evidence |
| `game/contract.go` | Engine hint preview and application boundary |

## Why a Shared Teaching Contract Exists

A position, value, technique name, and prose reason are enough to apply a hint but not enough to teach it consistently. A browser otherwise invents highlights, a TUI invents symbols, and a CLI invents ordering independently, which lets presentation layers disagree about the same deduction.

The protocol standardizes meaning rather than styling. Strategy and engine code describe what facts matter, what changed, and why the conclusion follows; renderers decide whether semantic roles become color, borders, motion, symbols, speech, or plain text.

The project is still in development, so the protocol replaces the current hint shape instead of preserving it through compatibility adapters. `protocol_version` protects future semantic evolution; it does not require support for the superseded position/value/reason contract.

## Ownership Boundaries

A strategy produces typed evidence for one deduction: examined units, candidate constraints, eliminations, placements, and the technique identity. Placement and candidate elimination are equally valid conclusions; elimination strategies are not duplicated as hint-only variants. Strategy evidence contains no colors, animation timing, widget commands, prose parsing requirements, or transport models.

A shared engine composer converts typed evidence into one `HintPlan`. The composer owns default teaching order and wording so 23 strategies do not duplicate client choreography. A strategy-specific composer is justified only when the technique requires a genuinely different teaching sequence, and it still emits the same protocol.

A renderer consumes a complete plan and never infers Sudoku reasoning from technique names or fallback prose. Renderers may omit unsupported reinforcement, but must preserve the ordered text and logical conclusion.

## HintPlan Contract

Every plan contains:

| Field | Meaning |
|-------|---------|
| `protocol_version` | Semantic schema version, initially `1` |
| `plan_id` | Deterministic identity bound to the source game state and conclusion |
| `strategy` | Stable strategy ID, display name, and strategy grade |
| `summary` | Complete plain-text fallback for the deduction |
| `steps` | Ordered, independently renderable teaching steps |
| `conclusion` | Machine-readable placement or candidate eliminations |

Every step contains a stable ID, a semantic kind, a complete fallback message, a declarative scene, and an optional logical effect. Initial step kinds are `observe`, `compare`, `eliminate`, and `conclude`; adding a kind requires a protocol-version decision when an older renderer could change meaning.

Every scene is a complete snapshot for that step rather than a delta from the previous scene. A client can therefore move forward or backward, render one step, skip unsupported visual reinforcement, or recover after interruption without retaining hidden highlight state.

## Semantic Scene Vocabulary

Scene targets are typed references to a cell, one candidate within a cell, or a row, column, or box. Rows and columns use the engine's internal zero-based coordinates and the HTTP adapter converts them to the transport boundary's one-based coordinates.

Scene marks pair one target with one semantic role:

| Role | Meaning |
|------|---------|
| `focus` | Region or item currently under examination |
| `premise` | Fact already established and used by the deduction |
| `eliminated` | Candidate proven impossible by this deduction |
| `conclusion` | Placement or elimination that completes the teaching step |

Semantic roles carry no palette or animation requirement. Color cannot be the sole carrier of meaning, motion is optional, and the fallback message names every fact needed to understand the step.

Logical effects are separate from marks. A mark explains presentation emphasis; an effect declares the verifiable placement or candidate eliminations produced by the step. The plan conclusion equals the combined final effect and must be valid against the source game state.

## Deduction Selection

Interactive hint orchestration returns the first meaningful deduction from the configured strategy order, whether the deduction places a value or only eliminates candidates. The hint path never consumes elimination-only progress on a detached board merely to search for a later placement; doing so would hide the technique the player needs to learn.

Full solving and difficulty classification may continue chaining eliminations on their private solving state until a placement or completed solve. The shared strategy implementation and evidence therefore serve teaching, solving, and classification even though those callers stop at different boundaries.

Direct note hygiene is not a strategy conclusion. Removing a note that a placed value already makes illegal is deterministic validation owned by normal value and note handling; a teaching hint is reserved for a logical deduction that is not visible from row, column, or box legality alone.

## Query, Navigation, and Application

One read-only hint query returns the complete `HintPlan`. The client fetches that plan once; Next and Back navigate its ordered steps locally without another request or any puzzle, dirty-state, revision, or history change.

Applying the conclusion is a separate player intent because the player may inspect and decline a hint, the state may change after preview, and one accepted conclusion must become one atomic undoable transition. Application reuses the existing engine action boundary and HTTP `POST /sessions/{id}/actions`; `apply-hint` is an action kind, not a dedicated endpoint or second mutation system. Value placement and note replacement remain the underlying state changes.

`plan_id` is derived from a canonical source-state digest, strategy ID, ordered evidence, and conclusion. Repeating a hint query against unchanged game state returns the same plan and IDs. An `apply-hint` action carries the previewed `plan_id`, allowing the engine to reject a changed source state or recomputed conclusion instead of applying a deduction different from the one taught. The HTTP revision remains the transport concurrency guard; `plan_id` is the engine identity shared by local and network clients.

A client does not translate a preview into unrelated ordinary actions because that would discard exact-plan verification and could split a multi-cell elimination across history entries. The `apply-hint` kind exists for provenance, validation, and atomicity while remaining part of the ordinary action union.

A placement conclusion records one value transition. An elimination conclusion records one atomic manual-note transition across every affected cell: existing notes are filtered without adding candidates, while an affected cell with no notes uses its current legal candidates as the local baseline before the proven digits are removed. This bounded materialization makes the deduction visible without replacing unrelated player notes or changing `core.Board.Candidates`.

Elimination application never places a value, even when the remaining notes form a single. The complete note delta is one undoable history entry, and a conclusion already absent from every affected note set is stale or consumed rather than an accepted no-op.

## Graceful Degradation and Accessibility

A fully capable renderer presents each scene and its message in order. A limited renderer presents the same ordered messages and conclusion while ignoring unsupported marks or target types. The simplest renderer may present `summary` and the conclusion, but must not invent omitted reasoning.

Step order, messages, roles, and explicit effects carry the meaning independently of color, position, or motion. Renderers expose the active step and step count to assistive technology, preserve user-controlled pacing, and avoid automatic animation that blocks reading or input.

Unsupported semantic data is ignored visibly rather than guessed. A renderer may report reduced presentation capability, but protocol consumption remains successful when fallback text and conclusion are supported.

## Reference Teaching Plans

`.aidoc/designs/hint-reference-plans.md` defines the canonical Naked Single, Hidden Single, and Naked Pair examples. The examples prove that the same plan contract expresses placement and elimination conclusions without renderer-owned reasoning.

## Delivery Boundary

`game.Game.Hint` now returns the complete plan, `game.ApplyHint` requires its deterministic `plan_id`, and the engine applies either a placement or a multi-cell note elimination as one undoable transition. `solver.Evidence` supplies typed Naked Single, Hidden Single, and Naked Pair facts; the shared composer owns their reference teaching order. Other strategies use the same plan envelope and plain-text conclusion until their typed evidence migrates.

The line CLI consumes the plan summary and applies the exact plan in one command. The TUI keeps the complete plan during preview and submits its identity on Enter. The current version 1 HTTP adapter temporarily projects a plan into its existing hint shape and binds an apply request to the engine's current plan; the next backend slice replaces that transport schema with the complete plan and explicit `plan_id`, regenerates adapters, and extends built-binary API acceptance.

A strategy joins the typed-evidence set only with evidence tests, plan contract tests, and at least one renderer-neutral reference assertion. The migration is complete when the HTTP and graphical clients consume complete plans and no renderer parses fallback prose or switches on a technique name to reconstruct Sudoku logic.
