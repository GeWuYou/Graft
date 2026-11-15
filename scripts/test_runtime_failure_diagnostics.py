from contextlib import redirect_stdout
import io
import json
from pathlib import Path
from types import SimpleNamespace
import tempfile
import unittest
from unittest.mock import patch

from scripts.ci_runtime_artifacts import failure_diagnostics, redact_diagnostic


class FailureDiagnosticsTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)
        self.state = {"project": "graft-isolated-unit", "source_revision": "a" * 40,
                      "environment": {"POSTGRES_PASSWORD": "generated-db-password", "GRAFT_AUTH_JWT_SECRET": "generated-jwt-secret"}}
        (self.directory / "instance.json").write_text(json.dumps(self.state))
        self.prefix = ["docker", "compose", "--project-name", self.state["project"]]
        fixture = SimpleNamespace(prefix=self.prefix, environment={"ONLY_REGISTERED_ENV": "true"})
        self.loader = patch("scripts.ci_runtime_artifacts.runpy.run_path", return_value={"Fixture": lambda _: fixture})
        self.loader.start()
        self.addCleanup(self.loader.stop)

    def test_database_failure_collects_exit_status_and_redacts_every_secret_source(self):
        official = self.directory / "official"
        official.mkdir()
        (official / ".env").write_text("GRAFT_AGENT_ENROLLMENT_PEPPER=generated-pepper-value\n")
        build = self.directory / "build-api"
        build.mkdir()
        (build / "credentials.json").write_text(json.dumps({"credentials": [{"password": "generated-registry-password"}]}))
        (build / "netrc").write_text("machine registry login fixture password generated-netrc-password\n")
        message = "migration permission denied generated-db-password generated-jwt-secret generated-pepper-value generated-registry-password generated-netrc-password"
        (self.directory / "command-error.log").write_text(message)
        calls = []

        def run(command, **kwargs):
            calls.append(command)
            self.assertEqual(kwargs["timeout"], 15)
            self.assertEqual(kwargs["env"], {"ONLY_REGISTERED_ENV": "true"})
            if "ps" in command:
                return SimpleNamespace(returncode=0, stdout=json.dumps({"Service": "migrate", "State": "exited", "ExitCode": 1}), stderr="")
            uri = "postgres" + "://" + "db:" + "unknown-uri-password" + "@postgres/db"
            return SimpleNamespace(returncode=0, stdout=message + " " + uri + " Authorization: Bearer unknown-bearer-token\n", stderr="")

        output = io.StringIO()
        with patch("scripts.ci_runtime_artifacts.subprocess.run", side_effect=run), redirect_stdout(output):
            failure_diagnostics(self.directory, self.directory)
        evidence = json.loads((self.directory / "failure-evidence.json").read_text())
        encoded = json.dumps(evidence) + output.getvalue()
        for secret in ["generated-db-password", "generated-jwt-secret", "generated-pepper-value", "generated-registry-password",
                       "generated-netrc-password", "unknown-uri-password", "unknown-bearer-token"]:
            self.assertNotIn(secret, encoded)
        self.assertIn("permission denied", encoded)
        self.assertFalse(evidence["acceptance_passed"])
        self.assertEqual(evidence["topologies"]["private"]["services"], [{"Service": "migrate", "State": "exited", "ExitCode": 1}])
        self.assertIn("official", evidence["topologies"])
        self.assertTrue(any("exec" in command and "docker-daemon" in command for command in calls))
        self.assertFalse(any("down" in command or "rm" in command for command in calls))
        self.assertEqual((self.directory / "command-error.log").read_text(), message)

    def test_permission_diagnostics_remain_bounded_and_private_keys_are_masked(self):
        message = "x" * 20000 + " permission denied\n-----BEGIN PRIVATE KEY-----\nprivate-material\n-----END PRIVATE KEY-----\npassword=unknown-password token=unknown-token Authorization: Basic c2VjcmV0OnBhc3N3b3Jk"
        redacted = redact_diagnostic(message, set())
        self.assertLessEqual(len(redacted), 8192)
        self.assertIn("permission denied", redacted)
        for secret in ["private-material", "unknown-password", "unknown-token", "c2VjcmV0OnBhc3N3b3Jk"]:
            self.assertNotIn(secret, redacted)

    def test_collection_failure_is_explicit_and_has_no_success_claim(self):
        (self.directory / "command-error.log").write_text("database unavailable")
        with patch("scripts.ci_runtime_artifacts.subprocess.run", side_effect=OSError("sensitive command details")), redirect_stdout(io.StringIO()):
            failure_diagnostics(self.directory, self.directory)
        proof = json.loads((self.directory / "failure-evidence.json").read_text())
        self.assertEqual(proof["topologies"]["private"]["status_exit_code"], 127)
        self.assertEqual(proof["topologies"]["private"]["status_error"], "OSError")
        self.assertFalse(proof["acceptance_passed"])
        self.assertNotIn("sensitive command details", json.dumps(proof))

    def test_unregistered_or_unrelated_project_cannot_query_docker(self):
        (self.directory / "instance.json").unlink()
        with patch("scripts.ci_runtime_artifacts.subprocess.run") as run, redirect_stdout(io.StringIO()):
            failure_diagnostics(self.directory, self.directory)
        run.assert_not_called()
        (self.directory / "instance.json").write_text(json.dumps({**self.state, "project": "unrelated-project"}))
        with self.assertRaisesRegex(ValueError, "registered isolated"):
            failure_diagnostics(self.directory, self.directory)
