---
domain: Designs
status: Active
entry_points:
  - solver/move.go
  - game/contract.go
  - game/hint_plan.go
  - api/openapi.yaml
dependencies:
  - .aidoc/architecture/guidelines.md
  - .aidoc/designs/game-engine.md
  - .aidoc/designs/web-api.md
  - .aidoc/designs/localization-contract.md
  - .aidoc/designs/hint-reference-plans.md
  - .aidoc/designs/e2e-test-scenarios.md
---

# Hint Presentation Protocol

The hint presentation protocol is the portable teaching contract within the one shared hint system.
It turns one solver recommendation into a plan that CLI, TUI, web, and future clients can render
without reinterpreting Sudoku logic. The engine owns semantic evidence, typed reasoning, ordered
explanation, and the logical conclusion; each client owns visual style, interaction, and
capability-aware rendering.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/architecture/guidelines.md` | Solver and package dependency boundaries |
| `.aidoc/designs/game-engine.md` | Authoritative hint query and action semantics |
| `.aidoc/designs/web-api.md` | Revisioned transport and client-neutral contract |
| `.aidoc/designs/localization-contract.md` | Semantic message descriptors and shared English fallback |
| `.aidoc/designs/hint-reference-plans.md` | Concrete placement and elimination teaching plans |
| `.aidoc/designs/e2e-test-scenarios.md` | Maintained black-box suite map and scenario ownership |
| `solver/evidence.go` | Typed candidate, unit, elimination, and placement evidence |
| `game/contract.go` | Engine hint preview and application boundary |
| `game/hint_plan.go` | Hint-plan data model, deterministic identity, and family teaching composers |

## Why One Shared Hint System Exists

A position, value, technique name, and prose reason are enough to apply a hint but not enough to
teach it consistently. A browser otherwise invents highlights, a TUI invents symbols, and a CLI
invents ordering independently, which lets presentation layers disagree about the same deduction.

The protocol standardizes meaning rather than styling. Strategy and engine code describe what facts
matter, what changed, and why the conclusion follows; renderers decide whether semantic roles become
color, borders, motion, symbols, speech, or plain text.

The project is still in development, so the protocol replaces superseded hint shapes instead of
preserving them through compatibility adapters. The codebase keeps only the current contract; a
future semantic change replaces it across the coordinated clients rather than introducing parallel
versions.

## Ownership Boundaries

The hint system has four owners. Keeping them separate prevents a renderer from becoming a second
Sudoku engine.

| Owner | Owns | Does not own |
|---|---|---|
| Strategy solver | Typed evidence for one deduction | Teaching order, prose, or visual styling |
| Engine composer | `HintPlan`, teaching order, scenes, and descriptor arguments | Locale wording or client interaction |
| Message catalog | Canonical English descriptors and argument contracts | Deduction selection or scene geometry |
| Client renderer | Interaction, visual treatment, and capability-aware degradation | Reconstructing logic from names or prose |

Placement and candidate elimination are equal conclusions. Unsupported or incomplete family evidence
produces no plan; there is no generic strategic observe/compare/apply fallback. Backtracker is the
separate complete-search boundary and uses its own explicit explanation.

### Teaching families

The engine composes strategy plans from closed family evidence. The family object carries only the
facts needed to prove the deduction; scenes provide presentation emphasis, and the conclusion
remains the sole applicable effect.

| Family evidence | Strategies | Teaching shape |
|---|---|---|
| `subset-reservation` | Naked Pair–Quad | Introduce the cells, establish the reserved digits, then apply the exact consequence |
| `hidden-subset-confinement` | Hidden Pair–Quad | Confine the digits to the cells, identify extras, then remove only those extras |
| `intersection-confinement` | Pointing Pair, Box-Line Reduction | Establish source confinement, show the intersecting unit, then apply the outside consequence |
| `fish-confinement` | X-Wing, Swordfish, Jellyfish | Introduce base lines incrementally, establish cover-line confinement, then apply exact eliminations |
| Typed pivots and links | XY-Wing, XYZ-Wing, W-Wing | Introduce roles and causal links before the exact target consequence |
| Typed uniqueness evidence | BUG+1, Unique Rectangle Types 1–4 | Establish the exception or rectangle relation before placement or elimination |
| Typed graph or chain evidence | Simple Coloring, X-Cycles, XY-Chain | Grow one bounded relation into the complete proof, preserving order and exact endpoints |

`.aidoc/designs/hint-reference-plans.md` is the canonical owner of maintained storyboards,
coordinates, scene counts, and family examples. This protocol owns the portable rules those examples
must satisfy. Clients link to the reference plans and generated fixtures instead of restating every
strategy sequence.

Valid alternate evidence shapes stay inside the same family vocabulary. Simple Coloring transports
only deduplicated links from the selected component. X-Cycles and XY-Chain scale cumulative scenes
to the actual loop or chain. Unique Rectangle Types 1–4 preserve their typed family descriptors for
alternate corner cardinalities or elimination-only conclusions. Hidden Single treats placed blockers
as optional witnesses, so valid empty-witness evidence still composes a complete plan.

### Narration boundary

A renderer consumes a complete plan and never infers reasoning from technique names or fallback
prose. The narration IR is a closed token stream of semantic message descriptors, digits, and typed
targets. An invalid or unknown token invalidates that step's complete stream, causing the renderer
to use the full fallback descriptor rather than guess at a partial sentence.

Every valid step has a complete descriptor, and every conclusion identifies its exact digit and cell
target. Limited renderers may use the complete summary or fallback descriptor without changing the
underlying plan.

## HintPlan Contract

Every plan contains:

| Field | Meaning |
|-------|---------|
| `plan_id` | Deterministic identity bound to the source game state and conclusion |
| `kind` | `value-correction`, `candidate-correction`, or `strategy` |
| `title` | Semantic presentation-title descriptor valid for corrections and strategies |
| `strategy` | Optional strategy ID, name descriptor, grade ID, and grade-name descriptor; present only for strategy plans |
| `summary` | Complete semantic summary descriptor with canonical English fallback |
| `reason` | Optional closed family-level reason for strategies with a dedicated reason variant; carries the typed facts that make the deduction necessary |
| `steps` | Ordered, independently renderable teaching steps |
| `conclusion` | Typed clear, candidate replacement, placement, or elimination effect |

Every step contains a stable ID, a semantic kind, a complete message descriptor, structured
narration, a declarative scene, and an optional logical effect. Structured narration never replaces
the fallback: every emitted stream contains only valid descriptor, target, or digit variants; every
descriptor matches the canonical argument contract; every referenced target is valid in the source
plan; and an unsupported stream is discarded as a whole. Scenes carry `marks`, stable candidate
`nodes`, and typed `links`; each step sends complete arrays, including empty arrays, so a renderer
never retains stale visual state. Step kinds are `observe`, `explain`, and `apply`. Observation and
explanation steps have no effect; the final apply step owns the plan's one atomic effect.
Logic-sized strategies may emit several explanation steps, but each one must add a fact or relation.

Every scene is a complete snapshot for that step rather than a delta from the previous scene. A
client can therefore move forward or backward, render one step, skip unsupported visual
reinforcement, or recover after interruption without retaining hidden highlight state.

## Typed Reasoning and Teaching Sequence

A strategy with a dedicated reason variant carries one closed family-level `reason` object. The
object records only the facts needed to establish the logical bridge; scenes still own visual
emphasis, and the conclusion remains the sole applicable effect.

| `reason.kind` | Required facts | Current strategies |
|---|---|---|
| `subset-reservation` | one unit, reserved digits, premise cells, and exact action targets | Naked Pair, Naked Triple, Naked Quad |
| `hidden-subset-confinement` | one unit, confined digits, their exact cells, and candidates removed from those cells | Hidden Pair, Hidden Triple, Hidden Quad |
| `intersection-confinement` | one digit, source unit, intersecting unit, exact premise candidates, and action targets | Pointing Pair, Box-Line Reduction |
| `fish-confinement` | one digit, two to four ordered base units, the same number of ordered cover units, pattern candidates, and exact action targets | X-Wing, Swordfish, Jellyfish |
| `implication-chain` | ordered bivalue groups, typed links, endpoint digit, and exact action targets | XY-Chain |

A variant follows teaching logic rather than display names and may serve several techniques when the
same facts prove their deductions. The naked-subset variant therefore stores the complete
reserved-digit union for pairs, triples, and quads. The hidden-subset variant stores the digits
confined to the same number of cells and the exact non-confined candidates removed from those cells.
A new variant is justified only when another family needs facts that an existing variant cannot
express without an overloaded flag or client inference.

Teaching-sequence length follows the deduction rather than a fixed three-step template:

| Kind | Required answer | Responsibility |
|---|---|---|
| `observe` | What should the player notice? | Introduce one bounded premise; repeat only for distinct facts. |
| `explain` | Why does the premise force a result? | Add one logical relation at a time until the bridge is complete. |
| `apply` | What exactly should change? | Preserve necessary context and isolate the one atomic placement or grouped elimination. |

Adjacent steps cannot repeat the same statement and scene without adding a fact or relation. Only
the final `apply` step carries the plan effect; `observe` and `explain` remain read-only. Complete
descriptors use typed scalar or bounded list arguments, so a locale can reorder and pluralize
digits, cells, candidates, and units without parsing English or replaying backend token order. The
localization contract owns the argument vocabulary and English fallback; this document owns why each
descriptor appears in the reasoning sequence.

## Strategic Hint Review Contract

A strategic-hint design is grounded in one maintained real example at the exact reached board state.
The report records the target revisions, reproduction route, complete board and authoritative
candidate grid, typed premises, exact target, and one supported placement or grouped elimination.
Synthetic candidate-only boards may supplement an explanation but cannot replace the maintained
product example when that example is available.

The actual product board is the primary storyboard surface. Its automatic candidate grid is visible
from the first teaching step, and each scene is a complete snapshot. Every step adds one
learner-visible fact or relation; later scenes retain earlier evidence whenever the causal chain
still depends on it. Formal strategy vocabulary appears only with the applying conclusion. Marks and
links carry typed logic rather than decoration, and coordinates, symbols, line styles, labels,
narration, and accessible descriptions preserve meaning without relying on color.

Review evidence covers every teaching step on the actual board plus the final English, Simplified
Chinese, and Traditional Chinese presentation at desktop and phone sizes. Inspection checks
candidate visibility, cumulative proof, clipping, overflow, collisions, obscured candidates, board
geometry, focus, and non-color meaning. Visual fit does not establish linguistic quality; the
localization contract owns authorship and editorial acceptance.

The exact maintained build, strategy-matrix, complete-game, terminal, API, and browser commands
remain owned by `AGENT.md`, `.aidoc/designs/e2e-test-scenarios.md`, its scenario catalogs, and the
coordinated browser repository. A quality report points to those owners and records results instead
of copying their command inventory or acceptance policy.

## Semantic Scene Vocabulary

Scene targets are typed references to a cell, one candidate within a cell, or a row, column, or box.
Rows and columns use the engine's internal zero-based coordinates and the HTTP adapter converts them
to the transport boundary's one-based coordinates.

Scene marks pair one target with one semantic role:

| Role | Meaning |
|------|---------|
| `focus` | Region or item currently under examination |
| `premise` | Fact already established and used by the deduction |
| `eliminated` | Candidate proven impossible by this deduction |
| `conclusion` | Applied change that completes the teaching step |
| `problem` | Cell whose current value or candidate set is inconsistent |
| `extra` | Candidate ruled out by accepted values |

Semantic roles carry no palette or animation requirement. Color cannot be the sole carrier of
meaning, motion is optional, and the descriptor fallback remains understandable alongside the step's
typed targets, scene, and effect without requiring prose parsing.

Candidate nodes provide stable IDs and may carry an `A`/`B` alternating-group label or a one-based
chain order. Typed links connect node IDs as `strong`, `weak`, or `implication`; renderers express
those meanings with solid lines, dashed lines, or arrows and provide an equivalent legend or
accessible description. Unit marks communicate scope only when the unit itself is the fact under
examination. Chain and coloring scenes therefore avoid painting every cell in every participating
unit.

Complex scenes progress from one bounded relation to the complete pattern. W-Wing begins with its
matching bivalue cells and introduces its bridge on the next step; XY-Wing begins with the pivot,
adds one implication per case, preserves both for the combined guarantee, and adds both
target-witness links only on the exact result; XYZ-Wing follows the same two implication cases,
keeps them through its explicit pivot-value and combined-guarantee stages, and adds three
target-witness links on the exact result; coloring and cycle plans introduce one link and then
reveal the complete relationship graph. XY-Chain starts with its first bivalue cell, adds exactly
one cumulative implication link per explanation step, and preserves the complete chain while
isolating the exact result in the apply step. Backward navigation remains deterministic because each
scene is complete.

Logical effects are separate from marks. A mark explains presentation emphasis; an effect declares
the verifiable clear, complete candidate replacement, placement, or candidate elimination produced
by the step. The plan conclusion equals the final effect and must be valid against the source game
state.

## Validation and Deduction Selection

Interactive hint orchestration validates the authoritative player state before choosing a strategy.
A visible invalid value returns a value-correction plan for that exact cell. The engine then checks
each editable empty cell against accepted-value legality and the puzzle's unique continuation. The
first set with an impossible digit, an empty state, or a missing solution-supporting digit returns a
plan that discards the cell's deductions and restores every candidate allowed directly by accepted
row, column, and box values. The plan may mark locally impossible extras, but it never identifies
the solved digit or labels a restored digit as missing.

Only a consistent board and candidate grid proceed to the configured strategy order. The engine
applies the grid as eliminations on a detached solver board, so the selected technique and
explanation use exactly the state the player sees. Strategy selection returns the first meaningful
deduction, whether it places a value or only eliminates candidates; it never silently consumes
elimination progress to search for a later placement.

Full solving and difficulty classification may continue chaining eliminations on their private
solving state until a placement or completed solve. The shared strategy implementation and evidence
therefore serve teaching, solving, and classification even though those callers stop at different
boundaries.

Candidate hygiene is a correction plan rather than a strategy conclusion. Corrections use the same
teaching surface and application guarantees but are titled as fixes, not as Sudoku techniques.

## Query, Navigation, and Application

One read-only hint query returns the complete `HintPlan`. The client fetches that plan once; Next
and Back navigate its ordered steps locally without another request or any puzzle, dirty-state,
revision, or history change.

Applying the conclusion is a separate player intent because the player may inspect and decline a
hint, the state may change after preview, and one accepted conclusion must become one atomic
undoable transition. Application reuses the engine action boundary: sealed guests post to
`/api/v1/guest/games/actions`, while authenticated owners post to
`/api/v1/account/games/{accountGameId}/actions`. `apply-hint` is an action kind, not a dedicated
endpoint or second mutation system. Value placement and authoritative candidate replacement or
elimination remain the underlying state changes.

`plan_id` is derived from a canonical source-state digest, strategy ID, ordered evidence, and
conclusion. Repeating a hint query against unchanged game state returns the same plan and IDs. An
`apply-hint` action carries the previewed `plan_id`, allowing the engine to reject a changed source
state or recomputed conclusion instead of applying a deduction different from the one taught. The
HTTP revision remains the transport concurrency guard; `plan_id` is the engine identity shared by
local and network clients.

A client does not translate a preview into unrelated ordinary actions because that would discard
exact-plan verification and could split a multi-cell elimination across history entries. The
`apply-hint` kind exists for provenance, validation, and atomicity while remaining part of the
ordinary action union.

A value correction clears one invalid value. A candidate correction replaces one complete cell set.
A placement conclusion records one value transition, and an elimination conclusion removes its
proven digits from the authoritative grid across every affected cell. Each conclusion is one
undoable history entry; an already-consumed conclusion is stale rather than an accepted no-op.

Elimination application never places a value even when one candidate remains. Later hints read the
reduced authoritative grid instead of rediscovering the same elimination.

## Graceful Degradation and Accessibility

A fully capable renderer presents each scene and renders narration chunks in order, withholding
formal strategy metadata until the concluding step. Family-specific facts remain in typed marks,
nodes, links, targets, digits, and effects; shared semantic descriptors provide the translatable
teaching instruction instead of duplicating a preformatted English reconstruction of that evidence.
The TUI uses compact `rNcN` locators, and the browser uses direct highlighted references with a
visible secondary locator plus a fully expanded accessible name. Renderers fall back to
`message.text` only when structured narration cannot be consumed. A limited renderer presents the
same ordered descriptor fallbacks and conclusion while ignoring unsupported marks or target types.
The line CLI is deliberately narrower: it renders the concluding placement narration as
row-and-column prose; for candidate corrections and eliminations it retains the exact plan but shows
only a capability limitation. The player may explicitly apply that hidden candidate conclusion and
request the next hint, while `sudoku tui` remains the complete teaching surface. The simplest
renderer may present `summary.text` and the conclusion, but must not invent omitted reasoning.

Step order, messages, roles, and explicit effects carry the meaning independently of color,
position, or motion. Renderers expose the active step and step count to assistive technology,
preserve user-controlled pacing, and avoid automatic animation that blocks reading or input.

Unsupported semantic data is ignored visibly rather than guessed. A renderer may report reduced
presentation capability, but protocol consumption remains successful when fallback text and
conclusion are supported.

Localization consumes stable descriptor IDs and typed arguments without duplicating narration arrays
or reconstructing Sudoku logic. Browser catalogs control translated ordering and grammar; the
target, scene, teaching-order, canonical English fallback, and effect contracts remain unchanged.

## Reference Teaching Plans

`.aidoc/designs/hint-reference-plans.md` defines canonical placement, subset, intersection, fish,
relational, uniqueness, coloring, cycle, and chain examples. The examples prove that the same plan
contract expresses placement and elimination conclusions without renderer-owned reasoning.

## Delivery Boundary

`game.Game.Hint` returns the complete plan, `game.ApplyHint` requires its deterministic `plan_id`,
and the engine applies a value correction, candidate correction, placement, or multi-cell candidate
elimination as one undoable transition. `game.completeTeachingEvidence` preserves strategy-authored
evidence and completes missing typed candidate neighborhoods, affected units, ruled-out candidates,
and exact elimination effects from the solver board transition. All 23 registered strategies
therefore produce ordered semantic descriptor steps and a machine-readable conclusion; canonical
English fallback remains presentation text rather than a client parsing contract.

The line CLI obtains one plan, renders placement narration and the exact cell or one
candidate-capability limitation, then immediately submits that exact `plan_id`. Each command applies
one conclusion as one Undo entry and stops; generation or application failure leaves state
unchanged, and a later `hint` queries the resulting state. The TUI keeps every complete plan during
read-only preview, navigates its steps locally, uses color-independent semantic marks, and submits
its identity on Enter. The HTTP contract returns the complete plan with one-based transport
coordinates, requires its explicit `plan_id` on `apply-hint`, and returns the applied plan in the
action result. Built-binary terminal and API acceptance proves line-CLI degradation and progression,
complete TUI hint-driven games, exact-plan application, stale-plan rejection at engine boundaries,
and one revision or undo record per accepted conclusion.

`game.TestEveryRegisteredStrategyBuildsTypedPlanFromRealGame` drives the canonical solver to each
strategy using complete valid games from the maintained calibration corpus and catalog audit.
`game.TestStrategicHintRegressionCorpusHasNoGenericFallbackOrPanic` replays the captured
alternate-shape and empty-witness games through exact preview/application until solved; every
emitted step must remain family-specific, and composition must remain total. The coverage contract
rejects synthetic candidate-only boards, missing typed premises, invalid or absent narration on any
step, concluding narration without exact digit and target tokens, unstable plan identities,
incomplete effects, generic strategic fallback, and any registered strategy without a placement or
elimination plan.
