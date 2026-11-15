#!/usr/bin/env python3
"""Regression tests for AI governance validation helpers."""

from __future__ import annotations

import importlib.util
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock


SCRIPT_PATH = Path(__file__).with_name("validate_ai_governance.py")
MODULE_SPEC = importlib.util.spec_from_file_location("validate_ai_governance", SCRIPT_PATH)

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

    def test_parse_frontmatter_rejects_unquoted_colon_space(self) -> None:
        text = "---\nname: graft-example\ndescription: Review changes: compare both authorities.\n---\n"

        with self.assertRaisesRegex(MODULE.FrontmatterError, "unquoted colon-space"):
            MODULE.parse_frontmatter(text)

        trailing_colon = "---\nname: graft-example\ndescription: Review changes:\n---\n"
        with self.assertRaisesRegex(MODULE.FrontmatterError, "unquoted colon-space"):
            MODULE.parse_frontmatter(trailing_colon)

    def test_parse_frontmatter_rejects_yaml_collection_and_alias_indicators(self) -> None:
        for invalid_value in ("[broken", "{broken", "*missing"):
            with self.subTest(invalid_value=invalid_value):
                text = f"---\nname: graft-example\ndescription: {invalid_value}\n---\n"

                with self.assertRaisesRegex(MODULE.FrontmatterError, "reserved YAML indicator"):
                    MODULE.parse_frontmatter(text)

    def test_parse_frontmatter_rejects_implicit_non_string_scalar(self) -> None:
        for invalid_value in ("true", "0xdeadbeef", "2026-08-18"):
            with self.subTest(invalid_value=invalid_value):
                text = f"---\nname: graft-example\ndescription: {invalid_value}\n---\n"

                with self.assertRaisesRegex(MODULE.FrontmatterError, "non-string YAML scalar"):
                    MODULE.parse_frontmatter(text)

    def test_validate_frontmatter_rejects_unknown_schema_field(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_dir:
            skill_dir = Path(temporary_dir) / "graft-example"
            skill_dir.mkdir()
            skill_md = skill_dir / "SKILL.md"
            skill_md.write_text(
                "---\n"
                "name: graft-example\n"
                "description: A sufficiently explicit repository skill description used to test strict schema validation behavior.\n"
                "extra: unsupported\n"
                "---\n",
                encoding="utf-8",
            )

            findings = MODULE.validate_skill_frontmatter(skill_md)

        self.assertTrue(any("unsupported frontmatter fields" in finding.message for finding in findings))


class SkillMetadataTests(unittest.TestCase):
    def test_openai_metadata_requires_interface_mapping(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_dir:
            root = Path(temporary_dir)
            skill_dir = root / "graft-example"
            yaml_path = skill_dir / "agents" / "openai.yaml"
            yaml_path.parent.mkdir(parents=True)
            yaml_path.write_text(
                'display_name: "Example"\n'
                'short_description: "Example description"\n'
                'default_prompt: "Use $graft-example."\n',
                encoding="utf-8",
            )
            with mock.patch.object(MODULE, "REPO_ROOT", root):
                findings = MODULE.validate_openai_yaml(skill_dir, {"graft-example/agents/openai.yaml"})

        self.assertTrue(any("top-level interface" in finding.message for finding in findings))

    def test_openai_metadata_requires_quoted_interface_fields(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_dir:
            root = Path(temporary_dir)
            skill_dir = root / "graft-example"
            yaml_path = skill_dir / "agents" / "openai.yaml"
            yaml_path.parent.mkdir(parents=True)
            yaml_path.write_text(
                "interface:\n"
                "  display_name: Example\n"
                '  short_description: "Example description"\n'
                '  default_prompt: "Use $graft-example."\n',
                encoding="utf-8",
            )
            with mock.patch.object(MODULE, "REPO_ROOT", root):
                findings = MODULE.validate_openai_yaml(skill_dir, {"graft-example/agents/openai.yaml"})

        self.assertTrue(any("quoted interface string" in finding.message for finding in findings))

    def test_openai_metadata_rejects_retired_authority(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_dir:
            root = Path(temporary_dir)
            skill_dir = root / "graft-example"
            yaml_path = skill_dir / "agents" / "openai.yaml"
            yaml_path.parent.mkdir(parents=True)
            yaml_path.write_text(
                "interface:\n"
                '  display_name: "Example"\n'
                '  short_description: "Review the historical server/plugins/example authority."\n'
                '  default_prompt: "Use $graft-example."\n',
                encoding="utf-8",
            )
            with mock.patch.object(MODULE, "REPO_ROOT", root):
                findings = MODULE.validate_openai_yaml(skill_dir, {"graft-example/agents/openai.yaml"})

        self.assertTrue(any("active skill metadata references historical server plugin path" in finding.message for finding in findings))


class SkillAuthorityTests(unittest.TestCase):
    def _validate(self, body: str, extra_files: tuple[str, ...] = ()) -> list[MODULE.Finding]:
        with tempfile.TemporaryDirectory() as temporary_dir:
            root = Path(temporary_dir)
            skill_dir = root / ".agents" / "skills" / "graft-example"
            skill_dir.mkdir(parents=True)
            skill_md = skill_dir / "SKILL.md"
            skill_md.write_text(body, encoding="utf-8")
            for relative_path in extra_files:
                target = skill_dir / relative_path
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_text("reference\n", encoding="utf-8")
            with mock.patch.object(MODULE, "REPO_ROOT", root):
                return MODULE.validate_skill_authority_and_references(skill_md)

    def test_rejects_retired_plugin_authority(self) -> None:
        findings = self._validate("Use server/plugins/example and internal/pluginapi through graft-plugin-scaffold.\n")

        self.assertGreaterEqual(len(findings), 3)

    def test_allows_negative_dynamic_plugin_guidance(self) -> None:
        findings = self._validate("Do not introduce dynamic plugin loading.\n")

        self.assertEqual(findings, [])

    def test_rejects_missing_relative_markdown_reference(self) -> None:
        findings = self._validate("Read [details](references/missing.md).\n")

        self.assertTrue(any("does not exist" in finding.message for finding in findings))

    def test_accepts_existing_relative_markdown_reference(self) -> None:
        findings = self._validate("Read [details](references/details.md).\n", ("references/details.md",))

        self.assertEqual(findings, [])

    def test_rejects_markdown_reference_that_escapes_repository_root(self) -> None:
        findings = self._validate("Read [outside](../../../../outside.md).\n")

        self.assertTrue(any("escapes repository root" in finding.message for finding in findings))

    def test_rejects_markdown_reference_through_external_symlink(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_dir:
            root = Path(temporary_dir) / "repo"
            outside = Path(temporary_dir) / "outside"
            skill_dir = root / ".agents" / "skills" / "graft-example"
            skill_dir.mkdir(parents=True)
            outside.mkdir()
            (outside / "details.md").write_text("outside\n", encoding="utf-8")
            (skill_dir / "references").symlink_to(outside, target_is_directory=True)
            skill_md = skill_dir / "SKILL.md"
            skill_md.write_text("Read [details](references/details.md).\n", encoding="utf-8")

            with mock.patch.object(MODULE, "REPO_ROOT", root):
                findings = MODULE.validate_skill_authority_and_references(skill_md)

        self.assertTrue(any("escapes repository root" in finding.message for finding in findings))

    def test_rejects_private_ip_but_allows_loopback_and_config_placeholder(self) -> None:
        findings = self._validate(
            "Use http://172.21.235.129:3002; loopback http://127.0.0.1:8080 is allowed. "
            "Load [private targets](/.ai/private/graft-browser-targets.yaml).\n"
        )

        self.assertEqual(sum("private network address" in finding.message for finding in findings), 1)

    def test_rfc1918_172_range_uses_exact_network_boundaries(self) -> None:
        findings = self._validate(
            "Public-adjacent 172.15.255.255 and 172.32.0.0 are allowed; "
            "private 172.16.0.0 and 172.31.255.255 are rejected.\n"
        )

        messages = [finding.message for finding in findings if "private network address" in finding.message]
        self.assertEqual(len(messages), 2)
        self.assertTrue(any("172.16.0.0" in message for message in messages))
        self.assertTrue(any("172.31.255.255" in message for message in messages))


class FindingTests(unittest.TestCase):
    def test_finding_formats_repo_relative_path(self) -> None:
        finding = MODULE.Finding(MODULE.REPO_ROOT / "AGENTS.md", "example issue")

        self.assertEqual(finding.format(), "AGENTS.md: example issue")








class EnvironmentInventoryTests(unittest.TestCase):
    def test_environment_inventory_covers_adopted_and_pilot_mcp_servers(self) -> None:
        self.assertEqual(MODULE.validate_environment_inventory(), [])

    def test_environment_inventory_rejects_headroom_without_optional_pilot_adoption(self) -> None:
        original_read_text = MODULE.read_text
        current_text = original_read_text(MODULE.TOOLS_AI)
        mutated_text = current_text.replace('adoption: "optional-pilot"', 'adoption: "adopted"', 1)
        self.assertNotEqual(current_text, mutated_text)

        def read_mutated(path: MODULE.Path) -> str:
            return mutated_text if path == MODULE.TOOLS_AI else original_read_text(path)

        with mock.patch.object(MODULE, "read_text", side_effect=read_mutated):
            findings = MODULE.validate_environment_inventory()

        self.assertTrue(any("missing optional-pilot adoption status" in finding.message for finding in findings))


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

    def test_ai_tooling_doc_rejects_hard_coded_wsl_distro(self) -> None:
        current_text = MODULE.read_text(MODULE.AI_TOOLING_DOC)
        mutated_text = current_text.replace(
            "codex mcp add github -- wsl.exe -- bash -lc",
            "codex mcp add github -- wsl.exe -d Ubuntu-24.04 -- bash -lc",
            1,
        )
        self.assertNotEqual(current_text, mutated_text)

        with mock.patch.object(MODULE, "read_text", return_value=mutated_text):
            findings = MODULE.validate_ai_tooling_doc()

        self.assertTrue(any("must not hard-code a WSL distro" in finding.message for finding in findings))








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


class GuidanceStructureTests(unittest.TestCase):
    def test_short_description_is_valid_but_empty_and_oversized_are_not(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            skill = Path(directory) / 'graft-example' / 'SKILL.md'
            skill.parent.mkdir()
            for description, valid in [('Review API contracts.', True), ('   ', False), ('a' * 1025, False)]:
                import json
                skill.write_text('---\nname: graft-example\ndescription: ' + json.dumps(description) + '\n---\n# Example\n')
                with self.subTest(description=description[:30]):
                    self.assertEqual(not MODULE.validate_skill_frontmatter(skill), valid)

    def test_utf8_bytes_are_measured_instead_of_character_count(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'AGENTS.md').write_text('中' * 4, encoding='utf-8')
            findings = MODULE.validate_guidance_budgets(root, budgets={'AGENTS.md': 10}, chains={'root': ('AGENTS.md',)})
            self.assertTrue(any('12 bytes > 10 bytes' in item.message for item in findings))

    def test_combined_explicit_chain_can_overflow_with_valid_individual_files(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            budgets = {'AGENTS.md': 20000, 'web/AGENTS.md': 20000}
            for relative in budgets:
                path = root / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text('a' * 17000)
            findings = MODULE.validate_guidance_budgets(root, budgets=budgets, chains={'explicit-web': tuple(budgets)})
            self.assertEqual(len(findings), 1)
            self.assertIn('34002 bytes', findings[0].message)
            self.assertIn('web/AGENTS.md=17000 bytes', findings[0].message)

    def test_instruction_chain_boundary_includes_blank_line_separators(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            budgets = {"AGENTS.md": 20000, "web/AGENTS.md": 20000}
            for relative in budgets:
                path = root / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("x" * 16383)
            self.assertEqual(MODULE.validate_guidance_budgets(root, budgets=budgets, chains={"web": tuple(budgets)}), [])
            (root / "AGENTS.md").write_text("x" * 16384)
            findings = MODULE.validate_guidance_budgets(root, budgets=budgets, chains={"web": tuple(budgets)})
            self.assertTrue(any("32769 bytes" in item.message for item in findings))

    def test_canonical_section_requires_content_beyond_heading(self) -> None:
        original = MODULE.read_text
        def heading_only(path: Path) -> str:
            if path == MODULE.AGENTS:
                return "# AGENTS\n\n" + "\n\n".join("## " + heading for heading in MODULE.ROOT_HEADINGS)
            return original(path)
        with mock.patch.object(MODULE, "read_text", side_effect=heading_only):
            findings = MODULE.validate_guidance_owners()
        self.assertEqual(sum("has no content" in item.message for item in findings), len(MODULE.ROOT_HEADINGS))

    def test_missing_instruction_file_is_reported_with_incomplete_chain(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            findings = MODULE.validate_guidance_budgets(root, budgets={'AGENTS.md': 8192}, chains={'root': ('AGENTS.md',)})
            self.assertTrue(any('file is missing' in item.message for item in findings))
            self.assertTrue(any('incomplete' in item.message for item in findings))

    def test_instruction_budget_accepts_exact_boundary(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'AGENTS.md').write_text('a' * 8192)
            self.assertEqual(MODULE.validate_guidance_budgets(root, budgets={'AGENTS.md': 8192}, chains={'root': ('AGENTS.md',)}), [])

    def test_owner_routes_follow_conditional_references_without_prose_duplication(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            skill = root / '.agents/skills/example/SKILL.md'
            detail = skill.parent / 'references/details.md'
            detail.parent.mkdir(parents=True)
            (root / 'AGENTS.md').write_text('# Root\n')
            skill.write_text('[When needed](references/details.md)\n')
            detail.write_text('[Authority](/AGENTS.md)\n')
            with mock.patch.object(MODULE, 'REPO_ROOT', root):
                self.assertEqual(MODULE.require_owner_reference(skill, root / 'AGENTS.md', skill_references=True), [])
                detail.write_text('Root rules are important.\n')
                self.assertTrue(MODULE.require_owner_reference(skill, root / 'AGENTS.md', skill_references=True))

    def test_missing_owner_is_reported_even_when_link_is_present(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / 'SKILL.md'
            source.write_text('[Owner](missing.md)\n')
            with mock.patch.object(MODULE, 'REPO_ROOT', root):
                findings = MODULE.require_owner_reference(source, root / 'missing.md')
            self.assertIn('owner is missing', findings[0].message)

    def test_links_check_explicit_ids_heading_anchors_and_missing_anchors(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / 'SKILL.md'
            (root / 'owner.md').write_text('# Owner\n\n## Repeated\n\n## Repeated\n\n<a id="custom"></a>\n')
            source.write_text('[one](owner.md#repeated-1) [two](owner.md#custom)\n')
            with mock.patch.object(MODULE, 'REPO_ROOT', root):
                self.assertEqual(MODULE.validate_markdown_references(source), [])
                source.write_text('[missing](owner.md#deleted)\n')
                self.assertIn('anchor does not exist', MODULE.validate_markdown_references(source)[0].message)

    def test_conditional_reference_cycles_are_bounded(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            skill = root / 'SKILL.md'
            detail = root / 'reference.md'
            skill.write_text('[details](reference.md)\n')
            detail.write_text('[back](SKILL.md)\n')
            with mock.patch.object(MODULE, 'REPO_ROOT', root):
                self.assertEqual(MODULE.linked_skill_documents(skill), {skill, detail})

    def test_size_report_includes_all_cross_domain_instruction_files(self) -> None:
        report = MODULE.guidance_size_report()
        chain = next(item for item in report['chains'] if item['name'] == 'cross-boundary+docs')
        self.assertEqual(set(chain['paths']), set(MODULE.GUIDANCE_BUDGETS))
        self.assertEqual(chain['bytes'], sum(item['bytes'] for item in report['files']) + 6)


class PluginDiscoveryTests(unittest.TestCase):
    def _fixture(self, root: Path, skill_path: str = './skills/') -> Path:
        import json
        repo_skill = root / '.agents/skills/repository-example/SKILL.md'
        repo_skill.parent.mkdir(parents=True)
        repo_skill.write_text('# Repository\n')
        manifest = root / MODULE.REPOSITORY_PLUGIN_MANIFESTS[0]
        manifest.parent.mkdir(parents=True)
        manifest.write_text(json.dumps({'name': 'graft-frontend-vibe-toolchain', 'skills': skill_path}))
        plugin_skill = manifest.parent.parent / 'skills/plugin-example/SKILL.md'
        plugin_skill.parent.mkdir(parents=True)
        plugin_skill.write_text('# Plugin\n')
        return manifest

    def test_discovers_only_manifest_declared_repository_plugin_skills(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self._fixture(root)
            ignored = root / 'plugins/unregistered/skills/unregistered/SKILL.md'
            ignored.parent.mkdir(parents=True)
            ignored.write_text('# Not registered\n')
            skills, findings = MODULE.discover_skill_files(root)
            self.assertEqual(findings, [])
            self.assertEqual({skill.parent.name for skill in skills}, {'repository-example', 'plugin-example'})

    def test_rejects_escaping_absolute_and_external_plugin_skill_paths(self) -> None:
        for skill_path in ('../../outside', '/tmp/personal-skills', 'https://example.com/skills', 'C:\\Users\\someone\\skills'):
            with self.subTest(skill_path=skill_path), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                self._fixture(root, skill_path)
                _, findings = MODULE.discover_skill_files(root)
                self.assertTrue(findings)

    def test_rejects_plugin_skill_symlink_escape(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / 'repo'
            manifest = self._fixture(root, './external')
            outside = Path(directory) / 'personal-skills'
            outside.mkdir()
            (manifest.parent.parent / 'external').symlink_to(outside, target_is_directory=True)
            _, findings = MODULE.discover_skill_files(root)
            self.assertTrue(any('escapes' in item.message for item in findings))

    def test_repository_skill_root_cannot_point_at_personal_skills(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / "repo"
            root.mkdir()
            (root / ".agents").mkdir()
            outside = Path(directory) / "personal-skills"
            outside.mkdir()
            (root / ".agents/skills").symlink_to(outside, target_is_directory=True)
            roots, findings = MODULE.discover_skill_roots(root)
            self.assertNotIn(root / ".agents/skills", roots)
            self.assertTrue(any("escapes repository root" in item.message for item in findings))

    def test_rejects_missing_skill_frontmatter_entrypoint(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifest = self._fixture(root)
            (manifest.parent.parent / 'skills/plugin-example/SKILL.md').unlink()
            _, findings = MODULE.discover_skill_files(root)
            self.assertTrue(any('missing SKILL.md' in item.message for item in findings))

    def test_current_repository_preserves_48_repository_and_5_plugin_entries(self) -> None:
        skills, findings = MODULE.discover_skill_files()
        self.assertEqual(findings, [])
        self.assertGreaterEqual(sum(skill.is_relative_to(MODULE.SKILLS_DIR) for skill in skills), 48)
        self.assertGreaterEqual(len(skills), 53)


class PRMachineContractTests(unittest.TestCase):
    def test_missing_publication_and_revision_guards_are_reported(self) -> None:
        original_read_text = MODULE.read_text
        def without_safety_flag(path: Path) -> str:
            text = original_read_text(path)
            return text.replace('--resolve-expected-head', '--unguarded-resolve') if path == MODULE.PR_REVIEW_HELPER else text
        with mock.patch.object(MODULE, 'read_text', side_effect=without_safety_flag):
            findings = MODULE.validate_pr_machine_contracts()
        self.assertTrue(any(item.path == MODULE.PR_REVIEW_HELPER and '--resolve-expected-head' in item.message for item in findings))


if __name__ == '__main__':
    unittest.main()
