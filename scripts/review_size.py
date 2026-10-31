#!/usr/bin/env python3
"""Emit non-blocking review signals for unusually large maintained Go sources."""

from __future__ import annotations

import argparse
from pathlib import Path

MAX_FILE_LINES = 1_000
MAX_LINE_LENGTH = 140
EXEMPT_PATHS = {
    Path("webapi/generated.go"),
    Path("message/english.go"),
}


def findings(root: Path) -> list[tuple[Path, int, str]]:
    result: list[tuple[Path, int, str]] = []
    for path in sorted(root.rglob("*.go")):
        relative = path.relative_to(root)
        if relative in EXEMPT_PATHS or "testdata" in relative.parts:
            continue
        lines = path.read_text(encoding="utf-8").splitlines()
        if len(lines) > MAX_FILE_LINES:
            result.append((relative, 1, f"file has {len(lines)} lines (review target: {MAX_FILE_LINES})"))
        long_lines = [(number, len(line)) for number, line in enumerate(lines, 1) if len(line) > MAX_LINE_LENGTH]
        if long_lines:
            number, width = max(long_lines, key=lambda item: item[1])
            result.append(
                (
                    relative,
                    number,
                    f"{len(long_lines)} long line(s); widest has {width} characters "
                    f"(review target: {MAX_LINE_LENGTH})",
                )
            )
    return result


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("root", nargs="?", type=Path, default=Path("."))
    args = parser.parse_args()
    issues = findings(args.root)
    for path, line, message in issues:
        print(f"::warning file={path},line={line}::{message}")
    if issues:
        print(f"Readability review: {len(issues)} non-blocking signal(s).")
        return 1
    print("Readability review: no size signals.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
