---
domain: Designs
status: Active
entry_points:
  - solver/move.go
  - game/contract.go
dependencies:
  - .aidoc/designs/hint-presentation-protocol.md
---

# Hint Reference Plans

The reference plans make the portable teaching protocol concrete for placement and elimination conclusions. Each plan preserves one machine-readable deduction while allowing every renderer to choose its own visual treatment.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/hint-presentation-protocol.md` | Canonical plan schema, ownership, application, and degradation rules |
| `.aidoc/designs/game-engine.md` | Authoritative hint query and action semantics |
| `solver/evidence.go` | Typed candidate and unit evidence consumed by every plan |
| `game/contract.go` | Engine hint preview and application boundary |

## Why Reference Plans Exist

Reference plans prevent strategy composers and renderers from interpreting semantic roles, ordered steps, or logical effects differently. Every reference includes `protocol_version=1`, a deterministic `plan_id`, registered strategy metadata, a complete summary, ordered steps, and one typed placement or elimination conclusion.

## Naked Single Placement

The Naked Single reference uses cell `r4c2`, whose candidate set is exactly `{7}`, and concludes with placement `r4c2=7`:

| Step | Kind | Complete scene | Effect | Fallback message |
|------|------|----------------|--------|------------------|
| `inspect-r4c2` | `observe` | focus cell `r4c2`; premise candidate `r4c2:7` | none | “Look at row 4, column 2. It has only one candidate: 7.” |
| `place-r4c2-7` | `conclude` | focus cell `r4c2`; conclusion candidate `r4c2:7` | place `7` at `r4c2` | “Because no other digit can go in row 4, column 2, place 7 there. This is called a Naked Single.” |

## Hidden Single Placement

The Hidden Single reference examines digit `7` in row 4, records that row constraints exclude every location except `r4c2`, and concludes with the same placement:

| Step | Kind | Complete scene | Effect | Fallback message |
|------|------|----------------|--------|------------------|
| `scan-row4-for-7` | `observe` | focus row 4; premise candidate `r4c2:7` | none | “In row 4, consider where 7 can appear.” |
| `compare-row4-7` | `compare` | focus row 4; eliminated candidate targets for every ruled-out empty cell; premise candidate `r4c2:7` | none | “Every other empty cell in row 4 cannot contain 7, leaving only row 4, column 2.” |
| `place-r4c2-7` | `conclude` | focus row 4 and cell `r4c2`; conclusion candidate `r4c2:7` | place `7` at `r4c2` | “Because row 4 still needs 7 and row 4, column 2 is the only possible cell, place 7 there. This is called a Hidden Single.” |

## Naked Pair Elimination

The Naked Pair reference examines row 4, where `r4c2` and `r4c7` each contain exactly `{2,7}`, and concludes only that candidate `2` must be removed from `r4c9`:

| Step | Kind | Complete scene | Effect | Fallback message |
|------|------|----------------|--------|------------------|
| `find-r4c2-r4c7` | `observe` | focus row 4; premise candidates `r4c2:{2,7}` and `r4c7:{2,7}` | none | “In row 4, the cells at row 4, column 2 and row 4, column 7 each have exactly the same two candidates: 2 and 7.” |
| `reserve-r4c2-r4c7` | `compare` | focus row 4; premise pair cells; focus candidate `r4c9:2` | none | “Those two cells must contain 2 and 7 in some order, so no other cell in row 4 can contain either digit.” |
| `remove-pair-candidates` | `eliminate` | premise pair cells; conclusion candidate `r4c9:2` | eliminate candidate `2` from `r4c9` | “Because those two cells reserve 2 and 7, remove candidate 2 from row 4, column 9; this deduction does not place a value. This pattern is called a Naked Pair.” |

Every renderer applies the same conclusion without parsing fallback prose or reconstructing logic from the technique name. The Naked Pair reference explicitly proves that a valid hint may change notes without placing a value. The remaining registered strategies use the same semantic roles and exact effects through the shared evidence composition defined in `.aidoc/designs/hint-presentation-protocol.md`; separate examples are unnecessary because they would duplicate the protocol rather than establish another invariant.
