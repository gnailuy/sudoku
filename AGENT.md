# AGENT.md — AI Operator Instructions

You are an AI agent working on **Sudoku**, a CLI Sudoku game written in Go.

## Quick Start

```bash
go build && go test ./...
```

## Entry Points

- **This file** — global rules, branching, CI requirements.
- **`.aidoc/INDEX.md`** — discovery index. Start here for architecture, designs, and reading chains.
- **`README.md`** — human-facing project summary.

## Repository Layout

```
.
├── main.go              # Entry point — delegates to cmd package
├── cmd/                 # Cobra CLI commands (play, generate, import)
├── cli/                 # CLI controller, signal handling (interactive play)
├── core/                # Board, cell, position, validator, normalizer, string serialization
├── solver/              # Solver interfaces, solver store, backtracking solver, classification
├── generator/           # Puzzle generation — solved board + cell removal, best-effort with limits
├── game/                # Game state — pure logic (undo/redo/hints), no I/O
├── db/                  # SQLite puzzle database — schema, CRUD, random query, dedup
├── util/                # Random shuffle, array helpers
├── .aidoc/              # AI-native documentation
│   ├── INDEX.md
│   ├── architecture/
│   └── designs/
└── .github/workflows/   # CI (go test, go vet, golangci-lint)
```

## Rules

### Branching

- Work on feature branches (`feature/<short>` or `fix/<short>`), never directly on `main`.
- PRs target `main`. Squash-merge only.

### Code Style

- Follow existing Go conventions in the codebase.
- Use `gofmt` / `goimports` formatting.
- No new dependencies without justification.
- Keep packages focused: one responsibility per package.

### Testing

- Every new solver must have tests.
- Every feature and bug-fix PR must update the applicable scenario catalog linked from `.aidoc/designs/e2e-test-scenarios.md` when user-visible behavior or coverage changes, then run all applicable black-box E2E scenarios against a built binary.
- Run `go test ./...` before committing.
- Run `go vet ./...` to catch issues.
- CI must pass before merge.

### Commit Messages

- Use conventional style: `feat:`, `fix:`, `docs:`, `test:`, `chore:`.
- Keep subject line under 72 characters.

### Documentation

- Maintain `.aidoc/` throughout development; every code or workflow change updates affected docs in the same PR rather than deferring documentation cleanup.
- Follow DocGuidelines: docs capture the *why* and *constraints*, not the *how* that code already expresses.
- Describe only current state and approved future direction. Git owns history; remove completed work from the roadmap and remove superseded proposal or delivery language from other docs.
- Keep active contributor instructions, review gates, and execution checklists in `AGENT.md`; design docs retain rationale, invariants, and verification facts.
- Keep documents near 100 lines. Split or compress documents over 120 lines unless an index or reference map clearly benefits from the extra length.
- Make each paragraph self-contained by naming its subject instead of relying on context-dependent pronouns.
- Audit every affected `.aidoc/` document against all DocGuidelines before review; documentation-wide changes require a full `.aidoc/` audit.
- `README.md` is for humans; `.aidoc/` is for AI agents.

## Domain Context

This is a Sudoku puzzle game with 23 strategy solvers across five deterministic strategy grades (Easy through Evil). The canonical solver's highest required technique tier assigns the grade; clue count guides generation, and HoDoKu-based score orders puzzles only within a grade. These labels do not predict human experience, and `strategy-unsolved` remains separate from Evil. The reusable game engine exposes a stable action/snapshot API with note-taking, unified undo/redo, and versioned serialization. The CLI consumes that boundary; see `.aidoc/designs/game-engine.md` for the current design and `.aidoc/designs/roadmap.md` for scope and deferred work.
