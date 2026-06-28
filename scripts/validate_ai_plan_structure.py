#!/usr/bin/env python3
"""Validate the adopted ai-plan governance structure guard."""

from __future__ import annotations

import argparse
import json
import sys
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[1]

REQUIRED_ROOT_FILES = (
    "ai-plan/AGENTS.md",
    "ai-plan/README.md",
    "ai-plan/public/README.md",
    "ai-plan/catalog.json",
)

REQUIRED_TEMPLATE_FILES = (
    "ai-plan/templates/active-topic/README.md",
    "ai-plan/templates/active-topic/startup-prompt.md",
    "ai-plan/templates/active-topic/todos/topic-tracking.md",
    "ai-plan/templates/active-topic/traces/topic-trace.md",
    "ai-plan/templates/adr/ADR-XXX-short-title.md",
)

ACTIVE_INDEX_GUARD_TOPICS = {
    "compose-project-management": "ai-plan/public/compose-project-management/README.md",
}

ARCHIVED_TOPIC_GUARD_TOPICS = {
    "ai-plan-ia-governance": "ai-plan/public/archive/ai-plan-ia-governance/README.md",
}

EXPECTED_CATALOG_INCLUDED_PATHS = (
    "ai-plan/AGENTS.md",
    "ai-plan/README.md",
    "ai-plan/public/README.md",
    "ai-plan/design/decisions/ADR-001-ai-plan-authority-and-metadata-model.md",
    "ai-plan/design/decisions/ADR-002-ai-plan-lifecycle-and-archive-model.md",
    "ai-plan/public/archive/ai-plan-ia-governance/README.md",
    "ai-plan/public/archive/ai-plan-ia-governance/startup-prompt.md",
    "ai-plan/public/archive/ai-plan-ia-governance/todos/ai-plan-ia-governance-tracking.md",
    "ai-plan/public/archive/ai-plan-ia-governance/traces/ai-plan-ia-governance-trace.md",
)

EXPECTED_CATALOG_EXCLUDED_BY_DESIGN = (
    "whole-repository ai-plan inventory",
    "per-document frontmatter retrofit",
    "projection or graph/search derivative files",
)

EXPECTED_CATALOG_AUTHORITY_RULES = {
    "repository_startup": "AGENTS.md",
    "ai_plan_execution": "ai-plan/AGENTS.md",
    "active_topic_router": "ai-plan/public/README.md",
    "topic_metadata_source": "topic-local README/startup-prompt/tracking/trace files",
}

EXPECTED_CATALOG_ENTRIES = (
    {"path": "ai-plan/AGENTS.md", "kind": "governance", "role": "local-execution-truth"},
    {"path": "ai-plan/README.md", "kind": "governance", "role": "directory-readme"},
    {"path": "ai-plan/public/README.md", "kind": "recovery-router", "role": "active-topic-index"},
    {
        "path": "ai-plan/design/decisions/ADR-001-ai-plan-authority-and-metadata-model.md",
        "kind": "adr",
        "role": "metadata-model-authority",
    },
    {
        "path": "ai-plan/design/decisions/ADR-002-ai-plan-lifecycle-and-archive-model.md",
        "kind": "adr",
        "role": "lifecycle-model-authority",
    },
    {
        "path": "ai-plan/public/archive/ai-plan-ia-governance/README.md",
        "kind": "archive-topic-doc",
        "role": "topic-readme",
        "topic": "ai-plan-ia-governance",
        "topic_state": "archived",
    },
    {
        "path": "ai-plan/public/archive/ai-plan-ia-governance/startup-prompt.md",
        "kind": "archive-topic-doc",
        "role": "startup-prompt",
        "topic": "ai-plan-ia-governance",
        "topic_state": "archived",
    },
    {
        "path": "ai-plan/public/archive/ai-plan-ia-governance/todos/ai-plan-ia-governance-tracking.md",
        "kind": "archive-topic-doc",
        "role": "tracking",
        "topic": "ai-plan-ia-governance",
        "topic_state": "archived",
    },
    {
        "path": "ai-plan/public/archive/ai-plan-ia-governance/traces/ai-plan-ia-governance-trace.md",
        "kind": "archive-topic-doc",
        "role": "trace",
        "topic": "ai-plan-ia-governance",
        "topic_state": "archived",
    },
)

ACTIVE_TOPIC_SECTION_RE = re.compile(r"^## Active Topics\n(?P<body>.*?)(?:\n## |\Z)", re.MULTILINE | re.DOTALL)
ACTIVE_TOPIC_ENTRY_RE = re.compile(r"^- `(?P<topic>[^`]+)`\n  - Recovery entry: `(?P<entry>[^`]+)`", re.MULTILINE)


