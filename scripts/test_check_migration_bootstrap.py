#!/usr/bin/env python3
"""Focused tests for the disposable migration bootstrap checker."""

from __future__ import annotations

import importlib.util
from pathlib import Path
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
        environment = MODULE.migration_environment(MODULE.BootstrapTarget("temporary-postgres", 42424))

        self.assertEqual(environment["GRAFT_APP_ENV"], "ci")
        self.assertEqual(
            environment["GRAFT_DATABASE_URL"],
            "postgres://graft:graft@127.0.0.1:42424/graft?sslmode=disable",
        )
        self.assertEqual(environment["GRAFT_REDIS_ADDR"], "127.0.0.1:6379")


class ConstraintAssertionTests(unittest.TestCase):
    def test_assert_task_receipt_constraints_accepts_expected_catalog_state(self) -> None:
        target = MODULE.BootstrapTarget("temporary-postgres", 42424)
        with mock.patch.object(MODULE, "query_boolean", side_effect=[True, True]) as query_boolean:
            MODULE.assert_task_receipt_constraints(target)

        self.assertEqual(query_boolean.call_count, 2)
        self.assertEqual(query_boolean.call_args_list[0].args, (target, MODULE.COMPOSITE_FOREIGN_KEY_QUERY))
        self.assertEqual(query_boolean.call_args_list[1].args, (target, MODULE.LEGACY_SINGLE_STAGE_FOREIGN_KEY_QUERY))

    def test_assert_task_receipt_constraints_rejects_missing_composite_foreign_key(self) -> None:
        with mock.patch.object(MODULE, "query_boolean", return_value=False):
            with self.assertRaisesRegex(RuntimeError, "composite task/stage"):
                MODULE.assert_task_receipt_constraints(MODULE.BootstrapTarget("temporary-postgres", 42424))

    def test_assert_task_receipt_constraints_rejects_legacy_foreign_key(self) -> None:
        with mock.patch.object(MODULE, "query_boolean", side_effect=[True, False]):
            with self.assertRaisesRegex(RuntimeError, "legacy single-column"):
                MODULE.assert_task_receipt_constraints(MODULE.BootstrapTarget("temporary-postgres", 42424))


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


class LifecycleTests(unittest.TestCase):
    def test_main_cleans_up_after_success(self) -> None:
        target = MODULE.BootstrapTarget("temporary-postgres", 42424)
        with mock.patch.object(MODULE, "parse_args", return_value=mock.Mock(keep_container=False)), mock.patch.object(
            MODULE, "uuid", mock.Mock(uuid4=lambda: mock.Mock(hex="abc123def456"))
        ), mock.patch.object(MODULE, "start_postgres", return_value=target), mock.patch.object(
            MODULE, "wait_for_postgres"
        ), mock.patch.object(MODULE, "apply_migrations"), mock.patch.object(
            MODULE, "assert_task_receipt_constraints"
        ), mock.patch.object(MODULE, "remove_postgres") as remove_postgres:
            self.assertEqual(MODULE.main(), 0)

        remove_postgres.assert_called_once_with("graft-migration-bootstrap-abc123def456")

    def test_main_emits_diagnostics_and_cleans_up_after_failure(self) -> None:
        with mock.patch.object(MODULE, "parse_args", return_value=mock.Mock(keep_container=False)), mock.patch.object(
            MODULE, "uuid", mock.Mock(uuid4=lambda: mock.Mock(hex="abc123def456"))
        ), mock.patch.object(MODULE, "start_postgres", side_effect=RuntimeError("docker unavailable")), mock.patch.object(
            MODULE, "print_diagnostics"
        ) as print_diagnostics, mock.patch.object(MODULE, "remove_postgres") as remove_postgres:
            self.assertEqual(MODULE.main(), 1)

        print_diagnostics.assert_called_once_with("graft-migration-bootstrap-abc123def456")
        remove_postgres.assert_called_once_with("graft-migration-bootstrap-abc123def456")


if __name__ == "__main__":
    unittest.main()
