#!/usr/bin/env python3
"""Regression tests for the Graft PR review fetch helper."""

from __future__ import annotations

import argparse
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import unittest
from unittest import mock


SCRIPT_PATH = Path(__file__).with_name("fetch_current_pr_review.py")
MODULE_SPEC = importlib.util.spec_from_file_location("fetch_current_pr_review", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"Unable to load module from {SCRIPT_PATH}.")

MODULE = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = MODULE
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

    def test_parse_latest_review_body_expands_multiple_findings_for_one_path(self) -> None:
        """A folded file card may contain several independently actionable findings."""
        review_body = """
<details><summary>Nitpick comments (2)</summary><blockquote>
<details><summary>server/main.go (2)</summary><blockquote>
`L10-L12`: **First finding**
First description.
<!-- cr-comment:v1:first -->

---

`L20-L22`: **Second finding**
Second description.
<!-- cr-comment:v1:second -->
</blockquote></details>
</blockquote></details>
"""

        parsed = MODULE.parse_latest_review_body(review_body)

        self.assertEqual(parsed["nitpick_count"], 2)
        self.assertEqual(len(parsed["nitpick_comments"]), 2)
        self.assertEqual([item["range"] for item in parsed["nitpick_comments"]], ["L10-L12", "L20-L22"])
        self.assertNotIn("---", parsed["nitpick_comments"][0]["description"])
        self.assertIn("First description.", parsed["nitpick_comments"][0]["description"])
        self.assertNotIn("Second description.", parsed["nitpick_comments"][0]["description"])
        self.assertIn("Second description.", parsed["nitpick_comments"][1]["description"])
        self.assertNotIn("First description.", parsed["nitpick_comments"][1]["description"])


class ParsePreMergeChecksTests(unittest.TestCase):
    """Cover CodeRabbit pre-merge status extraction and handling policy."""

    def test_parse_pre_merge_checks_keeps_warning_and_inconclusive_rows(self) -> None:
        """Warning and inconclusive rows must remain visible even without live CI failures."""
        summary = """
### ❌ Failed checks (1 warning, 1 inconclusive)

| Check name | Status | Explanation | Resolution |
| --- | --- | --- | --- |
| Docstring Coverage | ⚠️ Warning | Docstring coverage is 21.21% which is insufficient. | Write valuable docstrings where required. |
| Title check | ❓ Inconclusive | 标题过于笼统。 | 请改为更具体的标题。 |

<details>
<summary>✅ Passed checks (1 passed)</summary>
"""

        checks = MODULE.parse_pre_merge_checks(summary, source_commit="abc123")

        self.assertEqual([check["status_kind"] for check in checks], ["warning", "inconclusive"])
        self.assertEqual(checks[0]["handling_policy"], "verify-and-decide")
        self.assertEqual(checks[1]["handling_policy"], "verify-and-resolve")
        self.assertEqual(checks[1]["source_commit"], "abc123")
        self.assertEqual(
            MODULE.summarize_pre_merge_checks(checks),
            {"failed": 0, "warning": 1, "inconclusive": 1, "passed": 0, "unknown": 0},
        )

    def test_parse_pre_merge_checks_keeps_passed_rows(self) -> None:
        """Passed rows should remain in the machine-readable inventory."""
        summary = """
### ❌ Failed checks (1 warning)
| Check name | Status | Explanation | Resolution |
| --- | --- | --- | --- |
| Coverage | ⚠️ Warning | 21% | Add docs |
<details>
<summary>✅ Passed checks (1 passed)</summary>
| Check name | Status | Explanation | Resolution |
| --- | --- | --- | --- |
| Title | ✅ Passed | Clear | Keep |
</details>
"""
        checks = MODULE.parse_pre_merge_checks(summary, source_commit="abc123")
        self.assertEqual([check["status_kind"] for check in checks], ["warning", "passed"])

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

    def test_resolve_git_command_prefers_native_git_before_windows_fallback(self) -> None:
        """A usable native git should win over the repository's Windows fallback path."""
        with mock.patch.dict(os.environ, {MODULE.GIT_ENVIRONMENT_KEY: ""}, clear=False), mock.patch.object(
            MODULE.os.path,
            "exists",
            side_effect=lambda path: path in ("/usr/bin/git", MODULE.DEFAULT_WINDOWS_GIT),
        ), mock.patch.object(
            MODULE.shutil,
            "which",
            side_effect=lambda name: "/usr/bin/git" if name == "git" else None,
        ):
            self.assertEqual(MODULE.resolve_git_command(), "/usr/bin/git")

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
        ), mock.patch.object(MODULE.shutil, "which", return_value=None):
            self.assertEqual(
                MODULE.build_github_request_headers("application/vnd.github+json"),
                {
                    "Accept": "application/vnd.github+json",
                    "User-Agent": MODULE.USER_AGENT,
                },
            )

    def test_resolve_github_token_falls_back_to_gh_auth_token(self) -> None:
        """When env vars are empty, gh auth token should become the fallback source."""
        with mock.patch.dict(
            os.environ,
            {"GRAFT_GITHUB_TOKEN": "", "GITHUB_TOKEN": "", "GH_TOKEN": ""},
            clear=False,
        ), mock.patch.object(
            MODULE.shutil,
            "which",
            side_effect=lambda name: "/usr/bin/gh" if name == MODULE.GH_CLI_COMMAND else None,
        ), mock.patch.object(
            MODULE.subprocess,
            "run",
            return_value=subprocess.CompletedProcess(
                args=["gh", "auth", "token"],
                returncode=0,
                stdout="gho_from_gh\n",
                stderr="",
            ),
        ):
            self.assertEqual(MODULE.resolve_github_token(), "gho_from_gh")


class WorkflowCommandTests(unittest.TestCase):
    """Cover local reproduction command extraction from workflow run blocks."""

    def test_select_primary_run_command_prefers_substantive_validation_command(self) -> None:
        """Control-flow setup lines should not hide the real validation command."""
        run_script = """
scanner="scripts/magic_value/check_magic_values.py"
if [ ! -f "$scanner" ]; then
  echo "skip"
  exit 0
fi
python3 "$scanner" --mode ci --output-json /tmp/contract-governance-ci.json
"""

        self.assertEqual(
            MODULE.select_primary_run_command(run_script),
            'python3 "scripts/magic_value/check_magic_values.py" --mode ci --output-json /tmp/contract-governance-ci.json',
        )

    def test_build_local_repro_command_uses_workflow_step_and_working_directory(self) -> None:
        """The helper should derive a local repro command from the repository workflow."""
        command = MODULE.build_local_repro_command("Web Check", "Run unified web validation entrypoint")

        self.assertEqual(command, "cd web && bun run check")


