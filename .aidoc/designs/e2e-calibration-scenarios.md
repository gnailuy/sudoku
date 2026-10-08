---
domain: Designs
status: Active
entry_points:
  - cmd/calibrate.go
  - calibration/runner.go
  - difficultyaudit/audit.go
dependencies:
  - .aidoc/designs/e2e-test-scenarios.md
  - .aidoc/designs/difficulty-calibration.md
---

# E2E Calibration Scenarios

The calibration scenario catalog verifies immutable corpus preparation, resumable measurement, and commit-bound full-catalog audit output through the built calibration command.

## Related Docs

| Document | Relationship |
|----------|-------------|
| `.aidoc/designs/e2e-test-scenarios.md` | E2E discovery map, isolation rules, and automation entry points |
| `AGENT.md` | Required black-box verification discipline |

## Why This Boundary

Calibration evidence must be reproducible across runs and must never combine observations from different corpus identities. The black-box boundary verifies files, checkpoints, reports, and rejection behavior together.

## 3. Difficulty Measurement CLI

### 3.0 Prepare a Canonical Corpus
Create a candidate version 2 manifest with a name and ordered `puzzles` array. Include unique IDs, puzzle strings, source category and identifier, license and redistribution status, collection method, and exploratory/held-out split; generated records also carry generator metadata. Run `sudoku calibrate prepare --input <candidate> --output <manifest>`.

**Expected:** Zero notation and whitespace are normalized to 81-cell dot notation, each record receives a matching SHA-256 content hash, order is preserved, and an existing output is never overwritten. Duplicate normalized puzzle content or incomplete provenance exits non-zero.

### 3.1 Start and Resume an Immutable Corpus
Use the prepared version 2 manifest. Run `sudoku calibrate --manifest <path> --output <directory>` twice with the same paths.

**Expected:** The first run appends one observation per puzzle and writes a manifest-bound checkpoint plus JSON and Markdown reports. The reports include reproducibility count, source/split outcome groups, strategy-grade distributions, descriptive cross-grade score overlap where both grades exist, optional external source comparison, and generated-target measurements. The second run reports zero new observations, leaves `observations.jsonl` unchanged, and keeps `report.json` deterministic. The built-binary harness also verifies the completed checkpoint index and representative stratified fields.

### 3.2 Reject a Changed Manifest
Complete a measurement run, then change the manifest name, puzzle order, IDs, or puzzle text and reuse the existing output directory.

**Expected:** The command exits non-zero because existing observations are bound to the original exact manifest SHA-256. Existing observations remain unchanged.

### 3.3 Create an Immutable Full-Catalog Audit
Run `sudoku calibrate audit` with an independently rated version 2 manifest, a new output directory, the exact repository commit, and two workers. Retry with the same output directory.

**Expected:** The first run writes one manifest-ordered `evidence.jsonl` row per puzzle plus deterministic JSON and Markdown reports. The report binds the manifest hash, repository commit, solver configuration, schema, and runtime; representative distributions and correlation counts are present. The retry fails before replacing any artifact because audit output is immutable.
