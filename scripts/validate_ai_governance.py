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
AI_TOOLING_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "ai" / "AI工具与MCP接入治理规范.md"
CODEGRAPH_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "ai" / "CodeGraph-MCP-辅助开发规范.md"
TDESIGN_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "frontend" / "TDesign-MCP-辅助开发规范.md"
TOOLS_AI = REPO_ROOT / ".ai" / "environment" / "tools.ai.yaml"
GITIGNORE = REPO_ROOT / ".gitignore"
AGENTS = REPO_ROOT / "AGENTS.md"
AI_PLAN_AGENTS = REPO_ROOT / "ai-plan" / "AGENTS.md"
AI_PLAN_README = REPO_ROOT / "ai-plan" / "README.md"
AI_TASK_TRACKING_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "ai" / "AI任务追踪与恢复设计.md"
WEB_BROWSER_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-web-browser-agent" / "SKILL.md"
VALIDATION_RUNNER_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-validation-runner" / "SKILL.md"
TASK_CLOSEOUT_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-task-closeout" / "SKILL.md"
WEB_VIBE_CODING_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-web-vibe-coding" / "SKILL.md"
WEB_AGENTS = REPO_ROOT / "web" / "AGENTS.md"
PR_REVIEW_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-pr-review" / "SKILL.md"
PR_CREATE_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-pr-create" / "SKILL.md"
AI_AUDIT_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-ai-governance-audit" / "SKILL.md"
AI_PLAN_GOVERNANCE_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-ai-plan-governance" / "SKILL.md"
WORK_INTAKE_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-work-intake" / "SKILL.md"
WORKTREE_MANAGER_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-worktree-manager" / "SKILL.md"
PUSH_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-push" / "SKILL.md"
COMMIT_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-commit" / "SKILL.md"
TABLE_DESIGN_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-table-design" / "SKILL.md"
SQL_MIGRATION_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-sql-migration" / "SKILL.md"
SHARED_ASSET_REUSE_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-shared-asset-reuse" / "SKILL.md"
SHARED_ASSET_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "platform" / "共享资产复用治理规范.md"
SHARED_ASSET_VALIDATOR = REPO_ROOT / "scripts" / "validate_shared_asset_registries.py"
BACKEND_QUERY_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "backend" / "后端查询与数据库访问治理规范.md"
SERVER_API_GOVERNANCE_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "backend" / "服务端API边界与兼容治理规范.md"
BACKEND_SECURITY_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "backend" / "后端安全与信任边界治理规范.md"
BACKEND_TEST_MAINTAIN_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "backend" / "后端测试与可维护性治理规范.md"
AI_CODE_REVIEW_DOC = REPO_ROOT / "ai-plan" / "design" / "governance" / "ai" / "AI代码生成与Review规范.md"
SERVER_AGENTS = REPO_ROOT / "server" / "AGENTS.md"
SUBAGENT_DELEGATION_SKILLS = (
    REPO_ROOT / ".agents" / "skills" / "graft-multi-agent-batch" / "SKILL.md",
    REPO_ROOT / ".agents" / "skills" / "graft-multi-agent-loop" / "SKILL.md",
    REPO_ROOT / ".agents" / "skills" / "graft-multi-agent-task" / "SKILL.md",
    REPO_ROOT / ".agents" / "skills" / "graft-comment-governance" / "SKILL.md",
)

FRONTMATTER_RE = re.compile(r"\A---\n(?P<body>.*?)\n---\n", re.DOTALL)
HEADROOM_RTK_START = "<!-- headroom:rtk-instructions -->"
HEADROOM_RTK_END = "<!-- /headroom:rtk-instructions -->"
GOVERNED_GUIDANCE_PREFIXES = (".agents/", "ai-plan/", ".ai/")
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
        "preferred: \"headroom mcp\"",
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


