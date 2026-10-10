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
---

# Hint Presentation Protocol

The hint presentation protocol turns one solver recommendation into a portable teaching plan that CLI, TUI, web, and future clients can render without reinterpreting Sudoku logic. The engine owns semantic evidence, ordered explanation, and the logical conclusion; each client owns visual style, interaction, and capability-aware rendering.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/architecture/guidelines.md` | Solver and package dependency boundaries |
| `.aidoc/designs/game-engine.md` | Authoritative hint query and action semantics |
| `.aidoc/designs/web-api.md` | Revisioned transport and client-neutral contract |
| `solver/move.go` | Current recommendation boundary to be replaced by typed evidence |
| `game/contract.go` | Engine hint preview and application boundary |

## Why a Shared Teaching Contract Exists

A position, value, technique name, and prose reason are enough to apply a hint but not enough to teach it consistently. A browser otherwise invents highlights, a TUI invents symbols, and a CLI invents ordering independently, which lets presentation layers disagree about the same deduction.

The protocol standardizes meaning rather than styling. Strategy and engine code describe what facts matter, what changed, and why the conclusion follows; renderers decide whether semantic roles become color, borders, motion, symbols, speech, or plain text.

The project is still in development, so the protocol replaces the current hint shape instead of preserving it through compatibility adapters. `protocol_version` protects future semantic evolution; it does not require support for the superseded position/value/reason contract.

## Ownership Boundaries

A strategy produces typed evidence for one deduction: examined units, candidate constraints, eliminations, placements, and the technique identity. Strategy evidence contains no colors, animation timing, widget commands, prose parsing requirements, or transport models.

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

## Determinism and Application

`plan_id` is derived from a canonical source-state digest, strategy ID, ordered evidence, and conclusion. Repeating a hint query against unchanged game state returns the same plan and IDs.

Applying a hint submits the previewed `plan_id`. The engine rejects a plan when the game state or recomputed conclusion no longer matches, preventing a client from teaching one deduction and applying another. The HTTP revision remains the transport concurrency guard; `plan_id` is the engine-level identity used by local and network clients alike.

Hint application records only the logical conclusion in game history. Teaching-step navigation is presentation state and never mutates the puzzle, dirty state, revision, or undo stack.

## Graceful Degradation and Accessibility

A fully capable renderer presents each scene and its message in order. A limited renderer presents the same ordered messages and conclusion while ignoring unsupported marks or target types. The simplest renderer may present `summary` and the conclusion, but must not invent omitted reasoning.

Step order, messages, roles, and explicit effects carry the meaning independently of color, position, or motion. Renderers expose the active step and step count to assistive technology, preserve user-controlled pacing, and avoid automatic animation that blocks reading or input.

Unsupported semantic data is ignored visibly rather than guessed. A renderer may report reduced presentation capability, but protocol consumption remains successful when fallback text and conclusion are supported.

## Reference Teaching Plans

The Naked Single reference uses cell `r4c2`, whose candidate set is exactly `{7}`, and concludes with placement `r4c2=7`:

| Step | Kind | Complete scene | Effect | Fallback message |
|------|------|----------------|--------|------------------|
| `inspect-r4c2` | `observe` | focus cell `r4c2`; premise candidate `r4c2:7` | none | “Cell r4c2 has only one candidate: 7.” |
| `place-r4c2-7` | `conclude` | focus cell `r4c2`; conclusion candidate `r4c2:7` | place `7` at `r4c2` | “Therefore r4c2 must be 7.” |

The Hidden Single reference examines digit `7` in row 4, records that row constraints exclude every location except `r4c2`, and concludes with the same placement:

| Step | Kind | Complete scene | Effect | Fallback message |
|------|------|----------------|--------|------------------|
| `scan-row4-for-7` | `observe` | focus row 4; premise candidate `r4c2:7` | none | “In row 4, consider where 7 can appear.” |
| `compare-row4-7` | `compare` | focus row 4; eliminated candidate targets for every ruled-out empty cell; premise candidate `r4c2:7` | none | “Every other empty cell in row 4 is ruled out for 7.” |
| `place-row4-7` | `conclude` | focus row 4 and cell `r4c2`; conclusion candidate `r4c2:7` | place `7` at `r4c2` | “Therefore r4c2 must be 7.” |

Each reference plan includes `protocol_version=1`, a deterministic `plan_id`, the registered strategy metadata, a complete summary, the listed ordered steps, and the placement conclusion. Every renderer applies the same machine-readable placement while choosing its own presentation.

## Delivery Boundary

The first implementation slice defines typed strategy evidence, the shared composer, deterministic IDs, Naked Single and Hidden Single plans, and engine contract tests. The slice replaces the old engine hint shape rather than maintaining parallel contracts.

The next backend slice replaces the OpenAPI hint schema and generated adapters, binds `apply-hint` to `plan_id`, and extends built-binary API acceptance. CLI and TUI then render the same plans before remaining strategies migrate; the separate web project consumes the published contract and proves the same plans at desktop and phone widths.

A strategy joins the protocol only with evidence tests, plan contract tests, and at least one renderer-neutral reference assertion. The migration is complete when no renderer parses `Reason`, switches on a technique name to reconstruct logic, or owns Sudoku-specific teaching order.
