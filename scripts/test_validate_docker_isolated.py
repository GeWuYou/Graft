"""Regression checks for the private Docker boundary in the conformance overlay."""
import copy
import unittest
import json
import io
from urllib.error import HTTPError
from pathlib import Path
import tempfile
import sys
import subprocess
from unittest.mock import Mock, patch
sys.path.insert(0, str(Path(__file__).resolve().parent))

from validate_docker_isolated import Fixture, validate_topology


class PrivateDockerTopologyTests(unittest.TestCase):
    def setUp(self):
        self.configuration = {"services": {}}
        for name in ("docker-daemon", "backend", "conformance-driver", "docker-runtime-agent"):
            self.configuration["services"][name] = {"volumes": [
                {"type": "volume", "source": "daemon-socket", "target": "/run"},
                {"type": "volume", "source": "application-workspaces", "target": "/opt/graft/apps"},
            ]}

    def test_accepts_private_socket_and_consistent_workspace(self):
        validate_topology(self.configuration)

    def test_rejects_host_socket_retained_by_compose_merge(self):
        configuration = copy.deepcopy(self.configuration)
        configuration["services"]["backend"]["volumes"].append({
            "type": "bind", "source": "/var/run/docker.sock", "target": "/var/run/docker.sock"})
        with self.assertRaisesRegex(RuntimeError, "host Docker socket"):
            validate_topology(configuration)

    def test_rejects_private_socket_missing_from_driver(self):
        self.configuration["services"]["conformance-driver"]["volumes"].pop(0)
        with self.assertRaisesRegex(RuntimeError, "private daemon socket"):
            validate_topology(self.configuration)

    def test_rejects_daemon_and_agent_workspace_path_drift(self):
        self.configuration["services"]["docker-runtime-agent"]["volumes"][1]["target"] = "/workspace"
        with self.assertRaisesRegex(RuntimeError, "workspace mapping"):
            validate_topology(self.configuration)

    def test_rejects_readonly_parent_hiding_nested_mountpoints(self):
        self.configuration["services"]["backend"]["volumes"][0]["read_only"] = True
        with self.assertRaisesRegex(RuntimeError, "nested secret mountpoints"):
            validate_topology(self.configuration)


class FixtureLifecycleTests(unittest.TestCase):
    def test_waits_for_successful_one_shot_jobs_separately(self):
        with tempfile.TemporaryDirectory() as temporary:
            fixture = Fixture.__new__(Fixture)
            fixture.directory = Path(temporary)
            fixture.environment = {"GRAFT_CONFORMANCE_AGENT_IMAGE_DIGEST": "sha256:" + "a" * 64}
            topology = PrivateDockerTopologyTests()
            topology.setUp()
            def compose_result(*args):
                if args[0] == "config":
                    return json.dumps(topology.configuration)
                if args[:4] == ("exec", "-T", "vault", "sh"):
                    return '{"approle_login":true,"pki_ca_matches_delivery":true,"ca_sha256":"proof"}'
                return ""
            fixture.compose = Mock(side_effect=compose_result)
            fixture.driver = Mock(return_value={"agent_id": "fixture", "target_id": 1,
                "identity_id": "identity", "generation": 1, "ledger_receipt_count": 2})
            fixture.engine = Mock(return_value="private-engine")
            fixture.up()
            calls = [call.args for call in fixture.compose.call_args_list]
            self.assertIn(("wait", "workspace-init", "vault-init"), calls)
            self.assertIn(("wait", "migrate"), calls)
            vault_check = next(args for args in calls if args[:4] == ("exec", "-T", "vault", "sh"))
            self.assertIn("auth/approle/login", vault_check[-1])
            self.assertIn("pki/cert/ca", vault_check[-1])
            self.assertLess(calls.index(vault_check), calls.index(("up", "-d", "--no-deps", "migrate")))
            readiness = [args for args in calls if "--wait" in args]
            self.assertEqual(readiness, [("up", "-d", "--wait", "docker-daemon", "postgres", "redis", "vault"),
                                         ("up", "-d", "--no-deps", "--wait", "backend")])
            self.assertTrue((fixture.directory / "prepared-evidence.json").exists())
            self.assertFalse(any("--once" in args for args in calls))
            self.assertEqual(calls.count(("up", "-d", "--no-deps", "docker-runtime-agent")), 2)
            self.assertIn(("stop", "docker-runtime-agent"), calls)
            verification_calls = [call.args for call in fixture.driver.call_args_list if "verify-restart" in call.args]
            self.assertEqual(verification_calls, [("--phase", "verify-restart", "--agent-id", "fixture")])

    def test_stop_preserves_vault_and_stops_all_other_declared_services(self):
        fixture = Fixture.__new__(Fixture)
        services = {name: {} for name in ("vault", "backend", "docker-runtime-agent", "postgres", "redis", "docker-daemon", "vault-init")}
        fixture.compose = Mock(side_effect=[json.dumps({"services": services}), ""])
        fixture.stop()
        command = fixture.compose.call_args_list[-1].args
        self.assertEqual(command[0], "stop")
        self.assertEqual(set(command[1:]), set(services) - {"vault"})

    def test_stop_with_only_vault_never_expands_to_stop_all(self):
        fixture = Fixture.__new__(Fixture)
        fixture.compose = Mock(return_value='{"services":{"vault":{}}}')
        fixture.stop()
        self.assertEqual(fixture.compose.call_count, 1)

    def test_lost_vault_authority_aborts_before_migrations_or_backend(self):
        with tempfile.TemporaryDirectory() as temporary:
            fixture = Fixture.__new__(Fixture)
            fixture.directory = Path(temporary)
            topology = PrivateDockerTopologyTests()
            topology.setUp()
            def compose_result(*args):
                if args[0] == "config":
                    return json.dumps(topology.configuration)
                if args[:4] == ("exec", "-T", "vault", "sh"):
                    raise RuntimeError("current Vault rejected stored AppRole")
                return ""
            fixture.compose = Mock(side_effect=compose_result)
            fixture.driver = Mock()
            with self.assertRaisesRegex(RuntimeError, "Vault rejected"):
                fixture.up()
            calls = [call.args for call in fixture.compose.call_args_list]
            self.assertNotIn(("up", "-d", "--no-deps", "migrate"), calls)
            self.assertFalse(any("backend" in args for args in calls))
            fixture.driver.assert_not_called()

    def test_reauthenticates_after_empty_password_change_response(self):
        fixture = Fixture.__new__(Fixture)
        fixture.environment = {"GRAFT_FIXTURE_ADMIN_PASSWORD": "synthetic-password"}
        fixture.api = Mock(side_effect=[(401, {}),
            (200, {"data": {"access_token": "restricted", "must_change_password": True}}),
            (200, {"data": None}), (200, {"data": {"access_token": "normal"}})])
        fixture.authenticate()
        self.assertEqual(fixture.token, "normal")


