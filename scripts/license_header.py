#!/usr/bin/env python3
# Copyright (c) 2025-2026 GeWuYou
# SPDX-License-Identifier: Apache-2.0

"""Check and apply repository Apache-2.0 license headers."""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable


COPYRIGHT_LINE = "Copyright (c) 2025-2026 GeWuYou"
SPDX_LINE = "SPDX-License-Identifier: Apache-2.0"

LINE_COMMENT_EXTENSIONS = {
    ".cjs": "//",
    ".go": "//",
    ".js": "//",
    ".mjs": "//",
    ".ps1": "#",
    ".sh": "#",
    ".sql": "--",
    ".ts": "//",
    ".tsx": "//",
}

BLOCK_COMMENT_EXTENSIONS = {
    ".css": ("/*", " *", " */"),
    ".scss": ("/*", " *", " */"),
}

HTML_COMMENT_EXTENSIONS = {
    ".vue",
}

SUPPORTED_EXTENSIONS = (
    set(LINE_COMMENT_EXTENSIONS)
    | set(BLOCK_COMMENT_EXTENSIONS)
    | HTML_COMMENT_EXTENSIONS
)

EXCLUDED_PREFIXES = (
    ".ai/",
    ".agents/",
    ".git/",
    ".output/",
    ".tmp/",
    "ai-plan/",
    "build/",
    "coverage/",
    "dist/",
    "docs/",
    "node_modules/",
    "openapi/generated/",
    "server/tmp/",
    "temp/",
    "vendor/",
    "web/coverage/",
    "web/build/",
    "web/dist/",
    "web/node_modules/",
    "web/src/api/generated/",
    "server/internal/contract/openapi/",
)

EXCLUDED_PATH_PARTS = {
    "__pycache__",
    "generated",
    "zz_generated",
}

EXCLUDED_EXACT_PATHS = {
    "bun.lock",
    "go.mod",
    "go.sum",
    "LICENSE",
    "LICENSE.md",
    "package-lock.json",
    "pnpm-lock.yaml",
    "web/bun.lock",
    "web/package-lock.json",
    "web/pnpm-lock.yaml",
}

EXCLUDED_SUFFIXES = (
    ".bundle.json",
    ".gen.go",
    ".json",
    ".lock",
    ".lockb",
    ".md",
    ".min.css",
    ".min.js",
    ".pyc",
    ".yaml",
    ".yml",
)

HEADER_MARKERS = (SPDX_LINE,)


@dataclass(frozen=True)
class HeaderResult:
    """Describes the license header status for a repository file."""

    path: Path
    relative_path: str
    has_header: bool


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Check or apply Apache-2.0 source file headers.")
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true", help="Fail when supported files are missing a license header.")
    mode.add_argument("--fix", action="store_true", help="Write missing license headers to supported files.")
    parser.add_argument("--dry-run", action="store_true", help="List files that would be changed without writing them.")
    parser.add_argument(
        "--scope",
        choices=("all", "added"),
        default="all",
        help="Check all tracked files or only files newly added relative to --base-ref.",
    )
    parser.add_argument("--base-ref", help="Base git ref used when --scope added is selected.")
    parser.add_argument("--head-ref", default="HEAD", help="Head git ref used when --scope added is selected.")
    parser.add_argument(
        "--root",
        type=Path,
        default=Path.cwd(),
        help="Repository root. Defaults to the current working directory.",
    )
    parser.add_argument(
        "--paths",
        nargs="*",
        help="Optional repository-relative file list. Intended for tests and targeted local checks.",
    )

    args = parser.parse_args(argv)
    root = args.root.resolve()
    relative_paths = resolve_input_paths(args, root)
    results = list(scan_files(root, relative_paths))
    missing = [result for result in results if not result.has_header]

    if not missing:
        print("All supported files include an Apache-2.0 license header.")
        return 0

    if args.check or args.dry_run:
        print_results(missing, "Missing Apache-2.0 license header:")

    if args.check:
        return 1

    if args.dry_run:
        return 0

    for result in missing:
        add_license_header(result.path)

    print_results(missing, "Added Apache-2.0 license header:")
    return 0


def resolve_input_paths(args: argparse.Namespace, root: Path) -> list[str]:
    if args.paths is not None:
        return list(args.paths)

    if args.scope == "added":
        if not args.base_ref:
            raise SystemExit("--base-ref is required when --scope added is selected.")
        return list_added_files(root, args.base_ref, args.head_ref)

    return list_tracked_files(root)


