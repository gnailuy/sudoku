#!/usr/bin/env python3
"""Black-box acceptance for the guest-first HTTP runtime boundary."""

import base64
import json
import os
import signal
import socket
import stat
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


def environment(root):
    return dict(
        os.environ,
        XDG_DATA_HOME=os.path.join(root, "data"),
        XDG_STATE_HOME=os.path.join(root, "state"),
    )


def start(binary, root, port, extra_args=None):
    command = [binary, "api", "--listen", f"127.0.0.1:{port}", "--allowed-origin", ORIGIN]
    command.extend(extra_args or [])
    process = subprocess.Popen(command, env=environment(root), stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
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
    expect(process.returncode, 0, "graceful shutdown")


def expect(actual, expected, message):
    if actual != expected:
        raise AssertionError(f"{message}: got {actual!r}, want {expected!r}")


def assert_accounts_disabled(base):
    status, body, _ = request(base, "GET", "/api/v1/account")
    expect(status, 503, "account route without account configuration")
    expect(body["error"]["code"], "account-unavailable", "bounded account unavailable response")
    descriptor = body["error"]["message"]
    expect(descriptor["id"], "error.account-unavailable", "stable unavailable descriptor")
    expect(descriptor["arguments"], [], "unavailable descriptor arguments")
    expect(bool(descriptor["text"]), True, "unavailable descriptor fallback")


def write_private_json(path, value):
    with open(path, "w", encoding="utf-8") as stream:
        json.dump(value, stream)
    os.chmod(path, 0o600)


def main():
    binary = os.path.abspath(sys.argv[1] if len(sys.argv) > 1 else "./sudoku")
    with tempfile.TemporaryDirectory(prefix="sudoku-api-e2e-") as root:
        database = os.path.join(root, "sudoku.db")
        process, base = start(binary, root, free_port(), ["--db", database])
        try:
            expect(request(base, "GET", "/healthz")[0], 200, "health")
            for path in ("/api/v1/sessions", "/api/v1/sessions/legacy", "/api/v1/sessions/import"):
                expect(request(base, "GET", path)[0], 404, f"removed route {path}")
            status, guest, _ = request(base, "POST", "/api/v1/guest/games", {"source": {"kind": "puzzle", "puzzle": PUZZLE}})
            expect(status, 201, "zero-config guest create")
            assert_accounts_disabled(base)
            localized_status, localized_body, _ = request(
                base,
                "GET",
                "/api/v1/account",
                headers={"Accept-Language": "zh-Hant"},
            )
            baseline_status, baseline_body, _ = request(base, "GET", "/api/v1/account")
            expect(localized_status, baseline_status, "Accept-Language status invariance")
            expect(localized_body, baseline_body, "Accept-Language descriptor invariance")
            expect(request(base, "POST", "/api/v1/guest/games", {}, headers={"Origin": ORIGIN})[2].get("Access-Control-Allow-Origin"), ORIGIN, "allowed origin")
            expect(request(base, "OPTIONS", "/api/v1/guest/games", headers={"Origin": "http://denied.example", "Access-Control-Request-Method": "POST"})[0], 403, "denied origin")
        finally:
            stop(process)

        key_path = os.path.join(root, "data", "sudoku", "guest-sealing.key")
        expect(os.path.getsize(key_path), 32, "managed guest key size")
        expect(stat.S_IMODE(os.stat(key_path).st_mode), 0o600, "managed guest key mode")

        process, base = start(binary, root, free_port(), ["--db", database])
        try:
            status, applied, _ = request(
                base,
                "POST",
                "/api/v1/guest/games/actions",
                {"document": guest["document"], "action": {"kind": "set-value", "expected_revision": 0, "row": 1, "column": 1, "value": 4}},
            )
            expect(status, 200, "managed key survives restart")
            expect(applied["revision"], 1, "restarted guest revision")
            assert_accounts_disabled(base)
        finally:
            stop(process)

        guest_config = os.path.join(root, "guest-only.json")
        write_private_json(
            guest_config,
            {
                "guest": {
                    "active_key_id": "operator-v1",
                    "keys": {"operator-v1": base64.urlsafe_b64encode(bytes(range(32))).decode().rstrip("=")},
                    "lifetime_seconds": 86400,
                }
            },
        )
        process, base = start(binary, root, free_port(), ["--db", database, "--web-config", guest_config])
        try:
            expect(request(base, "POST", "/api/v1/guest/games", {"source": {"kind": "puzzle", "puzzle": PUZZLE}})[0], 201, "configured guest-only create")
            assert_accounts_disabled(base)
        finally:
            stop(process)

        account_only = os.path.join(root, "account-only.json")
        write_private_json(
            account_only,
            {
                "account": {
                    "csrf_key": base64.urlsafe_b64encode(bytes(range(32))).decode().rstrip("="),
                    "google": {"client_id": "client", "client_secret": "secret", "redirect_url": "https://client.example/callback"},
                    "return_path_prefix": "/",
                    "cookie_path": "/",
                    "session_idle_seconds": 3600,
                    "session_absolute_seconds": 7200,
                }
            },
        )
        rejected = subprocess.run([binary, "api", "--listen", "127.0.0.1:0", "--web-config", account_only], env=environment(root), capture_output=True, text=True, timeout=5)
        if rejected.returncode == 0 or "requires a complete guest section" not in rejected.stderr:
            raise AssertionError("account-only configuration was not rejected before startup")

        os.chmod(key_path, 0o644)
        unsafe_key = subprocess.run([binary, "api", "--listen", "127.0.0.1:0"], env=environment(root), capture_output=True, text=True, timeout=5)
        if unsafe_key.returncode == 0 or "managed guest key must be a regular owner-only file" not in unsafe_key.stderr:
            raise AssertionError("unsafe managed key permissions were not rejected")

        unsafe = subprocess.run([binary, "api", "--listen", "0.0.0.0:0"], env=environment(root), capture_output=True, text=True, timeout=5)
        if unsafe.returncode == 0 or "--listen must use a loopback address" not in unsafe.stderr:
            raise AssertionError("non-loopback startup was not rejected")

        help_result = subprocess.run([binary, "api", "--help"], capture_output=True, text=True, timeout=5)
        if help_result.returncode != 0 or "--web-config" not in help_result.stdout or "--accounts-config" in help_result.stdout:
            raise AssertionError("API help did not expose the direct --web-config cutover")

    print("HTTP API guest-first E2E passed")


if __name__ == "__main__":
    main()