def validate_ai_plan_governance_skill() -> list[Finding]:
    findings: list[Finding] = []

    if not AI_PLAN_GOVERNANCE_SKILL.is_file():
        return [Finding(AI_PLAN_GOVERNANCE_SKILL, "AI plan governance skill is missing")]

    skill_text = read_text(AI_PLAN_GOVERNANCE_SKILL)
    for term in (
        "root `AGENTS.md`",
        "`ai-plan/AGENTS.md`",
        "`ai-plan/README.md`",
        "`ai-plan/public/README.md`",
        "`ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`",
        "`ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`",
        "python3 scripts/validate_ai_plan_structure.py",
        "python3 scripts/validate_ai_governance.py",
        "compose-project-management",
        "second startup",
        "second validation",
    ):
        if term not in skill_text:
            findings.append(Finding(AI_PLAN_GOVERNANCE_SKILL, f"missing ai-plan governance skill term {term!r}"))

    findings.extend(validate_openai_yaml(AI_PLAN_GOVERNANCE_SKILL.parent, tracked_files()))

    reference_checks = (
        (AGENTS, "graft-ai-plan-governance"),
        (AI_PLAN_AGENTS, "graft-ai-plan-governance"),
        (AI_PLAN_README, "graft-ai-plan-governance"),
        (AI_TOOLING_DOC, "graft-ai-plan-governance"),
    )
    for path, term in reference_checks:
        if path.is_file() and term not in read_text(path):
            findings.append(Finding(path, f"missing ai-plan governance skill reference {term!r}"))

    return findings


def validate_work_intake_skill() -> list[Finding]:
    findings: list[Finding] = []

    if not WORK_INTAKE_SKILL.is_file():
        return [Finding(WORK_INTAKE_SKILL, "work intake skill is missing")]

    text = read_text(WORK_INTAKE_SKILL)
    for term in (
        "root `AGENTS.md`",
        "`ai-plan/AGENTS.md`",
        "`ai-plan/README.md`",
        "`ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`",
        "`ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`",
        "`ai-plan/design/decisions/ADR-003-work-intake-and-bootstrap-model.md`",
        "Work Contract",
        "contract-driven minimal bootstrap",
        "do not define independent business rules",
        "do not create a second startup path",
        "do not create a standalone `work-contract.yaml`",
        "do not put the full contract in `ai-plan/catalog.json`",
        "graft-multi-agent-loop",
    ):
        if term not in text:
            findings.append(Finding(WORK_INTAKE_SKILL, f"missing work intake governance term {term!r}"))

    findings.extend(validate_openai_yaml(WORK_INTAKE_SKILL.parent, tracked_files()))

    reference_checks = (
        (AGENTS, "graft-work-intake"),
        (AI_PLAN_AGENTS, "graft-work-intake"),
        (AI_PLAN_README, "graft-work-intake"),
        (AI_TOOLING_DOC, "graft-work-intake"),
    )
    for path, term in reference_checks:
        if path.is_file() and term not in read_text(path):
            findings.append(Finding(path, f"missing work intake skill reference {term!r}"))

    return findings


