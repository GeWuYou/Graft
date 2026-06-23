#!/usr/bin/env python3
"""Validate repository AI governance documents and skills stay aligned."""

from __future__ import annotations

import argparse
import importlib.util
import re
import shutil
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
SKILLS_DIR = REPO_ROOT / ".agents" / "skills"
AI_TOOLING_DOC = REPO_ROOT / "ai-plan" / "design" / "AI工具与MCP接入治理规范.md"
CODEGRAPH_DOC = REPO_ROOT / "ai-plan" / "design" / "CodeGraph-MCP-辅助开发规范.md"
TDESIGN_DOC = REPO_ROOT / "ai-plan" / "design" / "TDesign-MCP-辅助开发规范.md"
TOOLS_AI = REPO_ROOT / ".ai" / "environment" / "tools.ai.yaml"
GITIGNORE = REPO_ROOT / ".gitignore"
AGENTS = REPO_ROOT / "AGENTS.md"
WEB_BROWSER_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-web-browser-agent" / "SKILL.md"
PR_REVIEW_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-pr-review" / "SKILL.md"
PR_CREATE_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-pr-create" / "SKILL.md"
AI_AUDIT_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-ai-governance-audit" / "SKILL.md"
PUSH_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-push" / "SKILL.md"
TABLE_DESIGN_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-table-design" / "SKILL.md"
SQL_MIGRATION_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-sql-migration" / "SKILL.md"
SHARED_ASSET_REUSE_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-shared-asset-reuse" / "SKILL.md"
SHARED_ASSET_DOC = REPO_ROOT / "ai-plan" / "design" / "共享资产复用治理规范.md"
SHARED_ASSET_VALIDATOR = REPO_ROOT / "scripts" / "validate_shared_asset_registries.py"
BACKEND_QUERY_DOC = REPO_ROOT / "ai-plan" / "design" / "后端查询与数据库访问治理规范.md"
SERVER_API_GOVERNANCE_DOC = REPO_ROOT / "ai-plan" / "design" / "服务端API边界与兼容治理规范.md"
BACKEND_SECURITY_DOC = REPO_ROOT / "ai-plan" / "design" / "后端安全与信任边界治理规范.md"
BACKEND_TEST_MAINTAIN_DOC = REPO_ROOT / "ai-plan" / "design" / "后端测试与可维护性治理规范.md"
AI_CODE_REVIEW_DOC = REPO_ROOT / "ai-plan" / "design" / "AI代码生成与Review规范.md"
SERVER_AGENTS = REPO_ROOT / "server" / "AGENTS.md"

FRONTMATTER_RE = re.compile(r"\A---\n(?P<body>.*?)\n---\n", re.DOTALL)
HEADROOM_RTK_START = "<!-- headroom:rtk-instructions -->"
HEADROOM_RTK_END = "<!-- /headroom:rtk-instructions -->"


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


