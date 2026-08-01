#!/usr/bin/env python3
"""Manually validate controller-owned multi-agent loop contracts."""

from __future__ import annotations

import argparse
import json
import re
import sys
from collections.abc import Mapping
from dataclasses import dataclass
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[1]
LOOP_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-multi-agent-loop" / "SKILL.md"
BATCH_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-multi-agent-batch" / "SKILL.md"
TASK_SKILL = REPO_ROOT / ".agents" / "skills" / "graft-multi-agent-task" / "SKILL.md"
WORKER_CONTROLLER_FIELDS = frozenset(
    {
        "continue",
        "pending_batches",
        "next_batch",
        "archive_ready",
        "topic_complete",
        "stop_loop",
        "suspend_topic",
        "wait_for_user",
    }
)
RECOVERY_NONTERMINAL_STATES = frozenset(
    {"RECOVERY_REQUIRED", "RECOVERY_CONTEXT_RESTORED", "CONTEXT_RESTORED", "AUTHORITY_CONFIRMED", "RECOVERY_COMPLETE", "RESUME_CURRENT_BATCH"}
)
OUTER_CONTROLLER_REQUIRED_FIELDS = frozenset(
    {
        "closeout_status",
        "continue",
        "loop_mode",
        "current_batch",
        "completed_batches",
        "pending_batches",
        "next_batch",
        "next_batch_prompt",
        "next_prompt",
        "stop_reason",
        "validation",
        "commit",
        "consumed_budget",
        "remaining_budget",
        "scope_expanded",
        "risk_level",
        "recovery",
    }
)
RECOVERY_REQUIRED_FIELDS = frozenset(
    {
        "status",
        "resume_target",
        "current_batch_preserved",
        "pending_batches_preserved",
        "failed_batch_settled",
        "retry_exhausted",
        "repair_authority",
        "repair_eligible",
        "required_context",
    }
)


@dataclass(frozen=True)
class Finding:
    scenario: str
    message: str

    def format(self) -> str:
        return f"{self.scenario}: {self.message}"


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def has_concepts(text: str, concepts: tuple[tuple[str, ...], ...]) -> bool:
    """Match required concepts without depending on headings, line numbers, or exact sentences."""
    return all(
        all(re.search(pattern, text, re.IGNORECASE | re.DOTALL) for pattern in alternatives)
        for alternatives in concepts
    )


def validate_worker_handoff(worker: Mapping[str, Any], scenario: str) -> list[Finding]:
    findings: list[Finding] = []
    for field in sorted(WORKER_CONTROLLER_FIELDS.intersection(worker)):
        findings.append(Finding(scenario, f"worker cannot terminate or transition the loop through {field!r}"))

    suggested = worker.get("suggested_follow_up")
    if suggested is not None and not isinstance(suggested, Mapping):
        findings.append(Finding(scenario, "suggested_follow_up must be advisory metadata object"))
    return findings


def validate_controller_transition(controller: Mapping[str, Any], scenario: str) -> list[Finding]:
    findings: list[Finding] = []
    source_state = controller.get("source_state")
    target_state = controller.get("target_state")
    pending_batches = controller.get("pending_batches", [])
    archive_checked = controller.get("archive_readiness_completed", False)

    if target_state == "ARCHIVE_READY":
        if pending_batches:
            findings.append(Finding(scenario, "controller cannot archive-ready while pending_batches remain"))
        if archive_checked is not True:
            findings.append(Finding(scenario, "controller cannot archive-ready before archive readiness completes"))

    if target_state == "BLOCKED" and not controller.get("stop_reason"):
        findings.append(Finding(scenario, "controller blocked state requires an explicit stop_reason"))

    if target_state == "END":
        findings.append(Finding(scenario, "END is not a controller state; transition to ARCHIVE_READY or BLOCKED"))

    if source_state in RECOVERY_NONTERMINAL_STATES and target_state == "END":
        findings.append(Finding(scenario, f"{source_state} is non-terminal and cannot transition to END"))

    if source_state == "RECOVERY_COMPLETE" and target_state != "RESUME_CURRENT_BATCH":
        findings.append(Finding(scenario, "RECOVERY_COMPLETE must transition to RESUME_CURRENT_BATCH"))

    if source_state == "RESUME_CURRENT_BATCH" and target_state != "DISPATCH":
        findings.append(Finding(scenario, "RESUME_CURRENT_BATCH must transition to DISPATCH"))
    return findings


