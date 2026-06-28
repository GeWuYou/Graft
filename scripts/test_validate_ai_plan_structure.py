#!/usr/bin/env python3
"""Tests for the ai-plan structure guard."""

from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))

import validate_ai_plan_structure as validator


class ValidateAiPlanStructureTest(unittest.TestCase):
    def create_repo(self) -> tuple[tempfile.TemporaryDirectory[str], Path]:
        tmp = tempfile.TemporaryDirectory()
        root = Path(tmp.name)

        for relative in validator.REQUIRED_ROOT_FILES:
            self.write_file(root, relative, "# stub\n")
        for relative in validator.REQUIRED_TEMPLATE_FILES:
            self.write_file(root, relative, "# template\n")
        for relative in validator.EXPECTED_CATALOG_INCLUDED_PATHS:
            if relative not in validator.REQUIRED_ROOT_FILES:
                self.write_file(root, relative, "# catalog target\n")
        for topic in validator.ACTIVE_INDEX_GUARD_TOPICS:
            for relative in validator.topic_required_files(topic):
                self.write_file(root, relative, "# topic artifact\n")
        for topic in validator.ARCHIVED_TOPIC_GUARD_TOPICS:
            for relative in validator.topic_required_files(topic, archived=True):
                self.write_file(root, relative, "# topic artifact\n")

        self.write_file(
            root,
            "ai-plan/public/README.md",
            "\n".join(
                (
                    "# AI Plan Public Recovery Index",
                    "",
                    "## Active Topics",
                    "",
                    "- `compose-project-management`",
                    "  - Recovery entry: `ai-plan/public/compose-project-management/README.md`",
                    "",
                )
            ),
        )
        self.write_file(root, "ai-plan/AGENTS.md", "# local agents\n")
        self.write_file(root, "ai-plan/README.md", "# ai-plan readme\n")
        self.write_file(root, "ai-plan/catalog.json", self.catalog_json())

        return tmp, root

    def write_file(self, root: Path, relative: str, content: str) -> None:
        path = root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")

    def catalog_json(self) -> str:
        payload = {
            "schema_version": 1,
            "catalog_kind": "ai-plan-machine-index",
            "coverage": {
                "mode": "bounded-governance-set",
                "note": "Initial machine index for the ai-plan IA governance rollout only. This is not a whole-repository retrofit mandate.",
                "included_paths": list(validator.EXPECTED_CATALOG_INCLUDED_PATHS),
                "excluded_by_design": list(validator.EXPECTED_CATALOG_EXCLUDED_BY_DESIGN),
            },
            "authority_rules": dict(validator.EXPECTED_CATALOG_AUTHORITY_RULES),
            "entries": list(validator.EXPECTED_CATALOG_ENTRIES),
        }
        return json.dumps(payload, ensure_ascii=False, indent=2) + "\n"

    def test_accepts_current_adopted_structure(self) -> None:
        tmp, root = self.create_repo()
        self.addCleanup(tmp.cleanup)

        findings = validator.run_validation(root)

        self.assertEqual([], findings)

    def test_rejects_missing_compose_compatibility_file(self) -> None:
        tmp, root = self.create_repo()
        self.addCleanup(tmp.cleanup)
        (root / "ai-plan/public/compose-project-management/startup-prompt.md").unlink()

        findings = validator.run_validation(root)

        self.assertTrue(any("compose-project-management" in finding.message for finding in findings))

    def test_rejects_wrong_public_index_recovery_entry(self) -> None:
        tmp, root = self.create_repo()
        self.addCleanup(tmp.cleanup)
        index_path = root / "ai-plan/public/README.md"
        index_path.write_text(
            index_path.read_text(encoding="utf-8").replace(
                "ai-plan/public/compose-project-management/README.md",
                "ai-plan/public/compose-project-management/todos/compose-project-management-tracking.md",
            ),
            encoding="utf-8",
        )

        findings = validator.run_validation(root)

        self.assertTrue(any("must point at" in finding.message for finding in findings))

    def test_rejects_archived_topic_left_in_active_index(self) -> None:
        tmp, root = self.create_repo()
        self.addCleanup(tmp.cleanup)
        index_path = root / "ai-plan/public/README.md"
        index_path.write_text(
            index_path.read_text(encoding="utf-8").replace(
                "## Active Topics\n\n",
                "## Active Topics\n\n- `ai-plan-ia-governance`\n  - Recovery entry: `ai-plan/public/archive/ai-plan-ia-governance/README.md`\n",
            ),
            encoding="utf-8",
        )

        findings = validator.run_validation(root)

        self.assertTrue(any("must not remain in the active-topic index" in finding.message for finding in findings))

    def test_rejects_catalog_scope_expansion(self) -> None:
        tmp, root = self.create_repo()
        self.addCleanup(tmp.cleanup)
        catalog_path = root / "ai-plan/catalog.json"
        payload = json.loads(catalog_path.read_text(encoding="utf-8"))
        payload["coverage"]["included_paths"].append("ai-plan/public/compose-project-management/README.md")
        catalog_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

        findings = validator.run_validation(root)

        self.assertTrue(any("approved bounded governance set" in finding.message for finding in findings))

    def test_rejects_catalog_entry_extra_metadata(self) -> None:
        tmp, root = self.create_repo()
        self.addCleanup(tmp.cleanup)
        catalog_path = root / "ai-plan/catalog.json"
        payload = json.loads(catalog_path.read_text(encoding="utf-8"))
        payload["entries"][0]["extra"] = "unexpected"
        catalog_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

        findings = validator.run_validation(root)

        self.assertTrue(any("adds unapproved keys" in finding.message for finding in findings))


if __name__ == "__main__":
    unittest.main()