class FixtureAPIContractTests(unittest.TestCase):
    def fixture(self, directory):
        fixture = Fixture.__new__(Fixture)
        fixture.directory = Path(directory)
        fixture.environment = {"GRAFT_CONFORMANCE_BACKEND_PORT": "28080"}
        fixture.token = "synthetic-fixture-token"
        return fixture

    def test_operation_posts_supply_and_reuse_caller_idempotency_key(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            response = Mock()
            response.status = 202
            response.read.return_value = b'{"data":{"task_id":1}}'
            response.__enter__ = Mock(return_value=response)
            response.__exit__ = Mock(return_value=False)
            with patch("validate_docker_isolated.request.urlopen", return_value=response) as transport:
                fixture.api("POST", "/api/ops/containers/probe/start", {}, idempotency_key="same-operation")
                fixture.api("POST", "/api/ops/containers/probe/start", {}, idempotency_key="same-operation")
                fixture.api("POST", "/api/ops/docker/images/pull", {"reference": "alpine"})
            headers = [dict(call.args[0].header_items()) for call in transport.call_args_list]
            self.assertEqual(headers[0]["Idempotency-key"], "same-operation")
            self.assertEqual(headers[1]["Idempotency-key"], "same-operation")
            self.assertEqual(len(headers[2]["Idempotency-key"]), 32)

    def test_nonblocking_command_preserves_failure_exit_code(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            fixture.prefix = ["docker", "compose"]
            fixture.environment = {}
            result = Mock(returncode=7, stdout="", stderr="socket connection denied")
            with patch("validate_docker_isolated.subprocess.run", return_value=result):
                self.assertEqual(fixture.compose("exec", "backend", "curl", check=False), "")
            self.assertEqual(fixture.last_command_returncode, 7)

    def test_failed_http_evidence_excludes_request_payload_and_response_details(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            failure = HTTPError("http://fixture", 400, "Bad Request", {},
                io.BytesIO(b'{"code":"INVALID","traceId":"safe-trace","message":"sensitive provider detail"}'))
            with patch("validate_docker_isolated.request.urlopen", side_effect=failure):
                fixture.api("POST", "/api/ops/containers/probe/start", {"secret": "sensitive payload"})
            evidence = (Path(directory) / "api-errors.json").read_text()
            self.assertIn("safe-trace", evidence)
            self.assertIn("INVALID", evidence)
            self.assertNotIn("sensitive", evidence)


class RuntimeRepairEvidenceTests(unittest.TestCase):
    def fixture(self, directory):
        fixture = Fixture.__new__(Fixture)
        fixture.directory = Path(directory)
        fixture.state = {"project": "graft-isolated-test"}
        fixture.environment = {}
        fixture.compose = Mock(return_value="")
        fixture.record = Mock()
        return fixture

    def identity(self, receipts):
        return {"agent_id": "fixture", "target_id": 1, "identity_id": "identity", "generation": 2,
                "ledger_receipt_count": receipts}

    def test_once_requires_successful_exit_and_advanced_receipt(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            (Path(directory) / "trust-evidence.json").write_text(json.dumps({"restart": self.identity(1)}))
            fixture.wait_driver = Mock(side_effect=[self.identity(2), self.identity(3)])
            with patch("validate_docker_isolated.subprocess.run") as cleanup:
                fixture.once_probe()
            fixture.record.assert_called_once()
            self.assertEqual(fixture.record.call_args.args, ("agent-once-exit", "passed"))
            calls = fixture.compose.call_args_list
            one_shot = next(call for call in calls if "--once" in call.args)
            self.assertEqual(one_shot.kwargs["timeout"], 60)
            self.assertEqual(calls[-1].args, ("up", "-d", "--no-deps", "docker-runtime-agent"))
            self.assertEqual(cleanup.call_args.args[0], ["docker", "rm", "-f", "graft-isolated-test-once-probe"])
            self.assertTrue(fixture.record.call_args.kwargs["ledger_receipt_advanced"])

    def test_once_timeout_fails_and_restores_regular_agent(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            (Path(directory) / "trust-evidence.json").write_text(json.dumps({"restart": self.identity(1)}))
            fixture.wait_driver = Mock(return_value=self.identity(2))
            def compose(*args, **kwargs):
                if "--once" in args:
                    raise subprocess.TimeoutExpired("fixture-agent", 60)
                return ""
            fixture.compose.side_effect = compose
            with patch("validate_docker_isolated.subprocess.run") as cleanup:
                fixture.once_probe()
            self.assertEqual(fixture.record.call_args.args, ("agent-once-exit", "failed"))
            self.assertEqual(fixture.compose.call_args.args, ("up", "-d", "--no-deps", "docker-runtime-agent"))
            cleanup.assert_called_once()

    def test_once_receipt_without_advance_cannot_pass(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            (Path(directory) / "trust-evidence.json").write_text(json.dumps({"restart": self.identity(1)}))
            fixture.wait_driver = Mock(return_value=self.identity(2))
            with patch("validate_docker_isolated.subprocess.run"):
                fixture.once_probe()
            self.assertEqual(fixture.record.call_args.args, ("agent-once-exit", "failed"))

    def test_import_readback_detects_persisted_field_drift(self):
        self.assert_import_readback(persisted_target=99, expected_status="failed")

    def test_import_readback_and_conflict_preserve_persisted_fields(self):
        self.assert_import_readback(persisted_target=1, expected_status="passed")

    def assert_import_readback(self, persisted_target, expected_status):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            identifier = "app_" + "0" * 26
            expected = {"display_name": "Imported graft-import-probe-test", "runtime_target_id": 1,
                        "compose_project_name": "compose-probe", "compose_project_name_source": "computed"}
            detail = {key: value for key, value in expected.items() if key != "runtime_target_id"}
            detail["runtime_target"] = {"id": 1}
            fixture.api = Mock(side_effect=[
                (200, {"data": {"items": [{"managed": True, "provider": "local", "root_id": "root"}]}}),
                (200, {"data": {"inspection_id": "first", "compose_project_name": "compose-probe", "compose_project_name_source": "computed"}}),
                (201, {"data": {"application": {"application_id": identifier}}}),
                (200, {"data": detail}), (200, {"data": {"inspection_id": "second"}}), (409, {})])
            fixture.persisted_application = Mock(return_value={**expected, "runtime_target_id": persisted_target})
            with patch("validate_docker_isolated.secrets.token_hex", return_value="test"):
                fixture.import_probe("/workspace", "services: {}", "container", 1, {})
            results = {call.args[0]: call for call in fixture.record.call_args_list}
            self.assertEqual(results["compose-import-readback"].args[1], expected_status)
            self.assertEqual(results["compose-reimport-conflict"].args[1], "passed")
            self.assertEqual(fixture.persisted_application.call_count, 2)
            self.assertEqual(fixture.api.call_args_list[-1].args[2]["inspection_id"], "second")

    def test_persisted_readback_rejects_noncanonical_application_identifier(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            with self.assertRaisesRegex(RuntimeError, "invalid application identity"):
                fixture.persisted_application("app_'; DELETE FROM applications")
            fixture.compose.assert_not_called()


if __name__ == "__main__":
    unittest.main()
