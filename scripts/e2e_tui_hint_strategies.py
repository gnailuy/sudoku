#!/usr/bin/env python3
"""Black-box hint acceptance for every registered strategy through the TUI."""

import argparse
import fcntl
import json
import os
from pathlib import Path
import pty
import re
import select
import signal
import struct
import subprocess
import tempfile
import termios
import time

ANSI = re.compile(rb"\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07]*(?:\x07|\x1b\\)|[()][A-Z0-9])")
MAX_HINTS_PER_GAME = 128


def load_cases():
    manifest = Path(__file__).resolve().parents[1] / "game" / "testdata" / "hint_strategy_games.json"
    data = json.loads(manifest.read_text(encoding="utf-8"))
    cases = data.get("cases", [])
    if data.get("schema_version") != 1 or len(cases) != 23:
        raise AssertionError("hint strategy manifest must contain schema version 1 and 23 cases")
    return cases


def isolated_env(root):
    env = os.environ.copy()
    env["XDG_DATA_HOME"] = str(root / "data")
    env["XDG_STATE_HOME"] = str(root / "state")
    env["LOCALAPPDATA"] = str(root / "local-app-data")
    env["NO_COLOR"] = "1"
    return env


def drain(fd, quiet=0.1, timeout=5.0):
    chunks = []
    deadline = time.monotonic() + timeout
    quiet_deadline = time.monotonic() + quiet
    while time.monotonic() < deadline:
        wait = min(0.04, max(0, quiet_deadline - time.monotonic()))
        ready, _, _ = select.select([fd], [], [], wait)
        if ready:
            try:
                chunks.append(os.read(fd, 65536))
            except OSError:
                break
            quiet_deadline = time.monotonic() + quiet
        elif chunks and time.monotonic() >= quiet_deadline:
            break
    return ANSI.sub(b"", b"".join(chunks)).decode("utf-8", "replace")


def send(fd, keys):
    os.write(fd, keys)
    return drain(fd)


def run_case(binary, case, root):
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
    expected = f"Hint — {case['display_name']} ({case['grade'].title()})"
    observed = []
    try:
        for attempt in range(1, MAX_HINTS_PER_GAME + 1):
            preview = send(master, b"i")
            transcript += preview
            observed.extend(
                match.strip() for match in re.findall(r"Hint — ([^(]+) \([^)]+\)", preview)
            )
            matched = expected in preview
            if "Step 1 of" not in preview:
                raise AssertionError(f"{case['strategy']} did not render the first teaching step")
            if matched:
                next_step = send(master, b"\x1b[C")
                transcript += next_step
                if "Step 2 of" not in next_step:
                    raise AssertionError(f"{case['strategy']} did not navigate to the second teaching step")
                previous_step = send(master, b"\x1b[D")
                transcript += previous_step
                if "Step 1 of" not in previous_step:
                    raise AssertionError(f"{case['strategy']} did not navigate back locally")
            applied = send(master, b"\r")
            transcript += applied
            if matched:
                if "Hint applied." not in applied:
                    raise AssertionError(f"{case['strategy']} did not apply through the TUI")
                send(master, b"q")
                if process.poll() is None:
                    send(master, b"y")
                process.wait(timeout=3)
                if process.returncode != 0:
                    raise AssertionError(f"{case['strategy']} TUI exited {process.returncode}")
                return attempt
            if process.poll() is not None:
                break
        raise AssertionError(
            f"{case['strategy']} was not reached; observed strategies: {sorted(set(observed))}"
        )
    except Exception as error:
        if process.poll() is None:
            process.send_signal(signal.SIGKILL)
            process.wait(timeout=3)
        raise AssertionError(f"{error}\n{transcript[-5000:]}") from error
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

    results = []
    with tempfile.TemporaryDirectory(prefix="sudoku-tui-hint-e2e-") as directory:
        root = Path(directory)
        for case in load_cases():
            count = run_case(binary, case, root / case["strategy"])
            results.append((case["strategy"], count))

    print(f"TUI hint strategy E2E passed: {len(results)}/23 strategies")
    for strategy, hints in results:
        print(f"  {strategy}: navigated and applied after {hints} hint(s)")


if __name__ == "__main__":
    main()
