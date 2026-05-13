#!/usr/bin/env python3
"""Regression tests for the Graft PR review fetch helper."""

from __future__ import annotations

import importlib.util
import os
from pathlib import Path
import unittest
from unittest import mock


SCRIPT_PATH = Path(__file__).with_name("fetch_current_pr_review.py")
MODULE_SPEC = importlib.util.spec_from_file_location("fetch_current_pr_review", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"Unable to load module from {SCRIPT_PATH}.")

MODULE = importlib.util.module_from_spec(MODULE_SPEC)
MODULE_SPEC.loader.exec_module(MODULE)


class ParseFailedTestDetailsTests(unittest.TestCase):
    """Cover failed-test table parsing edge cases for CTRF comments."""

    def test_parse_failed_test_details_ignores_trailing_columns(self) -> None:
        """Extra columns should not prevent extracting the name and failure message."""
        block = """
### ❌ **Some tests failed!**
<table>
  <tbody>
    <tr>
      <td>❌ RegisterMigration_During_Cache_Rebuild_Should_Not_Leave_Stale_Type_Cache</td>
      <td><pre>Expected: False\nBut was: True</pre></td>
      <td>failed</td>
      <td>35.3s</td>
    </tr>
  </tbody>
</table>
"""

        details = MODULE.parse_failed_test_details(block)

        self.assertEqual(
            details,
            [
                {
                    "name": "RegisterMigration_During_Cache_Rebuild_Should_Not_Leave_Stale_Type_Cache",
                    "failure_message": "Expected: False\nBut was: True",
                }
            ],
        )


class ParseLatestReviewBodyTests(unittest.TestCase):
    """Cover folded CodeRabbit review-body parsing for grouped findings."""

    def test_parse_latest_review_body_extracts_outside_diff_and_nitpick_groups(self) -> None:
        """Grouped sections in the latest review body should stay machine-readable."""
        review_body = """
**Actionable comments posted: 2**
<details><summary>Outside diff range comments (1)</summary><blockquote>
<details><summary>server/main.go (1)</summary><blockquote>
`L10-L12`: **Clarify startup flow**
This path hides boot ordering.
</blockquote></details>
</blockquote></details>
<details><summary>Nitpick comments (1)</summary><blockquote>
<details><summary>AGENTS.md (1)</summary><blockquote>
`L1-L2`: **Tighten wording**
This sentence is redundant.
</blockquote></details>
</blockquote></details>
"""

        parsed = MODULE.parse_latest_review_body(review_body)

        self.assertEqual(parsed["actionable_count"], 2)
        self.assertEqual(parsed["outside_diff_count"], 1)
        self.assertEqual(parsed["nitpick_count"], 1)
        self.assertEqual(parsed["outside_diff_comments"][0]["path"], "server/main.go")
        self.assertEqual(parsed["nitpick_comments"][0]["path"], "AGENTS.md")

    def test_parse_latest_review_body_extracts_duplicate_major_and_minor_groups(self) -> None:
        """Additional CodeRabbit severity groups should be parsed from the latest review body."""
        review_body = """
**Actionable comments posted: 3**
<details><summary>♻️ Duplicate comments (1)</summary><blockquote>
<details><summary>server/internal/container/container.go (1)</summary><blockquote>
`L60-L90`: **Reuse existing helper**
This block duplicates in-flight coordination logic.
</blockquote></details>
</blockquote></details>
<details><summary>🟠 Major comments (2)</summary><blockquote>
<details><summary>.github/workflows/pull-request-validation.yml (1)</summary><blockquote>
`L30-L30`: **Use supported GitHub Actions context**
Job-level hashFiles is invalid here.
</blockquote></details>
<details><summary>AGENTS.md (1)</summary><blockquote>
`L87-L90`: **Register the review skill**
The skill list should mention this workflow.
</blockquote></details>
</blockquote></details>
<details><summary>🟡 Minor comments (1)</summary><blockquote>
<details><summary>.agents/skills/graft-pr-review/SKILL.md (1)</summary><blockquote>
`L20-L20`: **Broaden the examples**
Mention additional grouped review sections.
</blockquote></details>
</blockquote></details>
"""

        parsed = MODULE.parse_latest_review_body(review_body)

        self.assertEqual(parsed["duplicate_count"], 1)
        self.assertEqual(parsed["major_count"], 2)
        self.assertEqual(parsed["minor_count"], 1)
        self.assertEqual(parsed["duplicate_comments"][0]["path"], "server/internal/container/container.go")
        self.assertEqual(parsed["major_comments"][0]["path"], ".github/workflows/pull-request-validation.yml")
        self.assertEqual(parsed["minor_comments"][0]["path"], ".agents/skills/graft-pr-review/SKILL.md")
        self.assertEqual(parsed["comment_groups"]["major"]["section_name"], "Major comments")

    def test_parse_latest_review_body_keeps_extensionless_paths(self) -> None:
        """Common extensionless file names should survive grouped comment parsing."""
        review_body = """
**Actionable comments posted: 3**
<details><summary>🟠 Major comments (3)</summary><blockquote>
<details><summary>Dockerfile (1)</summary><blockquote>
`L1-L3`: **Use a pinned base image**
Floating tags make rebuilds non-deterministic.
</blockquote></details>
<details><summary>Makefile (1)</summary><blockquote>
`L5-L5`: **Quote the shell variable**
This target breaks when the path includes spaces.
</blockquote></details>
<details><summary>Justfile (1)</summary><blockquote>
`L9-L10`: **Avoid a duplicated recipe**
This recipe can call the shared helper.
</blockquote></details>
</blockquote></details>
"""

        parsed = MODULE.parse_latest_review_body(review_body)

        self.assertEqual(parsed["major_count"], 3)
        self.assertEqual(
            [comment["path"] for comment in parsed["major_comments"]],
            ["Dockerfile", "Makefile", "Justfile"],
        )