def validate_controller_path(states: tuple[str, ...], scenario: str) -> list[Finding]:
    findings: list[Finding] = []
    required_suffix = ("RECOVERY_COMPLETE", "RESUME_CURRENT_BATCH", "DISPATCH")
    for recovery_index, state in enumerate(states):
        if state != "RECOVERY_COMPLETE":
            continue
        if states[recovery_index : recovery_index + len(required_suffix)] != required_suffix:
            findings.append(Finding(scenario, "recovery completion must resume and dispatch the current batch"))
    return findings


def validate_outer_controller_closeout(closeout: Mapping[str, Any], scenario: str) -> list[Finding]:
    findings: list[Finding] = []
    missing = sorted(OUTER_CONTROLLER_REQUIRED_FIELDS.difference(closeout))
    if missing:
        findings.append(Finding(scenario, f"outer-controller closeout is missing fields: {', '.join(missing)}"))

    if closeout.get("closeout_status") == "exhausted_retry":
        findings.append(Finding(scenario, "exhausted_retry is wave evidence, not an outer-controller closeout status"))

    recovery = closeout.get("recovery")
    if not isinstance(recovery, Mapping):
        findings.append(Finding(scenario, "outer-controller closeout recovery must be an object"))
        return findings

    missing_recovery = sorted(RECOVERY_REQUIRED_FIELDS.difference(recovery))
    if missing_recovery:
        findings.append(Finding(scenario, f"recovery is missing fields: {', '.join(missing_recovery)}"))

    if (
        recovery.get("retry_exhausted") is True
        and closeout.get("closeout_status") in {"blocked", "cancelled", "unsafe"}
        and recovery.get("failed_batch_settled") is not True
    ):
        findings.append(Finding(scenario, "retry exhaustion must be settled before a terminal closeout"))

    if recovery.get("status") == "required":
        required_context = recovery.get("required_context")
        if not isinstance(required_context, Mapping):
            findings.append(Finding(scenario, "required recovery must preserve required_context as an object"))
        else:
            for field in ("failed_round", "evidence"):
                if field not in required_context:
                    findings.append(Finding(scenario, f"required recovery context is missing {field!r}"))
        if recovery.get("resume_target") != "RESUME_CURRENT_BATCH":
            findings.append(Finding(scenario, "required recovery must preserve RESUME_CURRENT_BATCH"))
        if recovery.get("current_batch_preserved") is not True or recovery.get("pending_batches_preserved") is not True:
            findings.append(Finding(scenario, "required recovery must preserve current_batch and pending_batches"))
        if recovery.get("failed_batch_settled") is not False:
            findings.append(Finding(scenario, "required recovery must keep the failed batch unsettled"))
    return findings


