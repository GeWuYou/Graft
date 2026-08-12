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

    def test_worktree_manager_governance_rejects_missing_integration_authorization_terms(self) -> None:
        original_read_text = MODULE.read_text
        target = MODULE.WORKTREE_MANAGER_SKILL
        current_text = original_read_text(target)
        mutated_text = current_text.replace("source ref/commit", "source reference", 1)
        self.assertNotEqual(current_text, mutated_text)

        def read_mutated(path: MODULE.Path) -> str:
            if path == target:
                return mutated_text
            return original_read_text(path)

        with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
            findings = MODULE.validate_worktree_manager_skill()

        self.assertTrue(any("source ref/commit" in finding.message for finding in findings))

    def test_worktree_manager_governance_rejects_missing_fail_closed_receipt_rule(self) -> None:
        original_read_text = MODULE.read_text
        target = MODULE.AGENTS
        current_text = original_read_text(target)
        mutated_text = current_text.replace("complete integration authorization evidence", "integration evidence", 1)
        self.assertNotEqual(current_text, mutated_text)

        def read_mutated(path: MODULE.Path) -> str:
            if path == target:
                return mutated_text
            return original_read_text(path)

        with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
            findings = MODULE.validate_worktree_manager_skill()

        self.assertTrue(any("complete integration authorization evidence" in finding.message for finding in findings))

    def test_worktree_manager_governance_rejects_command_authorization_mismatch(self) -> None:
        original_read_text = MODULE.read_text
        target = MODULE.AGENTS
        current_text = original_read_text(target)
        mutated_text = current_text.replace(
            "actual merge or cherry-pick command differs from the authorized operation record",
            "actual integration command is checked",
            1,
        )
        self.assertNotEqual(current_text, mutated_text)

        def read_mutated(path: MODULE.Path) -> str:
            if path == target:
                return mutated_text
            return original_read_text(path)

        with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
            findings = MODULE.validate_worktree_manager_skill()

        self.assertTrue(
            any(
                "actual merge or cherry-pick command differs from the authorized operation record" in finding.message
                for finding in findings
            )
        )

    def test_worktree_manager_governance_checks_root_authority_source(self) -> None:
        original_read_text = MODULE.read_text
        target = MODULE.AGENTS
        current_text = original_read_text(target)
        required_term = "complete integration authorization evidence"
        mutated_text = current_text.replace(required_term, "integration evidence", 1)
        self.assertNotEqual(current_text, mutated_text)

        def read_mutated(path: MODULE.Path) -> str:
            return mutated_text if path == target else original_read_text(path)

        with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
            findings = MODULE.validate_worktree_manager_skill()

        self.assertTrue(any(finding.path == target and required_term in finding.message for finding in findings))

    def test_worktree_manager_governance_checks_tracking_authority_source(self) -> None:
        original_read_text = MODULE.read_text
        target = MODULE.AI_TASK_TRACKING_DOC
        current_text = original_read_text(target)
        required_term = "最终仓库状态 authority"
        mutated_text = current_text.replace(required_term, "仓库状态", 1)
        self.assertNotEqual(current_text, mutated_text)

        def read_mutated(path: MODULE.Path) -> str:
            return mutated_text if path == target else original_read_text(path)

        with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
            findings = MODULE.validate_worktree_manager_skill()

        self.assertTrue(any(finding.path == target and required_term in finding.message for finding in findings))


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


class PRReplyPublicationGovernanceTests(unittest.TestCase):
    def test_pr_reply_publication_governance_is_currently_satisfied(self) -> None:
        self.assertEqual(MODULE.validate_pr_reply_publication_governance(), [])

    def test_pr_reply_publication_governance_rejects_local_commit_as_reply_proof(self) -> None:
        original_read_text = MODULE.read_text
        current_text = original_read_text(MODULE.PR_REVIEW_SKILL)
        mutated_text = current_text.replace("require its SHA to equal `git rev-parse HEAD`", "accept a local commit", 1)
        self.assertNotEqual(current_text, mutated_text)

        def read_mutated(path: MODULE.Path) -> str:
            if path == MODULE.PR_REVIEW_SKILL:
                return mutated_text
            return original_read_text(path)

        with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
            findings = MODULE.validate_pr_reply_publication_governance()

        self.assertTrue(any(finding.path == MODULE.PR_REVIEW_SKILL for finding in findings))


