---
name: sudoku-strategic-hint-quality
description: "Design or audit Sudoku strategic hints; run one approval-gated teaching workflow. Excludes unrelated game, solver, or UI work."
---

# Sudoku Strategic Hint Quality

1. Classify one bounded request.
   - Classify it as `design`, `audit`, `update`, or `new-strategy`.
   - Read `AGENT.md`, follow the hint and localization routes in `.aidoc/INDEX.md`, and follow the browser repository's declared localization and E2E routes when browser evidence is in scope.
   - Record one strategy, the current review boundary, explicitly deferred work, and the canonical documents that own the applicable product rules.
   - Stop when the requested strategy, repository pair, maintained example, or outcome is ambiguous.
   - **Done when:** one strategy and one review boundary are recorded with their canonical owners.

2. Build one structured quality report.
   - Copy `assets/quality-report.md` and fill it from the canonical documents, current code, maintained fixtures, and repository-owned acceptance routes.
   - Reproduce the authoritative example on the current target revisions and record the complete source state, exact conclusion, teaching sequence, exact locale copy, and evidence artifacts required by the owning documents.
   - Keep current approval and temporary project status in the report, not in `.aidoc/` or this skill.
   - Stop rather than substituting a synthetic example, guessed project state, or remembered command for a missing canonical source.
   - **Done when:** another reviewer can verify every report field from a named owner or artifact.

3. Obtain the required independent evaluations.
   - Give each reviewer the locale-neutral teaching intent, exact locale sequence, typed scene/effect evidence, and the canonical acceptance contract without supplying a desired verdict.
   - Record editorial findings and any fresh-context navigation result in the report. Preserve quoted concrete findings when a reviewer rejects copy or routing.
   - On failure, return to the owning source and revise the complete affected report section; do not weaken or restate the canonical rule inside the skill.
   - **Done when:** every required independent evaluation is recorded as passed, failed, pending, or blocked with evidence.

4. Enforce the design approval stop.
   - For `design` and `new-strategy`, return the self-contained report and exactly one approval question before changing production behavior.
   - For `audit`, return `pass`, `revise`, or `block`; change production behavior only when the request also authorizes an update.
   - **Done when:** the design decision is explicit and unapproved implementation has not started.

5. Implement only the approved report.
   - Follow the repository routes named by the report for backend semantics, browser rendering and localization, documentation ownership, and black-box acceptance.
   - Keep the change limited to the approved strategy. Update a canonical document only when its durable contract changes; keep report state, approval history, and PR status out of durable docs.
   - **Done when:** code, fixtures, catalogs, and any changed canonical document match the approved report.

6. Run the owners' validation and collect evidence.
   - Run the exact project commands and scenario routes named by `AGENT.md`, `.aidoc/INDEX.md`, the affected canonical documents, and the browser repository.
   - Record mechanical, semantic, editorial, visual, fresh-context, and CI outcomes separately. Do not treat one category as proof of another.
   - Inspect generated visual artifacts when the owning acceptance route requires them, and identify every skipped or unavailable check.
   - **Done when:** each required category passes or has one exact blocker.

7. Return one decision and stop at the boundary.
   - Report `awaiting-design-approval`, `awaiting-implementation-review`, `complete`, `revise`, or `block` with the filled quality report and exactly one next approval or action.
   - After an authorized merge, follow the repository PR workflow for final-head approval, metadata, mergeability, checks, synchronized defaults, and post-merge proof.
   - Do not start the next strategy or a deferred improvement.
   - **Done when:** the output names one status, evidence by category, blockers, and one next action.

## Routing Check

Use `references/routing-fixtures.md` to test the trigger boundary. The fixtures test discovery only; they do not prove domain correctness or replace a fresh-context evaluation.
