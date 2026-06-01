#!/usr/bin/env python3
"""Repository-local contract governance scanner.

Phase 1 scope:
- detect new high-risk contract literals in server/web automation-safe layers
- support baseline / allowlist metadata
- provide report-only duplicate candidates and drift candidates

This script is intentionally conservative. It prefers a smaller set of
high-signal findings over broad low-confidence repository linting.
"""

from __future__ import annotations

import argparse
import dataclasses
import datetime as dt
import json
import os
import pathlib
import re
import subprocess
import sys
from collections import Counter, defaultdict
from typing import Any, Iterable


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT_DIR = pathlib.Path(__file__).resolve().parent
BASELINE_PATH = SCRIPT_DIR / "baseline.json"
ALLOWLIST_PATH = SCRIPT_DIR / "allowlist.json"

SEVERITY_ORDER = {"P0": 0, "P1": 1, "P2": 2, "P3": 3}
BLOCKING_SEVERITIES = {"P0", "P1"}

SKIP_DIR_PREFIXES = (
    ".git/",
    ".agents/",
    ".ai/",
    "openapi/dist/",
    "server/internal/ent/",
    "server/vendor/",
    "web/node_modules/",
    "web/dist/",
    "web/src/assets/",
)

WARNING_ONLY_PREFIXES = (
    "web/mock/",
    "web/src/pages/dashboard/",
    "web/src/pages/detail/",
    "web/src/pages/form/",
    "web/src/pages/list/",
    "web/src/pages/result/",
    "web/src/constants/",
)

ALWAYS_INCLUDE_FILES = {
    "web/src/utils/request.ts",
    "web/src/store/modules/user.ts",
    "web/src/router/index.ts",
    "server/internal/httpx/response.go",
    "server/internal/i18n/service.go",
    "server/internal/pluginapi/audit.go",
    "server/modules/user/plugin_routes.go",
}

HEADER_NAMES = {
    "Authorization",
    "Accept-Language",
    "X-Graft-Locale",
    "X-Request-Id",
    "X-Trace-Id",
}

AUTH_CODES = {
    "AUTH_INVALID_CREDENTIALS",
    "AUTH_INVALID_REFRESH_SESSION",
    "AUTH_MISSING_ACTOR",
    "AUTH_MISSING_PERMISSION",
    "AUTH_TOKEN_MISSING",
    "AUTH_TOKEN_EXPIRED",
    "AUTH_TOKEN_INVALID",
    "AUTH_FORBIDDEN",
    "AUTH_PASSWORD_POLICY_VIOLATION",
    "AUTH_PASSWORD_REUSE_FORBIDDEN",
    "AUTH_CURRENT_PASSWORD_INVALID",
    "AUTH_SESSION_NOT_FOUND",
    "COMMON_INVALID_ARGUMENT",
    "COMMON_INTERNAL_ERROR",
    "USER_NOT_FOUND",
}

KNOWN_STORAGE_KEYS = {
    "user",
    "tdesign-starter-locale",
}

KNOWN_EVENT_NAMES = {
    "audit.record",
}

KNOWN_SPECIAL_ROUTES = {
    "/auth/restricted-session",
    "/api/auth/refresh",
}

KNOWN_PERMISSION_CODES = {
    "user.read",
    "user.session.read",
    "user.session.revoke",
    "dashboard.view",
}

KNOWN_MESSAGE_KEYS = {
    "auth.invalid_credentials",
    "auth.token_missing",
    "auth.token_expired",
    "auth.token_invalid",
    "auth.forbidden",
    "auth.invalid_refresh_session",
    "auth.password_policy_violation",
    "auth.password_reuse_forbidden",
    "auth.current_password_invalid",
    "auth.missing_actor",
    "auth.missing_permission",
    "auth.session_not_found",
    "common.conjunction",
    "common.copyright",
    "common.internal_error",
    "common.invalid_argument",
    "user.not_found",
}