class ResolveGitInvocationTests(unittest.TestCase):
    """Cover explicit repository binding for unusual shell contexts."""

    def test_resolve_git_command_prefers_explicit_override(self) -> None:
        """An explicit git executable override should win over PATH discovery."""
        with mock.patch.dict(
            os.environ,
            {MODULE.GIT_ENVIRONMENT_KEY: "/tmp/custom/git.exe"},
            clear=False,
        ), mock.patch.object(MODULE.os.path, "exists", side_effect=lambda path: path == "/tmp/custom/git.exe"), mock.patch.object(
            MODULE.shutil,
            "which",
            side_effect=lambda name: "/usr/bin/git" if name == "git" else None,
        ):
            self.assertEqual(MODULE.resolve_git_command(), "/tmp/custom/git.exe")

    def test_resolve_git_invocation_prefers_explicit_git_dir_and_work_tree(self) -> None:
        """Configured repository bindings should win over implicit git context."""
        with mock.patch.dict(
            os.environ,
            {
                MODULE.GIT_ENVIRONMENT_KEY: "/tmp/custom/git.exe",
                MODULE.GIT_DIR_ENVIRONMENT_KEY: "/tmp/graft.git",
                MODULE.WORK_TREE_ENVIRONMENT_KEY: "/tmp/graft-worktree",
            },
            clear=False,
        ), mock.patch.object(MODULE.os.path, "exists", side_effect=lambda path: path == "/tmp/custom/git.exe"), mock.patch.object(
            MODULE.shutil,
            "which",
            side_effect=lambda name: "/usr/bin/git" if name == "git" else None,
        ):
            self.assertEqual(
                MODULE.resolve_git_invocation(),
                ["/tmp/custom/git.exe", "--git-dir=/tmp/graft.git", "--work-tree=/tmp/graft-worktree"],
            )

    def test_resolve_git_invocation_supports_git_dir_without_work_tree(self) -> None:
        """A bare git-dir binding should still be applied when no work tree override is needed."""
        with mock.patch.dict(
            os.environ,
            {MODULE.GIT_DIR_ENVIRONMENT_KEY: "/tmp/graft.git"},
            clear=False,
        ), mock.patch.object(
            MODULE.os.path,
            "exists",
            side_effect=lambda path: path == "/usr/bin/git",
        ), mock.patch.object(
            MODULE.shutil,
            "which",
            side_effect=lambda name: "/usr/bin/git" if name == "git" else None,
        ):
            self.assertEqual(MODULE.resolve_git_invocation(), ["/usr/bin/git", "--git-dir=/tmp/graft.git"])


class GithubRequestHeaderTests(unittest.TestCase):
    """Cover optional GitHub token authentication wiring."""

    def test_build_github_request_headers_uses_first_available_token(self) -> None:
        """The helper should prefer its own token env key before generic GitHub keys."""
        with mock.patch.dict(
            os.environ,
            {
                "GRAFT_GITHUB_TOKEN": "repo-token",
                "GITHUB_TOKEN": "generic-token",
                "GH_TOKEN": "cli-token",
            },
            clear=False,
        ):
            self.assertEqual(
                MODULE.build_github_request_headers("application/vnd.github+json"),
                {
                    "Accept": "application/vnd.github+json",
                    "User-Agent": MODULE.USER_AGENT,
                    "Authorization": "Bearer repo-token",
                },
            )

    def test_build_github_request_headers_omits_authorization_when_unconfigured(self) -> None:
        """No Authorization header should be sent when no token environment is configured."""
        with mock.patch.dict(
            os.environ,
            {"GRAFT_GITHUB_TOKEN": "", "GITHUB_TOKEN": "", "GH_TOKEN": ""},
            clear=False,
        ):
            self.assertEqual(
                MODULE.build_github_request_headers("application/vnd.github+json"),
                {
                    "Accept": "application/vnd.github+json",
                    "User-Agent": MODULE.USER_AGENT,
                },
            )


class SelectLatestCoderabbitGroupedReviewTests(unittest.TestCase):
    """Prefer the latest CodeRabbit review that preserves grouped comment sections."""

    def test_select_latest_coderabbit_grouped_review_prefers_grouped_body_over_newer_prompt_only_body(self) -> None:
        """A newer prompt-only review should not hide an older grouped review on the same commit."""
        grouped_review = {
            "id": 1,
            "submitted_at": "2026-05-13T00:10:00Z",
            "user": {"login": MODULE.CODERABBIT_LOGIN},
            "body": "<details><summary>🟠 Major comments (2)</summary><blockquote></blockquote></details>",
        }
        prompt_only_review = {
            "id": 2,
            "submitted_at": "2026-05-13T00:20:00Z",
            "user": {"login": MODULE.CODERABBIT_LOGIN},
            "body": "**Actionable comments posted: 4**",
        }

        selected = MODULE.select_latest_coderabbit_grouped_review([grouped_review, prompt_only_review])

        self.assertEqual(selected, grouped_review)


if __name__ == "__main__":
    unittest.main()
