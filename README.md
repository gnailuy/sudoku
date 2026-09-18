# Sudoku

A CLI Sudoku game in Go. Generate puzzles at various strategy grades, solve them, and play interactively with undo/redo and hints.

Easy through Evil are deterministic solver-relative strategy grades, not predictions of human solving experience. The highest technique tier required by the canonical solver assigns the grade; weighted score orders puzzles only within that grade. Puzzles the strategy inventory cannot finish remain `strategy-unsolved`, never Evil.

Features:
- 23 strategy solvers across 5 strategy grades (Easy → Evil)
- Canonical strategy classification with within-grade HoDoKu-weighted scoring
- Best-effort puzzle generator with time/round limits
- SQLite puzzle database with automatic storage, exact-grade acquisition, balanced reuse, completion statistics, and explicit history reset
- Batch generation CLI for offline puzzle creation
- Import CLI for loading puzzles from files
- Interactive play with manual notes, undo/redo, and technique-aware hints
- Explicit atomic session saves with validated resume
- Opt-in full-screen Bubble Tea interface with keyboard navigation
- Opt-in automatic legal candidates in the full-screen TUI
- Private background autosave and crash recovery for TUI games
- Versioned HTTP API with optimistic concurrency, recovery, bearer auth, and explicit CORS allowlisting

## Build

```bash
go build
```

## Play

```bash
# Random puzzle (default: hard)
./sudoku

# Choose difficulty
./sudoku -l medium
./sudoku -l hard
./sudoku -l expert
./sudoku -l evil

# Custom board (use . for empty cells)
./sudoku -i .56.4.7...1.5....6.......19...9.....3.58..2...4...6...1.....93....4....22.3.1....

# Select an exact-grade stored puzzle without generation
./sudoku --from-db --level hard
./sudoku --from-db --level hard --db ./puzzles.db

# Resume a session saved from the interactive prompt
./sudoku --resume game.json
```

## Full-Screen TUI

The existing line-oriented game remains the default. Launch the optional terminal UI with the same puzzle sources:

```bash
./sudoku tui --level medium
./sudoku tui --input "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."
./sudoku tui --resume game.json
./sudoku tui --no-autosave
```

The TUI autosaves gameplay changes to a private XDG state directory after a one-second pause. A later plain `sudoku tui` start offers valid recent games for recovery; records use random durable identifiers, never process IDs. Explicit `--input`, `--level`, and `--resume` starts bypass recovery selection, and `--no-autosave` disables both writing and discovery.

Use arrows or `h`/`j`/`k`/`l` to move, `1`–`9` to enter a value, `n` to toggle note mode, `a` to show or hide automatic legal candidates, `u`/`r` for history, `i` then Enter to preview/apply a hint, `?` for keyboard help, `S` to save, and `q` to quit. Automatic candidates start hidden and are not saved; manual notes remain player-owned. Unsaved sessions require confirmation.

The TUI defaults to a deterministic dark palette. Set `SUDOKU_THEME=light` for the light palette or `SUDOKU_THEME=no-color` (also selected by `NO_COLOR`) for an attribute-only accessible fallback.

## HTTP API

Start the backend-only API on its safe loopback default:

```bash
./sudoku api
./sudoku api --listen 127.0.0.1:9090
```

For non-loopback addresses, configure a bearer token. Browser origins remain denied unless each exact origin is explicitly allowed:

```bash
./sudoku api --listen 0.0.0.0:8080 --auth-token "$SUDOKU_API_TOKEN" \
  --allowed-origin https://sudoku.example.com
```

The OpenAPI 3.1 contract is [`api/openapi.yaml`](api/openapi.yaml). All game operations are under `/api/v1`; `GET /healthz` is payload-free. API sessions use opaque identifiers and monotonic revisions, persist accepted mutations to the private recovery store, and reject stale writes with `409 Conflict`.

## Portable Deployment

The API binds to loopback by default and can run behind any reverse proxy that owns the public route. The portable service example in [`deploy/sudoku-api.service.example`](deploy/sudoku-api.service.example) keeps application files separate from private XDG data and recovery state; operators choose their own release directories, listener, browser origins, and host policy.

Trusted branch workflows can provide the backend binary, Git commit, and checksum to a serialized host-side deployment flow. Pairing with the static client, destination hostnames, ports, credentials, and active release selection remain outside this repository. See [the portable deployment contract](.aidoc/designs/deployment-hardening.md).

## Measure Strategy Ratings Locally

Run deterministic classifications over an immutable JSON corpus and resume safely in the same output directory:

```bash
./sudoku calibrate --manifest corpus.json --output calibration-run
```

The command classifies each puzzle twice, appends manifest-bound observations to `observations.jsonl`, checkpoints progress, and derives JSON and Markdown reports. Reusing the output directory with a changed manifest is rejected rather than mixing evidence.

## Database Statistics

Inspect acquisition and completion history independently:

```bash
./sudoku db stats
./sudoku db stats --level hard --db ./puzzles.db
```

Reset only an explicitly selected history dimension. Interactive use asks for confirmation; automation must pass `--yes`:

```bash
./sudoku db reset-history --history completion --level hard
./sudoku db reset-history --history all --yes --db ./puzzles.db
```

History reset preserves puzzle rows, classifications, sources, save files, and recovery records.

## Batch Generate

Generate puzzles and store them in the database:

```bash
# Generate 100 puzzles targeting hard difficulty
./sudoku generate -n 100 -d hard

# Generate 500 evil puzzles with 4 parallel workers
./sudoku generate -n 500 -d evil -w 4

# Custom timeout and rounds per puzzle
./sudoku generate -n 50 -d expert -t 60s --rounds 20
```

## Import Puzzles

Import puzzles from a text file (one per line, 81 chars):

```bash
# Import from file (supports . or 0 for empty cells)
./sudoku import -f puzzles.txt

# Custom source label
./sudoku import -f top1465.txt --source "top1465"
```

## In-Game Commands

During play, enter moves as `row col value` (for example, `1 2 5`). Commands accept the displayed long form or alias:

- `add`, `a <row> <column> <value>` — set a value
- `clear`, `d <row> <column>` — clear a value
- `note`, `n <row> <column> <value>` — toggle a manual candidate note
- `notes-clear`, `x <row> <column>` — clear a cell's notes
- `save <path>` — atomically save values, notes, and undo/redo history
- `undo`, `u` / `redo`, `r` — move through value and note history
- `hint`, `i` — apply a technique-aware hint
- `check`, `c` / `repair`, `f` — inspect or remove invalid entries
- `reset`, `e` / `solve`, `s` — restart or solve the puzzle
- `help`, `h` / `quit`, `q` — show command help or exit without saving

Manual notes use a fixed 3×3 candidate layout. Saving is explicit: use `save` before `quit`, then pass the same file to `--resume` later.

## Development

```bash
go test ./...    # Run tests
go vet ./...     # Static analysis
```

For AI agents: start with [`AGENT.md`](AGENT.md) → [`.aidoc/INDEX.md`](.aidoc/INDEX.md).

## License

MIT
