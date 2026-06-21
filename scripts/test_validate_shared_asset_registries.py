#!/usr/bin/env python3
"""Tests for shared asset registry validation."""

from __future__ import annotations

import tempfile
import unittest
import sys
from pathlib import Path
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parent))

import validate_shared_asset_registries as validator


class ValidateSharedAssetRegistriesTest(unittest.TestCase):
    def write_registry(self, root: Path, name: str, content: str) -> Path:
        path = root / name
        path.write_text(content, encoding="utf-8")
        return path

    def valid_content(self) -> str:
        return """
schema_version: 1
updated_at: "2026-06-11"
assets:
  - name: "sample-asset"
    type: "frontend-utility"
    path: "existing"
    owner: "web/shared"
    purpose: "Shared sample asset."
    use_when:
      - "Need the sample."
    do_not_use_when:
      - "Not a sample."
    examples:
      - "existing/example.ts"
"""

    def test_accepts_valid_registry(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "existing").mkdir()
            (root / "existing" / "example.ts").write_text("export {};\n", encoding="utf-8")
            registry = self.write_registry(root, "registry.yaml", self.valid_content())

            with mock.patch.object(validator, "REPO_ROOT", root):
                findings = validator.validate_registry(registry, set())

        self.assertEqual([], findings)

    def test_rejects_missing_required_fields(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "existing").mkdir()
            registry = self.write_registry(
                root,
                "registry.yaml",
                """
schema_version: 1
updated_at: "2026-06-11"
assets:
  - name: "sample-asset"
    type: "frontend-utility"
    path: "existing"
""",
            )

            with mock.patch.object(validator, "REPO_ROOT", root):
                findings = validator.validate_registry(registry, set())

        self.assertTrue(any("missing required fields" in finding.message for finding in findings))

    def test_rejects_generated_only_authority_path(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            generated = root / "web" / "src" / "contracts" / "openapi" / "generated"
            generated.mkdir(parents=True)
            (generated / "schema.ts").write_text("export {};\n", encoding="utf-8")
            registry = self.write_registry(
                root,
                "registry.yaml",
                self.valid_content().replace("existing", "web/src/contracts/openapi/generated"),
            )

            with mock.patch.object(validator, "REPO_ROOT", root):
                findings = validator.validate_registry(registry, set())

        self.assertTrue(any("generated-only artifact" in finding.message for finding in findings))

    def test_rejects_generated_only_authority_path_when_generated_is_final_segment(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            generated = root / "web" / "src" / "contracts" / "openapi" / "generated"
            generated.mkdir(parents=True)
            registry = self.write_registry(
                root,
                "registry.yaml",
                """
schema_version: 1
updated_at: "2026-06-11"
assets:
  - name: "sample-asset"
    type: "frontend-utility"
    path: "web/src/contracts/openapi/generated"
    owner: "web/shared"
    purpose: "Shared sample asset."
    use_when:
      - "Need the sample."
    do_not_use_when:
      - "Not a sample."
""",
            )

            with mock.patch.object(validator, "REPO_ROOT", root):
                findings = validator.validate_registry(registry, set())

        self.assertTrue(any("generated-only artifact" in finding.message for finding in findings))

    def test_rejects_missing_registered_path(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            registry = self.write_registry(root, "registry.yaml", self.valid_content())

            with mock.patch.object(validator, "REPO_ROOT", root):
                findings = validator.validate_registry(registry, set())

        self.assertTrue(any("path does not exist" in finding.message for finding in findings))

    def test_rejects_forbidden_output_path(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            forbidden = root / ".ai" / "venv"
            forbidden.mkdir(parents=True)
            registry = self.write_registry(
                root,
                "registry.yaml",
                self.valid_content().replace("existing", ".ai/venv"),
            )

            with mock.patch.object(validator, "REPO_ROOT", root):
                findings = validator.validate_registry(registry, set())

        self.assertTrue(any("forbidden generated/local output path" in finding.message for finding in findings))


if __name__ == "__main__":
    unittest.main()