def parse_frontmatter(text: str) -> dict[str, str] | None:
    match = FRONTMATTER_RE.match(text)
    if match is None:
        return None

    values: dict[str, str] = {}
    for raw_line in match.group("body").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        key, sep, raw_value = line.partition(":")
        if not sep:
            continue
        value = raw_value.strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in {'"', "'"}:
            value = value[1:-1]
        values[key.strip()] = value
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
    if not AI_TOOLING_DOC.is_file():
        return []
    text = read_text(AI_TOOLING_DOC)
    findings: list[Finding] = []
    exact_terms = (
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
    return findings


def validate_skill_mcp_guidance() -> list[Finding]:
    checks = (
        (
            WEB_BROWSER_SKILL,
            ("Playwright MCP", "browser_agent.py", "playwright_mcp_used"),
        ),
        (
            PR_REVIEW_SKILL,
            ("GitHub MCP", "fetch_current_pr_review.py", "deterministic fallback"),
        ),
        (
            PR_CREATE_SKILL,
            ("GitHub MCP", "ensure_pr.py", "deterministic fallback"),
        ),
        (
            AI_AUDIT_SKILL,
            (
                "codex mcp get context7",
                "codex mcp get github",
                "codex mcp get playwright",
                "codex mcp get headroom",
            ),
        ),
    )
    findings: list[Finding] = []
    for path, terms in checks:
        if not path.is_file():
            findings.append(Finding(path, "MCP-aware skill file is missing"))
            continue
        text = read_text(path)
        for term in terms:
            if term not in text:
                findings.append(Finding(path, f"missing MCP guidance term {term!r}"))
    return findings


def validate_sql_migration_governance() -> list[Finding]:
    findings: list[Finding] = []
    if not SQL_MIGRATION_SKILL.is_file():
        findings.append(Finding(SQL_MIGRATION_SKILL, "SQL migration skill is missing"))
    else:
        text = read_text(SQL_MIGRATION_SKILL)
        for term in (
            "python3 scripts/validate_sql_migrations.py",
            "COMMENT ON TABLE",
            "COMMENT ON COLUMN",
            "server/internal/ent/migrate/migrations/**",
            "globally unique",
            "legacy migration",
        ):
            if term not in text:
                findings.append(Finding(SQL_MIGRATION_SKILL, f"missing SQL migration governance term {term!r}"))

    if TABLE_DESIGN_SKILL.is_file():
        text = read_text(TABLE_DESIGN_SKILL)
        for term in ("graft-sql-migration", "python3 scripts/validate_sql_migrations.py"):
            if term not in text:
                findings.append(Finding(TABLE_DESIGN_SKILL, f"missing SQL migration skill handoff term {term!r}"))

    if AI_TOOLING_DOC.is_file():
        text = read_text(AI_TOOLING_DOC)
        if "graft-sql-migration" not in text:
            findings.append(Finding(AI_TOOLING_DOC, "AI tooling governance should mention graft-sql-migration"))
        if "scripts/validate_sql_migrations.py" not in text:
            findings.append(Finding(AI_TOOLING_DOC, "AI tooling governance should mention SQL migration validation helper"))
    return findings


def validate_environment_inventory() -> list[Finding]:
    if not TOOLS_AI.is_file():
        return []
    text = read_text(TOOLS_AI)
    findings: list[Finding] = []
    exact_terms = (
        "preferred: \"headroom mcp\"",
        ".ai/headroom/memory",
        ".ai/headroom/learn",
        "rtk instruction injection",
        "automatic instructions write",
        "default_command:",
        "instructions_auto_write: \"disabled\"",
    )
    findings.extend(missing_exact_terms(text, TOOLS_AI, "AI environment inventory", exact_terms))
    findings.extend(
        missing_concepts(
            text,
            TOOLS_AI,
            "AI environment inventory",
            (
                ("Headroom CLI and MCP capabilities", (r"ai_headroom:\s*true", r"ai_headroom_mcp:\s*true")),
                ("context compression tool selection", (r"context_compression:", r"preferred:\s+\"headroom mcp\"")),
                ("controlled local Headroom directories", (r"controlled_local_dirs:", r"\.ai/headroom/memory", r"\.ai/headroom/learn")),
                ("disallowed Headroom automation", (r"disallowed_by_default:", r"rtk instruction injection", r"automatic instructions write")),
                ("Headroom AI tool record", (r"ai_tools:", r"headroom:", r"instructions_auto_write:\s+\"disabled\"")),
                ("adopted and pilot MCP server records", (r"mcp_servers:", r"codegraph:", r"tdesign:", r"context7:", r"github:", r"playwright:", r"headroom:")),
            ),
        )
    )
    for disallowed in ("headroom wrap codex", "headroom proxy", "wrapper_available:", "proxy_available:"):
        if disallowed in text:
            findings.append(Finding(TOOLS_AI, f"AI environment inventory should keep only MCP entry content, found {disallowed!r}"))
    return findings


def validate_skill_frontmatter(skill_md: Path) -> list[Finding]:
    text = read_text(skill_md)
    metadata = parse_frontmatter(text)
    findings: list[Finding] = []
    if metadata is None:
        return [Finding(skill_md, "missing YAML frontmatter")]

    skill_dir_name = skill_md.parent.name
    name = metadata.get("name", "")
    description = metadata.get("description", "")
    if name != skill_dir_name:
        findings.append(Finding(skill_md, f"frontmatter name {name!r} does not match directory {skill_dir_name!r}"))
    if not description:
        findings.append(Finding(skill_md, "frontmatter description is required"))
    if len(description) < 80:
        findings.append(Finding(skill_md, "frontmatter description should be explicit enough for skill discovery"))
    return findings


def validate_openai_yaml(skill_dir: Path, tracked: set[str]) -> list[Finding]:
    yaml_path = skill_dir / "agents" / "openai.yaml"
    if str(yaml_path.relative_to(REPO_ROOT)) not in tracked and not yaml_path.is_file():
        return []
    if not yaml_path.is_file():
        return [Finding(yaml_path, "tracked or expected agents/openai.yaml is missing")]

    text = read_text(yaml_path)
    findings: list[Finding] = []
    for key in ("display_name:", "short_description:", "default_prompt:"):
        if key not in text:
            findings.append(Finding(yaml_path, f"missing interface field {key}"))
    skill_name = skill_dir.name
    if f"${skill_name}" not in text:
        findings.append(Finding(yaml_path, f"default_prompt should mention ${skill_name}"))
    return findings


def validate_skills() -> list[Finding]:
    if not SKILLS_DIR.is_dir():
        return [Finding(SKILLS_DIR, "skills directory is missing")]

    tracked = tracked_files()
    findings: list[Finding] = []
    skill_dirs = sorted(path for path in SKILLS_DIR.iterdir() if path.is_dir())
    if not skill_dirs:
        findings.append(Finding(SKILLS_DIR, "no repository skills found"))

    for skill_dir in skill_dirs:
        skill_md = skill_dir / "SKILL.md"
        if not skill_md.is_file():
            findings.append(Finding(skill_md, "skill directory missing SKILL.md"))
            continue
        findings.extend(validate_skill_frontmatter(skill_md))
        findings.extend(validate_openai_yaml(skill_dir, tracked))

    audit_skill = SKILLS_DIR / "graft-ai-governance-audit" / "SKILL.md"
    if not audit_skill.is_file():
        findings.append(Finding(audit_skill, "AI governance audit skill is missing"))
    return findings


def validate_agents_skill_list() -> list[Finding]:
    """
    验证 AGENTS.md 包含所需的技能列表且不含禁止的治理内容。
    
    Returns:
    	list[Finding]: 验证失败项的 Finding 列表；文件不存在或通过验证时返回空列表。
    """
    if not AGENTS.is_file():
        return []
    text = read_text(AGENTS)
    findings: list[Finding] = []
    for skill_name in (
        "graft-codegraph-mcp",
        "graft-ai-governance-audit",
        "graft-validation-runner",
        "graft-sql-migration",
        "graft-shared-asset-reuse",
    ):
        if skill_name not in text:
            findings.append(Finding(AGENTS, f"repository skill list does not mention {skill_name}"))
    if contains_headroom_rtk_injection(text):
        findings.append(Finding(AGENTS, "Headroom/RTK automatic instruction block must not be committed"))
    if contains_project_rtk_prefix_rule(text):
        findings.append(Finding(AGENTS, "project governance must not require agents to always prefix commands with rtk"))
    return findings


def validate_push_branch_governance() -> list[Finding]:
    findings: list[Finding] = []

    if AGENTS.is_file():
        text = read_text(AGENTS)
        for term in (
            "For push-triggered branch-name hygiene:",
            "`$graft-push` must inspect the commits that are about to be pushed",
            "`@{upstream}..HEAD`",
            "lowercase kebab-case",
            "do not rename branches during ordinary `$graft-commit` runs",
        ):
            if term not in text:
                findings.append(Finding(AGENTS, f"missing push branch governance term {term!r}"))

    if not PUSH_SKILL.is_file():
        findings.append(Finding(PUSH_SKILL, "push skill is missing"))
        return findings

    text = read_text(PUSH_SKILL)
    for term in (
        "git log --oneline @{upstream}..HEAD",
        "merge-base with the intended base branch, normally `main`",
        "branch names must follow `<type>/<topic-or-scope>`",
        "lowercase kebab-case",
        "generic `wt-*` placeholders",
        "rename the local branch before pushing",
        "do not auto-delete the old remote branch after a rename unless the user explicitly asks",
        "whether a branch-name check ran, what commit range it used, and whether a rename happened",
    ):
        if term not in text:
            findings.append(Finding(PUSH_SKILL, f"missing push branch governance term {term!r}"))

    return findings


def validate_backend_guardrail_governance() -> list[Finding]:
    """
    验证后端守护栏治理文档的存在性和完整性。
    
    检查必需的后端治理文档（查询、API、安全、测试、代码Review规范）是否存在，
    各文档是否包含所需的治理术语，以及AGENTS.md和server/AGENTS.md是否对这些规范进行了引用。
    
    Returns:
    	list[Finding]: 各文件缺失或不符合要求的Finding列表。
    """
    findings: list[Finding] = []
    required_docs = (
        BACKEND_QUERY_DOC,
        SERVER_API_GOVERNANCE_DOC,
        BACKEND_SECURITY_DOC,
        BACKEND_TEST_MAINTAIN_DOC,
        AI_CODE_REVIEW_DOC,
    )
    for path in required_docs:
        if not path.is_file():
            findings.append(Finding(path, "backend guardrail governance file is missing"))

    if BACKEND_QUERY_DOC.is_file():
        text = read_text(BACKEND_QUERY_DOC)
        findings.extend(
            missing_exact_terms(
                text,
                BACKEND_QUERY_DOC,
                "backend query governance",
                ("N+1", "全表扫描", "分页", "SELECT *", "Count", "EXPLAIN", "查询超时", "大字段", "批量", "CI"),
            )
        )

    if SERVER_API_GOVERNANCE_DOC.is_file():
        text = read_text(SERVER_API_GOVERNANCE_DOC)
        findings.extend(
            missing_exact_terms(
                text,
                SERVER_API_GOVERNANCE_DOC,
                "server API governance",
                ("Entity", "DTO", "VO", "Request", "Response", "OpenAPI", "兼容", "废弃", "Ent entity", "CI"),
            )
        )

    if BACKEND_SECURITY_DOC.is_file():
        text = read_text(BACKEND_SECURITY_DOC)
        findings.extend(
            missing_exact_terms(
                text,
                BACKEND_SECURITY_DOC,
                "backend security governance",
                ("authz", "审计", "危险操作", "信任边界", "前端", "批量", "CI"),
            )
        )

    if BACKEND_TEST_MAINTAIN_DOC.is_file():
        text = read_text(BACKEND_TEST_MAINTAIN_DOC)
        findings.extend(
            missing_exact_terms(
                text,
                BACKEND_TEST_MAINTAIN_DOC,
                "backend test maintainability governance",
                ("query-count", "public API", "service", "复杂函数", "兼容", "导出符号", "魔法值", "lint", "CI"),
            )
        )

    if AI_CODE_REVIEW_DOC.is_file():
        text = read_text(AI_CODE_REVIEW_DOC)
        findings.extend(
            missing_exact_terms(
                text,
                AI_CODE_REVIEW_DOC,
                "AI code review governance",
                ("跨模块重构", "自动", "依赖升级", "TODO", "closeout", "rollback", "多 agent", "CI"),
            )
        )

    if AGENTS.is_file():
        root_text = read_text(AGENTS)
        for term in (
            "后端查询与数据库访问治理规范.md",
            "服务端API边界与兼容治理规范.md",
            "后端安全与信任边界治理规范.md",
            "后端测试与可维护性治理规范.md",
            "AI代码生成与Review规范.md",
        ):
            if term not in root_text:
                findings.append(Finding(AGENTS, f"root AGENTS should reference backend guardrail doc {term!r}"))

    if SERVER_AGENTS.is_file():
        text = read_text(SERVER_AGENTS)
        for term in (
            "后端查询与数据库访问治理规范.md",
            "服务端API边界与兼容治理规范.md",
            "后端安全与信任边界治理规范.md",
            "后端测试与可维护性治理规范.md",
            "AI代码生成与Review规范.md",
            "### Backend Guardrails",
            "禁止引入 N+1 查询",
            "列表接口默认分页",
            "不暴露 Ent entity",
            "不信任前端上传",
            "写接口必须做后端权限校验",
            "危险操作必须具备权限、审计",
            "query-count regression",
            "禁止超范围修改",
            "禁止自动数据库迁移",
            "回滚方案",
        ):
            if term not in text:
                findings.append(Finding(SERVER_AGENTS, f"server AGENTS missing backend guardrail term {term!r}"))

    if AI_TOOLING_DOC.is_file():
        text = read_text(AI_TOOLING_DOC)
        for term in (
            "规范",
            "`AGENTS.md`",
            "CI / validation script",
            "review checklist",
            "AI guardrail",
        ):
            if term not in text:
                findings.append(Finding(AI_TOOLING_DOC, f"AI tooling governance should mention guardrail adoption term {term!r}"))

    return findings


def validate_shared_asset_governance() -> list[Finding]:
    """
    验证共享资产治理文档、注册表和验证器脚本。
    
    Returns:
        list[Finding]: 包含所有检测到的问题（包括缺失文件、缺失条款或验证失败）的 Finding 对象列表
    """
    findings: list[Finding] = []
    required = (
        SHARED_ASSET_DOC,
        SHARED_ASSET_REUSE_SKILL,
        SHARED_ASSET_VALIDATOR,
        REPO_ROOT / ".ai" / "registries" / "web-shared-assets.yaml",
        REPO_ROOT / ".ai" / "registries" / "server-shared-assets.yaml",
        REPO_ROOT / ".ai" / "registries" / "cross-boundary-assets.yaml",
    )
    for path in required:
        if not path.is_file():
            findings.append(Finding(path, "shared asset governance file is missing"))

    if SHARED_ASSET_DOC.is_file():
        text = read_text(SHARED_ASSET_DOC)
        for term in (
            "人工策展的治理索引",
            "不是源码树清单",
            "维护触发",
            "登记标准",
            "移除与重命名",
            "scripts/validate_shared_asset_registries.py",
            "新发现的未登记文件最多产生 warning",
        ):
            if term not in text:
                findings.append(Finding(SHARED_ASSET_DOC, f"missing shared asset governance term {term!r}"))

    if SHARED_ASSET_REUSE_SKILL.is_file():
        text = read_text(SHARED_ASSET_REUSE_SKILL)
        for term in (
            "shared_asset_preflight",
            "registries_checked",
            "assets_reused",
            "assets_considered_but_rejected",
            "new_registry_entries",
            "registry_entries_removed_or_replaced",
            "validation_commands",
        ):
            if term not in text:
                findings.append(Finding(SHARED_ASSET_REUSE_SKILL, f"missing shared asset closeout term {term!r}"))
    if SHARED_ASSET_VALIDATOR.is_file():
        try:
            spec = importlib.util.spec_from_file_location("validate_shared_asset_registries", SHARED_ASSET_VALIDATOR)
            if spec is None or spec.loader is None:
                findings.append(Finding(SHARED_ASSET_VALIDATOR, "could not load shared asset registry validator"))
                return findings
            module = importlib.util.module_from_spec(spec)
            sys.modules[spec.name] = module
            spec.loader.exec_module(module)
            for finding in module.validate_registries():
                findings.append(Finding(finding.path, finding.message))
        except Exception as exc:
            findings.append(Finding(SHARED_ASSET_VALIDATOR, f"shared asset registry validator failed: {exc}"))
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


def run_validation() -> list[Finding]:
    """
    执行所有 AI 治理验证。
    
    Returns:
        list[Finding]: 验证中发现的所有问题列表。
    """
    findings: list[Finding] = []
    findings.extend(validate_required_files())
    tracked = tracked_files()
    findings.extend(validate_gitignore())
    findings.extend(validate_ai_tooling_doc())
    findings.extend(validate_skills())
    findings.extend(validate_skill_mcp_guidance())
    findings.extend(validate_sql_migration_governance())
    findings.extend(validate_shared_asset_governance())
    findings.extend(validate_agents_skill_list())
    findings.extend(validate_push_branch_governance())
    findings.extend(validate_backend_guardrail_governance())
    findings.extend(validate_environment_inventory())
    findings.extend(validate_no_private_config_tracked(tracked))
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate AI governance documents, skills, and local artifact hygiene.")
    parser.add_argument("--format", choices=("text", "json"), default="text", help="output format")
    args = parser.parse_args()

    findings = run_validation()
    if args.format == "json":
        import json

        payload = {"ok": not findings, "findings": [finding.format() for finding in findings]}
        print(json.dumps(payload, ensure_ascii=False, indent=2))
    elif findings:
        print("AI governance validation failed:", file=sys.stderr)
        for finding in findings:
            print(f"- {finding.format()}", file=sys.stderr)
    else:
        print("AI governance validation passed")

    return 1 if findings else 0


if __name__ == "__main__":
    raise SystemExit(main())