class ReviewThreadStatusTests(unittest.TestCase):
    """Cover conservative status classification for latest review threads."""

    def test_classify_review_thread_status_marks_visible_addressed_text_as_addressed(self) -> None:
        """Visible addressed-in-commit text should close CodeRabbit threads too."""
        latest_comment = {
            "user": MODULE.CODERABBIT_LOGIN,
            "body": "✅ Addressed in commit 4d6e4c5",
        }

        self.assertEqual(MODULE.classify_review_thread_status(latest_comment), "addressed")

    def test_classify_review_thread_status_marks_supported_ai_reviewer_comments_as_open(self) -> None:
        """Supported AI reviewer comments should stay visible until an addressed signal appears."""
        latest_comment = {
            "user": MODULE.GREPTILE_LOGIN,
            "body": "Please simplify this helper.",
        }

        self.assertEqual(MODULE.classify_review_thread_status(latest_comment), "open")

    def test_classify_review_thread_status_marks_github_advanced_security_comments_as_open(self) -> None:
        """GitHub Advanced Security suggestions should stay in the review inventory."""
        latest_comment = {
            "user": MODULE.GITHUB_ADVANCED_SECURITY_LOGIN,
            "body": "This code scanning alert needs review.",
        }

        self.assertEqual(MODULE.classify_review_thread_status(latest_comment), "open")

    def test_classify_review_thread_status_keeps_unknown_for_untracked_human_comments(self) -> None:
        """Untracked reviewer comments still default to unknown without a resolution signal."""
        latest_comment = {
            "user": "reviewer@example",
            "body": "Please simplify this helper.",
        }

        self.assertEqual(MODULE.classify_review_thread_status(latest_comment), "unknown")


class ReplyStateTests(unittest.TestCase):
    """Cover reply-state classification for AI-review threads."""

    def test_classify_reply_state_marks_pending_after_human_reply(self) -> None:
        """A human reply on an open thread should wait for the AI's next reaction."""
        thread = {
            "status": "open",
            "latest_comment": {"user": "developer"},
            "replies": [{"user": "developer"}],
        }

        self.assertEqual(MODULE.classify_reply_state(thread), "pending_ai_followup")

    def test_classify_reply_state_marks_contested_when_ai_replies_again(self) -> None:
        """A reopened disagreement should surface as contested for human follow-up."""
        thread = {
            "status": "open",
            "latest_comment": {"user": MODULE.CODERABBIT_LOGIN},
            "replies": [{"user": "developer"}, {"user": MODULE.CODERABBIT_LOGIN}],
        }

        self.assertEqual(MODULE.classify_reply_state(thread), "contested")

    def test_classify_reply_state_marks_resolved_when_thread_is_closed_after_reply(self) -> None:
        """A replied thread that is no longer open should be treated as resolved."""
        thread = {
            "status": "addressed",
            "latest_comment": {"user": "developer"},
            "replies": [{"user": "developer"}],
        }

        self.assertEqual(MODULE.classify_reply_state(thread), "resolved_after_reply")


class ReviewThreadResolutionTests(unittest.TestCase):
    """Cover guarded GraphQL resolution for supported-AI review threads."""

    def test_find_review_thread_paginates_and_matches_root_comment(self) -> None:
        """The lookup should follow review-thread cursors and match only the root database id."""
        first_page = {
            "repository": {
                "pullRequest": {
                    "headRefOid": "a" * 40,
                    "reviewThreads": {
                        "nodes": [],
                        "pageInfo": {"hasNextPage": True, "endCursor": "next"},
                    },
                }
            }
        }
        second_page = {
            "repository": {
                "pullRequest": {
                    "headRefOid": "a" * 40,
                    "reviewThreads": {
                        "nodes": [
                            {
                                "id": "PRRT_thread",
                                "isResolved": False,
                                "comments": {
                                    "nodes": [
                                        {"databaseId": 42, "author": {"login": "coderabbitai"}},
                                        {"databaseId": 43, "author": {"login": "developer"}},
                                    ]
                                },
                            }
                        ],
                        "pageInfo": {"hasNextPage": False, "endCursor": None},
                    },
                }
            }
        }

        with mock.patch.object(MODULE, "perform_graphql", side_effect=[first_page, second_page]) as graphql:
            thread = MODULE.find_review_thread_by_root_comment(287, 42)

        self.assertEqual(thread["thread_id"], "PRRT_thread")
        self.assertEqual(thread["root_author"], "coderabbitai")
        self.assertEqual(graphql.call_args_list[1].args[1]["cursor"], "next")

    def test_resolution_dry_run_requires_exact_head_and_supported_ai_author(self) -> None:
        """A dry run should validate the target without mutating GitHub."""
        thread = {
            "thread_id": "PRRT_thread",
            "is_resolved": False,
            "root_comment_id": 42,
            "root_author": "coderabbitai",
            "head_sha": "a" * 40,
        }
        with mock.patch.object(MODULE, "resolve_github_token", return_value="token"), mock.patch.object(
            MODULE, "find_review_thread_by_root_comment", return_value=thread
        ), mock.patch.object(MODULE, "perform_graphql") as graphql:
            result = MODULE.perform_review_thread_resolution(287, 42, "a" * 40, dry_run=True)

        self.assertTrue(result["dry_run"])
        self.assertFalse(result["already_resolved"])
        graphql.assert_not_called()

    def test_resolution_executes_mutation_and_requires_confirmation(self) -> None:
        """An authorized resolution should require GitHub to return isResolved true."""
        thread = {
            "thread_id": "PRRT_thread",
            "is_resolved": False,
            "root_comment_id": 42,
            "root_author": MODULE.GREPTILE_LOGIN,
            "head_sha": "b" * 40,
        }
        mutation_result = {
            "resolveReviewThread": {"thread": {"id": "PRRT_thread", "isResolved": True}}
        }
        with mock.patch.object(MODULE, "resolve_github_token", return_value="token"), mock.patch.object(
            MODULE, "find_review_thread_by_root_comment", return_value=thread
        ), mock.patch.object(MODULE, "perform_graphql", return_value=mutation_result) as graphql:
            result = MODULE.perform_review_thread_resolution(287, 42, "b" * 40)

        self.assertTrue(result["resolved"])
        graphql.assert_called_once_with(
            MODULE.RESOLVE_REVIEW_THREAD_MUTATION,
            {"threadId": "PRRT_thread"},
        )

    def test_resolution_rejects_unconfirmed_github_response(self) -> None:
        """GitHub must confirm isResolved before the helper reports success."""
        thread = {
            "thread_id": "PRRT_thread",
            "is_resolved": False,
            "root_comment_id": 42,
            "root_author": MODULE.CODERABBIT_LOGIN,
            "head_sha": "b" * 40,
        }
        unconfirmed = {"resolveReviewThread": {"thread": {"id": "PRRT_thread", "isResolved": False}}}
        with mock.patch.object(MODULE, "resolve_github_token", return_value="token"), mock.patch.object(
            MODULE, "find_review_thread_by_root_comment", return_value=thread
        ), mock.patch.object(MODULE, "perform_graphql", return_value=unconfirmed):
            with self.assertRaisesRegex(RuntimeError, "did not confirm"):
                MODULE.perform_review_thread_resolution(287, 42, "b" * 40)

    def test_resolution_rejects_abbreviated_expected_head_before_lookup(self) -> None:
        """Abbreviated SHAs must fail before any review-thread lookup."""
        with mock.patch.object(MODULE, "resolve_github_token", return_value="token"), mock.patch.object(
            MODULE, "find_review_thread_by_root_comment"
        ) as lookup:
            with self.assertRaisesRegex(RuntimeError, "40-character"):
                MODULE.perform_review_thread_resolution(287, 42, "b" * 8, dry_run=True)

        lookup.assert_not_called()

    def test_resolution_rejects_head_mismatch_and_human_thread(self) -> None:
        """Publication drift and human-authored threads must fail closed."""
        ai_thread = {
            "thread_id": "PRRT_ai",
            "is_resolved": False,
            "root_comment_id": 42,
            "root_author": MODULE.CODERABBIT_LOGIN,
            "head_sha": "c" * 40,
        }
        human_thread = {**ai_thread, "root_author": "maintainer", "head_sha": "d" * 40}
        with mock.patch.object(MODULE, "resolve_github_token", return_value="token"), mock.patch.object(
            MODULE, "find_review_thread_by_root_comment", return_value=ai_thread
        ):
            with self.assertRaisesRegex(RuntimeError, "PR head does not match"):
                MODULE.perform_review_thread_resolution(287, 42, "d" * 40, dry_run=True)
        with mock.patch.object(MODULE, "resolve_github_token", return_value="token"), mock.patch.object(
            MODULE, "find_review_thread_by_root_comment", return_value=human_thread
        ):
            with self.assertRaisesRegex(RuntimeError, "unsupported reviewer"):
                MODULE.perform_review_thread_resolution(287, 42, "d" * 40, dry_run=True)


