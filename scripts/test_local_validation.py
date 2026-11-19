from pathlib import Path
import hashlib
import json
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from scripts.local_validation import (
    Check, compare_reports, execute, execute_with_results, main, make_plan,
    relative_path, staged_divergence,
)


class LocalValidationTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)

    def write(self, name, content=""):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)

    def init_repo(self):
        subprocess.run(["git", "init", "-q", "-b", "main"], cwd=self.root, check=True)
        subprocess.run(
            ["git", "-c", "user.email=t@example.test", "-c", "user.name=t",
             "commit", "--allow-empty", "-q", "-m", "init"],
            cwd=self.root, check=True,
        )

    def head(self):
        return subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=self.root).decode().strip()

    def test_docs_do_not_trigger_runtime_checks(self):
        plan = make_plan(self.root, ["docs/guide.md"], [])
        self.assertEqual([item["name"] for item in plan["checks"]], ["whitespace"])
        self.assertEqual(plan["gaps"], [])

    def test_empty_selection_is_incomplete(self):
        plan = make_plan(self.root, [], [])
        self.assertEqual(plan["checks"], [])
        self.assertEqual(execute(plan, self.root), 2)

    def test_go_tests_use_owning_and_explicit_consumer_packages(self):
        self.write("server/modules/task/runtime_test.go")
        self.write("server/modules/project/service.go")
        plan = make_plan(self.root, ["server/modules/project/service.go"], ["server/modules/task/runtime_test.go"])
        tests = next(item for item in plan["checks"] if item["name"] == "server-tests")
        self.assertEqual(tests["argv"], ["go", "test", "./modules/project", "./modules/task"])

    def test_web_selects_companion_and_styles(self):
        self.write("web/src/widget.vue")
        self.write("web/src/widget.test.ts")
        plan = make_plan(self.root, ["web/src/widget.vue"], [])
        self.assertEqual(plan["gaps"], [])
        tests = next(item for item in plan["checks"] if item["name"] == "web-tests")
        self.assertEqual(tests["argv"][-1], "src/widget.test.ts")
        self.assertIn("web-style", [item["name"] for item in plan["checks"]])

    def test_missing_tests_and_unknown_files_are_visible(self):
        plan = make_plan(self.root, ["web/src/widget.ts", "assets/binary.bin"], ["scripts/missing.py"])
        self.assertTrue(any("does not exist" in gap for gap in plan["gaps"]))
        self.assertTrue(any("unknown" in gap for gap in plan["gaps"]))

    def test_paths_cannot_escape_checkout(self):
        for name in ["../other/file", "/tmp/file", ""]:
            with self.assertRaises(ValueError):
                relative_path(self.root, name)

    def test_independent_checks_continue_after_failure(self):
        plan = {"checks": [vars(Check("first", ["one"])), vars(Check("second", ["two"]))], "gaps": []}
        with patch("scripts.local_validation.subprocess.run", side_effect=[subprocess.CompletedProcess([], 1), subprocess.CompletedProcess([], 0)]) as run:
            self.assertEqual(execute(plan, self.root), 1)
        self.assertEqual(run.call_count, 2)

    def test_partial_staging_is_detected(self):
        subprocess.run(["git", "init", "-q"], cwd=self.root, check=True)
        self.write("file.txt", "initial")
        subprocess.run(["git", "add", "file.txt"], cwd=self.root, check=True)
        self.write("file.txt", "different")
        self.assertEqual(staged_divergence(self.root, ["file.txt"]), ["file.txt"])

    def test_unrelated_explicit_test_cannot_cover_other_source_families(self):
        self.write("scripts/test_planner.py")
        for source in ["server/new/package.go", "web/src/new.ts", "scripts/new.mjs"]:
            with self.subTest(source=source):
                self.write(source)
                plan = make_plan(self.root, [source], ["scripts/test_planner.py"])
                self.assertTrue(plan["gaps"])
                self.assertTrue(any("consumer tests" in gap for gap in plan["gaps"]))

    def test_missing_or_unsupported_test_cannot_cover_source(self):
        self.write("server/new/package.go")
        for selected in ["server/new/missing_test.go", "server/new/package.go"]:
            with self.subTest(selected=selected):
                plan = make_plan(self.root, ["server/new/package.go"], [selected])
                self.assertTrue(any("Go consumer tests" in gap for gap in plan["gaps"]))

    def test_declared_matching_consumer_can_cover_deleted_go_package(self):
        self.write("server/consumer/use_test.go")
        plan = make_plan(self.root, ["server/removed/only.go"], ["server/consumer/use_test.go"])
        self.assertEqual(plan["gaps"], [])
        command = next(check["argv"] for check in plan["checks"] if check["name"] == "server-tests")
        self.assertEqual(command, ["go", "test", "./consumer"])

    def test_deleted_go_package_reports_gap_without_consumer(self):
        plan = make_plan(self.root, ["server/removed/only.go"], [])
        self.assertTrue(plan["gaps"])
        self.assertNotIn("server-tests", [check["name"] for check in plan["checks"]])

    def test_deleted_go_file_checks_existing_package(self):
        self.write("server/remaining/keep.go")
        self.write("server/remaining/keep_test.go")
        plan = make_plan(self.root, ["server/remaining/deleted.go"], [])
        self.assertEqual(plan["gaps"], [])
        command = next(check["argv"] for check in plan["checks"] if check["name"] == "server-tests")
        self.assertEqual(command, ["go", "test", "./remaining"])

    def test_deleted_web_test_requires_replacement_coverage(self):
        plan = make_plan(self.root, ["web/src/deleted.test.ts"], [])
        self.assertTrue(any("consumer tests" in gap for gap in plan["gaps"]))
        self.assertFalse(any("does not exist" in gap for gap in plan["gaps"]))
        self.assertNotIn("web-tests", [check["name"] for check in plan["checks"]])
        self.write("web/src/replacement.test.ts")
        replacement = make_plan(self.root, ["web/src/deleted.test.ts"], ["web/src/replacement.test.ts"])
        self.assertEqual(replacement["gaps"], [])
        tests = next(check["argv"] for check in replacement["checks"] if check["name"] == "web-tests")
        self.assertEqual(tests[-1], "src/replacement.test.ts")

    def test_deleted_python_or_node_test_requires_matching_replacement(self):
        self.write("scripts/test_node.py")
        for name in ["scripts/test-deleted.mjs", "scripts/test_deleted.py"]:
            with self.subTest(name=name):
                plan = make_plan(self.root, [name], [])
                self.assertTrue(plan["gaps"])
                self.assertEqual(len(plan["checks"]), 1)

    def test_multiple_locale_files_share_one_typecheck_and_keep_other_checks(self):
        self.write("web/src/widget.ts")
        self.write("web/src/widget.test.ts")
        files = [f"web/src/locales/locale{index}.json" for index in range(4)] + ["web/src/widget.ts"]
        plan = make_plan(self.root, files, [])
        self.assertEqual(plan["gaps"], [])
        typechecks = [check for check in plan["checks"] if check["name"] == "web-typecheck"]
        self.assertEqual(len(typechecks), 1)
        self.assertEqual(typechecks[0]["argv"], ["bun", "run", "typecheck"])
        names = [check["name"] for check in plan["checks"]]
        self.assertIn("web-format", names)
        self.assertIn("web-lint", names)
        self.assertIn("web-tests", names)
        self.assertEqual(len({(check["cwd"], tuple(check["argv"])) for check in plan["checks"]}), len(plan["checks"]))

    def test_plan_binds_head_and_digests(self):
        self.init_repo()
        self.write("docs/guide.md", "content")
        plan = make_plan(self.root, ["docs/guide.md"], [])
        self.assertEqual(plan["head"], self.head())
        expected = hashlib.sha256(b"content").hexdigest()
        self.assertEqual(plan["digests"], {"docs/guide.md": expected})

    def test_selection_records_reasons_and_coverage(self):
        self.write("web/src/widget.vue")
        self.write("web/src/widget.test.ts")
        plan = make_plan(self.root, ["web/src/widget.vue"], [])
        reasons = {item["file"]: item for item in plan["selection"]}
        self.assertEqual(reasons["web/src/widget.vue"]["reason"], "worktree diff vs HEAD (changed or untracked)")
        self.assertIn("web-format", reasons["web/src/widget.vue"]["covered_by"])
        self.assertIn("web-tests", reasons["web/src/widget.vue"]["covered_by"])
        self.assertEqual(reasons["web/src/widget.test.ts"]["reason"], "derived consumer test for web/src/widget.vue")
        self.assertEqual(reasons["web/src/widget.test.ts"]["covered_by"], ["web-tests"])

    def test_explicit_selection_reason_is_recorded(self):
        self.write("docs/guide.md")
        plan = make_plan(self.root, ["docs/guide.md"], [], "explicit --file selection")
        self.assertEqual(plan["selection"][0]["reason"], "explicit --file selection")

    def test_compare_reports_applies_to_unchanged_worktree(self):
        self.init_repo()
        self.write("docs/guide.md", "content")
        plan = make_plan(self.root, ["docs/guide.md"], [])
        comparison = compare_reports(plan, plan, self.root)
        self.assertEqual(comparison["verdict"], "applies")
        self.assertEqual(comparison["reasons"], [])
        self.assertEqual(comparison["files"], {"docs/guide.md": "match"})

    def test_compare_reports_detects_modified_and_deleted_files(self):
        self.write("docs/guide.md", "content")
        plan = make_plan(self.root, ["docs/guide.md"], [])
        self.write("docs/guide.md", "changed")
        comparison = compare_reports(plan, plan, self.root)
        self.assertEqual(comparison["verdict"], "stale")
        self.assertIn("Selected file modified since report: docs/guide.md", comparison["reasons"])
        (self.root / "docs/guide.md").unlink()
        deletion = compare_reports(plan, plan, self.root)
        self.assertIn("Selected file deleted since report: docs/guide.md", deletion["reasons"])

    def test_compare_reports_detects_head_move(self):
        self.init_repo()
        self.write("docs/guide.md")
        plan = make_plan(self.root, ["docs/guide.md"], [])
        other = make_plan(self.root, ["docs/guide.md"], [])
        moved = compare_reports({**plan, "head": "0" * 40}, other, self.root)
        self.assertEqual(moved["verdict"], "stale")
        self.assertTrue(any("Head moved" in reason for reason in moved["reasons"]))

    def test_compare_reports_requires_sha_and_digest_bindings(self):
        self.write("docs/guide.md")
        plan = make_plan(self.root, ["docs/guide.md"], [])
        for stripped in [{"files": ["docs/guide.md"]}, {"head": plan["head"]}]:
            comparison = compare_reports(stripped, plan, self.root)
            self.assertEqual(comparison["verdict"], "stale")
            self.assertTrue(any("regenerate" in reason for reason in comparison["reasons"]))

    def test_compare_reports_detects_uncovered_current_changes(self):
        self.write("docs/guide.md")
        plan = make_plan(self.root, ["docs/guide.md"], [])
        self.write("docs/other.md", "new")
        other = make_plan(self.root, ["docs/guide.md", "docs/other.md"], [])
        comparison = compare_reports(plan, other, self.root)
        self.assertEqual(comparison["verdict"], "stale")
        self.assertIn("Current selection is not covered by the report: docs/other.md", comparison["reasons"])

    def test_main_plan_fails_on_gaps_and_passes_clean_scope(self):
        self.init_repo()
        self.write("web/src/widget.ts", "value")
        with patch("scripts.local_validation.ROOT", self.root):
            self.assertEqual(main(["--plan", "--file", "web/src/widget.ts"]), 2)
            self.assertEqual(main(["--plan", "--file", "docs/guide.md"]), 0)

    def test_main_plan_fails_on_empty_selection(self):
        self.init_repo()
        with patch("scripts.local_validation.ROOT", self.root):
            self.assertEqual(main(["--plan", "--file", "docs/guide.md", "--file", "docs/guide.md"]), 0)
        with patch("scripts.local_validation.ROOT", self.root), patch(
            "scripts.local_validation.changed_files", return_value=[],
        ):
            self.assertEqual(main(["--plan"]), 2)

    def test_main_stale_report_blocks_execution(self):
        self.init_repo()
        self.write("docs/guide.md", "content")
        report_path = self.root / "previous.json"
        report_path.write_text(json.dumps({"head": self.head(), "digests": {"docs/guide.md": "0" * 64}}))
        with patch("scripts.local_validation.ROOT", self.root), patch(
            "scripts.local_validation.execute_with_results",
        ) as run:
            code = main(["--execute", "--file", "docs/guide.md", "--compare-report", str(report_path)])
        self.assertEqual(code, 2)
        run.assert_not_called()

    def test_main_execute_writes_reusable_check_report(self):
        self.init_repo()
        self.write("docs/guide.md", "content")
        report_path = self.root / "check-report.json"
        with patch("scripts.local_validation.ROOT", self.root):
            code = main(["--execute", "--file", "docs/guide.md", "--report-out", str(report_path)])
        self.assertEqual(code, 0)
        report = json.loads(report_path.read_text())
        self.assertEqual(report["head"], self.head())
        self.assertEqual([item["name"] for item in report["results"]], ["whitespace"])
        self.assertEqual(report["results"][0]["exit"], 0)

    def test_derived_python_test_is_bound_into_digests_and_selection(self):
        self.write("scripts/check_suite.py", "value")
        self.write("scripts/test_check_suite.py", "value")
        plan = make_plan(self.root, ["scripts/check_suite.py"], [])
        self.assertIn("scripts/test_check_suite.py", plan["digests"])
        selection = {item["file"]: item for item in plan["selection"]}
        self.assertIn("scripts/test_check_suite.py", selection)
        self.assertEqual(selection["scripts/test_check_suite.py"]["reason"],
                         "derived consumer test for scripts/check_suite.py")

    def test_derived_node_test_is_bound_into_digests(self):
        self.write("scripts/probe.mjs")
        self.write("scripts/test-probe.mjs")
        plan = make_plan(self.root, ["scripts/probe.mjs"], [])
        self.assertIn("scripts/test-probe.mjs", plan["digests"])
        self.assertIn("scripts/test-probe.mjs", plan["tests"])

    def test_toolchain_selection_binds_declared_consumers(self):
        self.write("package.json")
        self.write("scripts/test-turbo.mjs")
        self.write("scripts/test_local_validation.py")
        plan = make_plan(self.root, ["package.json"], [])
        self.assertIn("scripts/test-turbo.mjs", plan["digests"])
        self.assertIn("scripts/test_local_validation.py", plan["digests"])
        self.assertIn("scripts/test-turbo.mjs", plan["tests"])
        self.assertIn("scripts/test_local_validation.py", plan["tests"])

    def test_execute_report_round_trip_covers_derived_tests(self):
        self.init_repo()
        self.write("scripts/check_suite.py", "value")
        self.write("scripts/test_check_suite.py",
                   "import unittest\n\n\nclass DerivedTests(unittest.TestCase):\n"
                   "    def test_placeholder(self):\n        self.assertTrue(True)\n")
        report_path = self.root / "check-report.json"
        with patch("scripts.local_validation.ROOT", self.root):
            self.assertEqual(main(["--execute", "--file", "scripts/check_suite.py",
                                   "--report-out", str(report_path)]), 0)
            report = json.loads(report_path.read_text())
            self.assertIn("scripts/test_check_suite.py", report["digests"])
            self.assertEqual(main(["--plan", "--file", "scripts/check_suite.py",
                                   "--compare-report", str(report_path)]), 0)
        self.write("scripts/test_check_suite.py", "modified")
        with patch("scripts.local_validation.ROOT", self.root):
            self.assertEqual(main(["--plan", "--file", "scripts/check_suite.py",
                                   "--compare-report", str(report_path)]), 2)

    def test_execute_with_results_reports_each_check(self):
        plan = {"checks": [vars(Check("first", ["one"])), vars(Check("second", ["two"]))], "gaps": []}
        with patch("scripts.local_validation.subprocess.run", side_effect=[
            subprocess.CompletedProcess([], 0), subprocess.CompletedProcess([], 3),
        ]) as run:
            code, results = execute_with_results(plan, self.root)
        self.assertEqual(code, 1)
        self.assertEqual([item["exit"] for item in results], [0, 3])
        self.assertEqual(run.call_count, 2)
