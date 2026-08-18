#!/usr/bin/env python3
"""Regression tests for plugin residual governance scanning."""

from __future__ import annotations

import importlib.util
import os
from pathlib import Path
import subprocess
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
    def test_tracked_files_uses_nul_delimiters_for_special_filenames(self) -> None:
        completed = subprocess.CompletedProcess(
            args=["git", "ls-files", "-z"],
            returncode=0,
            stdout=b"normal.md\0line\nbreak.md\0back\\slash.md\0",
            stderr=b"",
        )
        with mock.patch.object(MODULE.subprocess, "run", return_value=completed) as run_mock:
            self.assertEqual(
                MODULE.tracked_files(),
                ["normal.md", "line\nbreak.md", "back\\slash.md"],
            )

        run_mock.assert_called_once_with(
            ["git", "ls-files", "-z"],
            cwd=MODULE.REPO_ROOT,
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

    def test_safe_relative_parts_follows_platform_path_semantics(self) -> None:
        self.assertEqual(
            MODULE.safe_relative_parts("back\\slash.md", platform="posix"),
            ("back\\slash.md",),
        )
        self.assertEqual(
            MODULE.safe_relative_parts("folder\\file.md", platform="nt"),
            ("folder", "file.md"),
        )
        self.assertIsNone(MODULE.safe_relative_parts("C:relative.md", platform="nt"))
        self.assertIsNone(MODULE.safe_relative_parts("\\rooted.md", platform="nt"))

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

    def test_find_matches_rejects_symlinked_parent_and_path_escape(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_dir:
            root = Path(temporary_dir) / "repo"
            outside = Path(temporary_dir) / "outside"
            root.mkdir()
            outside.mkdir()
            (outside / "SKILL.md").write_text("plugin\n", encoding="utf-8")
            (root / "skills-link").symlink_to(outside, target_is_directory=True)

            with mock.patch.object(MODULE, "REPO_ROOT", root):
                self.assertEqual(MODULE.find_matches("skills-link/SKILL.md"), [])
                self.assertEqual(MODULE.find_matches("../outside/SKILL.md"), [])
                self.assertEqual(MODULE.find_matches(str(outside / "SKILL.md")), [])

    @unittest.skipUnless(os.name == "posix", "POSIX permits backslashes in filenames")
    def test_find_matches_preserves_posix_backslash_filename(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_dir:
            root = Path(temporary_dir)
            filename = "back\\slash.md"
            (root / filename).write_text("plugin\n", encoding="utf-8")

            with mock.patch.object(MODULE, "REPO_ROOT", root):
                self.assertEqual(
                    MODULE.find_matches(filename),
                    [MODULE.Match(path=filename, line_no=1, line="plugin")],
                )


if __name__ == "__main__":
    unittest.main()
