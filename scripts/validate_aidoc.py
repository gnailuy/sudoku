#!/usr/bin/env python3
"""Validate the repository-owned AI-native documentation metadata contract."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path, PurePosixPath

REQUIRED_FIELDS = ("domain", "status", "entry_points", "dependencies")
VALID_STATUSES = {"Active", "Draft", "Deprecated"}


def _portable(path: str) -> bool:
    return bool(path) and "\\" not in path and not path.startswith(("/", "~")) and ".." not in PurePosixPath(path).parts


def _parse_frontmatter(path: Path, text: str, errors: list[str]) -> dict[str, object]:
    lines = text.splitlines()
    if not lines or lines[0] != "---":
        errors.append(f"{path}: missing YAML frontmatter")
        return {}
    try:
        end = lines.index("---", 1)
    except ValueError:
        errors.append(f"{path}: unterminated YAML frontmatter")
        return {}

    metadata: dict[str, object] = {}
    current_list: str | None = None
    for number, raw in enumerate(lines[1:end], start=2):
        if not raw.strip():
            continue
        item = re.match(r"^\s{2}-\s+(.+?)\s*$", raw)
        if item and current_list:
            value = metadata.setdefault(current_list, [])
            assert isinstance(value, list)
            value.append(item.group(1))
            continue
        field = re.match(r"^([a-z_]+):(?:\s*(.*))?$", raw)
        if not field:
            errors.append(f"{path}:{number}: unsupported frontmatter syntax")
            current_list = None
            continue
        key, value = field.group(1), (field.group(2) or "").strip()
        current_list = key if value == "" else None
        if value == "[]":
            metadata[key] = []
        elif value:
            metadata[key] = value
        else:
            metadata[key] = []
    return metadata


def validate(root: Path) -> list[str]:
    errors: list[str] = []
    aidoc = root / ".aidoc"
    if not aidoc.is_dir():
        return [f"{aidoc}: directory does not exist"]

    for path in sorted(aidoc.rglob("*.md")):
        relative = path.relative_to(root).as_posix()
        metadata = _parse_frontmatter(path, path.read_text(encoding="utf-8"), errors)

        for field in REQUIRED_FIELDS:
            if field not in metadata:
                errors.append(f"{relative}: missing frontmatter field {field}")
        domain = metadata.get("domain")
        if not isinstance(domain, str) or not domain:
            errors.append(f"{relative}: domain must be a non-empty string")
        status = metadata.get("status")
        if status not in VALID_STATUSES:
            errors.append(f"{relative}: status must be one of {sorted(VALID_STATUSES)}")
        for field in ("entry_points", "dependencies"):
            values = metadata.get(field)
            if values is not None and not isinstance(values, list):
                errors.append(f"{relative}: {field} must be a YAML list")
                continue
            for target in values or []:
                if not _portable(str(target)):
                    errors.append(f"{relative}: non-portable {field} path {target!r}")
                elif not (root / str(target)).exists():
                    errors.append(f"{relative}: missing {field} target {target!r}")

    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()
    errors = validate(args.root.resolve())
    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        print(f"AI-native documentation metadata validation failed with {len(errors)} error(s).", file=sys.stderr)
        return 1
    count = len(list((args.root / ".aidoc").rglob("*.md")))
    print(f"AI-native documentation metadata validation passed: {count} documents.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
