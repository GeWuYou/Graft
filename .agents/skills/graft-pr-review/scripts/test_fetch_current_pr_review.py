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
                "path": "server/plugins/user/session.go",
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
                "path": "server/plugins/user/session.go",
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
        self.assertEqual(result["open_threads"][0]["path"], "server/plugins/user/session.go")
        self.assertEqual(len(result["all_open_threads"]), 2)
        self.assertEqual(result["all_open_thread_counts_by_user"][MODULE.CODERABBIT_LOGIN], 1)
        self.assertEqual(result["all_open_thread_counts_by_user"][MODULE.GREPTILE_LOGIN], 1)


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


if __name__ == "__main__":
    unittest.main()
