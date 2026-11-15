from pathlib import Path
import json
import os
import subprocess
import sys
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[1]


class GovernanceWorkflowTests(unittest.TestCase):
    def setUp(self):
        self.jobs = yaml.safe_load((ROOT / ".github/workflows/pull-request-validation.yml").read_text())["jobs"]

    def test_server_tests_and_report_share_authoritative_job(self):
        self.assertNotIn("server-test-report", self.jobs)
        steps = self.jobs["server-build-test"]["steps"]
        commands = [step.get("run", "") for step in steps]
        self.assertEqual(sum("validate backend --stage buildtest --test-report" in command for command in commands), 1)
        self.assertFalse(any("run_server_ctrf_report.py" in command for command in commands))
        self.assertTrue(any("Upload server test results" == step.get("name") for step in steps))

    def aggregate(self, results, event="pull_request"):
        script = self.jobs["required-checks"]["steps"][0]["run"].splitlines()
        program = "\n".join(script[1:-1])
        return subprocess.run([sys.executable, "-c", program], env={
            **os.environ, "NEEDS_JSON": json.dumps({name: {"result": result} for name, result in results.items()}),
            "EVENT_NAME": event,
        }, capture_output=True, text=True)

    def test_summary_collects_independent_failures(self):
        result = self.aggregate({"web-check": "failure", "server-build-test": "cancelled"})
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("web-check", result.stderr)
        self.assertIn("server-build-test", result.stderr)

    def test_summary_allows_only_expected_skips(self):
        self.assertEqual(self.aggregate({"web-check": "success", "secret-scan": "skipped"}, "push").returncode, 0)
        self.assertNotEqual(self.aggregate({"secret-scan": "skipped"}).returncode, 0)
        self.assertNotEqual(self.aggregate({"web-check": "skipped"}, "push").returncode, 0)

    def test_workspace_install_uses_root_lock(self):
        for workflow in ["pull-request-validation.yml", "publish.yml"]:
            jobs = yaml.safe_load((ROOT / ".github/workflows" / workflow).read_text())["jobs"]
            for job in jobs.values():
                for step in job.get("steps", []):
                    if "bun install --frozen-lockfile" in step.get("run", ""):
                        self.assertNotIn("working-directory", step)
        self.assertFalse((ROOT / "web/bun.lock").exists())
