---
domain: Designs
status: Draft
entry_points:
  - solver/evidence.go
  - game/hint_plan.go
  - message/descriptor.go
  - api/openapi.yaml
dependencies:
  - .aidoc/designs/hint-presentation-protocol.md
  - .aidoc/designs/localization-contract.md
  - .aidoc/designs/hint-reference-plans.md
---

# Beginner Hint Reasoning

This proposal makes each strategy hint explain what to notice, why the pattern forces a result, and what to change. The backend remains authoritative for Sudoku reasoning, while every client receives enough typed meaning to render natural localized teaching without parsing English or reconstructing a strategy.

## Related Docs

| Document | Relationship |
|---|---|
| `.aidoc/designs/hint-presentation-protocol.md` | Current authoritative hint-plan, scene, and exact-application contract |
| `.aidoc/designs/localization-contract.md` | Current descriptor, typed-argument, and browser-localization boundary |
| `.aidoc/designs/hint-reference-plans.md` | Current family examples and evidence ownership |
| `solver/evidence.go` | Current renderer-neutral units, premises, links, partitions, and eliminations |
| `game/hint_plan.go` | Current family composers, generic comparison wording, and per-target conclusion narration |
| `api/openapi.yaml` | Current closed narration, target, scene, and effect transport |

## Problem

The current plan carries enough evidence to apply and visualize a deduction, but most strategy composers do not carry enough explicit teaching semantics to explain it. `populateStrategyStep` gives every non-specialized middle step the same `hint.strategy.compare` descriptor, so a learner is told to identify a forced result without being told which relationship makes it forced. Fish and subset composers may also repeat the same scene between observation and comparison, leaving no visible or verbal progression.

Conclusion effects are machine-readable, but generic narration serializes eliminations as repeated digit/from/cell fragments. That representation is exact but does not express one grouped action such as “remove candidate 7 from these cells,” and it forces awkward repetition in languages that should place the digit or target list elsewhere.

Copy-only changes cannot close these gaps. The missing relationship must be represented before the canonical English catalog or browser catalogs can describe it faithfully.

## Goals

The replacement contract must:

1. make the observation, logical bridge, and action distinct and complete;
2. carry the strategy-specific reason as typed data owned by the backend;
3. let a locale reorder and format complete target, unit, candidate, and digit lists naturally;
4. make consecutive scenes advance when a new relation is introduced;
5. preserve deterministic preview, exact `plan_id`, atomic application, and renderer-neutral styling; and
6. support bounded family-by-family delivery without leaving two production contracts.

This work does not add server-side locale negotiation, move Sudoku inference into a client, change strategy selection, or change the placement and elimination effects that the engine applies.

## Proposed Contract

A strategy `HintPlan` gains one required `reason` object. `reason.kind` is a closed family-level union, not a technique name or prose label. Each variant carries only facts needed to establish the logical bridge; existing scenes continue to carry visual emphasis, and the existing conclusion remains the sole applicable effect.

The initial union has three variants:

| `reason.kind` | Required facts | Pilot strategy |
|---|---|---|
| `subset-reservation` | one unit, reserved digits, premise cells, and exact action targets | Naked Pair |
| `fish-confinement` | one digit, ordered base units, ordered cover units, pattern candidates, and exact action targets | X-Wing |
| `implication-chain` | ordered bivalue groups, typed links, endpoint digit, and exact action targets | XY-Chain |

The full rollout adds variants only when a reasoning family needs facts that cannot be represented by an existing variant. A variant may serve several techniques—for example, the fish variant serves X-Wing, Swordfish, and Jellyfish—so the wire contract follows teaching logic rather than the 23 display names.

Every strategy plan uses exactly three semantic stages:

| Stage | Required answer | Scene responsibility |
|---|---|---|
| `observe` | What should the player notice? | Show the bounded premise without the conclusion |
| `explain` | Why does that premise force a result? | Add the relation that connects the premise to the conclusion |
| `apply` | What exactly should change? | Preserve necessary context and isolate the placement or grouped eliminations |

`observe`, `explain`, and `apply` replace the strategy meanings of the current generic `observe`, `compare`, and `eliminate`/`conclude` sequence in one coordinated cutover. Correction plans may retain their focused correction flow because they repair inconsistent player state rather than teach a Sudoku strategy.

