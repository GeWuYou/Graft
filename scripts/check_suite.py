#!/usr/bin/env python3
"""Run grouped repository checks sequentially with per-check logs and a deadline.

Groups mirror the repository entrypoints in the Justfile and pull-request
validation workflow; they do not replace the full `just check` acceptance
gates. Checks fail independently, dependencies are resolved across groups,
and the suite exits nonzero when any check failed, timed out, or the
process-level deadline was exceeded.
"""
from __future__ import annotations

import argparse
from dataclasses import dataclass
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import time
from typing import TextIO

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_DEADLINE_SECONDS = 3600
DEFAULT_ARTIFACTS_DIR = ".artifacts/checks"


@dataclass(frozen=True)
class SuiteCheck:
    name: str
    argv: list[str]
    cwd: str = "."
    requires: tuple[str, ...] = ()
    group: str = ""


def _python(argv_extra: list[str]) -> list[str]:
    return [sys.executable, "-B", *argv_extra]


GROUPS: dict[str, tuple[SuiteCheck, ...]] = {
    "governance": (
        SuiteCheck("ai-governance", _python(["scripts/validate_ai_governance.py"]), group="governance"),
        SuiteCheck("plan-structure", _python(["scripts/validate_ai_plan_structure.py"]), group="governance"),
        SuiteCheck("shared-asset-registries", _python(["scripts/validate_shared_asset_registries.py"]), group="governance"),
        SuiteCheck(
            "governance-python-tests",
            _python(["-m", "unittest",
                     "scripts/test_local_validation.py",
                     "scripts/test_governance_workflow.py",
                     "scripts/test_ci_runtime_artifacts.py",
                     "scripts/test_build_runtime_acceptance.py",
                     "scripts/test_check_suite.py",
                     "scripts/test_ci_impact.py"]),
            group="governance",
        ),
        SuiteCheck("turbo-tests", ["node", "--test", "scripts/test-turbo.mjs"], group="governance"),
    ),
    "openapi": (
        SuiteCheck("openapi-bundle", ["node", "scripts/ensure-openapi-bundle.mjs"], group="openapi"),
        SuiteCheck("openapi-check", ["just", "openapi-check"], group="openapi"),
        SuiteCheck("openapi-breaking", ["just", "openapi-breaking"], group="openapi"),
    ),
    "core": (
        SuiteCheck("server-lint", ["just", "lint-server"], group="core"),
        SuiteCheck("server-buildtest", ["just", "buildtest-server"], requires=("openapi-bundle",), group="core"),
    ),
    "compose": (
        SuiteCheck("migration-governance", ["just", "migration-check"], group="compose"),
        SuiteCheck("compose-config", ["docker", "compose", "config", "--quiet"], group="compose"),
    ),
    "frontend-source": (
        SuiteCheck("web-lint-i18n", ["bun", "run", "lint:i18n"], cwd="web", group="frontend-source"),
        SuiteCheck("web-lint", ["bun", "run", "lint"], cwd="web", group="frontend-source"),
        SuiteCheck("web-stylelint", ["bun", "run", "stylelint"], cwd="web", group="frontend-source"),
        SuiteCheck("web-contract-changed", ["bun", "run", "contract:check:changed"], cwd="web", group="frontend-source"),
    ),
    "frontend-artifacts": (
        SuiteCheck("web-typecheck", ["bun", "run", "typecheck"], cwd="web", group="frontend-artifacts"),
        SuiteCheck("web-tests", ["bun", "run", "test:run"], cwd="web", group="frontend-artifacts"),
        SuiteCheck("web-build", ["bun", "run", "build"], cwd="web", requires=("web-typecheck",), group="frontend-artifacts"),
    ),
}


def resolve_checks(group_names: list[str], registry: dict[str, tuple[SuiteCheck, ...]] = GROUPS) -> list[SuiteCheck]:
    """Resolve named groups into an ordered check list, auto-including requires."""
    for name in group_names:
        if name not in registry:
            raise ValueError(f"Unknown check group: {name} (available: {', '.join(sorted(registry))})")
    ordered: list[SuiteCheck] = []
    index: dict[str, SuiteCheck] = {}
    for group in group_names:
        for check in registry[group]:
            if check.name not in index:
                ordered.append(check)
                index[check.name] = check

    resolved: list[SuiteCheck] = []
    resolved_index: dict[str, SuiteCheck] = {}
    all_checks = {check.name: check for checks in registry.values() for check in checks}

    def visit(check: SuiteCheck, chain: tuple[str, ...]) -> None:
        if check.name in resolved_index:
            return
        if check.name in chain:
            raise ValueError(f"Cyclic check dependency: {' -> '.join(chain + (check.name,))}")
        for dependency in check.requires:
            if dependency not in all_checks:
                raise ValueError(f"Check {check.name} requires unknown check {dependency}")
            visit(all_checks[dependency], chain + (check.name,))
        resolved.append(check)
        resolved_index[check.name] = check

    for check in ordered:
        visit(check, ())
    return resolved


