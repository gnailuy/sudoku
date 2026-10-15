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
| `solver/evidence.go` | Typed candidate, unit, elimination, and placement evidence |
| `game/contract.go` | Engine hint preview and application boundary |

## Why a Shared Teaching Contract Exists

A position, value, technique name, and prose reason are enough to apply a hint but not enough to teach it consistently. A browser otherwise invents highlights, a TUI invents symbols, and a CLI invents ordering independently, which lets presentation layers disagree about the same deduction.

The protocol standardizes meaning rather than styling. Strategy and engine code describe what facts matter, what changed, and why the conclusion follows; renderers decide whether semantic roles become color, borders, motion, symbols, speech, or plain text.

The project is still in development, so the protocol replaces superseded hint shapes instead of preserving them through compatibility adapters. The codebase keeps only the current contract; a future semantic change replaces it across the coordinated clients rather than introducing parallel versions.

## Ownership Boundaries

A strategy produces typed evidence for one deduction: examined units, candidate constraints, eliminations, placements, and the technique identity. Placement and candidate elimination are equally valid conclusions; elimination strategies are not duplicated as hint-only variants. Strategy evidence contains no colors, animation timing, widget commands, prose parsing requirements, or transport models.

A shared engine composer converts typed evidence into one `HintPlan`. The composer owns default teaching order and wording so 23 strategies do not duplicate client choreography. Naked Single and Hidden Single establish direct and unit-scoped placement teaching. Naked/Hidden Pair–Quad and Pointing Pair/Box-Line Reduction establish subset and intersection teaching. Fish and wing families identify their typed lines, intersections, pivots, wings, and candidate links before explaining the consequence. XY-Wing records the two pivot-to-wing implications explicitly: the first step introduces one arrow, and the comparison and conclusion preserve both arrows. BUG+1 and Unique Rectangle Types 1–4 identify their candidate exceptions and uniqueness relationships. Simple Coloring records alternating candidate partitions and must-contain links; X-Cycles records an ordered loop of must-contain and cannot-both links; XY-Chain records ordered bivalue cells and their shared-candidate implications. Every family reveals formal strategy vocabulary only in the concluding because-clause.

A renderer consumes a complete plan and never infers Sudoku reasoning from technique names or fallback prose. The hint narration IR is a renderer-neutral token stream that separates literal text, digits, and typed semantic targets, allowing each interface to name the same cell, candidate, or unit naturally without parsing a message. Each token is one closed discriminated-union variant with exactly one required payload; invalid or unknown variants invalidate the whole stream for that step so the renderer uses the complete fallback message instead of guessing. Every step for all 23 strategies carries valid structured narration, and every concluding stream identifies its exact digit and cell target. Specialized placement and chain wording may expose additional targets while the complete plain-text message remains available as a defensive fallback. Limited renderers receive the same fallback messages joined in order as `summary`.

## HintPlan Contract

Every plan contains:

| Field | Meaning |
|-------|---------|
| `plan_id` | Deterministic identity bound to the source game state and conclusion |
| `strategy` | Stable strategy ID, display name, and strategy grade |
| `summary` | Complete plain-text fallback for the deduction |
| `steps` | Ordered, independently renderable teaching steps |
| `conclusion` | Machine-readable placement or candidate eliminations |

Every step contains a stable ID, a semantic kind, a complete fallback message, optional structured narration, a declarative scene, and an optional logical effect. Structured narration never replaces the fallback: every emitted stream contains only valid text, target, or digit variants; every referenced target is valid in the source plan; and an unsupported stream is discarded as a whole. Scenes carry `marks`, stable candidate `nodes`, and typed `links`; each step sends complete arrays, including empty arrays, so a renderer never retains stale visual state. Initial step kinds are `observe`, `compare`, `eliminate`, and `conclude`; changing their meaning requires a coordinated contract replacement across the engine and clients.

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

Candidate nodes provide stable IDs and may carry an `A`/`B` alternating-group label or a one-based chain order. Typed links connect node IDs as `strong`, `weak`, or `implication`; renderers express those meanings with solid lines, dashed lines, or arrows and provide an equivalent legend or accessible description. Unit marks communicate scope only when the unit itself is the fact under examination. Chain and coloring scenes therefore avoid painting every cell in every participating unit.

