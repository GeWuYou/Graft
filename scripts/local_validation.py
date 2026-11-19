#!/usr/bin/env python3
"""Plan and run worktree-scoped commit preparation checks without installing tools."""
from __future__ import annotations

import argparse
from dataclasses import asdict, dataclass
import hashlib
import json
from pathlib import Path, PurePosixPath
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[1]
WEB_SOURCE = (".ts", ".tsx", ".js", ".jsx", ".mjs", ".vue")
STYLE_SOURCE = (".vue", ".css", ".less", ".sass", ".scss", ".html")
DELETED_DIGEST = "deleted"
WORKTREE_DIFF_REASON = "worktree diff vs HEAD (changed or untracked)"
EXPLICIT_FILE_REASON = "explicit --file selection"
EXPLICIT_TEST_REASON = "explicit --test consumer"


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


def head_commit(root: Path) -> str | None:
    try:
        result = subprocess.run(
            ["git", "rev-parse", "HEAD"], cwd=root, check=False, capture_output=True,
        )
    except OSError:
        return None
    if result.returncode != 0:
        return None
    return result.stdout.decode().strip() or None


def file_digest(root: Path, name: str) -> str:
    path = root / name
    if not path.is_file():
        return DELETED_DIGEST
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(65536), b""):
            digest.update(block)
    return digest.hexdigest()


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


