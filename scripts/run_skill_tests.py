#!/usr/bin/env python3
"""Discover and run every repository skill-local Python test file."""

from __future__ import annotations

from collections.abc import Callable, Sequence
from pathlib import Path
import subprocess
import sys


REPO_ROOT = Path(__file__).resolve().parents[1]
SKILLS_DIR = REPO_ROOT / ".agents" / "skills"


def discover_skill_tests(skills_dir: Path = SKILLS_DIR) -> list[Path]:
    """Return skill-local unittest files in stable repository-relative order."""
    return sorted(path for path in skills_dir.rglob("test_*.py") if path.is_file())


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
    test_paths = discover_skill_tests()
    if not test_paths:
        print("No skill-local tests were discovered.", file=sys.stderr)
        return 1
    return run_skill_tests(test_paths)


if __name__ == "__main__":
    raise SystemExit(main())