@dataclass(frozen=True)
class Finding:
    path: Path
    message: str

    def format(self, root: Path) -> str:
        try:
            display_path = self.path.relative_to(root)
        except ValueError:
            display_path = self.path
        return f"{display_path}: {self.message}"


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


DISALLOWED_ACTIVE_TOPICS = frozenset(ARCHIVED_TOPIC_GUARD_TOPICS)


def topic_required_files(topic: str, *, archived: bool = False) -> tuple[str, ...]:
    base = f"ai-plan/public/archive/{topic}" if archived else f"ai-plan/public/{topic}"
    return (
        f"{base}/README.md",
        f"{base}/startup-prompt.md",
        f"{base}/todos/{topic}-tracking.md",
        f"{base}/traces/{topic}-trace.md",
    )


def validate_required_files(root: Path) -> list[Finding]:
    findings: list[Finding] = []
    for relative in REQUIRED_ROOT_FILES:
        path = root / relative
        if not path.is_file():
            findings.append(Finding(path, "required ai-plan governance file is missing"))
    return findings


def validate_template_files(root: Path) -> list[Finding]:
    findings: list[Finding] = []
    for relative in REQUIRED_TEMPLATE_FILES:
        path = root / relative
        if not path.is_file():
            findings.append(Finding(path, "required ai-plan template file is missing"))
    return findings


def parse_active_topic_index(text: str) -> tuple[dict[str, str], list[str]] | None:
    match = ACTIVE_TOPIC_SECTION_RE.search(text)
    if match is None:
        return None

    entries: dict[str, str] = {}
    duplicates: list[str] = []
    for item in ACTIVE_TOPIC_ENTRY_RE.finditer(match.group("body")):
        topic = item.group("topic")
        entry = item.group("entry")
        if topic in entries:
            duplicates.append(topic)
            continue
        entries[topic] = entry
    return entries, duplicates


def validate_active_topic_index(root: Path) -> list[Finding]:
    path = root / "ai-plan/public/README.md"
    if not path.is_file():
        return []

    parsed = parse_active_topic_index(read_text(path))
    if parsed is None:
        return [Finding(path, "missing '## Active Topics' section in ai-plan public index")]

    entries, duplicates = parsed
    findings: list[Finding] = []
    for topic in duplicates:
        findings.append(Finding(path, f"duplicate active topic entry {topic!r}"))
    for topic, expected_entry in ACTIVE_INDEX_GUARD_TOPICS.items():
        actual_entry = entries.get(topic)
        if actual_entry is None:
            findings.append(Finding(path, f"guarded active topic {topic!r} is missing from the shared public index"))
            continue
        if actual_entry != expected_entry:
            findings.append(
                Finding(
                    path,
                    f"guarded active topic {topic!r} must point at {expected_entry!r}; found {actual_entry!r}",
                )
            )
    for topic in sorted(DISALLOWED_ACTIVE_TOPICS):
        if topic in entries:
            findings.append(Finding(path, f"archived topic {topic!r} must not remain in the active-topic index"))
    return findings


def validate_topic_contract(root: Path, topic: str, *, archived: bool = False) -> list[Finding]:
    findings: list[Finding] = []
    for relative in topic_required_files(topic, archived=archived):
        path = root / relative
        if not path.is_file():
            topic_kind = "archived topic" if archived else "guarded active topic"
            findings.append(Finding(path, f"{topic_kind} {topic!r} is missing a required recovery file"))
    return findings


def validate_catalog_note(path: Path, note: Any) -> list[Finding]:
    if not isinstance(note, str) or not note.strip():
        return [Finding(path, "coverage.note must be a non-empty string")]
    findings: list[Finding] = []
    for phrase in ("Initial machine index", "whole-repository retrofit mandate"):
        if phrase not in note:
            findings.append(Finding(path, f"coverage.note must mention {phrase!r}"))
    return findings


