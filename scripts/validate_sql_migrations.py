#!/usr/bin/env python3
"""Validate live PostgreSQL migration SQL comments and versions."""

from __future__ import annotations

import argparse
import hashlib
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path

import yaml

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
MIGRATION_PRECHECK_RE = re.compile(r"^(?P<stem>.+)\.preflight\.ya?ml$")
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
RISK_LEVELS = ("L0", "L1", "L2", "L3", "L4", "L5")
RISK_INDEX = {value: index for index, value in enumerate(RISK_LEVELS)}
GOVERNANCE_PATH = "ai-plan/design/governance/backend/数据库表设计与迁移规范.md"
LESSONS_PATH = "ai-plan/lessons/migrations.md"


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


@dataclass(frozen=True)
class MigrationPreflight:
    path: Path
    data: dict[str, object]


def content_revision(path: Path) -> str:
    return f"sha256:{hashlib.sha256(path.read_bytes()).hexdigest()}"


def mapping(value: object) -> dict[str, object]:
    return value if isinstance(value, dict) else {}


def string_list(value: object) -> list[str]:
    return [item for item in value if isinstance(item, str)] if isinstance(value, list) else []


def active_migration_lessons(path: Path) -> set[str]:
    if not path.is_file():
        return set()
    return set(re.findall(r"^##\s+(MIG-\d{3})\b.*?\n\s*- Status:\s*active\b", path.read_text(encoding="utf-8"), re.MULTILINE))


def sidecar_candidates(sql_path: Path) -> list[Path]:
    pattern = f"{sql_path.stem}.preflight.y*ml"
    return sorted(path for path in sql_path.parent.glob(pattern) if MIGRATION_PRECHECK_RE.match(path.name))


def load_preflight(sql_path: Path) -> tuple[MigrationPreflight | None, list[Finding]]:
    candidates = sidecar_candidates(sql_path)
    if len(candidates) != 1:
        return None, [Finding(sql_path, "live migration must have exactly one .preflight.yaml sidecar")]
    try:
        raw = yaml.safe_load(candidates[0].read_text(encoding="utf-8"))
    except yaml.YAMLError as error:
        return None, [Finding(candidates[0], f"invalid migration preflight YAML: {error}")]
    if not isinstance(raw, dict):
        return None, [Finding(candidates[0], "migration preflight YAML root must be a mapping")]
    return MigrationPreflight(candidates[0], raw), []


def derive_minimum_risk(sql: str) -> str:
    normalized = re.sub(r"--.*?$", "", sql, flags=re.MULTILINE).upper()
    if re.search(r"\b(DROP\s+|DELETE\s+FROM|TRUNCATE\b|RECONCIL|RETIR|SET\s+DELETED_AT)", normalized):
        return "L4"
    if re.search(r"\b(UPDATE\b|INSERT\s+INTO.*SELECT\b|BACKFILL)\b", normalized):
        return "L3"
    if re.search(r"\b(CREATE\s+UNIQUE\s+INDEX|\bUNIQUE\b|ADD\s+CONSTRAINT|FOREIGN\s+KEY)\b", normalized):
        return "L2"
    if re.search(r"\bALTER\s+TABLE\b", normalized):
        return "L1"
    return "L0"


def required_safety_levels(sql: str) -> tuple[str, ...]:
    normalized = re.sub(r"--.*?$", "", sql, flags=re.MULTILINE).upper()
    levels: list[str] = []
    if re.search(r"\b(CREATE\s+UNIQUE\s+INDEX|\bUNIQUE\b|ADD\s+CONSTRAINT|FOREIGN\s+KEY)\b", normalized):
        levels.append("L2")
    if re.search(r"\b(UPDATE\b|INSERT\s+INTO.*SELECT\b|BACKFILL)\b", normalized):
        levels.append("L3")
    if re.search(r"\b(DROP\s+|DELETE\s+FROM|TRUNCATE\b|RECONCIL|RETIR|SET\s+DELETED_AT)", normalized):
        levels.append("L4")
    if not levels:
        levels.append(derive_minimum_risk(sql))
    return tuple(levels)


