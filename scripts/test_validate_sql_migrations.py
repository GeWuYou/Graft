#!/usr/bin/env python3
"""Unit tests for live migration SQL governance checks."""

from __future__ import annotations

import sys
import unittest
import hashlib
import io
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parent))
import validate_sql_migrations as validator
from validate_sql_migrations import validate, validate_file


class ValidateSqlMigrationsTest(unittest.TestCase):
    def write_authorities(self, root: Path) -> tuple[str, str]:
        governance = root / validator.GOVERNANCE_PATH
        lessons = root / validator.LESSONS_PATH
        governance.parent.mkdir(parents=True, exist_ok=True)
        lessons.parent.mkdir(parents=True, exist_ok=True)
        governance.write_text("# migration governance\n", encoding="utf-8")
        lessons.write_text("## MIG-001: uniqueness\n- Status: active\n", encoding="utf-8")
        return (
            f"sha256:{hashlib.sha256(governance.read_bytes()).hexdigest()}",
            f"sha256:{hashlib.sha256(lessons.read_bytes()).hexdigest()}",
        )

    def write_preflight(self, root: Path, sql: Path, risk: str = "L0", lesson_ids: str = "[]") -> Path:
        governance_revision, lessons_revision = self.write_authorities(root)
        sidecar = sql.with_suffix("").with_name(f"{sql.stem}.preflight.yaml")
        sidecar.write_text(
            f"""migration:
  path: {sql.relative_to(root)}
  version: '{sql.name.split('_', 1)[0]}'
owner: test
risk_level: {risk}
affected_tables: [demo_events]
operation_categories: [schema]
historical_data_assumptions: [empty]
referenced_tables: [none]
planned_upgrade_order: [preflight, migrate]
safety_strategy:
  duplicate_scan: SELECT 1
  live_reference_scan: SELECT 1
  reconcile_or_abort: abort
  post_migration_invariant: unique
  bounded_backfill: bounded
  reference_impact: checked
  recovery_or_retirement_rationale: documented
  backup_restore_owner: operator
  release_upgrade_documentation: docs
validation_scenarios: [empty-bootstrap]
retrieval_receipt:
  governance:
    path: {validator.GOVERNANCE_PATH}
    revision: {governance_revision}
  lessons:
    path: {validator.LESSONS_PATH}
    revision: {lessons_revision}
  lesson_ids: {lesson_ids}
""",
            encoding="utf-8",
        )
        return sidecar

    def test_valid_create_table_and_add_column_pass(self) -> None:
        with TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "202606110001_valid.sql"
            path.write_text(
                """
CREATE TABLE IF NOT EXISTS "demo_events" (
  "id" BIGSERIAL PRIMARY KEY,
  "context_json" JSONB NOT NULL DEFAULT '{}'::jsonb,
  "enabled" BOOLEAN NOT NULL DEFAULT TRUE,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE "demo_events"
  ADD COLUMN IF NOT EXISTS "status" VARCHAR(32) NOT NULL DEFAULT 'enabled';

COMMENT ON TABLE "demo_events" IS '演示事件表';
COMMENT ON COLUMN "demo_events"."id" IS '演示事件主键';
COMMENT ON COLUMN "demo_events"."context_json" IS '演示事件上下文 JSON，用于详情展示';
COMMENT ON COLUMN "demo_events"."enabled" IS '是否启用演示事件，true 表示启用，false 表示停用';
COMMENT ON COLUMN "demo_events"."created_at" IS '演示事件创建时间';
COMMENT ON COLUMN "demo_events"."status" IS '演示事件状态，取值来自演示状态枚举';
""".strip()
                + "\n",
                encoding="utf-8",
            )

            self.assertEqual(validate_file(path), [])

    def test_reports_missing_table_and_column_comments(self) -> None:
        with TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "202606110001_missing.sql"
            path.write_text(
                """
CREATE TABLE demo_events (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
""".strip()
                + "\n",
                encoding="utf-8",
            )

            messages = [finding.message for finding in validate_file(path)]

            self.assertIn("CREATE TABLE is missing COMMENT ON TABLE", messages)
            self.assertEqual(messages.count("CREATE TABLE column is missing COMMENT ON COLUMN"), 2)

    def test_reports_add_column_missing_comment(self) -> None:
        with TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "202606110001_add_column.sql"
            path.write_text(
                """
ALTER TABLE demo_events
  ADD COLUMN status VARCHAR(32) NOT NULL DEFAULT 'enabled',
  ADD COLUMN context_json JSONB NULL;
COMMENT ON COLUMN demo_events.status IS '演示事件状态，取值来自演示状态枚举';
""".strip()
                + "\n",
                encoding="utf-8",
            )

            findings = validate_file(path)

            self.assertEqual(len(findings), 1)
            self.assertEqual(findings[0].table, "demo_events")
            self.assertEqual(findings[0].column, "context_json")
            self.assertEqual(findings[0].message, "ALTER TABLE ADD COLUMN is missing COMMENT ON COLUMN")

    def test_reports_invalid_comment_content(self) -> None:
        with TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "202606110001_invalid_comment.sql"
            path.write_text(
                """
CREATE TABLE demo_events (
  id BIGSERIAL PRIMARY KEY,
  status VARCHAR(32) NOT NULL
);
COMMENT ON TABLE demo_events IS 'Notification events';
COMMENT ON COLUMN demo_events.id IS 'id';
COMMENT ON COLUMN demo_events.status IS 'TODO';
""".strip()
                + "\n",
                encoding="utf-8",
            )

            messages = [finding.message for finding in validate_file(path)]

            self.assertIn("table comment must contain Chinese text", messages)
            self.assertIn("column comment must describe business meaning instead of restating the identifier", messages)
            self.assertIn("column comment must not use TODO/TBD/placeholder wording", messages)

    def test_reports_duplicate_live_versions(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            first = root / "server" / "modules" / "user" / "migrations" / "202606110001_user.sql"
            second = root / "server" / "modules" / "rbac" / "migrations" / "202606110001_rbac.sql"
            first.parent.mkdir(parents=True)
            second.parent.mkdir(parents=True)
            first.write_text("SELECT 1;\n", encoding="utf-8")
            second.write_text("SELECT 1;\n", encoding="utf-8")

            findings = validate([first, second], root)

            self.assertEqual(len(findings), 1)
            self.assertIn("live migration version 202606110001 is reused by", findings[0].message)

    def test_path_mode_checks_versions_against_all_live_migrations(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            incoming = root / "server" / "modules" / "scheduler" / "migrations" / "202606110001_scheduler.sql"
            existing = root / "server" / "modules" / "audit" / "migrations" / "202606110001_audit.sql"
            incoming.parent.mkdir(parents=True)
            existing.parent.mkdir(parents=True)
            incoming.write_text("SELECT 1;\n", encoding="utf-8")
            existing.write_text("SELECT 1;\n", encoding="utf-8")

            with (
                mock.patch.object(validator, "live_sql_files", return_value=[existing]),
                mock.patch.object(validator, "validate_file", return_value=[]) as validate_file_mock,
            ):
                findings = validator.validate([incoming], root)

            validate_file_mock.assert_called_once_with(incoming)
            self.assertEqual(len(findings), 1)
            self.assertIn("live migration version 202606110001 is reused by", findings[0].message)
            self.assertIn(str(existing.relative_to(root)), findings[0].message)

    def test_preflight_requires_one_sidecar_and_receipt(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            sql = root / "server/modules/demo/migrations/202608120001_demo.sql"
            sql.parent.mkdir(parents=True)
            sql.write_text("CREATE TABLE demo_events (id BIGINT);\n", encoding="utf-8")
            findings = validator.validate_preflight(sql, root)
            self.assertIn("exactly one .preflight.yaml", findings[0].message)

            self.write_preflight(root, sql, lesson_ids="[MIG-001]")
            self.assertEqual(validator.validate_preflight(sql, root), [])

    def test_preflight_rejects_stale_receipt_and_inactive_lesson(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            sql = root / "server/modules/demo/migrations/202608120001_demo.sql"
            sql.parent.mkdir(parents=True)
            sql.write_text("CREATE TABLE demo_events (id BIGINT);\n", encoding="utf-8")
            sidecar = self.write_preflight(root, sql, lesson_ids="[MIG-999]")
            text = sidecar.read_text(encoding="utf-8").replace("revision: sha256:", "revision: stale-sha256:", 1)
            sidecar.write_text(text, encoding="utf-8")
            messages = [finding.message for finding in validator.validate_preflight(sql, root)]
            self.assertIn("retrieval_receipt.governance.revision does not match canonical content", messages)
            self.assertTrue(any("not active migration lessons" in message for message in messages))

    def test_sql_risk_cannot_be_understated(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            sql = root / "server/modules/demo/migrations/202608120001_demo.sql"
            sql.parent.mkdir(parents=True)
            sql.write_text("CREATE UNIQUE INDEX uq_demo ON demo_events (id);\n", encoding="utf-8")
            self.write_preflight(root, sql, risk="L0")
            messages = [finding.message for finding in validator.validate_preflight(sql, root)]
            self.assertIn("risk_level L0 understates SQL-derived minimum L2", messages)

    def test_combined_unique_index_and_delete_requires_all_safety_evidence(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            sql = root / "server/modules/demo/migrations/202608120001_demo.sql"
            sql.parent.mkdir(parents=True)
            sql.write_text("CREATE UNIQUE INDEX uq_demo ON demo_events (id);\nDELETE FROM demo_events WHERE id = 0;\n", encoding="utf-8")
            sidecar = self.write_preflight(root, sql, risk="L4")
            sidecar.write_text(sidecar.read_text(encoding="utf-8").replace("  duplicate_scan: SELECT 1\n", "").replace("  live_reference_scan: SELECT 1\n", "").replace("  post_migration_invariant: unique\n", ""), encoding="utf-8")

            messages = [finding.message for finding in validator.validate_preflight(sql, root)]

            self.assertIn("safety_strategy.duplicate_scan is required for L2", messages)
            self.assertIn("safety_strategy.live_reference_scan is required for L2", messages)
            self.assertIn("safety_strategy.post_migration_invariant is required for L2", messages)

    def test_changed_mode_requires_a_base_ref(self) -> None:
        with TemporaryDirectory() as temp_dir:
            with self.assertRaisesRegex(ValueError, "--changed requires --base-ref"):
                validator.git_changed_sql_entries(Path(temp_dir), None, staged=False)

    def test_changed_command_rejects_a_missing_sidecar(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            sql = root / "server/modules/demo/migrations/202608120001_demo.sql"
            sql.parent.mkdir(parents=True)
            sql.write_text("CREATE TABLE demo_events (id BIGINT);\n", encoding="utf-8")

            with (
                mock.patch.object(validator, "repo_root", return_value=root),
                mock.patch.object(validator, "git_changed_sql_entries", return_value=[validator.ChangedSqlEntry("A", sql, True)]),
                mock.patch.object(sys, "argv", ["validate_sql_migrations.py", "--changed", "--base-ref", "origin/main"]),
                mock.patch("sys.stderr", new_callable=io.StringIO) as stderr,
            ):
                self.assertEqual(validator.main(), 1)

            self.assertIn("exactly one .preflight.yaml sidecar", stderr.getvalue())

    def test_changed_command_reports_deleted_historical_migration_before_skip(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            deleted = root / "server/modules/demo/migrations/202608120000_deleted.sql"

            with (
                mock.patch.object(validator, "repo_root", return_value=root),
                mock.patch.object(validator, "git_changed_sql_entries", return_value=[validator.ChangedSqlEntry("D", deleted, False)]),
                mock.patch.object(sys, "argv", ["validate_sql_migrations.py", "--changed", "--base-ref", "origin/main"]),
                mock.patch("sys.stderr", new_callable=io.StringIO) as stderr,
            ):
                self.assertEqual(validator.main(), 1)

            output = stderr.getvalue()
            self.assertIn("sql migration gate: failed", output)
            self.assertIn("historical live migration was modified", output)

    def test_changed_command_validates_rename_current_path_without_reading_old_path(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            old = root / "server/modules/demo/migrations/202608120000_old.sql"
            new = root / "server/modules/demo/migrations/202608120001_new.sql"
            new.parent.mkdir(parents=True)
            new.write_text("SELECT 1;\n", encoding="utf-8")

            with (
                mock.patch.object(validator, "repo_root", return_value=root),
                mock.patch.object(
                    validator,
                    "git_changed_sql_entries",
                    return_value=[
                        validator.ChangedSqlEntry("R100", old, False),
                        validator.ChangedSqlEntry("R100", new, True),
                    ],
                ),
                mock.patch.object(sys, "argv", ["validate_sql_migrations.py", "--changed", "--base-ref", "origin/main"]),
                mock.patch("sys.stderr", new_callable=io.StringIO) as stderr,
            ):
                self.assertEqual(validator.main(), 1)

            output = stderr.getvalue()
            self.assertIn("202608120000_old.sql", output)
            self.assertIn("historical live migration was modified", output)
            self.assertIn("202608120001_new.sql", output)
            self.assertIn("exactly one .preflight.yaml sidecar", output)

    def test_rename_is_treated_as_historical_migration_change(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            sql = root / "server/modules/demo/migrations/202608120001_demo.sql"
            sql.parent.mkdir(parents=True)
            sql.write_text("SELECT 1;\n", encoding="utf-8")

            with (
                mock.patch.object(validator.subprocess, "check_output", return_value=b"R100\x00server/modules/demo/migrations/202608120000_old.sql\x00server/modules/demo/migrations/202608120001_demo.sql\x00"),
                mock.patch.object(validator, "default_migration_dirs", return_value=[sql.parent]),
            ):
                entries = validator.git_changed_sql_entries(root, "origin/main", staged=False)

            self.assertEqual(
                entries,
                [
                    validator.ChangedSqlEntry("R100", root / "server/modules/demo/migrations/202608120000_old.sql", False),
                    validator.ChangedSqlEntry("R100", sql, True),
                ],
            )
            messages = [finding.message for finding in validator.validate_historical_immutability(entries)]
            self.assertTrue(any("historical live migration was modified" in message for message in messages))

    def test_deleted_migration_is_tracked_for_historical_validation(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            migrations = root / "server/modules/demo/migrations"
            migrations.mkdir(parents=True)

            with (
                mock.patch.object(validator.subprocess, "check_output", return_value=b"D\x00server/modules/demo/migrations/202608120000_deleted.sql\x00"),
                mock.patch.object(validator, "default_migration_dirs", return_value=[migrations]),
            ):
                entries = validator.git_changed_sql_entries(root, "origin/main", staged=False)

            self.assertEqual(entries, [validator.ChangedSqlEntry("D", root / "server/modules/demo/migrations/202608120000_deleted.sql", False)])
            messages = [finding.message for finding in validator.validate_historical_immutability(entries)]
            self.assertTrue(any("historical live migration was modified" in message for message in messages))


if __name__ == "__main__":
    unittest.main()
