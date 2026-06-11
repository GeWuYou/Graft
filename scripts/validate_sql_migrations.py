#!/usr/bin/env python3
# Copyright (c) 2025-2026 GeWuYou
# SPDX-License-Identifier: Apache-2.0

"""Validate live PostgreSQL migration SQL comments and versions."""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path

from check_migration_versions import default_migration_dirs, iter_sql_files, repo_root


CREATE_TABLE_RE = re.compile(
    r"CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?P<table>\"[^\"]+\"|[A-Za-z_][\w]*)\s*\((?P<body>.*?)\)\s*;",
    re.IGNORECASE | re.DOTALL,
)
ALTER_TABLE_RE = re.compile(
    r"ALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?P<table>\"[^\"]+\"|[A-Za-z_][\w]*)\s+(?P<body>.*?);",
    re.IGNORECASE | re.DOTALL,
)
ADD_COLUMN_RE = re.compile(
    r"ADD\s+COLUMN\s+(?:IF\s+NOT\s+EXISTS\s+)?(?P<column>\"[^\"]+\"|[A-Za-z_][\w]*)",
    re.IGNORECASE,
)
COMMENT_TABLE_RE = re.compile(
    r"COMMENT\s+ON\s+TABLE\s+(?P<table>\"[^\"]+\"|[A-Za-z_][\w]*)\s+IS\s+'(?P<comment>(?:''|[^'])*)'",
    re.IGNORECASE | re.DOTALL,
)
COMMENT_COLUMN_RE = re.compile(
    r"COMMENT\s+ON\s+COLUMN\s+"
    r"(?:(?:\"(?P<quoted_table>[^\"]+)\"\.\"(?P<quoted_column>[^\"]+)\")|"
    r"(?:(?P<table>[A-Za-z_][\w]*)\.(?P<column>[A-Za-z_][\w]*)))"
    r"\s+IS\s+'(?P<comment>(?:''|[^'])*)'",
    re.IGNORECASE | re.DOTALL,
)
SQL_NAME_RE = re.compile(r"^(?P<version>\d+)_.+\.sql$")
CHINESE_RE = re.compile(r"[\u4e00-\u9fff]")
PLACEHOLDER_RE = re.compile(r"\b(TODO|TBD|placeholder)\b|待补充|临时说明", re.IGNORECASE)
CONSTRAINT_PREFIXES = (
    "PRIMARY ",
    "UNIQUE ",
    "CONSTRAINT ",
    "FOREIGN ",
    "CHECK ",
    "EXCLUDE ",
    "LIKE ",
)


@dataclass(frozen=True)
class Finding:
    path: Path
    message: str
    table: str | None = None
    column: str | None = None
    line: int | None = None

    def format(self, root: Path) -> str:
        try:
            display_path = self.path.relative_to(root)
        except ValueError:
            display_path = self.path
        location = str(display_path)
        if self.line is not None:
            location += f":{self.line}"
        target = ""
        if self.table:
            target = f" table={self.table}"
        if self.column:
            target += f" column={self.column}"
        return f"{location}:{target} {self.message}"


@dataclass(frozen=True)
class Comment:
    text: str
    line: int


def sql_unquote(identifier: str) -> str:
    value = identifier.strip()
    if len(value) >= 2 and value[0] == value[-1] == '"':
        return value[1:-1]
    return value


def sql_unescape(value: str) -> str:
    return value.replace("''", "'")


def line_number(sql: str, offset: int) -> int:
    return sql.count("\n", 0, offset) + 1


def split_sql_list(body: str) -> list[str]:
    parts: list[str] = []
    current: list[str] = []
    depth = 0
    in_string = False
    index = 0
    while index < len(body):
        char = body[index]
        if char == "'":
            current.append(char)
            if in_string and index + 1 < len(body) and body[index + 1] == "'":
                index += 1
                current.append(body[index])
            else:
                in_string = not in_string
        elif not in_string and char == "(":
            depth += 1
            current.append(char)
        elif not in_string and char == ")":
            depth -= 1
            current.append(char)
        elif not in_string and depth == 0 and char == ",":
            part = "".join(current).strip()
            if part:
                parts.append(part)
            current = []
        else:
            current.append(char)
        index += 1

    part = "".join(current).strip()
    if part:
        parts.append(part)
    return parts


def is_column_definition(part: str) -> bool:
    upper = part.lstrip().upper()
    return bool(upper) and not upper.startswith(CONSTRAINT_PREFIXES) and not upper.startswith("--")


def column_name(part: str) -> str:
    return sql_unquote(part.strip().split()[0])


def is_identifier_restatement(comment: str, identifier: str) -> bool:
    normalized_comment = re.sub(r"[\s_`\"'.-]+", "", comment).lower()
    normalized_identifier = re.sub(r"[\s_`\"'.-]+", "", identifier).lower()
    return normalized_comment == normalized_identifier