class BuildAllOpenReviewThreadsTests(unittest.TestCase):
    """Cover PR-wide unresolved AI-thread aggregation beyond the latest commit only."""

    def test_build_all_open_review_threads_keeps_supported_ai_threads_from_older_commits(self) -> None:
        """Older unresolved AI threads should still be surfaced in the all-open view."""
        comments = [
            {
                "id": 1,
                "path": "scripts/magic_value/check_magic_values.py",
                "line": 10,
                "side": "RIGHT",
                "created_at": "2026-05-16T10:00:00Z",
                "updated_at": "2026-05-16T10:00:00Z",
                "user": {"login": MODULE.CODERABBIT_LOGIN},
                "commit_id": "older-commit",
                "in_reply_to_id": None,
                "body": "Still open on an older commit",
            },
            {
                "id": 2,
                "path": "server/modules/user/session.go",
                "line": 20,
                "side": "RIGHT",
                "created_at": "2026-05-16T11:00:00Z",
                "updated_at": "2026-05-16T11:00:00Z",
                "user": {"login": MODULE.GREPTILE_LOGIN},
                "commit_id": "latest-commit",
                "in_reply_to_id": None,
                "body": "Still open on latest commit",
            },
        ]

        threads = MODULE.build_all_open_review_threads(comments)

        self.assertEqual(len(threads), 2)
        self.assertEqual(
            [thread["root_comment"]["user"] for thread in threads],
            [MODULE.CODERABBIT_LOGIN, MODULE.GREPTILE_LOGIN],
        )

    def test_build_all_open_review_threads_excludes_graphql_resolved_threads(self) -> None:
        """Authoritative GraphQL state should remove resolved threads from the open inventory."""
        comments = [
            {
                "id": 7,
                "path": "server/modules/audit/service.go",
                "line": 30,
                "side": "RIGHT",
                "created_at": "2026-05-16T10:00:00Z",
                "updated_at": "2026-05-16T10:00:00Z",
                "user": {"login": MODULE.CODERABBIT_LOGIN},
                "commit_id": "older-commit",
                "in_reply_to_id": None,
                "body": "Old finding",
            }
        ]
        review_thread_index = {
            7: {
                "thread_id": "PRRT_resolved",
                "is_resolved": True,
                "root_comment_id": 7,
                "root_author": MODULE.CODERABBIT_LOGIN,
                "head_sha": "a" * 40,
            }
        }

        threads = MODULE.build_all_open_review_threads(comments, review_thread_index)

        self.assertEqual(threads, [])

    def test_graphql_unresolved_state_reopens_rest_addressed_thread(self) -> None:
        """Authoritative GraphQL false must override a REST addressed marker."""
        comments = [
            {
                "id": 8,
                "path": ".agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py",
                "line": 10,
                "side": "RIGHT",
                "created_at": "2026-08-18T10:00:00Z",
                "updated_at": "2026-08-18T10:00:00Z",
                "user": {"login": MODULE.CODERABBIT_LOGIN},
                "commit_id": "latest-commit",
                "in_reply_to_id": None,
                "body": f"old marker {MODULE.REVIEW_COMMENT_ADDRESSED_MARKER}",
            }
        ]
        review_thread_index = {
            8: {
                "thread_id": "PRRT_open",
                "is_resolved": False,
                "root_comment_id": 8,
                "root_author": MODULE.CODERABBIT_LOGIN,
                "head_sha": "a" * 40,
            }
        }

        threads = MODULE.build_all_open_review_threads(comments, review_thread_index)

        self.assertEqual(len(threads), 1)
        self.assertEqual(threads[0]["status"], "open")
        self.assertEqual(threads[0]["reply_state"], "unreplied")


