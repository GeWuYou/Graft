#!/usr/bin/env python3
"""Regression tests for plugin residual governance scanning."""

from __future__ import annotations

import importlib.util
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock


SCRIPT_PATH = Path(__file__).with_name("check_plugin_residuals.py")
MODULE_SPEC = importlib.util.spec_from_file_location("check_plugin_residuals", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"Unable to load module from {SCRIPT_PATH}.")

MODULE = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = MODULE
MODULE_SPEC.loader.exec_module(MODULE)


class PluginResidualTests(unittest.TestCase):
    def test_allowlist_contains_expected_historical_rule(self) -> None:
        rules = MODULE.load_allowlist()
        self.assertTrue(any(rule.path == "AGENTS.md" for rule in rules))
        self.assertTrue(any(rule.path_prefix == ".agents/skills/" for rule in rules))

    def test_skip_known_dependency_files(self) -> None:
        self.assertTrue(MODULE.should_skip("web/bun.lock"))
        self.assertTrue(MODULE.should_skip("ai-plan/public/archive/topic/README.md"))
        self.assertFalse(MODULE.should_skip("AGENTS.md"))

    def test_classify_accepts_historical_governance_line(self) -> None:
        rules = MODULE.load_allowlist()
        match = MODULE.Match(
            path="AGENTS.md",
            line_no=1,
            line="- early dynamic plugin hot-loading",
        )
        rule = MODULE.classify(match, rules)
        self.assertIsNotNone(rule)
        assert rule is not None
        self.assertEqual(rule.category, "historical_governance")

    def test_classify_rejects_uncategorized_line(self) -> None:
        rules = MODULE.load_allowlist()
        match = MODULE.Match(
            path="docs/example.md",
            line_no=3,
            line="plugin should not appear here as current authority",
        )
        rule = MODULE.classify(match, rules)
        self.assertIsNone(rule)

    def test_find_matches_skips_directory_symlink_and_non_regular_path(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_dir:
            root = Path(temporary_dir)
            target_dir = root / "target"
            target_dir.mkdir()
            (target_dir / "SKILL.md").write_text("plugin\n", encoding="utf-8")
            (root / "skills-link").symlink_to(target_dir, target_is_directory=True)
            (root / "plain-directory").mkdir()
            with mock.patch.object(MODULE, "REPO_ROOT", root):
                self.assertEqual(MODULE.find_matches("skills-link"), [])
                self.assertEqual(MODULE.find_matches("plain-directory"), [])


if __name__ == "__main__":
    unittest.main()
