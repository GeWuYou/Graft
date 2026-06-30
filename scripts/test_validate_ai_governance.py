#!/usr/bin/env python3
"""Regression tests for AI governance validation helpers."""

from __future__ import annotations

import importlib.util
from pathlib import Path
import sys
import unittest
from unittest import mock


SCRIPT_PATH = Path(__file__).with_name("validate_ai_governance.py")
MODULE_SPEC = importlib.util.spec_from_file_location("validate_ai_governance", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"Unable to load module from {SCRIPT_PATH}.")

MODULE = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = MODULE
MODULE_SPEC.loader.exec_module(MODULE)


class FrontmatterTests(unittest.TestCase):
    def test_parse_frontmatter_extracts_name_and_description(self) -> None:
        text = "---\nname: graft-example\ndescription: Example skill for testing governance parsing.\n---\n# Body\n"

        metadata = MODULE.parse_frontmatter(text)

        self.assertEqual(metadata["name"], "graft-example")
        self.assertEqual(metadata["description"], "Example skill for testing governance parsing.")

    def test_parse_frontmatter_rejects_missing_block(self) -> None:
        self.assertIsNone(MODULE.parse_frontmatter("# Body only\n"))


class FindingTests(unittest.TestCase):
    def test_finding_formats_repo_relative_path(self) -> None:
        finding = MODULE.Finding(MODULE.REPO_ROOT / "AGENTS.md", "example issue")

        self.assertEqual(finding.format(), "AGENTS.md: example issue")


class SkillMcpGuidanceTests(unittest.TestCase):
    def test_skill_mcp_guidance_is_currently_satisfied(self) -> None:
        self.assertEqual(MODULE.validate_skill_mcp_guidance(), [])


class WorkIntakeGovernanceTests(unittest.TestCase):
    def test_work_intake_governance_is_currently_satisfied(self) -> None:
        self.assertEqual(MODULE.validate_work_intake_skill(), [])


class EnvironmentInventoryTests(unittest.TestCase):
    def test_environment_inventory_covers_adopted_and_pilot_mcp_servers(self) -> None:
        self.assertEqual(MODULE.validate_environment_inventory(), [])

    def test_environment_inventory_covers_eff_u_code_optional_helper(self) -> None:
        self.assertEqual(MODULE.validate_environment_inventory(), [])

    def test_environment_inventory_rejects_overbroad_eff_u_code_manifest_guardrail(self) -> None:
        text = MODULE.read_text(MODULE.TOOLS_AI).replace(
            "Keep eff-u-code as a local helper and raw JSON source; the repository root package.json wrapper and repository-owned evaluator are allowed, but do not add eff-u-code directly to server/go.mod, web/package.json, runtime scripts, deployment flows, or completion gates, and do not use the upstream total score as the gate contract.",
            "Keep eff-u-code developer-local only; do not add it to package manifests, CI, hooks, or completion gates.",
            1,
        )

        with mock.patch.object(MODULE, "read_text", return_value=text):
            findings = MODULE.validate_environment_inventory()

        self.assertTrue(any("package.json wrapper" in finding.message for finding in findings))

    def test_environment_inventory_requires_eff_u_code_repo_wrapper_command(self) -> None:
        text = MODULE.read_text(MODULE.TOOLS_AI).replace(
            'default_command: "bun run quality:eff-u-code --"',
            'default_command: "/root/.bun/bin/fuck-u-code"',
            1,
        )

        with mock.patch.object(MODULE, "read_text", return_value=text):
            findings = MODULE.validate_environment_inventory()

        self.assertTrue(any("bun run quality:eff-u-code --" in finding.message for finding in findings))

    def test_environment_inventory_requires_structured_score_entrypoint(self) -> None:
        text = MODULE.read_text(MODULE.TOOLS_AI).replace(
            'score_entrypoint: "bun run quality:eff-u-code:score:changed"',
            'quality:eff-u-code:score:changed',
            1,
        )

        with mock.patch.object(MODULE, "read_text", return_value=text):
            findings = MODULE.validate_environment_inventory()

        self.assertTrue(any('score_entrypoint: "bun run quality:eff-u-code:score:changed"' in finding.message for finding in findings))

    def test_environment_inventory_rejects_stale_never_gate_rule(self) -> None:
        text = MODULE.read_text(MODULE.TOOLS_AI).replace(
            "Keep eff-u-code as an optional developer-local helper and raw JSON source; when a repository quality gate or CI job is enabled, the repository-owned evaluator owns the gate contract rather than the upstream total score.",
            "Keep eff-u-code as an optional developer-local helper; never treat it as a repository validation gate.",
            1,
        )

        with mock.patch.object(MODULE, "read_text", return_value=text):
            findings = MODULE.validate_environment_inventory()

        self.assertTrue(any("stale" in finding.message for finding in findings))

    def test_environment_inventory_requires_split_repository_gate_role(self) -> None:
        text = MODULE.read_text(MODULE.TOOLS_AI).replace(
            'repository_gate_role: "When quality gating is enabled, the repository-owned evaluator and CI job own the gate contract; eff-u-code remains a raw JSON source only and is not itself the acceptance contract."\n',
            "",
            1,
        )

        with mock.patch.object(MODULE, "read_text", return_value=text):
            findings = MODULE.validate_environment_inventory()

        self.assertTrue(any("repository_gate_role" in finding.message or "split local helper and repository gate roles" in finding.message for finding in findings))