def parse_comments(sql: str) -> tuple[dict[str, Comment], dict[tuple[str, str], Comment]]:
    table_comments: dict[str, Comment] = {}
    column_comments: dict[tuple[str, str], Comment] = {}

    for match in COMMENT_TABLE_RE.finditer(sql):
        table = sql_unquote(match.group("table"))
        table_comments[table] = Comment(sql_unescape(match.group("comment")), line_number(sql, match.start()))

    for match in COMMENT_COLUMN_RE.finditer(sql):
        table = match.group("quoted_table") or match.group("table")
        column = match.group("quoted_column") or match.group("column")
        column_comments[(table, column)] = Comment(sql_unescape(match.group("comment")), line_number(sql, match.start()))

    return table_comments, column_comments


def validate_comment(path: Path, target: str, comment: Comment, kind: str, table: str, column: str | None = None) -> list[Finding]:
    findings: list[Finding] = []
    text = comment.text.strip()
    if not text:
        findings.append(Finding(path, f"{kind} comment is empty", table, column, comment.line))
    if not CHINESE_RE.search(text):
        findings.append(Finding(path, f"{kind} comment must contain Chinese text", table, column, comment.line))
    if PLACEHOLDER_RE.search(text):
        findings.append(Finding(path, f"{kind} comment must not use TODO/TBD/placeholder wording", table, column, comment.line))
    if is_identifier_restatement(text, target):
        findings.append(Finding(path, f"{kind} comment must describe business meaning instead of restating the identifier", table, column, comment.line))
    return findings


def validate_file(path: Path) -> list[Finding]:
    sql = path.read_text(encoding="utf-8")
    table_comments, column_comments = parse_comments(sql)
    findings: list[Finding] = []

    for match in CREATE_TABLE_RE.finditer(sql):
        table = sql_unquote(match.group("table"))
        create_line = line_number(sql, match.start())
        table_comment = table_comments.get(table)
        if table_comment is None:
            findings.append(Finding(path, "CREATE TABLE is missing COMMENT ON TABLE", table, line=create_line))
        else:
            findings.extend(validate_comment(path, table, table_comment, "table", table))

        for part in split_sql_list(match.group("body")):
            if not is_column_definition(part):
                continue
            column = column_name(part)
            column_comment = column_comments.get((table, column))
            if column_comment is None:
                findings.append(Finding(path, "CREATE TABLE column is missing COMMENT ON COLUMN", table, column, create_line))
            else:
                findings.extend(validate_comment(path, column, column_comment, "column", table, column))

    for match in ALTER_TABLE_RE.finditer(sql):
        table = sql_unquote(match.group("table"))
        alter_line = line_number(sql, match.start())
        for add_match in ADD_COLUMN_RE.finditer(match.group("body")):
            column = sql_unquote(add_match.group("column"))
            column_comment = column_comments.get((table, column))
            if column_comment is None:
                findings.append(Finding(path, "ALTER TABLE ADD COLUMN is missing COMMENT ON COLUMN", table, column, alter_line))
            else:
                findings.extend(validate_comment(path, column, column_comment, "column", table, column))

    return findings


def validate_versions(files: list[Path], root: Path) -> list[Finding]:
    versions: dict[str, list[Path]] = {}
    for path in files:
        match = SQL_NAME_RE.match(path.name)
        if not match:
            continue
        versions.setdefault(match.group("version"), []).append(path)

    findings: list[Finding] = []
    for version, paths in sorted(versions.items()):
        if len(paths) <= 1:
            continue
        joined = ", ".join(str(path.relative_to(root)) for path in sorted(paths))
        findings.append(Finding(paths[0], f"live migration version {version} is reused by: {joined}"))
    return findings


def live_sql_files(root: Path) -> list[Path]:
    dirs = sorted(default_migration_dirs(root))
    return [path for _, path in iter_sql_files(dirs)]


def validate(paths: list[Path], root: Path) -> list[Finding]:
    findings: list[Finding] = []
    findings.extend(validate_versions(paths, root))
    for path in sorted(paths):
        findings.extend(validate_file(path))
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate live migration SQL comments and globally unique versions.")
    parser.add_argument(
        "--paths",
        nargs="*",
        type=Path,
        help="Optional SQL files to validate. Defaults to all live default-chain migration files.",
    )
    args = parser.parse_args()

    root = repo_root()
    paths = [path if path.is_absolute() else root / path for path in args.paths] if args.paths else live_sql_files(root)
    if not paths:
        print("sql migration gate: skip (no live migration SQL files found)")
        return 0

    findings = validate(paths, root)
    if not findings:
        print(f"sql migration gate: ok ({len(paths)} files)")
        return 0

    print("sql migration gate: failed", file=sys.stderr)
    for finding in findings:
        print(f"- {finding.format(root)}", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