class FetchLatestCommitReviewTests(unittest.TestCase):
    """Cover latest-commit review payload shape plus PR-wide unresolved thread view."""

    def test_fetch_latest_commit_review_exposes_all_open_threads_beyond_latest_commit(self) -> None:
        """The helper should keep latest-commit and PR-wide open-thread views separate."""
        commits = [
            {"sha": "older-commit", "commit": {"message": "older"}},
            {"sha": "latest-commit", "commit": {"message": "latest"}},
        ]
        reviews = [
            {
                "id": 10,
                "commit_id": "latest-commit",
                "submitted_at": "2026-05-16T12:00:00Z",
                "state": "COMMENTED",
                "user": {"login": MODULE.CODERABBIT_LOGIN},
                "body": "",
            }
        ]
        comments = [
            {
                "id": 1,
                "path": "scripts/magic_value/check_magic_values.py",
                "line": 10,
                "side": "RIGHT",
                "created_at": "2026-05-16T10:00:00Z",
                "updated_at": "2026-05-16T10:00:00Z",
                "user": {"login": MODULE.CODERABBIT_LOGIN},
                "commit_id": "older-commit",
                "in_reply_to_id": None,
                "body": "Older unresolved thread",
            },
            {
                "id": 2,
                "path": "server/modules/user/session.go",
                "line": 20,
                "side": "RIGHT",
                "created_at": "2026-05-16T11:00:00Z",
                "updated_at": "2026-05-16T11:00:00Z",
                "user": {"login": MODULE.GREPTILE_LOGIN},
                "commit_id": "latest-commit",
                "in_reply_to_id": None,
                "body": "Latest unresolved thread",
            },
        ]

        with mock.patch.object(
            MODULE,
            "fetch_paged_json",
            side_effect=[commits, reviews, comments],
        ):
            result = MODULE.fetch_latest_commit_review(12)

        self.assertEqual(result["latest_commit"]["sha"], "latest-commit")
        self.assertEqual(len(result["open_threads"]), 1)
        self.assertEqual(result["open_threads"][0]["path"], "server/modules/user/session.go")
        self.assertEqual(len(result["all_open_threads"]), 2)
        self.assertEqual(result["all_open_thread_counts_by_user"][MODULE.CODERABBIT_LOGIN], 1)
        self.assertEqual(result["all_open_thread_counts_by_user"][MODULE.GREPTILE_LOGIN], 1)

    def test_fetch_latest_commit_review_uses_pr_wide_grouped_coderabbit_review(self) -> None:
        """A newer empty head review should not hide an older grouped CodeRabbit review on the PR."""
        commits = [
            {"sha": "older-commit", "commit": {"message": "older"}},
            {"sha": "latest-commit", "commit": {"message": "latest"}},
        ]
        reviews = [
            {
                "id": 100,
                "commit_id": "older-commit",
                "submitted_at": "2026-07-09T03:39:39Z",
                "state": "CHANGES_REQUESTED",
                "user": {"login": MODULE.CODERABBIT_LOGIN},
                "body": "<details><summary>🧹 Nitpick comments (1)</summary><blockquote></blockquote></details>",
            },
            {
                "id": 101,
                "commit_id": "latest-commit",
                "submitted_at": "2026-07-09T04:10:04Z",
                "state": "APPROVED",
                "user": {"login": MODULE.CODERABBIT_LOGIN},
                "body": "",
            },
        ]

        with mock.patch.object(
            MODULE,
            "fetch_paged_json",
            side_effect=[commits, reviews, []],
        ):
            result = MODULE.fetch_latest_commit_review(136)

        self.assertEqual(result["latest_coderabbit_review_with_body"]["id"], 100)
        self.assertFalse(result["latest_coderabbit_review_with_body"]["is_latest_commit_review"])


class WorkflowChecksTests(unittest.TestCase):
    """Cover live GitHub checks, actions job details, and log fallback handling."""

    def test_fetch_workflow_checks_includes_failed_step_and_repro_command(self) -> None:
        """Failed checks should expose step-level root cause data and a local repro command."""
        payload = {
            "check_runs": [
                {
                    "id": 101,
                    "name": "Web Check",
                    "status": "completed",
                    "conclusion": "failure",
                    "app": {"slug": "github-actions"},
                    "details_url": "https://github.com/GeWuYou/Graft/actions/runs/1/job/2",
                    "html_url": "https://github.com/GeWuYou/Graft/actions/runs/1/job/2",
                }
            ]
        }
        job_payload = {
            "steps": [
                {"name": "Install dependencies", "number": 4, "status": "completed", "conclusion": "success"},
                {
                    "name": "Run unified web validation entrypoint",
                    "number": 5,
                    "status": "completed",
                    "conclusion": "failure",
                },
            ]
        }
        annotations = [{"path": "web/src/foo.ts", "start_line": 12, "message": "type error"}]

        with mock.patch.object(MODULE, "fetch_json", return_value=(payload, {})), mock.patch.object(
            MODULE,
            "fetch_check_run_annotations",
            return_value=annotations,
        ), mock.patch.object(
            MODULE,
            "fetch_actions_job",
            return_value=job_payload,
        ), mock.patch.object(
            MODULE,
            "build_local_repro_command",
            return_value="cd web && bun run check",
        ), mock.patch.object(MODULE, "resolve_github_token", return_value=""):
            result = MODULE.fetch_workflow_checks("abc123")

        self.assertEqual(result["head_sha"], "abc123")
        self.assertEqual(len(result["failed"]), 1)
        self.assertEqual(result["failed"][0]["failed_step"]["name"], "Run unified web validation entrypoint")
        self.assertEqual(result["failed"][0]["local_repro_command"], "cd web && bun run check")
        self.assertEqual(result["failed"][0]["annotations"][0]["message"], "type error")

    def test_fetch_workflow_checks_warns_when_log_download_fails(self) -> None:
        """403-style log failures should degrade into warnings instead of breaking the result."""
        payload = {
            "check_runs": [
                {
                    "id": 101,
                    "name": "Contract Governance Check",
                    "status": "completed",
                    "conclusion": "failure",
                    "app": {"slug": "github-actions"},
                    "details_url": "https://github.com/GeWuYou/Graft/actions/runs/1/job/2",
                    "html_url": "https://github.com/GeWuYou/Graft/actions/runs/1/job/2",
                }
            ]
        }

        with mock.patch.object(MODULE, "fetch_json", return_value=(payload, {})), mock.patch.object(
            MODULE,
            "fetch_check_run_annotations",
            return_value=[],
        ), mock.patch.object(
            MODULE,
            "fetch_actions_job",
            return_value={"steps": []},
        ), mock.patch.object(
            MODULE,
            "build_local_repro_command",
            return_value='python3 "scripts/magic_value/check_magic_values.py" --mode ci --output-json /tmp/contract-governance-ci.json',
        ), mock.patch.object(MODULE, "resolve_github_token", return_value="repo-token"), mock.patch.object(
            MODULE,
            "fetch_job_log_tail",
            side_effect=RuntimeError("HTTP Error 403: Forbidden"),
        ):
            result = MODULE.fetch_workflow_checks("abc123")

        self.assertEqual(len(result["failed"]), 1)
        self.assertIn("Actions logs could not be fetched", result["warnings"][0])


