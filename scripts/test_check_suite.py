import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from scripts.check_suite import GROUPS, SuiteCheck, resolve_checks, run_checks


class CheckSuiteTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)

    def run_small_suite(self, checks, deadline=60.0, artifacts=None):
        return run_checks(checks, self.root, deadline, artifacts, ["probe"])

    def test_all_groups_resolve_and_unknown_group_fails(self):
        checks = resolve_checks(list(GROUPS))
        names = [check.name for check in checks]
        self.assertIn("ai-governance", names)
        self.assertIn("openapi-bundle", names)
        self.assertIn("server-buildtest", names)
        self.assertIn("compose-config", names)
        self.assertIn("web-lint", names)
        self.assertIn("web-build", names)
        with self.assertRaises(ValueError):
            resolve_checks(["nonexistent-group"])

    def test_requires_are_auto_included_across_groups(self):
        checks = resolve_checks(["core"])
        names = [check.name for check in checks]
        self.assertIn("openapi-bundle", names)
        self.assertLess(names.index("openapi-bundle"), names.index("server-buildtest"))

    def test_dependency_cycle_is_rejected(self):
        registry = {
            "g": (
                SuiteCheck("a", ["true"], requires=("b",), group="g"),
                SuiteCheck("b", ["true"], requires=("a",), group="g"),
            ),
        }
        with self.assertRaises(ValueError):
            resolve_checks(["g"], registry)

    def test_unknown_dependency_is_rejected(self):
        registry = {"g": (SuiteCheck("a", ["true"], requires=("missing",), group="g"),)}
        with self.assertRaises(ValueError):
            resolve_checks(["g"], registry)

    def test_independent_failures_continue_and_aggregate(self):
        checks = [
            SuiteCheck("bad", ["false"], group="probe"),
            SuiteCheck("good", ["true"], group="probe"),
        ]
        code, report = self.run_small_suite(checks)
        self.assertEqual(code, 1)
        statuses = {item["name"]: item["status"] for item in report["results"]}
        self.assertEqual(statuses, {"bad": "failed", "good": "passed"})
        self.assertEqual(report["summary"]["failed"], 1)
        self.assertEqual(report["summary"]["passed"], 1)

    def test_dependency_failure_skips_dependents(self):
        checks = [
            SuiteCheck("dep", ["false"], group="probe"),
            SuiteCheck("dependent", ["true"], requires=("dep",), group="probe"),
            SuiteCheck("after", ["true"], requires=("dependent",), group="probe"),
        ]
        code, report = self.run_small_suite(checks)
        self.assertEqual(code, 1)
        statuses = {item["name"]: item["status"] for item in report["results"]}
        self.assertEqual(statuses["dep"], "failed")
        self.assertEqual(statuses["dependent"], "skipped")
        self.assertEqual(statuses["after"], "skipped")
        skipped = next(item for item in report["results"] if item["name"] == "dependent")
        self.assertIn("dependency dep", skipped["reason"])

    def test_all_passing_suite_exits_zero(self):
        code, report = self.run_small_suite([SuiteCheck("ok", ["true"], group="probe")])
        self.assertEqual(code, 0)
        self.assertEqual(report["summary"], {"passed": 1, "failed": 0, "skipped": 0, "total": 1})

    def test_deadline_kills_running_check_and_marks_rest_failed(self):
        checks = [
            SuiteCheck("slow", [sys.executable, "-c", "import time; time.sleep(30)"], group="probe"),
            SuiteCheck("later", ["true"], group="probe"),
        ]
        code, report = self.run_small_suite(checks, deadline=1.0)
        self.assertEqual(code, 1)
        statuses = {item["name"]: item["status"] for item in report["results"]}
        self.assertEqual(statuses["slow"], "timeout")
        self.assertEqual(statuses["later"], "failed")
        self.assertIn("deadline", report["results"][1]["reason"])
        self.assertLess(report["elapsed_seconds"], 30)

    def test_artifacts_logs_and_gitignore_are_written(self):
        with tempfile.TemporaryDirectory() as artifacts:
            checks = [SuiteCheck(
                "loud", [sys.executable, "-c", "print('hello-from-check')"], group="probe",
            )]
            code, report = self.run_small_suite(checks, artifacts=Path(artifacts))
            self.assertEqual(code, 0)
            log = Path(artifacts) / "probe" / "loud.log"
            self.assertTrue(log.is_file())
            self.assertIn("hello-from-check", log.read_text())
            self.assertTrue((Path(artifacts) / ".gitignore").is_file())

    def test_dry_run_main_lists_plan_without_executing(self):
        from scripts.check_suite import main
        with tempfile.TemporaryDirectory() as artifacts:
            code = main(["--groups", "governance", "--dry-run", "--artifacts-dir", artifacts])
        self.assertEqual(code, 0)
        self.assertFalse((Path(artifacts)).exists())

    def test_main_list_groups_and_unknown_group_exit(self):
        from scripts.check_suite import main
        self.assertEqual(main(["--list-groups"]), 0)
        self.assertEqual(main(["--groups", "nope", "--dry-run"]), 2)


if __name__ == "__main__":
    unittest.main()
