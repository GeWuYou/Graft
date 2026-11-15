"""Regression checks for the private Docker boundary in the conformance overlay."""
import copy
import unittest
from validate_docker_isolated import validate_topology


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


if __name__ == "__main__":
    unittest.main()
