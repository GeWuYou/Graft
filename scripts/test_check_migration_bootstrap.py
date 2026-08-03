#!/usr/bin/env python3
"""Focused tests for the disposable migration bootstrap checker."""

from __future__ import annotations

import importlib.util
from pathlib import Path
import subprocess
import sys
import unittest
from unittest import mock


SCRIPT_PATH = Path(__file__).with_name("check_migration_bootstrap.py")
MODULE_SPEC = importlib.util.spec_from_file_location("check_migration_bootstrap", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"Unable to load module from {SCRIPT_PATH}.")
MODULE = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = MODULE
MODULE_SPEC.loader.exec_module(MODULE)


class MigrationEnvironmentTests(unittest.TestCase):
    def test_migration_environment_targets_disposable_database(self) -> None:
        target = MODULE.BootstrapTarget("temporary-postgres", 42424)
        environment = MODULE.migration_environment(target)

        self.assertEqual(environment["GRAFT_APP_ENV"], "ci")
        self.assertEqual(environment["GRAFT_CONFIG_SCHEMA_VERSION"], "1")
        self.assertEqual(
            environment["GRAFT_DATABASE_URL"],
            f"postgres://graft:graft@127.0.0.1:{target.port}/graft?sslmode=disable",
        )
        self.assertEqual(environment["GRAFT_REDIS_ADDR"], "127.0.0.1:6379")


class HistoricalMigrationTests(unittest.TestCase):
    def test_materialize_historical_server_archives_only_server_tree(self) -> None:
        with self.subTest("returns extracted server directory"):
            with mock.patch.object(MODULE, "run_command") as run_command:
                with mock.patch.object(MODULE.Path, "is_dir", return_value=True):
                    historical_server = MODULE.materialize_historical_server("v0.11.0-beta.38", Path("/tmp/historical"))

        self.assertEqual(historical_server, Path("/tmp/historical/server"))
        self.assertEqual(
            run_command.call_args_list[0].args[0],
            [
                "git",
                "archive",
                "--format=tar",
                "--output=/tmp/historical/historical-server.tar",
                "v0.11.0-beta.38",
                "server",
            ],
        )
        self.assertEqual(
            run_command.call_args_list[1].args[0],
            ["tar", "--extract", "--file", "/tmp/historical/historical-server.tar", "--directory", "/tmp/historical"],
        )

    def test_apply_historical_migrations_uses_historical_server_with_target_environment(self) -> None:
        target = MODULE.BootstrapTarget("temporary-postgres", 42424)
        temporary_directory = mock.Mock()
        temporary_directory.__enter__ = mock.Mock(return_value="/tmp/historical")
        temporary_directory.__exit__ = mock.Mock(return_value=None)
        historical_server = Path("/tmp/historical/server")
        with mock.patch.object(MODULE.tempfile, "TemporaryDirectory", return_value=temporary_directory), mock.patch.object(
            MODULE, "materialize_historical_server", return_value=historical_server
        ) as materialize_historical_server, mock.patch.object(MODULE, "run_command") as run_command:
            MODULE.apply_historical_migrations(target, "v0.11.0-beta.38")

        materialize_historical_server.assert_called_once_with("v0.11.0-beta.38", Path("/tmp/historical"))
        self.assertEqual(run_command.call_args.args[0], ["go", "run", "./cmd/graft", "migrate", "up", "--allow-dirty"])
        self.assertEqual(run_command.call_args.kwargs["cwd"], historical_server)
        self.assertEqual(run_command.call_args.kwargs["env"], MODULE.migration_environment(target))


class SchemaContractTests(unittest.TestCase):
    def test_check_schema_contract_uses_disposable_database_environment(self) -> None:
        target = MODULE.BootstrapTarget("temporary-postgres", 42424)
        completed = mock.Mock(stdout='{"findings": []}\n')
        with mock.patch.object(MODULE, "run_command", return_value=completed) as run_command:
            report = MODULE.check_schema_contract(target)

        self.assertEqual(report, '{"findings": []}\n')
        command = run_command.call_args.args[0]
        self.assertEqual(command[-4:], ["--mode", "enforce", "--format", "json"])
        self.assertEqual(run_command.call_args.kwargs["cwd"], MODULE.SERVER_DIR)


class ReadinessTests(unittest.TestCase):
    def test_wait_for_postgres_requires_stable_readiness(self) -> None:
        target = MODULE.BootstrapTarget("temporary-postgres", 42424)
        readiness = [mock.Mock(returncode=0), mock.Mock(returncode=1), mock.Mock(returncode=0), mock.Mock(returncode=0), mock.Mock(returncode=0)]
        with mock.patch.object(MODULE, "run_command", side_effect=readiness) as run_command, mock.patch.object(
            MODULE.time, "sleep"
        ) as sleep:
            MODULE.wait_for_postgres(target)

        self.assertEqual(run_command.call_count, 5)
        self.assertEqual(sleep.call_count, 4)


