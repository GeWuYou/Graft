#!/usr/bin/env python3
"""Focused tests for graft-multi-agent-loop."""

from __future__ import annotations

import argparse
import subprocess
import sys
import tempfile
import textwrap
import unittest
from pathlib import Path
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parent))
import run_loop


class ParseCloseoutMessageTests(unittest.TestCase):
    def setUp(self) -> None:
        self.limits = run_loop.BudgetLimits(
            max_rounds=5,
            max_files_changed=30,
            max_commits=3,
            max_runtime_minutes=90,
            validation_mode="task-required",
            stop_on_validation_failure=True,
            require_clean_worktree=True,
            scope_expansion_policy="forbid",
            allowed_scopes=["docs", "automation"],
        )
        self.aggregate = run_loop.BudgetUsage()

    def test_parse_json_closeout(self) -> None:
        message = textwrap.dedent(
            """
            closeout
            Next-session startup prompt: 下一轮继续清理追踪文档
            ```json
            {
              "closeout_status": "handoff_only",
              "continue": true,
              "next_prompt": "下一轮继续清理追踪文档",
              "stop_reason": null,
              "validation": {"status": "passed", "commands": ["python3 -m unittest"], "note": null},
              "commit": {"created": false, "sha": null, "title": null},
              "consumed_budget": {"rounds": 1, "files_changed": 2, "commits": 0, "runtime_minutes": 7},
              "remaining_budget": {"rounds": 4, "files_changed": 28, "commits": 3, "runtime_minutes": 83},
              "scope_expanded": false,
              "risk_level": "low"
            }
            ```
            """
        ).strip()
        closeout = run_loop.parse_closeout_message(
            message=message,
            default_runtime_minutes=7,
            aggregate=self.aggregate,
            limits=self.limits,
        )
        self.assertTrue(closeout.continue_loop)
        self.assertEqual(closeout.next_prompt, "下一轮继续清理追踪文档")
        self.assertEqual(closeout.validation.status, "passed")
        self.assertEqual(closeout.consumed_budget.files_changed, 2)
        self.assertEqual(closeout.parse_mode, "json")

    def test_keyword_fallback_without_json(self) -> None:
        message = "Closeout\nNext-session startup prompt: 下一轮继续"
        closeout = run_loop.parse_closeout_message(
            message=message,
            default_runtime_minutes=5,
            aggregate=self.aggregate,
            limits=self.limits,
        )
        self.assertTrue(closeout.continue_loop)
        self.assertEqual(closeout.next_prompt, "下一轮继续")
        self.assertEqual(closeout.validation.status, "not_run")
        self.assertEqual(closeout.parse_mode, "keyword_fallback")

    def test_continue_without_prompt_fails(self) -> None:
        message = textwrap.dedent(
            """
            ```json
            {
              "closeout_status": "handoff_only",
              "continue": true,
              "next_prompt": null,
              "stop_reason": null,
              "validation": {"status": "passed", "commands": [], "note": null},
              "commit": {"created": false, "sha": null, "title": null},
              "consumed_budget": {"rounds": 1, "files_changed": 1, "commits": 0, "runtime_minutes": 4},
              "remaining_budget": {"rounds": 4, "files_changed": 29, "commits": 3, "runtime_minutes": 86},
              "scope_expanded": false,
              "risk_level": "low"
            }
            ```
            """
        ).strip()
        with self.assertRaises(run_loop.LoopError):
            run_loop.parse_closeout_message(
                message=message,
                default_runtime_minutes=4,
                aggregate=self.aggregate,
                limits=self.limits,
            )

    def test_json_and_keyword_mismatch_fails(self) -> None:
        message = textwrap.dedent(
            """
            Next-session startup prompt: 中文提示词
            ```json
            {
              "closeout_status": "handoff_only",
              "continue": true,
              "next_prompt": "different prompt",
              "stop_reason": null,
              "validation": {"status": "passed", "commands": [], "note": null},
              "commit": {"created": false, "sha": null, "title": null},
              "consumed_budget": {"rounds": 1, "files_changed": 1, "commits": 0, "runtime_minutes": 4},
              "remaining_budget": {"rounds": 4, "files_changed": 29, "commits": 3, "runtime_minutes": 86},
              "scope_expanded": false,
              "risk_level": "low"
            }
            ```
            """
        ).strip()
        with self.assertRaises(run_loop.LoopError):
            run_loop.parse_closeout_message(
                message=message,
                default_runtime_minutes=4,
                aggregate=self.aggregate,
                limits=self.limits,
            )