def kill_process_group(process: subprocess.Popen) -> None:
    """Terminate the check and its children so the deadline is process-level."""
    if os.name == "posix":
        try:
            os.killpg(os.getpgid(process.pid), signal.SIGKILL)
        except (ProcessLookupError, PermissionError, OSError):
            process.kill()
    else:
        process.kill()
    try:
        process.wait(timeout=10)
    except subprocess.TimeoutExpired:
        pass


def run_checks(
    checks: list[SuiteCheck],
    root: Path,
    deadline_seconds: float,
    artifacts_dir: Path | None,
    requested_groups: list[str],
    progress: TextIO | None = None,
) -> tuple[int, dict]:
    progress = progress if progress is not None else sys.stdout
    started = time.monotonic()
    deadline_at = started + deadline_seconds
    results: list[dict] = []
    statuses: dict[str, str] = {}
    log_paths: dict[str, str] = {}
    if artifacts_dir is not None:
        artifacts_dir.mkdir(parents=True, exist_ok=True)
        write_gitignore(artifacts_dir)

    for check in checks:
        if artifacts_dir is not None:
            log_dir = artifacts_dir / check.group
            log_dir.mkdir(parents=True, exist_ok=True)
            log_path = log_dir / f"{check.name}.log"
            try:
                log_paths[check.name] = str(log_path.relative_to(root))
            except ValueError:
                log_paths[check.name] = str(log_path)
        remaining = deadline_at - time.monotonic()
        if remaining <= 0:
            statuses[check.name] = "failed"
            results.append({
                "name": check.name, "group": check.group, "status": "failed", "exit": None,
                "seconds": 0.0, "reason": "suite deadline exceeded before the check started",
            })
            print(f"DEADLINE {check.group}/{check.name}", file=progress, flush=True)
            continue
        dependency_failure = next(
            (name for name in check.requires if statuses.get(name) != "passed"),
            None,
        )
        if dependency_failure is not None:
            statuses[check.name] = "skipped"
            results.append({
                "name": check.name, "group": check.group, "status": "skipped", "exit": None,
                "seconds": 0.0, "reason": f"dependency {dependency_failure} did not pass",
            })
            print(f"SKIP {check.group}/{check.name} (dependency {dependency_failure})", file=progress, flush=True)
            continue
        print(f"RUN {check.group}/{check.name}", file=progress, flush=True)
        check_started = time.monotonic()
        status = "passed"
        reason = None
        code = None
        log_handle = None
        try:
            if artifacts_dir is not None:
                log_handle = open(log_path, "w", encoding="utf-8")
            popen_kwargs = {"stdout": log_handle or subprocess.DEVNULL,
                            "stderr": subprocess.STDOUT}
            if os.name == "posix":
                popen_kwargs["start_new_session"] = True
            process = subprocess.Popen(check.argv, cwd=root / check.cwd, **popen_kwargs)
            try:
                code = process.wait(timeout=remaining)
            except subprocess.TimeoutExpired:
                kill_process_group(process)
                status = "timeout"
                reason = f"killed after exceeding the suite deadline budget ({remaining:.1f}s remaining)"
        except OSError as error:
            status = "failed"
            reason = f"missing prerequisite: {error}"
            code = 127
        finally:
            if log_handle is not None:
                log_handle.close()
        seconds = time.monotonic() - check_started
        if code is not None and status != "timeout":
            status = "passed" if code == 0 else "failed"
        if status == "failed" and reason is None:
            reason = f"exit code {code}"
        statuses[check.name] = status
        results.append({
            "name": check.name, "group": check.group, "status": status, "exit": code,
            "seconds": round(seconds, 2), "reason": reason,
        })
        marker = "PASS" if status == "passed" else status.upper()
        print(f"{marker} {check.group}/{check.name} seconds={seconds:.2f}", file=progress, flush=True)

    failed = [item for item in results if item["status"] in ("failed", "timeout")]
    code = 1 if failed else 0
    report = {
        "suite": "graft-check-suite",
        "head": head_commit(root),
        "groups": requested_groups,
        "deadline_seconds": deadline_seconds,
        "elapsed_seconds": round(time.monotonic() - started, 2),
        "checks": [{"name": check.name, "group": check.group, "argv": check.argv,
                    "cwd": check.cwd, "requires": list(check.requires)} for check in checks],
        "results": results,
        "logs": log_paths,
        "summary": {
            "passed": sum(1 for item in results if item["status"] == "passed"),
            "failed": len(failed),
            "skipped": sum(1 for item in results if item["status"] == "skipped"),
            "total": len(results),
        },
    }
    return code, report


