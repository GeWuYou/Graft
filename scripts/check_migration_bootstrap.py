#!/usr/bin/env python3
"""Apply Graft's full migration chain to an empty disposable PostgreSQL database."""

from __future__ import annotations

import argparse
from dataclasses import dataclass
import os
from pathlib import Path
import subprocess
import sys
import time
import uuid


REPO_ROOT = Path(__file__).resolve().parent.parent
SERVER_DIR = REPO_ROOT / "server"
POSTGRES_IMAGE = "postgres:16-alpine"
POSTGRES_USER = "graft"
POSTGRES_PASSWORD = "graft"
POSTGRES_DATABASE = "graft"
READINESS_ATTEMPTS = 30
READINESS_DELAY_SECONDS = 1
REQUIRED_STABLE_READINESS_CHECKS = 3

COMPOSITE_FOREIGN_KEY_QUERY = """
SELECT EXISTS (
  SELECT 1
  FROM pg_constraint AS foreign_key
  JOIN pg_class AS relation ON relation.oid = foreign_key.conrelid
  JOIN pg_namespace AS namespace ON namespace.oid = relation.relnamespace
  WHERE namespace.nspname = current_schema()
    AND relation.relname = 'task_external_receipts'
    AND foreign_key.conname = 'task_external_receipts_task_stage_fkey'
    AND foreign_key.contype = 'f'
    AND pg_get_constraintdef(foreign_key.oid) LIKE
      'FOREIGN KEY (task_id, stage_id) REFERENCES task_stages(task_id, id)%'
);
"""

LEGACY_SINGLE_STAGE_FOREIGN_KEY_QUERY = """
SELECT NOT EXISTS (
  SELECT 1
  FROM pg_constraint AS foreign_key
  JOIN pg_class AS relation ON relation.oid = foreign_key.conrelid
  JOIN pg_namespace AS namespace ON namespace.oid = relation.relnamespace
  WHERE namespace.nspname = current_schema()
    AND relation.relname = 'task_external_receipts'
    AND foreign_key.contype = 'f'
    AND pg_get_constraintdef(foreign_key.oid) LIKE
      'FOREIGN KEY (stage_id) REFERENCES task_stages(id)%'
);
"""


@dataclass(frozen=True)
class BootstrapTarget:
    container_name: str
    port: int


class CommandError(RuntimeError):
    """Raised when an external command fails after its output has been reported."""


def run_command(
    command: list[str], *, cwd: Path | None = None, env: dict[str, str] | None = None, check: bool = True
) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(command, cwd=cwd, env=env, text=True, capture_output=True, check=False)
    if check and result.returncode != 0:
        raise CommandError(format_command_failure(command, result))
    return result


def format_command_failure(command: list[str], result: subprocess.CompletedProcess[str]) -> str:
    output = (result.stdout + result.stderr).strip()
    return f"command failed ({result.returncode}): {' '.join(command)}\n{output}"


def start_postgres(container_name: str) -> BootstrapTarget:
    run_command(
        [
            "docker",
            "run",
            "--detach",
            "--name",
            container_name,
            "--publish",
            "127.0.0.1::5432",
            "--env",
            f"POSTGRES_USER={POSTGRES_USER}",
            "--env",
            f"POSTGRES_PASSWORD={POSTGRES_PASSWORD}",
            "--env",
            f"POSTGRES_DB={POSTGRES_DATABASE}",
            POSTGRES_IMAGE,
        ]
    )
    port_output = run_command(["docker", "port", container_name, "5432/tcp"]).stdout.strip()
    _, separator, port_text = port_output.rpartition(":")
    if not separator or not port_text.isdigit():
        raise RuntimeError(f"unable to resolve PostgreSQL host port from {port_output!r}")
    return BootstrapTarget(container_name=container_name, port=int(port_text))


def wait_for_postgres(target: BootstrapTarget) -> None:
    command = ["docker", "exec", target.container_name, "pg_isready", "-U", POSTGRES_USER, "-d", POSTGRES_DATABASE]
    stable_checks = 0
    for _ in range(READINESS_ATTEMPTS):
        if run_command(command, check=False).returncode == 0:
            stable_checks += 1
            if stable_checks >= REQUIRED_STABLE_READINESS_CHECKS:
                return
        else:
            stable_checks = 0
        time.sleep(READINESS_DELAY_SECONDS)
    raise RuntimeError(f"PostgreSQL container {target.container_name} did not become ready")


def migration_environment(target: BootstrapTarget) -> dict[str, str]:
    environment = os.environ.copy()
    environment.update(
        {
            "GRAFT_APP_ENV": "ci",
            "GRAFT_DATABASE_DRIVER": "postgres",
            "GRAFT_DATABASE_URL": (
                f"postgres://{POSTGRES_USER}:{POSTGRES_PASSWORD}@127.0.0.1:{target.port}/"
                f"{POSTGRES_DATABASE}?sslmode=disable"
            ),
            "GRAFT_REDIS_ADDR": "127.0.0.1:6379",
            "GRAFT_AUTH_JWT_SECRET": "migration-bootstrap-ci-secret",
        }
    )
    return environment


def apply_migrations(target: BootstrapTarget) -> None:
    run_command(
        ["go", "run", "./cmd/graft", "migrate", "up", "--allow-dirty"],
        cwd=SERVER_DIR,
        env=migration_environment(target),
    )


def query_boolean(target: BootstrapTarget, query: str) -> bool:
    result = run_command(
        [
            "docker",
            "exec",
            target.container_name,
            "psql",
            "--tuples-only",
            "--no-align",
            "--set",
            "ON_ERROR_STOP=1",
            "--username",
            POSTGRES_USER,
            "--dbname",
            POSTGRES_DATABASE,
            "--command",
            query,
        ]
    )
    return result.stdout.strip() == "t"


def assert_task_receipt_constraints(target: BootstrapTarget) -> None:
    if not query_boolean(target, COMPOSITE_FOREIGN_KEY_QUERY):
        raise RuntimeError("task_external_receipts composite task/stage foreign key is missing")
    if not query_boolean(target, LEGACY_SINGLE_STAGE_FOREIGN_KEY_QUERY):
        raise RuntimeError("task_external_receipts still has a legacy single-column stage foreign key")


def print_diagnostics(container_name: str) -> None:
    print(f"\n--- migration bootstrap diagnostics: {container_name} ---", file=sys.stderr)
    for command in (
        ["docker", "ps", "--all", "--filter", f"name={container_name}"],
        ["docker", "logs", container_name],
    ):
        result = run_command(command, check=False)
        if result.stdout:
            print(result.stdout, file=sys.stderr, end="" if result.stdout.endswith("\n") else "\n")
        if result.stderr:
            print(result.stderr, file=sys.stderr, end="" if result.stderr.endswith("\n") else "\n")


def remove_postgres(container_name: str) -> None:
    run_command(["docker", "rm", "--force", container_name], check=False)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--keep-container", action="store_true", help="retain the disposable PostgreSQL container for diagnosis")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    container_name = f"graft-migration-bootstrap-{uuid.uuid4().hex[:12]}"
    print(f"starting disposable PostgreSQL container {container_name}")
    try:
        target = start_postgres(container_name)
        wait_for_postgres(target)
        apply_migrations(target)
        assert_task_receipt_constraints(target)
    except (CommandError, RuntimeError) as error:
        print(f"migration bootstrap failed: {error}", file=sys.stderr)
        print_diagnostics(container_name)
        return 1
    finally:
        if args.keep_container:
            print(f"retaining disposable PostgreSQL container {container_name}")
        else:
            remove_postgres(container_name)
    print("migration bootstrap: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
