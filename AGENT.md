# AGENT.md — AI Operator Instructions

You are working on **Sudoku**, a Go platform with a shared game engine, line CLI, full-screen TUI, SQLite catalog, and guest/account HTTP API.

## Start Here

1. Run `git status --short --branch`; work on `feature/<short>` or `fix/<short>`, never directly on `main`.
2. Open [`.aidoc/INDEX.md`](.aidoc/INDEX.md) and follow only the reading chain for the task.
3. Read the named owning document, then its code entry points and tests. Code and schemas are the implementation facts; documentation preserves rationale, invariants, and cross-component contracts.
4. Build and test the smallest affected boundary before broad validation.

Human users start with [`README.md`](README.md). Do not duplicate its onboarding prose here or copy `.aidoc` contracts into this instruction file.

## Required Commands

```bash
go build ./...
go test ./...
go vet ./...
python3 scripts/validate_aidoc.py
python3 -m unittest scripts/test_validate_aidoc.py
```

Run the applicable built-binary E2E harnesses from the scenario route in `.aidoc/INDEX.md`. API changes also run `scripts/check-api-contract.sh`; deployment changes run `python3 scripts/check_deployment.py`.

## Repository Boundaries

| Path | Responsibility |
|---|---|
| `core/` | Board primitives, candidates, validation, normalization, canonical identity |
| `solver/` | Strategy solvers, evidence, classification, and scoring |
| `game/` | Frontend-neutral actions, snapshots, history, hints, and serialization |
| `cli/`, `tui/` | Line and full-screen presentation adapters |
| `api/`, `webapi/` | OpenAPI contract and HTTP translation |
| `db/`, `catalogbundle/` | Catalog, provenance, acquisition, accounts, and embedded supply |
| `generator/`, `calibration/`, `difficultyaudit/` | Generation and measurement |
| `cmd/` | Cobra command composition and process boundaries |
| `.aidoc/` | Durable rationale, invariants, and cross-component contracts |
| `scripts/` | Validation and black-box acceptance harnesses |

Keep packages focused and preserve dependency direction. The engine contains no terminal or HTTP behavior; clients submit typed actions instead of reconstructing Sudoku rules.

## Change Rules

- Follow existing Go conventions; run `gofmt` and `goimports` where applicable.
- Add no dependency without a concrete need and reviewable justification.
- Keep public behavior in one canonical owner. Neighbors link to it instead of restating it.
- Update the owning `.aidoc` document in the same change when code changes architecture, behavior, or workflow.
- Treat code, generated contracts, fixtures, and executable tests as the core evidence when auditing or updating documentation. Resolve a disagreement by fixing the documentation or code under the approved scope, never by copying an unverified claim.
- Describe current behavior and approved future direction only. Git and reviews own chronology.
- Every `.aidoc` document must remain registered; every Draft must appear once in the index's **Draft Work** section and stay out of current-state routes.
- Keep contributor instructions and review gates here, human setup and supported workflows in `README.md`, and durable design knowledge in `.aidoc/`.
- Audit every changed section for unique value (`ADDS`, `MIXED`, `DUPLICATES`) and readability (`CLEAR`, `DENSE`, `PADDED`). Documentation-wide changes require a full `.aidoc` audit.

## Test and Review Gates

- New solvers require package tests and registration updates.
- User-visible behavior or coverage changes require the applicable scenario catalog update and black-box E2E run against a built binary.
- Bug fixes should encode the reported preconditions and action order in a focused regression when practical.
- Run `go test ./...` and `go vet ./...` before committing; CI, lint, contract checks, and applicable E2E must pass before review.
- Use conventional commit subjects (`feat:`, `fix:`, `docs:`, `test:`, `chore:`) under 72 characters.
- PRs target `main` and are squash-merged only.

## Domain Orientation

Easy through Evil are deterministic solver-relative strategy grades; the highest required technique tier assigns the grade, weighted score orders puzzles only within a grade, and `strategy-unsolved` remains separate. For any deeper claim, use the bounded route in `.aidoc/INDEX.md` and verify the named implementation before editing.