class CommitCompletionGovernanceTests(unittest.TestCase):
    def test_bare_graft_commit_requires_clean_worktree(self) -> None:
        self.assertEqual(MODULE.validate_commit_completion_governance(), [])

    def test_bare_graft_commit_rejects_missing_clean_worktree_rule(self) -> None:
        current_text = MODULE.read_text(MODULE.AGENTS)
        mutated_text = current_text.replace("must finish with an empty `git status --short`", "may stop early", 1)
        self.assertNotEqual(current_text, mutated_text)
        original_read_text = MODULE.read_text

        def read_mutated(path: MODULE.Path) -> str:
            if path == MODULE.AGENTS:
                return mutated_text
            return original_read_text(path)

        with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
            findings = MODULE.validate_commit_completion_governance()

        self.assertTrue(any(finding.path == MODULE.AGENTS for finding in findings))


class RepairConfirmationInteractionTests(unittest.TestCase):
    def test_repair_confirmation_interaction_is_currently_satisfied(self) -> None:
        self.assertEqual(MODULE.validate_repair_confirmation_interaction_contract(), [])

    def test_repair_confirmation_rejects_missing_fallback_option_descriptions(self) -> None:
        current_text = MODULE.read_text(MODULE.AGENTS)
        mutated_text = current_text.replace("Fallback choices:", "manual reply", 1)
        self.assertNotEqual(current_text, mutated_text)
        original_read_text = MODULE.read_text

        def read_mutated(path: MODULE.Path) -> str:
            if path == MODULE.AGENTS:
                return mutated_text
            return original_read_text(path)

        with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
            findings = MODULE.validate_repair_confirmation_interaction_contract()

        self.assertTrue(any(finding.path == MODULE.AGENTS for finding in findings))

    def test_repair_confirmation_rejects_native_approval_bypass(self) -> None:
        current_text = MODULE.read_text(MODULE.AGENTS)
        mutated_text = current_text.replace(
            "numeric fallback is unavailable while native structured approval is available",
            "numeric fallback may replace the native choice control",
            1,
        )
        self.assertNotEqual(current_text, mutated_text)
        original_read_text = MODULE.read_text

        def read_mutated(path: MODULE.Path) -> str:
            if path == MODULE.AGENTS:
                return mutated_text
            return original_read_text(path)

        with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
            findings = MODULE.validate_repair_confirmation_interaction_contract()

        self.assertTrue(any(finding.path == MODULE.AGENTS for finding in findings))


class BackendGuardrailGovernanceTests(unittest.TestCase):
    def test_backend_guardrail_governance_is_currently_satisfied(self) -> None:
        self.assertEqual(MODULE.validate_backend_guardrail_governance(), [])


class VerificationResponsibilityGovernanceTests(unittest.TestCase):
    def test_verification_responsibility_governance_is_currently_satisfied(self) -> None:
        self.assertEqual(MODULE.validate_verification_responsibility_governance(), [])

    def test_verification_responsibility_rejects_default_browser_gate(self) -> None:
        target = MODULE.WEB_AGENTS
        original_read_text = MODULE.read_text
        current_text = original_read_text(target)
        mutated_text = current_text.replace(
            "本地 Agent 默认不启动服务、不操作浏览器、不截图",
            "本地 Agent 默认启动服务并执行浏览器截图",
            1,
        )
        self.assertNotEqual(current_text, mutated_text)

        def read_mutated(path: MODULE.Path) -> str:
            return mutated_text if path == target else original_read_text(path)

        with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
            findings = MODULE.validate_verification_responsibility_governance()

        self.assertTrue(any(finding.path == target for finding in findings))

    def test_verification_responsibility_rejects_missing_human_acceptance_status(self) -> None:
        target = MODULE.AI_CODE_REVIEW_DOC
        original_read_text = MODULE.read_text
        current_text = original_read_text(target)
        mutated_text = current_text.replace("`awaiting_human_acceptance`", "`accepted`", 1)
        self.assertNotEqual(current_text, mutated_text)

        def read_mutated(path: MODULE.Path) -> str:
            return mutated_text if path == target else original_read_text(path)

        with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
            findings = MODULE.validate_verification_responsibility_governance()

        self.assertTrue(any(finding.path == target for finding in findings))


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
