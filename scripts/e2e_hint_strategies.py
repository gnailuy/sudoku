#!/usr/bin/env python3
"""Black-box hint acceptance for every registered strategy through the line CLI."""

import argparse
import json
import os
from pathlib import Path
import pty
import select
import subprocess
import tempfile
import time

MAX_HINTS_PER_GAME = 128
PROMPT = "Enter a command (Enter 'help' or 'h' for help):"


def isolated_env(root):
    env = os.environ.copy()
    env["XDG_DATA_HOME"] = str(root / "data")
    env["XDG_STATE_HOME"] = str(root / "state")
    return env


def load_cases():
    manifest = Path(__file__).resolve().parents[1] / "game" / "testdata" / "hint_strategy_games.json"
    data = json.loads(manifest.read_text(encoding="utf-8"))
    if data.get("schema_version") != 1 or len(data.get("cases", [])) != 23:
        raise AssertionError("hint strategy manifest must contain schema version 1 and 23 cases")
    return data["cases"]


def read_response(master, process, timeout=10):
    output = bytearray()
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        ready, _, _ = select.select([master], [], [], 0.1)
        if ready:
            try:
                output.extend(os.read(master, 65536))
            except OSError:
                break
            if PROMPT.encode() in output:
                return output.decode(errors="replace")
        if process.poll() is not None:
            while True:
                try:
                    output.extend(os.read(master, 65536))
                except OSError:
                    break
            return output.decode(errors="replace")
    raise TimeoutError(output.decode(errors="replace"))


def send(master, process, command):
    os.write(master, (command + "\n").encode())
    return read_response(master, process)


def run_case(binary, case, root):
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
    try:
        read_response(master, process)
        expected = f"Strategy: {case['display_name']} ({case['grade']})"
        observed = []
        for attempt in range(1, MAX_HINTS_PER_GAME + 1):
            preview = send(master, process, "hint")
            observed.extend(
                line.strip("\r")
                for line in preview.splitlines()
                if line.startswith("Strategy:")
            )
            matched = expected in preview
            if matched and "Conclusion:" not in preview:
                raise AssertionError(f"{case['strategy']} preview did not expose a conclusion")

            applied = send(master, process, "hint apply")
            if matched:
                if "Applied hint:" not in applied:
                    raise AssertionError(f"{case['strategy']} preview was not applied through the CLI")
                if process.poll() is None:
                    os.write(master, b"q\n")
                process.wait(timeout=5)
                return attempt
            if process.poll() is not None:
                break

        raise AssertionError(
            f"{case['strategy']} was not reached; observed strategies: {sorted(set(observed))}"
        )
    except TimeoutError as error:
        process.kill()
        process.wait(timeout=5)
        raise AssertionError(
            f"{case['strategy']} CLI stopped responding\n{str(error)[-4000:]}"
        ) from error
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

    results = []
    with tempfile.TemporaryDirectory(prefix="sudoku-hint-e2e-") as directory:
        root = Path(directory)
        for case in load_cases():
            count = run_case(binary, case, root / case["strategy"])
            results.append((case["strategy"], count))

    print(f"Hint strategy E2E passed: {len(results)}/23 strategies")
    for strategy, hints in results:
        print(f"  {strategy}: reached and applied after {hints} hint(s)")


if __name__ == "__main__":
    main()
