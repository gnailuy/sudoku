---
domain: Designs
status: Active
entry_points: []
dependencies:
  - .aidoc/architecture/guidelines.md
  - .aidoc/designs/difficulty-model.md
---

# Roadmap

Future development phases for the Sudoku project.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/architecture/guidelines.md` | Current layer boundaries and solver contract |
| `.aidoc/designs/difficulty-model.md` | Difficulty model design (strategy-based, with scoring and puzzle database) |
| `.aidoc/INDEX.md` | Discovery index |

## Phase 4: Generator and Puzzle Database

### Goal

Replace the current generate-or-reject loop with a best-effort generator backed by a
persistent puzzle database. When real-time generation can't produce a puzzle at the
requested difficulty within a time/iteration budget, fall back to a database lookup.

### Architecture

```
User requests puzzle
        │
        ▼
┌─────────────────┐
│  Best-effort     │  Try generating with time/iteration limit
│  Generator       │
└────────┬────────┘
         │
    ┌────┴─────┐
    │ Success? │
    └────┬─────┘
     yes │        no
         │         │
         ▼         ▼
    Return     ┌──────────────┐
    puzzle     │  DB Lookup    │  Random unplayed puzzle at requested level
               └──────┬───────┘
                      │
                 ┌────┴─────┐
                 │  Found?  │
                 └────┬─────┘
                  yes │       no
                      │        │
                      ▼        ▼
                 Return    Return best-effort puzzle
                 puzzle    with difficulty mismatch warning:
                           "Expected: Hard, got: Medium"
```

### Puzzle Database (SQLite)

Store puzzles in a local SQLite database. Each puzzle is stored in its normalized
(canonical) form — digit-swapped equivalents map to the same record.

**Schema (conceptual):**

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PRIMARY KEY | Auto-increment row id |
| `puzzle` | TEXT UNIQUE | 81-char normalized puzzle string (`.` for empty cells) |
| `solution` | TEXT | 81-char solved board string |
| `difficulty` | TEXT | Difficulty level name (easy/medium/hard/expert/evil) |
| `clues` | INTEGER | Number of given clues |
| `score` | INTEGER | Total difficulty score (Σ technique weights) |
| `max_technique` | TEXT | Highest-tier technique required (solver key) |
| `played` | BOOLEAN DEFAULT FALSE | Whether this puzzle has been played |
| `source` | TEXT | Origin: "generated", "imported", or source name |
| `created_at` | TIMESTAMP | When the puzzle was added |

**Normalization as dedup key:** The existing `Board.Normalize()` remaps digits so the
first row is always 1–9. Two puzzles that differ only by digit permutation share the
same normalized form → stored once.

### Best-Effort Generator

Enhance the existing generator with configurable limits:

- **Max iterations** (already exists): cap on cell-removal attempts.
- **Max duration**: wall-clock time limit (e.g., 5 seconds default).
- **Max rounds**: number of full generate-from-scratch attempts before giving up.

When the budget is exhausted, the generator returns whatever it has — even if the
difficulty tier doesn't match the request. The caller decides whether to use it
or fall back to the database.

### Fallback Flow

When the generator fails to produce a puzzle at the target difficulty:

1. Query the database for a random unplayed puzzle at the requested level.
2. If found: return it and mark it as played.
3. If not found: return the best-effort puzzle with a warning message:
   `"Requested difficulty: Hard. Generated puzzle difficulty: Medium. Enjoy!"`

### Batch Generation CLI

A new CLI command for offline puzzle generation:

```bash
sudoku generate --count 100 --difficulty hard --timeout 30s --db puzzles.db
```

**Behavior:**
- Generate `N` puzzles at the specified difficulty (best-effort per puzzle).
- Classify each puzzle: determine actual difficulty tier + score using `ScorePuzzle()`.
- Normalize and deduplicate against the database.
- Store new unique puzzles.
- Output a report:

```
Generated: 100
Stored (new): 73
Duplicates: 27

By difficulty:
  Easy:   12
  Medium: 31
  Hard:   22
  Expert:  7
  Evil:    1
```

### Puzzle Sources

Three approaches to populate the database:

1. **Batch generation:** Run the CLI command above repeatedly (offline, low priority).
   Random generation is inefficient for hard+ puzzles, but it's free and accumulates
   over time.

2. **Public puzzle databases:** Import puzzles from established collections
   (e.g., HoDoKu test puzzles, Gordon Royle's 17-clue collection, top1465).
   Each import run normalizes, classifies, and deduplicates.

3. **Session collection:** Puzzles from email threads and interactive sessions
   are already in normalized string form — import them into the database with
   their known difficulty classification.

### Implementation Plan

| PR | Scope | Description |
|----|-------|-------------|
| A | Database layer | New `db/` package: SQLite schema, CRUD operations, random-unplayed query, dedup by normalized key. |
| B | Best-effort generator | Add time/round limits to generator. Return partial result when budget exhausted. Classify result by difficulty. |
| C | Fallback flow | Wire generator → DB fallback in `game/` or `cli/`. Show mismatch warning when downgrading difficulty. |
| D | Batch CLI | `sudoku generate` command: generate N puzzles, classify, store, report. |
| E | Import CLI | `sudoku import` command: load puzzles from files (one 81-char string per line), classify, deduplicate, store. |
| F | Played tracking | Mark puzzles as played during interactive sessions. Filter played puzzles from DB lookup. |

PRs are sequential: A → B → C → D → E → F.

### Package Layout

```
db/
├── db.go          # Open/close, schema migration
├── puzzle.go      # InsertPuzzle, GetRandomUnplayed, MarkPlayed, Stats
└── db_test.go     # Integration tests with in-memory SQLite
```

The `db` package depends on `core` (for normalization) and `solver` (for scoring/classification).
It does NOT depend on `generator`, `game`, or `cli` — keeping the dependency graph clean.

## Phase 5: UI-Ready Core Engine

Refactor the core game into a reusable engine that can serve as the backend for
multiple UI implementations — similar to how GNU Go provides a core engine
used by various graphical frontends.

Goals:
- **Clean API boundary:** The engine exposes game state, moves, undo/redo, hints,
  and validation through a well-defined API. No terminal I/O assumptions in the core.
- **Note-taking support:** Players can annotate cells with candidate values
  (pencil marks). The engine tracks notes as part of the game state, including
  undo/redo for note operations.
- **CLI as one frontend:** The existing CLI is already separated into `cli/controller.go`,
  consuming the Game API. Additional UIs follow the same pattern.
- **UI independence:** The engine makes no assumptions about rendering, input method,
  or platform. A web UI, TUI, or mobile app should all be viable frontends.

The core Game struct is already pure state (no I/O) after Phase 2. This phase
extends it with note-taking and formalizes the API as a stable engine boundary.
