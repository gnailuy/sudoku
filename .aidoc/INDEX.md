---
domain: Conventions
status: Active
entry_points: []
dependencies: []
---

# .aidoc/INDEX.md — Discovery Index

This index provides reading chains for common starting points and a complete document map.

## Reading Chains

### Understanding the Architecture
1. `AGENT.md` — global rules, repo layout
2. `.aidoc/architecture/guidelines.md` — design constraints, layer boundaries, solver interface contract
3. `core/candidates.go` — `CandidateSet` bitfield type
4. `core/board.go` — `Board` struct with compute-on-fly `Candidates()` method
5. `solver/solver.go` — `Solver`, `StrategySolver`, `CompleteSolver` interfaces and `Base`
6. `solver/move.go` — `Move` struct (cell + technique + reason)
7. `solver/store.go` — solver registry with typed access
8. `game/game.go` — `Game` struct (pure state, no I/O)
9. `cli/controller.go` — CLI controller (terminal I/O, commands, display)

### Understanding Puzzle Generation
1. `.aidoc/designs/difficulty-model.md` — current model (clue-count), limitations, target model
2. `generator/difficulty.go` — difficulty levels and `StrategySolverKeys`
3. `generator/generator.go` — board generation and cell removal logic

### Understanding the Roadmap
1. `.aidoc/designs/roadmap.md` — future phases: generator + puzzle database, UI-ready engine
2. `.aidoc/designs/difficulty-model.md` — difficulty model, scoring, and puzzle classification
3. `.aidoc/architecture/guidelines.md` — current architecture and solver contract

### Adding a New Strategy Solver
1. `.aidoc/architecture/guidelines.md` — constraints, interface contract, step-by-step
2. `solver/solver.go` — implement `StrategySolver`
3. `solver/move.go` — return `*Move` from `Apply()`
4. `solver/store.go` — register with `RegisterStrategy()`
5. Write tests in `solver/<name>_solver_test.go`
6. Update `generator/difficulty.go` to reference the new solver key

## Document Map

| Path | Purpose |
|------|---------|
| `AGENT.md` | AI operator entry point — rules and repo layout |
| `.aidoc/INDEX.md` | This file — discovery index and reading chains |
| `.aidoc/architecture/guidelines.md` | Design constraints, layer boundaries, solver contract |
| `.aidoc/designs/difficulty-model.md` | Difficulty model: current state, limitations, and target design |
| `.aidoc/designs/roadmap.md` | Future phases: core refactoring, strategy solvers, puzzle database, UI-ready engine |
| `README.md` | Human-facing project summary |