MESSAGE_KEY_PREFIXES = ("auth.", "common.", "validation.")
HEADER_PREFIX_RE = re.compile(r"^X-[A-Za-z-]+$")
ERROR_CODE_RE = re.compile(r"^(?:AUTH|COMMON|USER|RBAC|AUDIT|SCHEDULER)_[A-Z0-9_]+$")
MESSAGE_KEY_RE = re.compile(r"^(?:auth|common|validation|user)\.[a-z0-9_.-]+$")
PERMISSION_CODE_RE = re.compile(r"^[a-z][a-z0-9]*(?:\.[a-z][a-z0-9]*)+$")
API_PATH_RE = re.compile(r"^/api/[A-Za-z0-9/_:-]+$")
ROUTE_PATH_RE = re.compile(r"^/(?:[A-Za-z0-9_.:-]+/?)*$")
LOCAL_STORAGE_CALL_RE = re.compile(
    r"""(?P<api>localStorage|sessionStorage)\.(?P<op>getItem|setItem|removeItem)\(\s*["'](?P<key>[^"']+)["']"""
)
ROUTER_PUSH_LITERAL_RE = re.compile(r"""router\.push\(\s*["'](?P<path>/[^"']*)["']""")
HEADER_LITERAL_RE = re.compile(r"""["'](?P<header>Authorization|Accept-Language|X-[A-Za-z-]+)["']""")
MESSAGE_KEY_LITERAL_RE = re.compile(r"""["'](?P<key>(?:auth|common|validation|user)\.[a-z0-9_.-]+)["']""")
ERROR_CODE_LITERAL_RE = re.compile(r"""["'](?P<code>(?:AUTH|COMMON|USER|RBAC|AUDIT|SCHEDULER)_[A-Z0-9_]+)["']""")
EVENT_NAME_LITERAL_RE = re.compile(r"""["'](?P<event>[a-z][a-z0-9]*(?:\.[a-z][a-z0-9]*)+)["']""")
PERMISSION_LITERAL_RE = re.compile(r"""["'](?P<permission>[a-z][a-z0-9]*(?:\.[a-z][a-z0-9]*)+)["']""")
API_PATH_LITERAL_RE = re.compile(r"""["'](?P<path>/api/[A-Za-z0-9/_:-]+)["']""")
GO_TEST_FILE_RE = re.compile(r"_test\.go$")
TS_TEST_FILE_RE = re.compile(r"\.(?:test|spec)\.(?:ts|tsx)$")
TEST_API_PATH_DEFINITION_RE = re.compile(r"""^\s*const\s+[A-Z0-9_]+_PATH\s*=\s*["']/api/[A-Za-z0-9/_:-]+["']""")
TEST_ASSERTION_RE = re.compile(r"\b(?:expect|assert(?:\w+)?|require(?:\w+)?|t\.(?:Fatal|Fatalf|Error|Errorf))\b")
TEST_GO_COMPARISON_ASSERTION_RE = re.compile(r"""^\s*if\b.*(?:==|!=).*(?:["'][^"']+["'])""")
TEST_OBJECT_FIELD_RE = re.compile(
    r"""\b(?:code|messageKey|permissions?|path|title_key|permission|Name)\s*:\s*["'\[]"""
)
TEST_ROUTE_SETUP_RE = re.compile(r"""\.\s*(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS|Any)\(\s*["']/""")
TEST_HTTP_REQUEST_RE = re.compile(r"""(?:httptest\.NewRequest|newBearerRequest)\(""")
TEST_HEADER_SETUP_RE = re.compile(r"""\.Header\.(?:Set|Add)\(""")
TEST_EVENT_SETUP_RE = re.compile(r"""\b(?:Subscribe|Publish)\(""")
TEST_PERMISSION_SETUP_RE = re.compile(r"""\b(?:RequirePermission|grantedCodes)\b""")
TEXT_FILE_SUFFIXES = {
    ".go",
    ".ts",
    ".tsx",
    ".js",
    ".mjs",
    ".cjs",
    ".jsx",
    ".vue",
    ".md",
    ".yml",
    ".yaml",
    ".json",
}


