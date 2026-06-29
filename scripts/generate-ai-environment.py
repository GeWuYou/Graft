#!/usr/bin/env python3

from __future__ import annotations

import subprocess
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

SCRIPT_DIR = Path(__file__).resolve().parent
ROOT_DIR = Path(
    subprocess.run(
        ["git", "-C", str(SCRIPT_DIR.parent), "rev-parse", "--show-toplevel"],
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()
)
RAW_PATH = ROOT_DIR / ".ai" / "environment" / "tools.raw.yaml"
AI_PATH = ROOT_DIR / ".ai" / "environment" / "tools.ai.yaml"


def parse_scalar(value: str) -> Any:
    if value == "true":
        return True
    if value == "false":
        return False
    if value.startswith('"') and value.endswith('"'):
        return value[1:-1]
    return value


def parse_simple_yaml(path: Path) -> dict[str, Any]:
    root: dict[str, Any] = {}
    stack: list[tuple[int, dict[str, Any]]] = [(-1, root)]

    for raw_line in path.read_text(encoding="utf-8").splitlines():
        if not raw_line.strip():
            continue
        if raw_line.lstrip().startswith("#"):
            continue

        indent = len(raw_line) - len(raw_line.lstrip(" "))
        key, _, tail = raw_line.strip().partition(":")

        while len(stack) > 1 and indent <= stack[-1][0]:
            stack.pop()

        current = stack[-1][1]
        value = tail.strip()

        if value == "":
            child: dict[str, Any] = {}
            current[key] = child
            stack.append((indent, child))
            continue

        current[key] = parse_scalar(value)

    return root


def bool_value(data: dict[str, Any], *keys: str) -> bool:
    current: Any = data
    for key in keys:
        current = current[key]
    return bool(current)


def string_value(data: dict[str, Any], *keys: str) -> str:
    current: Any = data
    for key in keys:
        current = current[key]
    return str(current)


def choose(preferred: str | None, fallback: str | None) -> str:
    if preferred:
        return preferred
    return fallback or "unavailable"


def available_tool(raw: dict[str, Any], section: str, name: str) -> bool:
    return bool_value(raw, section, name, "installed")


def optional_bool_value(data: dict[str, Any], default: bool, *keys: str) -> bool:
    current: Any = data
    for key in keys:
        if not isinstance(current, dict) or key not in current:
            return default
        current = current[key]
    return bool(current)


def optional_string_value(data: dict[str, Any], default: str, *keys: str) -> str:
    current: Any = data
    for key in keys:
        if not isinstance(current, dict) or key not in current:
            return default
        current = current[key]
    return str(current)


def select_tool(use_for: str, preferred: str | None, fallback: str | None) -> dict[str, str]:
    return {
        "preferred": choose(preferred, fallback),
        "fallback": fallback or "unavailable",
        "use_for": use_for,
    }


def build_ai_inventory(raw: dict[str, Any]) -> dict[str, Any]:
    """
    根据原始环境描述构建用于生成 AI 环境清单的数据结构。
    
    Parameters:
    	raw (dict[str, Any]): 解析后的原始环境数据。
    
    Returns:
    	dict[str, Any]: 用于生成 `.ai/environment/tools.ai.yaml` 的完整清单结构。
    """
    has_go = available_tool(raw, "required_runtimes", "go")
    has_python = available_tool(raw, "required_runtimes", "python3")
    has_node = available_tool(raw, "required_runtimes", "node")
    has_npm = available_tool(raw, "required_runtimes", "npm")
    has_bun = available_tool(raw, "required_runtimes", "bun")
    has_rg = available_tool(raw, "required_tools", "rg")
    has_jq = available_tool(raw, "required_tools", "jq")
    has_bash = available_tool(raw, "required_tools", "bash")
    has_docker = available_tool(raw, "project_tools", "docker")
    has_gh = optional_bool_value(raw, False, "project_tools", "gh", "installed")
    has_gh_authenticated = optional_bool_value(raw, False, "project_tools", "gh", "authenticated")
    has_eff_u_code = optional_bool_value(raw, False, "ai_tools", "eff_u_code", "installed")
    eff_u_code_path = optional_string_value(raw, "", "ai_tools", "eff_u_code", "path")
    eff_u_code_version = optional_string_value(raw, "not-installed", "ai_tools", "eff_u_code", "version")
    has_headroom = optional_bool_value(raw, False, "ai_tools", "headroom", "installed")
    headroom_path = optional_string_value(raw, "", "ai_tools", "headroom", "path")
    headroom_mcp_command = optional_string_value(raw, "", "ai_tools", "headroom", "mcp_command")
    has_mcp_codegraph = optional_bool_value(raw, False, "mcp_servers", "codegraph", "configured")
    has_mcp_tdesign = optional_bool_value(raw, False, "mcp_servers", "tdesign", "configured")
    has_mcp_context7 = optional_bool_value(raw, False, "mcp_servers", "context7", "configured")
    has_mcp_github = optional_bool_value(raw, False, "mcp_servers", "github", "configured")
    has_mcp_playwright = optional_bool_value(raw, False, "mcp_servers", "playwright", "configured")
    has_mcp_headroom = optional_bool_value(raw, False, "mcp_servers", "headroom", "configured")
    has_python_venv = optional_bool_value(raw, False, "python_environment", "venv", "available")
    has_project_venv = optional_bool_value(raw, False, "python_environment", "project_venv", "present")
    has_playwright = optional_bool_value(raw, False, "python_packages", "playwright", "installed")
    has_playwright_browsers = optional_bool_value(raw, False, "python_environment", "playwright_browsers", "present")
    has_playwright_system_deps = optional_bool_value(
        raw, False, "python_environment", "playwright_system_deps", "available"
    )
    has_ai_browser = (
        has_python
        and has_python_venv
        and has_project_venv
        and has_playwright
        and has_playwright_browsers
        and has_playwright_system_deps
    )

    server_scaffolded = bool_value(raw, "repository", "server_go_mod", "present")
    web_scaffolded = bool_value(raw, "repository", "web_package_json", "present")
    web_bun_lock = bool_value(raw, "repository", "web_bun_lock", "present")

    search = select_tool(
        use_for="Repository text search.",
        preferred="rg" if has_rg else None,
        fallback="grep",
    )
    json = select_tool(
        use_for="Inspecting or transforming JSON command output.",
        preferred="jq" if has_jq else None,
        fallback="python3" if has_python else None,
    )
    scripting = select_tool(
        use_for="Non-trivial local automation and helper scripts.",
        preferred="python3" if has_python else None,
        fallback="bash" if has_bash else None,
    )
    shell = select_tool(
        use_for="Repository shell scripts and command execution.",
        preferred="bash" if has_bash else None,
        fallback="sh",
    )
    web_package_manager = select_tool(
        use_for="Installing and validating the web toolchain.",
        preferred=(
            "bun" if web_bun_lock and has_bun else None if web_bun_lock else "npm" if has_npm else None
        ),
        fallback="npm" if has_npm and not web_bun_lock else None,
    )
    server_build_and_test = select_tool(
        use_for="Server build and test workflows once server/go.mod exists.",
        preferred="go" if has_go and server_scaffolded else None,
        fallback=None,
    )
    github_cli = select_tool(
        use_for="GitHub API authentication and PR-related local automation.",
        preferred="gh" if has_gh and has_gh_authenticated else None,
        fallback="environment token",
    )
    ai_browser = select_tool(
        use_for="AI-assisted local web UI screenshots and simple browser interactions.",
        preferred="graft-web-browser-agent" if has_ai_browser else None,
        fallback=None,
    )
    headroom_command = display_path(headroom_path)
    headroom_mcp_display = display_command_path(headroom_mcp_command)
    context_compression = {
        "preferred": "headroom mcp" if has_mcp_headroom else "unavailable",
        "fallback": "unavailable",
        "use_for": "Optional local MCP compression, retrieval, and stats for AI-assisted context management.",
        "mcp_command": headroom_mcp_display if has_headroom else "unavailable",
        "allowed_controlled_local": ["headroom memory", "headroom learn"],
        "controlled_local_dirs": [".ai/headroom/memory", ".ai/headroom/learn"],
        "disallowed_by_default": [
            "rtk instruction injection",
            "automatic instructions write",
        ],
    }

    if bool_value(raw, "platform", "wsl"):
        platform_family = "wsl-linux"
    else:
        platform_family = string_value(raw, "platform", "os").lower()

    return {
        "schema_version": 1,
        "generated_at_utc": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "generated_from": ".ai/environment/tools.raw.yaml",
        "generator": "scripts/generate-ai-environment.py",
        "platform": {
            "family": platform_family,
            "os": string_value(raw, "platform", "os"),
            "distro": string_value(raw, "platform", "distro"),
            "shell": string_value(raw, "platform", "shell"),
        },
        "repository": {
            "server_scaffolded": server_scaffolded,
            "web_scaffolded": web_scaffolded,
            "web_bun_lock": web_bun_lock,
        },
        "capabilities": {
            "go": has_go,
            "python": has_python,
            "node": has_node,
            "npm": has_npm,
            "bun": has_bun,
            "docker": has_docker,
            "gh": has_gh,
            "gh_authenticated": has_gh_authenticated,
            "ai_headroom": has_headroom,
            "mcp_codegraph": has_mcp_codegraph,
            "mcp_tdesign": has_mcp_tdesign,
            "mcp_context7": has_mcp_context7,
            "mcp_github": has_mcp_github,
            "mcp_playwright": has_mcp_playwright,
            "mcp_headroom": has_mcp_headroom,
            "fast_search": has_rg,
            "json_cli": has_jq,
            "ai_browser": has_ai_browser,
            "ai_headroom_mcp": has_mcp_headroom,
            "playwright_python": has_playwright,
            "playwright_browsers": has_playwright_browsers,
            "playwright_system_deps": has_playwright_system_deps,
            "server_scaffolded": server_scaffolded,
            "web_scaffolded": web_scaffolded,
        },
        "tool_selection": {
            "search": search,
            "json": json,
            "shell": shell,
            "github_cli": github_cli,
            "scripting": scripting,
            "ai_browser": ai_browser,
            "context_compression": context_compression,
            "web_package_manager": web_package_manager,
            "server_build_and_test": server_build_and_test,
        },
        "ai_tools": {
            "eff_u_code": {
                "installed": has_eff_u_code,
                "risk_level": "L1",
                "use_for": "Optional developer-local code quality hotspot inspection and raw JSON input for a repository-owned quality policy evaluator outside the formal validation flow.",
                "default_command": "bun run quality:eff-u-code --",
                "gate_entrypoint": "bun run quality:eff-u-code:gate --",
                "score_entrypoint": "bun run quality:eff-u-code:score:changed",
                "version": eff_u_code_version,
                "guardrail": "Keep eff-u-code as a local helper and raw JSON source; the repository root package.json wrapper and repository-owned evaluator are allowed, but do not add eff-u-code directly to server/go.mod, web/package.json, runtime scripts, deployment flows, or completion gates, and do not use the upstream total score as the gate contract.",
            },
            "headroom": {
                "installed": has_headroom,
                "risk_level": "L1",
                "use_for": "Optional local user-level MCP-based AI context compression tool.",
                "default_command": headroom_mcp_display if has_headroom else "unavailable",
                "memory_status": "controlled-local-only",
                "memory_dir": ".ai/headroom/memory",
                "learn_status": "controlled-local-only",
                "learn_dir": ".ai/headroom/learn",
                "instructions_auto_write": "disabled",
                "guardrail": "Use Headroom through Codex MCP by default; RTK injection and automatic instructions writes are disallowed by default.",
            },
        },
        "mcp_servers": {
            "codegraph": {
                "configured": has_mcp_codegraph,
                "risk_level": "L0",
                "use_for": "Local source navigation and impact discovery.",
            },
            "tdesign": {
                "configured": has_mcp_tdesign,
                "risk_level": "L0",
                "use_for": "TDesign Vue Next component API, DOM, and changelog lookup.",
            },
            "context7": {
                "configured": has_mcp_context7,
                "risk_level": "L1",
                "use_for": "Current third-party library documentation lookup.",
            },
            "github": {
                "configured": has_mcp_github,
                "risk_level": "L1",
                "access_policy": "read-only default; write actions require repository skill ownership.",
                "use_for": "GitHub PR, Actions, and repository context lookup for review workflows.",
            },
            "playwright": {
                "configured": has_mcp_playwright,
                "risk_level": "L1",
                "use_for": "Exploratory browser interaction before graft-web-browser-agent evidence capture.",
            },
            "headroom": {
                "configured": has_mcp_headroom,
                "risk_level": "L1",
                "use_for": "On-demand local compression, retrieval, and stats for AI-assisted context management.",
                "access_policy": "Compression, retrieval, and stats only by default; memory and learn are controlled-local-only under .ai/headroom/**, and automatic instructions writes are disabled.",
            },
        },
        "python": {
            "available": has_python,
            "venv_available": has_python_venv,
            "project_venv_present": has_project_venv,
            "playwright_browsers_present": has_playwright_browsers,
            "playwright_system_deps_available": has_playwright_system_deps,
            "helper_packages": {
                "requests": bool_value(raw, "python_packages", "requests", "installed"),
                "rich": bool_value(raw, "python_packages", "rich", "installed"),
                "openai": bool_value(raw, "python_packages", "openai", "installed"),
                "tiktoken": bool_value(raw, "python_packages", "tiktoken", "installed"),
                "pydantic": bool_value(raw, "python_packages", "pydantic", "installed"),
                "pytest": bool_value(raw, "python_packages", "pytest", "installed"),
                "pyyaml": bool_value(raw, "python_packages", "pyyaml", "installed"),
                "playwright": has_playwright,
            },
        },
        "preferences": {
            "prefer_project_listed_tools": True,
            "prefer_python_for_non_trivial_automation": has_python,
            "avoid_unlisted_system_tools": True,
            "avoid_unscaffolded_server_commands": not server_scaffolded,
        },
        "rules": [
            "Use rg instead of grep for repository search when rg is available.",
            "Use jq for JSON inspection; fall back to python3 if jq is unavailable.",
            "Prefer gh-authenticated access for GitHub PR automation when gh is installed and logged in.",
            "Prefer python3 over complex bash for non-trivial scripting when python3 is available.",
            "Prefer bun for web installs when web/bun.lock exists; otherwise fall back to npm.",
            "Keep eff-u-code as an optional developer-local helper and raw JSON source; when quality gating is needed, gate through a repository-owned evaluator rather than the upstream total score.",
            "Use graft-web-browser-agent for AI-assisted frontend screenshots and interactions; do not treat it as the web validation entrypoint.",
            "Use Playwright MCP only as a browser exploration aid; keep graft-web-browser-agent artifacts as auditable evidence.",
            "Use Context7 MCP for current third-party library documentation when repository truth is insufficient.",
            "Use GitHub MCP as a read-only PR and Actions context source unless a repository skill explicitly owns a write path.",
            "Do not assume server build or test commands are available when server/go.mod is missing.",
            "Do not assume unrelated system tools are part of the supported project environment.",
        ],
    }


def display_path(raw_path: str) -> str:
    if not raw_path:
        return "headroom"
    path = Path(raw_path)
    try:
        return str(path.relative_to(ROOT_DIR))
    except ValueError:
        return raw_path


def display_command_path(raw_command: str) -> str:
    if not raw_command:
        return "headroom mcp serve"
    parts = raw_command.split(" ", 1)
    command = display_path(parts[0])
    if len(parts) == 1:
        return command
    return f"{command} {parts[1]}"


def emit_yaml(value: Any, indent: int = 0) -> list[str]:
    prefix = " " * indent

    if isinstance(value, dict):
        lines: list[str] = []
        for key, nested in value.items():
            if isinstance(nested, (dict, list)):
                lines.append(f"{prefix}{key}:")
                lines.extend(emit_yaml(nested, indent + 2))
            else:
                lines.append(f"{prefix}{key}: {format_scalar(nested)}")
        return lines

    if isinstance(value, list):
        lines = []
        for item in value:
            if isinstance(item, (dict, list)):
                lines.append(f"{prefix}-")
                lines.extend(emit_yaml(item, indent + 2))
            else:
                lines.append(f"{prefix}- {format_scalar(item)}")
        return lines

    return [f"{prefix}{format_scalar(value)}"]


def format_scalar(value: Any) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, int):
        return str(value)
    text = str(value).replace('"', '\\"')
    return f'"{text}"'


def main() -> None:
    raw = parse_simple_yaml(RAW_PATH)
    ai_inventory = build_ai_inventory(raw)
    AI_PATH.parent.mkdir(parents=True, exist_ok=True)
    AI_PATH.write_text("\n".join(emit_yaml(ai_inventory)) + "\n", encoding="utf-8")
    print(f"Wrote {AI_PATH}")


if __name__ == "__main__":
    main()
