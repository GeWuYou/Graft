#!/usr/bin/env python3
"""Focused regression tests for local Graft validation deployment tooling."""

from __future__ import annotations

import contextlib
import io
import importlib.util
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock
import os
import subprocess
import yaml


SCRIPT_PATH = Path(__file__).with_name("local_graft_validation.py")
SPEC = importlib.util.spec_from_file_location("local_graft_validation", SCRIPT_PATH)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError("unable to load local validation script")
MODULE = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class LocalValidationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tempdir = tempfile.TemporaryDirectory()
        self.root = Path(self.tempdir.name) / "graft-validation"

    def tearDown(self) -> None:
        self.tempdir.cleanup()

    @mock.patch.object(MODULE, "validate_ports")
    def test_init_generates_distinct_production_instances(self, validate_ports: mock.Mock) -> None:
        MODULE.initialise(self.root)

        expected = {"beta": ("beta", "3101"), "latest": ("latest", "3102"), "fixed": ("v0.11.0-beta.39", "3103")}
        for instance, (tag, port) in expected.items():
            env = MODULE.parse_env(self.root / instance / ".env")
            self.assertEqual(env["GRAFT_IMAGE_TAG"], tag)
            self.assertEqual(env["GRAFT_APP_ENV"], "production")
            self.assertEqual(env["GRAFT_WEB_HOST_PORT"], port)
            self.assertEqual(env["GRAFT_UPDATE_STATE_VOLUME"], MODULE.update_volume_name(instance))
            self.assertEqual(env["GRAFT_DEPLOYMENT_COMPOSE_ROOT"], str((self.root / instance).resolve()))
            self.assertTrue((self.root / instance / ".data" / "postgres").is_dir())
        self.assertEqual(validate_ports.call_count, 1)

    @mock.patch.object(MODULE, "validate_ports")
    def test_readme_preserves_unmanaged_content(self, _validate_ports: mock.Mock) -> None:
        MODULE.initialise(self.root)
        readme = self.root.parent / "README.md"
        readme.write_text("operator note\n\n" + readme.read_text(encoding="utf-8"), encoding="utf-8")
        MODULE.refresh_readme(self.root)
        text = readme.read_text(encoding="utf-8")
        self.assertIn("operator note", text)
        self.assertEqual(text.count(MODULE.README_START), 1)
        self.assertIn("http://127.0.0.1:3103", text)

    @mock.patch.object(MODULE, "validate_ports")
    def test_init_does_not_overwrite_existing_instance_files(self, _validate_ports: mock.Mock) -> None:
        instance = self.root / "beta"
        instance.mkdir(parents=True)
        (instance / "compose.yml").write_text("existing compose\n", encoding="utf-8")
        (instance / ".env").write_text("EXISTING_VALUE=yes\n", encoding="utf-8")

        MODULE.initialise(self.root)

        self.assertEqual((instance / "compose.yml").read_text(encoding="utf-8"), "existing compose\n")
        self.assertEqual((instance / ".env").read_text(encoding="utf-8"), "EXISTING_VALUE=yes\n")
        self.assertTrue((instance / "data" / "apps").is_dir())

    @mock.patch.object(MODULE, "validate_ports")
    def test_require_initialised_rejects_missing_downstream_env_keys(self, _validate_ports: mock.Mock) -> None:
        MODULE.initialise(self.root)
        env_path = self.root / "beta" / ".env"
        env_path.write_text(
            f"GRAFT_DEPLOYMENT_COMPOSE_ROOT={self.root / 'beta'}\n",
            encoding="utf-8",
        )

        with self.assertRaisesRegex(MODULE.LocalValidationError, "missing required initialization keys"):
            MODULE.require_initialised(self.root, "beta")

    @mock.patch.object(MODULE, "docker_published_port_owners", return_value={})
    @mock.patch.object(MODULE, "is_socket_available", return_value=True)
    def test_port_collision_is_rejected(self, _socket_available: mock.Mock, _docker_ports: mock.Mock) -> None:
        self.root.mkdir(parents=True)
        for instance in MODULE.INSTANCES:
            directory = self.root / instance
            directory.mkdir()
            (directory / ".env").write_text("GRAFT_WEB_HOST_PORT=3101\n", encoding="utf-8")
        with self.assertRaisesRegex(MODULE.LocalValidationError, "shared"):
            MODULE.validate_ports(self.root)

    @mock.patch.object(MODULE, "docker_published_port_owners", return_value={3101: {"beta"}})
    @mock.patch.object(MODULE, "is_socket_available", return_value=True)
    def test_running_managed_instance_is_not_an_external_port_conflict(
        self, _socket_available: mock.Mock, _docker_owners: mock.Mock
    ) -> None:
        self.root.mkdir(parents=True)
        for instance, port in {"beta": 3101, "latest": 3102, "fixed": 3103}.items():
            directory = self.root / instance
            directory.mkdir()
            (directory / ".env").write_text(f"GRAFT_WEB_HOST_PORT={port}\n", encoding="utf-8")

        MODULE.validate_ports(self.root)

    @mock.patch.object(MODULE, "docker_published_port_owners", return_value={3101: set()})
    @mock.patch.object(MODULE, "is_socket_available", return_value=True)
    def test_external_docker_port_is_rejected(
        self, _socket_available: mock.Mock, _docker_owners: mock.Mock
    ) -> None:
        self.root.mkdir(parents=True)
        for instance, port in {"beta": 3101, "latest": 3102, "fixed": 3103}.items():
            directory = self.root / instance
            directory.mkdir()
            (directory / ".env").write_text(f"GRAFT_WEB_HOST_PORT={port}\n", encoding="utf-8")

        with self.assertRaisesRegex(MODULE.LocalValidationError, "published"):
            MODULE.validate_ports(self.root)

    @mock.patch.object(MODULE, "run_compose", return_value=0)
    @mock.patch.object(MODULE, "validate_ports")
    def test_set_config_restricts_keys_and_updates_origins(self, _validate_ports: mock.Mock, _run_compose: mock.Mock) -> None:
        MODULE.initialise(self.root)
        with self.assertRaisesRegex(MODULE.LocalValidationError, "not a supported"):
            MODULE.set_config(self.root, "beta", "POSTGRES_PASSWORD", "no")
        with self.assertRaisesRegex(MODULE.LocalValidationError, "between 1 and 65535"):
            MODULE.set_config(self.root, "beta", "GRAFT_WEB_HOST_PORT", "0")
        MODULE.set_config(self.root, "beta", "GRAFT_WEB_HOST_PORT", "3201")
        env = MODULE.parse_env(self.root / "beta" / ".env")
        self.assertEqual(env["GRAFT_WEB_HOST_PORT"], "3201")
        self.assertIn("localhost:3201", env["GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS"])

    @mock.patch.object(MODULE, "run_compose", return_value=1)
    @mock.patch.object(MODULE, "validate_ports")
    def test_set_config_restores_env_after_failed_compose_validation(
        self, _validate_ports: mock.Mock, _run_compose: mock.Mock
    ) -> None:
        MODULE.initialise(self.root)
        env_path = self.root / "beta" / ".env"
        previous = env_path.read_text(encoding="utf-8")

        with self.assertRaisesRegex(MODULE.LocalValidationError, "restored"):
            MODULE.set_config(self.root, "beta", "GRAFT_LOG_LEVEL", "debug")

        self.assertEqual(env_path.read_text(encoding="utf-8"), previous)

    @mock.patch.object(MODULE, "run_compose", return_value=0)
    @mock.patch.object(MODULE, "validate_ports")
    def test_fixed_tag_requires_explicit_release(self, _validate_ports: mock.Mock, _run_compose: mock.Mock) -> None:
        MODULE.initialise(self.root)
        with self.assertRaisesRegex(MODULE.LocalValidationError, "v-prefixed"):
            MODULE.set_config(self.root, "fixed", "GRAFT_IMAGE_TAG", "beta")

    @mock.patch.object(MODULE, "probe_health", return_value=False)
    @mock.patch.object(MODULE, "run_compose", return_value=0)
    @mock.patch.object(MODULE, "validate_ports")
    def test_doctor_fails_when_web_health_probe_fails(
        self, _validate_ports: mock.Mock, _run_compose: mock.Mock, _probe_health: mock.Mock
    ) -> None:
        MODULE.initialise(self.root)

        with contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(MODULE.doctor(self.root, "beta"), 1)

    @mock.patch.object(MODULE, "run_compose", return_value=0)
    @mock.patch.object(MODULE, "published_server_revision", return_value="a" * 40)
    @mock.patch.object(MODULE, "download_release_file")
    @mock.patch.object(MODULE, "validate_ports")
    def test_release_sync_preserves_local_env_values(
        self,
        _validate_ports: mock.Mock,
        download_release_file: mock.Mock,
        _published_server_revision: mock.Mock,
        _run_compose: mock.Mock,
    ) -> None:
        MODULE.initialise(self.root)
        env_path = self.root / "beta" / ".env"
        original_password = MODULE.parse_env(env_path)["POSTGRES_PASSWORD"]
        download_release_file.side_effect = [
            "services:\n  server:\n    image: example\n",
            "GRAFT_IMAGE_TAG=latest\nPOSTGRES_PASSWORD=change-me\n",
        ]

        revision = MODULE.sync_instance_compose(self.root, "beta")

        self.assertEqual(revision, "a" * 40)
        self.assertEqual(MODULE.parse_env(env_path)["POSTGRES_PASSWORD"], original_password)
        self.assertIn("services:", (self.root / "beta" / "compose.yml").read_text(encoding="utf-8"))

    @mock.patch.object(MODULE, "run_compose", return_value=0)
    @mock.patch.object(MODULE, "published_server_revision", return_value="a" * 40)
    @mock.patch.object(MODULE, "download_release_file")
    @mock.patch.object(MODULE, "validate_ports")
    def test_release_sync_preserves_legacy_agent_mode_and_other_profiles(
        self, _ports, download, _revision, _compose
    ) -> None:
        MODULE.initialise(self.root)
        path = self.root / "beta" / ".env"
        values = MODULE.parse_env(path)
        values.pop("GRAFT_EXECUTION_MODE", None)
        values["COMPOSE_PROFILES"] = "operator"
        path.write_text(MODULE.set_env_values("", values), encoding="utf-8")
        download.side_effect = ["services:\n  server:\n    image: example\n",
            "GRAFT_EXECUTION_MODE=local\nCOMPOSE_PROFILES=\n"]
        MODULE.sync_instance_compose(self.root, "beta")
        preserved = MODULE.parse_env(path)
        self.assertEqual(preserved["GRAFT_EXECUTION_MODE"], "agent")
        self.assertEqual(preserved["COMPOSE_PROFILES"], "operator,agent")


    @mock.patch.object(MODULE, "run_compose", return_value=1)
    @mock.patch.object(MODULE, "published_server_revision", return_value="a" * 40)
    @mock.patch.object(MODULE, "download_release_file")
    @mock.patch.object(MODULE, "validate_ports")
    def test_release_sync_restores_files_after_failed_compose_validation(
        self,
        _validate_ports: mock.Mock,
        download_release_file: mock.Mock,
        _published_server_revision: mock.Mock,
        _run_compose: mock.Mock,
    ) -> None:
        MODULE.initialise(self.root)
        directory = self.root / "beta"
        compose_path = directory / "compose.yml"
        env_path = directory / ".env"
        previous_compose = compose_path.read_text(encoding="utf-8")
        previous_env = env_path.read_text(encoding="utf-8")
        download_release_file.side_effect = [
            "services:\n  server:\n    image: example\n",
            "GRAFT_IMAGE_TAG=latest\nPOSTGRES_PASSWORD=change-me\n",
        ]

        with self.assertRaisesRegex(MODULE.LocalValidationError, "restored"):
            MODULE.sync_instance_compose(self.root, "beta")

        self.assertEqual(compose_path.read_text(encoding="utf-8"), previous_compose)
        self.assertEqual(env_path.read_text(encoding="utf-8"), previous_env)

    @mock.patch.object(MODULE.subprocess, "run")
    @mock.patch.object(MODULE, "run_compose", return_value=0)
    def test_reset_requires_explicit_confirmation(self, run_compose: mock.Mock, run: mock.Mock) -> None:
        with mock.patch.object(MODULE, "validate_ports"):
            MODULE.initialise(self.root)
        with self.assertRaisesRegex(MODULE.LocalValidationError, "--confirm"):
            MODULE.reset(self.root, "beta", False)
        self.assertTrue((self.root / "beta" / "data").is_dir())
        MODULE.reset(self.root, "beta", True)
        self.assertTrue((self.root / "beta" / "data" / "apps").is_dir())
        run_compose.assert_called_once_with(self.root, "beta", "down", "--volumes")
        run.assert_called_once_with(["docker", "volume", "rm", MODULE.update_volume_name("beta")], check=False)


