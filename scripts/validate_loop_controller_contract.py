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
    {"continue", "pending_batches", "next_batch", "archive_ready", "topic_complete"}
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
                (r"worker\s+completion", r"never\s+completes\s+a\s+topic\s+loop"),
                (r"suggested_follow_up",),
                (r"worker.*must\s+not.*continue", r"worker.*must\s+not.*pending_batches", r"worker.*must\s+not.*next_batch"),
            ),
        ),
        (
            BATCH_SKILL,
            "batch-skill",
            (
                (r"one\s+execution\s+wave",),
                (r"returns?\s+control", r"outer\s+(loop\s+)?controller"),
                (r"never\s+completes\s+the\s+topic\s+loop",),
                (r"suggested_follow_up",),
            ),
        ),
        (
            TASK_SKILL,
            "task-skill",
            (
                (r"round\s+evidence", r"suggested_follow_up"),
                (r"only\s+the\s+outer\s+(main\s+agent|controller)", r"next\s+batch"),
                (r"must\s+not\s+decide\s+topic\s+completion",),
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

    if validate_worker_handoff({"suggested_follow_up": {"next_batch": "phase-next"}}, "advisory-follow-up"):
        findings.append(Finding("advisory-follow-up", "suggested_follow_up must remain valid advisory metadata"))
    if validate_controller_transition(
        {"target_state": "ARCHIVE_READY", "pending_batches": [], "archive_readiness_completed": True},
        "archive-ready-after-check",
    ):
        findings.append(Finding("archive-ready-after-check", "valid controller archive transition was rejected"))
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
