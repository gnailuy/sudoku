#!/usr/bin/env python3
"""Black-box smoke test for the built Sudoku HTTP API."""

import json
import os
import signal
import sqlite3
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request

NEARLY_SOLVED = ".23456789456789123789123456214365897365897214897214365531642978642978531978531642"
PUZZLE = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."
ORIGIN = "http://client.example"


def free_port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def request(base, method, path, body=None, content_type="application/json", headers=None):
    data = body
    if isinstance(body, dict):
        data = json.dumps(body).encode()
    req_headers = dict(headers or {})
    if data is not None:
        req_headers["Content-Type"] = content_type
    req = urllib.request.Request(base + path, data=data, headers=req_headers, method=method)
    try:
        response = urllib.request.urlopen(req, timeout=5)
    except urllib.error.HTTPError as error:
        response = error
    payload = response.read()
    parsed = json.loads(payload) if payload and "json" in response.headers.get("Content-Type", "") else payload
    return response.status, parsed, response.headers


def start(binary, state, port, extra_args=None):
    env = dict(os.environ, XDG_STATE_HOME=state)
    command = [binary, "api", "--listen", f"127.0.0.1:{port}", "--allowed-origin", ORIGIN]
    command.extend(extra_args or [])
    process = subprocess.Popen(
        command,
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    base = f"http://127.0.0.1:{port}"
    for _ in range(50):
        if process.poll() is not None:
            raise RuntimeError(process.stderr.read())
        try:
            if request(base, "GET", "/healthz")[0] == 200:
                return process, base
        except OSError:
            time.sleep(0.1)
    process.kill()
    raise RuntimeError("API did not become healthy")


def stop(process):
    process.send_signal(signal.SIGTERM)
    process.wait(timeout=10)
    if process.returncode != 0:
        raise RuntimeError(process.stderr.read())


def expect(actual, expected, message):
    if actual != expected:
        raise AssertionError(f"{message}: got {actual!r}, want {expected!r}")


def main():
    binary = os.path.abspath(sys.argv[1] if len(sys.argv) > 1 else "./sudoku")
    with tempfile.TemporaryDirectory(prefix="sudoku-api-e2e-") as root:
        state = os.path.join(root, "state")
        database = os.path.join(root, "puzzles.db")
        process, base = start(binary, state, free_port(), ["--db", database])
        try:
            expect(request(base, "GET", "/missing")[0], 404, "unknown route")
            status, created, _ = request(base, "POST", "/api/v1/sessions", {"source": {"kind": "puzzle", "puzzle": PUZZLE}})
            expect(status, 201, "create")
            session_id = created["id"]
            expect(created["revision"], 0, "initial revision")

            status, difficulty_session, _ = request(
                base,
                "POST",
                "/api/v1/sessions",
                {"source": {"kind": "difficulty", "difficulty": "easy"}},
            )
            expect(status, 201, "create by canonical difficulty source")
            expect(difficulty_session["revision"], 0, "difficulty initial revision")

            expect(request(base, "POST", "/api/v1/sessions", b"{}", headers={"Origin": ORIGIN})[2].get("Access-Control-Allow-Origin"), ORIGIN, "allowed origin")
            expect(request(base, "OPTIONS", "/api/v1/sessions", headers={"Origin": "http://denied.example", "Access-Control-Request-Method": "POST"})[0], 403, "denied origin")
            expect(request(base, "POST", "/api/v1/sessions", {"source": {"kind": "puzzle", "puzzle": PUZZLE}, "extra": True})[0], 400, "unknown field")

            path = f"/api/v1/sessions/{session_id}"
            expect(request(base, "GET", path)[0], 200, "get")
            expect(request(base, "GET", path + "/hint")[0], 200, "hint")
            action = {"kind": "toggle-note", "expected_revision": 0, "row": 1, "column": 1, "value": 1}
            status, changed, _ = request(base, "POST", path + "/actions", action)
            expect(status, 200, "action")
            expect(changed["revision"], 1, "mutated revision")
            expect(request(base, "POST", path + "/actions", action)[0], 409, "stale revision")

            status, completion, _ = request(base, "POST", "/api/v1/sessions", {"source": {"kind": "puzzle", "puzzle": NEARLY_SOLVED}})
            expect(status, 201, "create completion session")
            completion_path = f"/api/v1/sessions/{completion['id']}/actions"
            status, completed, _ = request(base, "POST", completion_path, {"kind": "set-value", "expected_revision": 0, "row": 1, "column": 1, "value": 1})
            expect(status, 200, "complete through API")
            expect(completed["snapshot"]["status"], "solved", "API completion status")

            status, exported, _ = request(base, "GET", path + "/export")
            expect(status, 200, "export")

            contender = subprocess.run(
                [binary, "api", "--listen", f"127.0.0.1:{free_port()}"],
                env=dict(os.environ, XDG_STATE_HOME=state),
                capture_output=True,
                text=True,
                timeout=5,
            )
            if contender.returncode == 0 or "another sudoku api process" not in contender.stderr:
                raise AssertionError("second API process was not rejected by the recovery lock")
        finally:
            stop(process)

        with sqlite3.connect(database) as connection:
            expect(connection.execute("SELECT SUM(completion_count) FROM puzzles").fetchone()[0], 1, "API completion count")

        process, base = start(binary, state, free_port(), ["--db", database])
        try:
            expect(request(base, "GET", path)[1]["revision"], 1, "restart recovery")
            status, imported, _ = request(base, "POST", "/api/v1/sessions/import", exported, "application/vnd.sudoku.session+json")
            expect(status, 201, "import")
            expect(request(base, "GET", "/api/v1/sessions")[0], 200, "list")
            expect(request(base, "DELETE", f"/api/v1/sessions/{imported['id']}")[0], 204, "discard")
        finally:
            stop(process)

        unsafe = subprocess.run([binary, "api", "--listen", "0.0.0.0:0"], env=dict(os.environ, XDG_STATE_HOME=state), capture_output=True, text=True, timeout=5)
        if unsafe.returncode == 0 or "--auth-token is required" not in unsafe.stderr:
            raise AssertionError("non-loopback startup did not require authentication")

        for invalid_origin in ("*", "null", "https://client.example/path"):
            invalid = subprocess.run(
                [binary, "api", "--listen", "127.0.0.1:0", "--allowed-origin", invalid_origin],
                env=dict(os.environ, XDG_STATE_HOME=state),
                capture_output=True,
                text=True,
                timeout=5,
            )
            if invalid.returncode == 0 or "expected an exact http or https origin" not in invalid.stderr:
                raise AssertionError(f"invalid origin was accepted: {invalid_origin}")

        process, base = start(binary, state, free_port(), ["--auth-token", "secret"])
        try:
            expect(request(base, "GET", "/api/v1/sessions")[0], 401, "missing bearer token")
            expect(request(base, "GET", "/api/v1/sessions", headers={"Authorization": "secret"})[0], 401, "bare token")
            expect(request(base, "GET", "/api/v1/sessions", headers={"Authorization": "Bearer secret"})[0], 200, "valid bearer token")
        finally:
            stop(process)

    print("HTTP API E2E passed")


if __name__ == "__main__":
    main()
