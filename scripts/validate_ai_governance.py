#!/usr/bin/env python3
"""Validate repository AI governance documents and skills stay aligned."""

from __future__ import annotations

import argparse
import ast
import importlib.util
import ipaddress
import json
import re
import shutil
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from urllib.parse import unquote, urlsplit


REPO_ROOT = Path(__file__).resolve().parents[1]
SKILLS_DIR = REPO_ROOT / ".agents" / "skills"
AI_TOOLING_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "ai" / "AI工具与MCP接入治理规范.md"
CODEGRAPH_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "ai" / "CodeGraph-MCP-辅助开发规范.md"
TDESIGN_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "frontend" / "TDesign-MCP-辅助开发规范.md"
TOOLS_AI = REPO_ROOT / ".ai" / "environment" / "tools.ai.yaml"
GITIGNORE = REPO_ROOT / ".gitignore"
AGENTS = REPO_ROOT / "AGENTS.md"
AI_PLAN_AGENTS = REPO_ROOT / "ai-plan" / "AGENTS.md"
AI_TASK_TRACKING_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "ai" / "AI任务追踪与恢复设计.md"
WEB_AGENTS = REPO_ROOT / "web" / "AGENTS.md"
PR_REVIEW_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-pr-review" / "SKILL.md"
PR_REMEDIATION_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-pr-remediation" / "SKILL.md"
PR_REVIEW_HELPER = PR_REVIEW_SKILL.parent / "scripts" / "fetch_current_pr_review.py"
PR_REVIEW_COMMANDS = PR_REVIEW_SKILL.parent / "references" / "commands.md"
WORKTREE_MANAGER_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-worktree-manager" / "SKILL.md"
PUSH_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-push" / "SKILL.md"
COMMIT_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-commit" / "SKILL.md"
SHARED_ASSET_VALIDATOR = REPO_ROOT / "scripts" / "validate_shared_asset_registries.py"
BACKEND_QUERY_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "backend" / "后端查询与数据库访问治理规范.md"
SERVER_API_GOVERNANCE_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "backend" / "服务端API边界与兼容治理规范.md"
BACKEND_SECURITY_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "backend" / "后端安全与信任边界治理规范.md"
BACKEND_TEST_MAINTAIN_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "backend" / "后端测试与可维护性治理规范.md"
AI_CODE_REVIEW_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "ai" / "AI代码生成与Review规范.md"
SERVER_AGENTS = REPO_ROOT / "server" / "AGENTS.md"

FRONTMATTER_RE = re.compile(r"\A---\n(?P<body>.*?)\n---\n", re.DOTALL)
SKILL_NAME_RE = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
FRONTMATTER_KEY_RE = re.compile(r"^[a-z][a-z0-9_-]*$")
YAML_PLAIN_FORBIDDEN_INITIAL = frozenset("-?:,[]{}#&*!|>'\"%@`")
YAML_IMPLICIT_NON_STRING_RE = re.compile(
    r"(?ix)(?:"
    r"~|null|true|false|yes|no|on|off|"
    r"[-+]?0(?:b[01_]+|o[0-7_]+|x[0-9a-f_]+)|"
    r"[-+]?(?:0|[1-9][0-9_]*)(?:\.[0-9_]*)?(?:e[-+]?[0-9]+)?|"
    r"[-+]?\.(?:inf|nan)|"
    r"[0-9][0-9_]*(?::[0-5]?[0-9])+|"
    r"[0-9]{4}-[0-9]{2}-[0-9]{2}(?:(?:[Tt]|[ \t]+)[0-9]{2}:[0-9]{2}:[0-9]{2}.*)?"
    r")"
)
MARKDOWN_LINK_RE = re.compile(r"(?<!!)\[[^\]\n]+\]\((?P<target>[^)\n]+)\)")
ALLOWED_MISSING_PRIVATE_REFERENCES = {".ai/private/graft-browser-targets.yaml"}
PRIVATE_IPV4_RE = re.compile(r"(?<![\w.])(?:10(?:\.\d{1,3}){3}|172(?:\.\d{1,3}){3}|192\.168(?:\.\d{1,3}){2})(?![\w.])")
RFC1918_NETWORKS = tuple(
    ipaddress.ip_network(cidr) for cidr in ("10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16")
)
FORBIDDEN_ACTIVE_SKILL_PATTERNS = (
    ("historical server plugin path", re.compile(r"server/plugins(?:/|\b)")),
    ("historical plugin API boundary", re.compile(r"(?:server/)?internal/pluginapi(?:/|\b)")),
    ("retired skill name", re.compile(r"\bgraft-plugin-scaffold\b")),
    ("historical backend plugin wording", re.compile(r"\bbackend plugin\b", re.IGNORECASE)),
)
HEADROOM_RTK_START = "<!-- headroom:rtk-instructions -->"
HEADROOM_RTK_END = "<!-- /headroom:rtk-instructions -->"
GOVERNED_GUIDANCE_PREFIXES = (".agents/", "plugins/", "ai-plan/", ".ai/")
FORBIDDEN_PERSONAL_GUIDANCE_PATTERNS = (
    (
        "personal absolute tooling path",
        re.compile(
            r"(?<![\w])(?:~|/(?:root|home/[A-Za-z0-9._-]+|Users/[A-Za-z0-9._-]+)|[A-Za-z]:[\\/]+Users[\\/][A-Za-z0-9._-]+)[\\/]"
            r"(?:\.codex|\.claude)[\\/]skills(?:[\\/]|$)",
            re.IGNORECASE,
        ),
    ),
    (
        "device-level command",
        re.compile(
            r"(?im)(?<![\w-])(?:shutdown(?:\.exe)?\s+/[str]\b|"
            r"(?:poweroff|reboot|halt)(?:\s|$)|systemctl\s+(?:poweroff|reboot|halt)\b|"
            r"Stop-Computer\b)",
        ),
    ),
    (
        "personal skill link",
        re.compile(r"(?<![\w-])\$shutdown-after-completion\b", re.IGNORECASE),
    ),
)


