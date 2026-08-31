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
WORKER_CONTRACT = REPO_ROOT / ".agents" / "skills" / "graft-multi-agent-batch" / "references" / "worker-contract.md"
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


def _string_set(value: Any) -> set[str] | None:
    if not isinstance(value, list) or any(not isinstance(item, str) or not item for item in value):
        return None
    return set(value)


def validate_topology_plan(plan: Mapping[str, Any], scenario: str) -> list[Finding]:
    findings: list[Finding] = []
    revision = plan.get("topology_revision")
    if not isinstance(revision, int) or revision < 1:
        findings.append(Finding(scenario, "topology_revision must be a positive integer"))
    raw_nodes = plan.get("nodes")
    if not isinstance(raw_nodes, list) or not raw_nodes:
        findings.append(Finding(scenario, "topology plan must contain a non-empty nodes list"))
        return findings
    nodes: dict[str, Mapping[str, Any]] = {}
    for index, node in enumerate(raw_nodes):
        if not isinstance(node, Mapping):
            findings.append(Finding(scenario, f"topology node {index} must be an object"))
            continue
        node_id = node.get("node_id")
        if not isinstance(node_id, str) or not node_id:
            findings.append(Finding(scenario, f"topology node {index} must have a node_id"))
            continue
        if node_id in nodes:
            findings.append(Finding(scenario, f"topology node id {node_id!r} must be unique"))
        nodes[node_id] = node
        for field in ("objective", "authority_owner", "validation", "acceptance_gate", "execution_context"):
            if not isinstance(node.get(field), str) or not node[field]:
                findings.append(Finding(scenario, f"topology node {node_id!r} requires non-empty {field}"))
        for field in ("depends_on", "owned_scope", "forbidden_scope"):
            if _string_set(node.get(field)) is None:
                findings.append(Finding(scenario, f"topology node {node_id!r} requires a string list for {field}"))
    dependencies: dict[str, set[str]] = {}
    for node_id, node in nodes.items():
        deps = _string_set(node.get("depends_on")) or set()
        dependencies[node_id] = deps
        missing = sorted(deps.difference(nodes))
        if missing:
            findings.append(Finding(scenario, f"topology node {node_id!r} references missing dependencies: {', '.join(missing)}"))
    visit_state: dict[str, int] = {}

    def visit(node_id: str) -> None:
        state = visit_state.get(node_id, 0)
        if state == 1:
            findings.append(Finding(scenario, f"topology dependency cycle includes {node_id!r}"))
            return
        if state == 2:
            return
        visit_state[node_id] = 1
        for dependency in dependencies.get(node_id, set()):
            if dependency in nodes:
                visit(dependency)
        visit_state[node_id] = 2
    for node_id in nodes:
        visit(node_id)
    return findings


def validate_ready_frontier(
    plan: Mapping[str, Any], completed: set[str], dispatched: set[str], ready: list[str], scenario: str
) -> list[Finding]:
    findings: list[Finding] = []
    nodes = {
        node["node_id"]: node
        for node in plan.get("nodes", [])
        if isinstance(node, Mapping) and node.get("node_id")
    }
    if len(ready) != len(set(ready)):
        findings.append(Finding(scenario, "ready frontier node ids must be unique"))
    for node_id in ready:
        node = nodes.get(node_id)
        if node is None:
            findings.append(Finding(scenario, f"ready frontier contains unknown node {node_id!r}"))
            continue
        if node_id in completed:
            findings.append(Finding(scenario, f"ready frontier cannot contain completed node {node_id!r}"))
        elif node_id in dispatched:
            findings.append(Finding(scenario, f"ready frontier cannot contain dispatched-but-unsettled node {node_id!r}"))
        if not (_string_set(node.get("depends_on")) or set()).issubset(completed):
            findings.append(Finding(scenario, f"ready node {node_id!r} has unsettled dependencies"))
    for index, left_id in enumerate(ready):
        left = nodes.get(left_id)
        if left is None:
            continue
        left_scope = _string_set(left.get("owned_scope")) or set()
        for right_id in ready[index + 1 :]:
            right = nodes.get(right_id)
            if right is None:
                continue
            right_scope = _string_set(right.get("owned_scope")) or set()
            if (
                left_scope.intersection(right_scope)
                or left.get("authority_owner") == right.get("authority_owner")
                or left.get("execution_context") == right.get("execution_context")
                or left.get("acceptance_gate") == right.get("acceptance_gate")
            ):
                findings.append(Finding(scenario, f"ready nodes {left_id!r} and {right_id!r} are not independent"))
    return findings