class ImagePullTests(unittest.TestCase):
    def test_pull_postgres_image_retries_before_succeeding(self) -> None:
        failed = mock.Mock(returncode=1, stdout="", stderr="network reset")
        succeeded = mock.Mock(returncode=0, stdout="pulled", stderr="")
        with mock.patch.object(MODULE, "run_command", side_effect=[failed, succeeded]) as run_command, mock.patch.object(
            MODULE.time, "sleep"
        ) as sleep:
            MODULE.pull_postgres_image()

        self.assertEqual(run_command.call_count, 2)
        self.assertEqual(sleep.call_count, 1)
        self.assertEqual(run_command.call_args.args[0], ["docker", "pull", MODULE.POSTGRES_IMAGE])
        self.assertFalse(run_command.call_args.kwargs["check"])
        self.assertEqual(run_command.call_args.kwargs["timeout"], MODULE.IMAGE_PULL_TIMEOUT_SECONDS)

    def test_pull_postgres_image_reports_final_failure(self) -> None:
        failed = mock.Mock(returncode=1, stdout="", stderr="network reset")
        with mock.patch.object(MODULE, "run_command", return_value=failed) as run_command, mock.patch.object(
            MODULE.time, "sleep"
        ) as sleep:
            with self.assertRaisesRegex(MODULE.CommandError, "docker pull postgres:16-alpine"):
                MODULE.pull_postgres_image()

        self.assertEqual(run_command.call_count, MODULE.IMAGE_PULL_ATTEMPTS)
        self.assertEqual(sleep.call_count, MODULE.IMAGE_PULL_ATTEMPTS - 1)

    def test_pull_postgres_image_retries_after_timeout(self) -> None:
        timeout = subprocess.TimeoutExpired(
            ["docker", "pull", MODULE.POSTGRES_IMAGE], MODULE.IMAGE_PULL_TIMEOUT_SECONDS
        )
        succeeded = mock.Mock(returncode=0, stdout="pulled", stderr="")
        with mock.patch.object(MODULE, "run_command", side_effect=[timeout, succeeded]) as run_command, mock.patch.object(
            MODULE.time, "sleep"
        ) as sleep:
            MODULE.pull_postgres_image()

        self.assertEqual(run_command.call_count, 2)
        self.assertEqual(sleep.call_count, 1)
        self.assertEqual(run_command.call_args.kwargs["timeout"], MODULE.IMAGE_PULL_TIMEOUT_SECONDS)

    def test_pull_postgres_image_reports_final_timeout(self) -> None:
        timeout = subprocess.TimeoutExpired(
            ["docker", "pull", MODULE.POSTGRES_IMAGE], MODULE.IMAGE_PULL_TIMEOUT_SECONDS
        )
        with mock.patch.object(MODULE, "run_command", side_effect=timeout) as run_command, mock.patch.object(
            MODULE.time, "sleep"
        ) as sleep:
            with self.assertRaisesRegex(MODULE.CommandError, "timed out after"):
                MODULE.pull_postgres_image()

        self.assertEqual(run_command.call_count, MODULE.IMAGE_PULL_ATTEMPTS)
        self.assertEqual(sleep.call_count, MODULE.IMAGE_PULL_ATTEMPTS - 1)


