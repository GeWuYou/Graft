import unittest

from scripts.ci_impact import ALL_JOBS, select_jobs


class CiImpactTests(unittest.TestCase):
    def assert_full(self, files, reason_part):
        selection = select_jobs(files)
        self.assertEqual(selection["mode"], "full", msg=f"files={files}")
        self.assertEqual(sorted(selection["jobs"]), sorted(ALL_JOBS))
        if reason_part:
            self.assertTrue(
                any(reason_part in reason for reason in selection["full_reasons"]),
                msg=selection["full_reasons"],
            )

    def test_unknown_path_forces_full(self):
        self.assert_full(["assets/unknown.bin"], "unknown path")

    def test_workflow_change_forces_full(self):
        self.assert_full([".github/workflows/pull-request-validation.yml"], "workflow")

    def test_repository_automation_forces_full(self):
        self.assert_full([".github/ISSUE_TEMPLATE/bug.md"], "automation")

    def test_shared_automation_forces_full(self):
        self.assert_full(["scripts/local_validation.py"], "shared")

    def test_toolchain_and_image_surfaces_force_full(self):
        for path in ("Justfile", "server/go.mod", "bun.lock", "package.json", "turbo.json",
                     "Dockerfile", "compose.yml", "server/runner/compose/Dockerfile",
                     ".gitignore", ".editorconfig"):
            with self.subTest(path=path):
                self.assert_full([path], "")

    def test_empty_selection_forces_full(self):
        self.assert_full([], "")

    def test_server_change_selects_backend_subset(self):
        selection = select_jobs(["server/modules/task/runtime.go"])
        self.assertEqual(selection["mode"], "subset")
        self.assertTrue(set(selection["jobs"]) < set(ALL_JOBS))
        for job in ("server-lint", "server-build-test", "local-runtime-acceptance",
                    "cross-boundary-contract-freshness", "line-ending-check",
                    "secret-scan", "required-checks"):
            self.assertIn(job, selection["jobs"])
        self.assertNotIn("web-check", selection["jobs"])
        self.assertIn("core", selection["suite_groups"])

    def test_web_change_selects_frontend_subset(self):
        selection = select_jobs(["web/src/app.vue"])
        self.assertEqual(selection["mode"], "subset")
        self.assertIn("web-check", selection["jobs"])
        self.assertIn("local-runtime-acceptance", selection["jobs"])
        self.assertNotIn("server-lint", selection["jobs"])
        self.assertEqual(sorted(selection["suite_groups"]), ["compose", "frontend-artifacts", "frontend-source"])

    def test_openapi_change_selects_contract_jobs(self):
        selection = select_jobs(["openapi/openapi.yaml"])
        self.assertEqual(selection["mode"], "subset")
        for job in ("cross-boundary-contract-freshness", "openapi-breaking-check",
                    "server-build-test", "web-check"):
            self.assertIn(job, selection["jobs"])
        self.assertIn("openapi", selection["suite_groups"])

    def test_migration_sql_accumulates_backend_and_migration_gate(self):
        selection = select_jobs(["server/modules/task/migrations/20260101000000_task.sql"])
        self.assertEqual(selection["mode"], "subset")
        self.assertIn("migration-bootstrap", selection["jobs"])
        self.assertIn("server-lint", selection["jobs"])
        self.assertIn("compose", selection["suite_groups"])

    def test_governance_change_selects_governance_jobs(self):
        selection = select_jobs(["ai-plan/public/README.md", "docs/development/validation.md"])
        self.assertEqual(selection["mode"], "subset")
        self.assertIn("ai-governance-check", selection["jobs"])
        self.assertNotIn("server-lint", selection["jobs"])
        self.assertNotIn("web-check", selection["jobs"])

    def test_reasons_record_rule_evidence(self):
        selection = select_jobs(["server/modules/task/runtime.go"])
        reasons = {item["path"]: item["rule"] for item in selection["reasons"]}
        self.assertEqual(reasons["server/modules/task/runtime.go"], "backend-source")

    def test_mixed_changes_accumulate_targets(self):
        selection = select_jobs(["server/modules/task/runtime.go", "web/src/app.vue"])
        self.assertEqual(selection["mode"], "subset")
        self.assertIn("server-lint", selection["jobs"])
        self.assertIn("web-check", selection["jobs"])


if __name__ == "__main__":
    unittest.main()
