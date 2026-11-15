#!/usr/bin/env python3
"""Plan and run worktree-scoped commit preparation checks without installing tools."""
from __future__ import annotations

import argparse
from dataclasses import asdict, dataclass
import json
from pathlib import Path, PurePosixPath
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[1]
WEB_SOURCE = (".ts", ".tsx", ".js", ".jsx", ".mjs", ".vue")
STYLE_SOURCE = (".vue", ".css", ".less", ".sass", ".scss", ".html")


@dataclass
class Check:
    name: str
    argv: list[str]
    cwd: str = "."


def relative_path(root: Path, name: str) -> str:
    path = Path(name)
    if path.is_absolute() or ".." in path.parts or not name.strip():
        raise ValueError(f"Expected a repository-relative file path: {name}")
    normalized = path.as_posix().removeprefix("./")
    if not (root / normalized).resolve().is_relative_to(root.resolve()):
        raise ValueError(f"Path escapes this checkout: {name}")
    return normalized


def changed_files(root: Path) -> list[str]:
    paths = subprocess.check_output(
        ["git", "diff", "HEAD", "--name-only", "-z"], cwd=root,
    ).decode().split("\0")
    untracked = subprocess.check_output(
        ["git", "ls-files", "--others", "--exclude-standard", "-z"], cwd=root,
    ).decode().split("\0")
    return sorted(set(filter(None, paths + untracked)))


def staged_divergence(root: Path, names: list[str]) -> list[str]:
    staged = set(subprocess.check_output(
        ["git", "diff", "--cached", "--name-only", "-z"], cwd=root,
    ).decode().split("\0"))
    unstaged = set(subprocess.check_output(
        ["git", "diff", "--name-only", "-z"], cwd=root,
    ).decode().split("\0"))
    return sorted(set(names) & staged & unstaged)


def test_family(name: str) -> str | None:
    path = PurePosixPath(name)
    if name.startswith("server/") and name.endswith("_test.go"):
        return "go"
    if name.startswith("web/") and any(marker in name for marker in (".test.", ".spec.")):
        return "web"
    if path.suffix == ".py" and path.name.startswith("test_"):
        return "python"
    if path.suffix == ".mjs" and (path.name.startswith("test-") or ".test." in name or ".spec." in name):
        return "node"
    return None


