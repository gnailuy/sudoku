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
  - AGENT.md
dependencies: []
---

This prose intentionally omits the normal document shape and contains a broken link.

[Missing prose target](missing.md)
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

{{PROSE_IS_NOT_METADATA}}
""",
            encoding="utf-8",
        )

    def test_valid_metadata_passes_without_inspecting_prose(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.fixture(root)
            self.assertEqual([], MODULE.validate(root))

    def test_invalid_metadata_reports_contract_failures(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.fixture(root)
            feature = root / ".aidoc" / "designs" / "feature.md"
            broken = feature.read_text(encoding="utf-8")
            broken = broken.replace("status: Active", "status: Historical")
            broken = broken.replace("entry_points:\n  - src/feature.go", "entry_points: src/feature.go")
            broken = broken.replace(".aidoc/INDEX.md", ".aidoc/missing.md")
            feature.write_text(broken, encoding="utf-8")
            index = root / ".aidoc" / "INDEX.md"
            index.write_text(index.read_text(encoding="utf-8").replace("domain: Conventions\n", ""), encoding="utf-8")

            errors = MODULE.validate(root)
            self.assertTrue(any("missing frontmatter field domain" in error for error in errors))
            self.assertTrue(any("domain must be a non-empty string" in error for error in errors))
            self.assertTrue(any("status must be" in error for error in errors))
            self.assertTrue(any("entry_points must be a YAML list" in error for error in errors))
            self.assertTrue(any("missing dependencies target" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
