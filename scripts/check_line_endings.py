#!/usr/bin/env python3
"""Reject carriage returns in text blobs from the Git index."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path


def repo_root() -> Path:
    return Path(
        subprocess.check_output(
            ["git", "rev-parse", "--show-toplevel"],
            text=True,
        ).strip()
    )


def paths_with_carriage_returns(root: Path) -> list[str]:
    result = subprocess.run(
        ["git", "grep", "--cached", "-Il", "-e", "\r", "--"],
        cwd=root,
        check=False,
        capture_output=True,
        text=True,
    )
    if result.returncode == 1:
        return []
    if result.returncode != 0:
        detail = result.stderr.strip() or "git grep failed without an error message"
        raise RuntimeError(detail)
    return sorted(path for path in result.stdout.splitlines() if path)


def main() -> int:
    try:
        violations = paths_with_carriage_returns(repo_root())
    except (OSError, subprocess.CalledProcessError, RuntimeError) as error:
        print(f"line ending gate: failed to inspect the Git index: {error}", file=sys.stderr)
        return 1

    if not violations:
        print("line ending gate: ok (tracked text uses LF)")
        return 0

    print("line ending gate: tracked text files must use LF; carriage returns found:", file=sys.stderr)
    for path in violations:
        print(f"  - {path}", file=sys.stderr)
    print(
        "normalize the listed files to LF, stage them again, and rerun this check.",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
