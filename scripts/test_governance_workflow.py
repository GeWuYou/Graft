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
        self.assertEqual(self.jobs["server-build-test"]["timeout-minutes"], 25)
        steps = self.jobs["server-build-test"]["steps"]
        commands = [step.get("run", "") for step in steps]
        self.assertEqual(sum("validate backend --stage buildtest --test-report" in command for command in commands), 1)
        self.assertFalse(any("run_server_ctrf_report.py" in command for command in commands))
        self.assertTrue(any("Upload server test results" == step.get("name") for step in steps))

    def test_backend_validation_keeps_sqlite_cgo_and_static_builds_are_isolated(self):
        steps = self.jobs["server-build-test"]["steps"]
        validation = next(step for step in steps if step.get("name") == "Run repository backend build/test stage")
        self.assertEqual(validation["env"]["CGO_ENABLED"], "1")
        self.assertNotIn("GOOS", validation["env"])
        self.assertNotIn("GOARCH", validation["env"])
        static_build = next(step for step in steps if step.get("name") == "Link versioned runtime acceptance binaries")
        self.assertEqual(static_build["run"], "bash scripts/build_runtime_acceptance.sh")
        self.assertIn("export CGO_ENABLED=0 GOOS=linux GOARCH=amd64", (ROOT / "scripts/build_runtime_acceptance.sh").read_text())

    def test_secret_scan_preserves_complete_history_and_exact_result_policy(self):
        steps = self.jobs["secret-scan"]["steps"]
        checkout = next(step for step in steps if step.get("name") == "Checkout repository")
        self.assertEqual(checkout["with"]["fetch-depth"], 0)
        scanner = next(step for step in steps if "scripts/scan_secrets.py" in step.get("run", ""))
        self.assertIn("github.event.pull_request.base.sha", scanner["env"]["SCAN_BASE"])
        self.assertIn("github.event.pull_request.head.sha", scanner["env"]["SCAN_HEAD"])
        self.assertNotIn("continue-on-error", scanner)
        source = (ROOT / "scripts/scan_secrets.py").read_text()
        self.assertIn("--since-commit", source)
        self.assertIn("--branch", source)
        self.assertIn("--fail-on-scan-errors", source)
        self.assertIn("--exclude-paths=/tmp/exclude-paths.txt", source)
        self.assertIn("trufflehog@sha256:2ec535840f860aa27f11fee20222fb790326ecd66325c96bd49d02ec6af46eca", source)

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

    def test_release_smoke_declares_local_execution_and_isolated_storage(self):
        script = (ROOT / "scripts/smoke_release_compose.sh").read_text()
        fixture = script.split('cat > "${workspace}/.env" <<EOF\n', 1)[1].split("\nEOF", 1)[0]
        environment = dict(line.split("=", 1) for line in fixture.splitlines()
                           if line and not line.startswith("#"))
        self.assertEqual(environment["GRAFT_EXECUTION_MODE"], "local")
        self.assertEqual(environment["COMPOSE_PROFILES"], "")
        self.assertEqual(environment["GRAFT_APPLICATION_ROOT_HOST_PATH"], "${workspace}/apps")
        self.assertEqual(environment["GRAFT_APPLICATION_ROOT_CONTAINER_PATH"],
                         environment["GRAFT_APPLICATION_ROOT_HOST_PATH"])
        self.assertEqual(environment["GRAFT_UPDATE_STATE_VOLUME"], "${project}-update-state")
        self.assertEqual(environment["GRAFT_BUILD_SNAPSHOT_VOLUME"], "${project}-build-snapshots")

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
        diagnostic = next(step for step in job["steps"] if step.get("name") == "Collect sanitized failure diagnostics before cleanup")
        self.assertEqual(diagnostic["if"], "failure()")
        self.assertLess(job["steps"].index(diagnostic), job["steps"].index(cleanup))
        self.assertIn("failure-diagnostics", diagnostic["run"])
        self.assertIn('--confirm-project "$GRAFT_CI_PROJECT"', cleanup["run"])
        upload = next(step for step in job["steps"] if step.get("name") == "Upload sanitized runtime acceptance evidence")
        self.assertEqual(upload["if"], "always()")
        self.assertEqual(set(upload["with"]["path"].splitlines()), {
            ".local/docker-isolated/ci/api-evidence.json",
            ".local/docker-isolated/ci/source-assets.json",
            ".local/docker-isolated/ci/cleanup.json",
            ".local/docker-isolated/ci/failure-evidence.json",
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

    def test_summary_needs_cover_every_other_job(self):
        needs = set(self.jobs["required-checks"]["needs"])
        others = set(self.jobs) - {"required-checks"}
        self.assertEqual(needs, others)

    def test_summary_verifies_every_job_via_api_for_current_run_attempt_sha(self):
        job = self.jobs["required-checks"]
        steps = job["steps"]
        # The needs-based aggregate must stay the first step; it is the fast local gate.
        self.assertTrue(steps[0]["run"].startswith("python3 - <<'PY'"))
        cross = next(step for step in steps
                     if step.get("name") == "Verify every workflow job for the current run, attempt and SHA")
        env = cross["env"]
        self.assertEqual(env["RUN_ID"], "${{ github.run_id }}")
        self.assertEqual(env["RUN_ATTEMPT"], "${{ github.run_attempt }}")
        self.assertEqual(env["HEAD_SHA"], "${{ github.sha }}")
        self.assertEqual(env["EVENT_NAME"], "${{ github.event_name }}")
        self.assertEqual(env["GITHUB_TOKEN"], "${{ secrets.GITHUB_TOKEN }}")
        program = "\n".join(cross["run"].splitlines()[1:-1])
        self.assertIn('"gh", "api", "--method", "GET"', program)
        # -f query fields imply POST unless --method GET pins the jobs-list verb.
        self.assertNotIn('"-F"', program)
        self.assertIn('"-f", "per_page=100"', program)
        self.assertIn("attempts/{attempt}/jobs", program)
        self.assertIn('job.get("head_sha") != head_sha', program)
        self.assertIn('conclusion == "success"', program)
        self.assertIn('conclusion == "skipped" and name in allowed_skips', program)
        self.assertIn('allowed_skips = {"Secret Scan"} if event_name == "push" else set()', program)
        # No raw HTTP header literals or direct HTTP clients in the cross-check.
        self.assertNotIn("Authorization", cross["run"])
        self.assertNotIn("urllib", program)

    def test_workflow_has_no_hard_coded_authorization_header_literals(self):
        text = (ROOT / ".github/workflows/pull-request-validation.yml").read_text()
        self.assertNotIn("Authorization", text)
        self.assertNotIn("Bearer ", text)

    def test_every_step_job_declares_a_timeout(self):
        for name, job in self.jobs.items():
            if "steps" not in job:
                # Reusable workflow callers delegate their timeout to the called workflow.
                self.assertIn("uses", job, msg=f"job {name} has neither steps nor uses")
                continue
            with self.subTest(job=name):
                self.assertIn("timeout-minutes", job)

