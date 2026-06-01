#!/usr/bin/env python3
"""Validate module-owned Atlas migration filenames for the default backend chain."""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from collections import defaultdict
from pathlib import Path


SQL_NAME_RE = re.compile(r"^(?P<version>\d+)_(?P<name>.+)\.sql$")
MODULE_MIGRATION_PARTS = ("server", "modules")


def repo_root() -> Path:
    return Path(
        subprocess.check_output(
            ["git", "rev-parse", "--show-toplevel"],
            text=True,
        ).strip()
    )


def staged_paths(root: Path) -> list[Path]:
    output = subprocess.check_output(
        ["git", "diff", "--cached", "--name-only", "--diff-filter=ACMR"],
        cwd=root,
        text=True,
    )
    return [root / line for line in output.splitlines() if line]


def candidate_dirs(root: Path, mode: str) -> list[Path]:
    base = root / "server" / "modules"
    all_dirs = {path for path in base.glob("*/migrations") if path.is_dir()}
    if mode == "all":
        return sorted(all_dirs)

    staged = staged_paths(root)
    dirs: set[Path] = set(all_dirs)
    for path in staged:
        try:
            relative = path.relative_to(root)
        except ValueError:
            continue
        parts = relative.parts
        if len(parts) >= 4 and parts[:2] == MODULE_MIGRATION_PARTS and parts[3] == "migrations":
            dirs.add(root / "server" / "modules" / parts[2] / "migrations")

    return sorted(path for path in dirs if path.is_dir())


def iter_sql_files(migration_dirs: list[Path]) -> list[tuple[str, Path]]:
    files: list[tuple[str, Path]] = []
    for migration_dir in migration_dirs:
        for path in sorted(migration_dir.iterdir()):
            if not path.is_file() or path.suffix != ".sql":
                continue
            match = SQL_NAME_RE.match(path.name)
            if not match:
                continue
            files.append((match.group("version"), path))
    return files


def validate(migration_dirs: list[Path], root: Path) -> list[str]:
    duplicates: dict[str, list[str]] = defaultdict(list)
    for version, path in iter_sql_files(migration_dirs):
        duplicates[version].append(str(path.relative_to(root)))

    errors = []
    for version in sorted(duplicates):
        paths = duplicates[version]
        if len(paths) <= 1:
            continue
        errors.append(
            "default migration chain version conflict: "
            f"{version} appears in {', '.join(paths)}"
        )
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Validate module-owned migration versions remain globally unique in the default chain."
    )
    parser.add_argument(
        "--mode",
        choices=("changed", "all"),
        default="changed",
        help="check only staged module migration directories or all module migration directories",
    )
    args = parser.parse_args()

    root = repo_root()
    dirs = candidate_dirs(root, args.mode)
    if not dirs:
        print("migration version gate: skip (no matching module migration directories)")
        return 0

    errors = validate(dirs, root)
    if not errors:
        print("migration version gate: ok")
        return 0

    for error in errors:
        print(error, file=sys.stderr)
    print(
            "migration version gate: Atlas default chain aggregates module migrations into one directory; "
            "use globally unique numeric versions across modules.",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
