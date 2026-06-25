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
  Store in DB  ┌──────────────┐
  (if new) &   │  DB Lookup    │  Random puzzle at requested level
  return       └──────┬───────┘
                      │
                 ┌────┴─────┐
                 │  Found?  │
                 └────┬─────┘
                  yes │       no
                      │        │
                      ▼        ▼
                 Return    Store best-effort in DB (if new)
                 puzzle    & return with mismatch warning:
                           "Expected: Hard, got: Medium"
```

### Puzzle Database (SQLite)

Store puzzles in a local SQLite database. Each puzzle is stored in its normalized
(canonical) form — digit-swapped equivalents map to the same record.

**Schema (conceptual):**

| Column | Type | Description |
|--------|------|-------------|
| `puzzle` | TEXT PRIMARY KEY | 81-char normalized puzzle string (`.` for empty cells) |
| `difficulty` | TEXT NOT NULL | Difficulty level name (easy/medium/hard/expert/evil) |
| `score` | INTEGER NOT NULL | Total difficulty score (Σ technique weights) |
| `max_technique` | TEXT NOT NULL | Highest-tier technique required (solver key) |
| `source` | TEXT | Origin: "generated", "imported", or source name |
| `created_at` | TIMESTAMP | When the puzzle was added |

**Why no `solution` or `clues` columns:** Both are trivially derivable from the
puzzle string — solve it for the solution, count non-`.` characters for clues.
Storing them would be redundant.

**Why `puzzle` as primary key:** The normalized puzzle string is already unique
(that's the whole point of normalization). Using it directly as the PK avoids an
extra surrogate `id` column and makes dedup lookups a simple primary key check.

**Played tracking is deferred.** Played/completed status, game intermediate state,
and the question of tracking normalized vs. unnormalized puzzles will be designed
separately in a future version. This keeps the initial schema focused.

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

### Auto-Store on Generation

Every puzzle that is generated (whether during interactive play or batch generation)
is automatically stored in the database in normalized form, if it doesn't already
exist. This means the database grows organically through normal usage, not just
through explicit batch runs or imports.

### Fallback Flow

When the generator fails to produce a puzzle at the target difficulty:

1. Query the database for a random puzzle at the requested level.
2. If found: return it.
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
| 1 | Database + generator + fallback | New `db/` package (SQLite schema, CRUD, random query, dedup by normalized key). Best-effort generator with time/round limits. Fallback flow wired in `game/`/`cli/` with mismatch warning. Auto-store generated puzzles in DB. |
| 2 | Batch CLI + import CLI | `sudoku generate` command (batch generation, classify, store, report). `sudoku import` command (load from files, classify, deduplicate, store). |

PRs are sequential: 1 → 2. Played tracking is deferred to a future version.

### Package Layout

```
db/
├── db.go          # Open/close, schema migration
├── puzzle.go      # InsertPuzzle, GetRandom, Stats
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
