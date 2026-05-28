from __future__ import annotations

import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import patch

from check_migration_versions import candidate_dirs, validate


class CandidateDirsTest(unittest.TestCase):
    def test_changed_mode_collects_staged_plugin_migration_dirs(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            migration_dir = root / "server" / "plugins" / "user" / "migrations"
            migration_dir.mkdir(parents=True)

            with patch(
                "check_migration_versions.subprocess.check_output",
                return_value="server/plugins/user/migrations/202605280001_user.sql\nserver/plugins/user/README.md\n",
            ):
                dirs = candidate_dirs(root, "changed")

            self.assertEqual(dirs, [migration_dir])


class ValidateTest(unittest.TestCase):
    def test_validate_reports_duplicate_versions(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            user_dir = root / "server" / "plugins" / "user" / "migrations"
            rbac_dir = root / "server" / "plugins" / "rbac" / "migrations"
            user_dir.mkdir(parents=True)
            rbac_dir.mkdir(parents=True)
            (user_dir / "202605280001_user.sql").write_text("SELECT 1;\n", encoding="utf-8")
            (rbac_dir / "202605280001_rbac.sql").write_text("SELECT 1;\n", encoding="utf-8")

            errors = validate([user_dir, rbac_dir], root)

            self.assertEqual(
                errors,
                [
                    "default migration chain version conflict: 202605280001 appears in "
                    "server/plugins/user/migrations/202605280001_user.sql, "
                    "server/plugins/rbac/migrations/202605280001_rbac.sql"
                ],
            )

    def test_validate_allows_unique_versions(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            user_dir = root / "server" / "plugins" / "user" / "migrations"
            rbac_dir = root / "server" / "plugins" / "rbac" / "migrations"
            user_dir.mkdir(parents=True)
            rbac_dir.mkdir(parents=True)
            (user_dir / "202605280001_user.sql").write_text("SELECT 1;\n", encoding="utf-8")
            (rbac_dir / "202605280002_rbac.sql").write_text("SELECT 1;\n", encoding="utf-8")

            errors = validate([user_dir, rbac_dir], root)

            self.assertEqual(errors, [])


if __name__ == "__main__":
    unittest.main()