def validate_catalog_entries(root: Path, path: Path, entries: Any) -> list[Finding]:
    if not isinstance(entries, list):
        return [Finding(path, "entries must be a list")]

    expected_by_path = {entry["path"]: entry for entry in EXPECTED_CATALOG_ENTRIES}
    actual_by_path: dict[str, dict[str, Any]] = {}
    findings: list[Finding] = []

    for entry in entries:
        if not isinstance(entry, dict):
            findings.append(Finding(path, "each catalog entry must be an object"))
            continue

        raw_path = entry.get("path")
        if not isinstance(raw_path, str) or not raw_path:
            findings.append(Finding(path, "each catalog entry must include a non-empty 'path'"))
            continue
        if raw_path in actual_by_path:
            findings.append(Finding(path, f"duplicate catalog entry for {raw_path!r}"))
            continue
        actual_by_path[raw_path] = entry

        entry_path = root / raw_path
        if not entry_path.is_file():
            findings.append(Finding(entry_path, "catalog entry path does not exist"))

    actual_paths = set(actual_by_path)
    expected_paths = set(expected_by_path)
    missing_paths = sorted(expected_paths - actual_paths)
    unexpected_paths = sorted(actual_paths - expected_paths)
    if missing_paths:
        findings.append(Finding(path, f"catalog entries are missing approved paths: {', '.join(missing_paths)}"))
    if unexpected_paths:
        findings.append(Finding(path, f"catalog entries contain unapproved paths: {', '.join(unexpected_paths)}"))

    for relative, expected in expected_by_path.items():
        actual = actual_by_path.get(relative)
        if actual is None:
            continue
        actual_keys = set(actual)
        expected_keys = set(expected)
        extra_keys = sorted(actual_keys - expected_keys)
        missing_keys = sorted(expected_keys - actual_keys)
        if extra_keys:
            findings.append(Finding(path, f"catalog entry {relative!r} adds unapproved keys: {', '.join(extra_keys)}"))
        if missing_keys:
            findings.append(Finding(path, f"catalog entry {relative!r} is missing required keys: {', '.join(missing_keys)}"))
        for key, expected_value in expected.items():
            if actual.get(key) != expected_value:
                findings.append(
                    Finding(
                        path,
                        f"catalog entry {relative!r} field {key!r} must be {expected_value!r}; found {actual.get(key)!r}",
                    )
                )

    return findings


def validate_catalog(root: Path) -> list[Finding]:
    path = root / "ai-plan/catalog.json"
    if not path.is_file():
        return []

    try:
        data = json.loads(read_text(path))
    except json.JSONDecodeError as exc:
        return [Finding(path, f"invalid JSON: {exc}")]

    findings: list[Finding] = []
    if data.get("schema_version") != 1:
        findings.append(Finding(path, "schema_version must be 1"))
    if data.get("catalog_kind") != "ai-plan-machine-index":
        findings.append(Finding(path, "catalog_kind must be 'ai-plan-machine-index'"))

    coverage = data.get("coverage")
    if not isinstance(coverage, dict):
        findings.append(Finding(path, "coverage must be an object"))
    else:
        if coverage.get("mode") != "bounded-governance-set":
            findings.append(Finding(path, "coverage.mode must stay 'bounded-governance-set'"))
        findings.extend(validate_catalog_note(path, coverage.get("note")))

        included_paths = coverage.get("included_paths")
        if not isinstance(included_paths, list) or not all(isinstance(item, str) for item in included_paths):
            findings.append(Finding(path, "coverage.included_paths must be a list of strings"))
        else:
            if included_paths != list(EXPECTED_CATALOG_INCLUDED_PATHS):
                findings.append(Finding(path, "coverage.included_paths must match the approved bounded governance set"))
            for relative in included_paths:
                included_path = root / relative
                if not included_path.is_file():
                    findings.append(Finding(included_path, "catalog coverage path does not exist"))

        excluded_by_design = coverage.get("excluded_by_design")
        if excluded_by_design != list(EXPECTED_CATALOG_EXCLUDED_BY_DESIGN):
            findings.append(Finding(path, "coverage.excluded_by_design must match the approved bounded exclusions"))

    authority_rules = data.get("authority_rules")
    if authority_rules != EXPECTED_CATALOG_AUTHORITY_RULES:
        findings.append(Finding(path, "authority_rules must match the approved ai-plan authority map"))

    findings.extend(validate_catalog_entries(root, path, data.get("entries")))
    return findings


def run_validation(root: Path) -> list[Finding]:
    findings: list[Finding] = []
    findings.extend(validate_required_files(root))
    findings.extend(validate_template_files(root))
    findings.extend(validate_active_topic_index(root))
    for topic in ACTIVE_INDEX_GUARD_TOPICS:
        findings.extend(validate_topic_contract(root, topic))
    for topic in ARCHIVED_TOPIC_GUARD_TOPICS:
        findings.extend(validate_topic_contract(root, topic, archived=True))
    findings.extend(validate_catalog(root))
    return findings


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.parse_args(argv)

    findings = run_validation(REPO_ROOT)
    if findings:
        print("ai-plan structure guard: failed", file=sys.stderr)
        for finding in findings:
            print(f"- {finding.format(REPO_ROOT)}", file=sys.stderr)
        return 1

    print(
        "ai-plan structure guard: ok "
        f"({len(ACTIVE_INDEX_GUARD_TOPICS)} active guard topics, {len(ARCHIVED_TOPIC_GUARD_TOPICS)} archived guard topics)"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
