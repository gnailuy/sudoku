---
name: sudoku-strategic-hint-quality
description: "Design, audit, update, or add Sudoku strategic hints; produce an approval-gated teaching package and implementation evidence. Excludes unrelated game or UI work."
---

# Sudoku Strategic Hint Quality

1. Establish the task boundary and current state.
   - Classify the request as `design`, `audit`, `update`, or `new-strategy`.
   - Read `AGENT.md`, then follow the hint and localization routes in `.aidoc/INDEX.md`; in the browser repository, follow its localization and E2E routes.
   - Record the one strategy, exact maintained example, current approval boundary, and explicitly deferred work. Do not infer completion from family-level work or start an adjacent strategy.
   - Stop if the strategy, authoritative example, or requested outcome is ambiguous.
   - **Done when:** one strategy and one review boundary are written down with canonical owners.

2. Reconstruct the authoritative deduction.
   - Reach the maintained example through the repository-owned fixture or real-game path on current default branches.
   - Record the complete reached board, authoritative candidate grid, typed premises, exact target, and exactly one supported placement or grouped elimination.
   - Show the automatic candidate grid from Step 1. Use the actual product board as primary evidence; use an abstract diagram only as secondary explanation.
   - Reject synthetic candidate-only evidence when the maintained real example is required.
   - **Done when:** the source state reproduces deterministically and the proposed conclusion matches the engine effect without extra deductions.

3. Build a locale-neutral teaching intent before writing copy.
   - For each step, state: what the learner notices, what new inference becomes possible, what prior evidence remains visible, and which typed visual facts carry the meaning.
   - Use as many steps as the logic requires. Add one causal idea or relation per step; merge or delete a step that merely repeats the prior scene.
   - Keep scenes cumulative where the proof depends on earlier facts. Introduce formal strategy vocabulary only in the final applying step.
   - Use links only for real strong, weak, or implication relationships; do not add decorative arrows to unit confinement or subset evidence.
   - Keep color supplementary by preserving coordinates, symbols, line styles, labels, text, and accessible descriptions.
   - **Done when:** a reader can follow an uninterrupted premise-to-conclusion chain without knowing the strategy name.

4. Author and review language independently.
   - Write complete English, Simplified Chinese, and Traditional Chinese sequences from the locale-neutral intent, not by transforming English syntax line by line.
   - In Chinese, describe row, column, box, confinement, and candidate consequences directly; do not translate English Sudoku metaphors such as “sees” literally.
   - Read each locale as a complete sequence before comparing it with the typed intent, scene, and action. Check glossary consistency, coordinate wording, clause order, redundant quantifiers, calques, and unnatural nominal phrases.
   - Run a separate fresh-context naturalness review for Simplified and Traditional Chinese. The reviewer may reject the copy and must quote concrete calques, English-shaped order, or unnatural terminology. Screenshots and catalog tests are not linguistic evidence.
   - If review fails, discard and rewrite the complete affected sequence from the intent brief; do not patch one sentence or self-certify an “independent rewrite.”
   - **Done when:** mechanical parity and independent editorial evidence are both recorded, or the package is explicitly blocked as not review-ready.

5. Produce one self-contained design package.
   - Use the template in `assets/quality-report.md` and the stop gates in `references/gates.md`.
   - Include the reached-board identity, exact candidates and action, step-by-step locale-neutral intent, exact three-locale copy, one actual-board storyboard image per step, scene mark/link progression, responsive expectations, and implementation acceptance plan.
   - Keep the package understandable without reconstructing earlier messages.
   - Stop before implementation for `design` and `new-strategy` work until the design is explicitly approved.
   - **Done when:** the package contains every decision the reviewer needs and names one approval question.

6. Implement only an approved package.
   - Make one coordinated backend/browser change. The backend owns typed evidence, reasoning order, complete scenes, descriptors, and the exact atomic effect; the browser owns rendering, accessibility, and locale catalogs without reconstructing Sudoku logic.
   - Update all production locale catalogs and reject English fallback leakage. Preserve scalar versus grouped descriptor cardinality as semantic contracts.
   - Update the canonical owning documentation only when the shipped contract changes; do not add chronology, review history, rejected alternatives, or duplicate owners.
   - Keep implementation limited to the approved strategy. Record adjacent improvements separately without starting them.
   - **Done when:** code, fixtures, catalogs, and current-state documentation express the approved package exactly.

7. Prove the complete boundary.
   - Add or update the focused regression for the exact maintained example and action order.
   - Run the backend's applicable build, test, vet, documentation, API-contract, CLI/API/TUI, 23-strategy matrix, and nine-game hint-only journey gates named by repository docs.
   - Run the browser's format, lint, unit, build, documentation/catalog/descriptor, complete strategy matrix, and real-stack nine-game journey gates named by repository docs.
   - Capture and inspect every teaching step on the actual board plus final English, Simplified Chinese, and Traditional Chinese desktop/phone evidence. Check the full candidate grid, cumulative proof, clipping, overflow, collisions, obscured candidates, board geometry, focus, and non-color meaning.
   - Report mechanical, semantic, editorial, visual, fresh-context, and CI evidence separately; never substitute one category for another.
   - **Done when:** every required gate passes or the strategy remains blocked with exact failed evidence.

8. Return the quality decision and stop at the boundary.
   - For `audit`, report `pass`, `revise`, or `block`, cite exact step/copy/scene failures, and propose the smallest corrected package without changing production code unless the request authorizes an update.
   - For `update` or `new-strategy`, report design approval separately from implementation readiness and implementation review separately from merge readiness.
   - After an approved merge, verify exact-head approval, metadata, documentation, mergeability, checks, synchronized defaults, and post-merge CI before marking the strategy complete.
   - Stop before the next strategy or deferred improvement.
   - **Done when:** the output has one status, evidence by category, open blockers, and exactly one next approval or action.