class GithubAdvancedSecurityReportTests(unittest.TestCase):
    """Cover focused GitHub Advanced Security signal extraction."""

    def test_build_github_advanced_security_report_collects_checks_and_threads(self) -> None:
        """Advanced Security checks and review threads should be grouped for inventory closure."""
        workflow_checks = {
            "all": [
                {
                    "name": "CodeQL / Analyze (javascript-typescript)",
                    "status": "completed",
                    "conclusion": "failure",
                    "app": "github-advanced-security",
                    "details_url": "https://github.com/GeWuYou/Graft/security/code-scanning",
                },
                {
                    "name": "Web Check",
                    "status": "completed",
                    "conclusion": "success",
                    "app": "github-actions",
                    "details_url": "https://example.com/web",
                },
            ],
            "failed": [
                {
                    "name": "CodeQL / Analyze (javascript-typescript)",
                    "status": "completed",
                    "conclusion": "failure",
                    "app": "github-advanced-security",
                    "details_url": "https://github.com/GeWuYou/Graft/security/code-scanning",
                }
            ],
        }
        security_thread = {
            "thread_id": 42,
            "path": "web/src/api/client.ts",
            "line": 12,
            "root_comment": {"user": MODULE.GITHUB_ADVANCED_SECURITY_LOGIN, "body": "Sanitize this value."},
            "latest_comment": {"id": 42, "user": MODULE.GITHUB_ADVANCED_SECURITY_LOGIN, "body": "Sanitize this value."},
            "reply_state": "unreplied",
        }
        latest_commit_review = {
            "open_threads": [security_thread],
            "all_open_threads": [security_thread],
        }

        report = MODULE.build_github_advanced_security_report(workflow_checks, latest_commit_review)

        self.assertTrue(report["has_findings"])
        self.assertEqual(len(report["failed_checks"]), 1)
        self.assertEqual(len(report["all_open_threads"]), 1)
        self.assertEqual(report["reviewer_login"], MODULE.GITHUB_ADVANCED_SECURITY_LOGIN)


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


class BuildResultWarningTests(unittest.TestCase):
    """Cover warning decisions that depend on parsed review groups."""

    def test_build_result_does_not_warn_when_grouped_review_has_major_comments(self) -> None:
        """A parsed grouped review should suppress the missing-actionable warning."""
        latest_review_body = """
**Actionable comments posted: 1**
<details><summary>🟠 Major comments (1)</summary><blockquote>
<details><summary>Dockerfile (1)</summary><blockquote>
`L1-L3`: **Pin base image**
Use a fixed tag.
</blockquote></details>
</blockquote></details>
"""
        latest_commit_review = {
            "latest_reviews_by_user": {
                MODULE.CODERABBIT_LOGIN: {
                    "id": 1,
                    "user": MODULE.CODERABBIT_LOGIN,
                    "body": latest_review_body,
                }
            },
            "open_thread_counts_by_user": {},
            "all_open_thread_counts_by_user": {},
            "threads": [],
            "open_threads": [],
            "all_open_threads": [],
            "latest_coderabbit_review_with_body": {
                "id": 1,
                "user": MODULE.CODERABBIT_LOGIN,
                "body": latest_review_body,
            },
        }

        with mock.patch.object(
            MODULE,
            "fetch_pull_request_metadata",
            return_value={
                "number": 1,
                "title": "Test PR",
                "state": "OPEN",
                "head_branch": "feat/test",
                "head_sha": "abc123",
                "base_branch": "main",
                "url": "https://example.com/pr/1",
            },
        ), mock.patch.object(MODULE, "fetch_issue_comments", return_value=[]), mock.patch.object(
            MODULE,
            "fetch_workflow_checks",
            return_value={"head_sha": "abc123", "all": [], "failed": [], "warnings": []},
        ), mock.patch.object(
            MODULE,
            "fetch_latest_commit_review",
            return_value=latest_commit_review,
        ):
            result = MODULE.build_result(1, "feat/test")

        self.assertNotIn(
            "CodeRabbit actionable comments block was not found in issue comments.",
            result["parse_warnings"],
        )

    def test_build_result_prefers_live_workflow_failures_over_missing_coderabbit_failed_checks(self) -> None:
        """Live GitHub checks should populate failed checks even when CodeRabbit has no summary block."""
        with mock.patch.object(
            MODULE,
            "fetch_pull_request_metadata",
            return_value={
                "number": 34,
                "title": "PR",
                "state": "OPEN",
                "head_branch": "feat/test",
                "head_sha": "abc123",
                "base_branch": "main",
                "url": "https://example.com/pr/34",
            },
        ), mock.patch.object(MODULE, "fetch_issue_comments", return_value=[]), mock.patch.object(
            MODULE,
            "fetch_workflow_checks",
            return_value={
                "head_sha": "abc123",
                "all": [],
                "failed": [{"name": "Web Check", "status": "completed", "conclusion": "failure"}],
                "warnings": [],
            },
        ), mock.patch.object(
            MODULE,
            "fetch_latest_commit_review",
            return_value={"threads": [], "latest_reviews_by_user": {}, "open_thread_counts_by_user": {}, "all_open_thread_counts_by_user": {}},
        ):
            result = MODULE.build_result(34, "feat/test")

        self.assertEqual(result["workflow_checks"]["failed"][0]["name"], "Web Check")

    def test_build_result_includes_github_advanced_security_report(self) -> None:
        """The full payload should include a focused Advanced Security inventory section."""
        security_thread = {
            "thread_id": 99,
            "path": "server/main.go",
            "line": 7,
            "root_comment": {"user": MODULE.GITHUB_ADVANCED_SECURITY_LOGIN, "body": "Security suggestion."},
            "latest_comment": {"id": 99, "user": MODULE.GITHUB_ADVANCED_SECURITY_LOGIN, "body": "Security suggestion."},
            "reply_state": "unreplied",
        }
        with mock.patch.object(
            MODULE,
            "fetch_pull_request_metadata",
            return_value={
                "number": 35,
                "title": "PR",
                "state": "OPEN",
                "head_branch": "feat/test",
                "head_sha": "abc123",
                "base_branch": "main",
                "url": "https://example.com/pr/35",
            },
        ), mock.patch.object(MODULE, "fetch_issue_comments", return_value=[]), mock.patch.object(
            MODULE,
            "fetch_workflow_checks",
            return_value={
                "head_sha": "abc123",
                "all": [
                    {
                        "name": "Code scanning results / CodeQL",
                        "status": "completed",
                        "conclusion": "failure",
                        "app": "github-advanced-security",
                    }
                ],
                "failed": [
                    {
                        "name": "Code scanning results / CodeQL",
                        "status": "completed",
                        "conclusion": "failure",
                        "app": "github-advanced-security",
                    }
                ],
                "warnings": [],
            },
        ), mock.patch.object(
            MODULE,
            "fetch_latest_commit_review",
            return_value={
                "threads": [security_thread],
                "open_threads": [security_thread],
                "all_open_threads": [security_thread],
                "latest_reviews_by_user": {},
                "open_thread_counts_by_user": {},
                "all_open_thread_counts_by_user": {MODULE.GITHUB_ADVANCED_SECURITY_LOGIN: 1},
            },
        ):
            result = MODULE.build_result(35, "feat/test")

        self.assertTrue(result["github_advanced_security"]["has_findings"])
        self.assertEqual(len(result["github_advanced_security"]["failed_checks"]), 1)
        self.assertEqual(len(result["github_advanced_security"]["all_open_threads"]), 1)


