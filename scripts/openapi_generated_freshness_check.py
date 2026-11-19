#!/usr/bin/env python3

from __future__ import annotations

import argparse
import difflib
import shlex
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


REPO_SENTINEL = "AGENTS.md"
OPENAPI_BINDINGS = Path("server/internal/contract/openapi")
MONITOR_TARGET = Path("server/internal/contract/openapi/monitor/zz_generated.types.go")
HEALTH_TARGET = Path("server/internal/contract/openapi/health/zz_generated.health.go")
RBAC_MANAGEMENT_TARGET = Path("server/internal/contract/openapi/rbac/zz_generated.management.go")
USER_MANAGEMENT_TARGET = Path("server/internal/contract/openapi/user/zz_generated.management.go")
AUTH_TARGET = Path("server/internal/contract/openapi/auth/zz_generated.auth.go")
MODULES_TARGET = Path("server/internal/contract/openapi/modules/zz_generated.modules.go")
NETWORK_CONNECTIVITY_TARGET = Path("server/internal/contract/openapi/network/zz_generated.connectivity.go")
MONITOR_SPEC = Path("openapi/openapi.yaml")
SERVER_MODULE_ROOT = Path("server")
HEALTH_ARGS = [
    "--include-operation-ids",
    "getHealthz",
    "--generate",
    "types",
    "--package",
    "healthopenapi",
]
MODULES_ARGS = [
    "--include-operation-ids",
    "getModulesRuntime,getModulesRuntimeModule",
    "--generate",
    "types",
    "--package",
    "modulesopenapi",
]
def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Check or regenerate repository-owned OpenAPI generated artifacts without editing them by default."
    )
    parser.add_argument(
        "--target",
        choices=["backend-monitor", "backend-health", "backend-rbac-permissions", "backend-rbac-read", "backend-rbac-management", "backend-user-write", "backend-auth-session", "backend-modules-runtime", "backend-network-connectivity", "backend-openapi-servers"],
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


def strict_server_targets(repo_root: Path) -> list[tuple[str, Path]]:
    """Use package generator directives as the inventory of strict artifacts."""
    targets: list[tuple[str, Path]] = []
    for config in sorted((repo_root / OPENAPI_BINDINGS).glob("*/oapi-codegen.yaml")):
        generator = config.with_name("generate.go")
        directives = [
            shlex.split(line.removeprefix("//go:generate "))
            for line in generator.read_text(encoding="utf-8").splitlines()
            if line.startswith("//go:generate go tool oapi-codegen -config oapi-codegen.yaml -o ")
        ]
        if len(directives) != 1 or len(directives[0]) != 8 or Path(directives[0][6]).name != directives[0][6]:
            raise SystemExit(f"{generator}: expected one local strict server output")
        artifact = config.with_name(directives[0][6]).relative_to(repo_root)
        targets.append((config.parent.name, artifact))
    if not targets:
        raise SystemExit("could not locate any per-domain strict server configs")
    return targets


def run_strict_server_target_by_artifact(repo_root: Path, mode: str, artifact: Path) -> int:
    """Validate one artifact through the per-domain config it is generated from.

    Artifacts regenerated with models + gin-server + strict-server no longer match the
    legacy types-only CLI regeneration, so legacy targets whose artifact moved to a
    per-domain config must compare against that config's output.
    """
    for _, target in strict_server_targets(repo_root):
        if target == artifact:
            return run_strict_server_target(
                repo_root=repo_root,
                target=target,
                config=target.parent / "oapi-codegen.yaml",
                mode=mode,
                temp_prefix=f"graft-openapi-strict-{target.parent.name}-",
            )
    raise SystemExit(f"artifact {artifact} has no per-domain oapi-codegen config target")


def run_backend_monitor(repo_root: Path, mode: str) -> int:
    # monitor/zz_generated.types.go is now generated through its per-domain config
    # (models + gin-server + strict-server); compare against that config's output.
    return run_strict_server_target_by_artifact(repo_root, mode, MONITOR_TARGET)


def run_backend_health(repo_root: Path, mode: str) -> int:
    return run_generated_target(
        repo_root=repo_root,
        target=HEALTH_TARGET,
        spec=repo_root / MONITOR_SPEC,
        generator_args=HEALTH_ARGS,
        mode=mode,
        temp_prefix="graft-openapi-health-",
    )


def run_backend_rbac_permissions(repo_root: Path, mode: str) -> int:
    return run_strict_server_target_by_artifact(repo_root, mode, RBAC_MANAGEMENT_TARGET)


def run_backend_rbac_read(repo_root: Path, mode: str) -> int:
    return run_strict_server_target_by_artifact(repo_root, mode, RBAC_MANAGEMENT_TARGET)


def run_backend_rbac_management(repo_root: Path, mode: str) -> int:
    return run_strict_server_target_by_artifact(repo_root, mode, RBAC_MANAGEMENT_TARGET)


def run_backend_user_write(repo_root: Path, mode: str) -> int:
    return run_strict_server_target_by_artifact(repo_root, mode, USER_MANAGEMENT_TARGET)


def run_backend_auth_session(repo_root: Path, mode: str) -> int:
    # auth/zz_generated.auth.go is now generated through its per-domain config
    # (models + gin-server + strict-server); compare against that config's output.
    return run_strict_server_target_by_artifact(repo_root, mode, AUTH_TARGET)


def run_backend_modules_runtime(repo_root: Path, mode: str) -> int:
    return run_generated_target(
        repo_root=repo_root,
        target=MODULES_TARGET,
        spec=repo_root / MONITOR_SPEC,
        generator_args=MODULES_ARGS,
        mode=mode,
        temp_prefix="graft-openapi-modules-runtime-",
    )


def run_backend_network_connectivity(repo_root: Path, mode: str) -> int:
    return run_strict_server_target_by_artifact(repo_root, mode, NETWORK_CONNECTIVITY_TARGET)


def run_backend_openapi_servers(repo_root: Path, mode: str) -> int:
    failed: list[str] = []
    for name, target in strict_server_targets(repo_root):
        config = target.parent / "oapi-codegen.yaml"
        status = run_strict_server_target(
            repo_root=repo_root,
            target=target,
            config=config,
            mode=mode,
            temp_prefix=f"graft-openapi-strict-{name}-",
        )
        if status != 0:
            failed.append(name)
    if failed:
        sys.stderr.write(
            "\nstrict server generated artifacts are stale: "
            + ", ".join(failed)
            + "; rerun `cd server && go generate ./internal/contract/openapi/...` or this script with `--mode fix`.\n"
        )
        return 1
    return 0


def run_strict_server_target(
    repo_root: Path,
    target: Path,
    config: Path,
    mode: str,
    temp_prefix: str,
) -> int:
    tracked_target = repo_root / target
    server_module_root = repo_root / SERVER_MODULE_ROOT
    spec = repo_root / MONITOR_SPEC

    with tempfile.TemporaryDirectory(prefix=temp_prefix) as temp_dir:
        temp_output = Path(temp_dir) / tracked_target.name
        command = [
            "go",
            "tool",
            "oapi-codegen",
            "-config",
            str(repo_root / config),
            "-o",
            str(temp_output),
            str(spec),
        ]
        subprocess.run(command, cwd=server_module_root, check=True)

        actual = tracked_target.read_text(encoding="utf-8")
        expected = temp_output.read_text(encoding="utf-8")
        if actual == expected:
            print(f"{target}: fresh")
            return 0

        if mode == "fix":
            shutil.copyfile(temp_output, tracked_target)
            print(f"{target}: regenerated from {MONITOR_SPEC} via {config.name}")
            return 0

        diff = difflib.unified_diff(
            actual.splitlines(keepends=True),
            expected.splitlines(keepends=True),
            fromfile=str(target),
            tofile=f"{target} (expected regenerated output)",
        )
        sys.stderr.writelines(diff)
        return 1


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
    if args.target == "backend-health":
        return run_backend_health(repo_root, args.mode)
    if args.target == "backend-rbac-permissions":
        return run_backend_rbac_permissions(repo_root, args.mode)
    if args.target == "backend-rbac-read":
        return run_backend_rbac_read(repo_root, args.mode)
    if args.target == "backend-rbac-management":
        return run_backend_rbac_management(repo_root, args.mode)
    if args.target == "backend-user-write":
        return run_backend_user_write(repo_root, args.mode)
    if args.target == "backend-auth-session":
        return run_backend_auth_session(repo_root, args.mode)
    if args.target == "backend-modules-runtime":
        return run_backend_modules_runtime(repo_root, args.mode)
    if args.target == "backend-network-connectivity":
        return run_backend_network_connectivity(repo_root, args.mode)
    if args.target == "backend-openapi-servers":
        return run_backend_openapi_servers(repo_root, args.mode)

    raise SystemExit(f"unsupported target: {args.target}")


if __name__ == "__main__":
    raise SystemExit(main())