def validate_skill_contracts() -> list[Finding]:
    findings: list[Finding] = []
    checks = (
        (
            LOOP_SKILL,
            "loop-skill",
            (
                (r"outer\s+main\s+agent|outer\s+controller", r"topic\s+lifecycle\s+owner", r"terminal\s+decision\s+owner"),
                (r"INIT", r"DISPATCH", r"VERIFY", r"SETTLE", r"ARCHIVE_CHECK"),
                (r"RECOVERY_REQUIRED", r"RECOVERY_CONTEXT_RESTORED", r"RECOVERY_COMPLETE", r"RESUME_CURRENT_BATCH"),
                (r"RECOVERY_COMPLETE.*RESUME_CURRENT_BATCH.*DISPATCH",),
                (r"worker\s+completion", r"never\s+completes\s+a\s+topic\s+loop"),
                (r"suggested_follow_up",),
                (r"worker.*must\s+not.*continue", r"worker.*must\s+not.*pending_batches", r"worker.*must\s+not.*next_batch"),
                (r"canonical\s+outer-controller\s+closeout\s+schema", r"retry_exhausted.*wave\s+evidence", r"required_context"),
            ),
        ),
        (
            BATCH_SKILL,
            "batch-skill",
            (
                (r"one\s+execution\s+wave",),
                (r"returns?\s+control", r"outer\s+(loop\s+)?controller"),
                (r"never\s+completes(?:\s+or\s+suspends)?\s+the\s+topic\s+loop",),
                (r"suggested_follow_up",),
                (r"retry-exhaustion", r"wave\s+evidence", r"outer\s+controller"),
            ),
        ),
        (
            TASK_SKILL,
            "task-skill",
            (
                (r"round\s+evidence", r"suggested_follow_up"),
                (r"only\s+the\s+outer\s+(main\s+agent|controller)", r"next\s+batch"),
                (r"must\s+not\s+decide\s+topic\s+completion",),
                (r"must\s+not\s+resume\s+the\s+current\s+batch",),
                (r"retry exhaustion", r"wave evidence", r"outer controller"),
            ),
        ),
    )

    for path, scenario, concepts in checks:
        if not path.is_file():
            findings.append(Finding(scenario, f"required skill is missing: {path.relative_to(REPO_ROOT)}"))
            continue
        if not has_concepts(read_text(path), concepts):
            findings.append(Finding(scenario, "missing controller/worker contract concepts"))
    return findings