class AiToolingDocTests(unittest.TestCase):
    def test_ai_tooling_doc_requires_explicit_eff_u_code_independence_boundary(self) -> None:
        text = MODULE.read_text(MODULE.AI_TOOLING_DOC).replace(
            "文档 / 治理 Gate 必须独立于 `eff-u-code`：README、ADR、Contract、OpenAPI、Public API Comment 等结构化治理规则属于 `Graft`，不是 `eff-u-code` 的一部分。",
            "README、ADR、Contract、OpenAPI、Public API Comment 等结构化治理规则由仓库维护。",
            1,
        )

        with mock.patch.object(MODULE, "read_text", return_value=text):
            findings = MODULE.validate_ai_tooling_doc()

        self.assertTrue(any("independent from eff-u-code" in finding.message for finding in findings))

    def test_ai_tooling_doc_requires_score_gate_layered_diagnostics(self) -> None:
        text = MODULE.read_text(MODULE.AI_TOOLING_DOC).replace(
            "Project Score Gate 的终端输出应采用 layered diagnostics：默认先给 Scope / Project Score、Threshold、Coverage、Top Contributors、Rule Category Summary、Severity Summary 和 Potential Score Gain，再按 Top N 展开详情；避免直接输出“最差代码排行榜”或全量规则刷屏。",
            "",
            1,
        )

        with mock.patch.object(MODULE, "read_text", return_value=text):
            findings = MODULE.validate_ai_tooling_doc()

        self.assertTrue(any("layered diagnostics" in finding.message for finding in findings))


class PushBranchGovernanceTests(unittest.TestCase):
    def test_push_branch_governance_is_currently_satisfied(self) -> None:
        self.assertEqual(MODULE.validate_push_branch_governance(), [])


class BackendGuardrailGovernanceTests(unittest.TestCase):
    def test_backend_guardrail_governance_is_currently_satisfied(self) -> None:
        self.assertEqual(MODULE.validate_backend_guardrail_governance(), [])


class HeadroomGovernanceTests(unittest.TestCase):
    def test_detects_headroom_rtk_injection_block(self) -> None:
        text = "<!-- headroom:rtk-instructions -->\ncontent\n<!-- /headroom:rtk-instructions -->\n"

        self.assertTrue(MODULE.contains_headroom_rtk_injection(text))

    def test_allows_text_without_headroom_rtk_injection_block(self) -> None:
        text = "Headroom MCP may compress context through explicit tool calls.\n"

        self.assertFalse(MODULE.contains_headroom_rtk_injection(text))

    def test_detects_project_rtk_prefix_rule(self) -> None:
        self.assertTrue(MODULE.contains_project_rtk_prefix_rule("Agents must always prefix with `rtk`.\n"))

    def test_allows_rtk_mentions_without_project_prefix_rule(self) -> None:
        text = "Do not require project agents to use RTK instruction injection.\n"

        self.assertFalse(MODULE.contains_project_rtk_prefix_rule(text))


if __name__ == "__main__":
    unittest.main()
