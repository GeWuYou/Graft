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


def mask_non_code(source: str) -> str:
    """Replace comments and literals while preserving positions for code-only checks."""
    masked = list(source)
    index = 0
    while index < len(source):
        if source.startswith("//", index):
            end = source.find("\n", index)
            end = len(source) if end == -1 else end
            for position in range(index, end):
                masked[position] = " "
            index = end
            continue
        if source.startswith("/*", index):
            end = source.find("*/", index + 2)
            end = len(source) if end == -1 else end + 2
            for position in range(index, end):
                if source[position] != "\n":
                    masked[position] = " "
            index = end
            continue
        if source[index] in {'"', "'", chr(96)}:
            quote = source[index]
            index += 1
            while index < len(source):
                escaped = source[index] == "\\" and quote != chr(96)
                if source[index] == quote and not escaped:
                    index += 1
                    break
                if source[index] != "\n":
                    masked[index] = " "
                if escaped:
                    index += 1
                    if index < len(source) and source[index] != "\n":
                        masked[index] = " "
                index += 1
            continue
        index += 1
    return "".join(masked)


def fmt_errorf_findings(path: Path, source: str) -> list[Finding]:
    masked = mask_non_code(source)
    findings: list[Finding] = []
    for match in re.finditer(r"\bfmt\.Errorf\s*\(", masked):
        opening = match.end() - 1
        depth = 0
        closing = None
        in_string = False
        escaped = False
        for position in range(opening, len(source)):
            char = source[position]
            if in_string:
                if escaped:
                    escaped = False
                elif char == "\\":
                    escaped = True
                elif char == '"':
                    in_string = False
                continue
            if char == '"':
                in_string = True
            elif char == '(':
                depth += 1
            elif char == ')':
                depth -= 1
                if depth == 0:
                    closing = position
                    break
        if closing is None:
            continue
        call = source[opening + 1:closing]
        format_match = re.match(r'\s*"((?:\\.|[^"\\])*)"\s*,(.*)', call, re.DOTALL)
        if format_match is None:
            continue
        format_text, arguments = format_match.groups()
        has_unwrapped_value = re.search(r"%(?!%)\S*v", format_text)
        has_wrapped_error = re.search(r"%(?!%)\S*w", format_text)
        if has_unwrapped_value and not has_wrapped_error and re.search(r"\berr\b", mask_non_code(arguments)):
            findings.append(Finding(path, source.count("\n", 0, match.start()) + 1, "fmt.Errorf must wrap propagated errors with %w"))
    return findings


def scan_source(path: Path, source: str) -> list[Finding]:
    findings: list[Finding] = []
    code = mask_non_code(source)
    code_with_data = list(code)
    for match in re.finditer(r'"data"\s*:', source):
        if code[match.end() - 1] == ":":
            code_with_data[match.start():match.end()] = source[match.start():match.end()]
    code_with_data = "".join(code_with_data)
    findings.extend(findings_for_pattern(path, code, re.compile(r"\bgin\.Recovery\s*\("), "generic Gin recovery bypasses correlated panic logging"))
    findings.extend(fmt_errorf_findings(path, source))
    findings.extend(findings_for_pattern(path, code, re.compile(r"\berrors\.New\s*\(\s*\berr\.Error\s*\(\s*\)\s*\)"), "errors.New(err.Error()) drops the cause chain"))
    findings.extend(findings_for_pattern(path, code, re.compile(r"\baccessLogger\.(?:Error|Warn)\s*\("), "Access Log must remain INFO request facts"))
    findings.extend(findings_for_pattern(path, code_with_data, re.compile(r"(?:\"data\"|\bData)\s*:\s*\berr\.Error\s*\(\s*\)"), "public error data must not expose err.Error()"))
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