def validate_topology_replan(
    replan: Mapping[str, Any], completed_or_dispatched: set[str], scenario: str
) -> list[Finding]:
    findings: list[Finding] = []
    if (
        not isinstance(replan.get("topology_revision"), int)
        or replan["topology_revision"] <= replan.get("previous_revision", 0)
    ):
        findings.append(Finding(scenario, "topology replan must increment topology_revision"))
    affected = _string_set(replan.get("affected_nodes"))
    if affected is None or not affected:
        findings.append(Finding(scenario, "topology replan must identify affected nodes"))
    elif affected.intersection(completed_or_dispatched):
        findings.append(Finding(scenario, "topology replan cannot modify completed or dispatched nodes"))
    for field in ("reason", "evidence", "dependency_delta", "authority_impact", "validation_impact"):
        if not replan.get(field):
            findings.append(Finding(scenario, f"topology replan requires {field} evidence"))
    return findings


def validate_topology_evidence(worker: Mapping[str, Any], scenario: str) -> list[Finding]:
    findings: list[Finding] = []
    evidence = worker.get("topology_evidence")
    if not isinstance(evidence, Mapping):
        findings.append(Finding(scenario, "worker topology_evidence must be an object"))
        return findings
    for field in ("topology_revision", "node_id", "node_status", "dependency_observed", "gate_evidence"):
        if field not in evidence:
            findings.append(Finding(scenario, f"worker topology_evidence is missing {field!r}"))
    revision = evidence.get("topology_revision")
    if "topology_revision" in evidence and (type(revision) is not int or revision < 1):
        findings.append(Finding(scenario, "worker topology_evidence topology_revision must be a positive integer"))
    node_id = evidence.get("node_id")
    if "node_id" in evidence and (not isinstance(node_id, str) or not node_id.strip()):
        findings.append(Finding(scenario, "worker topology_evidence node_id must be a non-empty string"))
    if "node_status" in evidence and evidence.get("node_status") not in {"completed", "blocked", "retry-needed"}:
        findings.append(Finding(scenario, "worker topology_evidence node_status must be completed, blocked, or retry-needed"))
    for field in sorted(WORKER_CONTROLLER_FIELDS.intersection(evidence)):
        findings.append(Finding(scenario, f"worker topology_evidence cannot carry controller field {field!r}"))
    return findings


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
                (r"hybrid\s+DAG|DAG", r"ready\s+frontier"),
                (r"topology_revision", r"undispatched"),
                (r"write\s+sets", r"authority", r"execution\s+contexts?"),
                (r"returns?\s+control", r"outer\s+(loop\s+)?controller"),
                (r"never\s+completes(?:\s+or\s+suspends)?\s+the\s+topic\s+loop",),
                (r"workers?.*(?:must\s+not|cannot).*topology|worker.*mutate.*topology",),
                (r"suggested_follow_up",),
                (r"retry-exhaustion", r"wave\s+evidence", r"outer\s+controller"),
            ),
        ),
        (
            WORKER_CONTRACT,
            "worker-contract",
            (
                (r"topology\s+revision", r"node\s+id", r"dependencies"),
                (r"topology_evidence", r"node_status"),
                (r"immutable\s+topology|cannot\s+change.*dependencies|must\s+not.*select.*frontier",),
                (r"never\s+emits\s+controller\s+state",),
            ),
        ),
        (
            TASK_SKILL,
            "task-skill",
            (
                (r"round\s+evidence", r"suggested_follow_up"),
                (r"topology_revision", r"node_id", r"ready\s+node"),
                (r"must\s+not.*mutate.*topology|outer\s+controller.*wave\s+boundary",),
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


def _sample_topology() -> dict[str, Any]:
    return {
        "topology_revision": 1,
        "nodes": [
            {
                "node_id": "authority-audit",
                "objective": "confirm authority",
                "depends_on": [],
                "owned_scope": ["docs/a"],
                "forbidden_scope": ["server"],
                "authority_owner": "outer-controller",
                "validation": "authority-check",
                "acceptance_gate": "gate-a",
                "execution_context": "primary-readonly",
            },
            {
                "node_id": "implementation-a",
                "objective": "implement slice",
                "depends_on": ["authority-audit"],
                "owned_scope": ["docs/b"],
                "forbidden_scope": ["server"],
                "authority_owner": "worker-a",
                "validation": "focused-check-a",
                "acceptance_gate": "gate-b",
                "execution_context": "worker-a",
            },
            {
                "node_id": "implementation-b",
                "objective": "implement independent slice",
                "depends_on": ["authority-audit"],
                "owned_scope": ["docs/c"],
                "forbidden_scope": ["server"],
                "authority_owner": "worker-b",
                "validation": "focused-check-b",
                "acceptance_gate": "gate-c",
                "execution_context": "worker-b",
            },
        ],
    }


def run_validation() -> list[Finding]:
    findings = validate_skill_contracts()
    valid_topology = _sample_topology()
    if validate_topology_plan(valid_topology, "valid-topology-plan"):
        findings.append(Finding("valid-topology-plan", "valid hybrid DAG was rejected"))
    if validate_ready_frontier(
        valid_topology,
        {"authority-audit"},
        {"authority-audit"},
        ["implementation-a", "implementation-b"],
        "valid-ready-frontier",
    ):
        findings.append(Finding("valid-ready-frontier", "independent ready frontier was rejected"))
    valid_replan = {
        "previous_revision": 1,
        "topology_revision": 2,
        "affected_nodes": ["implementation-b"],
        "reason": "new evidence",
        "evidence": "validation output",
        "dependency_delta": {"implementation-b": ["authority-audit"]},
        "authority_impact": "none",
        "validation_impact": "rerun focused check",
    }
    if validate_topology_replan(valid_replan, {"authority-audit", "implementation-a"}, "valid-topology-replan"):
        findings.append(Finding("valid-topology-replan", "valid undispatched topology replan was rejected"))
    if validate_topology_evidence(
        {
            "topology_evidence": {
                "topology_revision": 1,
                "node_id": "implementation-a",
                "node_status": "completed",
                "dependency_observed": "passed",
                "gate_evidence": "focused-check-a",
            }
        },
        "valid-topology-evidence",
    ):
        findings.append(Finding("valid-topology-evidence", "valid worker topology evidence was rejected"))

    cycle_topology = _sample_topology()
    cycle_topology['nodes'][0]["node_id"] = "cycle-a"
    cycle_topology['nodes'][0]["depends_on"] = ["cycle-b"]
    cycle_topology['nodes'][1]["node_id"] = "cycle-b"
    cycle_topology['nodes'][1]["depends_on"] = ["cycle-a"]
    cycle_topology['nodes'][2]["depends_on"] = ["cycle-a"]
    findings.extend(validate_topology_plan(cycle_topology, "topology-cycle-rejected"))
    findings.extend(
        validate_ready_frontier(
            valid_topology,
            set(),
            set(),
            ["implementation-a"],
            "ready-frontier-unsettled-dependency",
        )
    )
    findings.extend(
        validate_ready_frontier(
            valid_topology,
            {"authority-audit"},
            set(),
            ["implementation-a", "implementation-a"],
            "ready-frontier-duplicate-node",
        )
    )
    findings.extend(
        validate_ready_frontier(
            valid_topology,
            {"authority-audit", "implementation-a"},
            {"authority-audit", "implementation-a"},
            ["implementation-a"],
            "ready-frontier-completed-node",
        )
    )
    findings.extend(
        validate_ready_frontier(
            valid_topology,
            {"authority-audit"},
            {"authority-audit", "implementation-a"},
            ["implementation-a"],
            "ready-frontier-dispatched-unsettled-node",
        )
    )

    conflict_topology = _sample_topology()
    conflict_topology['nodes'][1]["owned_scope"] = ["docs/shared"]
    conflict_topology['nodes'][2]["owned_scope"] = ["docs/shared"]
    findings.extend(
        validate_ready_frontier(
            conflict_topology,
            {"authority-audit"},
            {"authority-audit"},
            ["implementation-a", "implementation-b"],
            "ready-frontier-overlap",
        )
    )
    findings.extend(
        validate_topology_replan(
            {**valid_replan, "affected_nodes": ["authority-audit"]},
            {"authority-audit"},
            "replan-cannot-rewrite-dispatched-node",
        )
    )
    findings.extend(
        validate_topology_evidence(
            {
                "topology_evidence": {
                    "topology_revision": 1,
                    "node_id": "implementation-a",
                    "node_status": "completed",
                    "dependency_observed": "passed",
                    "gate_evidence": "focused-check-a",
                    "next_batch": "implementation-b",
                }
            },
            "topology-evidence-cannot-transition-controller",
        )
    )
    findings.extend(
        validate_topology_evidence(
            {
                "topology_evidence": {
                    "topology_revision": 0,
                    "node_id": "",
                    "node_status": "running",
                    "dependency_observed": "passed",
                    "gate_evidence": "focused-check-a",
                }
            },
            "topology-evidence-validates-identity-and-status",
        )
    )
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
        "topology-cycle-rejected",
        "ready-frontier-unsettled-dependency",
        "ready-frontier-duplicate-node",
        "ready-frontier-completed-node",
        "ready-frontier-dispatched-unsettled-node",
        "ready-frontier-overlap",
        "replan-cannot-rewrite-dispatched-node",
        "topology-evidence-cannot-transition-controller",
        "topology-evidence-validates-identity-and-status",
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
