#!/usr/bin/env python3
"""Solve the maintained complete-hint corpus through the full-screen TUI."""

import argparse
import fcntl
import os
from pathlib import Path
import pty
import re
import signal
import struct
import subprocess
import tempfile
import termios

from e2e_tui_hint_strategies import MAX_HINTS_PER_GAME, drain, isolated_env, load_cases, send

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
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 42, 100, 0, 0))
    process = subprocess.Popen(
        [binary, "tui", "--input", case["puzzle"], "--no-autosave"],
        stdin=slave,
        stdout=slave,
        stderr=slave,
        env=isolated_env(root),
        close_fds=True,
    )
    os.close(slave)
    transcript = drain(master, quiet=0.1, timeout=3)
    observed = []
    try:
        for _ in range(MAX_HINTS_PER_GAME):
            preview = send(master, b"i")
            transcript += preview
            observed.extend(
                match.strip() for match in re.findall(r"Hint — ([^(]+) \([^)]+\)", preview)
            )
            if "Step 1 of" not in preview:
                raise AssertionError("hint preview did not render its first teaching step")
            applied = send(master, b"\r")
            transcript += applied
            if "Hint applied." not in applied:
                raise AssertionError("hint conclusion did not apply through the TUI")
            if "solved" in applied.lower():
                send(master, b"q")
                if process.poll() is None:
                    send(master, b"y")
                process.wait(timeout=3)
                if process.returncode != 0:
                    raise AssertionError(f"TUI exited {process.returncode}")
                return observed
            if process.poll() is not None:
                raise AssertionError("TUI exited before the game was solved")
        raise AssertionError("game did not solve within the bounded hint sequence")
    except Exception as error:
        if process.poll() is None:
            process.send_signal(signal.SIGKILL)
            process.wait(timeout=3)
        raise AssertionError(f"{case['strategy']}: {error}\n{transcript[-5000:]}") from error
    finally:
        if process.poll() is None:
            process.kill()
            process.wait(timeout=3)
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
    with tempfile.TemporaryDirectory(prefix="sudoku-tui-full-game-e2e-") as directory:
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
    print(f"TUI full-game hint E2E passed: {len(corpus)} solved games covered {len(observed)}/23 strategies")
    for strategy, hints in counts:
        print(f"  {strategy}: solved in {hints} hint(s)")


if __name__ == "__main__":
    main()