Complex scenes progress from one bounded relation to the complete pattern. The first XY-Wing, coloring, cycle, or chain step introduces one link and its endpoints; the comparison step reveals the complete relationship graph; the conclusion preserves that graph while isolating the exact result. Backward navigation remains deterministic because each scene is complete.

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

A placement conclusion records one value transition. An elimination conclusion records one atomic manual-note transition across every affected cell. When the manual-note map is empty, the engine first materializes the complete current legal-candidate grid and then removes the proven digits; clients can therefore move from automatic-candidate preview to the reduced player-owned notes without losing the rest of the grid. When any manual notes already exist, existing notes are preserved and only an affected empty note set uses its current legal candidates as the local baseline. The same action records the strategy-derived eliminations in the solver-safe board, so later hints and automatic candidates continue from the consumed deduction instead of rediscovering it.

Elimination application never places a value, even when the remaining notes form a single. The complete note delta is one undoable history entry, and a conclusion already absent from every affected note set is stale or consumed rather than an accepted no-op.

## Graceful Degradation and Accessibility

A fully capable renderer presents each scene and renders available narration chunks in order, withholding formal strategy metadata until the concluding step. The line CLI expands targets as row-and-column prose, the TUI uses compact `rNcN` locators, and the browser uses direct highlighted references with a visible secondary locator plus a fully expanded accessible name. Renderers fall back to `message` when narration is absent. A limited renderer presents the same ordered messages and conclusion while ignoring unsupported marks or target types. The simplest renderer may present `summary`, which joins those messages in teaching order, and the conclusion, but must not invent omitted reasoning.

Step order, messages, roles, and explicit effects carry the meaning independently of color, position, or motion. Renderers expose the active step and step count to assistive technology, preserve user-controlled pacing, and avoid automatic animation that blocks reading or input.

Unsupported semantic data is ignored visibly rather than guessed. A renderer may report reduced presentation capability, but protocol consumption remains successful when fallback text and conclusion are supported.

Localization is a required later evolution, not a reason to duplicate translated token arrays now. When a second locale is introduced, the literal text layer becomes locale message IDs plus typed digit and target arguments, using ICU MessageFormat 2 or Fluent semantics for translator-controlled ordering and grammar; the Sudoku target, scene, teaching-order, fallback, and effect contracts remain unchanged.

## Reference Teaching Plans

`.aidoc/designs/hint-reference-plans.md` defines canonical placement, subset, intersection, fish, relational, uniqueness, coloring, cycle, and chain examples. The examples prove that the same plan contract expresses placement and elimination conclusions without renderer-owned reasoning.

## Delivery Boundary

`game.Game.Hint` returns the complete plan, `game.ApplyHint` requires its deterministic `plan_id`, and the engine applies either a placement or a multi-cell note elimination as one undoable transition. `game.completeTeachingEvidence` preserves strategy-authored evidence and completes missing typed candidate neighborhoods, affected units, ruled-out candidates, and exact elimination effects from the solver board transition. All 23 registered strategies therefore produce ordered semantic steps and a machine-readable conclusion; fallback prose remains presentation text rather than a client parsing contract.

The line CLI prints and caches the complete plan without mutation, then requires `hint apply` or `hint cancel`; apply submits only the cached identity. Any successful non-hint CLI mutation explicitly discards the cached preview, while failed and no-op commands retain it. The TUI keeps the complete plan during preview, navigates its steps locally, uses color-independent semantic marks, and submits its identity on Enter. The HTTP contract returns the complete plan with one-based transport coordinates, requires its explicit `plan_id` on `apply-hint`, and returns the applied plan in the action result. Built-binary terminal and API acceptance proves read-only preview, exact-plan application, mutation-driven CLI discard, stale-plan rejection at engine boundaries, and one revision or undo record per accepted conclusion.

`game.TestEveryRegisteredStrategyBuildsTypedPlanFromRealGame` drives the canonical solver to each strategy using complete valid games from the maintained calibration corpus and catalog audit. The coverage contract rejects synthetic candidate-only boards, missing typed premises, invalid or absent narration on any step, concluding narration without exact digit and target tokens, unstable plan identities, incomplete effects, and any registered strategy without a placement or elimination plan.