def validate_agents_skill_list() -> list[Finding]:
    """
    验证 AGENTS.md 中的技能清单与治理约束。
    
    Returns:
        list[Finding]: 校验失败项；当文件不存在或全部通过时为空列表。
    """
    if not AGENTS.is_file():
        return []
    text = read_text(AGENTS)
    findings: list[Finding] = []
    for skill_name in (
        "graft-codegraph-mcp",
        "graft-ai-governance-audit",
        "graft-ai-plan-governance",
        "graft-work-intake",
        "graft-worktree-manager",
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


def validate_worktree_manager_skill() -> list[Finding]:
    """Validate the reusable worktree manager's required safety boundaries."""
    if not WORKTREE_MANAGER_SKILL.is_file():
        return [Finding(WORKTREE_MANAGER_SKILL, "worktree manager skill is missing")]

    checks = (
        (
            WORKTREE_MANAGER_SKILL,
            "worktree manager",
            (
                "`status`",
                "`acquire`",
                "`release`",
                "`main` is the stable baseline",
                "must not perform the final merge or cherry-pick",
                "Review remains developer-owned and is not an Agent-executable integration operation.",
                "exact integration operation in the current task",
                "primary checkout",
                "final repository authority",
                "auditable",
                "source ref/commit",
                "target workspace",
                "owned scope",
                "before-and-after validation",
                "rollback procedure",
                "invalidation conditions",
                "linear resources",
                "`git worktree remove`",
            ),
        ),
        (
            AGENTS,
            "root worktree integration governance",
            (
                "an explicit integration authorization is an auditable record",
                "source ref/commit",
                "target workspace",
                "owned scope",
                "before-and-after validation",
                "rollback procedure",
                "invalidation conditions",
                "complete integration authorization evidence",
                "actual merge or cherry-pick command differs from the authorized operation record",
            ),
        ),
        (
            AI_TASK_TRACKING_DOC,
            "AI task tracking integration governance",
            (
                "review 是开发者负责的审查活动，不是 Agent 可执行的集成操作",
                "merge 或 cherry-pick 默认不由 Agent 执行",
                "最终仓库状态 authority",
            ),
        ),
    )
    findings: list[Finding] = []
    for path, label, terms in checks:
        if not path.is_file():
            findings.append(Finding(path, f"{label} source is missing"))
            continue
        findings.extend(missing_exact_terms(read_text(path), path, label, terms))
    helper = WORKTREE_MANAGER_SKILL.parent / "scripts" / "worktree_manager.py"
    test = WORKTREE_MANAGER_SKILL.parent / "scripts" / "test_worktree_manager.py"
    for path in (helper, test):
        if not path.is_file():
            findings.append(Finding(path, "worktree manager helper or regression test is missing"))
    return findings


def validate_openapi_worktree_governance() -> list[Finding]:
    """Ensure OpenAPI contract work remains complete and verifiable in task branches."""
    checks = (
        (
            AGENTS,
            (
                "OpenAPI source and its deterministic generated artifacts",
                "agents must generate, validate, and commit",
                "merged canonical OpenAPI source",
                "`just generate`",
                "`just openapi-check`",
            ),
        ),
        (
            AI_CODE_REVIEW_DOC,
            (
                "OpenAPI source 与确定性 generated artifacts",
                "同步生成、验证并提交",
                "合并后的 canonical OpenAPI source",
                "`just generate`",
                "`just openapi-check`",
            ),
        ),
        (
            AI_TOOLING_DOC,
            (
                "自动验证、浏览器检查与人工验收的责任边界",
                "scripts/validate_ai_governance.py",
                "browser evidence 写成默认 closeout gate",
            ),
        ),
        (
            WORKTREE_MANAGER_SKILL,
            (
                "OpenAPI source and its deterministic generated artifacts",
                "validate, and commit the affected source",
                "merged canonical OpenAPI source",
                "`just generate`",
                "`just openapi-check`",
            ),
        ),
    )
    findings: list[Finding] = []
    for path, terms in checks:
        if not path.is_file():
            findings.append(Finding(path, "OpenAPI worktree governance source is missing"))
            continue
        findings.extend(missing_exact_terms(read_text(path), path, "OpenAPI worktree governance", terms))
    return findings


def validate_subagent_model_governance() -> list[Finding]:
    """Ensure direct subagent delegation entrypoints carry the model-level guardrail."""
    findings: list[Finding] = []

    if AGENTS.is_file():
        root_text = read_text(AGENTS)
        for term in (
            "Model-level delegation guardrail:",
            "same level as or lower",
            "fork_context=true",
            "fork_context=false",
            "higher-level worker model",
            "fail closed",
            "model names, availability, or reasoning effort",
        ):
            if term not in root_text:
                findings.append(Finding(AGENTS, f"missing subagent model governance term {term!r}"))

    for skill_path in SUBAGENT_DELEGATION_SKILLS:
        if not skill_path.is_file():
            findings.append(Finding(skill_path, "direct subagent delegation skill is missing"))
            continue
        text = read_text(skill_path)
        if not ("same level as or lower" in text or "same level or lower" in text):
            findings.append(Finding(skill_path, "missing subagent model governance term for same-or-lower model relation"))
        for term in ("model_relation", "comparison evidence"):
            if term not in text:
                findings.append(Finding(skill_path, f"missing subagent model governance term {term!r}"))
        if "availability" not in text:
            findings.append(Finding(skill_path, "missing subagent model governance term 'availability'"))
        if "reasoning" not in text:
            findings.append(Finding(skill_path, "missing subagent model governance term 'reasoning'"))
        if "model=gpt-" in text or "model: gpt-" in text:
            findings.append(Finding(skill_path, "direct delegation skill must not hard-code an unconditional worker model"))

    comment_skill = SUBAGENT_DELEGATION_SKILLS[-1]
    if comment_skill.is_file() and "default worker model is exactly one model level lower" not in read_text(comment_skill):
        findings.append(Finding(comment_skill, "comment governance must define a one-level-lower default worker model"))

    for path, terms in (
        (
            AI_TOOLING_DOC,
            (
                "模型等级护栏",
                "fork_context=true",
                "fork_context=false",
                "不得根据模型名称、可用性或 reasoning effort 推断",
            ),
        ),
        (
            AI_CODE_REVIEW_DOC,
            (
                "多 agent 委派的模型约束",
                "子 agent 模型不得高于直接委派者模型",
                "parent_model",
                "model_relation",
            ),
        ),
    ):
        if not path.is_file():
            findings.append(Finding(path, "AI model delegation governance document is missing"))
            continue
        text = read_text(path)
        for term in terms:
            if term not in text:
                findings.append(Finding(path, f"missing subagent model governance term {term!r}"))

    return findings


def validate_push_branch_governance() -> list[Finding]:
    """
    检查推送触发的分支命名与推送治理说明是否齐备。
    
    遍历 `AGENTS.md` 与推送技能文档中的固定治理术语；缺少任一要求项时返回对应的 `Finding`。
    """
    findings: list[Finding] = []

    if AGENTS.is_file():
        text = read_text(AGENTS)
        for term in (
            "For push-triggered branch-name hygiene:",
            "`$graft-push` in the developer-owned primary checkout must first distinguish an incremental push to an existing open",
            "`@{upstream}..HEAD`",
            "`origin/main..HEAD`",
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
        "git log --oneline origin/main..HEAD",
        "incremental push to an existing open PR",
        "branch-naming input",
        "branch names must follow `<type>/<topic-or-scope>`",
        "lowercase kebab-case",
        "generic `wt-*` placeholders",
        "rename before pushing",
        "do not auto-delete the old remote branch after a rename unless the user explicitly asks",
        "whether a branch-name check ran, what commit range it used, and whether a rename happened",
    ):
        if term not in text:
            findings.append(Finding(PUSH_SKILL, f"missing push branch governance term {term!r}"))

    return findings


def validate_pr_reply_publication_governance() -> list[Finding]:
    """Require PR-review replies to wait for exact remote branch publication."""
    findings: list[Finding] = []
    checks = (
        (
            PR_REVIEW_SKILL,
            (
                "git ls-remote --exit-code origin refs/heads/<current-branch>",
                "require its SHA to equal `git rev-parse HEAD`",
                "leave PR threads and the ledger untouched",
            ),
        ),
        (
            PUSH_SKILL,
            (
                "git ls-remote --exit-code origin refs/heads/<current-branch>",
                "compare the returned SHA with `git rev-parse HEAD`",
                "never reverse this order",
            ),
        ),
    )
    for path, terms in checks:
        if not path.is_file():
            findings.append(Finding(path, "PR reply publication governance skill is missing"))
            continue
        text = read_text(path)
        for term in terms:
            if term not in text:
                findings.append(Finding(path, f"missing PR reply publication governance term {term!r}"))
    return findings


def validate_commit_completion_governance() -> list[Finding]:
    """Ensure a bare graft-commit closes every captured worktree change."""
    checks = (
        (
            AGENTS,
            (
                "a bare `$graft-commit` means commit the complete initial working-tree inventory",
                "complete commit authority for its captured entries",
                "must finish with an empty `git status --short`",
                "committed and the worktree is clean",
            ),
        ),
        (
            COMMIT_SKILL,
            (
                "every commit-eligible tracked and untracked entry",
                "complete commit authority for its captured",
                "must finish with an empty `git status --short`",
                "continue until every captured entry is committed",
                "worktree is clean",
            ),
        ),
    )
    findings: list[Finding] = []
    for path, terms in checks:
        if not path.is_file():
            findings.append(Finding(path, "graft-commit completion governance source is missing"))
            continue
        findings.extend(missing_exact_terms(read_text(path), path, "graft-commit completion governance", terms))
    return findings


def validate_repair_confirmation_interaction_contract() -> list[Finding]:
    """Ensure repair authorization is a structured numbered decision, not a binary prompt."""
    checks = (
        (
            AGENTS,
            (
                "Repair Confirmation Interaction Contract:",
                "Repair required",
                "Failed command:",
                "Root cause:",
                "Changes:",
                "Impact:",
                "Validation:",
                "Commit strategy:",
                "native structured approval",
                "`execute_repair`: Execute repair (recommended)",
                "`continue_current_scope`: Continue current scope only",
                "`show_detailed_diff`: Show detailed diff",
                "`cancel_workflow`: Cancel",
                "Approve?",
                "Should I fix this?",
                "Confirm repair?",
                "approval transport priority is mandatory:",
                "when the runtime supports a native structured-choice interaction, use it",
                "four visible fallback option descriptions",
                "Fallback choices:",
                "`1`: `execute_repair` - Execute repair (recommended)",
                "`2`: `continue_current_scope` - Do not repair",
                "`3`: `show_detailed_diff` - Show the proposed patch",
                "`4`: `cancel_workflow` - Stop the workflow",
                "请输入：",
                "1 / 2 / 3 / 4",
                "numeric fallback is unavailable while native structured approval is available",
            ),
        ),
        (
            REPO_ROOT / ".agents" / "skills" / "graft-commit" / "SKILL.md",
            (
                "Repair Confirmation Interaction Contract",
                "Repair required",
                "native structured approval",
                "next-turn `1 / 2 / 3 / 4` fallback",
                "four visible option descriptions",
                "Only `execute_repair`",
            ),
        ),
        (
            PUSH_SKILL,
            (
                "Repair Confirmation Interaction Contract",
                "Repair required",
                "native structured approval",
                "next-turn `1 / 2 / 3 / 4` fallback",
                "all four visible",
                "Only `execute_repair` authorizes",
            ),
        ),
        (
            AI_CODE_REVIEW_DOC,
            (
                "Repair Confirmation Interaction Contract",
                "Repair required",
                "structured-choice interaction",
                "`execute_repair`",
                "`continue_current_scope`",
                "`show_detailed_diff`",
                "`cancel_workflow`",
                "四个数字选项各自的说明与后果",
                "下一轮用户仅回复 `1 / 2 / 3 / 4`",
                "Approve?",
            ),
        ),
    )
    findings: list[Finding] = []
    for path, terms in checks:
        if not path.is_file():
            findings.append(Finding(path, "repair confirmation governance source is missing"))
            continue
        findings.extend(missing_exact_terms(read_text(path), path, "repair confirmation interaction", terms))
    return findings


def validate_backend_guardrail_governance() -> list[Finding]:
    """
    验证后端治理守护栏文档及其引用是否完整。
    
    检查后端查询、服务端 API、安全、测试可维护性和 AI 代码 Review 相关治理文档是否存在，并验证这些文档及 `AGENTS.md`、`server/AGENTS.md`、AI 工具治理文档中是否包含所需的治理术语。
    
    Returns:
        list[Finding]: 缺失、未引用或内容不符合要求的发现列表。
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


def validate_verification_responsibility_governance() -> list[Finding]:
    """Keep automated verification, browser inspection, and human acceptance distinct."""
    findings: list[Finding] = []
    checks = (
        (
            AGENTS,
            (
                "### 10.5 Verification Responsibility Boundary",
                "`agent-only`",
                "`human-acceptance-required`",
                "`browser-automation-required`",
                "`mixed`",
                "awaiting human acceptance",
                "Local browser interaction is opt-in",
            ),
        ),
        (
            AI_CODE_REVIEW_DOC,
            (
                "## 6. 验证责任与人工验收",
                "`agent-only`",
                "`human-acceptance-required`",
                "`browser-automation-required`",
                "`mixed`",
                "最小人工验收契约",
                "`awaiting_human_acceptance`",
            ),
        ),
        (
            WEB_AGENTS,
            (
                "本地 Agent 默认不启动服务、不操作浏览器、不截图",
                "当前仓库没有正式 CI browser test 基线",
                "inspection evidence，不是 acceptance proof",
            ),
        ),
        (
            VALIDATION_RUNNER_SKILL,
            (
                "Record the verification classification",
                "`agent-only`, `human-acceptance-required`, `browser-automation-required`, or `mixed`",
                "`browser_status`",
                "human_acceptance: awaiting_human_acceptance",
            ),
        ),
        (
            TASK_CLOSEOUT_SKILL,
            (
                "human acceptance contract",
                "Implementation complete; automated verification passed;",
                "awaiting human acceptance.",
                "`verification`",
            ),
        ),
        (
            WEB_BROWSER_SKILL,
            (
                "verification classification",
                "task-local user or developer authorization",
                "Do not use this skill to replace an outstanding human acceptance flow",
            ),
        ),
        (
            WEB_VIBE_CODING_SKILL,
            (
                "verification classification is recorded",
                "minimal human acceptance contract",
            ),
        ),
    )
    for path, terms in checks:
        if not path.is_file():
            findings.append(Finding(path, "verification responsibility governance source is missing"))
            continue
        findings.extend(missing_exact_terms(read_text(path), path, "verification responsibility", terms))

    active_topic = REPO_ROOT / "ai-plan" / "public" / "docker-resource-context-ia"
    active_files = (
        active_topic / "README.md",
        active_topic / "startup-prompt.md",
        active_topic / "todos" / "docker-resource-context-ia-tracking.md",
        active_topic / "traces" / "docker-resource-context-ia-trace.md",
    )
    for path in active_files:
        if not path.is_file():
            findings.append(Finding(path, "active verification-boundary recovery file is missing"))
            continue
        text = read_text(path)
        normalized_text = " ".join(text.split())
        has_conditional_browser_authorization = (
            "conditional browser inspection" in normalized_text
            or "Browser inspection is conditional on" in text
            or "Do not start browser work unless it is explicitly authorized" in text
        )
        has_human_acceptance_handoff = "human acceptance" in text
        if not has_conditional_browser_authorization or not has_human_acceptance_handoff:
            findings.append(Finding(path, "active recovery must not make browser evidence an unconditional closeout gate"))

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


def run_validation() -> list[Finding]:
    """
    汇总并执行所有 AI 治理校验。
    
    Returns:
        list[Finding]: 所有校验发现的问题列表。
    """
    findings: list[Finding] = []
    findings.extend(validate_required_files())
    tracked = tracked_files()
    findings.extend(validate_gitignore())
    findings.extend(validate_ai_tooling_doc())
    findings.extend(validate_skills())
    findings.extend(validate_ai_plan_governance_skill())
    findings.extend(validate_work_intake_skill())
    findings.extend(validate_worktree_manager_skill())
    findings.extend(validate_openapi_worktree_governance())
    findings.extend(validate_skill_mcp_guidance())
    findings.extend(validate_sql_migration_governance())
    findings.extend(validate_shared_asset_governance())
    findings.extend(validate_agents_skill_list())
    findings.extend(validate_subagent_model_governance())
    findings.extend(validate_push_branch_governance())
    findings.extend(validate_pr_reply_publication_governance())
    findings.extend(validate_commit_completion_governance())
    findings.extend(validate_repair_confirmation_interaction_contract())
    findings.extend(validate_backend_guardrail_governance())
    findings.extend(validate_verification_responsibility_governance())
    findings.extend(validate_environment_inventory())
    findings.extend(validate_no_private_config_tracked(tracked))
    findings.extend(validate_no_personal_skill_refs(tracked))
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
