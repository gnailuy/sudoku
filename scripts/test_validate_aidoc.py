import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path

SPEC = importlib.util.spec_from_file_location(
    "validate_aidoc", Path(__file__).with_name("validate_aidoc.py")
)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class ValidateAIDocTest(unittest.TestCase):
    def fixture(self, root: Path) -> None:
        (root / ".aidoc" / "designs").mkdir(parents=True)
        (root / "src").mkdir()
        (root / "src" / "feature.go").write_text("package src\n", encoding="utf-8")
        (root / "AGENT.md").write_text("# Agent\n", encoding="utf-8")
        (root / ".aidoc" / "INDEX.md").write_text(
            """---
domain: Conventions
status: Active
entry_points:
dependencies:
---

# Documentation Index

This index registers project knowledge and provides bounded discovery routes.

## Related Docs

| Document | Relationship |
|---|---|
| `AGENT.md` | Repository instructions |

## Draft Work

No Draft documents are active.

## Registry

| Document | Purpose |
|---|---|
| `.aidoc/designs/feature.md` | Feature contract |
""",
            encoding="utf-8",
        )
        (root / ".aidoc" / "designs" / "feature.md").write_text(
            """---
domain: Designs
status: Active
entry_points:
  - src/feature.go
dependencies:
  - .aidoc/INDEX.md
---

# Feature Contract

This document preserves the feature invariant that code alone does not explain.

## Related Docs

| Document | Relationship |
|---|---|
| [Index](../INDEX.md) | Discovery entry point |

## Invariant

The feature remains bounded by its public entry point.
""",
            encoding="utf-8",
        )

    def test_valid_repository_passes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.fixture(root)
            self.assertEqual([], MODULE.validate(root))

    def test_invalid_repository_reports_contract_failures(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.fixture(root)
            feature = root / ".aidoc" / "designs" / "feature.md"
            broken = feature.read_text().replace("status: Active", "status: Historical").replace("## Related Docs", "## Neighbors")
            broken = broken.replace("dependencies:\n", "dependencies:\n### scenario accidentally inside frontmatter\n")
            broken += "\n[Missing neighbor](missing.md)\n"
            feature.write_text(broken, encoding="utf-8")
            index = root / ".aidoc" / "INDEX.md"
            index.write_text(index.read_text().replace("| `.aidoc/designs/feature.md` | Feature contract |\n", ""), encoding="utf-8")
            errors = MODULE.validate(root)
            self.assertTrue(any("unsupported frontmatter syntax" in error for error in errors))
            self.assertTrue(any("status must be" in error for error in errors))
            self.assertTrue(any("missing Related Docs" in error for error in errors))
            self.assertTrue(any("broken local link" in error for error in errors))
            self.assertTrue(any("unregistered document" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
