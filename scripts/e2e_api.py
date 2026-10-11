#!/usr/bin/env python3
"""Black-box acceptance for the clean guest/account HTTP boundary."""

import json
import os
import signal
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request

PUZZLE = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."
ORIGIN = "http://client.example"


def free_port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def request(base, method, path, body=None, headers=None):
    data = json.dumps(body).encode() if isinstance(body, dict) else body
    request_headers = dict(headers or {})
    if data is not None:
        request_headers["Content-Type"] = "application/json"
    req = urllib.request.Request(base + path, data=data, headers=request_headers, method=method)
    try:
        response = urllib.request.urlopen(req, timeout=5)
    except urllib.error.HTTPError as error:
        response = error
    payload = response.read()
    parsed = json.loads(payload) if payload and "json" in response.headers.get("Content-Type", "") else payload
    return response.status, parsed, response.headers


def start(binary, root, port, extra_args=None):
    command = [binary, "api", "--listen", f"127.0.0.1:{port}", "--allowed-origin", ORIGIN]
    command.extend(extra_args or [])
    process = subprocess.Popen(command, env=dict(os.environ, XDG_STATE_HOME=os.path.join(root, "state")), stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
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


def expect(actual, expected, message):
    if actual != expected:
        raise AssertionError(f"{message}: got {actual!r}, want {expected!r}")


def main():
    binary = os.path.abspath(sys.argv[1] if len(sys.argv) > 1 else "./sudoku")
    with tempfile.TemporaryDirectory(prefix="sudoku-api-e2e-") as root:
        process, base = start(binary, root, free_port(), ["--db", os.path.join(root, "sudoku.db")])
        try:
            expect(request(base, "GET", "/healthz")[0], 200, "health")
            for path in ("/api/v1/sessions", "/api/v1/sessions/legacy", "/api/v1/sessions/import"):
                expect(request(base, "GET", path)[0], 404, f"removed route {path}")
            status, body, _ = request(base, "POST", "/api/v1/guest/games", {"source": {"kind": "puzzle", "puzzle": PUZZLE}})
            expect(status, 503, "guest route without private configuration")
            expect(body["error"]["code"], "account-unavailable", "bounded unavailable response")
            expect(request(base, "POST", "/api/v1/guest/games", {}, headers={"Origin": ORIGIN})[2].get("Access-Control-Allow-Origin"), ORIGIN, "allowed origin")
            expect(request(base, "OPTIONS", "/api/v1/guest/games", headers={"Origin": "http://denied.example", "Access-Control-Request-Method": "POST"})[0], 403, "denied origin")
        finally:
            process.send_signal(signal.SIGTERM)
            process.wait(timeout=10)
            expect(process.returncode, 0, "graceful shutdown")

        unsafe = subprocess.run([binary, "api", "--listen", "0.0.0.0:0"], capture_output=True, text=True, timeout=5)
        if unsafe.returncode == 0 or "--listen must use a loopback address" not in unsafe.stderr:
            raise AssertionError("non-loopback startup was not rejected")

    print("HTTP API clean-cutover E2E passed")


if __name__ == "__main__":
    main()