def make_plan(root: Path, names: list[str], explicit_tests: list[str]) -> dict:
    names = sorted({relative_path(root, name) for name in names})
    requested_tests = {relative_path(root, name) for name in explicit_tests}
    tests: set[str] = set()
    checks: list[Check] = []
    gaps: list[str] = []
    consumer_families: set[str] = set()
    for name in sorted(requested_tests):
        family = test_family(name)
        if not (root / name).is_file():
            gaps.append(f"Selected test does not exist: {name}")
        elif family is None:
            gaps.append(f"Unsupported selected test: {name}")
        else:
            tests.add(name)
            consumer_families.add(family)
    if not names:
        return {"source": "worktree", "files": [], "checks": [], "gaps": ["Empty selection; no checks executed."]}
    checks.append(Check("whitespace", ["git", "diff", "--check", "HEAD", "--", *names]))
    go_packages: set[str] = set()
    web_files: list[str] = []
    style_files: list[str] = []
    python_tests: set[str] = set()
    node_tests: set[str] = set()
    for name in names:
        path = root / name
        pure = PurePosixPath(name)
        suffix = path.suffix
        if name.startswith("server/") and suffix == ".go":
            if list(path.parent.glob("*.go")):
                go_packages.add("./" + pure.parent.relative_to("server").as_posix())
            if not list(path.parent.glob("*_test.go")) and "go" not in consumer_families:
                gaps.append(f"Select Go consumer tests for {name} with --test.")
        elif name.startswith("web/") and suffix in WEB_SOURCE:
            if path.is_file():
                web_files.append(pure.relative_to("web").as_posix())
            companions = [str(pure.with_suffix("")) + ending for ending in (".test.ts", ".spec.ts")]
            found = [item for item in companions if (root / item).is_file()]
            if test_family(name) == "web" and path.is_file():
                found.append(name)
            tests.update(found)
            if not found and "web" not in consumer_families:
                gaps.append(f"Select consumer tests for {name} with --test.")
        elif suffix == ".py":
            candidate = str(pure.with_name("test_" + pure.name))
            if pure.name.startswith("test_"):
                candidate = name
            if (root / candidate).is_file():
                python_tests.add(candidate)
            elif "python" not in consumer_families:
                gaps.append(f"Select Python consumer tests for {name} with --test.")
        elif suffix == ".mjs":
            candidate = str(pure.with_name("test-" + pure.name))
            if pure.name.startswith("test-"):
                candidate = name
            if (root / candidate).is_file():
                node_tests.add(candidate)
            elif "node" not in consumer_families:
                gaps.append(f"Select Node consumer tests for {name} with --test.")
        elif name in ("package.json", "bun.lock", "turbo.json", "server/package.json", "web/package.json"):
            node_tests.add("scripts/test-turbo.mjs")
            python_tests.add("scripts/test_local_validation.py")
        elif name.startswith("web/") and suffix == ".json":
            checks.append(Check("web-typecheck", ["bun", "run", "typecheck"], "web"))
        elif name.startswith("openapi/"):
            gaps.append("OpenAPI changes require just openapi-check and explicit consumer tests.")
        elif suffix not in (".md", ".yaml", ".yml", ".json", ".lock", ".toml", ".txt") and name not in ("Justfile", ".gitignore"):
            gaps.append(f"Select an owning diagnostic for {name}; automatic scope is unknown.")
        if name.startswith("web/") and suffix in STYLE_SOURCE and path.is_file():
            style_files.append(pure.relative_to("web").as_posix())

    for name in sorted(tests):
        if not (root / name).is_file():
            gaps.append(f"Selected test does not exist: {name}")
        elif name.startswith("server/") and name.endswith("_test.go"):
            go_packages.add("./" + PurePosixPath(name).parent.relative_to("server").as_posix())
        elif name.startswith("web/") and (".test." in name or ".spec." in name):
            continue
        elif test_family(name) == "python":
            python_tests.add(name)
        elif test_family(name) == "node":
            node_tests.add(name)
        else:
            gaps.append(f"Unsupported selected test: {name}")

    if go_packages:
        checks.append(Check("server-lint", ["go", "run", "./cmd/graft", "validate", "backend", "--stage", "lint"], "server"))
        checks.append(Check("server-tests", ["go", "test", *sorted(go_packages)], "server"))
    if web_files:
        checks.extend([
            Check("web-format", ["bun", "x", "--no-install", "prettier", "--check", *web_files], "web"),
            Check("web-lint", ["bun", "x", "--no-install", "eslint", "--max-warnings", "0", *web_files], "web"),
        ])
    if style_files:
        checks.append(Check("web-style", ["bun", "x", "--no-install", "stylelint", *style_files], "web"))
    web_tests = sorted(PurePosixPath(name).relative_to("web").as_posix()
                       for name in tests if name.startswith("web/") and (".test." in name or ".spec." in name)
                       and (root / name).is_file())
    if web_tests:
        checks.append(Check("web-tests", ["bun", "x", "--no-install", "vitest", "run", *web_tests], "web"))
    if python_tests:
        checks.append(Check("python-tests", [sys.executable, "-B", "-m", "unittest", *sorted(python_tests)]))
    if node_tests:
        checks.append(Check("node-tests", ["node", "--test", *sorted(node_tests)]))
    if any(name.startswith(("ai-plan/", ".agents/")) or name.endswith("AGENTS.md") for name in names):
        checks.append(Check("governance", [sys.executable, "-B", "scripts/validate_ai_governance.py"]))
        checks.append(Check("recovery-structure", [sys.executable, "-B", "scripts/validate_ai_plan_structure.py"]))
    if any(name.endswith((".yml", ".yaml")) and name.startswith(".github/") for name in names):
        checks.append(Check("workflow-yaml", [sys.executable, "-c",
            "import pathlib,yaml; [yaml.safe_load(pathlib.Path(p).read_text()) for p in __import__('sys').argv[1:]]",
            *[name for name in names if name.startswith(".github/") and name.endswith((".yml", ".yaml")) and (root / name).is_file()]]))
    unique_checks: list[Check] = []
    seen_commands: set[tuple[str, tuple[str, ...]]] = set()
    for check in checks:
        command = (check.cwd, tuple(check.argv))
        if command not in seen_commands:
            seen_commands.add(command)
            unique_checks.append(check)
    return {"source": "worktree", "files": names, "tests": sorted(tests | python_tests | node_tests),
            "checks": [asdict(check) for check in unique_checks], "gaps": gaps}


def execute(plan: dict, root: Path) -> int:
    failed = False
    for check in plan["checks"]:
        started = time.monotonic()
        print(f"RUN {check['name']} ({check['cwd']})", flush=True)
        try:
            result = subprocess.run(check["argv"], cwd=root / check["cwd"], check=False)
            code = result.returncode
        except OSError as error:
            print(f"Missing prerequisite for {check['name']}: {error}", file=sys.stderr)
            code = 127
        print(f"RESULT {check['name']} exit={code} seconds={time.monotonic() - started:.2f}", flush=True)
        failed = failed or code != 0
    if failed:
        return 1
    return 2 if plan["gaps"] else 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--plan", action="store_true")
    mode.add_argument("--execute", action="store_true")
    parser.add_argument("--file", action="append", default=[], help="Repository-relative changed file, repeatable.")
    parser.add_argument("--test", action="append", default=[], help="Explicit affected consumer test, repeatable.")
    args = parser.parse_args(argv)
    try:
        plan = make_plan(ROOT, args.file or changed_files(ROOT), args.test)
        divergence = staged_divergence(ROOT, plan["files"] + plan.get("tests", []))
        if divergence:
            plan["gaps"].append("Partially staged content differs from the worktree: " + ", ".join(divergence))
        print(json.dumps(plan, indent=2))
        if args.plan:
            return 0
        if divergence:
            return 2
        return execute(plan, ROOT)
    except (ValueError, subprocess.CalledProcessError) as error:
        print(error, file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
