#!/usr/bin/env python3
"""Conservative CI impact selection for changed paths.

Maps repository paths to pull-request validation jobs. Unknown paths and
shared surfaces force the FULL job set; known narrow surfaces select a subset
that always includes the hygiene and summary jobs. The mapping is advisory
for local runs; CI itself always runs the workflow as defined.
"""
from __future__ import annotations

import argparse
import json
from pathlib import Path, PurePosixPath
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]

FULL = "__full__"

# Blocking pull-request-validation jobs (report-only jobs are not selection targets).
ALL_JOBS = frozenset({
    "line-ending-check", "secret-scan", "contract-governance-check", "ai-governance-check",
    "web-check", "cross-boundary-contract-freshness", "openapi-breaking-check",
    "migration-bootstrap", "server-lint", "server-build-test", "local-runtime-acceptance",
    "required-checks",
})

# Jobs that run for every change: hygiene gates and the run summary.
ALWAYS_JOBS = frozenset({"line-ending-check", "secret-scan", "required-checks"})

BACKEND_JOBS = frozenset({
    "server-lint", "server-build-test", "cross-boundary-contract-freshness",
    "openapi-breaking-check", "local-runtime-acceptance",
})
FRONTEND_JOBS = frozenset({"web-check", "local-runtime-acceptance"})
CONTRACT_JOBS = frozenset({
    "cross-boundary-contract-freshness", "openapi-breaking-check",
    "server-build-test", "web-check", "local-runtime-acceptance",
})
GOVERNANCE_JOBS = frozenset({"ai-governance-check"})

SHARED_SURFACE_PATHS = (
    "Justfile", "package.json", "bun.lock", "turbo.json",
    "server/go.mod", "server/go.sum", "server/package.json", "web/package.json",
    ".golangci.yml", ".editorconfig", ".gitattributes", ".gitignore",
)

JOB_SUITE_GROUPS = {
    "ai-governance-check": ["governance"],
    "cross-boundary-contract-freshness": ["openapi"],
    "openapi-breaking-check": ["openapi"],
    "server-lint": ["core"],
    "server-build-test": ["core"],
    "migration-bootstrap": ["compose"],
    "web-check": ["frontend-source", "frontend-artifacts"],
    "local-runtime-acceptance": ["compose"],
}


def _rules() -> list[tuple[str, object, str]]:
    """Ordered (rule name, matcher, reason) rules; all matching rules accumulate."""
    return [
        ("workflow-definition", prefix(".github/workflows/"),
         "CI workflow definition change affects validation behavior itself"),
        ("repository-automation", prefix(".github/"),
         "repository automation surface"),
        ("backend-source", prefix("server/"),
         "backend source change"),
        ("frontend-source", prefix("web/"),
         "frontend source change"),
        ("contract-source", prefix("openapi/"),
         "cross-boundary contract source change"),
        ("governance-recovery", prefix("ai-plan/"),
         "governance recovery surface"),
        ("governance-skills", prefix(".agents/"),
         "governance skill surface"),
        ("root-governance", exact("AGENTS.md"),
         "root governance file"),
        ("documentation", prefix("docs/"),
         "documentation surface"),
        ("markdown", suffix(".md"),
         "documentation surface"),
        ("shared-automation", prefix("scripts/"),
         "scripts are consumed by several CI gates; shared surface"),
        ("toolchain", any_of(SHARED_SURFACE_PATHS),
         "shared toolchain or build input"),
        ("runtime-images", any_of_startswith(("Dockerfile", "compose")),
         "image or compose surface feeds runtime acceptance gates"),
        ("sql-migration", sql_migration(),
         "database migration asset"),
    ]


def prefix(value: str):
    return lambda path: path.startswith(value)


def suffix(value: str):
    return lambda path: path.endswith(value)


def exact(value: str):
    return lambda path: path == value


def any_of(values: tuple[str, ...]):
    return lambda path: path in values


def any_of_startswith(values: tuple[str, ...]):
    return lambda path: any(path.startswith(value) or f"/{value}" in path for value in values)


def sql_migration():
    return lambda path: path.endswith(".sql") and "/migrations/" in path