@dataclass(frozen=True)
class Finding:
    path: Path
    message: str

    def format(self) -> str:
        try:
            display_path = self.path.relative_to(REPO_ROOT)
        except ValueError:
            display_path = self.path
        return f"{display_path}: {self.message}"


def tracked_files() -> set[str]:
    git_path = shutil.which("git")
    if git_path is None:
        raise RuntimeError("git executable was not found on PATH; cannot inspect tracked governance files")
    completed = subprocess.run(
        [git_path, "-c", "core.quotePath=false", "ls-files"],
        cwd=REPO_ROOT,
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    return {line for line in completed.stdout.splitlines() if line}


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


class FrontmatterError(ValueError):
    """Raised when repository skill frontmatter is not strict flat YAML."""


def _parse_quoted_scalar(raw_value: str) -> str:
    try:
        value = json.loads(raw_value)
    except json.JSONDecodeError as exc:
        raise FrontmatterError("quoted scalars must use valid JSON-compatible double quotes") from exc
    if not isinstance(value, str):
        raise FrontmatterError("frontmatter values must be strings")
    return value


def _parse_plain_scalar(value: str, line_no: int) -> str:
    """Accept a conservative plain-string subset that is unambiguous YAML."""
    if value[0] in YAML_PLAIN_FORBIDDEN_INITIAL:
        raise FrontmatterError(f"line {line_no} starts with a reserved YAML indicator")
    if ": " in value or value.endswith(":"):
        raise FrontmatterError(f"line {line_no} contains an unquoted colon-space scalar")
    if re.search(r"\s#", value):
        raise FrontmatterError(f"line {line_no} contains an unquoted YAML comment")
    if any(ord(character) < 0x20 or ord(character) == 0x7F for character in value):
        raise FrontmatterError(f"line {line_no} contains a control character")
    if YAML_IMPLICIT_NON_STRING_RE.fullmatch(value):
        raise FrontmatterError(f"line {line_no} would resolve to a non-string YAML scalar")
    return value


def parse_frontmatter(text: str) -> dict[str, str] | None:
    """Parse the intentionally flat skill frontmatter without a YAML dependency."""
    match = FRONTMATTER_RE.match(text)
    if match is None:
        return None

    values: dict[str, str] = {}
    for line_no, raw_line in enumerate(match.group("body").splitlines(), start=2):
        if raw_line.startswith((" ", "\t")):
            raise FrontmatterError(f"line {line_no} must be a top-level scalar")
        line = raw_line.rstrip()
        if not line or line.startswith("#"):
            continue
        key, sep, raw_value = line.partition(":")
        if not sep:
            raise FrontmatterError(f"line {line_no} is not a key/value pair")
        if not FRONTMATTER_KEY_RE.fullmatch(key):
            raise FrontmatterError(f"line {line_no} has an invalid key")
        if key in values:
            raise FrontmatterError(f"line {line_no} duplicates key {key!r}")
        value = raw_value.strip()
        if not value:
            raise FrontmatterError(f"line {line_no} has an empty value")
        if value[0] == '"':
            if len(value) < 2 or value[-1] != '"':
                raise FrontmatterError(f"line {line_no} has an unterminated quoted scalar")
            value = _parse_quoted_scalar(value)
        else:
            value = _parse_plain_scalar(value, line_no)
        values[key] = value
    return values


def validate_required_files() -> list[Finding]:
    findings: list[Finding] = []
    for path in (AGENTS, TOOLS_AI, AI_TOOLING_DOC, CODEGRAPH_DOC, TDESIGN_DOC, GITIGNORE):
        if not path.is_file():
            findings.append(Finding(path, "required AI governance file is missing"))
    return findings


def validate_gitignore() -> list[Finding]:
    if not GITIGNORE.is_file():
        return []
    text = read_text(GITIGNORE)
    findings: list[Finding] = []
    for pattern in (
        ".codegraph/",
        ".headroom/",
        ".ai/headroom/",
        ".ai/private/",
        ".ai/venv/",
        ".ai/ms-playwright/",
        ".ai/artifacts/browser/",
    ):
        if pattern not in text:
            findings.append(Finding(GITIGNORE, f"missing ignored local AI artifact pattern {pattern!r}"))
    return findings


def contains_headroom_rtk_injection(text: str) -> bool:
    return HEADROOM_RTK_START in text or HEADROOM_RTK_END in text


def contains_project_rtk_prefix_rule(text: str) -> bool:
    return "always prefix with `rtk`" in text or "always prefix with rtk" in text


def missing_exact_terms(text: str, path: Path, label: str, terms: tuple[str, ...]) -> list[Finding]:
    return [Finding(path, f"missing {label} term {term!r}") for term in terms if term not in text]


def missing_concepts(
    text: str,
    path: Path,
    label: str,
    concepts: tuple[tuple[str, tuple[str, ...]], ...],
) -> list[Finding]:
    findings: list[Finding] = []
    for concept, patterns in concepts:
        if not all(re.search(pattern, text, re.IGNORECASE | re.DOTALL) for pattern in patterns):
            findings.append(Finding(path, f"missing {label} concept {concept!r}"))
    return findings


def validate_ai_tooling_doc() -> list[Finding]:
    """
    校验 AI 工具治理文档是否包含必需的治理术语与约束。
    
    Returns:
        list[Finding]: 缺失必需内容或包含禁用表述时的发现列表。
    """
    if not AI_TOOLING_DOC.is_file():
        return []
    text = read_text(AI_TOOLING_DOC)
    findings: list[Finding] = []
    exact_terms = (
        "graft validate backend",
        "bun run check",
        "codegraph",
        "tdesign",
        "context7",
        "github",
        "playwright",
        "@upstash/context7-mcp",
        "ghcr.io/github/github-mcp-server",
        "@playwright/mcp",
        "headroom",
        "optional / local / user-level / MCP-based AI context compression tool",
        "codex mcp add headroom",
        "headroom mcp serve",
        "headroom_compress",
        "headroom_retrieve",
        "headroom_stats",
        "headroom learn",
        ".ai/headroom/memory",
        ".ai/headroom/learn",
        "graft-ai-plan-governance",
        "ai-plan/public/**",
        "Codex `instructions.md`",
        "CLAUDE.md",
        "GEMINI.md",
        "AGENTS.md",
        "memory",
        "postgres",
        "AI tooling evidence",
    )
    findings.extend(missing_exact_terms(text, AI_TOOLING_DOC, "AI tooling governance", exact_terms))
    findings.extend(
        missing_concepts(
            text,
            AI_TOOLING_DOC,
            "AI tooling governance",
            (
                (
                    "headroom optional local user-level context compression",
                    (
                        r"headroom",
                        r"optional|可选",
                        r"local|本地",
                        r"user-level|用户级",
                        r"MCP",
                        r"context compression|上下文.*压缩",
                    ),
                ),
                ("RTK prefix rule is forbidden", (r"不得|must not", r"always prefix with\s+`?rtk`?")),
                ("raw output retained for precise validation", (r"raw output|原始命令输出", r"验证|validation|调试|debug")),
                ("manual confirmation boundary", (r"人工确认|manual confirmation|manual review|人工 review",)),
                ("runtime dependency guardrail", (r"server/go\.mod", r"web/package\.json", r"CI", r"runtime|运行时")),
                ("hidden recovery truth guardrail", (r"隐藏恢复真值|hidden recovery",)),
            ),
        )
    )
    for disallowed in ("headroom wrap codex", "headroom proxy"):
        if disallowed in text:
            findings.append(Finding(AI_TOOLING_DOC, f"Headroom governance should keep only MCP entry content, found {disallowed!r}"))
    if "codex mcp add github -- wsl.exe -- bash -lc" not in text:
        findings.append(Finding(AI_TOOLING_DOC, "GitHub MCP guidance must use the default portable WSL launcher"))
    if re.search(r"\bwsl\.exe\s+(?:-d|--distribution)\s+\S+", text):
        findings.append(Finding(AI_TOOLING_DOC, "shared GitHub MCP guidance must not hard-code a WSL distro"))
    return findings


def validate_environment_inventory() -> list[Finding]:
    """
    校验本地 AI 环境清单是否记录了受控的工具配置、Headroom/MCP 选项和目录约束。
    
    Returns:
        list[Finding]: 发现的治理缺失项列表；清单文件不存在时返回空列表。
    """
    if not TOOLS_AI.is_file():
        return []
    text = read_text(TOOLS_AI)
    findings: list[Finding] = []
    exact_terms = (
        ".ai/headroom/memory",
        ".ai/headroom/learn",
        "rtk instruction injection",
        "automatic instructions write",
        "instructions_auto_write: \"disabled\"",
    )
    findings.extend(missing_exact_terms(text, TOOLS_AI, "AI environment inventory", exact_terms))
    findings.extend(
        missing_concepts(
            text,
            TOOLS_AI,
            "AI environment inventory",
            (
                ("Headroom CLI and MCP capabilities", (r"ai_headroom:\s*(true|false)", r"ai_headroom_mcp:\s*(true|false)")),
                ("context compression tool selection", (r"context_compression:", r"preferred:\s+\"(headroom mcp|unavailable)\"")),
                ("controlled local Headroom directories", (r"controlled_local_dirs:", r"\.ai/headroom/memory", r"\.ai/headroom/learn")),
                ("disallowed Headroom automation", (r"disallowed_by_default:", r"rtk instruction injection", r"automatic instructions write")),
                ("Headroom AI tool record", (r"ai_tools:", r"headroom:", r"instructions_auto_write:\s+\"disabled\"")),
                ("adopted and pilot MCP server records", (r"mcp_servers:", r"codegraph:", r"tdesign:", r"context7:", r"github:", r"playwright:", r"headroom:")),
            ),
        )
    )
    mcp_section = text[text.find("mcp_servers:") :]
    for server_name in ("codegraph", "tdesign", "context7", "github", "playwright"):
        record_match = re.search(
            rf"(?ms)^  {re.escape(server_name)}:\n(?P<record>(?:    .*\n)+)",
            mcp_section,
        )
        if record_match is None or 'adoption: "adopted"' not in record_match.group("record"):
            findings.append(Finding(TOOLS_AI, f"formally adopted MCP server {server_name!r} is missing adoption status"))
    headroom_match = re.search(
        r"(?ms)^  headroom:\n(?P<record>(?:    .*\n)+)",
        mcp_section,
    )
    if headroom_match is None or 'adoption: "optional-pilot"' not in headroom_match.group("record"):
        findings.append(Finding(TOOLS_AI, "Headroom MCP server is missing optional-pilot adoption status"))
    for disallowed in ("headroom wrap codex", "headroom proxy", "wrapper_available:", "proxy_available:"):
        if disallowed in text:
            findings.append(Finding(TOOLS_AI, f"AI environment inventory should keep only MCP entry content, found {disallowed!r}"))
    return findings


def validate_skill_frontmatter(skill_md: Path) -> list[Finding]:
    text = read_text(skill_md)
    findings: list[Finding] = []
    try:
        metadata = parse_frontmatter(text)
    except FrontmatterError as exc:
        return [Finding(skill_md, f"invalid YAML frontmatter: {exc}")]
    if metadata is None:
        return [Finding(skill_md, "missing YAML frontmatter")]

    unexpected_keys = sorted(set(metadata) - {"name", "description"})
    if unexpected_keys:
        findings.append(Finding(skill_md, f"unsupported frontmatter fields: {', '.join(unexpected_keys)}"))

    skill_dir_name = skill_md.parent.name
    name = metadata.get("name", "")
    description = metadata.get("description", "")
    if name != skill_dir_name:
        findings.append(Finding(skill_md, f"frontmatter name {name!r} does not match directory {skill_dir_name!r}"))
    if not SKILL_NAME_RE.fullmatch(name) or len(name) > 64:
        findings.append(Finding(skill_md, "frontmatter name must be 1-64 lowercase letters, digits, or single hyphens"))
    if not description.strip():
        findings.append(Finding(skill_md, "frontmatter description is required"))
    if len(description) > 1024:
        findings.append(Finding(skill_md, "frontmatter description must not exceed 1024 characters"))
    return findings


def validate_openai_yaml(skill_dir: Path, tracked: set[str]) -> list[Finding]:
    yaml_path = skill_dir / "agents" / "openai.yaml"
    if str(yaml_path.relative_to(REPO_ROOT)) not in tracked and not yaml_path.is_file():
        return []
    if not yaml_path.is_file():
        return [Finding(yaml_path, "tracked or expected agents/openai.yaml is missing")]

    lines = read_text(yaml_path).splitlines()
    findings: list[Finding] = []
    content_lines = [line for line in lines if line.strip() and not line.lstrip().startswith("#")]
    if not content_lines or content_lines[0] != "interface:":
        return [Finding(yaml_path, "metadata must have a top-level interface mapping")]

    fields: dict[str, str] = {}
    for line_no, line in enumerate(content_lines[1:], start=2):
        match = re.fullmatch(r'  ([a-z_]+):\s*("(?:[^"\\]|\\.)*")', line)
        if match is None:
            findings.append(Finding(yaml_path, f"line {line_no} must be a quoted interface string field"))
            continue
        key, raw_value = match.groups()
        if key in fields:
            findings.append(Finding(yaml_path, f"duplicate interface field {key!r}"))
            continue
        try:
            fields[key] = ast.literal_eval(raw_value)
        except (SyntaxError, ValueError):
            findings.append(Finding(yaml_path, f"interface field {key!r} is not a valid quoted string"))

    required_fields = {"display_name", "short_description", "default_prompt"}
    for key in sorted(required_fields - fields.keys()):
        findings.append(Finding(yaml_path, f"missing interface field {key}:"))
    for key in sorted(fields.keys() - required_fields):
        findings.append(Finding(yaml_path, f"unsupported interface field {key!r}"))
    skill_name = skill_dir.name
    if f"${skill_name}" not in fields.get("default_prompt", ""):
        findings.append(Finding(yaml_path, f"default_prompt should mention ${skill_name}"))
    metadata_text = "\n".join(fields.values())
    for label, pattern in FORBIDDEN_ACTIVE_SKILL_PATTERNS:
        if pattern.search(metadata_text):
            findings.append(Finding(yaml_path, f"active skill metadata references {label}"))
    return findings


def validate_skill_authority_and_references(skill_md: Path) -> list[Finding]:
    """Reject retired authority and machine-local literals in active skills."""
    text = read_text(skill_md)
    findings: list[Finding] = []
    for label, pattern in FORBIDDEN_ACTIVE_SKILL_PATTERNS:
        if pattern.search(text):
            findings.append(Finding(skill_md, f"active skill references {label}"))

    findings.extend(validate_markdown_references(skill_md))

    for literal in PRIVATE_IPV4_RE.findall(text):
        try:
            address = ipaddress.ip_address(literal)
        except ValueError:
            continue
        if not any(address in network for network in RFC1918_NETWORKS):
            continue
        findings.append(Finding(skill_md, f"active skill must not hard-code private network address {literal!r}"))
    return findings


def validate_no_private_config_tracked(tracked: set[str]) -> list[Finding]:
    findings: list[Finding] = []
    forbidden_prefixes = (
        ".codegraph/",
        ".headroom/",
        ".ai/headroom/",
        ".ai/private/",
        ".ai/venv/",
        ".ai/ms-playwright/",
        ".ai/artifacts/browser/",
    )
    for path in sorted(tracked):
        if path.startswith(forbidden_prefixes):
            findings.append(Finding(REPO_ROOT / path, "private or generated AI artifact is tracked"))
    return findings


def validate_no_personal_skill_refs(tracked: set[str]) -> list[Finding]:
    """Reject personal skill paths and device-level actions from guidance files."""
    findings: list[Finding] = []
    for relative_path in sorted(tracked):
        is_guidance_file = Path(relative_path).name == "AGENTS.md"
        is_governed_prefix = any(
            relative_path == prefix.rstrip("/") or relative_path.startswith(prefix)
            for prefix in GOVERNED_GUIDANCE_PREFIXES
        )
        if not (is_guidance_file or is_governed_prefix):
            continue
        path = REPO_ROOT / relative_path
        if not path.is_file():
            continue
        text = read_text(path)
        for label, pattern in FORBIDDEN_PERSONAL_GUIDANCE_PATTERNS:
            if pattern.search(text):
                findings.append(Finding(path, f"repository guidance must not reference forbidden {label}"))
    return findings


# These byte limits implement ADR-028; they are project budgets, not a quality score.
GUIDANCE_BUDGETS = {"AGENTS.md": 8192, "server/AGENTS.md": 8192, "web/AGENTS.md": 8192, "ai-plan/AGENTS.md": 4096}
GUIDANCE_CHAINS = {
    "root": ("AGENTS.md",),
    "server": ("AGENTS.md", "server/AGENTS.md"),
    "web": ("AGENTS.md", "web/AGENTS.md"),
    "docs/automation": ("AGENTS.md", "ai-plan/AGENTS.md"),
    "cross-boundary": ("AGENTS.md", "server/AGENTS.md", "web/AGENTS.md"),
    "cross-boundary+docs": ("AGENTS.md", "server/AGENTS.md", "web/AGENTS.md", "ai-plan/AGENTS.md"),
}
PROJECT_INSTRUCTION_LIMIT = 32768
REPOSITORY_PLUGIN_MANIFESTS = ("plugins/graft-frontend-vibe-toolchain/.codex-plugin/plugin.json",)
DELIVERY_DOC = REPO_ROOT / "ai-plan/design/governance/ai/仓库协作与交付细则.md"
ROOT_HEADINGS = (
    "Startup Governance", "Authority and Risk Routing", "Authorized Work", "Validation and Completion",
    "Git and Worktree Safety", "Delegation", "License and Text Hygiene",
)


def discover_skill_roots(repo_root: Path | None = None) -> tuple[list[Path], list[Finding]]:
    """Resolve only repository-owned plugin manifests, never installed personal plugins."""
    root = (repo_root or REPO_ROOT).resolve()
    repository_skills = root / ".agents/skills"
    roots: list[Path] = []
    findings: list[Finding] = []
    if not repository_skills.resolve().is_relative_to(root):
        findings.append(Finding(repository_skills, "repository skills directory escapes repository root"))
    elif not repository_skills.is_dir():
        findings.append(Finding(repository_skills, "repository skills directory is missing"))
    else:
        roots.append(repository_skills)
    for relative_manifest in REPOSITORY_PLUGIN_MANIFESTS:
        manifest = root / relative_manifest
        if not manifest.resolve().is_relative_to(root):
            findings.append(Finding(manifest, "plugin manifest escapes repository root"))
            continue
        try:
            payload = json.loads(read_text(manifest))
        except (OSError, ValueError) as exc:
            findings.append(Finding(manifest, f"cannot read repository plugin manifest: {exc}"))
            continue
        skill_path = payload.get("skills") if isinstance(payload, dict) else None
        if not isinstance(skill_path, str) or not skill_path.strip():
            findings.append(Finding(manifest, "plugin skills must be a nonempty relative directory string"))
            continue
        if Path(skill_path).is_absolute() or urlsplit(skill_path).scheme or "\\" in skill_path:
            findings.append(Finding(manifest, "plugin skills must be a repository-local relative directory"))
            continue
        plugin_root = manifest.parent.parent.resolve()
        candidate = (plugin_root / skill_path).resolve()
        if candidate == plugin_root or not candidate.is_relative_to(plugin_root):
            findings.append(Finding(manifest, "plugin skills directory escapes its plugin root"))
        elif not candidate.is_dir():
            findings.append(Finding(manifest, f"plugin skills directory is missing: {skill_path!r}"))
        elif candidate in roots:
            findings.append(Finding(manifest, "duplicate plugin skills directory"))
        else:
            roots.append(candidate)
    return roots, findings


def discover_skill_files(repo_root: Path | None = None) -> tuple[list[Path], list[Finding]]:
    roots, findings = discover_skill_roots(repo_root)
    skill_files: list[Path] = []
    boundary = (repo_root or REPO_ROOT).resolve()
    for root in roots:
        if not root.is_dir():
            findings.append(Finding(root, "skills directory is missing"))
            continue
        directories = sorted(path for path in root.iterdir() if path.is_dir())
        if not directories:
            findings.append(Finding(root, "no repository skills found"))
        for directory in directories:
            skill = directory / "SKILL.md"
            if not skill.resolve().is_relative_to(boundary):
                findings.append(Finding(skill, "skill path escapes repository root"))
            elif not skill.is_file():
                findings.append(Finding(skill, "skill directory missing SKILL.md"))
            else:
                skill_files.append(skill)
    return sorted(skill_files), findings


def markdown_references(path: Path) -> list[tuple[str, Path, str]]:
    """Return local Markdown targets; root-relative links refer to the repository."""
    references = []
    for match in MARKDOWN_LINK_RE.finditer(read_text(path)):
        target = match.group("target").strip().removeprefix("<").removesuffix(">")
        split = urlsplit(target)
        if split.scheme or target.startswith("//"):
            continue
        relative = unquote(split.path)
        resolved = (REPO_ROOT / relative.lstrip("/") if relative.startswith("/") else path.parent / relative) if relative else path
        references.append((target, resolved.resolve(), unquote(split.fragment)))
    return references


def markdown_anchors(text: str) -> set[str]:
    """Recognize explicit IDs and ordinary GitHub Markdown heading slugs."""
    anchors = set(re.findall(r'<a\s+(?:[^>]*?\s)?(?:id|name)=[\"\']([^\"\']+)[\"\']', text))
    counts: dict[str, int] = {}
    for heading in re.findall(r"(?m)^#{1,6}\s+(.+?)\s*#*\s*$", text):
        heading = re.sub(r"<[^>]+>", "", heading).lower()
        slug = re.sub(r"[^\w\- ]", "", heading).replace(" ", "-")
        count = counts.get(slug, 0)
        counts[slug] = count + 1
        anchors.add(f"{slug}-{count}" if count else slug)
    return anchors


def validate_markdown_references(path: Path) -> list[Finding]:
    findings = []
    for raw, target, anchor in markdown_references(path):
        if raw.lstrip("/").split("#", 1)[0] in ALLOWED_MISSING_PRIVATE_REFERENCES:
            continue
        if not target.is_relative_to(REPO_ROOT.resolve()):
            findings.append(Finding(path, f"Markdown reference target escapes repository root: {raw!r}"))
        elif not target.exists():
            findings.append(Finding(path, f"Markdown reference target does not exist: {raw!r}"))
        elif anchor and target.is_file() and target.suffix == ".md" and anchor not in markdown_anchors(read_text(target)):
            findings.append(Finding(path, f"Markdown reference anchor does not exist: {raw!r}"))
    return findings


def linked_skill_documents(skill: Path) -> set[Path]:
    """Follow conditional local references without loading unrelated authority documents."""
    root = skill.parent.resolve()
    skill_roots, _ = discover_skill_roots()
    pending = [skill.resolve()]
    visited: set[Path] = set()
    while pending:
        path = pending.pop()
        if path in visited or not path.is_file():
            continue
        visited.add(path)
        for _, target, _ in markdown_references(path):
            is_local = target.is_relative_to(root)
            is_shared_reference = "references" in target.parts and any(target.is_relative_to(base) for base in skill_roots)
            if (is_local or is_shared_reference) and target.suffix == ".md" and target not in visited:
                pending.append(target)
    return visited


def require_owner_reference(source: Path, owner: Path, *, skill_references: bool = False) -> list[Finding]:
    """Check wiring to an authority, not whether prose implements that authority correctly."""
    if not source.is_file():
        return [Finding(source, "guidance entrypoint is missing")]
    if not owner.is_file():
        return [Finding(owner, "canonical guidance owner is missing")]
    documents = linked_skill_documents(source) if skill_references else {source}
    if any(target == owner.resolve() for document in documents for _, target, _ in markdown_references(document)):
        return []
    return [Finding(source, f"missing Markdown reference to canonical owner {owner.relative_to(REPO_ROOT)}")]


def guidance_size_report(repo_root: Path | None = None, *, budgets: dict[str, int] | None = None,
                         chains: dict[str, tuple[str, ...]] | None = None) -> dict[str, object]:
    root = repo_root or REPO_ROOT
    limits = GUIDANCE_BUDGETS if budgets is None else budgets
    chain_paths = GUIDANCE_CHAINS if chains is None else chains
    sizes = {name: (root / name).stat().st_size if (root / name).is_file() else None for name in limits}
    return {
        "files": [{"path": name, "bytes": size, "budget": limits[name]} for name, size in sizes.items()],
        "chains": [{"name": name, "paths": list(paths), "bytes": sum(sizes.get(path) or 0 for path in paths) + 2 * max(0, len(paths) - 1),
                    "separator_bytes": 2 * max(0, len(paths) - 1),
                    "complete": all(sizes.get(path) is not None for path in paths), "budget": PROJECT_INSTRUCTION_LIMIT}
                   for name, paths in chain_paths.items()],
    }


def validate_guidance_budgets(repo_root: Path | None = None, *, budgets: dict[str, int] | None = None,
                              chains: dict[str, tuple[str, ...]] | None = None) -> list[Finding]:
    root = repo_root or REPO_ROOT
    report = guidance_size_report(root, budgets=budgets, chains=chains)
    findings = []
    for entry in report["files"]:
        path = root / entry["path"]
        if entry["bytes"] is None:
            findings.append(Finding(path, "required AGENTS instruction file is missing"))
        elif entry["bytes"] > entry["budget"]:
            findings.append(Finding(path, f"UTF-8 instruction budget exceeded: {entry['bytes']} bytes > {entry['budget']} bytes"))
        try:
            if path.is_file():
                read_text(path)
        except UnicodeDecodeError:
            findings.append(Finding(path, "instruction file must be valid UTF-8"))
    for entry in report["chains"]:
        if not entry["complete"] or entry["bytes"] > entry["budget"]:
            evidence = ", ".join(f"{item['path']}={item['bytes']} bytes" for item in report["files"] if item["path"] in entry["paths"])
            findings.append(Finding(root / "AGENTS.md", f"instruction chain {entry['name']!r} is incomplete or exceeds {entry['budget']} bytes: {entry['bytes']} bytes ({evidence})"))
    return findings


def skill_size_report(skills: list[Path]) -> list[dict[str, object]]:
    records = []
    for path in skills:
        text = read_text(path)
        try:
            description = (parse_frontmatter(text) or {}).get("description", "")
        except FrontmatterError:
            description = ""
        body = FRONTMATTER_RE.sub("", text, count=1)
        records.append({"path": path.relative_to(REPO_ROOT).as_posix(), "description_chars": len(description),
                        "description_bytes": len(description.encode("utf-8")), "body_bytes": len(body.encode("utf-8"))})
    return records


def validate_skills() -> list[Finding]:
    skills, findings = discover_skill_files()
    tracked = tracked_files()
    for skill in skills:
        findings.extend(validate_skill_frontmatter(skill))
        findings.extend(validate_openai_yaml(skill.parent, tracked))
        for document in linked_skill_documents(skill):
            findings.extend(validate_skill_authority_and_references(document))
        findings.extend(require_owner_reference(skill, AGENTS, skill_references=True))
    return findings


def validate_guidance_owners() -> list[Finding]:
    """Keep the documented ownership graph intact; scenarios evaluate semantic decisions."""
    findings = []
    root_text = read_text(AGENTS) if AGENTS.is_file() else ""
    headings = set(re.findall(r"(?m)^##\s+(.+?)\s*$", root_text))
    for heading in ROOT_HEADINGS:
        if heading not in headings:
            findings.append(Finding(AGENTS, f"missing canonical governance section {heading!r}"))
    for source in (AGENTS, SERVER_AGENTS, WEB_AGENTS, AI_PLAN_AGENTS):
        if not source.is_file():
            continue
        sections = re.split(r"(?m)^##\s+(.+?)\s*$", read_text(source))
        for title, content in zip(sections[1::2], sections[2::2]):
            meaningful = re.sub(r"(?s)<!--.*?-->", "", content)
            meaningful = re.sub(r"(?m)^#{1,6}.*$", "", meaningful)
            if not meaningful.strip():
                findings.append(Finding(source, f"guidance section {title!r} has no content"))
    for owner in (SERVER_AGENTS, WEB_AGENTS, AI_PLAN_AGENTS, AI_TOOLING_DOC, AI_CODE_REVIEW_DOC,
                  AI_TASK_TRACKING_DOC, DELIVERY_DOC):
        findings.extend(require_owner_reference(AGENTS, owner))
    for source in (SERVER_AGENTS, WEB_AGENTS, AI_PLAN_AGENTS, DELIVERY_DOC):
        findings.extend(require_owner_reference(source, AGENTS))
    for owner in (BACKEND_QUERY_DOC, SERVER_API_GOVERNANCE_DOC, BACKEND_SECURITY_DOC, BACKEND_TEST_MAINTAIN_DOC):
        findings.extend(require_owner_reference(SERVER_AGENTS, owner))
    findings.extend(require_owner_reference(WEB_AGENTS, TDESIGN_DOC))
    for skill in (COMMIT_SKILL, PUSH_SKILL, WORKTREE_MANAGER_SKILL):
        findings.extend(require_owner_reference(skill, DELIVERY_DOC, skill_references=True))
    for source in (AGENTS, SERVER_AGENTS, WEB_AGENTS, AI_PLAN_AGENTS, DELIVERY_DOC):
        if source.is_file():
            findings.extend(validate_markdown_references(source))
            if contains_headroom_rtk_injection(read_text(source)) or contains_project_rtk_prefix_rule(read_text(source)):
                findings.append(Finding(source, "automatic Headroom/RTK instructions or mandatory command prefixes are forbidden"))
    return findings


def validate_pr_machine_contracts() -> list[Finding]:
    """Keep documented safety flags tied to their executable PR helper authority."""
    findings = []
    for source in (PR_REVIEW_SKILL, PR_REMEDIATION_SKILL):
        findings.extend(require_owner_reference(source, PR_REVIEW_COMMANDS, skill_references=True))
    flags = ("--resolve-comment-id", "--resolve-expected-head", "--resolve-dry-run", "--ledger-validate-body-file",
             "--ledger-expected-head", "--ledger-expected-revision", "--ledger-dry-run")
    for source in (PR_REVIEW_HELPER, PR_REVIEW_COMMANDS):
        if not source.is_file():
            findings.append(Finding(source, "PR helper or canonical command reference is missing"))
        else:
            findings.extend(missing_exact_terms(read_text(source), source, "PR helper interface", flags))
    return findings


def validate_owned_helpers() -> list[Finding]:
    """Retain executable helper/test coverage and delegate registry semantics to its owner."""
    findings = []
    for relative in (
        ".agents/skills/graft-worktree-manager/scripts/worktree_manager.py",
        ".agents/skills/graft-worktree-manager/scripts/test_worktree_manager.py",
        ".agents/skills/graft-pr-review/scripts/test_fetch_current_pr_review.py",
        "scripts/validate_sql_migrations.py", "scripts/validate_ai_plan_structure.py",
        "scripts/validate_loop_controller_contract.py", "scripts/run_skill_tests.py",
        ".ai/registries/web-shared-assets.yaml", ".ai/registries/server-shared-assets.yaml",
        ".ai/registries/cross-boundary-assets.yaml",
    ):
        path = REPO_ROOT / relative
        if not path.is_file():
            findings.append(Finding(path, "governed helper, regression test, or registry is missing"))
    if not SHARED_ASSET_VALIDATOR.is_file():
        return findings + [Finding(SHARED_ASSET_VALIDATOR, "shared asset validator is missing")]
    try:
        spec = importlib.util.spec_from_file_location("validate_shared_asset_registries", SHARED_ASSET_VALIDATOR)
        if spec is None or spec.loader is None:
            raise RuntimeError("could not load shared asset registry validator")
        module = importlib.util.module_from_spec(spec)
        sys.modules[spec.name] = module
        spec.loader.exec_module(module)
        findings.extend(Finding(item.path, item.message) for item in module.validate_registries())
    except Exception as exc:
        findings.append(Finding(SHARED_ASSET_VALIDATOR, f"shared asset registry validator failed: {exc}"))
    return findings


def run_validation() -> list[Finding]:
    findings = []
    tracked = tracked_files()
    for check in (validate_required_files, validate_gitignore, validate_ai_tooling_doc, validate_skills,
                  validate_guidance_budgets, validate_guidance_owners, validate_pr_machine_contracts,
                  validate_owned_helpers, validate_environment_inventory):
        findings.extend(check())
    findings.extend(validate_no_private_config_tracked(tracked))
    findings.extend(validate_no_personal_skill_refs(tracked))
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate AI guidance structure, byte budgets, and helper contracts.")
    parser.add_argument("--format", choices=("text", "json"), default="text", help="output format")
    args = parser.parse_args()
    findings = run_validation()
    skills, _ = discover_skill_files()
    statistics = {"instructions": guidance_size_report(), "skills": skill_size_report(skills)}
    if args.format == "json":
        print(json.dumps({"ok": not findings, "findings": [item.format() for item in findings],
                          "statistics": statistics}, ensure_ascii=False, indent=2))
    else:
        print(f"Guidance statistics (size is not quality): {len(skills)} skills; "
              f"{sum(item['description_chars'] for item in statistics['skills'])} description characters; "
              f"{sum(item['body_bytes'] for item in statistics['skills'])} body bytes")
        for item in statistics["instructions"]["files"]:
            print(f"  {item['path']}: {item['bytes']}/{item['budget']} UTF-8 bytes")
        for item in statistics["instructions"]["chains"]:
            print(f"  chain {item['name']}: {item['bytes']}/{item['budget']} bytes")
        if findings:
            print("AI governance validation failed:", file=sys.stderr)
            for finding in findings:
                print(f"- {finding.format()}", file=sys.stderr)
        else:
            print("AI governance validation passed")
    return 1 if findings else 0


if __name__ == "__main__":
    raise SystemExit(main())
