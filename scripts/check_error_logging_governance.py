#!/usr/bin/env python3
"""Reject narrow, mechanically detectable backend error logging regressions."""

from __future__ import annotations

import argparse
from dataclasses import dataclass
from pathlib import Path
import re
import sys


@dataclass(frozen=True)
class Finding:
    path: Path
    line: int
    rule: str


def is_production_go_path(path: Path) -> bool:
    parts = path.parts
    return (
        len(parts) > 1
        and parts[0] == "server"
        and path.suffix == ".go"
        and not path.name.endswith("_test.go")
        and "vendor" not in parts
        and "generated" not in parts
        and path.name != "generated.go"
    )


def findings_for_pattern(path: Path, source: str, pattern: re.Pattern[str], rule: str) -> list[Finding]:
    return [Finding(path, source.count("\n", 0, match.start()) + 1, rule) for match in pattern.finditer(source)]


def scan_source(path: Path, source: str) -> list[Finding]:
    findings: list[Finding] = []
    findings.extend(findings_for_pattern(path, source, re.compile(r"\bgin\.Recovery\s*\("), "generic Gin recovery bypasses correlated panic logging"))
    findings.extend(findings_for_pattern(path, source, re.compile(r"\bfmt\.Errorf\s*\(\s*[^\n]*%v[^\n]*\berr\b"), "fmt.Errorf must wrap propagated errors with %w"))
    findings.extend(findings_for_pattern(path, source, re.compile(r"\berrors\.New\s*\(\s*\berr\.Error\s*\(\s*\)\s*\)"), "errors.New(err.Error()) drops the cause chain"))
    findings.extend(findings_for_pattern(path, source, re.compile(r"\baccessLogger\.(?:Error|Warn)\s*\("), "Access Log must remain INFO request facts"))
    findings.extend(findings_for_pattern(path, source, re.compile(r"(?:\"data\"|\bData)\s*:\s*\berr\.Error\s*\(\s*\)"), "public error data must not expose err.Error()"))
    return findings


def scan_repository(root: Path) -> list[Finding]:
    findings: list[Finding] = []
    for path in sorted(root.joinpath("server").rglob("*.go")):
        relative = path.relative_to(root)
        if is_production_go_path(relative):
            findings.extend(scan_source(relative, path.read_text(encoding="utf-8")))
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description="Reject bounded backend error logging regressions.")
    parser.add_argument("--root", type=Path, default=Path.cwd(), help="repository root")
    args = parser.parse_args()
    findings = scan_repository(args.root.resolve())
    if not findings:
        print("error logging governance: ok")
        return 0
    for finding in findings:
        print(f"{finding.path}:{finding.line}: {finding.rule}", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
