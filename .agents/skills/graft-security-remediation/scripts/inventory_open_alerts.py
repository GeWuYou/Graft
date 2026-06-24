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
    """
    解析命令行参数，确定告警来源、类型和输出位置。
    
    Returns:
    	Namespace: 解析后的命令行参数对象。
    """
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
    """
    解析并校验 `--input-json` 参数中的本地输入文件映射。
    
    Parameters:
    	entries (list[str]): 形如 `KIND=PATH` 的参数值列表。
    
    Returns:
    	dict[str, Path]: 按 kind 映射到已展开且存在的输入文件路径。
    """
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
    """
    获取并解析 GitHub API 的分页告警响应。
    
    Parameters:
    	repo (str): 仓库标识，用于错误信息中定位请求来源。
    	endpoint (str): 要调用的 GitHub API 端点。
    
    Returns:
    	list[dict[str, Any]]: 解析后的对象列表。
    """
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
    """
    读取 JSON 数组文件并返回其中的对象元素。
    
    Parameters:
    	path (Path): JSON 文件路径。
    
    Returns:
    	list[dict[str, Any]]: 顶层数组中所有字典元素组成的列表。
    """
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise SystemExit(f"invalid JSON in {path}: {exc}") from exc
    if not isinstance(data, list):
        raise SystemExit(f"expected top-level JSON array in {path}")
    return [item for item in data if isinstance(item, dict)]


def source_label(path: Path) -> str:
    """
    生成文件来源标签。
    
    Parameters:
    	path (Path): 源文件路径。
    
    Returns:
    	str: 以 `file:<父目录名>/<文件名>` 或 `file:<文件名>` 形式表示的来源标签。
    """
    name = path.name
    parent = path.parent.name
    if parent:
        return f"file:{parent}/{name}"
    return f"file:{name}"


def rule_id(alert: dict[str, Any]) -> str | None:
    """
    提取告警对应的规则标识。
    
    Returns:
    	rule_id (str | None): 规则 ID；如果未找到有效的规则标识则为 `None`。
    """
    rule = alert.get("rule")
    if isinstance(rule, dict):
        candidate = rule.get("id") or rule.get("rule_id")
        if isinstance(candidate, str):
            return candidate
    return None


def first_location(alert: dict[str, Any]) -> tuple[str | None, int | None]:
    """
    提取告警最近实例的文件路径和起始行号。
    
    Returns:
    	(tuple[str | None, int | None]): 路径和起始行号；当相关结构缺失或类型不匹配时返回 `(None, None)`。
    """
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
    """
    将代码扫描告警归一化为统一结构。
    
    提取告警编号、规则 ID、严重级别、状态、工具名称、最近位置、消息文本以及相关时间和链接信息，生成标准化的 code-scanning 告警对象。
    
    返回：
    	(dict[str, Any]): 包含 `kind`、`number`、`rule_id`、`severity`、`state`、`tool`、`path`、`line`、`message`、`html_url`、`created_at` 和 `dismissed_at` 的归一化字典。
    """
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
    """
    将 Dependabot 告警归一化为统一结构。
    
    Returns:
        dict[str, Any]: 包含 `kind`、`number`、`package`、`ecosystem`、`manifest_path`、`scope`、`severity`、`state`、`vulnerable_version_range`、`first_patched_version`、`advisory_ids`、`summary`、`html_url`、`created_at` 和 `dismissed_at` 的归一化告警字典。
    """
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
    """
    获取指定类型告警的本地缓存或实时数据。
    
    Parameters:
    	repo (str | None): 用于实时拉取的仓库，格式为 `owner/name`。
    	kind (str): 告警类型。
    	inputs (dict[str, Path]): 按类型映射到本地 JSON 文件路径的输入集合。
    
    Returns:
    	list[dict[str, Any]]: 告警对象列表。
    """
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
    """
    统计告警列表的总数以及按类型和严重性分组的数量。
    
    Returns:
    	(dict[str, Any]): 包含 `total`、`by_kind` 和 `by_severity` 的汇总结果。
    """
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
    """
    生成并输出已归一化的 GitHub 安全告警库存 JSON。
    
    返回：
    	0 表示成功。
    """
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
