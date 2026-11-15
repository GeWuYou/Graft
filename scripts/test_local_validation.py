from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from scripts.local_validation import Check, execute, make_plan, relative_path, staged_divergence


class LocalValidationTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)

    def write(self, name, content=""):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)

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
