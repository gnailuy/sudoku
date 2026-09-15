#!/usr/bin/env python3
"""Black-box pseudo-terminal smoke test for `sudoku tui`."""
import argparse
import fcntl
import os
import pty
import re
import select
import signal
import sqlite3
import struct
import subprocess
import tempfile
import termios
import time

NEARLY_SOLVED = ".23456789456789123789123456214365897365897214897214365531642978642978531978531642"
PUZZLE = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."
ANSI = re.compile(rb"\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07]*(?:\x07|\x1b\\)|[()][A-Z0-9])")


def drain(fd, seconds=0.25):
    end = time.monotonic() + seconds
    chunks = []
    while time.monotonic() < end:
        ready, _, _ = select.select([fd], [], [], 0.05)
        if not ready:
            continue
        try:
            chunks.append(os.read(fd, 65536))
        except OSError:
            break
    return b"".join(chunks)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("binary", nargs="?", default="./sudoku")
    args = parser.parse_args()
    with tempfile.TemporaryDirectory() as directory:
        session = os.path.join(directory, "tui-session.json")
        master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 42, 90, 0, 0))
        env = os.environ.copy()
        env["XDG_DATA_HOME"] = os.path.join(directory, "data")
        env["XDG_STATE_HOME"] = os.path.join(directory, "state")
        recovery_directory = os.path.join(env["XDG_STATE_HOME"], "sudoku", "recovery")
        process = subprocess.Popen(
            [args.binary, "tui", "--input", PUZZLE],
            stdin=slave, stdout=slave, stderr=slave, env=env, close_fds=True,
        )
        os.close(slave)
        output = drain(master, 0.8)
        # A confirmed invalid value increments the authoritative count, and
        # Undo restores the board without decrementing that cumulative count.
        for keys in (b"3", b"u"):
            os.write(master, keys)
            output += drain(master)
        # Resize below and back above the minimum; game state must survive.
        fcntl.ioctl(master, termios.TIOCSWINSZ, struct.pack("HHHH", 20, 40, 0, 0))
        process.send_signal(signal.SIGWINCH)
        small_output = drain(master, 0.4)
        if b"Terminal too small" not in ANSI.sub(b"", small_output):
            raise AssertionError("small-terminal fallback was not rendered")
        output += small_output
        fcntl.ioctl(master, termios.TIOCSWINSZ, struct.pack("HHHH", 42, 90, 0, 0))
        process.send_signal(signal.SIGWINCH)
        output += drain(master, 0.4)
        # Adopt automatic candidates on the first note edit, prove the whole
        # transition is one undo/redo step, then exercise ordinary gameplay,
        # save explicitly, resume, and quit.
        for keys in (b"a", b"n", b"1", b"u", b"r", b"n", b"l", b"5", b"a", b"q", b"n", b"a", b"n", b"j", b"4", b"u", b"r", b"?", b"\x1b", b"i", b"\r", b"S"):
            os.write(master, keys)
            output += drain(master)
        os.write(master, session.encode() + b"\r")
        output += drain(master, 0.5)
        os.write(master, b"q")
        output += drain(master, 0.5)
        try:
            return_code = process.wait(timeout=3)
        except subprocess.TimeoutExpired:
            process.kill()
            raise AssertionError("TUI did not quit after a clean save")
        finally:
            os.close(master)
        text = ANSI.sub(b"", output).decode("utf-8", "replace")
        required = ("SUDOKU", "Mistakes: 0", "Mistakes: 1", "AUTO ON", "AUTO OFF", "Candidates copied. Notes on.", "NOTE  ", "KEYBOARD HELP", "Hint preview:", "Saved to ", "unsaved", "Unsaved changes")
        missing = [value for value in required if value not in text]
        if missing:
            raise AssertionError(f"screen output missing {missing}\n{text[-4000:]}")
        if return_code != 0:
            raise AssertionError(f"TUI exited {return_code}\n{text[-2000:]}")
        if not os.path.isfile(session) or os.stat(session).st_mode & 0o777 != 0o600:
            raise AssertionError("explicit save did not create a mode-0600 session")
        # A saved session must start in a second full-screen process and quit cleanly.
        master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 42, 90, 0, 0))
        resumed = subprocess.Popen([args.binary, "tui", "--resume", session], stdin=slave, stdout=slave, stderr=slave, env=env, close_fds=True)
        os.close(slave)
        resumed_output = drain(master, 0.7)
        os.write(master, b"q")
        resumed_output += drain(master, 0.3)
        resumed_text = ANSI.sub(b"", resumed_output)
        if resumed.wait(timeout=3) != 0 or b"SUDOKU" not in resumed_text or b"AUTO OFF" not in resumed_text:
            raise AssertionError("saved TUI session did not resume cleanly with candidates off")
        os.close(master)
        # An abnormal exit retains the latest debounced record, and plain
        # startup discovers it by durable random identifier.
        master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 42, 90, 0, 0))
        interrupted = subprocess.Popen([args.binary, "tui", "--input", PUZZLE], stdin=slave, stdout=slave, stderr=slave, env=env, close_fds=True)
        os.close(slave)
        drain(master, 0.7)
        os.write(master, b"2")
        interrupted_output = drain(master, 3.0)
        interrupted.kill()
        interrupted.wait(timeout=3)
        os.close(master)
        records = [name for name in os.listdir(recovery_directory) if name.endswith(".json")]
        if len(records) != 1 or len(records[0]) != 37:
            raise AssertionError(f"interrupted TUI did not leave one random recovery record: {records}")
        if os.stat(recovery_directory).st_mode & 0o777 != 0o700 or os.stat(os.path.join(recovery_directory, records[0])).st_mode & 0o777 != 0o600:
            raise AssertionError("recovery storage permissions are not private")

        master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 42, 90, 0, 0))
        recovery_process = subprocess.Popen([args.binary, "tui"], stdin=slave, stdout=slave, stderr=slave, env=env, close_fds=True)
        os.close(slave)
        recovery_output = drain(master, 8.0)
        recovery_text = ANSI.sub(b"", recovery_output)
        if b"RECOVER A GAME" not in recovery_text:
            recovery_process.kill()
            raise AssertionError(f"plain startup did not offer recovery\n{recovery_text[-3000:]!r}")
        os.write(master, b"\r")
        recovery_output += drain(master, 0.7)
        os.write(master, b"q")
        recovery_output += drain(master, 0.5)
        if recovery_process.wait(timeout=3) != 0 or b"Recovered game" not in ANSI.sub(b"", recovery_output):
            raise AssertionError("selected recovery did not restore and quit cleanly")
        os.close(master)
        if any(name.endswith(".json") for name in os.listdir(recovery_directory)):
            raise AssertionError("clean exit retained the selected recovery record")

        # Opt-out disables both discovery and record creation.
        master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 42, 90, 0, 0))
        opted_out = subprocess.Popen([args.binary, "tui", "--input", PUZZLE, "--no-autosave"], stdin=slave, stdout=slave, stderr=slave, env=env, close_fds=True)
        os.close(slave)
        drain(master, 0.7)
        os.write(master, b"2")
        drain(master, 1.5)
        opted_out.send_signal(signal.SIGINT)
        opted_out.wait(timeout=3)
        os.close(master)
        if any(name.endswith(".json") for name in os.listdir(recovery_directory)):
            raise AssertionError("--no-autosave created a recovery record")

        # A player-driven solve records completion through the same play-run tracker.
        completion_database = os.path.join(directory, "tui-completion.db")
        master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 42, 90, 0, 0))
        completion_process = subprocess.Popen([args.binary, "tui", "--input", NEARLY_SOLVED, "--db", completion_database, "--no-autosave"], stdin=slave, stdout=slave, stderr=slave, env=env, close_fds=True)
        os.close(slave)
        drain(master, 0.7)
        os.write(master, b"1")
        completion_output = drain(master, 0.5)
        os.write(master, b"q")
        completion_output += drain(master, 0.2)
        os.write(master, b"y")
        completion_output += drain(master, 0.3)
        if completion_process.wait(timeout=3) != 0 or b"solved" not in ANSI.sub(b"", completion_output).lower():
            raise AssertionError("TUI player completion did not reach solved state")
        os.close(master)
        with sqlite3.connect(completion_database) as connection:
            if connection.execute("SELECT SUM(completion_count) FROM puzzles").fetchone()[0] != 1:
                raise AssertionError("TUI player completion was not recorded")

        corrupt = os.path.join(directory, "corrupt.json")
        with open(corrupt, "w", encoding="utf-8") as handle:
            handle.write("{bad json")
        rejected = subprocess.run([args.binary, "tui", "--resume", corrupt], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, env=env, timeout=3)
        if rejected.returncode == 0 or b"resume saved session" not in rejected.stdout.lower():
            raise AssertionError("corrupt TUI restore was not rejected before startup")
        print("PASS: TUI PTY gameplay, save/resume, crash autosave, recovery selection, cleanup, and quit")


if __name__ == "__main__":
    main()