class MainOutputTests(unittest.TestCase):
    """Cover CLI output semantics for JSON and file-output combinations."""

    def test_format_text_shows_pre_merge_checks_separately_from_live_failures(self) -> None:
        """CodeRabbit inconclusive checks must be actionable when live CI is green."""
        output = MODULE.format_text(
            {
                "pull_request": {
                    "number": 184,
                    "title": "Feature/cross boundary contract projection",
                    "state": "OPEN",
                    "head_branch": "feature/test",
                    "base_branch": "main",
                    "head_sha": "c694815b",
                    "url": "https://example.com/pr/184",
                },
                "workflow_checks": {"failed": []},
                "managed_review_ledger": {
                    "comment_id": 1,
                    "incremental": {
                        "baseline_head_sha": "72638ae8",
                        "current_head_sha": "c694815b",
                        "must_rebuild_inventory": True,
                        "reason": "PR head changed",
                    },
                },
                "coderabbit_summary": {
                    "pre_merge_checks": [
                        {
                            "name": "Title check",
                            "status": "❓ Inconclusive",
                            "status_kind": "inconclusive",
                            "handling_policy": "verify-and-resolve",
                            "explanation": "标题过于笼统。",
                            "resolution": "请改为更具体的标题。",
                            "source_commit": "c694815b",
                        }
                    ],
                    "pre_merge_check_counts": {
                        "failed": 0,
                        "warning": 0,
                        "inconclusive": 1,
                        "passed": 0,
                    },
                },
                "coderabbit_comments": {},
                "coderabbit_review": {
                    "comment_groups": {"outside-diff": {"count": 1}},
                },
                "latest_commit_review": {},
                "parse_warnings": [],
            },
            sections=["pr", "failed-checks", "pre-merge-checks", "open-threads"],
        )

        self.assertIn("Failed checks: 0", output)
        self.assertIn("inspect --section pre-merge-checks before closeout", output)
        self.assertIn("Folded CodeRabbit review sections are present", output)
        self.assertIn("Review ledger action: rebuild the complete finding inventory", output)
        self.assertIn("CodeRabbit pre-merge checks: 1 total", output)
        self.assertIn("kind=inconclusive policy=verify-and-resolve", output)
        self.assertIn("请改为更具体的标题。", output)

    def test_main_prints_json_to_stdout_even_when_json_output_is_requested(self) -> None:
        """JSON mode should keep stdout machine-readable while still writing the file."""
        args = argparse.Namespace(
            branch="feat/mvp-extension-path",
            pr=1,
            format="json",
            json_output="/tmp/pr-review.json",
            section=None,
            path=None,
            max_description_length=400,
            reply_comment_id=None,
            reply_body=None,
            reply_body_file=None,
            reply_fixed_commit=None,
            reply_fixed_path=None,
            reply_dry_run=False,
            ledger_body=None,
            ledger_body_file=None,
            ledger_marker=MODULE.PR_REVIEW_LEDGER_MARKER,
            ledger_dry_run=False,
        )
        result = {"pull_request": {"number": 1}, "parse_warnings": []}

        with mock.patch.object(MODULE, "parse_args", return_value=args), mock.patch.object(
            MODULE,
            "build_result",
            return_value=result,
        ), mock.patch.object(
            MODULE,
            "write_json_output",
            return_value="/tmp/pr-review.json",
        ) as write_json_output, mock.patch.object(MODULE, "print") as print_mock:
            MODULE.main()

        write_json_output.assert_called_once_with(result, "/tmp/pr-review.json")
        print_mock.assert_called_once_with(json.dumps(result, ensure_ascii=False, indent=2))


class ReviewReplyTests(unittest.TestCase):
    """Cover reply CLI safety and dry-run behavior."""

    def test_perform_review_reply_requires_token(self) -> None:
        """Replying without a configured GitHub token should fail closed."""
        with mock.patch.object(MODULE, "resolve_github_token", return_value=""):
            with self.assertRaisesRegex(RuntimeError, "GitHub token"):
                MODULE.perform_review_reply(1, 123, "noise")

    def test_perform_review_reply_supports_dry_run(self) -> None:
        """Dry-run reply mode should return the payload without calling GitHub."""
        with mock.patch.object(MODULE, "resolve_github_token", return_value="repo-token"), mock.patch.object(
            MODULE,
            "post_json",
        ) as post_json:
            result = MODULE.perform_review_reply(1, 123, "noise", dry_run=True)

        post_json.assert_not_called()
        self.assertTrue(result["dry_run"])
        self.assertEqual(result["request_payload"]["body"], "noise")


