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

The maintained Naked Single reference uses cell `r2c1`, whose candidate set is exactly `{8}`, and concludes with placement `r2c1=8`:

| Step | Kind | Complete scene | Effect | Canonical English fallback |
|------|------|----------------|--------|------------------|
| `inspect-r2c1-constraints` | `observe` | focus `r2c1`, row 2, column 1, and box 1; mark one placed peer for each excluded digit 1–7 and 9 | none | “Start with row 2, column 1. Its row, column, and box already contain 1, 2, 3, 4, 5, 6, 7, and 9.” |
| `identify-r2c1-candidate` | `explain` | preserve the constraint witnesses; mark candidate `r2c1:8` as the sole premise | none | “Therefore, 8 is the only digit that can go in row 2, column 1.” |
| `place-r2c1-8` | `apply` | preserve the witnesses; change candidate `r2c1:8` from premise to conclusion | place `8` at `r2c1` | “Place 8 in row 2, column 1. A cell with only one candidate is a Naked Single.” |

## Hidden Single Placement

The maintained Hidden Single reference examines digit `9` in row 4, records the placed value that excludes every other empty location, and concludes with placement `r4c4=9`:

| Step | Kind | Complete scene | Effect | Canonical English fallback |
|------|------|----------------|--------|------------------|
| `scan-row4-for-9` | `observe` | focus row 4 and premise candidate `r4c4:9` | none | “In row 4, check where 9 can go.” |
| `rule-out-row4-9` | `explain` | preserve row 4 and `r4c4:9`; show placed blockers `r7c1`, `r6c2`, `r1c3`, `r9c6`, `r2c7`, and `r5c8`; cross candidate 9 at `r4c1`, `r4c2`, `r4c3`, `r4c6`, `r4c7`, and `r4c9` | none | “The placed 9s at row 7, column 1; row 6, column 2; row 1, column 3; row 9, column 6; row 2, column 7; and row 5, column 8 rule out 9 from every other empty cell in row 4.” |
| `identify-r4c4-9` | `explain` | preserve all blockers and crossed slots; retain premise candidate `r4c4:9` as the sole location | none | “Only row 4, column 4 can still contain 9.” |
| `place-r4c4-9` | `apply` | preserve the complete proof; change candidate `r4c4:9` from premise to conclusion | place `9` at `r4c4` | “Place 9 in row 4, column 4. When a digit has only one possible cell in a row, column, or box, it is a Hidden Single.” |

## Naked Pair Elimination

The maintained Naked Pair reference examines column 2, where `r4c2` and `r7c2` each contain exactly `{1,3}`, carries a `subset-reservation` reason for those cells and digits, and concludes only that candidate `1` must be removed from `r3c2`:

| Step | Kind | Complete scene | Effect | Canonical English fallback |
|------|------|----------------|--------|------------------|
| `inspect-r4c2` | `observe` | focus column 2 and premise candidates `r4c2:{1,3}` | none | “Start in column 2: row 4, column 2 can contain only 1 or 3.” |
| `match-r7c2` | `explain` | preserve `r4c2:{1,3}` and add premise candidates `r7c2:{1,3}` | none | “The matching cell, row 7, column 2, has the same two candidates: 1 and 3.” |
| `reserve-r4c2-r7c2` | `explain` | preserve both pair cells and column 2; identify candidate `r3c2:1` as the exact consequence | none | “These two cells must take 1 and 3 in some order, so candidate 1 cannot stay in any other cell in column 2.” |
| `apply-naked-pair` | `apply` | preserve the pair premise and column scope; mark candidate `r3c2:1` as the conclusion | eliminate candidate `1` from `r3c2` | “Remove candidate 1 from row 3, column 2; this step does not place a value. This pattern is a Naked Pair.” |

Every renderer applies the same conclusion from typed effects without parsing descriptor fallback prose or reconstructing logic from the technique name. The Naked Pair reference explicitly proves that a valid hint may change candidates without placing a value.

## Subset Family

Naked Pair uses the four-stage maintained progression above. Naked Triple and Naked Quad first identify the exact cells and their combined candidate set, then explain that the cells reserve the same number of digits before applying the placement or elimination. Every Naked Pair–Quad `subset-reservation` reason records the complete digit union, premise cells, unit, and exact action targets; it does not keep only digits shared by every premise cell. Hidden Pair–Quad plans invert the relationship: the selected digits appear only in the highlighted cells, so other candidates in those cells can be removed. Their `hidden-subset-confinement` reason records the complete confined-digit set, exact cells, unit, and action targets. The observation introduces the confinement, the explanation states why no other candidates can remain in those cells, and the final descriptor groups the exact removal or resulting placement. `solver.Evidence.Premises` and `solver.Evidence.Unit` carry these facts without prose parsing.

