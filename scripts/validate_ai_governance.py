#!/usr/bin/env python3
"""Validate repository AI governance documents and skills stay aligned."""

from __future__ import annotations

import argparse
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

FRONTMATTER_RE = re.compile(r"\A---\n(?P<body>.*?)\n---\n", re.DOTALL)


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
    for pattern in (".codegraph/", ".ai/private/", ".ai/venv/", ".ai/ms-playwright/", ".ai/artifacts/browser/"):
        if pattern not in text:
            findings.append(Finding(GITIGNORE, f"missing ignored local AI artifact pattern {pattern!r}"))
    return findings


def validate_ai_tooling_doc() -> list[Finding]:
    if not AI_TOOLING_DOC.is_file():
        return []
    text = read_text(AI_TOOLING_DOC)
    findings: list[Finding] = []
    required_terms = (
        "codegraph",
        "tdesign",
        "context7",
        "github",
        "playwright",
        "@upstash/context7-mcp",
        "ghcr.io/github/github-mcp-server",
        "@playwright/mcp",
        "memory",
        "postgres",
        "AI tooling evidence",
    )
    for term in required_terms:
        if term not in text:
            findings.append(Finding(AI_TOOLING_DOC, f"missing AI tooling governance term {term!r}"))
    for forbidden in ("server/go.mod`、`web/package.json`、CI", "隐藏恢复真值"):
        if forbidden not in text:
            findings.append(Finding(AI_TOOLING_DOC, f"missing guardrail phrase containing {forbidden!r}"))
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
            ("codex mcp get context7", "codex mcp get github", "codex mcp get playwright"),
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


def validate_environment_inventory() -> list[Finding]:
    if not TOOLS_AI.is_file():
        return []
    text = read_text(TOOLS_AI)
    findings: list[Finding] = []
    for term in ("mcp_servers:", "codegraph:", "tdesign:", "context7:", "github:", "playwright:"):
        if term not in text:
            findings.append(Finding(TOOLS_AI, f"missing AI environment MCP inventory term {term!r}"))
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
    if not AGENTS.is_file():
        return []
    text = read_text(AGENTS)
    findings: list[Finding] = []
    for skill_name in ("graft-codegraph-mcp", "graft-ai-governance-audit", "graft-validation-runner"):
        if skill_name not in text:
            findings.append(Finding(AGENTS, f"repository skill list does not mention {skill_name}"))
    return findings


def validate_no_private_config_tracked(tracked: set[str]) -> list[Finding]:
    findings: list[Finding] = []
    forbidden_prefixes = (
        ".codegraph/",
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
    findings: list[Finding] = []
    findings.extend(validate_required_files())
    tracked = tracked_files()
    findings.extend(validate_gitignore())
    findings.extend(validate_ai_tooling_doc())
    findings.extend(validate_skills())
    findings.extend(validate_skill_mcp_guidance())
    findings.extend(validate_agents_skill_list())
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
