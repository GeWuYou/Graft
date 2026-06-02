#!/usr/bin/env python3
"""Check non-archive plugin residuals against an explicit allowlist."""

from __future__ import annotations

import json
import pathlib
import re
import subprocess
import sys
from dataclasses import dataclass


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
ALLOWLIST_PATH = pathlib.Path(__file__).resolve().with_name("allowlist.json")
PATTERN = re.compile(r"\bplugin\b|\bplugins\b|\bPlugin\b|server/plugins")
SKIP_PATH_PREFIXES = (
    ".git/",
    "ai-plan/public/archive/",
    "web/bun.lock",
    "web/package.json",
    "web/env.d.ts",
    "web/eslint.config.js",
    "web/stylelint.config.js",
    "web/vite.config.ts",
    "web/mock/",
)


@dataclass(frozen=True)
class AllowRule:
    path: str | None
    path_prefix: str | None
    regex: re.Pattern[str]
    category: str
    reason: str


@dataclass(frozen=True)
class Match:
    path: str
    line_no: int
    line: str


def tracked_files() -> list[str]:
    completed = subprocess.run(
        ["git", "-c", "core.quotePath=false", "ls-files"],
        cwd=REPO_ROOT,
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    return [line for line in completed.stdout.splitlines() if line]


def should_skip(path: str) -> bool:
    return any(path.startswith(prefix) for prefix in SKIP_PATH_PREFIXES)


def load_allowlist() -> list[AllowRule]:
    payload = json.loads(ALLOWLIST_PATH.read_text(encoding="utf-8"))
    return [
        AllowRule(
            path=item.get("path"),
            path_prefix=item.get("path_prefix"),
            regex=re.compile(item["regex"]),
            category=item["category"],
            reason=item["reason"],
        )
        for item in payload["patterns"]
    ]


def find_matches(path: str) -> list[Match]:
    full_path = REPO_ROOT / path
    try:
        text = full_path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        text = full_path.read_text(encoding="utf-8", errors="ignore")

    matches: list[Match] = []
    for line_no, line in enumerate(text.splitlines(), start=1):
        if PATTERN.search(line):
            matches.append(Match(path=path, line_no=line_no, line=line))
    return matches


def classify(match: Match, rules: list[AllowRule]) -> AllowRule | None:
    for rule in rules:
        if rule.path is not None and rule.path != match.path:
            continue
        if rule.path_prefix is not None and not match.path.startswith(rule.path_prefix):
            continue
        if rule.regex.search(match.line):
            return rule
    return None


def main() -> int:
    rules = load_allowlist()
    uncategorized: list[Match] = []
    categorized = 0

    for path in tracked_files():
        if should_skip(path):
            continue
        for match in find_matches(path):
            rule = classify(match, rules)
            if rule is None:
                uncategorized.append(match)
                continue
            categorized += 1

    if uncategorized:
        print("Uncategorized plugin residuals found:", file=sys.stderr)
        for match in uncategorized:
            print(f"{match.path}:{match.line_no}: {match.line}", file=sys.stderr)
        return 1

    print(f"plugin residual scan passed: {categorized} categorized matches")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
