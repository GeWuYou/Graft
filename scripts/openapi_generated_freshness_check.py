#!/usr/bin/env python3

from __future__ import annotations

import argparse
import difflib
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


REPO_SENTINEL = "AGENTS.md"
MONITOR_TARGET = Path("server/internal/contract/openapi/monitor/zz_generated.types.go")
RBAC_PERMISSIONS_TARGET = Path("server/internal/contract/openapi/rbac/zz_generated.permissions.go")
MONITOR_SPEC = Path("openapi/openapi.yaml")
SERVER_MODULE_ROOT = Path("server")
MONITOR_ARGS = [
    "--include-operation-ids",
    "getMonitorServerStatus",
    "--generate",
    "types",
    "--package",
    "monitor",
]
RBAC_PERMISSIONS_ARGS = [
    "--include-operation-ids",
    "getPermissions",
    "--generate",
    "types",
    "--package",
    "rbacopenapi",
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Check or regenerate repository-owned OpenAPI generated artifacts without editing them by default."
    )
    parser.add_argument(
        "--target",
        choices=["backend-monitor", "backend-rbac-permissions"],
        default="backend-monitor",
        help="Generated artifact target to validate.",
    )
    parser.add_argument(
        "--mode",
        choices=["check", "fix"],
        default="check",
        help="`check` reports drift only; `fix` overwrites the tracked generated file explicitly.",
    )
    return parser.parse_args()


def find_repo_root() -> Path:
    current = Path.cwd().resolve()
    for candidate in (current, *current.parents):
        if (
            (candidate / REPO_SENTINEL).is_file()
            and (candidate / "openapi").is_dir()
            and (candidate / "server").is_dir()
        ):
            return candidate
    raise SystemExit(f"could not locate repository root containing {REPO_SENTINEL}")


def run_backend_monitor(repo_root: Path, mode: str) -> int:
    return run_generated_target(
        repo_root=repo_root,
        target=MONITOR_TARGET,
        spec=repo_root / MONITOR_SPEC,
        generator_args=MONITOR_ARGS,
        mode=mode,
        temp_prefix="graft-openapi-monitor-",
    )

def run_backend_rbac_permissions(repo_root: Path, mode: str) -> int:
    return run_generated_target(
        repo_root=repo_root,
        target=RBAC_PERMISSIONS_TARGET,
        spec=repo_root / MONITOR_SPEC,
        generator_args=RBAC_PERMISSIONS_ARGS,
        mode=mode,
        temp_prefix="graft-openapi-rbac-permissions-",
    )


def run_generated_target(
    repo_root: Path,
    target: Path,
    spec: Path,
    generator_args: list[str],
    mode: str,
    temp_prefix: str,
) -> int:
    tracked_target = repo_root / target
    server_module_root = repo_root / SERVER_MODULE_ROOT

    with tempfile.TemporaryDirectory(prefix=temp_prefix) as temp_dir:
        temp_output = Path(temp_dir) / tracked_target.name
        command = ["go", "tool", "oapi-codegen", *generator_args, "-o", str(temp_output), str(spec)]
        subprocess.run(command, cwd=server_module_root, check=True)

        actual = tracked_target.read_text(encoding="utf-8")
        expected = temp_output.read_text(encoding="utf-8")
        if actual == expected:
            print(f"{target}: fresh")
            return 0

        if mode == "fix":
            shutil.copyfile(temp_output, tracked_target)
            print(f"{target}: regenerated from {MONITOR_SPEC}")
            return 0

        diff = difflib.unified_diff(
            actual.splitlines(keepends=True),
            expected.splitlines(keepends=True),
            fromfile=str(target),
            tofile=f"{target} (expected regenerated output)",
        )
        sys.stderr.writelines(diff)
        sys.stderr.write(
            "\nbackend generated artifact is stale or manually edited; rerun with "
            "`--mode fix` after confirming the spec and generator inputs are correct.\n"
        )
        return 1


def main() -> int:
    args = parse_args()
    repo_root = find_repo_root()

    if args.target == "backend-monitor":
        return run_backend_monitor(repo_root, args.mode)
    if args.target == "backend-rbac-permissions":
        return run_backend_rbac_permissions(repo_root, args.mode)

    raise SystemExit(f"unsupported target: {args.target}")


if __name__ == "__main__":
    raise SystemExit(main())
