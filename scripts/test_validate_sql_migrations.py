#!/usr/bin/env python3
# Copyright (c) 2025-2026 GeWuYou
# SPDX-License-Identifier: Apache-2.0

"""Unit tests for live migration SQL governance checks."""

from __future__ import annotations

import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parent))
import validate_sql_migrations as validator
from validate_sql_migrations import validate, validate_file


class ValidateSqlMigrationsTest(unittest.TestCase):
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


if __name__ == "__main__":
    unittest.main()
