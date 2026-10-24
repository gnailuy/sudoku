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

The reference plans make the portable teaching protocol within the canonical hint system concrete for placement and elimination conclusions. Each plan preserves one machine-readable deduction while allowing every renderer to choose its own visual treatment.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/hint-presentation-protocol.md` | Canonical plan schema, ownership, application, and degradation rules |
| `.aidoc/designs/game-engine.md` | Authoritative hint query and action semantics |
| `solver/evidence.go` | Typed candidate and unit evidence consumed by every plan |
| `game/contract.go` | Engine hint preview and application boundary |

## Why Reference Plans Exist

Reference plans prevent strategy composers and renderers from interpreting semantic roles, ordered steps, or logical effects differently. Every reference includes a deterministic `plan_id`, registered strategy metadata, a complete semantic summary descriptor, ordered descriptor-backed steps, and one typed placement or elimination conclusion.

## Naked Single Placement

The Naked Single reference uses cell `r4c2`, whose candidate set is exactly `{7}`, and concludes with placement `r4c2=7`:

| Step | Kind | Complete scene | Effect | Canonical English fallback |
|------|------|----------------|--------|------------------|
| `inspect-r4c2` | `observe` | focus cell `r4c2`; premise candidate `r4c2:7` | none | “Look at row 4, column 2. It has only one candidate: 7.” |
| `place-r4c2-7` | `apply` | focus cell `r4c2`; conclusion candidate `r4c2:7` | place `7` at `r4c2` | “Because no other digit can go in row 4, column 2, place 7 there. This is called a Naked Single.” |

## Hidden Single Placement

The Hidden Single reference examines digit `7` in row 4, records that row constraints exclude every location except `r4c2`, and concludes with the same placement:

| Step | Kind | Complete scene | Effect | Canonical English fallback |
|------|------|----------------|--------|------------------|
| `scan-row4-for-7` | `observe` | focus row 4; premise candidate `r4c2:7` | none | “In row 4, consider where 7 can appear.” |
| `explain-row4-7` | `explain` | focus row 4; eliminated candidate targets for every ruled-out empty cell; premise candidate `r4c2:7` | none | “Every other empty cell in row 4 cannot contain 7, leaving only row 4, column 2.” |
| `place-r4c2-7` | `apply` | focus row 4 and cell `r4c2`; conclusion candidate `r4c2:7` | place `7` at `r4c2` | “Because row 4 still needs 7 and row 4, column 2 is the only possible cell, place 7 there. This is called a Hidden Single.” |

## Naked Pair Elimination

The Naked Pair reference examines row 4, where `r4c2` and `r4c7` each contain exactly `{2,7}`, carries a `subset-reservation` reason for those cells and digits, and concludes only that candidate `2` must be removed from `r4c9`:

| Step | Kind | Complete scene | Effect | Canonical English fallback |
|------|------|----------------|--------|------------------|
| `find-r4c2-r4c7` | `observe` | focus row 4; premise candidates `r4c2:{2,7}` and `r4c7:{2,7}` | none | “In row 4, the cells at row 4, column 2 and row 4, column 7 each have exactly the same two candidates: 2 and 7.” |
| `reserve-r4c2-r4c7` | `explain` | focus row 4; premise pair cells; focus candidate `r4c9:2` | none | “Those two cells must contain 2 and 7 in some order, so no other cell in row 4 can contain either digit.” |
| `remove-pair-candidates` | `apply` | premise pair cells; conclusion candidate `r4c9:2` | eliminate candidate `2` from `r4c9` | “Because those two cells reserve 2 and 7, remove candidate 2 from row 4, column 9; this deduction does not place a value. This pattern is called a Naked Pair.” |

Every renderer applies the same conclusion from typed effects without parsing descriptor fallback prose or reconstructing logic from the technique name. The Naked Pair reference explicitly proves that a valid hint may change notes without placing a value.

## Subset Family

Naked Pair–Quad plans first identify the exact cells and their combined candidate set, then explain that the cells reserve the same number of digits before applying the placement or elimination. Their `subset-reservation` reason records the complete digit union, premise cells, unit, and exact action targets; it does not keep only digits shared by every premise cell. Hidden Pair–Quad plans invert the relationship: the selected digits appear only in the highlighted cells, so other candidates in those cells can be removed. Their `hidden-subset-confinement` reason records the complete confined-digit set, exact cells, unit, and action targets. The observation introduces the confinement, the explanation states why no other candidates can remain in those cells, and the final descriptor groups the exact removal or resulting placement. `solver.Evidence.Premises` and `solver.Evidence.Unit` carry these facts without prose parsing.

## Box/Line Intersections

Pointing Pair and Box-Line Reduction plans identify one candidate, its exact premise cells, and both intersecting units. A box-first source names Pointing Pair because the candidate is confined to one row or column inside the box; a row- or column-first source names Box-Line Reduction because the candidate is confined to one box. `solver.Evidence.Unit` identifies the source direction, while `solver.Evidence.Units` preserves both units.

## Fish Family

X-Wing, Swordfish, and Jellyfish plans identify one candidate, two to four base lines, the same number of cross lines, and every highlighted intersection where that candidate remains. The explanation shows that confinement across the base lines removes the candidate from other cells on the cross lines. Every fish size carries a `fish-confinement` reason with the digit, ordered base and cover units, pattern candidates, and exact action targets. Its `observe` step shows the pattern candidates and base lines, its `explain` step adds the cover-line relationship, and its `apply` step isolates only the removable candidates while retaining enough pattern context to justify them. `solver.FishSolver` preserves the ordered units and exact premise candidates, so the composer never infers orientation from prose or strategy names.

## Relational Wing Family

XY-Wing and XYZ-Wing plans identify one pivot and two wings with their exact candidate sets before explaining the shared-candidate implication. W-Wing plans use four logic-sized stages: identify the two equal bivalue cells; show that the bridge digit has exactly two positions in one unit; explain how either bridge position forces the other digit into its corresponding bivalue cell; then remove that digit from cells that see both. Formal wing and strong-link vocabulary follows the plain causal chain rather than replacing it. `solver.XYWingSolver`, `solver.XYZWingSolver`, and `solver.WWingSolver` preserve the role order in typed premises, allowing the shared composer to name plain row-and-column locations without parsing solver reasons.

## Uniqueness Family

BUG+1 plans identify the only unsolved cell with three candidates after establishing that every other unsolved cell is bivalue. The comparison identifies the extra candidate whose removal would leave interchangeable solutions, and the conclusion places that candidate before naming BUG+1.

Unique Rectangle Types 1–4 first identify four corners spanning two rows, two columns, and two boxes plus the candidate pair common to every corner. The comparison then distinguishes the exact typed relationship: one expanded corner for Type 1, a shared extra candidate for Type 2, a unit-scoped subset for Type 3, or a conjugate pair for Type 4. `solver.Evidence.Premises`, `solver.Evidence.Unit`, and exact eliminations preserve those facts without parsing solver reasons.

## Coloring and Chain Families

Simple Coloring plans begin with one concrete forced pair in a named row, column, or box, then expand the complete typed A/B partitions and state the two possible assignments. Each elimination receives its own step naming the target and one visible witness from each group. The final step applies the complete atomic effect and names Simple Coloring; the earlier steps require no prior knowledge of strong links, coloring, or seeing both colors.

X-Cycles plans preserve the ordered single-candidate loop and classify each typed link as must-contain or cannot-both. Their current generic comparison and conclusion remain in place until the separate strategy progression is approved; the renderer still receives the complete ordered scene and never reconstructs it from diagnostic prose.

XY-Chain plans preserve every ordered bivalue cell plus the candidate shared by each neighboring pair in an `implication-chain` reason. The `observe` step introduces the first cell and its alternatives. Successive `explain` steps add exactly one typed implication and one cumulative relation to the complete scene, so sequence length follows the represented chain. The final explanation establishes why one endpoint must contain the common digit; the `apply` step preserves the full chain while isolating removal of that digit from cells that see both endpoints. Accessible narration communicates the same implication order as the links; arrows alone are never the explanation.