Each stage carries one complete `MessageDescriptor`. The descriptor argument union gains bounded list variants for `digits`, `cells`, `candidates`, and `units`. List values are typed arrays with the same coordinate and size limits as their scalar forms; they are not comma-joined text. A locale therefore receives one complete message ID plus named semantic arguments and may reorder, pluralize, and format the whole sentence without replaying backend token order.

Structured narration remains available for accessible target references and limited renderers, but it no longer assembles a conclusion from repeated scalar fragments. The stage descriptor is the canonical translatable statement; narration targets identify interactive references within that statement. The existing English fallback remains complete and deterministic.

## Pilot Teaching Plans

### X-Wing

The `fish-confinement` reason records candidate 7, base rows 4 and 8, cover columns 2 and 6, the four corner candidates, and the elimination targets.

1. `observe`: “In rows 4 and 8, candidate 7 appears only in columns 2 and 6.” The scene shows the four corners and the two base rows.
2. `explain`: “Whichever corner contains 7 in row 4, row 8 must use the other column, so columns 2 and 6 each receive a 7 in one of these rows.” The scene adds the two cover columns and a bounded rectangle or equivalent relation cue.
3. `apply`: “Remove candidate 7 from the other cells in columns 2 and 6: [grouped cells].” The scene isolates only the removable candidates while retaining enough pattern context to justify them.

The renderer chooses borders, lines, motion, speech, or text; it does not infer row-versus-column orientation from the strategy name.

### Naked Pair

The `subset-reservation` reason records the unit, the two cells, their shared two digits, and every elimination target.

1. `observe` identifies the two cells and their exact shared candidates.
2. `explain` states that two cells reserve those two digits in either order, so no other cell in the unit may contain either digit.
3. `apply` groups eliminations by action and digit instead of repeating one sentence fragment per target.

The same variant supports larger naked subsets and hidden subsets only if its facts remain sufficient; otherwise hidden confinement receives a separate family variant rather than an overloaded flag.

### XY-Chain

The `implication-chain` reason records ordered bivalue groups, typed implication links, the common endpoint digit, and cells that see both endpoints.

1. `observe` introduces the first bounded implication and its two bivalue cells.
2. `explain` walks the complete ordered implication chain and states why one endpoint must contain the common digit.
3. `apply` groups removal of that digit from every cell seeing both endpoints while preserving the complete chain scene.

The accessible statement must communicate the same implication order as the visual links; arrows alone are never the explanation.

## Validation

The pilot is accepted only when all three strategies prove the following at engine, HTTP, terminal, and browser boundaries:

- every stage has one valid family reason and one complete descriptor with correctly typed arguments;
- the explanation stage is strategy-specific and does not use the generic forced-result descriptor;
- the observation and explanation scenes differ meaningfully unless a fixture records why the same scene is sufficient;
- grouped action descriptors round-trip exact effects without parsing fallback text;
- English, Simplified Chinese, and Traditional Chinese receive editorial review before the family contract expands;
- desktop and phone browser tests capture screenshots for each pilot family and inspect both visual progression and readable text;
- the 23-strategy fixture matrix and nine-game full-stack hint journey remain green; and
- unsupported or malformed reason variants fail validation and use the complete safe fallback rather than inviting client inference.

The backend exports API-shaped pilot fixtures for the coordinated browser PR. The browser proves translated list formatting, accessible target references, scene progression, and exact application against those fixtures and the real Go service.

## Delivery

1. Replace the backend contract for X-Wing, Naked Pair, and XY-Chain; update OpenAPI, generated adapters, canonical English, terminal rendering, fixtures, and acceptance together.
2. Replace the browser contract for the same three strategies; update generated types, English and Chinese catalogs, accessible rendering, and responsive screenshot acceptance together.
3. Review the learner-facing pilot before adding more reason variants.
4. Roll the accepted model through subset, intersection, fish, wing, uniqueness, coloring, cycle, chain, direct-placement, and correction families in reviewable batches.
5. Finish with one complete 23-strategy English and Chinese audit plus the existing all-strategy and nine-game real-stack suites.

Each implementation batch removes the superseded generic path for its migrated family. The project is development-only, so it does not add compatibility aliases, parallel schema versions, or adapters that preserve an unimplemented intermediate contract.

## Promotion

This Draft becomes Active only after the coordinated backend and browser pilot ships and the three pilot strategies satisfy the validation above. At promotion, merge the resulting current-state contract into `hint-presentation-protocol.md`, `localization-contract.md`, and `hint-reference-plans.md`; retain this document only if it remains the clearest canonical owner, otherwise remove it after preserving all durable rationale.