@dataclasses.dataclass(frozen=True)
class Finding:
    rule: str
    severity: str
    path: str
    line: int
    value: str
    message: str
    action: str
    report_only: bool = False

    @property
    def key(self) -> tuple[str, str, str, int, str]:
        return (self.rule, self.path, self.severity, self.line, self.value)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--mode", choices=("changed", "ci", "report"), required=True)
    parser.add_argument("--baseline", default=str(BASELINE_PATH))
    parser.add_argument("--allowlist", default=str(ALLOWLIST_PATH))
    parser.add_argument("--output-json")
    return parser.parse_args()


def run_git(args: list[str]) -> str:
    completed = subprocess.run(
        ["git", *args],
        cwd=REPO_ROOT,
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if completed.returncode != 0:
        stderr = completed.stderr.strip()
        stdout = completed.stdout.strip()
        details = stderr or stdout or "unknown git failure"
        raise RuntimeError(f"git {' '.join(args)} failed: {details}")
    return completed.stdout.strip()


def tracked_files() -> list[str]:
    output = run_git(["ls-files"])
    if not output:
        return []
    return [line for line in output.splitlines() if line]


def staged_or_changed_files() -> list[str]:
    staged = run_git(["diff", "--cached", "--name-only", "--diff-filter=ACMR"])
    if staged:
        return [line for line in staged.splitlines() if line]

    changed = run_git(["diff", "HEAD", "--name-only", "--diff-filter=ACMR"])
    if changed:
        return [line for line in changed.splitlines() if line]

    return []


def ci_changed_files() -> list[str]:
    local_changed = staged_or_changed_files()
    if local_changed:
        return local_changed

    base_ref = os.environ.get("GITHUB_BASE_REF", "").strip()
    head_sha = os.environ.get("GITHUB_SHA", "").strip()

    if base_ref:
        try:
            merge_base = run_git(["merge-base", "HEAD", f"origin/{base_ref}"])
        except RuntimeError:
            merge_base = ""
        if merge_base:
            try:
                changed = run_git(["diff", "--name-only", "--diff-filter=ACMR", f"{merge_base}...HEAD"])
            except RuntimeError:
                changed = ""
            if changed:
                return [line for line in changed.splitlines() if line]

    if head_sha:
        try:
            previous = run_git(["rev-parse", f"{head_sha}^"])
        except RuntimeError:
            previous = ""
        if previous:
            try:
                changed = run_git(["diff", "--name-only", "--diff-filter=ACMR", f"{previous}...{head_sha}"])
            except RuntimeError:
                changed = ""
            if changed:
                return [line for line in changed.splitlines() if line]

    return tracked_files()


def is_skipped_path(path: str) -> bool:
    if any(path.startswith(prefix) for prefix in SKIP_DIR_PREFIXES):
        return True
    if path.endswith(".min.js") or path.endswith(".map") or "/migrations/" in path:
        return True
    if path.endswith(".gen.go"):
        return True
    return False


def is_warning_only_path(path: str) -> bool:
    return any(path.startswith(prefix) for prefix in WARNING_ONLY_PREFIXES)


def should_scan_file(path: str) -> bool:
    if is_skipped_path(path):
        return False
    suffix = pathlib.Path(path).suffix
    return suffix in TEXT_FILE_SUFFIXES or path in ALWAYS_INCLUDE_FILES


def load_metadata_entries(path: pathlib.Path) -> list[Any]:
    if not path.exists():
        return []
    with path.open("r", encoding="utf-8") as handle:
        data = json.load(handle)
    if not isinstance(data, list):
        raise ValueError(f"{path} must contain a top-level JSON array")
    return data


def validate_metadata_entries(entries: list[Any], kind: str) -> list[Finding]:
    findings: list[Finding] = []
    today = dt.date.today()
    required = {"id", "path", "rule", "severity", "owner", "reason", "created_at", "cleanup_phase", "expire_at"}
    for index, entry in enumerate(entries, start=1):
        if not isinstance(entry, dict):
            findings.append(
                Finding(
                    rule=f"{kind}-schema",
                    severity="P0",
                    path=str(SCRIPT_DIR / f"{kind}.json"),
                    line=index,
                    value=f"{kind}-{index}",
                    message=f"{kind} entry must be a JSON object, got {type(entry).__name__}",
                    action="fix metadata",
                )
            )
            continue
        missing = sorted(required - set(entry))
        if missing:
            findings.append(
                Finding(
                    rule=f"{kind}-schema",
                    severity="P0",
                    path=str(SCRIPT_DIR / f"{kind}.json"),
                    line=index,
                    value=entry.get("id", f"{kind}-{index}"),
                    message=f"{kind} entry is missing required fields: {', '.join(missing)}",
                    action="fix metadata",
                )
            )
            continue

        severity = str(entry["severity"])
        if severity not in SEVERITY_ORDER:
            findings.append(
                Finding(
                    rule=f"{kind}-schema",
                    severity="P0",
                    path=str(SCRIPT_DIR / f"{kind}.json"),
                    line=index,
                    value=str(entry["id"]),
                    message=f"{kind} entry has unsupported severity {severity}",
                    action="use P0/P1/P2/P3",
                )
            )

        reason = str(entry["reason"]).strip().lower()
        if reason in {"legacy", "todo", "temporary", "temp"}:
            findings.append(
                Finding(
                    rule=f"{kind}-weak-reason",
                    severity="P1",
                    path=str(SCRIPT_DIR / f"{kind}.json"),
                    line=index,
                    value=str(entry["id"]),
                    message=f"{kind} entry uses a weak reason field",
                    action="write a concrete engineering reason",
                )
            )

        try:
            expire_at = dt.date.fromisoformat(str(entry["expire_at"]))
            if expire_at < today:
                findings.append(
                    Finding(
                        rule=f"{kind}-expired",
                        severity="P1",
                        path=str(SCRIPT_DIR / f"{kind}.json"),
                        line=index,
                        value=str(entry["id"]),
                        message=f"{kind} entry expired on {expire_at.isoformat()}",
                        action="remove, renew, or clean up the exception",
                    )
                )
        except ValueError:
            findings.append(
                Finding(
                    rule=f"{kind}-schema",
                    severity="P0",
                    path=str(SCRIPT_DIR / f"{kind}.json"),
                    line=index,
                    value=str(entry["id"]),
                    message=f"{kind} entry has invalid expire_at date",
                    action="use YYYY-MM-DD",
                )
            )
    return findings


def build_suppression_index(entries: list[dict[str, Any]]) -> dict[tuple[str, str, str], dict[str, Any]]:
    index: dict[tuple[str, str, str], dict[str, Any]] = {}
    for entry in entries:
        key = (str(entry.get("path", "")), str(entry.get("rule", "")), str(entry.get("matched_value", entry.get("value", ""))))
        index[key] = entry
    return index


def read_file(path: str) -> str:
    try:
        return (REPO_ROOT / path).read_text(encoding="utf-8")
    except UnicodeDecodeError:
        return ""
    except FileNotFoundError:
        return ""


def is_definition_context(path: str, line_text: str, value: str) -> bool:
    if path.startswith("server/internal/contract/"):
        return True
    if path.startswith("server/modules/") and "/contract/" in path:
        return True
    if path.startswith("web/src/contracts/"):
        return True
    if path.startswith("web/src/modules/") and "/contract/permissions." in path:
        return True
    if path.startswith("web/src/modules/") and "/contract/paths." in path and API_PATH_RE.match(value):
        return True
    if path.startswith("web/src/api/model/") and "API_CODE" in line_text:
        return True
    if path == "web/src/router/index.ts" and value == "/auth/restricted-session":
        return True
    if (path.endswith(".test.ts") or path.endswith(".test.tsx") or path.endswith(".spec.ts")) and TEST_API_PATH_DEFINITION_RE.match(line_text):
        return True
    if path == "server/internal/i18n/service.go" and "messagecontract." in line_text:
        return True
    if path == "server/internal/httpx/response.go" and "httpheader." in line_text:
        return True
    if path == "server/internal/pluginapi/audit.go" and "AuditRecordEventName" in line_text:
        return True
    return False


def is_test_file(path: str) -> bool:
    return bool(GO_TEST_FILE_RE.search(path) or TS_TEST_FILE_RE.search(path))


def is_test_fixture_context(path: str, line_text: str, rule: str) -> bool:
    if not is_test_file(path):
        return False

    # Contract-literal checks focus on runtime drift. Test fixtures are allowed
    # to inline canonical contract values as sample inputs and assertions.
    if rule in {
        "header-literal",
        "message-key-literal",
        "error-code-literal",
        "api-path-literal",
        "permission-code-literal",
        "event-name-literal",
    }:
        return True

    if TEST_ASSERTION_RE.search(line_text):
        return True
    if TEST_GO_COMPARISON_ASSERTION_RE.search(line_text):
        return True

    if rule == "header-literal":
        return bool(TEST_HEADER_SETUP_RE.search(line_text))

    if rule == "api-path-literal":
        return bool(
            TEST_HTTP_REQUEST_RE.search(line_text)
            or TEST_ROUTE_SETUP_RE.search(line_text)
            or TEST_OBJECT_FIELD_RE.search(line_text)
        )

    if rule in {"message-key-literal", "error-code-literal"}:
        return bool(TEST_OBJECT_FIELD_RE.search(line_text))

    if rule == "event-name-literal":
        return bool(TEST_EVENT_SETUP_RE.search(line_text) or "Event{" in line_text)

    if rule == "permission-code-literal":
        return bool(TEST_OBJECT_FIELD_RE.search(line_text) or TEST_PERMISSION_SETUP_RE.search(line_text))

    return False


def adjust_severity(path: str, severity: str) -> str:
    if is_test_file(path):
        if severity in {"P0", "P1"}:
            return "P2"
        return severity
    if is_warning_only_path(path) and severity in {"P0", "P1"}:
        return "P2"
    return severity


def scan_file(path: str, text: str) -> list[Finding]:
    findings: list[Finding] = []
    lines = text.splitlines()
    for line_no, line in enumerate(lines, start=1):
        if not line.strip():
            continue

        if path.endswith(".md"):
            continue

        if path.endswith(".json") and "web/src/locales/lang/" in path:
            continue

        for match in LOCAL_STORAGE_CALL_RE.finditer(line):
            key = match.group("key")
            severity = "P1"
            if path.endswith(".test.ts") or path.endswith(".test.go"):
                severity = "P2"
            if key not in KNOWN_STORAGE_KEYS or not is_definition_context(path, line, key):
                findings.append(
                    Finding(
                        rule="storage-key-literal",
                        severity=adjust_severity(path, severity),
                        path=path,
                        line=line_no,
                        value=key,
                        message=f"hard-coded {match.group('api')} key '{key}'",
                        action="use a contract-owned storage key",
                    )
                )

        for match in ROUTER_PUSH_LITERAL_RE.finditer(line):
            route_path = match.group("path")
            if route_path == "/":
                severity = "P2"
            elif route_path in KNOWN_SPECIAL_ROUTES:
                severity = "P1"
            else:
                severity = "P2"
            findings.append(
                Finding(
                    rule="route-path-literal",
                    severity=adjust_severity(path, severity),
                    path=path,
                    line=line_no,
                    value=route_path,
                    message=f"hard-coded router push path '{route_path}'",
                    action="use route path contracts for stable navigation",
                )
            )

        for match in HEADER_LITERAL_RE.finditer(line):
            header = match.group("header")
            if header not in HEADER_NAMES:
                continue
            if is_definition_context(path, line, header):
                continue
            if is_test_fixture_context(path, line, "header-literal"):
                continue
            severity = "P0" if header in {"Authorization", "X-Graft-Locale", "X-Request-Id"} else "P1"
            findings.append(
                Finding(
                    rule="header-literal",
                    severity=adjust_severity(path, severity),
                    path=path,
                    line=line_no,
                    value=header,
                    message=f"hard-coded HTTP header '{header}'",
                    action="use a typed header contract",
                )
            )

        for match in MESSAGE_KEY_LITERAL_RE.finditer(line):
            key = match.group("key")
            if key in KNOWN_MESSAGE_KEYS and not is_definition_context(path, line, key):
                if is_test_fixture_context(path, line, "message-key-literal"):
                    continue
                findings.append(
                    Finding(
                        rule="message-key-literal",
                        severity=adjust_severity(path, "P1"),
                        path=path,
                        line=line_no,
                        value=key,
                        message=f"hard-coded message key '{key}'",
                        action="prefer a contract-owned message key",
                    )
                )

        for match in ERROR_CODE_LITERAL_RE.finditer(line):
            code = match.group("code")
            if code not in AUTH_CODES:
                continue
            if is_definition_context(path, line, code):
                continue
            if is_test_fixture_context(path, line, "error-code-literal"):
                continue
            findings.append(
                Finding(
                    rule="error-code-literal",
                    severity=adjust_severity(path, "P0"),
                    path=path,
                    line=line_no,
                    value=code,
                    message=f"hard-coded API error code '{code}'",
                    action="reuse the canonical error code contract",
                )
            )

        for match in API_PATH_LITERAL_RE.finditer(line):
            path_value = match.group("path")
            if is_definition_context(path, line, path_value):
                continue
            if is_test_fixture_context(path, line, "api-path-literal"):
                continue
            severity = "P1" if path_value in KNOWN_SPECIAL_ROUTES else "P2"
            findings.append(
                Finding(
                    rule="api-path-literal",
                    severity=adjust_severity(path, severity),
                    path=path,
                    line=line_no,
                    value=path_value,
                    message=f"hard-coded API path '{path_value}'",
                    action="prefer a contract-owned API path",
                )
            )

        for match in PERMISSION_LITERAL_RE.finditer(line):
            permission = match.group("permission")
            if permission not in KNOWN_PERMISSION_CODES:
                continue
            if is_definition_context(path, line, permission):
                continue
            if is_test_fixture_context(path, line, "permission-code-literal"):
                continue
            findings.append(
                Finding(
                    rule="permission-code-literal",
                    severity=adjust_severity(path, "P0"),
                    path=path,
                    line=line_no,
                    value=permission,
                    message=f"hard-coded permission code '{permission}'",
                    action="use a typed permission contract",
                )
            )

        for match in EVENT_NAME_LITERAL_RE.finditer(line):
            event_name = match.group("event")
            if event_name not in KNOWN_EVENT_NAMES:
                continue
            if is_definition_context(path, line, event_name):
                continue
            if is_test_fixture_context(path, line, "event-name-literal"):
                continue
            findings.append(
                Finding(
                    rule="event-name-literal",
                    severity=adjust_severity(path, "P0"),
                    path=path,
                    line=line_no,
                    value=event_name,
                    message=f"hard-coded event name '{event_name}'",
                    action="use the canonical event contract",
                )
            )

    return findings


def duplicate_string_candidates(files: Iterable[str]) -> list[Finding]:
    counter: Counter[str] = Counter()
    locations: dict[str, tuple[str, int]] = {}
    literal_re = re.compile(r'["\']([^"\']{8,})["\']')
    allowed_literal_re = re.compile(r"^[A-Za-z0-9._:/-]+$")

    for path in files:
        if is_skipped_path(path):
            continue
        text = read_file(path)
        if not text:
            continue
        if path.endswith(".md") or "web/src/locales/lang/" in path:
            continue

        for line_no, line in enumerate(text.splitlines(), start=1):
            for match in literal_re.finditer(line):
                literal = match.group(1)
                if "\n" in literal or literal.startswith("http"):
                    continue
                if not allowed_literal_re.match(literal):
                    continue
                if literal in KNOWN_STORAGE_KEYS or literal in HEADER_NAMES:
                    continue
                if MESSAGE_KEY_RE.match(literal) or ERROR_CODE_RE.match(literal):
                    continue
                if len(literal.split()) > 8:
                    continue
                counter[literal] += 1
                locations.setdefault(literal, (path, line_no))

    findings: list[Finding] = []
    for literal, count in counter.most_common():
        if count < 3:
            break
        path, line_no = locations[literal]
        findings.append(
            Finding(
                rule="duplicate-string-candidate",
                severity=adjust_severity(path, "P3"),
                path=path,
                line=line_no,
                value=literal,
                message=f"string literal appears {count} times and may deserve local or contract extraction",
                action="review whether the value is a contract or repeated module constant",
                report_only=True,
            )
        )
    return findings


def drift_candidates() -> list[Finding]:
    findings: list[Finding] = []

    server_auth_model = read_file("server/internal/contract/errorcode/code.go")
    web_auth_model = read_file("web/src/contracts/api/codes.ts")
    if server_auth_model and web_auth_model:
        server_codes = set(re.findall(r'"((?:AUTH|COMMON)_[A-Z0-9_]+)"', server_auth_model))
        server_codes.update(re.findall(r'"(USER_[A-Z0-9_]+)"', server_auth_model))
        web_codes = set(re.findall(r"'((?:AUTH|COMMON)_[A-Z0-9_]+)'", web_auth_model))
        web_codes.update(re.findall(r"'(USER_[A-Z0-9_]+)'", web_auth_model))

        for missing in sorted(server_codes - web_codes):
            findings.append(
                Finding(
                    rule="contract-drift-error-code",
                    severity="P1",
                    path="web/src/contracts/api/codes.ts",
                    line=1,
                    value=missing,
                    message=f"server error code '{missing}' is missing from web API contract",
                    action="align cross-end error code definitions",
                    report_only=True,
                )
            )
        for missing in sorted(web_codes - server_codes):
            findings.append(
                Finding(
                    rule="contract-drift-error-code",
                    severity="P1",
                    path="server/internal/contract/errorcode/code.go",
                    line=1,
                    value=missing,
                    message=f"web error code '{missing}' is missing from server error-code mapping",
                    action="align cross-end error code definitions",
                    report_only=True,
                )
            )

    server_messages = read_file("server/internal/contract/message/key.go")
    if server_messages:
        server_keys = set(re.findall(r'"((?:auth|common|user)\.[a-z0-9_.-]+)"', server_messages))
        web_key_usage: set[str] = set()
        for path in tracked_files():
            if not path.startswith("web/src/") or is_skipped_path(path) or path.endswith(".json"):
                continue
            text = read_file(path)
            if not text:
                continue
            web_key_usage.update(re.findall(r'["\']((?:auth|common|user)\.[a-z0-9_.-]+)["\']', text))
        for missing in sorted(key for key in web_key_usage - server_keys if key.startswith(("auth.", "common."))):
            findings.append(
                Finding(
                    rule="contract-drift-message-key",
                    severity="P2",
                    path="server/internal/contract/message/key.go",
                    line=1,
                    value=missing,
                    message=f"message key '{missing}' is used in web/runtime code but missing from server i18n catalog",
                    action="align public message-key contracts",
                    report_only=True,
                )
            )

    return findings


def find_orphan_candidates(files: Iterable[str]) -> list[Finding]:
    findings: list[Finding] = []
    definitions = {
        "audit.record": "server/internal/pluginapi/audit.go",
        "user.read": "server/modules/user/contract/permission.go",
        "user.session.read": "server/modules/user/contract/permission.go",
        "user.session.revoke": "server/modules/user/contract/permission.go",
    }

    repo_text_cache: dict[str, str] = {}
    for path in files:
        text = read_file(path)
        if text:
            repo_text_cache[path] = text

    for value, definition_path in definitions.items():
        references = 0
        for path, text in repo_text_cache.items():
            if path == definition_path:
                continue
            references += text.count(value)
        if references == 0:
            findings.append(
                Finding(
                    rule="orphan-contract-candidate",
                    severity="P3",
                    path=definition_path,
                    line=1,
                    value=value,
                    message=f"contract '{value}' has no non-definition references and may be orphaned",
                    action="confirm whether it is still runtime-reachable before removal",
                    report_only=True,
                )
            )
    return findings


def filter_findings(
    findings: list[Finding],
    baseline_entries: list[dict[str, Any]],
    allowlist_entries: list[dict[str, Any]],
    mode: str,
) -> tuple[list[Finding], list[Finding]]:
    baseline_index = build_suppression_index(baseline_entries)
    allowlist_index = build_suppression_index(allowlist_entries)

    active: list[Finding] = []
    suppressed: list[Finding] = []
    for finding in findings:
        suppression_key = (finding.path, finding.rule, finding.value)
        if suppression_key in allowlist_index or suppression_key in baseline_index:
            suppressed.append(finding)
            continue
        if mode == "changed" and finding.severity in {"P2", "P3"}:
            suppressed.append(finding)
            continue
        active.append(finding)
    return active, suppressed


def summarize(findings: list[Finding]) -> dict[str, int]:
    counts: dict[str, int] = {severity: 0 for severity in SEVERITY_ORDER}
    for finding in findings:
        counts[finding.severity] += 1
    return counts


def print_findings(title: str, findings: list[Finding]) -> None:
    if not findings:
        print(f"{title}: none")
        return

    print(title + ":")
    for finding in findings:
        print(
            f"- [{finding.severity}] {finding.path}:{finding.line} {finding.rule}: "
            f"{finding.message} ({finding.value})"
        )
        print(f"  action: {finding.action}")


def load_files_for_mode(mode: str) -> list[str]:
    if mode == "changed":
        files = staged_or_changed_files()
    elif mode == "ci":
        files = ci_changed_files()
    else:
        files = tracked_files()

    selected = [path for path in files if should_scan_file(path)]
    return sorted(dict.fromkeys(selected))


def collect_findings(files: list[str], mode: str) -> list[Finding]:
    findings: list[Finding] = []
    for path in files:
        text = read_file(path)
        if not text:
            continue
        findings.extend(scan_file(path, text))

    if mode == "report":
        findings.extend(duplicate_string_candidates(files))
        findings.extend(find_orphan_candidates(files))
        findings.extend(drift_candidates())
    return findings


def main() -> int:
    args = parse_args()
    baseline_entries = load_metadata_entries(pathlib.Path(args.baseline))
    allowlist_entries = load_metadata_entries(pathlib.Path(args.allowlist))

    metadata_findings = []
    metadata_findings.extend(validate_metadata_entries(baseline_entries, "baseline"))
    metadata_findings.extend(validate_metadata_entries(allowlist_entries, "allowlist"))

    files = load_files_for_mode(args.mode)
    scan_findings = collect_findings(files, args.mode)
    findings = metadata_findings + scan_findings
    active, suppressed = filter_findings(findings, baseline_entries, allowlist_entries, args.mode)

    summary = {
        "mode": args.mode,
        "scanned_files": len(files),
        "active_counts": summarize(active),
        "suppressed_counts": summarize(suppressed),
        "active_findings": [dataclasses.asdict(item) for item in active],
        "suppressed_findings": [dataclasses.asdict(item) for item in suppressed],
    }

    if args.output_json:
        output_path = pathlib.Path(args.output_json)
        output_path.write_text(json.dumps(summary, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")

    print(f"contract governance scan mode={args.mode} scanned_files={len(files)}")
    print_findings("active findings", active)
    if args.mode == "report":
        print_findings("suppressed findings", suppressed)

    blocking = [
        finding
        for finding in active
        if finding.severity in BLOCKING_SEVERITIES and not finding.report_only
    ]

    if args.mode == "report":
        return 0
    if args.mode == "changed":
        return 1 if any(finding.severity == "P0" for finding in blocking) else 0
    return 1 if blocking else 0


if __name__ == "__main__":
    sys.exit(main())