def validate_preflight(sql_path: Path, root: Path) -> list[Finding]:
    preflight, findings = load_preflight(sql_path)
    if preflight is None:
        return findings

    data = preflight.data
    migration = mapping(data.get("migration"))
    expected_path = str(sql_path.relative_to(root))
    version = SQL_NAME_RE.match(sql_path.name).group("version") if SQL_NAME_RE.match(sql_path.name) else ""
    if migration.get("path") != expected_path:
        findings.append(Finding(preflight.path, f"migration.path must be {expected_path}"))
    if str(migration.get("version", "")) != version:
        findings.append(Finding(preflight.path, f"migration.version must be {version}"))
    if not isinstance(data.get("owner"), str) or not data["owner"].strip():
        findings.append(Finding(preflight.path, "owner is required"))

    declared_risk = data.get("risk_level")
    if declared_risk not in RISK_LEVELS:
        findings.append(Finding(preflight.path, "risk_level must be one of L0, L1, L2, L3, L4, L5"))
    else:
        minimum = derive_minimum_risk(sql_path.read_text(encoding="utf-8"))
        if RISK_INDEX[declared_risk] < RISK_INDEX[minimum]:
            findings.append(Finding(preflight.path, f"risk_level {declared_risk} understates SQL-derived minimum {minimum}"))

    for key in ("affected_tables", "operation_categories", "historical_data_assumptions", "referenced_tables", "planned_upgrade_order", "validation_scenarios"):
        if not string_list(data.get(key)):
            findings.append(Finding(preflight.path, f"{key} must be a non-empty list of strings"))

    receipt = mapping(data.get("retrieval_receipt"))
    governance = mapping(receipt.get("governance"))
    lessons = mapping(receipt.get("lessons"))
    for label, expected, record in (("governance", GOVERNANCE_PATH, governance), ("lessons", LESSONS_PATH, lessons)):
        authority_path = root / expected
        if record.get("path") != expected:
            findings.append(Finding(preflight.path, f"retrieval_receipt.{label}.path must be {expected}"))
        elif not authority_path.is_file():
            findings.append(Finding(preflight.path, f"retrieval_receipt.{label} authority file is missing"))
        elif record.get("revision") != content_revision(authority_path):
            findings.append(Finding(preflight.path, f"retrieval_receipt.{label}.revision does not match canonical content"))
    lesson_ids = string_list(receipt.get("lesson_ids"))
    active_ids = active_migration_lessons(root / LESSONS_PATH)
    if any(not re.fullmatch(r"MIG-\d{3}", lesson_id) for lesson_id in lesson_ids):
        findings.append(Finding(preflight.path, "retrieval_receipt.lesson_ids must contain only MIG-### IDs"))
    inactive = sorted(set(lesson_ids) - active_ids)
    if inactive:
        findings.append(Finding(preflight.path, f"retrieval_receipt.lesson_ids are not active migration lessons: {', '.join(inactive)}"))

    safety = mapping(data.get("safety_strategy"))
    required_safety: dict[str, tuple[str, ...]] = {
        "L2": ("duplicate_scan", "live_reference_scan", "reconcile_or_abort", "post_migration_invariant"),
        "L3": ("bounded_backfill",),
        "L4": ("reference_impact", "recovery_or_retirement_rationale"),
        "L5": ("backup_restore_owner", "release_upgrade_documentation"),
    }
    applicable_levels = [level for level in required_safety_levels(sql_path.read_text(encoding="utf-8")) if level in required_safety]
    if applicable_levels:
        for level in applicable_levels:
            for key in required_safety[level]:
                if not isinstance(safety.get(key), str) or not safety[key].strip():
                    findings.append(Finding(preflight.path, f"safety_strategy.{key} is required for {level}"))
        checks = data.get("preflight_checks")
        if not isinstance(checks, list) or not checks:
            findings.append(Finding(preflight.path, "preflight_checks is required for risk-bearing migration operations"))
    return findings


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


