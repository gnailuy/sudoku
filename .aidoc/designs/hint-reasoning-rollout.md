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

# Hint Reasoning Rollout

This Draft defines the unimplemented family-specific reasoning and localized teaching work. The Active Hint Presentation Protocol owns current behavior; this proposal contains only future changes.

## Related Docs

| Document | Relationship |
|---|---|
| `.aidoc/designs/hint-presentation-protocol.md` | Canonical current hint-plan, reasoning, teaching, preview, and application contract |
| `.aidoc/designs/localization-contract.md` | Current descriptor, typed-argument, and English-fallback boundary |
| `.aidoc/designs/hint-reference-plans.md` | Current family examples and evidence ownership |
| `solver/evidence.go` | Current renderer-neutral units, premises, links, partitions, and eliminations |
| `game/hint_plan.go` | Current family composers and generic comparison paths |
| `api/openapi.yaml` | Current closed reasoning, narration, scene, and effect transport |

## Proposed Contract

Each remaining listed family replaces generic comparison with the smallest closed reason variant that expresses its proof. Hidden Pair–Quad have completed this migration through the Active Hint Presentation Protocol. Existing variants are reused when their facts are sufficient; a new variant is added only when reuse would require an overloaded flag, strategy-name inference, or prose parsing.

Every family change must:

1. distinguish the necessary observation, logical bridge, and exact action;
2. carry strategy-specific reason facts as backend-owned typed data;
3. provide one complete descriptor per step with bounded typed lists where grammar needs grouped values;
4. add one learner-visible fact or relation in each successive step;
5. place the plan's one atomic effect only on the final `apply` step; and
6. preserve deterministic preview, exact `plan_id`, renderer-neutral scenes, and client-owned visual treatment.

## Proposed Families

| Family | Proposed reasoning boundary |
|---|---|
| Pointing Pair and Box-Line Reduction | Identify the candidate's confinement in intersecting units and distinguish source and affected units without parsing the technique name. |
| Swordfish and Jellyfish | Use `fish-confinement` when their ordered base units, cover units, pattern candidates, and targets satisfy the existing variant. |
| XY-Wing, XYZ-Wing, and W-Wing | Carry pivot, wing, shared-candidate, and strong-link implications as complete typed relations. |
| BUG+1 and Unique Rectangle Types 1–4 | Express the exact uniqueness exception or relationship that forces the placement or elimination. |
| Simple Coloring and X-Cycles | Explain ordered strong/weak relationships, alternating partitions, conflict or visibility conditions, and the resulting action. |

If implementation evidence shows that a listed family does not need a new reason object, the change may instead complete its family descriptors using existing typed evidence. The acceptance requirement is explicit backend-owned reasoning, not one schema shape per display name.

## Delivery

Deliver one coherent reasoning family per coordinated backend and browser change:

1. replace the backend composer, reason schema, canonical English, fixtures, and focused engine/API/terminal acceptance together;
2. update generated browser fixtures, production locale catalogs, accessible rendering, and responsive screenshot acceptance in the companion change;
3. remove the generic path for that family rather than preserving compatibility aliases;
4. review learner-facing wording and scene progression before selecting another family; and
5. after all listed families are implemented, run one complete 23-strategy English and Chinese audit plus the maintained all-strategy matrix and nine-game real-stack journey.

Implement intersections first, then larger fish, relational wings, uniqueness, and coloring and cycles. Change that order only when concrete implementation evidence exposes a safer shared contract; do not combine unrelated reason families merely to reduce PR count.

## Validation

Each family change is accepted only when:

- every affected step has one complete valid descriptor with correctly typed arguments;
- no affected explanation uses `hint.strategy.compare` or reconstructs logic from fallback prose;
- consecutive scenes and statements add or focus a learner-visible fact or relation;
- grouped action descriptors round-trip the exact conclusion without client inference;
- unsupported or malformed reason variants fail validation and retain a complete safe fallback;
- applicable engine, HTTP, line-CLI, TUI, and browser acceptance remains green;
- UI tests capture and inspect desktop and phone screenshots for the changed family; and
- English, Simplified Chinese, and Traditional Chinese receive editorial review before that family is complete.

## Completion

When every proposed family is implemented and the cross-strategy audit passes, move any new durable invariants into the Active Hint Presentation Protocol, localization contract, and reference plans, then remove this Draft and its Draft Work entry.
