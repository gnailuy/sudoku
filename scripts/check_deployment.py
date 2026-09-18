#!/usr/bin/env python3
"""Validate the portable Sudoku API service lifecycle contract."""

import configparser
import pathlib
import sys

SERVICE = pathlib.Path(__file__).parents[1] / "deploy" / "sudoku-api.service.example"


def require(actual: str | None, expected: str, label: str) -> None:
    if actual != expected:
        raise AssertionError(f"{label}: got {actual!r}, want {expected!r}")


def main() -> None:
    text = SERVICE.read_text(encoding="utf-8")
    forbidden = ("/home/", "test.gnailuy.com", "SUDOKU_AUTH", "--auth-token")
    for value in forbidden:
        if value in text:
            raise AssertionError(f"service contains forbidden deployment value: {value}")

    service = configparser.ConfigParser(interpolation=None, strict=True)
    service.optionxform = str
    service.read_string(text)

    require(service.get("Unit", "After", fallback=None), "network.target", "network ordering")
    require(service.get("Unit", "StartLimitIntervalSec", fallback=None), "30", "restart window")
    require(service.get("Unit", "StartLimitBurst", fallback=None), "3", "restart limit")
    require(service.get("Service", "Type", fallback=None), "simple", "service type")
    require(
        service.get("Service", "ExecStart", fallback=None),
        "%h/.local/lib/sudoku/current/backend/sudoku api --listen 127.0.0.1:8080",
        "generic home-relative release command",
    )
    require(service.get("Service", "WorkingDirectory", fallback=None), "%h/.local/share/sudoku", "persistent working directory")
    require(
        service.get("Service", "Environment", fallback=None),
        '"XDG_DATA_HOME=%h/.local/share" "XDG_STATE_HOME=%h/.local/state"',
        "persistent XDG roots",
    )
    require(service.get("Service", "Restart", fallback=None), "on-failure", "restart policy")
    require(service.get("Service", "RestartSec", fallback=None), "3s", "restart backoff")
    require(service.get("Service", "TimeoutStopSec", fallback=None), "10s", "shutdown budget")
    require(service.get("Service", "KillSignal", fallback=None), "SIGTERM", "shutdown signal")
    require(service.get("Service", "UMask", fallback=None), "0077", "private file mode")
    require(service.get("Service", "NoNewPrivileges", fallback=None), "true", "privilege boundary")
    require(service.get("Service", "PrivateTmp", fallback=None), "true", "temporary-file boundary")
    require(service.get("Install", "WantedBy", fallback=None), "default.target", "startup target")

    print('{"status":"ok","service":"deploy/sudoku-api.service.example"}')


if __name__ == "__main__":
    try:
        main()
    except (AssertionError, configparser.Error, OSError) as error:
        print(f"deployment service validation failed: {error}", file=sys.stderr)
        raise SystemExit(1) from error