class LifecycleTests(unittest.TestCase):
    def test_main_cleans_up_after_success(self) -> None:
        target = MODULE.BootstrapTarget("temporary-postgres", 42424)
        with mock.patch.object(MODULE, "parse_args", return_value=mock.Mock(keep_container=False, schema_report=None, upgrade_from=None)), mock.patch.object(
            MODULE, "uuid", mock.Mock(uuid4=lambda: mock.Mock(hex="abc123def456"))
        ), mock.patch.object(MODULE, "start_postgres", return_value=target), mock.patch.object(
            MODULE, "wait_for_postgres"
        ), mock.patch.object(MODULE, "apply_migrations"), mock.patch.object(
            MODULE, "check_schema_contract", return_value='{"findings": []}\n'
        ), mock.patch.object(MODULE, "remove_postgres") as remove_postgres:
            self.assertEqual(MODULE.main(), 0)

        remove_postgres.assert_called_once_with("graft-migration-bootstrap-abc123def456")

    def test_main_emits_diagnostics_and_cleans_up_after_failure(self) -> None:
        with mock.patch.object(MODULE, "parse_args", return_value=mock.Mock(keep_container=False, schema_report=None, upgrade_from=None)), mock.patch.object(
            MODULE, "uuid", mock.Mock(uuid4=lambda: mock.Mock(hex="abc123def456"))
        ), mock.patch.object(MODULE, "start_postgres", side_effect=RuntimeError("docker unavailable")), mock.patch.object(
            MODULE, "print_diagnostics"
        ) as print_diagnostics, mock.patch.object(MODULE, "remove_postgres") as remove_postgres:
            self.assertEqual(MODULE.main(), 1)

        print_diagnostics.assert_called_once_with("graft-migration-bootstrap-abc123def456")
        remove_postgres.assert_called_once_with("graft-migration-bootstrap-abc123def456")

    def test_main_writes_schema_report_before_propagating_schema_check_failure(self) -> None:
        target = MODULE.BootstrapTarget("temporary-postgres", 42424)
        command_error = MODULE.CommandError("schema check failed", stdout='{"findings": [{"name": "users"}]}\n')
        schema_report = mock.Mock()
        with mock.patch.object(
            MODULE, "parse_args", return_value=mock.Mock(keep_container=False, schema_report=schema_report, upgrade_from=None)
        ) as parse_args, mock.patch.object(
            MODULE, "uuid", mock.Mock(uuid4=lambda: mock.Mock(hex="abc123def456"))
        ), mock.patch.object(MODULE, "start_postgres", return_value=target), mock.patch.object(
            MODULE, "wait_for_postgres"
        ), mock.patch.object(MODULE, "apply_migrations"), mock.patch.object(
            MODULE, "check_schema_contract", side_effect=command_error
        ), mock.patch.object(MODULE, "print_diagnostics"), mock.patch.object(
            MODULE, "remove_postgres"
        ):
            self.assertEqual(MODULE.main(), 1)

        parse_args.assert_called_once_with()
        schema_report.write_text.assert_called_once_with(command_error.stdout, encoding="utf-8")

    def test_main_does_not_write_schema_report_for_non_json_schema_check_failure(self) -> None:
        target = MODULE.BootstrapTarget("temporary-postgres", 42424)
        command_error = MODULE.CommandError("schema check failed", stdout="migration check output\n")
        schema_report = mock.Mock()
        with mock.patch.object(
            MODULE, "parse_args", return_value=mock.Mock(keep_container=False, schema_report=schema_report, upgrade_from=None)
        ), mock.patch.object(
            MODULE, "uuid", mock.Mock(uuid4=lambda: mock.Mock(hex="abc123def456"))
        ), mock.patch.object(MODULE, "start_postgres", return_value=target), mock.patch.object(
            MODULE, "wait_for_postgres"
        ), mock.patch.object(MODULE, "apply_migrations"), mock.patch.object(
            MODULE, "check_schema_contract", side_effect=command_error
        ), mock.patch.object(MODULE, "print_diagnostics"), mock.patch.object(
            MODULE, "remove_postgres"
        ):
            self.assertEqual(MODULE.main(), 1)

        schema_report.write_text.assert_not_called()

    def test_main_applies_historical_migrations_before_current_chain(self) -> None:
        target = MODULE.BootstrapTarget("temporary-postgres", 42424)
        call_order: list[str] = []
        with mock.patch.object(
            MODULE, "parse_args", return_value=mock.Mock(keep_container=False, schema_report=None, upgrade_from="v0.11.0-beta.38")
        ), mock.patch.object(MODULE, "uuid", mock.Mock(uuid4=lambda: mock.Mock(hex="abc123def456"))), mock.patch.object(
            MODULE, "start_postgres", return_value=target
        ), mock.patch.object(MODULE, "wait_for_postgres"), mock.patch.object(
            MODULE, "apply_historical_migrations", side_effect=lambda *args, **kwargs: call_order.append("historical")
        ) as apply_historical_migrations, mock.patch.object(
            MODULE, "apply_migrations", side_effect=lambda *args, **kwargs: call_order.append("current")
        ) as apply_migrations, mock.patch.object(
            MODULE, "check_schema_contract", return_value='{"findings": []}\n'
        ), mock.patch.object(MODULE, "remove_postgres"):
            self.assertEqual(MODULE.main(), 0)

        apply_historical_migrations.assert_called_once_with(target, "v0.11.0-beta.38")
        apply_migrations.assert_called_once_with(target)
        self.assertEqual(call_order, ["historical", "current"])


if __name__ == "__main__":
    unittest.main()
