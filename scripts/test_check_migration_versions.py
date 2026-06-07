from __future__ import annotations

import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import patch

from check_migration_versions import candidate_dirs, validate


def write_registry_fixture(root: Path, module_ids: list[str]) -> None:
    registry = root / "server" / "internal" / "moduleregistry" / "generated.go"
    registry.parent.mkdir(parents=True, exist_ok=True)
    imports = "\n".join(f'\t_ "graft/server/modules/{module_id}"' for module_id in module_ids)
    registry.write_text(f"package moduleregistry\n\nimport (\n{imports}\n)\n", encoding="utf-8")


def write_descriptor_fixture(root: Path, module_id: str, migration_path: str) -> None:
    descriptor = root / "server" / "modules" / module_id / "descriptor.go"
    descriptor.parent.mkdir(parents=True, exist_ok=True)
    descriptor.write_text(
        f'package {module_id.replace("-", "")}\n\n'
        "func NewModuleSpec() any {\n"
        "\treturn struct{\n"
        "\t\tMigrationPath []string\n"
        f'\t}}{{MigrationPath: []string{{"{migration_path}"}}}}\n'
        "}\n",
        encoding="utf-8",
    )


def write_atlas_state(path: Path) -> None:
    (path / "atlas.sum").write_text("atlas\n", encoding="utf-8")


class CandidateDirsTest(unittest.TestCase):
    def test_changed_mode_includes_all_default_chain_dirs_for_global_conflict_checks(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            httpx_dir = root / "server" / "internal" / "httpx" / "migrations"
            user_dir = root / "server" / "modules" / "user" / "migrations"
            rbac_dir = root / "server" / "modules" / "rbac" / "migrations"
            monitor_dir = root / "server" / "modules" / "monitor" / "migrations"
            httpx_dir.mkdir(parents=True)
            user_dir.mkdir(parents=True)
            rbac_dir.mkdir(parents=True)
            monitor_dir.mkdir(parents=True)
            write_registry_fixture(root, ["user", "rbac"])
            write_descriptor_fixture(root, "user", "modules/user/migrations")
            write_descriptor_fixture(root, "rbac", "modules/rbac/migrations")
            write_atlas_state(httpx_dir)
            write_atlas_state(user_dir)
            write_atlas_state(rbac_dir)
            write_atlas_state(monitor_dir)

            with patch(
                "check_migration_versions.subprocess.check_output",
                return_value="server/modules/user/migrations/202605280001_user.sql\nserver/modules/user/README.md\n",
            ):
                dirs = candidate_dirs(root, "changed")

            self.assertEqual(dirs, [httpx_dir, rbac_dir, user_dir])

    def test_all_mode_excludes_historical_shared_migration_dir(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            httpx_dir = root / "server" / "internal" / "httpx" / "migrations"
            historical_dir = root / "server" / "internal" / "ent" / "migrate" / "migrations"
            user_dir = root / "server" / "modules" / "user" / "migrations"
            httpx_dir.mkdir(parents=True)
            historical_dir.mkdir(parents=True)
            user_dir.mkdir(parents=True)
            write_registry_fixture(root, ["user"])
            write_descriptor_fixture(root, "user", "modules/user/migrations")
            write_atlas_state(httpx_dir)
            write_atlas_state(historical_dir)
            write_atlas_state(user_dir)

            dirs = candidate_dirs(root, "all")

            self.assertEqual(dirs, [httpx_dir, user_dir])

    def test_all_mode_excludes_registry_dirs_without_atlas_state(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            httpx_dir = root / "server" / "internal" / "httpx" / "migrations"
            user_dir = root / "server" / "modules" / "user" / "migrations"
            scheduler_dir = root / "server" / "modules" / "scheduler" / "migrations"
            httpx_dir.mkdir(parents=True)
            user_dir.mkdir(parents=True)
            scheduler_dir.mkdir(parents=True)
            write_registry_fixture(root, ["user", "scheduler"])
            write_descriptor_fixture(root, "user", "modules/user/migrations")
            write_descriptor_fixture(root, "scheduler", "modules/scheduler/migrations")
            write_atlas_state(httpx_dir)
            write_atlas_state(user_dir)

            dirs = candidate_dirs(root, "all")

            self.assertEqual(dirs, [httpx_dir, user_dir])


class ValidateTest(unittest.TestCase):
    def test_validate_reports_duplicate_versions(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            httpx_dir = root / "server" / "internal" / "httpx" / "migrations"
            user_dir = root / "server" / "modules" / "user" / "migrations"
            httpx_dir.mkdir(parents=True)
            user_dir.mkdir(parents=True)
            (httpx_dir / "202605280001_access_log.sql").write_text("SELECT 1;\n", encoding="utf-8")
            (user_dir / "202605280001_user.sql").write_text("SELECT 1;\n", encoding="utf-8")

            errors = validate([httpx_dir, user_dir], root)

            self.assertEqual(
                errors,
                [
                    "default migration chain version conflict: 202605280001 appears in "
                    "server/internal/httpx/migrations/202605280001_access_log.sql, "
                    "server/modules/user/migrations/202605280001_user.sql"
                ],
            )

    def test_validate_allows_unique_versions(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            user_dir = root / "server" / "modules" / "user" / "migrations"
            rbac_dir = root / "server" / "modules" / "rbac" / "migrations"
            user_dir.mkdir(parents=True)
            rbac_dir.mkdir(parents=True)
            (user_dir / "202605280001_user.sql").write_text("SELECT 1;\n", encoding="utf-8")
            (rbac_dir / "202605280002_rbac.sql").write_text("SELECT 1;\n", encoding="utf-8")

            errors = validate([user_dir, rbac_dir], root)

            self.assertEqual(errors, [])


if __name__ == "__main__":
    unittest.main()
