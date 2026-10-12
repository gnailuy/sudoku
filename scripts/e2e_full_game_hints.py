#!/usr/bin/env python3
"""Solve a minimal maintained game corpus through public line-CLI hints."""

import argparse
import os
from pathlib import Path
import pty
import subprocess
import tempfile

from e2e_hint_strategies import MAX_HINTS_PER_GAME, isolated_env, load_cases, read_response, send

CORPUS = {
    "hidden-triple",
    "w-wing",
    "swordfish",
    "naked-quad",
    "xyz-wing",
    "jellyfish",
    "unique-rectangle-2",
    "unique-rectangle-3",
    "unique-rectangle-4",
}


def solve_game(binary, case, root):
    master, slave = pty.openpty()
    process = subprocess.Popen(
        [binary, "--input", case["puzzle"]],
        stdin=slave,
        stdout=slave,
        stderr=slave,
        env=isolated_env(root),
        close_fds=True,
    )
    os.close(slave)
    transcript = read_response(master, process)
    observed = []
    try:
        for _ in range(MAX_HINTS_PER_GAME):
            preview = send(master, process, "hint")
            transcript += preview
            observed.extend(
                line.split(":", 1)[1].split("(", 1)[0].strip()
                for line in preview.splitlines()
                if line.startswith("Strategy:")
            )
            try:
                applied = send(master, process, "hint apply")
            except TimeoutError as error:
                applied = str(error)
                if "Congratulations! You have solved the problem." not in applied:
                    raise
            transcript += applied
            if "Congratulations! You have solved the problem." in transcript:
                process.wait(timeout=5)
                return observed
            if process.poll() is not None:
                raise AssertionError("game exited without the solved completion message")
        raise AssertionError("game did not solve within the bounded hint sequence")
    except Exception as error:
        if process.poll() is None:
            process.kill()
            process.wait(timeout=5)
        raise AssertionError(f"{case['strategy']}: {error}\n{transcript[-5000:]}") from error
    finally:
        if process.poll() is None:
            process.kill()
            process.wait(timeout=5)
        os.close(master)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("binary", nargs="?", default="./sudoku")
    args = parser.parse_args()
    binary = os.path.abspath(args.binary)
    cases = load_cases()
    expected_names = {case["display_name"] for case in cases}
    corpus = [case for case in cases if case["strategy"] in CORPUS]
    if len(corpus) != len(CORPUS):
        raise AssertionError("full-game corpus does not resolve every selected fixture")

    observed = set()
    counts = []
    with tempfile.TemporaryDirectory(prefix="sudoku-full-game-e2e-") as directory:
        root = Path(directory)
        for case in corpus:
            trace = solve_game(binary, case, root / case["strategy"])
            observed.update(trace)
            counts.append((case["strategy"], len(trace)))

    if observed != expected_names:
        raise AssertionError(
            f"full-game strategy set mismatch: missing={sorted(expected_names-observed)} "
            f"unexpected={sorted(observed-expected_names)}"
        )
    print(f"Full-game hint E2E passed: {len(corpus)} solved games covered {len(observed)}/23 strategies")
    for strategy, hints in counts:
        print(f"  {strategy}: solved in {hints} hint(s)")


if __name__ == "__main__":
    main()