def list_tracked_files(root: Path) -> list[str]:
    """Return repository-tracked paths so generated build output is not scanned."""

    return run_git_output(root, ["ls-files"])


def list_added_files(root: Path, base_ref: str, head_ref: str) -> list[str]:
    """Return files added between base_ref and head_ref."""

    return run_git_output(root, ["diff", "--name-only", "--diff-filter=A", f"{base_ref}...{head_ref}"])


def run_git_output(root: Path, args: list[str]) -> list[str]:
    completed = subprocess.run(["git", *args], cwd=root, check=True, capture_output=True, text=True)
    return [line for line in completed.stdout.splitlines() if line.strip()]


def scan_files(root: Path, relative_paths: Iterable[str]) -> Iterable[HeaderResult]:
    for relative_path in relative_paths:
        normalized = relative_path.replace(os.sep, "/")
        if not is_supported_path(normalized):
            continue

        path = root / normalized
        if not path.is_file():
            continue

        text, _ = read_text_preserving_bom(path)
        yield HeaderResult(
            path=path,
            relative_path=normalized,
            has_header=has_license_header(text),
        )


def is_supported_path(relative_path: str) -> bool:
    if relative_path in EXCLUDED_EXACT_PATHS:
        return False

    path = Path(relative_path)
    if path.name.startswith(".env"):
        return False

    if relative_path.endswith(EXCLUDED_SUFFIXES):
        return False

    if any(relative_path.startswith(prefix) for prefix in EXCLUDED_PREFIXES):
        return False

    if any(
        part in EXCLUDED_PATH_PARTS or part.startswith("zz_generated") or part.endswith(".generated")
        for part in path.parts
    ):
        return False

    if is_ent_generated_path(path):
        return False

    return path.suffix in SUPPORTED_EXTENSIONS


def is_ent_generated_path(path: Path) -> bool:
    parts = path.parts
    if "ent" not in parts:
        return False

    ent_index = parts.index("ent")
    if ent_index + 1 >= len(parts):
        return False

    return parts[ent_index + 1] != "schema"


def has_license_header(text: str) -> bool:
    search_window = text[:4096]
    return any(marker in search_window for marker in HEADER_MARKERS)


def add_license_header(path: Path) -> None:
    text, had_bom = read_text_preserving_bom(path)
    updated = insert_header(path, text)
    write_text_preserving_bom(path, updated, had_bom)


def insert_header(path: Path, text: str) -> str:
    newline = detect_newline(text)
    header = build_header(path.suffix, newline)

    if text.startswith("#!"):
        line_end = find_first_line_end(text)
        if line_end == len(text):
            return f"{text}{newline}{header}"
        return f"{text[:line_end]}{header}{text[line_end:]}"

    return f"{header}{text}"


def find_first_line_end(text: str) -> int:
    first_newline = text.find("\n")
    if first_newline == -1:
        return len(text)
    return first_newline + len("\n")


def build_header(suffix: str, newline: str) -> str:
    if suffix in HTML_COMMENT_EXTENSIONS:
        return (
            f"<!--{newline}"
            f"  {COPYRIGHT_LINE}{newline}"
            f"  {SPDX_LINE}{newline}"
            f"-->{newline}"
            f"{newline}"
        )

    if suffix in BLOCK_COMMENT_EXTENSIONS:
        start, middle, end = BLOCK_COMMENT_EXTENSIONS[suffix]
        return (
            f"{start}{newline}"
            f"{middle} {COPYRIGHT_LINE}{newline}"
            f"{middle} {SPDX_LINE}{newline}"
            f"{end}{newline}"
            f"{newline}"
        )

    comment = LINE_COMMENT_EXTENSIONS[suffix]
    return (
        f"{comment} {COPYRIGHT_LINE}{newline}"
        f"{comment} {SPDX_LINE}{newline}"
        f"{newline}"
    )


def detect_newline(text: str) -> str:
    return "\r\n" if "\r\n" in text[:4096] else "\n"


def read_text_preserving_bom(path: Path) -> tuple[str, bool]:
    content = path.read_bytes()
    had_bom = content.startswith(b"\xef\xbb\xbf")
    if had_bom:
        content = content[3:]
    return content.decode("utf-8"), had_bom


def write_text_preserving_bom(path: Path, text: str, had_bom: bool) -> None:
    content = text.encode("utf-8")
    if had_bom:
        content = b"\xef\xbb\xbf" + content
    path.write_bytes(content)


def print_results(results: Iterable[HeaderResult], heading: str) -> None:
    print(heading)
    for result in results:
        print(f"  {result.relative_path}")


if __name__ == "__main__":
    sys.exit(main())