def head_commit(root: Path) -> str | None:
    try:
        result = subprocess.run(["git", "rev-parse", "HEAD"], cwd=root, check=False, capture_output=True)
    except OSError:
        return None
    if result.returncode != 0:
        return None
    return result.stdout.decode().strip() or None


def write_gitignore(directory: Path) -> None:
    """Keep runtime check logs out of git status without editing the repo .gitignore."""
    marker = directory / ".gitignore"
    if not marker.exists():
        marker.write_text("*\n", encoding="utf-8")


def print_summary(report: dict, progress: TextIO | None = None) -> None:
    progress = progress if progress is not None else sys.stdout
    summary = report["summary"]
    print(
        f"SUITE {summary['passed']} passed, {summary['failed']} failed, "
        f"{summary['skipped']} skipped, {summary['total']} total, "
        f"elapsed {report['elapsed_seconds']}s",
        file=progress, flush=True,
    )
    for item in report["results"]:
        if item["status"] != "passed":
            print(f"  {item['status'].upper()} {item['group']}/{item['name']}: {item['reason']}",
                  file=progress, flush=True)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--groups", default="",
                        help="Comma-separated check groups (default: all groups).")
    parser.add_argument("--list-groups", action="store_true",
                        help="Print the available groups and their checks, then exit.")
    parser.add_argument("--dry-run", action="store_true",
                        help="Print the resolved check plan without executing it.")
    parser.add_argument("--json", action="store_true", help="Print the suite report JSON to stdout.")
    parser.add_argument("--deadline-seconds", type=float, default=DEFAULT_DEADLINE_SECONDS,
                        help=f"Process-level deadline for the whole suite (default {DEFAULT_DEADLINE_SECONDS}).")
    parser.add_argument("--artifacts-dir", default=DEFAULT_ARTIFACTS_DIR,
                        help=f"Directory for per-check logs and the suite report (default {DEFAULT_ARTIFACTS_DIR}).")
    parser.add_argument("--no-artifacts", action="store_true", help="Do not write artifact logs.")
    args = parser.parse_args(argv)

    group_names = [name.strip() for name in args.groups.split(",") if name.strip()]
    if args.list_groups:
        for group, checks in GROUPS.items():
            print(f"{group}:")
            for check in checks:
                requires = f" (requires: {', '.join(check.requires)})" if check.requires else ""
                print(f"  {check.name}: {check.argv}{requires}")
        return 0
    try:
        checks = resolve_checks(group_names or list(GROUPS))
    except ValueError as error:
        print(error, file=sys.stderr)
        return 2
    if args.dry_run:
        print(json.dumps({
            "suite": "graft-check-suite", "groups": group_names or list(GROUPS),
            "checks": [{"name": check.name, "group": check.group, "argv": check.argv,
                        "cwd": check.cwd, "requires": list(check.requires)} for check in checks],
        }, indent=2))
        return 0

    artifacts_dir = None
    if not args.no_artifacts:
        artifacts_dir = Path(args.artifacts_dir)
        if not artifacts_dir.is_absolute():
            artifacts_dir = ROOT / artifacts_dir
    progress = sys.stderr if args.json else sys.stdout
    code, report = run_checks(checks, ROOT, args.deadline_seconds, artifacts_dir,
                              group_names or list(GROUPS), progress=progress)
    if artifacts_dir is not None:
        report_path = artifacts_dir / "suite-report.json"
        report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        try:
            report["report"] = str(report_path.relative_to(ROOT))
        except ValueError:
            report["report"] = str(report_path)
    print_summary(report, progress)
    if args.json:
        print(json.dumps(report, indent=2))
    return code


if __name__ == "__main__":
    raise SystemExit(main())
