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
---

# Hint Presentation Protocol

The hint presentation protocol is the portable teaching contract within the one shared hint system. It turns one solver recommendation into a plan that CLI, TUI, web, and future clients can render without reinterpreting Sudoku logic. The engine owns semantic evidence, typed reasoning, ordered explanation, and the logical conclusion; each client owns visual style, interaction, and capability-aware rendering.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/architecture/guidelines.md` | Solver and package dependency boundaries |
| `.aidoc/designs/game-engine.md` | Authoritative hint query and action semantics |
| `.aidoc/designs/web-api.md` | Revisioned transport and client-neutral contract |
| `.aidoc/designs/localization-contract.md` | Semantic message descriptors and shared English fallback |
| `.aidoc/designs/hint-reference-plans.md` | Concrete placement and elimination teaching plans |
| `solver/evidence.go` | Typed candidate, unit, elimination, and placement evidence |
| `game/contract.go` | Engine hint preview and application boundary |
| `game/hint_plan.go` | Hint-plan data model, deterministic identity, and family teaching composers |

## Why One Shared Hint System Exists

A position, value, technique name, and prose reason are enough to apply a hint but not enough to teach it consistently. A browser otherwise invents highlights, a TUI invents symbols, and a CLI invents ordering independently, which lets presentation layers disagree about the same deduction.

The protocol standardizes meaning rather than styling. Strategy and engine code describe what facts matter, what changed, and why the conclusion follows; renderers decide whether semantic roles become color, borders, motion, symbols, speech, or plain text.

The project is still in development, so the protocol replaces superseded hint shapes instead of preserving them through compatibility adapters. The codebase keeps only the current contract; a future semantic change replaces it across the coordinated clients rather than introducing parallel versions.

## Ownership Boundaries

A strategy produces typed evidence for one deduction: examined units, candidate constraints, eliminations, placements, and the technique identity. Placement and candidate elimination are equally valid conclusions; elimination strategies are not duplicated as hint-only variants. Strategy evidence contains no colors, animation timing, widget commands, prose parsing requirements, or transport models.

A shared engine composer converts typed evidence into one `HintPlan`. The composer owns teaching order plus descriptor selection and arguments, while the canonical message catalog owns English wording, so 23 strategies do not duplicate client choreography or prose. Naked Pair–Quad use the closed `subset-reservation` reason, Hidden Pair–Quad use `hidden-subset-confinement`, Pointing Pair and Box-Line Reduction use `intersection-confinement`, X-Wing, Swordfish, and Jellyfish use `fish-confinement`, and XY-Chain uses `implication-chain`; other families use their current typed evidence and complete scenes. Naked Single uses three learner-sized stages: show one placed peer for every digit excluded by the target row, column, or box; reveal the sole remaining candidate; then place it and name the strategy in the applying teaching sentence. Hidden Single uses four learner-sized stages: choose one digit in one unit; retain one placed blocker and one crossed candidate slot for every excluded empty cell; identify the sole remaining location; then place the digit and name the strategy. Naked Pair uses four learner-sized stages: inspect the first two-candidate cell in its unit; retain it while adding the matching cell; explain that the two cells reserve both digits and identify the exact external candidate consequence; then remove only that candidate and name the strategy. Naked Triple uses five learner-sized stages: introduce each of its three cells one at a time, establish their three-digit reservation and the exact resulting candidate removal, then place the sole remaining digit and name the strategy. Hidden Pair uses four learner-sized stages: confine the first digit to two cells, add the second digit in those same cells, identify every extra candidate in the pair cells, then remove exactly those candidates and name the strategy. Its complete scenes contain 2, 5, 8, and 8 typed marks without decorative links. Hidden Triple and Hidden Quad retain the shared subset sequence. Pointing Pair uses four learner-sized stages on the maintained box-first example: introduce the first candidate in the box, add the only other candidate and their shared column, apply the confinement to the exact outside candidate, then place the resulting sole candidate and name the strategy. Its complete scenes contain 2, 4, 5, and 6 typed marks without decorative links. Other Box-Line Reduction directions retain the shared intersection-confinement sequence. X-Wing uses four learner-sized stages on the maintained row-based example: confine candidate 3 to columns 2 and 3 in row 4, add row 8's matching pair, explain the two-column consequence, then remove candidate 3 from `r1c2`, `r1c3`, and `r3c2` and name the strategy. Its complete scenes contain 2, 6, 8, and 11 typed marks without decorative links. Swordfish and Jellyfish retain the shared fish sequence. Wing families identify their typed pivots, wings, and candidate links before explaining the consequence. XY-Wing uses five learner-sized stages: identify the bivalue pivot, follow each pivot case to the corresponding wing, combine the two cases into one shared-digit guarantee, and apply the exact result. Its complete scenes progress through 0, 1, 2, 2, and 4 typed relationships; the final two weak links show that the target sees the shared candidate in both wings. XYZ-Wing uses six learner-sized stages: identify the three-candidate pivot, follow the two non-shared pivot cases to their wings, cover the shared-candidate pivot case, combine all three cases, and apply the exact result. Its complete scenes progress through 0, 1, 2, 2, 2, and 5 typed relationships; the final three weak links show that the target shares a unit with every possible shared candidate. W-Wing first identifies the two matching bivalue cells without formal vocabulary, then shows the conjugate bridge, carries each bridge outcome to the opposite candidate in its corresponding bivalue cell with two implication arrows, and only then applies the elimination. The bridge appears as one strong link from the second step onward; both case arrows appear on the third and final steps. BUG+1 names the sole trivalue cell, its candidates, the uniqueness-breaking extra candidate, and the affected row, column, and box. Unique Rectangle Types 1–4 name the four corners and shared pair before explaining the exact expanded-corner, shared-extra, unit-subset, or conjugate-pair relationship through complete descriptors. Simple Coloring begins with one forced pair, expands the alternating A/B chain, states the two complete possibilities, explains each exact elimination through one visible candidate from each group, and names the strategy only on the final action. X-Cycles begins with one strong pair, follows alternating weak/strong consequences through cumulative 1-, 3-, 5-, and 7-link scenes, closes the loop in the final named unit, and applies the exact placement only after exposing the contradiction. XY-Chain begins with one bivalue endpoint, carries the rejected endpoint candidate through two implications at a time, reaches the other endpoint, and applies one exact value only after proving that one endpoint must contain their common digit. Every family reveals formal strategy vocabulary only in the applying because-clause.

A renderer consumes a complete plan and never infers Sudoku reasoning from technique names or fallback prose. The hint narration IR is a renderer-neutral token stream that separates semantic message descriptors, digits, and typed targets, allowing each interface to name the same cell, candidate, or unit naturally without parsing prose. Each token is one closed discriminated-union variant with exactly one required payload; invalid or unknown variants invalidate the whole stream for that step so the renderer uses the complete fallback descriptor instead of guessing. Every step for all 23 strategies carries valid structured narration, and every concluding stream identifies its exact digit and cell target. Specialized placement and chain wording may expose additional targets while the complete step descriptor remains available as a defensive fallback. Limited renderers receive the plan's complete summary descriptor.

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

Every step contains a stable ID, a semantic kind, a complete message descriptor, structured narration, a declarative scene, and an optional logical effect. Structured narration never replaces the fallback: every emitted stream contains only valid descriptor, target, or digit variants; every descriptor matches the canonical argument contract; every referenced target is valid in the source plan; and an unsupported stream is discarded as a whole. Scenes carry `marks`, stable candidate `nodes`, and typed `links`; each step sends complete arrays, including empty arrays, so a renderer never retains stale visual state. Step kinds are `observe`, `explain`, and `apply`. Observation and explanation steps have no effect; the final apply step owns the plan's one atomic effect. Logic-sized strategies may emit several explanation steps, but each one must add a fact or relation.

Every scene is a complete snapshot for that step rather than a delta from the previous scene. A client can therefore move forward or backward, render one step, skip unsupported visual reinforcement, or recover after interruption without retaining hidden highlight state.

## Typed Reasoning and Teaching Sequence

A strategy with a dedicated reason variant carries one closed family-level `reason` object. The object records only the facts needed to establish the logical bridge; scenes still own visual emphasis, and the conclusion remains the sole applicable effect.

| `reason.kind` | Required facts | Current strategies |
|---|---|---|
| `subset-reservation` | one unit, reserved digits, premise cells, and exact action targets | Naked Pair, Naked Triple, Naked Quad |
| `hidden-subset-confinement` | one unit, confined digits, their exact cells, and candidates removed from those cells | Hidden Pair, Hidden Triple, Hidden Quad |
| `intersection-confinement` | one digit, source unit, intersecting unit, exact premise candidates, and action targets | Pointing Pair, Box-Line Reduction |
| `fish-confinement` | one digit, two to four ordered base units, the same number of ordered cover units, pattern candidates, and exact action targets | X-Wing, Swordfish, Jellyfish |
| `implication-chain` | ordered bivalue groups, typed links, endpoint digit, and exact action targets | XY-Chain |

A variant follows teaching logic rather than display names and may serve several techniques when the same facts prove their deductions. The naked-subset variant therefore stores the complete reserved-digit union for pairs, triples, and quads. The hidden-subset variant stores the digits confined to the same number of cells and the exact non-confined candidates removed from those cells. A new variant is justified only when another family needs facts that an existing variant cannot express without an overloaded flag or client inference.

Teaching-sequence length follows the deduction rather than a fixed three-step template:

| Kind | Required answer | Responsibility |
|---|---|---|
| `observe` | What should the player notice? | Introduce one bounded premise; repeat only for distinct facts. |
| `explain` | Why does the premise force a result? | Add one logical relation at a time until the bridge is complete. |
| `apply` | What exactly should change? | Preserve necessary context and isolate the one atomic placement or grouped elimination. |

Adjacent steps cannot repeat the same statement and scene without adding a fact or relation. Only the final `apply` step carries the plan effect; `observe` and `explain` remain read-only. Complete descriptors use typed scalar or bounded list arguments, so a locale can reorder and pluralize digits, cells, candidates, and units without parsing English or replaying backend token order. The localization contract owns the argument vocabulary and English fallback; this document owns why each descriptor appears in the reasoning sequence.

## Semantic Scene Vocabulary

Scene targets are typed references to a cell, one candidate within a cell, or a row, column, or box. Rows and columns use the engine's internal zero-based coordinates and the HTTP adapter converts them to the transport boundary's one-based coordinates.

Scene marks pair one target with one semantic role:

| Role | Meaning |
|------|---------|
| `focus` | Region or item currently under examination |
| `premise` | Fact already established and used by the deduction |
| `eliminated` | Candidate proven impossible by this deduction |
| `conclusion` | Applied change that completes the teaching step |
| `problem` | Cell whose current value or candidate set is inconsistent |
| `extra` | Candidate ruled out by accepted values |

Semantic roles carry no palette or animation requirement. Color cannot be the sole carrier of meaning, motion is optional, and the descriptor fallback remains understandable alongside the step's typed targets, scene, and effect without requiring prose parsing.

Candidate nodes provide stable IDs and may carry an `A`/`B` alternating-group label or a one-based chain order. Typed links connect node IDs as `strong`, `weak`, or `implication`; renderers express those meanings with solid lines, dashed lines, or arrows and provide an equivalent legend or accessible description. Unit marks communicate scope only when the unit itself is the fact under examination. Chain and coloring scenes therefore avoid painting every cell in every participating unit.

Complex scenes progress from one bounded relation to the complete pattern. W-Wing begins with its matching bivalue cells and introduces its bridge on the next step; XY-Wing begins with the pivot, adds one implication per case, preserves both for the combined guarantee, and adds both target-witness links only on the exact result; XYZ-Wing follows the same two implication cases, keeps them through its explicit pivot-value and combined-guarantee stages, and adds three target-witness links on the exact result; coloring and cycle plans introduce one link and then reveal the complete relationship graph. XY-Chain starts with its first bivalue cell, adds exactly one cumulative implication link per explanation step, and preserves the complete chain while isolating the exact result in the apply step. Backward navigation remains deterministic because each scene is complete.

Logical effects are separate from marks. A mark explains presentation emphasis; an effect declares the verifiable clear, complete candidate replacement, placement, or candidate elimination produced by the step. The plan conclusion equals the final effect and must be valid against the source game state.

## Validation and Deduction Selection

Interactive hint orchestration validates the authoritative player state before choosing a strategy. A visible invalid value returns a value-correction plan for that exact cell. The engine then checks each editable empty cell against accepted-value legality and the puzzle's unique continuation. The first set with an impossible digit, an empty state, or a missing solution-supporting digit returns a plan that discards the cell's deductions and restores every candidate allowed directly by accepted row, column, and box values. The plan may mark locally impossible extras, but it never identifies the solved digit or labels a restored digit as missing.

Only a consistent board and candidate grid proceed to the configured strategy order. The engine applies the grid as eliminations on a detached solver board, so the selected technique and explanation use exactly the state the player sees. Strategy selection returns the first meaningful deduction, whether it places a value or only eliminates candidates; it never silently consumes elimination progress to search for a later placement.

Full solving and difficulty classification may continue chaining eliminations on their private solving state until a placement or completed solve. The shared strategy implementation and evidence therefore serve teaching, solving, and classification even though those callers stop at different boundaries.

Candidate hygiene is a correction plan rather than a strategy conclusion. Corrections use the same teaching surface and application guarantees but are titled as fixes, not as Sudoku techniques.

## Query, Navigation, and Application

One read-only hint query returns the complete `HintPlan`. The client fetches that plan once; Next and Back navigate its ordered steps locally without another request or any puzzle, dirty-state, revision, or history change.

Applying the conclusion is a separate player intent because the player may inspect and decline a hint, the state may change after preview, and one accepted conclusion must become one atomic undoable transition. Application reuses the engine action boundary: sealed guests post to `/api/v1/guest/games/actions`, while authenticated owners post to `/api/v1/account/games/{accountGameId}/actions`. `apply-hint` is an action kind, not a dedicated endpoint or second mutation system. Value placement and authoritative candidate replacement or elimination remain the underlying state changes.

`plan_id` is derived from a canonical source-state digest, strategy ID, ordered evidence, and conclusion. Repeating a hint query against unchanged game state returns the same plan and IDs. An `apply-hint` action carries the previewed `plan_id`, allowing the engine to reject a changed source state or recomputed conclusion instead of applying a deduction different from the one taught. The HTTP revision remains the transport concurrency guard; `plan_id` is the engine identity shared by local and network clients.

A client does not translate a preview into unrelated ordinary actions because that would discard exact-plan verification and could split a multi-cell elimination across history entries. The `apply-hint` kind exists for provenance, validation, and atomicity while remaining part of the ordinary action union.

A value correction clears one invalid value. A candidate correction replaces one complete cell set. A placement conclusion records one value transition, and an elimination conclusion removes its proven digits from the authoritative grid across every affected cell. Each conclusion is one undoable history entry; an already-consumed conclusion is stale rather than an accepted no-op.

Elimination application never places a value even when one candidate remains. Later hints read the reduced authoritative grid instead of rediscovering the same elimination.

## Graceful Degradation and Accessibility

A fully capable renderer presents each scene and renders narration chunks in order, withholding formal strategy metadata until the concluding step. Family-specific facts remain in typed marks, nodes, links, targets, digits, and effects; shared semantic descriptors provide the translatable teaching instruction instead of duplicating a preformatted English reconstruction of that evidence. The TUI uses compact `rNcN` locators, and the browser uses direct highlighted references with a visible secondary locator plus a fully expanded accessible name. Renderers fall back to `message.text` only when structured narration cannot be consumed. A limited renderer presents the same ordered descriptor fallbacks and conclusion while ignoring unsupported marks or target types. The line CLI is deliberately narrower: it renders the concluding placement narration as row-and-column prose; for candidate corrections and eliminations it retains the exact plan but shows only a capability limitation. The player may explicitly apply that hidden candidate conclusion and request the next hint, while `sudoku tui` remains the complete teaching surface. The simplest renderer may present `summary.text` and the conclusion, but must not invent omitted reasoning.

Step order, messages, roles, and explicit effects carry the meaning independently of color, position, or motion. Renderers expose the active step and step count to assistive technology, preserve user-controlled pacing, and avoid automatic animation that blocks reading or input.

Unsupported semantic data is ignored visibly rather than guessed. A renderer may report reduced presentation capability, but protocol consumption remains successful when fallback text and conclusion are supported.

Localization consumes stable descriptor IDs and typed arguments without duplicating narration arrays or reconstructing Sudoku logic. Browser catalogs control translated ordering and grammar; the target, scene, teaching-order, canonical English fallback, and effect contracts remain unchanged.

## Reference Teaching Plans

`.aidoc/designs/hint-reference-plans.md` defines canonical placement, subset, intersection, fish, relational, uniqueness, coloring, cycle, and chain examples. The examples prove that the same plan contract expresses placement and elimination conclusions without renderer-owned reasoning.

## Delivery Boundary

`game.Game.Hint` returns the complete plan, `game.ApplyHint` requires its deterministic `plan_id`, and the engine applies a value correction, candidate correction, placement, or multi-cell candidate elimination as one undoable transition. `game.completeTeachingEvidence` preserves strategy-authored evidence and completes missing typed candidate neighborhoods, affected units, ruled-out candidates, and exact elimination effects from the solver board transition. All 23 registered strategies therefore produce ordered semantic descriptor steps and a machine-readable conclusion; canonical English fallback remains presentation text rather than a client parsing contract.

The line CLI obtains one plan, renders placement narration and the exact cell or one candidate-capability limitation, then immediately submits that exact `plan_id`. Each command applies one conclusion as one Undo entry and stops; generation or application failure leaves state unchanged, and a later `hint` queries the resulting state. The TUI keeps every complete plan during read-only preview, navigates its steps locally, uses color-independent semantic marks, and submits its identity on Enter. The HTTP contract returns the complete plan with one-based transport coordinates, requires its explicit `plan_id` on `apply-hint`, and returns the applied plan in the action result. Built-binary terminal and API acceptance proves line-CLI degradation and progression, complete TUI hint-driven games, exact-plan application, stale-plan rejection at engine boundaries, and one revision or undo record per accepted conclusion.

`game.TestEveryRegisteredStrategyBuildsTypedPlanFromRealGame` drives the canonical solver to each strategy using complete valid games from the maintained calibration corpus and catalog audit. The coverage contract rejects synthetic candidate-only boards, missing typed premises, invalid or absent narration on any step, concluding narration without exact digit and target tokens, unstable plan identities, incomplete effects, and any registered strategy without a placement or elimination plan.
