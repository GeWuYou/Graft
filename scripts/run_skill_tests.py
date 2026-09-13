#!/usr/bin/env python3
"""Discover and run every repository skill-local Python test file."""

from __future__ import annotations

from collections.abc import Callable, Sequence
from pathlib import Path
import importlib.util
import subprocess
import sys


REPO_ROOT = Path(__file__).resolve().parents[1]
SKILLS_DIR = REPO_ROOT / ".agents" / "skills"


def discover_skill_tests(skills_dir: Path | None = None) -> list[Path]:
    """Use the governance discovery owner for repository and declared plugin skills."""
    if skills_dir is not None:
        roots = [skills_dir]
    else:
        spec = importlib.util.spec_from_file_location("skill_test_governance", Path(__file__).with_name("validate_ai_governance.py"))
        if spec is None or spec.loader is None:
            raise RuntimeError("cannot load repository skill discovery owner")
        governance = importlib.util.module_from_spec(spec)
        sys.modules[spec.name] = governance
        spec.loader.exec_module(governance)
        roots, findings = governance.discover_skill_roots(REPO_ROOT)
        if findings:
            raise RuntimeError("; ".join(item.format() for item in findings))
    discovered = []
    for root in roots:
        for path in root.rglob("test_*.py"):
            if not path.resolve().is_relative_to(root.resolve()):
                raise RuntimeError(f"skill test escapes declared skill root: {path}")
            if path.is_file():
                discovered.append(path)
    return sorted(set(discovered))


def run_skill_tests(
    test_paths: Sequence[Path],
    repo_root: Path = REPO_ROOT,
    runner: Callable[..., subprocess.CompletedProcess[str]] = subprocess.run,
) -> int:
    """Run every test file independently and return a combined exit status."""
    failures = 0
    for test_path in test_paths:
        try:
            display_path = test_path.relative_to(repo_root)
        except ValueError:
            display_path = test_path
        print(f"[skill-test] {display_path}", flush=True)
        completed = runner([sys.executable, str(test_path)], cwd=repo_root, check=False)
        if completed.returncode != 0:
            failures += 1

    print(f"skill-local test files: {len(test_paths)}; failed files: {failures}")
    return 1 if failures else 0


def main() -> int:
    try:
        test_paths = discover_skill_tests()
    except RuntimeError as exc:
        print(f"Skill discovery failed: {exc}", file=sys.stderr)
        return 1
    if not test_paths:
        print("No skill-local tests were discovered.", file=sys.stderr)
        return 1
    return run_skill_tests(test_paths)


if __name__ == "__main__":
    raise SystemExit(main())
