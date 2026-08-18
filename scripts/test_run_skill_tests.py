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