def run_validation() -> list[Finding]:
    findings = validate_skill_contracts()
    findings.extend(validate_worker_handoff({"continue": False}, "worker-continue-cannot-terminate"))
    findings.extend(validate_worker_handoff({"next_batch": "phase-next"}, "worker-next-batch-is-not-controller-state"))
    findings.extend(
        validate_controller_transition(
            {
                "target_state": "ARCHIVE_READY",
                "pending_batches": ["phase-next"],
                "archive_readiness_completed": False,
            },
            "archive-ready-requires-settled-topic",
        )
    )
    findings.extend(
        validate_controller_transition(
            {"source_state": "RECOVERY_COMPLETE", "target_state": "END"},
            "recovery-complete-cannot-end",
        )
    )
    findings.extend(
        validate_controller_transition(
            {"source_state": "CONTEXT_RESTORED", "target_state": "END"},
            "context-restored-cannot-end",
        )
    )
    findings.extend(
        validate_controller_transition(
            {"source_state": "AUTHORITY_CONFIRMED", "target_state": "END"},
            "authority-confirmed-cannot-end",
        )
    )

    if validate_worker_handoff({"suggested_follow_up": {"next_batch": "phase-next"}}, "advisory-follow-up"):
        findings.append(Finding("advisory-follow-up", "suggested_follow_up must remain valid advisory metadata"))
    if validate_controller_transition(
        {"target_state": "ARCHIVE_READY", "pending_batches": [], "archive_readiness_completed": True},
        "archive-ready-after-check",
    ):
        findings.append(Finding("archive-ready-after-check", "valid controller archive transition was rejected"))
    if validate_controller_path(
        ("VALIDATION_FAILED", "RECOVERY_COMPLETE", "RESUME_CURRENT_BATCH", "DISPATCH", "WAIT", "VERIFY"),
        "validation-failure-recovery-resumes-current-batch",
    ):
        findings.append(Finding("validation-failure-recovery-resumes-current-batch", "valid recovery path was rejected"))
    findings.extend(
        validate_controller_path(
            (
                "VALIDATION_FAILED",
                "RECOVERY_COMPLETE",
                "RESUME_CURRENT_BATCH",
                "DISPATCH",
                "RECOVERY_COMPLETE",
                "WAIT",
            ),
            "every-recovery-complete-must-resume",
        )
    )

    valid_closeout = {
        "closeout_status": "recovery_handoff",
        "continue": False,
        "loop_mode": "topic-completion-loop",
        "current_batch": "phase-current",
        "completed_batches": [],
        "pending_batches": ["phase-current"],
        "next_batch": None,
        "next_batch_prompt": None,
        "next_prompt": "resume",
        "stop_reason": None,
        "validation": {"status": "failed", "commands": ["validate"], "note": "retry exhausted"},
        "commit": {"created": False, "sha": None, "title": None},
        "consumed_budget": {"rounds": 1, "files_changed": 0, "commits": 0, "runtime_minutes": 1},
        "remaining_budget": {"rounds": 0, "files_changed": 0, "commits": 0, "runtime_minutes": 0},
        "scope_expanded": False,
        "risk_level": "medium",
        "recovery": {
            "status": "required",
            "resume_target": "RESUME_CURRENT_BATCH",
            "current_batch_preserved": True,
            "pending_batches_preserved": True,
            "failed_batch_settled": False,
            "retry_exhausted": True,
            "repair_authority": "outer-controller",
            "repair_eligible": False,
            "required_context": {"failed_round": {"round_status": "retry-needed"}, "evidence": {"validation": "failed"}},
        },
    }
    if validate_outer_controller_closeout(valid_closeout, "valid-canonical-closeout"):
        findings.append(Finding("valid-canonical-closeout", "valid canonical closeout was rejected"))
    findings.extend(
        validate_outer_controller_closeout(
            {"closeout_status": "exhausted_retry", "recovery": {"retry_exhausted": True}},
            "invalid-exhausted-retry-closeout",
        )
    )
    findings.extend(
        validate_outer_controller_closeout(
            {
                **valid_closeout,
                "closeout_status": "blocked",
            },
            "invalid-retry-exhaustion-terminal-closeout",
        )
    )
    settled_terminal_closeout = {
        **valid_closeout,
        "closeout_status": "blocked",
        "recovery": {
            **valid_closeout["recovery"],
            "status": "complete",
            "resume_target": None,
            "failed_batch_settled": True,
        },
    }
    if validate_outer_controller_closeout(settled_terminal_closeout, "valid-settled-terminal-closeout"):
        findings.append(Finding("valid-settled-terminal-closeout", "settled retry exhaustion was rejected"))
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description="Manually validate multi-agent loop controller contracts.")
    parser.add_argument("--format", choices=("text", "json"), default="text", help="output format")
    args = parser.parse_args()

    findings = run_validation()
    expected_failures = {
        "worker-continue-cannot-terminate",
        "worker-next-batch-is-not-controller-state",
        "archive-ready-requires-settled-topic",
        "recovery-complete-cannot-end",
        "context-restored-cannot-end",
        "authority-confirmed-cannot-end",
        "invalid-exhausted-retry-closeout",
        "invalid-retry-exhaustion-terminal-closeout",
        "every-recovery-complete-must-resume",
    }
    observed_scenarios = {finding.scenario for finding in findings}
    for scenario in sorted(expected_failures - observed_scenarios):
        findings.append(Finding("regression", f"invalid scenario was accepted: {scenario}"))
    unexpected = [finding for finding in findings if finding.scenario not in expected_failures]

    if args.format == "json":
        print(
            json.dumps(
                {
                    "ok": not unexpected,
                    "expected_failures": [item.format() for item in findings if item.scenario in expected_failures],
                    "findings": [item.format() for item in unexpected],
                },
                ensure_ascii=False,
                indent=2,
            )
        )
    elif unexpected:
        print("Loop controller contract validation failed:", file=sys.stderr)
        for finding in unexpected:
            print(f"- {finding.format()}", file=sys.stderr)
    else:
        print("Loop controller contract validation passed")
    return 1 if unexpected else 0


if __name__ == "__main__":
    raise SystemExit(main())