class OfficialComposeExecutionTests(unittest.TestCase):
    def setUp(self) -> None:
        root = SCRIPT_PATH.parents[1]
        self.compose = yaml.safe_load((root / "compose.yml").read_text(encoding="utf-8"))

    def run_preflight(self, mode, profiles="", host_root="/opt/graft/apps", container_root="/opt/graft/apps"):
        script = self.compose["services"]["config-validate"]["command"][0]
        script = script.split("exec /app/graft", 1)[0].replace("$$", "$")
        return subprocess.run(["sh", "-ec", script], capture_output=True, text=True, env={**os.environ,
            "GRAFT_EXECUTION_MODE": mode, "COMPOSE_PROFILES": profiles,
            "GRAFT_APPLICATION_ROOT_HOST_PATH": host_root,
            "GRAFT_APPLICATION_ROOT_CONTAINER_PATH": container_root})

    def test_local_profile_accepts_same_absolute_workspace_without_agent(self):
        self.assertEqual(self.run_preflight("local").returncode, 0)
        self.assertEqual(self.compose["services"]["docker-runtime-agent"]["profiles"], ["agent"])

    def test_generated_helper_instances_pass_topology_preflight_with_isolated_roots(self):
        with tempfile.TemporaryDirectory() as temporary:
            roots = set()
            for instance in MODULE.INSTANCES:
                env = MODULE.desired_env(Path(temporary), instance)
                root = env["GRAFT_APPLICATION_ROOT_HOST_PATH"]
                self.assertEqual(root, env["GRAFT_APPLICATION_ROOT_CONTAINER_PATH"])
                self.assertTrue(Path(root).is_absolute())
                self.assertEqual(self.run_preflight(env["GRAFT_EXECUTION_MODE"], env["COMPOSE_PROFILES"], root, root).returncode, 0)
                roots.add(root)
            self.assertEqual(len(roots), len(MODULE.INSTANCES))

    def test_mode_profile_mismatch_reports_actionable_error(self):
        agent = self.run_preflight("agent")
        self.assertNotEqual(agent.returncode, 0)
        self.assertIn("COMPOSE_PROFILES=agent", agent.stderr)
        self.assertEqual(self.run_preflight("agent", "operator,agent").returncode, 0)
        self.assertNotEqual(self.run_preflight("local", "agent").returncode, 0)

    def test_local_relative_or_mismatched_roots_are_rejected(self):
        self.assertIn("absolute", self.run_preflight("local", host_root="./apps").stderr)
        self.assertIn("matching", self.run_preflight("local", host_root="/srv/apps").stderr)
        self.assertEqual(self.run_preflight("local", host_root="/srv/apps", container_root="/srv/apps").returncode, 0)
        self.assertNotEqual(self.run_preflight("local", host_root="/", container_root="/").returncode, 0)
        self.assertNotEqual(self.run_preflight("local", host_root="/srv/../apps", container_root="/srv/../apps").returncode, 0)

    def test_mode_preflight_runs_after_migrations_before_server(self):
        services = self.compose["services"]
        bootstrap = services["bootstrap"]["command"][0]
        self.assertLess(bootstrap.index("/app/graft migrate up"), bootstrap.index("runtime mode preflight"))
        self.assertEqual(services["server"]["depends_on"]["bootstrap"]["condition"], "service_completed_successfully")
        self.assertEqual(services["runtime-mode-preflight"]["image"], services["server"]["image"])


if __name__ == "__main__":
    unittest.main()
