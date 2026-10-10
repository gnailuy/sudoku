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
| `solver/move.go` | Current recommendation boundary to be replaced by typed evidence |
| `game/contract.go` | Engine hint preview and application boundary |

## Why Reference Plans Exist

Reference plans prevent strategy composers and renderers from interpreting semantic roles, ordered steps, or logical effects differently. Every reference includes `protocol_version=1`, a deterministic `plan_id`, registered strategy metadata, a complete summary, ordered steps, and one typed placement or elimination conclusion.

## Naked Single Placement

The Naked Single reference uses cell `r4c2`, whose candidate set is exactly `{7}`, and concludes with placement `r4c2=7`:

| Step | Kind | Complete scene | Effect | Fallback message |
|------|------|----------------|--------|------------------|
| `inspect-r4c2` | `observe` | focus cell `r4c2`; premise candidate `r4c2:7` | none | “Cell r4c2 has only one candidate: 7.” |
| `place-r4c2-7` | `conclude` | focus cell `r4c2`; conclusion candidate `r4c2:7` | place `7` at `r4c2` | “Therefore r4c2 must be 7.” |

## Hidden Single Placement

The Hidden Single reference examines digit `7` in row 4, records that row constraints exclude every location except `r4c2`, and concludes with the same placement:

| Step | Kind | Complete scene | Effect | Fallback message |
|------|------|----------------|--------|------------------|
| `scan-row4-for-7` | `observe` | focus row 4; premise candidate `r4c2:7` | none | “In row 4, consider where 7 can appear.” |
| `compare-row4-7` | `compare` | focus row 4; eliminated candidate targets for every ruled-out empty cell; premise candidate `r4c2:7` | none | “Every other empty cell in row 4 is ruled out for 7.” |
| `place-row4-7` | `conclude` | focus row 4 and cell `r4c2`; conclusion candidate `r4c2:7` | place `7` at `r4c2` | “Therefore r4c2 must be 7.” |

## Naked Pair Elimination

The Naked Pair reference examines row 4, where `r4c2` and `r4c7` each contain exactly `{2,7}`, and concludes only that candidate `2` must be removed from `r4c9`:

| Step | Kind | Complete scene | Effect | Fallback message |
|------|------|----------------|--------|------------------|
| `find-row4-pair` | `observe` | focus row 4; premise candidates `r4c2:{2,7}` and `r4c7:{2,7}` | none | “In row 4, r4c2 and r4c7 contain the same two candidates: 2 and 7.” |
| `reserve-row4-2-7` | `compare` | focus row 4; premise pair cells; focus candidate `r4c9:2` | none | “Those two digits must occupy the pair cells, so neither can appear elsewhere in row 4.” |
| `remove-r4c9-2` | `eliminate` | premise pair cells; conclusion candidate `r4c9:2` | eliminate candidate `2` from `r4c9` | “Remove candidate 2 from r4c9; this deduction does not place a value.” |

Every renderer applies the same conclusion without parsing fallback prose or reconstructing logic from the technique name. The Naked Pair reference explicitly proves that a valid hint may change notes without placing a value.