class ManagedIssueCommentTests(unittest.TestCase):
    """Cover append-only PR review ledger issue-comment management."""

    VALID_ENTRY_BODY = """- `coderabbit_handled`: fixed 1, delegated 0, blocked 0, stale 0, noise 0.
- `coderabbit_outside_diff_range`: declared 0, handled 0.
- `coderabbit_nitpick`: declared 0, handled 0.
- `coderabbit_pre_merge_checks`: total 2, warning 1, inconclusive 1, passed 0, handled 2.
- `open_suggestions`: 0 unresolved, 0 remaining.
- `greptile_suggestions`: 0 verified.
"""

    def test_parse_latest_ledger_run_extracts_head_and_inventory(self) -> None:
        """The second review round must recover the previous head from the managed ledger."""
        body = f"""{MODULE.PR_REVIEW_LEDGER_MARKER}
# Graft PR Review Ledger

## Run 2026-07-21T06:10:06Z
- Head SHA: 72638ae80cdfd269bdd353bb78fd8c671949522f
- `coderabbit_outside_diff_range`: declared=0, handled=0.
- `coderabbit_nitpick`: declared=7, handled=7.
"""

        parsed = MODULE.parse_latest_ledger_run(body)

        self.assertEqual(parsed["head_sha"], "72638ae80cdfd269bdd353bb78fd8c671949522f")
        self.assertEqual(parsed["inventory"]["coderabbit_nitpick"]["declared"], 7)

    def test_build_review_ledger_delta_requires_rebuild_after_new_head(self) -> None:
        """A changed PR head must never reuse the previous ledger as review closure."""
        ledger = {
            "comment_id": 1,
            "latest_run": {
                "head_sha": "72638ae8",
                "inventory": {"coderabbit_outside_diff_range": {"declared": 0}},
            },
        }
        delta = MODULE.build_review_ledger_delta(
            ledger,
            current_head_sha="c694815b",
            coderabbit_review={
                "comment_groups": {"outside-diff": {"count": 1}},
            },
            pre_merge_checks=[
                {"status_kind": "warning"},
                {"status_kind": "inconclusive"},
            ],
        )

        self.assertTrue(delta["head_changed"])
        self.assertTrue(delta["must_rebuild_inventory"])
        self.assertEqual(delta["new_group_counts"], {"outside-diff": 1})

    def test_validate_review_ledger_body_rejects_literal_escapes_and_missing_fields(self) -> None:
        """Ledger input must contain real newlines and the required inventory summary."""
        with self.assertRaisesRegex(RuntimeError, "literal"):
            MODULE.validate_review_ledger_body("coderabbit_handled: 1\\n")
        with self.assertRaisesRegex(RuntimeError, "missing required inventory fields"):
            MODULE.validate_review_ledger_body("coderabbit_handled: 1\n")

    def test_validate_managed_ledger_document_requires_marker_and_run(self) -> None:
        """The final managed comment must have its marker, title, and run heading."""
        document = (
            f"{MODULE.PR_REVIEW_LEDGER_MARKER}\n\n# Graft PR Review Ledger\n\n"
            "## Run 2026-07-13T00:00:00Z\n\n"
            f"{self.VALID_ENTRY_BODY}"
        )
        self.assertEqual(MODULE.validate_managed_ledger_document(document), document.strip())

    def test_managed_sync_rejects_invalid_entry_before_github_write(self) -> None:
        """Invalid final ledger content must fail before any GitHub API write."""
        result = {
            "pull_request": {"number": 136, "head_branch": "feat/test", "head_sha": "abc123"},
        }
        with mock.patch.object(MODULE, "resolve_github_token", return_value="repo-token"), mock.patch.object(
            MODULE,
            "post_json",
        ) as post_json:
            with self.assertRaisesRegex(RuntimeError, "literal"):
                MODULE.perform_managed_issue_comment_append(136, [], result, "bad\\ncontent")

        post_json.assert_not_called()

    def test_build_managed_issue_comment_body_normalizes_legacy_escaped_newlines(self) -> None:
        """Legacy literal newline escapes are repaired before the final payload is validated."""
        existing = f"{MODULE.PR_REVIEW_LEDGER_MARKER}\\n\\n# Graft PR Review Ledger\\n\\nExisting entry"
        result = MODULE.build_managed_issue_comment_body(
            existing,
            MODULE.build_review_ledger_entry(
                {"pull_request": {"number": 136, "head_branch": "feat/test", "head_sha": "abc123"}},
                self.VALID_ENTRY_BODY,
            ),
        )
        self.assertNotIn("\\n", result)
        self.assertIn("# Graft PR Review Ledger\n", result)

    def test_perform_managed_issue_comment_append_supports_create_dry_run(self) -> None:
        """Dry-run ledger sync should build a create payload when no managed comment exists."""
        result = {
            "pull_request": {"number": 136, "head_branch": "feat/test", "head_sha": "abc123"},
        }

        with mock.patch.object(MODULE, "resolve_github_token", return_value="repo-token"):
            action = MODULE.perform_managed_issue_comment_append(
                136,
                [],
                result,
                self.VALID_ENTRY_BODY,
                dry_run=True,
            )

        self.assertTrue(action["dry_run"])
        self.assertEqual(action["operation"], "create")
        self.assertIn(MODULE.PR_REVIEW_LEDGER_MARKER, action["request_payload"]["body"])
        self.assertIn("coderabbit_handled", action["request_payload"]["body"])
        self.assertEqual(action["baseline_revision"], "absent")
        self.assertTrue(action["entry_heading"].startswith("## Run sha256:"))

    def test_perform_managed_issue_comment_append_supports_update_dry_run(self) -> None:
        """Dry-run ledger sync should append to the existing managed comment."""
        result = {
            "pull_request": {"number": 136, "head_branch": "feat/test", "head_sha": "abc123"},
        }
        existing_comments = [
            {
                "id": 55,
                "body": f"{MODULE.PR_REVIEW_LEDGER_MARKER}\n\n# Graft PR Review Ledger\n\nExisting entry",
                "updated_at": "2026-07-09T04:00:00Z",
                "created_at": "2026-07-09T04:00:00Z",
            }
        ]

        with mock.patch.object(MODULE, "resolve_github_token", return_value="repo-token"):
            action = MODULE.perform_managed_issue_comment_append(
                136,
                existing_comments,
                result,
                self.VALID_ENTRY_BODY,
                dry_run=True,
            )

        self.assertTrue(action["dry_run"])
        self.assertEqual(action["operation"], "update")
        self.assertEqual(action["comment_id"], 55)
        self.assertIn("Existing entry", action["request_payload"]["body"])
        self.assertIn("coderabbit_nitpick", action["request_payload"]["body"])

    def test_managed_ledger_entry_is_deterministic_for_dry_run_and_write(self) -> None:
        """The same validated body and PR head must produce one stable idempotency key."""
        result = {
            "pull_request": {"number": 136, "head_branch": "fix/test", "head_sha": "a" * 40},
        }

        first = MODULE.build_review_ledger_entry(result, self.VALID_ENTRY_BODY)
        second = MODULE.build_review_ledger_entry(result, self.VALID_ENTRY_BODY)

        self.assertEqual(first, second)
        self.assertEqual(first.count("## Run sha256:"), 1)

    def test_managed_ledger_write_rejects_stale_preview_revision(self) -> None:
        """A live append must not rebuild a different payload from a stale preview snapshot."""
        expected_head = "b" * 40
        result = {
            "pull_request": {"number": 136, "head_branch": "fix/test", "head_sha": expected_head},
        }
        existing_comments = [
            {
                "id": 55,
                "body": f"{MODULE.PR_REVIEW_LEDGER_MARKER}\n\n# Graft PR Review Ledger\n\n## Run old",
                "updated_at": "2026-08-18T10:00:00Z",
                "created_at": "2026-08-18T10:00:00Z",
            }
        ]
        with mock.patch.object(MODULE, "resolve_github_token", return_value="repo-token"), mock.patch.object(
            MODULE, "patch_json"
        ) as patch_json:
            with self.assertRaisesRegex(RuntimeError, "revision does not match"):
                MODULE.perform_managed_issue_comment_append(
                    136,
                    existing_comments,
                    result,
                    self.VALID_ENTRY_BODY,
                    expected_head=expected_head,
                    expected_revision="c" * 64,
                )

        patch_json.assert_not_called()

    def test_managed_ledger_update_rejects_change_after_validated_snapshot(self) -> None:
        """A second revision check must reject a race immediately before PATCH."""
        expected_head = "c" * 40
        result = {
            "pull_request": {"number": 136, "head_branch": "fix/test", "head_sha": expected_head},
        }
        existing_comments = [
            {
                "id": 55,
                "body": f"{MODULE.PR_REVIEW_LEDGER_MARKER}\n\n# Graft PR Review Ledger\n\n## Run old",
                "updated_at": "2026-08-18T10:00:00Z",
                "created_at": "2026-08-18T10:00:00Z",
            }
        ]
        expected_revision = MODULE.managed_ledger_revision(existing_comments[0]["body"])
        with mock.patch.object(MODULE, "resolve_github_token", return_value="repo-token"), mock.patch.object(
            MODULE,
            "fetch_issue_comment_snapshot",
            return_value={"revision": "d" * 64},
        ), mock.patch.object(MODULE, "patch_json") as patch_json:
            with self.assertRaisesRegex(RuntimeError, "changed after the validated snapshot"):
                MODULE.perform_managed_issue_comment_append(
                    136,
                    existing_comments,
                    result,
                    self.VALID_ENTRY_BODY,
                    expected_head=expected_head,
                    expected_revision=expected_revision,
                )

        patch_json.assert_not_called()

    def test_managed_ledger_write_requires_full_expected_head(self) -> None:
        """A non-dry-run ledger append must carry exact PR-head publication proof."""
        result = {
            "pull_request": {"number": 136, "head_branch": "feat/test", "head_sha": "a" * 40},
        }
        with mock.patch.object(MODULE, "resolve_github_token", return_value="repo-token"), mock.patch.object(
            MODULE, "post_json"
        ) as post_json:
            with self.assertRaisesRegex(RuntimeError, "ledger-expected-head"):
                MODULE.perform_managed_issue_comment_append(136, [], result, self.VALID_ENTRY_BODY)

        post_json.assert_not_called()

    def test_managed_ledger_write_requires_preview_revision(self) -> None:
        """A non-dry-run append must carry the revision emitted by its preview."""
        expected_head = "a" * 40
        result = {
            "pull_request": {"number": 136, "head_branch": "fix/test", "head_sha": expected_head},
        }
        with mock.patch.object(MODULE, "resolve_github_token", return_value="repo-token"), mock.patch.object(
            MODULE, "post_json"
        ) as post_json:
            with self.assertRaisesRegex(RuntimeError, "ledger-expected-revision"):
                MODULE.perform_managed_issue_comment_append(
                    136,
                    [],
                    result,
                    self.VALID_ENTRY_BODY,
                    expected_head=expected_head,
                )

        post_json.assert_not_called()

    def test_managed_ledger_dry_run_rejects_pr_head_mismatch(self) -> None:
        """A supplied expected head must match even when previewing the ledger payload."""
        result = {
            "pull_request": {"number": 136, "head_branch": "feat/test", "head_sha": "a" * 40},
        }
        with mock.patch.object(MODULE, "resolve_github_token", return_value="repo-token"):
            with self.assertRaisesRegex(RuntimeError, "PR head does not match"):
                MODULE.perform_managed_issue_comment_append(
                    136,
                    [],
                    result,
                    self.VALID_ENTRY_BODY,
                    expected_head="b" * 40,
                    dry_run=True,
                )

    def test_managed_ledger_write_uses_verified_expected_head(self) -> None:
        """A matching full head should permit the append and remain visible in action evidence."""
        expected_head = "c" * 40
        result = {
            "pull_request": {"number": 136, "head_branch": "feat/test", "head_sha": expected_head},
        }
        response = {
            "id": 99,
            "html_url": "https://example.test/comment/99",
            "body": "ledger",
            "user": {"login": "developer"},
        }
        with mock.patch.object(MODULE, "resolve_github_token", return_value="repo-token"), mock.patch.object(
            MODULE, "fetch_issue_comments", return_value=[]
        ), mock.patch.object(MODULE, "post_json", return_value=(response, {})) as post_json, mock.patch.object(
            MODULE, "verify_managed_ledger_append", return_value=response
        ) as verify_append:
            action = MODULE.perform_managed_issue_comment_append(
                136,
                [],
                result,
                self.VALID_ENTRY_BODY,
                expected_head=expected_head,
                expected_revision="absent",
            )

        self.assertFalse(action["dry_run"])
        self.assertEqual(action["expected_head"], expected_head)
        self.assertEqual(action["expected_revision"], "absent")
        post_json.assert_called_once()
        verify_append.assert_called_once()

    def test_verify_managed_ledger_append_requires_one_exact_entry_and_body(self) -> None:
        """Post-write verification must reject duplicates and payload drift."""
        entry_heading = "## Run sha256:target"
        request_body = (
            f"{MODULE.PR_REVIEW_LEDGER_MARKER}\n\n# Graft PR Review Ledger\n\n{entry_heading}"
        )
        persisted = {"id": 99, "body": request_body}
        with mock.patch.object(MODULE, "fetch_issue_comments", return_value=[persisted]):
            result = MODULE.verify_managed_ledger_append(136, 99, request_body, entry_heading)

        self.assertEqual(result["id"], 99)
        duplicate_comments = [persisted, {"id": 100, "body": request_body}]
        with mock.patch.object(MODULE, "fetch_issue_comments", return_value=duplicate_comments):
            with self.assertRaisesRegex(RuntimeError, "exactly once"):
                MODULE.verify_managed_ledger_append(136, 99, request_body, entry_heading)
        with mock.patch.object(
            MODULE,
            "fetch_issue_comments",
            return_value=[{"id": 99, "body": f"{request_body}\nextra"}],
        ):
            with self.assertRaisesRegex(RuntimeError, "differs"):
                MODULE.verify_managed_ledger_append(136, 99, request_body, entry_heading)


if __name__ == "__main__":
    unittest.main()
