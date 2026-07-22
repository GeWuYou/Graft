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


class SubagentModelGovernanceTests(unittest.TestCase):
    def test_subagent_model_governance_is_currently_satisfied(self) -> None:
        self.assertEqual(MODULE.validate_subagent_model_governance(), [])

    def test_subagent_model_governance_rejects_fixed_worker_model(self) -> None:
        comment_skill = MODULE.SUBAGENT_DELEGATION_SKILLS[-1]
        current_text = MODULE.read_text(comment_skill)
        mutated_text = current_text + "\nThe target worker configuration is model=gpt-5.6-luna.\n"
        original_read_text = MODULE.read_text

        def read_mutated(path: MODULE.Path) -> str:
            if path == comment_skill:
                return mutated_text
            return original_read_text(path)

        with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
            findings = MODULE.validate_subagent_model_governance()

        self.assertTrue(any("must not hard-code" in finding.message for finding in findings))


class WorkIntakeGovernanceTests(unittest.TestCase):
    def test_work_intake_governance_is_currently_satisfied(self) -> None:
        self.assertEqual(MODULE.validate_work_intake_skill(), [])


class WorktreeManagerGovernanceTests(unittest.TestCase):
    def test_worktree_manager_governance_is_currently_satisfied(self) -> None:
        self.assertEqual(MODULE.validate_worktree_manager_skill(), [])


class OpenApiWorktreeGovernanceTests(unittest.TestCase):
    def test_openapi_worktree_governance_is_currently_satisfied(self) -> None:
        self.assertEqual(MODULE.validate_openapi_worktree_governance(), [])

    def test_openapi_worktree_governance_rejects_deferred_generated_artifacts(self) -> None:
        original_read_text = MODULE.read_text
        targets = (
            (MODULE.AGENTS, "agents must generate, validate, and commit"),
            (MODULE.AI_CODE_REVIEW_DOC, "同步生成、验证并提交"),
            (MODULE.WORKTREE_MANAGER_SKILL, "validate, and commit the affected source"),
        )

        for target, required_term in targets:
            with self.subTest(target=target):
                current_text = original_read_text(target)
                mutated_text = current_text.replace(required_term, "defer generated artifacts", 1)
                self.assertNotEqual(current_text, mutated_text)

                def read_mutated(path: MODULE.Path) -> str:
                    if path == target:
                        return mutated_text
                    return original_read_text(path)

                with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
                    findings = MODULE.validate_openapi_worktree_governance()

                self.assertTrue(any(finding.path == target for finding in findings))


class EnvironmentInventoryTests(unittest.TestCase):
    def test_environment_inventory_covers_adopted_and_pilot_mcp_servers(self) -> None:
        self.assertEqual(MODULE.validate_environment_inventory(), [])


class PersonalSkillReferenceTests(unittest.TestCase):
    def test_personal_skill_references_are_absent_from_repository_guidance(self) -> None:
        self.assertEqual(MODULE.validate_no_personal_skill_refs(MODULE.tracked_files()), [])

    def test_personal_skill_reference_is_rejected(self) -> None:
        with mock.patch.object(
            MODULE,
            "read_text",
            return_value="Use /root/.codex/skills/shutdown-after-completion/SKILL.md or $shutdown-after-completion.",
        ):
            findings = MODULE.validate_no_personal_skill_refs({"AGENTS.md"})

        self.assertEqual(len(findings), 2)
        self.assertTrue(any("personal absolute tooling path" in finding.message for finding in findings))
        self.assertTrue(any("personal skill link" in finding.message for finding in findings))

    def test_guidance_scope_includes_subdomain_agents_without_prefix_collisions(self) -> None:
        tracked = {"server/AGENTS.md", "web/AGENTS.md", ".agentsx/README.md", "server/README.md"}

        with mock.patch.object(MODULE, "read_text", return_value="shutdown.exe /s /t 0"):
            findings = MODULE.validate_no_personal_skill_refs(tracked)

        self.assertEqual({finding.path for finding in findings}, {
            MODULE.REPO_ROOT / "server/AGENTS.md",
            MODULE.REPO_ROOT / "web/AGENTS.md",
        })

    def test_forbidden_patterns_cover_device_commands_and_tool_paths(self) -> None:
        cases = (
            "Run /home/alice/.codex/skills/reboot/SKILL.md",
            "Run C:\\Users\\alice\\.claude\\skills\\shutdown\\SKILL.md",
            "Use systemctl poweroff after completion.",
            "Use Stop-Computer -Force.",
        )

        for text in cases:
            with self.subTest(text=text), mock.patch.object(MODULE, "read_text", return_value=text):
                findings = MODULE.validate_no_personal_skill_refs({"AGENTS.md"})
                self.assertTrue(findings)

    def test_similar_repository_paths_and_documented_commands_are_allowed(self) -> None:
        text = "Use /workspace/.codex/skills/shared/SKILL.md; do not run shutdown now."

        with mock.patch.object(MODULE, "read_text", return_value=text):
            findings = MODULE.validate_no_personal_skill_refs({"AGENTS.md"})

        self.assertEqual(findings, [])


class AiToolingDocTests(unittest.TestCase):
    def test_ai_tooling_doc_is_currently_satisfied(self) -> None:
        self.assertEqual(MODULE.validate_ai_tooling_doc(), [])

    def test_ai_tooling_doc_reports_missing_required_term(self) -> None:
        current_text = MODULE.read_text(MODULE.AI_TOOLING_DOC)
        mutated_text = current_text.replace("headroom_stats", "", 1)

        self.assertNotEqual(current_text, mutated_text)

        with mock.patch.object(MODULE, "read_text", return_value=mutated_text):
            findings = MODULE.validate_ai_tooling_doc()

        self.assertTrue(any("missing AI tooling governance term 'headroom_stats'" == finding.message for finding in findings))


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
