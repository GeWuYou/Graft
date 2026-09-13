#!/usr/bin/env python3
"""Regression tests for the deterministic skill-local test runner."""

from __future__ import annotations

import importlib.util
import contextlib
import io
from pathlib import Path
import subprocess
import sys
import tempfile
from unittest import mock
import unittest


SCRIPT_PATH = Path(__file__).with_name("run_skill_tests.py")
MODULE_SPEC = importlib.util.spec_from_file_location("run_skill_tests", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"Unable to load module from {SCRIPT_PATH}.")
MODULE = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = MODULE
MODULE_SPEC.loader.exec_module(MODULE)


class SkillTestRunnerTests(unittest.TestCase):
    def test_discovery_is_recursive_sorted_and_test_only(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_dir:
            skills_dir = Path(temporary_dir)
            first = skills_dir / "a-skill" / "scripts" / "test_first.py"
            second = skills_dir / "z-skill" / "tests" / "test_second.py"
            ignored = skills_dir / "z-skill" / "scripts" / "helper.py"
            for path in (first, second, ignored):
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("", encoding="utf-8")

            discovered = MODULE.discover_skill_tests(skills_dir)

        self.assertEqual(discovered, [first, second])

    def test_default_discovery_includes_manifest_plugin_tests(self) -> None:
        import json
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            repository_test = root / ".agents/skills/example/scripts/test_repo.py"
            plugin_root = root / "plugins/graft-frontend-vibe-toolchain"
            plugin_test = plugin_root / "skills/example/tests/test_plugin.py"
            manifest = plugin_root / ".codex-plugin/plugin.json"
            for path in (repository_test, plugin_test, manifest):
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(json.dumps({"skills": "./skills/"}) if path == manifest else "")
            with mock.patch.object(MODULE, "REPO_ROOT", root):
                self.assertEqual(MODULE.discover_skill_tests(), sorted([repository_test, plugin_test]))

    def test_invalid_plugin_manifest_fails_instead_of_running_partial_suite(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            with mock.patch.object(MODULE, "REPO_ROOT", Path(directory)):
                with self.assertRaisesRegex(RuntimeError, "plugin manifest"):
                    MODULE.discover_skill_tests()

    def test_discovery_rejects_test_symlink_outside_declared_root(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / "skills"
            root.mkdir()
            outside = Path(directory) / "untrusted.py"
            outside.write_text("raise RuntimeError('must not run')")
            (root / "test_external.py").symlink_to(outside)
            with self.assertRaisesRegex(RuntimeError, "escapes"):
                MODULE.discover_skill_tests(root)

    def test_runner_executes_all_files_and_combines_failures(self) -> None:
        calls: list[tuple[list[str], Path, bool]] = []

        def fake_runner(command: list[str], *, cwd: Path, check: bool) -> subprocess.CompletedProcess[str]:
            calls.append((command, cwd, check))
            return subprocess.CompletedProcess(command, 1 if command[-1].endswith("second.py") else 0)

        root = Path("/repo")
        paths = [root / "first.py", root / "second.py"]

        with contextlib.redirect_stdout(io.StringIO()):
            result = MODULE.run_skill_tests(paths, repo_root=root, runner=fake_runner)

        self.assertEqual(result, 1)
        self.assertEqual([call[0][-1] for call in calls], [str(path) for path in paths])
        self.assertTrue(all(call[1] == root and call[2] is False for call in calls))


if __name__ == "__main__":
    unittest.main()
