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


if __name__ == "__main__":
    unittest.main()
