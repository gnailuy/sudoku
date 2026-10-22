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

This Draft tracks only the unimplemented migration from generic strategy comparison text to typed family reasoning and complete localized teaching. The Active Hint System remains the sole owner of shipped behavior; this proposal neither describes current behavior nor creates a second hint system.

## Related Docs

| Document | Relationship |
|---|---|
| `.aidoc/designs/hint-presentation-protocol.md` | Canonical current hint-plan, reasoning, teaching, preview, and application contract |
| `.aidoc/designs/localization-contract.md` | Current descriptor, typed-argument, and English-fallback boundary |
| `.aidoc/designs/hint-reference-plans.md` | Current family examples and evidence ownership |
| `solver/evidence.go` | Current renderer-neutral units, premises, links, partitions, and eliminations |
| `game/hint_plan.go` | Current family composers and remaining generic comparison fallback |
| `api/openapi.yaml` | Current closed reasoning, narration, scene, and effect transport |

## Implemented Baseline

The coordinated pilot has shipped for Naked Pair, X-Wing, and XY-Chain. The next backend batch extends the same `subset-reservation` reason to Naked Triple and Naked Quad. These strategies have complete family descriptors, typed list arguments, logic-sized `observe` / `explain` / `apply` sequences, and one exact final effect.

The Active Hint System owns those contracts. Naked Single, Hidden Single, value correction, and candidate correction already have focused teaching flows and do not require migration merely for structural uniformity.

Other strategy composers still rely on the generic `hint.strategy.compare` descriptor and, for eliminations, scalar conclusion fragments. Their scenes and typed evidence remain valid current behavior, but they do not yet carry the complete family-specific logical bridge needed for natural localized teaching.

## Remaining Contract

Each remaining family must replace generic comparison with the smallest closed reason variant that expresses its proof. Existing variants are reused when their facts are sufficient; a new variant is added only when reuse would require an overloaded flag, strategy-name inference, or prose parsing.

Every migrated plan must:

1. distinguish the necessary observation, logical bridge, and exact action;
2. carry strategy-specific reason facts as backend-owned typed data;
3. provide one complete descriptor per step with bounded typed lists where grammar needs grouped values;
4. add one learner-visible fact or relation in each successive step;
5. place the plan's one atomic effect only on the final `apply` step; and
6. preserve deterministic preview, exact `plan_id`, renderer-neutral scenes, and client-owned visual treatment.

The remaining family scopes are:

| Family | Unimplemented reasoning boundary |
|---|---|
| Hidden Pair–Quad | Explain that selected digits are confined to selected cells and group removal of other candidates from those cells. |
| Pointing Pair and Box-Line Reduction | Identify the candidate's confinement in intersecting units and distinguish source and affected units without parsing the technique name. |
| Swordfish and Jellyfish | Reuse `fish-confinement` when their ordered base units, cover units, pattern candidates, and targets satisfy the existing variant. |
| XY-Wing, XYZ-Wing, and W-Wing | Carry pivot, wing, shared-candidate, and strong-link implications as complete typed relations. |
| BUG+1 and Unique Rectangle Types 1–4 | Express the exact uniqueness exception or relationship that forces the placement or elimination. |
| Simple Coloring and X-Cycles | Explain ordered strong/weak relationships, alternating partitions, conflict or visibility conditions, and the resulting action. |

If implementation evidence shows that a listed family does not need a new reason object, the batch may instead complete its family descriptors using existing typed evidence. The acceptance requirement is explicit backend-owned reasoning, not one schema shape per display name.

## Delivery

Migrate one coherent reasoning family per coordinated backend and browser batch:

1. replace the backend composer, reason schema, canonical English, fixtures, and focused engine/API/terminal acceptance together;
2. update generated browser fixtures, production locale catalogs, accessible rendering, and responsive screenshot acceptance in the companion change;
3. remove the superseded generic path for that family rather than preserving compatibility aliases;
4. review learner-facing wording and scene progression before starting the next family; and
5. finish with one complete 23-strategy English and Chinese audit plus the maintained all-strategy matrix and nine-game real-stack journey.

The expected order is hidden subsets, intersections and larger fish, relational wings, uniqueness, then coloring and cycles. A batch may change that order when concrete implementation evidence exposes a safer shared contract, but must not combine unrelated reason families merely to reduce PR count.

## Validation

Each family batch is accepted only when:

- every migrated step has one complete valid descriptor with correctly typed arguments;
- no migrated explanation uses `hint.strategy.compare` or reconstructs logic from fallback prose;
- consecutive scenes and statements add or focus a learner-visible fact or relation;
- grouped action descriptors round-trip the exact conclusion without client inference;
- unsupported or malformed reason variants fail validation and retain a complete safe fallback;
- applicable engine, HTTP, line-CLI, TUI, and browser acceptance remains green;
- UI tests capture and inspect desktop and phone screenshots for the changed family; and
- English, Simplified Chinese, and Traditional Chinese receive editorial review before that family is considered complete.

## Completion

This Draft is complete when every remaining family has shipped and the final cross-strategy audit passes. At that point, merge any newly durable invariants into the Active Hint System, localization contract, and reference plans, remove this rollout document, and clear its Draft Work entry.