class StopConditionTests(unittest.TestCase):
    def setUp(self) -> None:
        self.limits = run_loop.BudgetLimits(
            max_rounds=5,
            max_files_changed=10,
            max_commits=2,
            max_runtime_minutes=30,
            validation_mode="task-required",
            stop_on_validation_failure=True,
            require_clean_worktree=True,
            scope_expansion_policy="forbid",
            allowed_scopes=["docs"],
        )

    def test_repeated_prompt_stops(self) -> None:
        closeout = run_loop.CloseoutResult(
            closeout_status="handoff_only",
            continue_loop=True,
            next_prompt="继续",
            stop_reason=None,
            validation=run_loop.ValidationResult(status="passed", commands=[], note=None),
            commit=run_loop.CommitResult(created=False, sha=None, title=None),
            consumed_budget=run_loop.BudgetUsage(rounds=1, files_changed=1, commits=0, runtime_minutes=2),
            remaining_budget=run_loop.BudgetUsage(rounds=4, files_changed=9, commits=2, runtime_minutes=28),
            scope_expanded=False,
            risk_level="low",
            raw_message="",
            parse_mode="json",
        )
        reason = run_loop.evaluate_stop_condition(
            closeout=closeout,
            aggregate=run_loop.BudgetUsage(rounds=1, files_changed=1, commits=0, runtime_minutes=2),
            limits=self.limits,
            seen_prompts={"继续"},
        )
        self.assertEqual(reason, "repeated_next_prompt")

    def test_scope_expanded_stops(self) -> None:
        closeout = run_loop.CloseoutResult(
            closeout_status="handoff_only",
            continue_loop=True,
            next_prompt="继续",
            stop_reason=None,
            validation=run_loop.ValidationResult(status="passed", commands=[], note=None),
            commit=run_loop.CommitResult(created=False, sha=None, title=None),
            consumed_budget=run_loop.BudgetUsage(rounds=1, files_changed=1, commits=0, runtime_minutes=2),
            remaining_budget=run_loop.BudgetUsage(rounds=4, files_changed=9, commits=2, runtime_minutes=28),
            scope_expanded=True,
            risk_level="low",
            raw_message="",
            parse_mode="json",
        )
        reason = run_loop.evaluate_stop_condition(
            closeout=closeout,
            aggregate=run_loop.BudgetUsage(rounds=1, files_changed=1, commits=0, runtime_minutes=2),
            limits=self.limits,
            seen_prompts=set(),
        )
        self.assertEqual(reason, "scope_expanded")

    def test_validation_failure_stops(self) -> None:
        closeout = run_loop.CloseoutResult(
            closeout_status="blocked",
            continue_loop=False,
            next_prompt=None,
            stop_reason="validation_failed",
            validation=run_loop.ValidationResult(status="failed", commands=["bun run check"], note=None),
            commit=run_loop.CommitResult(created=False, sha=None, title=None),
            consumed_budget=run_loop.BudgetUsage(rounds=1, files_changed=1, commits=0, runtime_minutes=2),
            remaining_budget=run_loop.BudgetUsage(rounds=4, files_changed=9, commits=2, runtime_minutes=28),
            scope_expanded=False,
            risk_level="medium",
            raw_message="",
            parse_mode="json",
        )
        reason = run_loop.evaluate_stop_condition(
            closeout=closeout,
            aggregate=run_loop.BudgetUsage(rounds=1, files_changed=1, commits=0, runtime_minutes=2),
            limits=self.limits,
            seen_prompts=set(),
        )
        self.assertEqual(reason, "validation_failed")

    def test_budget_exhaustion_stops(self) -> None:
        closeout = run_loop.CloseoutResult(
            closeout_status="handoff_only",
            continue_loop=True,
            next_prompt="继续",
            stop_reason=None,
            validation=run_loop.ValidationResult(status="passed", commands=[], note=None),
            commit=run_loop.CommitResult(created=False, sha=None, title=None),
            consumed_budget=run_loop.BudgetUsage(rounds=1, files_changed=3, commits=1, runtime_minutes=5),
            remaining_budget=run_loop.BudgetUsage(rounds=3, files_changed=0, commits=0, runtime_minutes=10),
            scope_expanded=False,
            risk_level="low",
            raw_message="",
            parse_mode="json",
        )
        reason = run_loop.evaluate_stop_condition(
            closeout=closeout,
            aggregate=run_loop.BudgetUsage(rounds=2, files_changed=10, commits=2, runtime_minutes=15),
            limits=self.limits,
            seen_prompts=set(),
        )
        self.assertEqual(reason, "max_files_changed_exhausted")


