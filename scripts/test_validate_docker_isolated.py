"""Regression checks for the private Docker boundary in the conformance overlay."""
import copy
import unittest
import json
from pathlib import Path
import tempfile
from unittest.mock import Mock
from validate_docker_isolated import Fixture, validate_topology


class PrivateDockerTopologyTests(unittest.TestCase):
    def setUp(self):
        self.configuration = {"services": {}}
        for name in ("docker-daemon", "backend", "conformance-driver", "docker-runtime-agent"):
            self.configuration["services"][name] = {"volumes": [
                {"type": "volume", "source": "daemon-socket", "target": "/var/run"},
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


class FixtureLifecycleTests(unittest.TestCase):
    def test_waits_for_successful_one_shot_jobs_separately(self):
        with tempfile.TemporaryDirectory() as temporary:
            fixture = Fixture.__new__(Fixture)
            fixture.directory = Path(temporary)
            fixture.environment = {"GRAFT_CONFORMANCE_AGENT_IMAGE_DIGEST": "sha256:" + "a" * 64}
            topology = PrivateDockerTopologyTests()
            topology.setUp()
            fixture.compose = Mock(side_effect=lambda *args: json.dumps(topology.configuration) if args[0] == "config" else "")
            fixture.driver = Mock(return_value={"agent_id": "fixture", "target_id": 1,
                "identity_id": "identity", "generation": 1, "ledger_receipt_count": 2})
            fixture.engine = Mock(return_value="private-engine")
            fixture.up()
            calls = [call.args for call in fixture.compose.call_args_list]
            self.assertIn(("wait", "workspace-init", "vault-init"), calls)
            self.assertIn(("wait", "migrate"), calls)
            readiness = [args for args in calls if "--wait" in args]
            self.assertEqual(readiness, [("up", "-d", "--wait", "docker-daemon", "postgres", "redis", "vault"),
                                         ("up", "-d", "--no-deps", "--wait", "backend")])
            self.assertTrue((fixture.directory / "prepared-evidence.json").exists())

    def test_reauthenticates_after_empty_password_change_response(self):
        fixture = Fixture.__new__(Fixture)
        fixture.environment = {"GRAFT_FIXTURE_ADMIN_PASSWORD": "synthetic-password"}
        fixture.api = Mock(side_effect=[(401, {}),
            (200, {"data": {"access_token": "restricted", "must_change_password": True}}),
            (200, {"data": None}), (200, {"data": {"access_token": "normal"}})])
        fixture.authenticate()
        self.assertEqual(fixture.token, "normal")


if __name__ == "__main__":
    unittest.main()
