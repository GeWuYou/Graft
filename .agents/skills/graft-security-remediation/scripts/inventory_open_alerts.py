#!/usr/bin/env python3
"""Read-only inventory builder for GitHub security alerts."""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Fetch and normalize open GitHub Code Scanning and Dependabot alerts."
    )
    parser.add_argument(
        "--repo",
        help="GitHub repository in owner/name form. Required unless --input-json is used without fetch.",
    )
    parser.add_argument(
        "--kind",
        choices=("code-scanning", "dependabot", "all"),
        default="all",
        help="Alert class to inventory.",
    )
    parser.add_argument(
        "--input-json",
        action="append",
        default=[],
        metavar="KIND=PATH",
        help="Use saved API JSON instead of gh for one kind. Repeat for multiple kinds.",
    )
    parser.add_argument(
        "--output",
        default="-",
        help="Write normalized JSON to this path, or '-' for stdout.",
    )
    parser.add_argument(
        "--pretty",
        action="store_true",
        help="Pretty-print JSON output.",
    )
    return parser.parse_args()


def parse_input_json(entries: list[str]) -> dict[str, Path]:
    parsed: dict[str, Path] = {}
    for entry in entries:
        if "=" not in entry:
            raise SystemExit(f"invalid --input-json value: {entry!r}; expected KIND=PATH")
        kind, raw_path = entry.split("=", 1)
        if kind not in {"code-scanning", "dependabot"}:
            raise SystemExit(f"unsupported input kind: {kind!r}")
        path = Path(raw_path).expanduser()
        if not path.is_file():
            raise SystemExit(f"input file not found: {path}")
        parsed[kind] = path
    return parsed


def run_gh_api(repo: str, endpoint: str) -> list[dict[str, Any]]:
    cmd = [
        "gh",
        "api",
        "--paginate",
        "-H",
        "Accept: application/vnd.github+json",
        endpoint,
    ]
    try:
        proc = subprocess.run(
            cmd,
            check=True,
            capture_output=True,
            text=True,
        )
    except FileNotFoundError as exc:
        raise SystemExit("gh is required for live inventory fetches") from exc
    except subprocess.CalledProcessError as exc:
        message = exc.stderr.strip() or exc.stdout.strip() or str(exc)
        raise SystemExit(f"gh api failed for {repo}: {message}") from exc

    payload = proc.stdout.strip()
    if not payload:
        return []

    decoder = json.JSONDecoder()
    idx = 0
    items: list[dict[str, Any]] = []
    while idx < len(payload):
        while idx < len(payload) and payload[idx].isspace():
            idx += 1
        if idx >= len(payload):
            break
        obj, idx = decoder.raw_decode(payload, idx)
        if isinstance(obj, list):
            items.extend(obj)
        else:
            raise SystemExit(f"expected list payload from gh api for endpoint {endpoint}")
    return items


def load_json_array(path: Path) -> list[dict[str, Any]]:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise SystemExit(f"invalid JSON in {path}: {exc}") from exc
    if not isinstance(data, list):
        raise SystemExit(f"expected top-level JSON array in {path}")
    return [item for item in data if isinstance(item, dict)]


def source_label(path: Path) -> str:
    name = path.name
    parent = path.parent.name
    if parent:
        return f"file:{parent}/{name}"
    return f"file:{name}"


def rule_id(alert: dict[str, Any]) -> str | None:
    rule = alert.get("rule")
    if isinstance(rule, dict):
        candidate = rule.get("id") or rule.get("rule_id")
        if isinstance(candidate, str):
            return candidate
    return None


def first_location(alert: dict[str, Any]) -> tuple[str | None, int | None]:
    instances = alert.get("most_recent_instance")
    if isinstance(instances, dict):
        location = instances.get("location")
        if isinstance(location, dict):
            path = location.get("path")
            start_line = location.get("start_line")
            return (
                path if isinstance(path, str) else None,
                start_line if isinstance(start_line, int) else None,
            )
    return (None, None)


def normalize_code_scanning(alert: dict[str, Any]) -> dict[str, Any]:
    path, line = first_location(alert)
    tool = alert.get("tool")
    tool_name = None
    if isinstance(tool, dict):
        candidate = tool.get("name")
        if isinstance(candidate, str):
            tool_name = candidate

    return {
        "kind": "code-scanning",
        "number": alert.get("number"),
        "rule_id": rule_id(alert),
        "severity": alert.get("rule_severity"),
        "state": alert.get("state"),
        "tool": tool_name,
        "path": path,
        "line": line,
        "message": alert.get("most_recent_instance", {}).get("message", {}).get("text"),
        "html_url": alert.get("html_url"),
        "created_at": alert.get("created_at"),
        "dismissed_at": alert.get("dismissed_at"),
    }


