#!/usr/bin/env python3
"""Validate empty-database bootstrap and historical-database migration upgrades."""

from __future__ import annotations

import argparse
from dataclasses import dataclass
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
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
IMAGE_PULL_ATTEMPTS = 3
IMAGE_PULL_DELAY_SECONDS = 2
IMAGE_PULL_TIMEOUT_SECONDS = 120

@dataclass(frozen=True)
class BootstrapTarget:
    container_name: str
    port: int


class CommandError(RuntimeError):
    """Raised when an external command fails after its output has been reported."""

    def __init__(self, message: str, *, stdout: str = "") -> None:
        super().__init__(message)
        self.stdout = stdout


def run_command(
    command: list[str],
    *,
    cwd: Path | None = None,
    env: dict[str, str] | None = None,
    check: bool = True,
    timeout: float | None = None,
) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(command, cwd=cwd, env=env, text=True, capture_output=True, check=False, timeout=timeout)
    if check and result.returncode != 0:
        raise CommandError(format_command_failure(command, result), stdout=result.stdout)
    return result


def format_command_failure(command: list[str], result: subprocess.CompletedProcess[str]) -> str:
    output = (result.stdout + result.stderr).strip()
    return f"command failed ({result.returncode}): {' '.join(command)}\n{output}"


def pull_postgres_image() -> None:
    command = ["docker", "pull", POSTGRES_IMAGE]
    last_error: CommandError | None = None
    for attempt in range(1, IMAGE_PULL_ATTEMPTS + 1):
        try:
            result = run_command(command, check=False, timeout=IMAGE_PULL_TIMEOUT_SECONDS)
        except subprocess.TimeoutExpired:
            last_error = CommandError(
                f"command timed out after {IMAGE_PULL_TIMEOUT_SECONDS} seconds: {' '.join(command)}"
            )
        else:
            if result.returncode == 0:
                return
            last_error = CommandError(format_command_failure(command, result), stdout=result.stdout)
        if attempt < IMAGE_PULL_ATTEMPTS:
            print(
                f"PostgreSQL image pull attempt {attempt}/{IMAGE_PULL_ATTEMPTS} failed; retrying.",
                file=sys.stderr,
            )
            time.sleep(IMAGE_PULL_DELAY_SECONDS)
    assert last_error is not None
    raise last_error


def start_postgres(container_name: str) -> BootstrapTarget:
    pull_postgres_image()
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
            "GRAFT_CONFIG_SCHEMA_VERSION": "1",
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


def materialize_historical_server(ref: str, destination: Path) -> Path:
    archive_path = destination / "historical-server.tar"
    run_command(
        ["git", "archive", "--format=tar", f"--output={archive_path}", ref, "server"],
        cwd=REPO_ROOT,
    )
    run_command(["tar", "--extract", "--file", str(archive_path), "--directory", str(destination)])
    historical_server = destination / "server"
    if not historical_server.is_dir():
        raise RuntimeError(f"historical migration source {ref!r} does not contain server/")
    return historical_server


def apply_historical_migrations(target: BootstrapTarget, ref: str) -> None:
    with tempfile.TemporaryDirectory(prefix="graft-migration-upgrade-") as temporary_directory:
        historical_server = materialize_historical_server(ref, Path(temporary_directory))
        run_command(
            ["go", "run", "./cmd/graft", "migrate", "up", "--allow-dirty"],
            cwd=historical_server,
            env=migration_environment(target),
        )


def check_schema_contract(target: BootstrapTarget) -> str:
    result = run_command(
        ["go", "run", "./cmd/graft", "migrate", "check-schema", "--mode", "enforce", "--format", "json"],
        cwd=SERVER_DIR,
        env=migration_environment(target),
    )
    return result.stdout


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
    parser.add_argument("--schema-report", type=Path, help="write the PostgreSQL catalog report to this path")
    parser.add_argument(
        "--upgrade-from",
        help="Git ref whose server migration chain is applied before the current default chain",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    container_name = f"graft-migration-bootstrap-{uuid.uuid4().hex[:12]}"
    print(f"starting disposable PostgreSQL container {container_name}")
    try:
        target = start_postgres(container_name)
        wait_for_postgres(target)
        if args.upgrade_from is not None:
            print(f"applying historical migration chain from {args.upgrade_from}")
            apply_historical_migrations(target, args.upgrade_from)
        apply_migrations(target)
        schema_report = check_schema_contract(target)
        if args.schema_report is not None:
            args.schema_report.write_text(schema_report, encoding="utf-8")
    except (CommandError, RuntimeError) as error:
        if args.schema_report is not None and isinstance(error, CommandError) and error.stdout:
            try:
                json.loads(error.stdout)
            except json.JSONDecodeError:
                pass
            else:
                args.schema_report.write_text(error.stdout, encoding="utf-8")
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