class LoopExecutionTests(unittest.TestCase):
    def test_dirty_worktree_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            repo = Path(tmpdir)
            subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True, text=True)
            (repo / "dirty.txt").write_text("x\n", encoding="utf-8")
            args = argparse.Namespace(
                repo_root=str(repo),
                task_file=None,
                task_text="整理 docs",
                task_class="docs/automation",
                recovery_source="parent topic",
                owned_scope="docs",
                allowed_scopes=[],
                max_rounds=2,
                max_files_changed=10,
                max_commits=2,
                max_runtime_minutes=30,
                validation_mode="task-required",
                stop_on_validation_failure=True,
                require_clean_worktree=True,
                scope_expansion_policy="forbid",
                codex_bin="codex",
                model=None,
                result_json=None,
                round_output_dir=None,
                dangerously_bypass_approvals_and_sandbox=False,
            )
            with self.assertRaises(run_loop.LoopError):
                run_loop.run_loop(args)

    def test_run_loop_with_stubbed_rounds(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            repo = Path(tmpdir)
            subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True, text=True)
            subprocess.run(
                [
                    "git",
                    "-c",
                    "user.name=Test User",
                    "-c",
                    "user.email=test@example.com",
                    "commit",
                    "--allow-empty",
                    "-m",
                    "init",
                ],
                cwd=repo,
                check=True,
                capture_output=True,
                text=True,
            )
            args = argparse.Namespace(
                repo_root=str(repo),
                task_file=None,
                task_text="整理 docs",
                task_class="docs/automation",
                recovery_source="parent topic",
                owned_scope="docs",
                allowed_scopes=[],
                max_rounds=3,
                max_files_changed=10,
                max_commits=2,
                max_runtime_minutes=30,
                validation_mode="task-required",
                stop_on_validation_failure=True,
                require_clean_worktree=True,
                scope_expansion_policy="forbid",
                codex_bin="codex",
                model=None,
                result_json=None,
                round_output_dir=None,
                dangerously_bypass_approvals_and_sandbox=False,
            )
            round_messages = [
                textwrap.dedent(
                    """
                    Next-session startup prompt: 第二轮继续
                    ```json
                    {
                      "closeout_status": "handoff_only",
                      "continue": true,
                      "next_prompt": "第二轮继续",
                      "stop_reason": null,
                      "validation": {"status": "passed", "commands": ["python3 -m unittest"], "note": null},
                      "commit": {"created": false, "sha": null, "title": null},
                      "consumed_budget": {"rounds": 1, "files_changed": 2, "commits": 0, "runtime_minutes": 3},
                      "remaining_budget": {"rounds": 2, "files_changed": 8, "commits": 2, "runtime_minutes": 27},
                      "scope_expanded": false,
                      "risk_level": "low"
                    }
                    ```
                    """
                ).strip(),
                textwrap.dedent(
                    """
                    ```json
                    {
                      "closeout_status": "completed_no_handoff",
                      "continue": false,
                      "next_prompt": null,
                      "stop_reason": "task_complete",
                      "validation": {"status": "passed", "commands": ["python3 -m unittest"], "note": null},
                      "commit": {"created": true, "sha": "abc1234", "title": "docs(loop): close task"},
                      "consumed_budget": {"rounds": 1, "files_changed": 1, "commits": 1, "runtime_minutes": 4},
                      "remaining_budget": {"rounds": 1, "files_changed": 7, "commits": 1, "runtime_minutes": 23},
                      "scope_expanded": false,
                      "risk_level": "low"
                    }
                    ```
                    """
                ).strip(),
            ]

            with mock.patch.object(run_loop, "run_codex_exec", side_effect=round_messages):
                result = run_loop.run_loop(args)

            self.assertEqual(result["total_rounds"], 2)
            self.assertEqual(result["stop_reason"], "task_complete")
            self.assertEqual(result["consumed_budget"]["commits"], 1)
            self.assertEqual(result["rounds"][0]["parse_mode"], "json")


if __name__ == "__main__":
    unittest.main()
