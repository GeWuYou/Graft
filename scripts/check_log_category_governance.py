#!/usr/bin/env python3
"""Reject literal logger categories in handwritten production Go."""

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


def find_matching_paren(source: str, open_index: int) -> int | None:
    depth = 0
    quote = ""
    escaped = False
    for index in range(open_index, len(source)):
        char = source[index]
        if quote:
            if quote == '"' and escaped:
                escaped = False
            elif quote == '"' and char == "\\":
                escaped = True
            elif char == quote:
                quote = ""
            continue
        if char in ('"', "'", "`"):
            quote = char
        elif char == "(":
            depth += 1
        elif char == ")":
            depth -= 1
            if depth == 0:
                return index
    return None


def split_call_arguments(source: str) -> list[str]:
    arguments: list[str] = []
    start = 0
    depth = 0
    quote = ""
    escaped = False
    for index, char in enumerate(source):
        if quote:
            if quote == '"' and escaped:
                escaped = False
            elif quote == '"' and char == "\\":
                escaped = True
            elif char == quote:
                quote = ""
            continue
        if char in ('"', "'", "`"):
            quote = char
        elif char in "([{":
            depth += 1
        elif char in ")]}":
            depth -= 1
        elif char == "," and depth == 0:
            arguments.append(source[start:index].strip())
            start = index + 1
    arguments.append(source[start:].strip())
    return arguments


def find_qualified_call(source: str, marker: str, start: int) -> int | None:
    index = start
    quote = ""
    escaped = False
    while index < len(source):
        char = source[index]
        if quote:
            if quote == '"' and escaped:
                escaped = False
            elif quote == '"' and char == "\\":
                escaped = True
            elif char == quote:
                quote = ""
            index += 1
            continue
        if source.startswith("//", index):
            newline = source.find("\n", index)
            index = len(source) if newline == -1 else newline + 1
            continue
        if source.startswith("/*", index):
            end = source.find("*/", index + 2)
            index = len(source) if end == -1 else end + 2
            continue
        if char in ('"', "'", "`"):
            quote = char
            index += 1
            continue
        if source.startswith(marker, index) and _has_exact_marker_boundary(source, index, marker):
            return index
        index += 1
    return None


def _is_identifier_char(char: str) -> bool:
    return char.isalnum() or char == "_"


def _has_exact_marker_boundary(source: str, index: int, marker: str) -> bool:
    if index > 0 and (_is_identifier_char(source[index - 1]) or source[index - 1] == "."):
        return False
    end = index + len(marker)
    return end >= len(source) or not _is_identifier_char(source[end])


def find_string_constants(source: str) -> set[str]:
    pattern = re.compile(
        r"\bconst\s+(?:\(\s*)?([A-Za-z_]\w*)\s+(?:[A-Za-z_]\w*\s+)?="
        r"\s*(?:\"(?:\\.|[^\"\\])*\"|`[^`]*`)",
    )
    return {match.group(1) for match in pattern.finditer(source)}


def scan_literal_argument_calls(path: Path, source: str, marker: str, argument_index: int) -> list[Finding]:
    findings: list[Finding] = []
    string_constants = find_string_constants(source)
    start = 0
    while (marker_index := find_qualified_call(source, marker, start)) is not None:
        open_index = marker_index + len(marker)
        while open_index < len(source) and source[open_index].isspace():
            open_index += 1
        if open_index >= len(source) or source[open_index] != "(":
            start = open_index
            continue
        close_index = find_matching_paren(source, open_index)
        if close_index is None:
            start = open_index + 1
            continue
        arguments = split_call_arguments(source[open_index + 1 : close_index])
        if len(arguments) > argument_index:
            argument = arguments[argument_index].lstrip()
            is_literal = argument.startswith(('"', '`'))
            is_string_constant = re.fullmatch(r"[A-Za-z_]\w*", argument) and argument in string_constants
            if is_literal or is_string_constant:
                findings.append(Finding(path, source.count("\n", 0, marker_index) + 1))
        start = close_index + 1
    return findings


def scan_source(path: Path, source: str) -> list[Finding]:
    findings = scan_literal_argument_calls(path, source, "logger.Category", 1)
    findings.extend(scan_literal_argument_calls(path, source, "logger.WithCategory", 1))
    findings.extend(scan_literal_argument_calls(path, source, "logger.LogCategory", 0))
    return findings


def scan_repository(root: Path) -> list[Finding]:
    findings: list[Finding] = []
    for path in sorted(root.joinpath("server").rglob("*.go")):
        relative = path.relative_to(root)
        if not is_production_go_path(relative):
            continue
        findings.extend(scan_source(relative, path.read_text(encoding="utf-8")))
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description="Reject literal logger categories in production server Go.")
    parser.add_argument("--root", type=Path, default=Path.cwd(), help="repository root")
    args = parser.parse_args()
    findings = scan_repository(args.root.resolve())
    if not findings:
        print("log category governance: ok")
        return 0
    for finding in findings:
        print(
            f"{finding.path}:{finding.line}: logger category APIs require a logger typed category constant, not a literal",
            file=sys.stderr,
        )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
