"""Regression tests for the graft-pr-create helper."""

from __future__ import annotations

from contextlib import redirect_stderr
import importlib.util
from io import StringIO
from pathlib import Path
from tempfile import NamedTemporaryFile
import unittest
from unittest.mock import patch


MODULE_PATH = Path(__file__).with_name("ensure_pr.py")
SPEC = importlib.util.spec_from_file_location("ensure_pr", MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
ensure_pr = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(ensure_pr)


class MergeBodyTests(unittest.TestCase):
    def test_merge_body_appends_description_and_metadata_to_existing_user_content(self) -> None:
        merged = ensure_pr.merge_body("User content", "description block", "metadata block")
        self.assertEqual(merged, "User content\n\ndescription block\n\nmetadata block")

    def test_merge_body_replaces_owned_regions_only(self) -> None:
        original = (
            "User intro\n\n"
            f"{ensure_pr.DESCRIPTION_BLOCK_START}\nold description\n{ensure_pr.DESCRIPTION_BLOCK_END}\n\n"
            "Reviewer content\n\n"
            f"{ensure_pr.METADATA_BLOCK_START}\nold metadata\n{ensure_pr.METADATA_BLOCK_END}\n\n"
            "User footer"
        )
        description = f"{ensure_pr.DESCRIPTION_BLOCK_START}\nnew description\n{ensure_pr.DESCRIPTION_BLOCK_END}"
        metadata = f"{ensure_pr.METADATA_BLOCK_START}\nnew metadata\n{ensure_pr.METADATA_BLOCK_END}"
        merged = ensure_pr.merge_body(original, description, metadata)
        self.assertEqual(
            merged,
            "User intro\n\n"
            f"{ensure_pr.DESCRIPTION_BLOCK_START}\nnew description\n{ensure_pr.DESCRIPTION_BLOCK_END}\n\n"
            "Reviewer content\n\n"
            f"{ensure_pr.METADATA_BLOCK_START}\nnew metadata\n{ensure_pr.METADATA_BLOCK_END}\n\n"
            "User footer",
        )

    def test_merge_managed_block_rejects_incomplete_markers(self) -> None:
        with self.assertRaisesRegex(ensure_pr.PrCreateError, "incomplete"):
            ensure_pr.merge_managed_block(
                ensure_pr.DESCRIPTION_BLOCK_START,
                start_marker=ensure_pr.DESCRIPTION_BLOCK_START,
                end_marker=ensure_pr.DESCRIPTION_BLOCK_END,
                managed_block="replacement",
            )


class DescriptionInputTests(unittest.TestCase):
    def test_load_description_file_rejects_empty_content(self) -> None:
        with NamedTemporaryFile(mode="w", encoding="utf-8") as handle:
            handle.write(" \n")
            handle.flush()
            with self.assertRaisesRegex(ensure_pr.PrCreateError, "empty"):
                ensure_pr.load_description_file(handle.name)

    def test_load_description_file_rejects_managed_markers(self) -> None:
        with NamedTemporaryFile(mode="w", encoding="utf-8") as handle:
            handle.write(ensure_pr.DESCRIPTION_BLOCK_START)
            handle.flush()
            with self.assertRaisesRegex(ensure_pr.PrCreateError, "managed markers"):
                ensure_pr.load_description_file(handle.name)

    def test_parse_args_requires_description_file(self) -> None:
        with patch("sys.argv", ["ensure_pr.py"]), redirect_stderr(StringIO()):
            with self.assertRaises(SystemExit) as error:
                ensure_pr.parse_args()
        self.assertEqual(error.exception.code, 2)


class BranchProtectionTests(unittest.TestCase):
    def test_branch_has_protection_requires_matching_rule(self) -> None:
        repo_info = {
            "branch_protection_rules": [
                {
                    "pattern": "main",
                    "requiresStatusChecks": True,
                    "requiresStrictStatusChecks": False,
                    "requiresApprovingReviews": False,
                }
            ]
        }
        self.assertTrue(ensure_pr.branch_has_protection(repo_info, "main"))
        self.assertFalse(ensure_pr.branch_has_protection(repo_info, "develop"))

    def test_branch_has_protection_rejects_non_enforcing_rule(self) -> None:
        repo_info = {
            "branch_protection_rules": [
                {
                    "pattern": "main",
                    "requiresStatusChecks": False,
                    "requiresStrictStatusChecks": False,
                    "requiresApprovingReviews": False,
                }
            ]
        }
        self.assertFalse(ensure_pr.branch_has_protection(repo_info, "main"))


class MergeMethodTests(unittest.TestCase):
    def test_choose_merge_method_prefers_viewer_default_when_allowed(self) -> None:
        repo_info = {
            "viewer_default_merge_method": "MERGE",
            "merge_commit_allowed": True,
            "squash_merge_allowed": True,
            "rebase_merge_allowed": True,
        }
        self.assertEqual(ensure_pr.choose_merge_method(repo_info), "MERGE")

    def test_choose_merge_method_falls_back_when_default_not_allowed(self) -> None:
        repo_info = {
            "viewer_default_merge_method": "MERGE",
            "merge_commit_allowed": False,
            "squash_merge_allowed": True,
            "rebase_merge_allowed": False,
        }
        self.assertEqual(ensure_pr.choose_merge_method(repo_info), "SQUASH")


class ResultRenderingTests(unittest.TestCase):
    def test_stable_managed_diagnostics_drops_ephemeral_creation_message(self) -> None:
        diagnostics = [
            "no open PR matched branch 'feat/test'",
            "base branch 'main' has no detected protection or required checks; auto-merge was not enabled",
        ]
        self.assertEqual(
            ensure_pr.stable_managed_diagnostics(diagnostics),
            ["base branch 'main' has no detected protection or required checks; auto-merge was not enabled"],
        )

    def test_render_managed_block_includes_diagnostics_without_closeout(self) -> None:
        block = ensure_pr.render_managed_block(
            head_branch="feat/test",
            base_branch="main",
            repo_info={"name_with_owner": "GeWuYou/Graft"},
            diagnostics=["needs checks"],
        )
        self.assertIn("repository: GeWuYou/Graft", block)
        self.assertIn("- diagnostics:", block)
        self.assertIn("  - needs checks", block)
        self.assertNotIn("- closeout:", block)

    def test_build_result_preserves_dirty_tree_status(self) -> None:
        result = ensure_pr.build_result(
            action="dry_run",
            pr=None,
            head_branch="feat/test",
            base_branch="main",
            auto_merge="would_enable",
            diagnostics=[],
            status_lines=[" M AGENTS.md"],
        )
        self.assertTrue(result["working_tree_dirty"])
        self.assertEqual(result["working_tree_status"], [" M AGENTS.md"])


if __name__ == "__main__":
    unittest.main()