def unique_paths(paths: list[Path]) -> list[Path]:
    unique: list[Path] = []
    seen: set[Path] = set()
    for path in paths:
        normalized = path.resolve(strict=False)
        if normalized in seen:
            continue
        seen.add(normalized)
        unique.append(path)
    return unique


def validate(paths: list[Path], root: Path, require_preflight: bool = False) -> list[Finding]:
    findings: list[Finding] = []
    findings.extend(validate_versions(unique_paths([*paths, *live_sql_files(root)]), root))
    for path in sorted(paths):
        if require_preflight or sidecar_candidates(path):
            findings.extend(validate_preflight(path, root))
        findings.extend(validate_file(path))
    return findings


def git_changed_sql_entries(root: Path, base_ref: str | None, staged: bool) -> list[tuple[str, Path]]:
    if not staged and not base_ref:
        raise ValueError("--changed requires --base-ref")

    command = ["git", "diff", "--name-status", "-z", "--diff-filter=ACMR"]
    if staged:
        command.append("--cached")
    else:
        command.append(f"{base_ref}...HEAD")
    output = subprocess.check_output(command, cwd=root).decode("utf-8")
    fields = output.split("\0")
    paths: list[tuple[str, Path]] = []
    index = 0
    while index < len(fields) - 1:
        status = fields[index]
        index += 1
        if not status:
            continue
        if status.startswith("R"):
            if index + 1 >= len(fields):
                break
            candidates = (fields[index], fields[index + 1])
            index += 2
        else:
            if index >= len(fields):
                break
            candidates = (fields[index],)
            index += 1
        for entry in candidates:
            path = root / entry
            if entry.endswith(".sql") and path.parent in default_migration_dirs(root):
                paths.append((status, path))
    return paths


def has_unpublished_exception(sql_path: Path) -> bool:
    preflight, _ = load_preflight(sql_path)
    if preflight is None:
        return False
    exception = mapping(preflight.data.get("unpublished_exception"))
    return all(isinstance(exception.get(key), str) and exception[key].strip() for key in ("reason", "evidence"))


def validate_historical_immutability(entries: list[tuple[str, Path]]) -> list[Finding]:
    findings: list[Finding] = []
    for status, path in entries:
        if status.startswith(("M", "R")) and not has_unpublished_exception(path):
            findings.append(Finding(path, "historical live migration was modified; add a higher-version migration or governed unpublished_exception evidence"))
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate live migration SQL comments, sidecars, and globally unique versions.")
    parser.add_argument(
        "--paths",
        nargs="*",
        type=Path,
        help="Optional SQL files to validate. Defaults to all live default-chain migration files.",
    )
    parser.add_argument("--changed", action="store_true", help="require sidecars for SQL files changed from --base-ref")
    parser.add_argument("--staged", action="store_true", help="require sidecars for staged migration SQL files")
    parser.add_argument("--base-ref", help="Git base ref used with --changed")
    args = parser.parse_args()

    root = repo_root()
    if args.changed or args.staged:
        try:
            changed_entries = git_changed_sql_entries(root, args.base_ref, args.staged)
        except ValueError as error:
            parser.error(str(error))
        paths = [path for _, path in changed_entries]
        require_preflight = True
    else:
        if not args.paths:
            parser.error("select migration files with --paths or use --changed --base-ref <ref>")
        paths = [path if path.is_absolute() else root / path for path in args.paths]
        require_preflight = True
    if not paths:
        print("sql migration gate: skip (no live migration SQL files found)")
        return 0

    findings = validate(paths, root, require_preflight=require_preflight)
    if args.changed or args.staged:
        findings.extend(validate_historical_immutability(changed_entries))
    if not findings:
        print(f"sql migration gate: ok ({len(paths)} files)")
        return 0

    print("sql migration gate: failed", file=sys.stderr)
    for finding in findings:
        print(f"- {finding.format(root)}", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