def make_plan(
    root: Path,
    names: list[str],
    explicit_tests: list[str],
    selection_reason: str = WORKTREE_DIFF_REASON,
) -> dict:
    names = sorted({relative_path(root, name) for name in names})
    requested_tests = {relative_path(root, name) for name in explicit_tests}
    tests: set[str] = set()
    checks: list[Check] = []
    gaps: list[str] = []
    consumer_families: set[str] = set()
    reasons: dict[str, str] = {}
    coverage: dict[str, list[str]] = {}
    for name in names:
        reasons.setdefault(name, selection_reason)
        coverage.setdefault(name, [])
    for name in sorted(requested_tests):
        family = test_family(name)
        if not (root / name).is_file():
            gaps.append(f"Selected test does not exist: {name}")
        elif family is None:
            gaps.append(f"Unsupported selected test: {name}")
        else:
            tests.add(name)
            reasons.setdefault(name, EXPLICIT_TEST_REASON)
            coverage.setdefault(name, [])
            consumer_families.add(family)
    if not names:
        return {
            "source": "worktree", "head": head_commit(root), "files": [], "tests": [],
            "selection": [], "digests": {}, "checks": [],
            "gaps": ["Empty selection; no checks executed."],
        }
    checks.append(Check("whitespace", ["git", "diff", "--check", "HEAD", "--", *names]))
    for name in names:
        coverage[name].append("whitespace")
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
                coverage[name].extend(["server-lint", "server-tests"])
            if not list(path.parent.glob("*_test.go")) and "go" not in consumer_families:
                gaps.append(f"Select Go consumer tests for {name} with --test.")
        elif name.startswith("web/") and suffix in WEB_SOURCE:
            if path.is_file():
                web_files.append(pure.relative_to("web").as_posix())
                coverage[name].extend(["web-format", "web-lint"])
            companions = [str(pure.with_suffix("")) + ending for ending in (".test.ts", ".spec.ts")]
            found = [item for item in companions if (root / item).is_file()]
            if test_family(name) == "web" and path.is_file():
                found.append(name)
            for item in found:
                reasons.setdefault(item, f"derived consumer test for {name}")
                coverage.setdefault(item, []).append("web-tests")
            tests.update(found)
            if found:
                coverage[name].append("web-tests")
            if not found and "web" not in consumer_families:
                gaps.append(f"Select consumer tests for {name} with --test.")
        elif suffix == ".py":
            candidate = str(pure.with_name("test_" + pure.name))
            if pure.name.startswith("test_"):
                candidate = name
            if (root / candidate).is_file():
                python_tests.add(candidate)
                reasons.setdefault(candidate, f"derived consumer test for {name}")
                coverage.setdefault(candidate, []).append("python-tests")
                coverage[name].append("python-tests")
            elif "python" not in consumer_families:
                gaps.append(f"Select Python consumer tests for {name} with --test.")
        elif suffix == ".mjs":
            candidate = str(pure.with_name("test-" + pure.name))
            if pure.name.startswith("test-"):
                candidate = name
            if (root / candidate).is_file():
                node_tests.add(candidate)
                reasons.setdefault(candidate, f"derived consumer test for {name}")
                coverage.setdefault(candidate, []).append("node-tests")
                coverage[name].append("node-tests")
            elif "node" not in consumer_families:
                gaps.append(f"Select Node consumer tests for {name} with --test.")
        elif name in ("package.json", "bun.lock", "turbo.json", "server/package.json", "web/package.json"):
            node_tests.add("scripts/test-turbo.mjs")
            python_tests.add("scripts/test_local_validation.py")
            for item in ("scripts/test-turbo.mjs", "scripts/test_local_validation.py"):
                reasons.setdefault(item, f"declared consumer for toolchain file {name}")
                coverage.setdefault(item, []).append("node-tests" if item.endswith(".mjs") else "python-tests")
            coverage[name].extend(["python-tests", "node-tests"])
        elif name.startswith("web/") and suffix == ".json":
            checks.append(Check("web-typecheck", ["bun", "run", "typecheck"], "web"))
            coverage[name].append("web-typecheck")
        elif name.startswith("openapi/"):
            gaps.append("OpenAPI changes require just openapi-check and explicit consumer tests.")
        elif suffix not in (".md", ".yaml", ".yml", ".json", ".lock", ".toml", ".txt") and name not in ("Justfile", ".gitignore"):
            gaps.append(f"Select an owning diagnostic for {name}; automatic scope is unknown.")
        if name.startswith("web/") and suffix in STYLE_SOURCE and path.is_file():
            style_files.append(pure.relative_to("web").as_posix())
            coverage[name].append("web-style")

    for name in sorted(tests):
        if not (root / name).is_file():
            gaps.append(f"Selected test does not exist: {name}")
        elif name.startswith("server/") and name.endswith("_test.go"):
            go_packages.add("./" + PurePosixPath(name).parent.relative_to("server").as_posix())
            coverage[name].append("server-tests")
        elif name.startswith("web/") and (".test." in name or ".spec." in name):
            coverage[name].append("web-tests")
        elif test_family(name) == "python":
            python_tests.add(name)
            coverage[name].append("python-tests")
        elif test_family(name) == "node":
            node_tests.add(name)
            coverage[name].append("node-tests")
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
        for name in names:
            if name.startswith(("ai-plan/", ".agents/")) or name.endswith("AGENTS.md"):
                coverage[name].extend(["governance", "recovery-structure"])
    if any(name.endswith((".yml", ".yaml")) and name.startswith(".github/") for name in names):
        github_yamls = [name for name in names if name.startswith(".github/")
                        and name.endswith((".yml", ".yaml")) and (root / name).is_file()]
        checks.append(Check("workflow-yaml", [sys.executable, "-c",
            "import pathlib,yaml; [yaml.safe_load(pathlib.Path(p).read_text()) for p in __import__('sys').argv[1:]]",
            *github_yamls]))
        for name in github_yamls:
            coverage[name].append("workflow-yaml")
    unique_checks: list[Check] = []
    seen_commands: set[tuple[str, tuple[str, ...]]] = set()
    for check in checks:
        command = (check.cwd, tuple(check.argv))
        if command not in seen_commands:
            seen_commands.add(command)
            unique_checks.append(check)
    available = {check.name for check in unique_checks}
    for name in list(coverage):
        seen: set[str] = set()
        ordered: list[str] = []
        for item in coverage[name]:
            if item in available and item not in seen:
                seen.add(item)
                ordered.append(item)
        coverage[name] = ordered
    all_selected = sorted(set(names) | tests | python_tests | node_tests)
    digests = {name: file_digest(root, name) for name in all_selected}
    selection = [{"file": name, "reason": reasons.get(name, selection_reason),
                  "covered_by": coverage.get(name, [])} for name in all_selected]
    return {"source": "worktree", "head": head_commit(root), "files": names,
            "tests": sorted(tests | python_tests | node_tests),
            "selection": selection, "digests": digests,
            "checks": [asdict(check) for check in unique_checks], "gaps": gaps}


