#!/usr/bin/env python3
"""Validate the repository-owned AI-native documentation contract."""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path, PurePosixPath

REQUIRED_FIELDS = ("domain", "status", "entry_points", "dependencies")
VALID_STATUSES = {"Active", "Draft", "Deprecated"}
LINK_RE = re.compile(r"\[[^\]]+\]\(([^)]+)\)")
CODE_PATH_RE = re.compile(r"`((?:\.aidoc|[A-Za-z0-9_.-]+)/(?:[^`\n]+))`")


@dataclass(frozen=True)
class Document:
    path: Path
    relative: str
    metadata: dict[str, object]
    body: str


def _portable(path: str) -> bool:
    return bool(path) and "\\" not in path and not path.startswith(("/", "~")) and ".." not in PurePosixPath(path).parts


def _parse_frontmatter(path: Path, text: str, errors: list[str]) -> tuple[dict[str, object], str]:
    lines = text.splitlines()
    if not lines or lines[0] != "---":
        errors.append(f"{path}: missing YAML frontmatter")
        return {}, text
    try:
        end = lines.index("---", 1)
    except ValueError:
        errors.append(f"{path}: unterminated YAML frontmatter")
        return {}, text

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
    return metadata, "\n".join(lines[end + 1 :]).lstrip("\n")


def _resolve(root: Path, document: Document, target: str) -> Path:
    del root
    target = target.split("#", 1)[0]
    return document.path.parent / target


def validate(root: Path) -> list[str]:
    errors: list[str] = []
    aidoc = root / ".aidoc"
    if not aidoc.is_dir():
        return [f"{aidoc}: directory does not exist"]

    documents: list[Document] = []
    for path in sorted(aidoc.rglob("*.md")):
        metadata, body = _parse_frontmatter(path, path.read_text(encoding="utf-8"), errors)
        relative = path.relative_to(root).as_posix()
        document = Document(path, relative, metadata, body)
        documents.append(document)

        for field in REQUIRED_FIELDS:
            if field not in metadata:
                errors.append(f"{relative}: missing frontmatter field {field}")
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

        body_lines = body.splitlines()
        title_index = next((i for i, line in enumerate(body_lines) if line.startswith("# ")), None)
        if title_index is None:
            errors.append(f"{relative}: missing level-one title")
        else:
            before_h2 = []
            for line in body_lines[title_index + 1 :]:
                if line.startswith("## "):
                    break
                if line.strip():
                    before_h2.append(line.strip())
            prose = [line for line in before_h2 if not line.startswith(("|", "[", "<"))]
            if not prose:
                errors.append(f"{relative}: missing orientation before first level-two heading")
        if "## Related Docs" not in body:
            errors.append(f"{relative}: missing Related Docs section")

        for raw_target in LINK_RE.findall(body):
            if raw_target.startswith(("http://", "https://", "mailto:", "#")):
                continue
            target = raw_target.strip().split()[0].strip("<>")
            resolved = _resolve(root, document, target)
            if not resolved.exists():
                errors.append(f"{relative}: broken local link {raw_target!r}")

    index_path = aidoc / "INDEX.md"
    index = index_path.read_text(encoding="utf-8") if index_path.exists() else ""
    registered = set(re.findall(r"`(\.aidoc/[^`]+\.md)`", index))
    expected = {document.relative for document in documents if document.relative != ".aidoc/INDEX.md"}
    for missing in sorted(expected - registered):
        errors.append(f".aidoc/INDEX.md: unregistered document {missing}")
    for extra in sorted(registered - expected):
        errors.append(f".aidoc/INDEX.md: registered document does not exist {extra}")

    if "## Draft Work" not in index:
        errors.append(".aidoc/INDEX.md: missing Draft Work section")
    drafts = {document.relative for document in documents if document.metadata.get("status") == "Draft"}
    draft_section = index.split("## Draft Work", 1)[1].split("\n## ", 1)[0] if "## Draft Work" in index else ""
    listed_drafts = set(re.findall(r"`(\.aidoc/[^`]+\.md)`", draft_section))
    for missing in sorted(drafts - listed_drafts):
        errors.append(f".aidoc/INDEX.md: Draft Work omits {missing}")
    for extra in sorted(listed_drafts - drafts):
        errors.append(f".aidoc/INDEX.md: Draft Work lists non-Draft {extra}")

    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()
    errors = validate(args.root.resolve())
    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        print(f"AI-native documentation validation failed with {len(errors)} error(s).", file=sys.stderr)
        return 1
    count = len(list((args.root / ".aidoc").rglob("*.md")))
    print(f"AI-native documentation validation passed: {count} documents.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