## Box/Line Intersections

Pointing Pair and Box-Line Reduction plans identify one candidate, its exact premise cells, and both intersecting units. A box-first source names Pointing Pair because the candidate is confined to one row or column inside the box; a row- or column-first source names Box-Line Reduction because the candidate is confined to one box. `solver.Evidence.Unit` identifies the source direction, while `solver.Evidence.Units` preserves both units.

## Fish Family

X-Wing, Swordfish, and Jellyfish plans identify one candidate, two to four base lines, the same number of cross lines, and every highlighted intersection where that candidate remains. The explanation shows that confinement across the base lines removes the candidate from other cells on the cross lines. Every fish size carries a `fish-confinement` reason with the digit, ordered base and cover units, pattern candidates, and exact action targets. Its `observe` step shows the pattern candidates and base lines, its `explain` step adds the cover-line relationship, and its `apply` step isolates only the removable candidates while retaining enough pattern context to justify them. `solver.FishSolver` preserves the ordered units and exact premise candidates, so the composer never infers orientation from prose or strategy names.

## Relational Wing Family

XY-Wing plans use five stages: identify the pivot; follow each pivot value to the wing that must contain the shared digit; combine the cases; then show that the exact target sees the shared candidate in both wings before applying one result. XYZ-Wing plans use six stages: identify the three-candidate pivot; follow each non-shared pivot value to the wing forced to the shared digit; cover the pivot-equals-shared-digit case; combine all three cases; then show that the exact target shares a unit with the pivot and both wings before applying one result. W-Wing plans use four logic-sized stages: identify the two equal bivalue cells; show that the bridge digit has exactly two positions in one unit; explain how either bridge position forces the other digit into its corresponding bivalue cell; then remove that digit from cells that see both. Formal wing and strong-link vocabulary follows the plain causal chain rather than replacing it. `solver.XYWingSolver`, `solver.XYZWingSolver`, and `solver.WWingSolver` preserve the role order in typed premises, allowing the shared composer to name plain row-and-column locations without parsing solver reasons.

## Uniqueness Family

BUG+1 plans identify the only unsolved cell with three candidates after establishing that every other unsolved cell is bivalue. The comparison identifies the extra candidate whose removal would leave interchangeable solutions, and the conclusion places that candidate before naming BUG+1.

Unique Rectangle Types 1–4 first identify four corners spanning two rows, two columns, and two boxes plus the candidate pair common to every corner. The comparison then distinguishes the exact typed relationship: one expanded corner for Type 1, a shared extra candidate for Type 2, a unit-scoped subset for Type 3, or a conjugate pair for Type 4. `solver.Evidence.Premises`, `solver.Evidence.Unit`, and exact eliminations preserve those facts without parsing solver reasons.

## Coloring and Chain Families

Simple Coloring plans begin with one concrete forced pair in a named row, column, or box, then expand the complete typed A/B partitions and state the two possible assignments. Each elimination receives its own step naming the target and one visible witness from each group. The final step applies the complete atomic effect and names Simple Coloring; the earlier steps require no prior knowledge of strong links, coloring, or seeing both colors.

X-Cycles plans begin with the opening strong pair, then add one weak/strong consequence pair at a time while preserving every earlier node and link. The maintained placement example follows the assumption through rows 8 and 2, closes the loop in the middle-right box, and applies exactly one value only after the starting assumption contradicts itself. Solid strong links and dashed weak links remain typed renderer-neutral evidence; each of the five complete descriptors names the cells, digit, and unit needed for its own inference.

XY-Chain plans preserve every ordered bivalue cell plus the candidate shared by each neighboring pair in an `implication-chain` reason. The maintained eight-cell placement example uses six learner-sized stages: introduce the first endpoint, carry two implications across row 1, carry two into row 2, carry two more across row 2, reach the second endpoint, and place the target value after proving one endpoint must contain their common digit. The cumulative link counts are 0, 2, 4, 6, 7, and 7. The final step preserves the complete chain while isolating only the exact placement; accessible descriptors communicate the same implication order as the arrows.