def select_jobs(files: list[str]) -> dict:
    """Map changed paths to jobs; unknown or shared-surface changes force FULL."""
    if not files:
        return {
            "mode": "full", "jobs": sorted(ALL_JOBS), "reasons": [],
            "full_reasons": ["no changed paths provided; forcing FULL"],
            "suite_groups": sorted({group for job in ALL_JOBS for group in JOB_SUITE_GROUPS.get(job, [])}),
        }
    jobs: set[str] = set()
    reasons: list[dict] = []
    full_reasons: list[str] = []
    for name in files:
        path = PurePosixPath(name).as_posix().removeprefix("./")
        matched = False
        for rule_name, matcher, reason in _rules():
            if not matcher(path):
                continue
            matched = True
            reasons.append({"path": path, "rule": rule_name, "reason": reason})
            targets = {
                "workflow-definition": FULL,
                "repository-automation": FULL,
                "shared-automation": FULL,
                "toolchain": FULL,
                "runtime-images": FULL,
                "backend-source": BACKEND_JOBS,
                "frontend-source": FRONTEND_JOBS,
                "contract-source": CONTRACT_JOBS,
                "governance-recovery": GOVERNANCE_JOBS,
                "governance-skills": GOVERNANCE_JOBS,
                "root-governance": GOVERNANCE_JOBS,
                "documentation": GOVERNANCE_JOBS,
                "markdown": GOVERNANCE_JOBS,
                "sql-migration": frozenset({"migration-bootstrap"}),
            }[rule_name]
            if targets == FULL:
                full_reasons.append(f"{path}: {reason}")
                jobs = set(ALL_JOBS)
            else:
                jobs |= set(targets)
        if not matched:
            full_reasons.append(f"{path}: unknown path; no conservative mapping exists")
            jobs = set(ALL_JOBS)
    mode = "full" if full_reasons else "subset"
    if mode == "full":
        jobs = set(ALL_JOBS)
    else:
        jobs |= ALWAYS_JOBS
    return {
        "mode": mode,
        "jobs": sorted(jobs),
        "reasons": reasons,
        "full_reasons": full_reasons,
        "suite_groups": sorted({
            group for job in jobs for group in JOB_SUITE_GROUPS.get(job, [])
        }),
    }


def worktree_files(root: Path) -> list[str] | None:
    try:
        paths = subprocess.check_output(
            ["git", "diff", "HEAD", "--name-only", "-z"], cwd=root,
        ).decode().split("\0")
        untracked = subprocess.check_output(
            ["git", "ls-files", "--others", "--exclude-standard", "-z"], cwd=root,
        ).decode().split("\0")
        return sorted(set(filter(None, paths + untracked)))
    except (subprocess.CalledProcessError, OSError):
        return None


def base_files(root: Path, base: str) -> list[str] | None:
    try:
        merge_base = subprocess.check_output(
            ["git", "merge-base", base, "HEAD"], cwd=root,
        ).decode().strip()
        paths = subprocess.check_output(
            ["git", "diff", "--name-only", "-z", f"{merge_base}..HEAD"], cwd=root,
        ).decode().split("\0")
        return sorted(set(filter(None, paths)))
    except (subprocess.CalledProcessError, OSError):
        return None


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--file", action="append", default=[],
                        help="Repository-relative changed file, repeatable.")
    parser.add_argument("--base", default=None,
                        help="Compare against a base ref (merge-base .. HEAD) instead of the worktree.")
    parser.add_argument("--json", action="store_true", help="Print the selection JSON to stdout.")
    args = parser.parse_args(argv)

    if args.file:
        files = args.file
        source = "explicit"
    elif args.base:
        files = base_files(ROOT, args.base)
        source = f"base {args.base}"
        if files is None:
            print(json.dumps({
                "mode": "full", "jobs": sorted(ALL_JOBS),
                "reasons": [],
                "full_reasons": [f"could not enumerate changes against base {args.base}; forcing FULL"],
                "suite_groups": sorted({group for job in ALL_JOBS for group in JOB_SUITE_GROUPS.get(job, [])}),
            }, indent=2))
            return 0
    else:
        files = worktree_files(ROOT)
        source = "worktree"
        if files is None:
            print("Could not enumerate worktree changes with git.", file=sys.stderr)
            return 2
    selection = select_jobs(files)
    selection["source"] = source
    selection["files"] = sorted(set(files))
    if args.json:
        print(json.dumps(selection, indent=2))
    else:
        print(f"CI impact ({source}): mode={selection['mode']} suite_groups={selection['suite_groups'] or []}")
        for job in selection["jobs"]:
            print(f"  {job}")
        for reason in selection["full_reasons"]:
            print(f"  FULL forced: {reason}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