def load_report(location: str) -> dict:
    if location == "-":
        raw = sys.stdin.read()
    else:
        raw = Path(location).read_text()
    report = json.loads(raw)
    if not isinstance(report, dict):
        raise ValueError(f"Expected a JSON object report: {location}")
    return report


def compare_reports(previous: dict, current_plan: dict, root: Path) -> dict:
    """Verify recorded evidence still applies to the current worktree state."""
    reasons: list[str] = []
    recorded_head = previous.get("head")
    current_head = current_plan.get("head")
    if not recorded_head:
        reasons.append("Report records no head SHA binding; regenerate it with the current tooling.")
    elif recorded_head != current_head:
        reasons.append(f"Head moved: recorded {recorded_head}, current {current_head or 'unknown'}.")
    recorded_digests = previous.get("digests")
    if not isinstance(recorded_digests, dict) or not recorded_digests:
        reasons.append("Report records no selected file digests; regenerate it with the current tooling.")
        recorded_digests = {}
    statuses: dict[str, str] = {}
    for name, recorded in sorted(recorded_digests.items()):
        current = file_digest(root, name)
        if current == recorded:
            statuses[name] = "match"
        elif current == DELETED_DIGEST:
            statuses[name] = "deleted"
            reasons.append(f"Selected file deleted since report: {name}")
        else:
            statuses[name] = "modified"
            reasons.append(f"Selected file modified since report: {name}")
    current_paths = set(current_plan["files"]) | set(current_plan.get("tests", []))
    for name in sorted(current_paths - set(recorded_digests)):
        statuses[name] = "uncovered"
        reasons.append(f"Current selection is not covered by the report: {name}")
    return {"verdict": "applies" if not reasons else "stale",
            "reasons": reasons, "files": statuses,
            "head": {"recorded": recorded_head, "current": current_head}}


def execute_with_results(plan: dict, root: Path) -> tuple[int, list[dict]]:
    results: list[dict] = []
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
        seconds = time.monotonic() - started
        print(f"RESULT {check['name']} exit={code} seconds={seconds:.2f}", flush=True)
        results.append({"name": check["name"], "argv": check["argv"], "cwd": check["cwd"],
                        "exit": code, "seconds": round(seconds, 2)})
        failed = failed or code != 0
    if failed:
        return 1, results
    return (2 if plan["gaps"] else 0), results


def execute(plan: dict, root: Path) -> int:
    code, _ = execute_with_results(plan, root)
    return code


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--plan", action="store_true")
    mode.add_argument("--execute", action="store_true")
    parser.add_argument("--file", action="append", default=[], help="Repository-relative changed file, repeatable.")
    parser.add_argument("--test", action="append", default=[], help="Explicit affected consumer test, repeatable.")
    parser.add_argument("--compare-report", metavar="PATH", default=None,
                        help="Previous plan/check report JSON ('-' for stdin); fail when its evidence no longer applies.")
    parser.add_argument("--report-out", metavar="PATH", default=None,
                        help="With --execute: write the executed check report JSON to PATH.")
    args = parser.parse_args(argv)
    try:
        selection_reason = EXPLICIT_FILE_REASON if args.file else WORKTREE_DIFF_REASON
        plan = make_plan(ROOT, args.file or changed_files(ROOT), args.test, selection_reason)
        divergence = staged_divergence(ROOT, plan["files"] + plan.get("tests", []))
        if divergence:
            plan["gaps"].append("Partially staged content differs from the worktree: " + ", ".join(divergence))
        comparison = None
        if args.compare_report:
            comparison = compare_reports(load_report(args.compare_report), plan, ROOT)
            plan["comparison"] = comparison
        print(json.dumps(plan, indent=2))
        if args.plan:
            if comparison and comparison["verdict"] != "applies":
                return 2
            return 2 if plan["gaps"] else 0
        if divergence:
            return 2
        if comparison and comparison["verdict"] != "applies":
            return 2
        code, results = execute_with_results(plan, ROOT)
        if args.report_out:
            report = dict(plan)
            report["results"] = results
            Path(args.report_out).write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
            print(f"REPORT {args.report_out}", flush=True)
        return code
    except (ValueError, subprocess.CalledProcessError, OSError, json.JSONDecodeError) as error:
        print(error, file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