def normalize_dependabot(alert: dict[str, Any]) -> dict[str, Any]:
    dependency = alert.get("dependency") if isinstance(alert.get("dependency"), dict) else {}
    security_advisory = (
        alert.get("security_advisory") if isinstance(alert.get("security_advisory"), dict) else {}
    )
    first_patched = alert.get("security_vulnerability")
    patched_version = None
    vulnerable_range = None
    if isinstance(first_patched, dict):
        patched = first_patched.get("first_patched_version")
        if isinstance(patched, dict):
            version = patched.get("identifier")
            if isinstance(version, str):
                patched_version = version
        vr = first_patched.get("vulnerable_version_range")
        if isinstance(vr, str):
            vulnerable_range = vr

    identifiers = security_advisory.get("identifiers")
    advisory_ids: list[str] = []
    if isinstance(identifiers, list):
        for item in identifiers:
            if not isinstance(item, dict):
                continue
            value = item.get("value")
            if isinstance(value, str):
                advisory_ids.append(value)

    return {
        "kind": "dependabot",
        "number": alert.get("number"),
        "package": dependency.get("package", {}).get("name"),
        "ecosystem": dependency.get("package", {}).get("ecosystem"),
        "manifest_path": dependency.get("manifest_path"),
        "scope": dependency.get("scope"),
        "severity": security_advisory.get("severity"),
        "state": alert.get("state"),
        "vulnerable_version_range": vulnerable_range,
        "first_patched_version": patched_version,
        "advisory_ids": advisory_ids,
        "summary": security_advisory.get("summary"),
        "html_url": alert.get("html_url"),
        "created_at": alert.get("created_at"),
        "dismissed_at": alert.get("dismissed_at"),
    }


def fetch_or_load(repo: str | None, kind: str, inputs: dict[str, Path]) -> list[dict[str, Any]]:
    if kind in inputs:
        return load_json_array(inputs[kind])
    if not repo:
        raise SystemExit(f"--repo is required for live fetch of {kind}")
    if kind == "code-scanning":
        endpoint = f"/repos/{repo}/code-scanning/alerts?state=open&per_page=100"
    elif kind == "dependabot":
        endpoint = f"/repos/{repo}/dependabot/alerts?state=open&per_page=100"
    else:
        raise SystemExit(f"unsupported alert kind: {kind}")
    return run_gh_api(repo, endpoint)


def summarize(alerts: list[dict[str, Any]]) -> dict[str, Any]:
    by_kind: dict[str, int] = {}
    by_severity: dict[str, int] = {}
    for alert in alerts:
        kind = str(alert.get("kind", "unknown"))
        by_kind[kind] = by_kind.get(kind, 0) + 1
        severity = str(alert.get("severity") or "unknown")
        by_severity[severity] = by_severity.get(severity, 0) + 1
    return {
        "total": len(alerts),
        "by_kind": by_kind,
        "by_severity": by_severity,
    }


def main() -> int:
    args = parse_args()
    inputs = parse_input_json(args.input_json)

    requested = ["code-scanning", "dependabot"] if args.kind == "all" else [args.kind]
    alerts: list[dict[str, Any]] = []
    sources: dict[str, str] = {}

    for kind in requested:
        raw = fetch_or_load(args.repo, kind, inputs)
        sources[kind] = source_label(inputs[kind]) if kind in inputs else "gh-api"
        if kind == "code-scanning":
            alerts.extend(normalize_code_scanning(item) for item in raw)
        else:
            alerts.extend(normalize_dependabot(item) for item in raw)

    payload = {
        "repo": args.repo,
        "requested_kind": args.kind,
        "sources": sources,
        "summary": summarize(alerts),
        "alerts": alerts,
    }

    text = json.dumps(payload, indent=2 if args.pretty else None, ensure_ascii=False)
    if args.output == "-":
        sys.stdout.write(text)
        if args.pretty:
            sys.stdout.write("\n")
    else:
        Path(args.output).write_text(text + ("\n" if args.pretty else ""), encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
