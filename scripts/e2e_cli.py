#!/usr/bin/env python3
"""Deterministic black-box E2E checks for the built line CLI and commands."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import sqlite3
import stat
import subprocess
import tempfile
import time

PUZZLE_DOTS = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."
PUZZLE_ZEROS = PUZZLE_DOTS.replace(".", "0")
SOLUTION = "483921657967345821251876493548132976729564138136798245372689514814253769695417382"
MULTIPLE_SOLUTIONS = "....7....6..195....98....6.8...6...34..8.3..17...2...6.6....28....419..5....8..79"
UNIQUE_SECOND = "53..7....6..195....98....6.8...6...34..8.3..17...2...6.6....28....419..5....8..79"
HARD_SEED = "...8.......5214.......5768.6...4.1...83...5.....5.1.2.2.1.....7....9....97...3..."


def isolated_env(root):
    env = os.environ.copy()
    env["XDG_DATA_HOME"] = str(root / "data")
    env["XDG_STATE_HOME"] = str(root / "state")
    return env


def run(binary, args, root, input_text=None, expected=0, timeout=20):
    result = subprocess.run(
        [binary, *args],
        input=input_text,
        capture_output=True,
        text=True,
        env=isolated_env(root),
        timeout=timeout,
    )
    output = result.stdout + result.stderr
    if result.returncode != expected:
        raise AssertionError(
            f"{' '.join(args)} exited {result.returncode}, want {expected}\n{output[-4000:]}"
        )
    return output


def contains(output, *needles):
    missing = [needle for needle in needles if needle not in output]
    if missing:
        raise AssertionError(f"output missing {missing}\n{output[-4000:]}")


def excludes(output, *needles):
    present = [needle for needle in needles if needle in output]
    if present:
        raise AssertionError(f"output unexpectedly contains {present}\n{output[-4000:]}")


def puzzle_rows(database):
    with sqlite3.connect(database) as connection:
        return connection.execute(
            "SELECT b.canonical_puzzle, b.difficulty, COALESCE((SELECT source FROM puzzle_provenance p WHERE p.base_puzzle_id=b.base_puzzle_id ORDER BY provenance_id LIMIT 1), '') FROM base_puzzles b ORDER BY b.canonical_puzzle"
        ).fetchall()


def acquisition_rows(database):
    with sqlite3.connect(database) as connection:
        return connection.execute(
            "SELECT canonical_puzzle, play_count, last_played_at FROM base_puzzles ORDER BY canonical_puzzle"
        ).fetchall()


def completion_commands(puzzle, solution):
    commands = []
    for index, value in enumerate(puzzle):
        if value == ".":
            commands.append(f"add {index // 9 + 1} {index % 9 + 1} {solution[index]}")
    return "\n".join(commands) + "\n"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("binary", nargs="?", default="./sudoku")
    args = parser.parse_args()
    binary = os.path.abspath(args.binary)

    with tempfile.TemporaryDirectory(prefix="sudoku-cli-e2e-") as directory:
        root = Path(directory)

        # Startup, parsing, help, and backward-compatible root flags.
        output = run(binary, ["--help"], root)
        contains(output, "calibrate", "experiment", "generate", "import", "tui", "--input", "--level")
        output = run(binary, ["generate", "--help"], root)
        contains(output, "--count", "--difficulty", "--workers", "--timeout", "--rounds", "--db")
        output = run(binary, ["import", "--help"], root)
        contains(output, "--file", "--source", "--format", "--sha256", "--workers", "--db")
        output = run(binary, ["calibrate", "--help"], root)
        contains(output, "--manifest", "--output", "append-only", "resumable")
        output = run(binary, ["experiment", "generation", "--help"], root)
        contains(output, "--manifest", "--output", "equal budgets", "resumable")

        # Difficulty measurement is deterministic and resumes without
        # duplicating append-only observations.
        candidate_manifest = root / "calibration-candidate.json"
        manifest = root / "calibration-manifest.json"
        calibration_run = root / "calibration-run"
        candidate_manifest.write_text(
            json.dumps(
                {
                    "version": 2,
                    "name": "e2e-pilot",
                    "puzzles": [
                        {
                            "id": "known-1",
                            "puzzle": PUZZLE_ZEROS,
                            "source_category": "pathological",
                            "source_id": "e2e-fixture:known-1",
                            "license": "repository-license",
                            "redistribution": "permitted",
                            "collection_method": "checked-in E2E fixture",
                            "split": "exploratory",
                        }
                    ],
                },
                indent=2,
            )
            + "\n",
            encoding="utf-8",
        )
        output = run(binary, ["calibrate", "prepare", "--input", str(candidate_manifest), "--output", str(manifest)], root)
        contains(output, "Prepared 1 normalized puzzles")
        prepared = json.loads(manifest.read_text(encoding="utf-8"))
        if prepared["puzzles"][0]["puzzle"] != PUZZLE_DOTS or not prepared["puzzles"][0]["puzzle_hash"]:
            raise AssertionError("calibration preparation did not normalize and hash the candidate")
        contains(
            run(binary, ["calibrate", "prepare", "--input", str(candidate_manifest), "--output", str(manifest)], root, expected=1),
            "already exists",
        )
        output = run(binary, ["calibrate", "--manifest", str(manifest), "--output", str(calibration_run)], root)
        contains(output, "Measured 1/1 puzzles (1 new).", "Manifest SHA-256")
        output = run(binary, ["calibrate", "--manifest", str(manifest), "--output", str(calibration_run)], root)
        contains(output, "Measured 1/1 puzzles (0 new).")
        observations_path = calibration_run / "observations.jsonl"
        observations = observations_path.read_text(encoding="utf-8").splitlines()
        report = json.loads((calibration_run / "report.json").read_text(encoding="utf-8"))
        checkpoint = json.loads((calibration_run / "checkpoint.json").read_text(encoding="utf-8"))
        if (
            len(observations) != 1
            or not report.get("complete")
            or report.get("reproducible") != 1
            or report.get("by_source", {}).get("pathological", {}).get("observed") != 1
            or report.get("by_split", {}).get("exploratory", {}).get("observed") != 1
            or not report.get("metrics_by_difficulty")
            or checkpoint.get("next_index") != 1
        ):
            raise AssertionError("calibration run did not preserve resumable stratified artifacts")
        manifest_data = json.loads(manifest.read_text(encoding="utf-8"))
        manifest_data["name"] = "changed-pilot"
        manifest.write_text(json.dumps(manifest_data, indent=2) + "\n", encoding="utf-8")
        contains(
            run(binary, ["calibrate", "--manifest", str(manifest), "--output", str(calibration_run)], root, expected=1),
            "immutable manifest",
        )
        if observations_path.read_text(encoding="utf-8").splitlines() != observations:
            raise AssertionError("changed manifest modified append-only observations")

        # Exact-grade generation experiments run both arms with equal budgets,
        # persist raw outcomes, and resume without repeating completed jobs.
        generation_manifest = root / "generation-experiment.json"
        generation_run = root / "generation-experiment-run"
        generation_manifest.write_text(
            json.dumps(
                {
                    "version": 1,
                    "name": "e2e-generation-experiment",
                    "repository_commit": "e2e-fixture",
                    "solver_config_hash": "e2e-fixture",
                    "policy_version": "trace-guided-v1",
                    "budget": {"max_duration_ms": 1, "max_classifications": 1},
                    "samples": [
                        {
                            "id": "hard-1",
                            "split": "exploratory",
                            "target_difficulty": "hard",
                            "seed_id": "e2e-hard-1",
                            "seed_puzzle": HARD_SEED,
                            "random_seed": 42,
                        }
                    ],
                },
                indent=2,
            )
            + "\n",
            encoding="utf-8",
        )
        output = run(binary, ["experiment", "generation", "--manifest", str(generation_manifest), "--output", str(generation_run)], root, timeout=30)
        contains(output, "Observed 2/2 jobs (2 new).", "Manifest SHA-256")
        output = run(binary, ["experiment", "generation", "--manifest", str(generation_manifest), "--output", str(generation_run)], root, timeout=30)
        contains(output, "Observed 2/2 jobs (0 new).")
        generation_observations = (generation_run / "observations.jsonl").read_text(encoding="utf-8").splitlines()
        generation_report = json.loads((generation_run / "report.json").read_text(encoding="utf-8"))
        if len(generation_observations) != 2 or not generation_report.get("complete"):
            raise AssertionError("generation experiment did not preserve resumable two-arm artifacts")

        contains(run(binary, ["--input", PUZZLE_DOTS], root, "q\n"), "Exiting the game.", PUZZLE_DOTS)
        contains(run(binary, ["--input", PUZZLE_ZEROS], root, "q\n"), "Exiting the game.", PUZZLE_DOTS)
        contains(run(binary, ["--input", "123"], root, expected=1), "not a valid Sudoku problem")
        contains(run(binary, ["--level", "banana"], root, expected=1), "invalid difficulty level")
        contains(
            run(binary, ["--level", "easy"], root, "q\n", timeout=60),
            "Generating a random Easy Sudoku problem...",
            "Exiting the game.",
            "Problem:",
        )
        contains(
            run(binary, ["--input", MULTIPLE_SOLUTIONS], root, "q\n"),
            "has 2 solutions",
            "Exiting the game.",
        )

        # Real line-controller lifecycle: invalid state, repair, values, clear,
        # history, hint metadata, reset, notes, and clean quit.
        commands = "\n".join(
            [
                "1 1 5",
                "check",
                "repair",
                "add 1 1 4",
                "clear 1 1",
                "undo",
                "redo",
                "note 1 1 1",
                "notes-clear 1 1",
                "hint",
                "reset",
                "q",
                "",
            ]
        )
        output = run(binary, ["--input", PUZZLE_DOTS], root, commands)
        contains(
            output,
            "You have entered incorrect value(s).",
            "Hint:",
            "Exiting the game.",
        )

        # A new action after undo must truncate the abandoned redo branch.
        output = run(
            binary,
            ["--input", PUZZLE_DOTS],
            root,
            "1 1 4\nu\n1 1 5\nr\nc\nq\n",
        )
        contains(output, "You have entered incorrect value(s).", "Exiting the game.")
        excludes(output, "The current board is correct.")
        contains(
            run(binary, ["--input", PUZZLE_DOTS], root, "solve\n"),
            "Congratulations! You have solved the problem.",
        )

        # Durable sessions preserve notes, invalid values, and redo history.
        session = root / "session.json"
        output = run(
            binary,
            ["--input", PUZZLE_DOTS],
            root,
            f"note 1 2 1\n1 1 5\n1 1 4\nu\nsave {session}\nq\n",
        )
        contains(output, f"Session saved to {session}.")
        if stat.S_IMODE(session.stat().st_mode) != 0o600:
            raise AssertionError("session file mode is not 0600")
        saved = json.loads(session.read_text(encoding="utf-8"))
        if saved.get("version") != 1:
            raise AssertionError("saved session does not use version 1")
        output = run(binary, ["--resume", str(session)], root, "check\nr\ncheck\nq\n")
        contains(
            output,
            "You have entered incorrect value(s).",
            "The current board is correct.",
        )

        corrupt = root / "corrupt.json"
        corrupt.write_text("{bad json", encoding="utf-8")
        unsupported = root / "unsupported.json"
        unsupported.write_text('{"version":999}', encoding="utf-8")
        oversized = root / "oversized.json"
        oversized.write_bytes(b"x" * (1024 * 1024 + 1))
        for source, message in (
            (corrupt, "resume saved session"),
            (unsupported, "resume saved session"),
            (oversized, "session file is too large"),
        ):
            contains(run(binary, ["--resume", str(source)], root, expected=1), message)
        contains(
            run(binary, ["--resume", str(session), "--input", PUZZLE_DOTS], root, expected=1),
            "none of the others can be",
        )
        contains(
            run(binary, ["--resume", str(session), "--level", "easy"], root, expected=1),
            "none of the others can be",
        )
        destination = root / "existing-destination"
        destination.mkdir()
        output = run(binary, ["--input", PUZZLE_DOTS], root, f"save {destination}\nq\n")
        contains(output, "Failed to run the save command")
        if not destination.is_dir() or list(root.glob(".sudoku-session-*")):
            raise AssertionError("failed save changed the destination or leaked a temporary file")

        # Import normalization, invalid lines, source labels, empty input, and dedup.
        database = root / "commands.db"
        puzzles = root / "puzzles.txt"
        transposed_relabelled = "".join(
            "." if PUZZLE_DOTS[column * 9 + row] == "." else str(10 - int(PUZZLE_DOTS[column * 9 + row]))
            for row in range(9)
            for column in range(9)
        )
        puzzles.write_text(
            "# fixture\n" + PUZZLE_DOTS + "\n" + transposed_relabelled + "\n123456\nabc\n",
            encoding="utf-8",
        )
        output = run(
            binary,
            ["import", "--file", str(puzzles), "--source", "e2e-fixture", "--db", str(database)],
            root,
        )
        contains(output, "Total lines: 4", "Valid: 2", "Invalid (skipped): 2", "Stored (new): 1", "Duplicates: 1")
        rows = puzzle_rows(database)
        if len(rows) != 1 or rows[0][2] != "e2e-fixture" or len(rows[0][0]) != 81 or "0" in rows[0][0]:
            raise AssertionError(f"unexpected imported database rows: {rows}")
        output = run(
            binary,
            ["import", "--file", str(puzzles), "--source", "second", "--db", str(database)],
            root,
        )
        contains(output, "Stored (new): 0", "Duplicates: 2")
        empty = root / "empty.txt"
        empty.write_text("# comments only\n\n", encoding="utf-8")
        contains(
            run(binary, ["import", "--file", str(empty), "--db", str(database)], root),
            "Total lines: 0",
            "Stored (new): 0",
        )
        contains(
            run(binary, ["import", "--file", str(root / "missing.txt")], root, expected=1),
            "open file",
        )

        pinned_record = (
            "00015097c6c3 083020090000800100029300008000098700070000060006740000300006980002005000010030540  7.2\n"
            "0001d2888928 200050006010000090600801003007090600000703000900080002100000005060902010003060200  7.1\n"
        )
        pinned_file = root / "sudoku-exchange.txt"
        pinned_file.write_text(pinned_record, encoding="utf-8")
        pinned_hash = hashlib.sha256(pinned_record.encode()).hexdigest()
        pinned_database = root / "pinned.db"
        contains(
            run(binary, ["import", "--file", str(pinned_file), "--format", "sudoku-exchange", "--db", str(pinned_database)], root, expected=1),
            "require --sha256",
        )
        output = run(
            binary,
            ["import", "--file", str(pinned_file), "--format", "sudoku-exchange", "--sha256", pinned_hash, "--source", "sudoku-exchange-diabolical@fixture", "--workers", "2", "--db", str(pinned_database)],
            root,
        )
        contains(output, "Stored (new): 1", "Strategy-unsolved (skipped): 1")
        with sqlite3.connect(pinned_database) as connection:
            provenance = connection.execute("SELECT source, source_ref FROM puzzle_provenance").fetchone()
        if provenance != ("sudoku-exchange-diabolical@fixture", "sha1:00015097c6c3"):
            raise AssertionError(f"unexpected pinned provenance: {provenance}")

        # Default high-grade play uses exact catalog supply without paying the
        # bounded-generation cost, and preserves transformed play-run linkage.
        with sqlite3.connect(pinned_database) as connection:
            catalog_level, catalog_id = connection.execute(
                "SELECT difficulty, base_puzzle_id FROM base_puzzles"
            ).fetchone()
        if catalog_level not in {"hard", "expert", "evil"}:
            raise AssertionError(f"catalog-first fixture has unexpected grade: {catalog_level}")
        started = time.monotonic()
        output = run(
            binary,
            ["--level", catalog_level, "--db", str(pinned_database)],
            root,
            "q\n",
        )
        elapsed = time.monotonic() - started
        contains(output, f"Selecting an exact {catalog_level.capitalize()} puzzle from the catalog...", "Exiting the game.")
        excludes(output, "Generating a random", "Falling back to bounded generation")
        if elapsed > 2:
            raise AssertionError(f"catalog-first play took {elapsed:.3f}s, want at most 2s")
        with sqlite3.connect(pinned_database) as connection:
            play_count = connection.execute(
                "SELECT play_count FROM base_puzzles WHERE base_puzzle_id = ?", (catalog_id,)
            ).fetchone()[0]
            linked = connection.execute(
                "SELECT COUNT(*) FROM play_runs WHERE base_puzzle_id = ? AND status = 'active'",
                (catalog_id,),
            ).fetchone()[0]
        if play_count != 1 or linked != 1:
            raise AssertionError(
                f"catalog-first linkage is incomplete: play_count={play_count}, active_runs={linked}"
            )

        # An unavailable catalog is visible before bounded generation begins;
        # generation may either produce a playable fallback or exhaust its budget.
        unavailable_database = root / "catalog-is-a-directory"
        unavailable_database.mkdir()
        started = time.monotonic()
        unavailable = subprocess.run(
            [binary, "--level", "evil", "--db", str(unavailable_database)],
            input="q\n",
            capture_output=True,
            text=True,
            env=isolated_env(root),
            timeout=12,
        )
        unavailable_output = unavailable.stdout + unavailable.stderr
        if unavailable.returncode not in {0, 1}:
            raise AssertionError(
                f"unavailable catalog exited {unavailable.returncode}\n{unavailable_output[-4000:]}"
            )
        contains(
            unavailable_output,
            "Exact-grade catalog unavailable (",
            "Falling back to bounded generation.",
        )
        if time.monotonic() - started > 10:
            raise AssertionError("catalog-unavailable fallback exceeded its bounded generation window")

        # Legacy rows require an explicit destructive rebuild; no identity or
        # provenance is guessed from the obsolete schema.
        legacy_database = root / "legacy.db"
        with sqlite3.connect(legacy_database) as connection:
            connection.execute("CREATE TABLE puzzles (puzzle TEXT PRIMARY KEY)")
            connection.execute("INSERT INTO puzzles (puzzle) VALUES (?)", (PUZZLE_DOTS,))
        contains(
            run(binary, ["db", "stats", "--db", str(legacy_database)], root, expected=1),
            "db rebuild",
        )
        contains(
            run(binary, ["db", "rebuild", "--db", str(legacy_database)], root, expected=1),
            "requires --yes",
        )
        contains(
            run(binary, ["db", "rebuild", "--db", str(legacy_database), "--yes"], root),
            "All catalog, provenance, and play-run rows will be deleted.",
            "Catalog rebuilt at schema version 2.",
        )
        with sqlite3.connect(legacy_database) as connection:
            names = {row[0] for row in connection.execute("SELECT name FROM sqlite_master WHERE type='table'")}
            version = connection.execute("SELECT version FROM schema_metadata WHERE singleton=1").fetchone()[0]
        if version != 2 or not {"base_puzzles", "puzzle_provenance", "play_runs"}.issubset(names) or "puzzles" in names:
            raise AssertionError(f"catalog rebuild produced unexpected schema: version={version}, tables={names}")

        # Public database acquisition exhausts never-played rows before reuse.
        acquisition_database = root / "acquisition.db"
        acquisition_source = root / "acquisition.txt"
        acquisition_source.write_text(PUZZLE_DOTS + "\n", encoding="utf-8")
        run(binary, ["import", "--file", str(acquisition_source), "--db", str(acquisition_database)], root)
        with sqlite3.connect(acquisition_database) as connection:
            difficulty, score, technique = connection.execute(
                "SELECT difficulty, score, max_technique FROM base_puzzles"
            ).fetchone()
            second_puzzle = "53..7....6..195....98....6.8...6...34..8.3..17...2...6.6....28....419..5....8..79"
            second_id = "bp_" + hashlib.sha256(second_puzzle.encode()).hexdigest()
            connection.execute(
                "INSERT INTO base_puzzles (base_puzzle_id, canonical_puzzle, difficulty, score, max_technique) VALUES (?, ?, ?, ?, ?)",
                (second_id, second_puzzle, difficulty, score, technique),
            )
            connection.execute(
                "INSERT INTO puzzle_provenance (base_puzzle_id, source) VALUES (?, ?)",
                (second_id, "e2e-second"),
            )
        for _ in range(2):
            contains(
                run(binary, ["--from-db", "--level", difficulty, "--db", str(acquisition_database)], root, "q\n"),
                "Exiting the game.",
            )
        rows = acquisition_rows(acquisition_database)
        if sorted(row[1] for row in rows) != [1, 1] or any(not row[2] for row in rows):
            raise AssertionError(f"database did not exhaust never-played rows: {rows}")
        run(binary, ["--from-db", "--level", difficulty, "--db", str(acquisition_database)], root, "q\n")
        rows = acquisition_rows(acquisition_database)
        if sorted(row[1] for row in rows) != [1, 2]:
            raise AssertionError(f"database reuse is not balanced: {rows}")
        contains(
            run(binary, ["--from-db", "--level", "evil", "--db", str(acquisition_database)], root, expected=1),
            "no evil puzzle is available in the database",
        )
        contains(
            run(binary, ["--from-db", "--input", PUZZLE_DOTS, "--db", str(acquisition_database)], root, expected=1),
            "none of the others can be",
        )

        # Acquisition and completion are separate, inspectable history dimensions.
        statistics_database = root / "statistics.db"
        run(binary, ["--input", PUZZLE_DOTS, "--db", str(statistics_database)], root, "q\n")
        with sqlite3.connect(statistics_database) as connection:
            statistics_level = connection.execute("SELECT difficulty FROM base_puzzles").fetchone()[0]
        run(binary, ["--from-db", "--level", statistics_level, "--db", str(statistics_database)], root, "q\n")
        contains(run(binary, ["--input", PUZZLE_DOTS, "--db", str(statistics_database)], root, "solve\n"), "Congratulations! You have solved the problem.")
        with sqlite3.connect(statistics_database) as connection:
            if connection.execute("SELECT completion_count FROM base_puzzles").fetchone()[0] != 0:
                raise AssertionError("automatic solve counted as player completion")
        contains(
            run(binary, ["--input", PUZZLE_DOTS, "--db", str(statistics_database)], root, completion_commands(PUZZLE_DOTS, SOLUTION)),
            "Congratulations! You have solved the problem.",
        )
        run(binary, ["--from-db", "--level", statistics_level, "--db", str(statistics_database)], root, "q\n")
        with sqlite3.connect(statistics_database) as connection:
            history = connection.execute("SELECT play_count, completion_count, last_played_at, last_completed_at FROM base_puzzles").fetchone()
        if history[0:2] != (2, 1) or not history[2] or not history[3]:
            raise AssertionError(f"separate history was not recorded: {history}")
        output = run(binary, ["db", "stats", "--db", str(statistics_database), "--level", statistics_level], root)
        contains(output, "ACQUISITIONS", "COMPLETIONS", statistics_level, "overall")
        contains(run(binary, ["db", "stats", "--level", "banana", "--db", str(root / "must-not-exist.db")], root, expected=1), "invalid difficulty level")
        if (root / "must-not-exist.db").exists():
            raise AssertionError("invalid statistics filter opened a database")
        contains(run(binary, ["db", "reset-history", "--history", "completion", "--level", statistics_level, "--db", str(statistics_database)], root, expected=1), "requires --yes")
        contains(run(binary, ["db", "reset-history", "--history", "completion", "--level", statistics_level, "--db", str(statistics_database), "--yes"], root), "Affected puzzles: 1", "History reset complete.")
        with sqlite3.connect(statistics_database) as connection:
            reset_history = connection.execute("SELECT play_count, completion_count, last_played_at, last_completed_at FROM base_puzzles").fetchone()
        if reset_history[0] != 2 or reset_history[1] != 0 or not reset_history[2] or reset_history[3] is not None:
            raise AssertionError(f"completion reset changed the wrong history: {reset_history}")
        run(binary, ["db", "reset-history", "--history", "all", "--db", str(statistics_database), "--yes"], root)
        if len(puzzle_rows(statistics_database)) != 1:
            raise AssertionError("history reset deleted a puzzle")
        if not os.path.isfile(session):
            raise AssertionError("history reset changed an explicit save file")

        # Independent processes import overlapping fixtures and read snapshots
        # through the public command boundary against one SQLite file.
        concurrent_database = root / "concurrent.db"
        import_a = root / "concurrent-a.txt"
        import_b = root / "concurrent-b.txt"
        import_a.write_text((PUZZLE_DOTS + "\n" + UNIQUE_SECOND + "\n") * 6, encoding="utf-8")
        import_b.write_text((UNIQUE_SECOND + "\n" + PUZZLE_DOTS + "\n") * 6, encoding="utf-8")
        run(binary, ["import", "--file", str(import_a), "--db", str(concurrent_database)], root, timeout=30)
        commands = [
            [binary, "import", "--file", str(import_a), "--source", "concurrent-a", "--db", str(concurrent_database)],
            [binary, "import", "--file", str(import_b), "--source", "concurrent-b", "--db", str(concurrent_database)],
            [binary, "db", "stats", "--db", str(concurrent_database)],
            [binary, "db", "stats", "--db", str(concurrent_database)],
        ]
        processes = [
            subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, env=isolated_env(root))
            for command in commands
        ]
        for command, process in zip(commands, processes):
            try:
                stdout, stderr = process.communicate(timeout=30)
            except subprocess.TimeoutExpired:
                process.kill()
                process.communicate()
                raise AssertionError(f"concurrent process timed out: {' '.join(command[1:])}")
            output = stdout + stderr
            if process.returncode != 0 or "database error" in output.lower():
                raise AssertionError(f"concurrent process failed: {' '.join(command[1:])}\n{output[-4000:]}")
            if command[1:3] == ["db", "stats"]:
                contains(output, "ACQUISITIONS", "COMPLETIONS", "overall")
        rows = puzzle_rows(concurrent_database)
        if len(rows) != 2:
            raise AssertionError(f"overlapping concurrent imports stored {len(rows)} rows, want 2: {rows}")
        with sqlite3.connect(concurrent_database) as connection:
            by_difficulty = connection.execute(
                "SELECT difficulty, COUNT(*) FROM base_puzzles GROUP BY difficulty ORDER BY difficulty"
            ).fetchall()
        for difficulty, count in by_difficulty:
            for _ in range(count):
                contains(
                    run(binary, ["--from-db", "--level", difficulty, "--db", str(concurrent_database)], root, "q\n"),
                    "Exiting the game.",
                )
        with sqlite3.connect(concurrent_database) as connection:
            acquisitions = connection.execute("SELECT SUM(play_count) FROM base_puzzles").fetchone()[0]
            integrity = connection.execute("PRAGMA quick_check").fetchone()[0]
        if acquisitions != 2 or integrity != "ok":
            raise AssertionError(f"post-contention state acquisitions={acquisitions}, quick_check={integrity}")

        # Generation validation and a tightly bounded real-worker smoke run.
        contains(run(binary, ["generate", "--count", "0"], root, expected=1), "count must be positive")
        contains(run(binary, ["generate", "--difficulty", "invalid"], root, expected=1), "invalid difficulty level")
        generated = root / "generated.db"
        started = time.monotonic()
        output = run(
            binary,
            [
                "generate",
                "--count",
                "1",
                "--difficulty",
                "hard",
                "--workers",
                "2",
                "--timeout",
                "1ms",
                "--rounds",
                "1",
                "--db",
                str(generated),
            ],
            root,
            timeout=5,
        )
        elapsed = time.monotonic() - started
        contains(output, "Attempted: 1", "Generated: 0", "Timed out: 1", "=== Generation Report ===")
        if elapsed > 2:
            raise AssertionError(f"hard generation deadline returned after {elapsed:.3f}s")
        if not generated.is_file() or puzzle_rows(generated):
            raise AssertionError("timed-out generation stored an incomplete puzzle")

        # Root play auto-stores through the default XDG data path.
        auto_database = root / "data" / "sudoku" / "puzzles.db"
        contains(run(binary, ["--input", PUZZLE_DOTS], root, "q\n"), "Exiting the game.")
        if not auto_database.is_file() or not puzzle_rows(auto_database):
            raise AssertionError("root play did not auto-store the puzzle")

    print("PASS: line CLI gameplay, sessions, calibration, generation experiments, import, generation, and SQLite composition")


if __name__ == "__main__":
    main()
