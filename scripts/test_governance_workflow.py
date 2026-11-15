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

    def test_runtime_job_reuses_validated_outputs_and_checks_actual_checkout(self):
        job = self.jobs["local-runtime-acceptance"]
        self.assertEqual(set(job["needs"]), {"server-build-test", "web-check"})
        commands = "\n".join(step.get("run", "") for step in job["steps"])
        self.assertIn('source_revision="$(git rev-parse HEAD)"', commands)
        self.assertIn('verify --kind server', commands)
        self.assertIn('verify --kind web', commands)
        self.assertIn('--build-context "release-binary=', commands)
        self.assertIn("--source-revision", commands)
        self.assertIn("--execution-mode local", commands)
        self.assertIn("--web-image", commands)
        self.assertIn("official-install", commands)
        self.assertIn("ci_runtime_artifacts.py acceptance", commands)
        self.assertNotIn("bun run build", commands)
        self.assertNotIn("go test", commands)

    def test_runtime_gate_rejects_skips_and_evidence_upload_is_an_exact_allowlist(self):
        self.assertIn("local-runtime-acceptance", self.jobs["required-checks"]["needs"])
        result = self.aggregate({"local-runtime-acceptance": "skipped"}, "push")
        self.assertNotEqual(result.returncode, 0)
        job = self.jobs["local-runtime-acceptance"]
        cleanup = next(step for step in job["steps"] if step.get("name") == "Destroy only the registered acceptance project")
        self.assertEqual(cleanup["if"], "always()")
        self.assertIn('--confirm-project "$GRAFT_CI_PROJECT"', cleanup["run"])
        upload = next(step for step in job["steps"] if step.get("name") == "Upload sanitized runtime acceptance evidence")
        self.assertEqual(upload["if"], "always()")
        self.assertEqual(set(upload["with"]["path"].splitlines()), {
            ".local/docker-isolated/ci/api-evidence.json",
            ".local/docker-isolated/ci/source-assets.json",
            ".local/docker-isolated/ci/cleanup.json",
            ".local/docker-isolated/ci/official/api-evidence.json",
            ".local/docker-isolated/ci/official/installation-evidence.json",
            ".tmp/runtime-server/provenance.json",
            ".tmp/runtime-web/provenance.json",
            ".tmp/runtime-target-server/provenance.json",
            ".tmp/runtime-runner/provenance.json",
            ".tmp/runtime-release-source/provenance.json",
            ".local/docker-isolated/ci/release-api/update-api-evidence.json",
            ".tmp/runtime-build-registry/provenance.json",
            ".local/docker-isolated/ci/build-api/build-api-evidence.json",
        })
        self.assertTrue(upload["with"]["include-hidden-files"])

    def test_public_update_job_uses_versioned_artifacts_and_production_controller(self):
        job = self.jobs["local-runtime-acceptance"]
        commands = "\n".join(step.get("run", "") for step in job["steps"])
        for kind in ("target-server", "runner", "release-source"):
            self.assertIn("verify --kind " + kind, commands)
        self.assertIn('--build-context "build=$PWD/.tmp/runtime-runner"', commands)
        self.assertIn('--build-context "build=$PWD/.tmp/runtime-release-source"', commands)
        self.assertIn("--target-backend-image", commands)
        self.assertIn("--release-fixture-image", commands)
        self.assertIn("--runner-image", commands)
        self.assertLess(commands.index("official-install"), commands.index("update-api"))
        self.assertIn("ci_runtime_artifacts.py update-acceptance", commands)
        self.assertNotIn("controller-smoke", commands)
        backend = "\n".join(step.get("run", "") for step in self.jobs["server-build-test"]["steps"])
        self.assertIn("bash scripts/build_runtime_acceptance.sh", backend)
        self.assertEqual(backend.count("validate backend --stage buildtest --test-report"), 1)

    def test_tool_dependencies_are_declared_at_their_actual_consumers(self):
        package = json.loads((ROOT / "package.json").read_text())
        web = json.loads((ROOT / "web/package.json").read_text())
        self.assertEqual(package["devDependencies"]["@redocly/openapi-core"], "1.34.17")
        self.assertNotIn("@redocly/openapi-core", web["devDependencies"])
        self.assertEqual(web["devDependencies"]["yaml"], "2.9.1")
        bundler = (ROOT / "scripts/openapi-bundle.mjs").read_text()
        self.assertIn("requireFromRoot('@redocly/openapi-core')", bundler)

    def test_build_acceptance_precedes_update_and_reuses_authenticated_fixture_binary(self):
        commands = "\n".join(step.get("run", "") for step in self.jobs["local-runtime-acceptance"]["steps"])
        self.assertIn("verify --kind build-registry", commands)
        self.assertIn('--build-context "build=$PWD/.tmp/runtime-build-registry"', commands)
        self.assertIn("Dockerfile.build-registry", commands)
        self.assertIn("--build-registry-image", commands)
        self.assertLess(commands.index("official-install"), commands.index("build-api"))
        self.assertLess(commands.index("build-api"), commands.index("update-api"))
        self.assertIn("ci_runtime_artifacts.py build-acceptance", commands)
        backend = "\n".join(step.get("run", "") for step in self.jobs["server-build-test"]["steps"])
        self.assertIn("registry-auth/main_test.go", backend)
