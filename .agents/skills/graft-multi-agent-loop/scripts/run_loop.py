#!/usr/bin/env python3
"""Run repeated fresh-session graft multi-agent slices until closeout stops."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
import tempfile
import textwrap
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any


NEXT_PROMPT_PREFIX = "Next-session startup prompt:"
DEFAULT_VALIDATION_MODE = "task-required"
JSON_BLOCK_PATTERN = re.compile(r"```json\s*(\{.*?\})\s*```", re.DOTALL)


class LoopError(RuntimeError):
    """Raised when the loop runner cannot continue safely."""


@dataclass
class BudgetLimits:
    max_rounds: int
    max_files_changed: int
    max_commits: int
    max_runtime_minutes: int
    validation_mode: str
    stop_on_validation_failure: bool
    require_clean_worktree: bool
    scope_expansion_policy: str
    allowed_scopes: list[str] = field(default_factory=list)


@dataclass
class BudgetUsage:
    rounds: int = 0
    files_changed: int = 0
    commits: int = 0
    runtime_minutes: int = 0

    @classmethod
    def from_mapping(cls, data: dict[str, Any] | None) -> "BudgetUsage":
        if not isinstance(data, dict):
            return cls()
        return cls(
            rounds=_coerce_non_negative_int(data.get("rounds"), 0),
            files_changed=_coerce_non_negative_int(data.get("files_changed"), 0),
            commits=_coerce_non_negative_int(data.get("commits"), 0),
            runtime_minutes=_coerce_non_negative_int(data.get("runtime_minutes"), 0),
        )

    def add(self, other: "BudgetUsage") -> None:
        self.rounds += other.rounds
        self.files_changed += other.files_changed
        self.commits += other.commits
        self.runtime_minutes += other.runtime_minutes

    def to_dict(self) -> dict[str, int]:
        return {
            "rounds": self.rounds,
            "files_changed": self.files_changed,
            "commits": self.commits,
            "runtime_minutes": self.runtime_minutes,
        }


@dataclass
class ValidationResult:
    status: str
    commands: list[str]
    note: str | None

    @classmethod
    def from_mapping(cls, data: dict[str, Any] | None) -> "ValidationResult":
        if not isinstance(data, dict):
            return cls(status="not_run", commands=[], note="JSON closeout missing validation details.")
        commands = data.get("commands")
        return cls(
            status=str(data.get("status") or "not_run"),
            commands=[str(item) for item in commands] if isinstance(commands, list) else [],
            note=str(data["note"]) if data.get("note") is not None else None,
        )

    def to_dict(self) -> dict[str, Any]:
        return {"status": self.status, "commands": self.commands, "note": self.note}


@dataclass
class CommitResult:
    created: bool
    sha: str | None
    title: str | None

    @classmethod
    def from_mapping(cls, data: dict[str, Any] | None) -> "CommitResult":
        if not isinstance(data, dict):
            return cls(created=False, sha=None, title=None)
        return cls(
            created=bool(data.get("created", False)),
            sha=str(data["sha"]) if data.get("sha") is not None else None,
            title=str(data["title"]) if data.get("title") is not None else None,
        )

    def to_dict(self) -> dict[str, Any]:
        return {"created": self.created, "sha": self.sha, "title": self.title}


@dataclass
class CloseoutResult:
    closeout_status: str
    continue_loop: bool
    next_prompt: str | None
    stop_reason: str | None
    validation: ValidationResult
    commit: CommitResult
    consumed_budget: BudgetUsage
    remaining_budget: BudgetUsage
    scope_expanded: bool
    risk_level: str
    raw_message: str
    parse_mode: str

    def to_dict(self) -> dict[str, Any]:
        return {
            "closeout_status": self.closeout_status,
            "continue": self.continue_loop,
            "next_prompt": self.next_prompt,
            "stop_reason": self.stop_reason,
            "validation": self.validation.to_dict(),
            "commit": self.commit.to_dict(),
            "consumed_budget": self.consumed_budget.to_dict(),
            "remaining_budget": self.remaining_budget.to_dict(),
            "scope_expanded": self.scope_expanded,
            "risk_level": self.risk_level,
            "parse_mode": self.parse_mode,
        }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Loop graft-multi-agent-task through fresh Codex sessions.")
    parser.add_argument("--repo-root", default=".", help="Repository root for Codex sessions and git checks.")
    parser.add_argument("--task-file", help="Path to a markdown or text file containing the initial task prompt.")
    parser.add_argument("--task-text", help="Inline initial task prompt. Mutually exclusive with --task-file.")
    parser.add_argument("--task-class", required=True, help="Inherited task class for the child session.")
    parser.add_argument("--recovery-source", required=True, help="Inherited recovery source for the child session.")
    parser.add_argument("--owned-scope", required=True, help="Owned scope for the looped slice.")
    parser.add_argument("--allowed-scope", action="append", dest="allowed_scopes", default=[], help="Allowed scope entry. Repeatable.")
    parser.add_argument("--max-rounds", type=int, default=5)
    parser.add_argument("--max-files-changed", type=int, default=30)
    parser.add_argument("--max-commits", type=int, default=3)
    parser.add_argument("--max-runtime-minutes", type=int, default=90)
    parser.add_argument("--validation-mode", default=DEFAULT_VALIDATION_MODE)
    parser.add_argument("--stop-on-validation-failure", action=argparse.BooleanOptionalAction, default=True)
    parser.add_argument("--require-clean-worktree", action=argparse.BooleanOptionalAction, default=True)
    parser.add_argument("--scope-expansion-policy", choices=["forbid", "allow"], default="forbid")
    parser.add_argument("--codex-bin", default="codex", help="Codex CLI binary to execute.")
    parser.add_argument("--model", help="Optional Codex model override for child sessions.")
    parser.add_argument("--result-json", help="Write the aggregated loop result to this file.")
    parser.add_argument("--round-output-dir", help="Directory where per-round prompt/message artifacts should be written.")
    parser.add_argument("--dangerously-bypass-approvals-and-sandbox", action="store_true", help="Pass through to codex exec.")
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    try:
        result = run_loop(args)
    except LoopError as exc:
        print(f"Loop failed: {exc}", file=sys.stderr)
        return 1
    if args.result_json:
        result_path = Path(args.result_json)
        result_path.parent.mkdir(parents=True, exist_ok=True)
        result_path.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 0


def run_loop(args: argparse.Namespace) -> dict[str, Any]:
    repo_root = Path(args.repo_root).resolve()
    ensure_repo_root(repo_root)
    ensure_task_input(args.task_file, args.task_text)

    limits = BudgetLimits(
        max_rounds=_require_positive_int(args.max_rounds, "max_rounds"),
        max_files_changed=_require_non_negative_int(args.max_files_changed, "max_files_changed"),
        max_commits=_require_non_negative_int(args.max_commits, "max_commits"),
        max_runtime_minutes=_require_positive_int(args.max_runtime_minutes, "max_runtime_minutes"),
        validation_mode=args.validation_mode,
        stop_on_validation_failure=bool(args.stop_on_validation_failure),
        require_clean_worktree=bool(args.require_clean_worktree),
        scope_expansion_policy=args.scope_expansion_policy,
        allowed_scopes=list(args.allowed_scopes or [args.owned_scope]),
    )

    if limits.require_clean_worktree:
        ensure_clean_worktree(repo_root)

    task_text = load_task_text(args.task_file, args.task_text)
    aggregate = BudgetUsage()
    round_results: list[dict[str, Any]] = []
    seen_prompts: set[str] = set()
    next_prompt = task_text
    stop_reason = "completed_without_handoff"
    final_status = "completed_no_handoff"

    output_dir = Path(args.round_output_dir).resolve() if args.round_output_dir else None
    if output_dir is not None:
        output_dir.mkdir(parents=True, exist_ok=True)

    started = time.monotonic()
    for round_index in range(1, limits.max_rounds + 1):
        remaining_before = compute_remaining_budget(limits, aggregate)
        prompt_text = build_round_prompt(
            round_index=round_index,
            task_class=args.task_class,
            recovery_source=args.recovery_source,
            owned_scope=args.owned_scope,
            allowed_scopes=limits.allowed_scopes,
            validation_mode=limits.validation_mode,
            remaining_budget=remaining_before,
            task_prompt=next_prompt,
        )
        round_started = time.monotonic()
        message = run_codex_exec(
            codex_bin=args.codex_bin,
            repo_root=repo_root,
            prompt_text=prompt_text,
            model=args.model,
            dangerously_bypass=args.dangerously_bypass_approvals_and_sandbox,
            output_dir=output_dir,
            round_index=round_index,
        )
        duration_minutes = max(1, int(round((time.monotonic() - round_started) / 60.0)))
        closeout = parse_closeout_message(
            message=message,
            default_runtime_minutes=duration_minutes,
            aggregate=aggregate,
            limits=limits,
        )
        aggregate.add(closeout.consumed_budget)
        round_record = closeout.to_dict()
        round_record["round"] = round_index
        round_results.append(round_record)

        stop_decision = evaluate_stop_condition(
            closeout=closeout,
            aggregate=aggregate,
            limits=limits,
            seen_prompts=seen_prompts,
        )
        final_status = closeout.closeout_status
        if stop_decision is not None:
            stop_reason = stop_decision
            break
        assert closeout.next_prompt is not None
        seen_prompts.add(closeout.next_prompt)
        next_prompt = closeout.next_prompt
    else:
        total_runtime_minutes = max(1, int(round((time.monotonic() - started) / 60.0)))
        aggregate.runtime_minutes = max(aggregate.runtime_minutes, total_runtime_minutes)
        stop_reason = "max_rounds_exhausted"

    remaining = compute_remaining_budget(limits, aggregate)
    result = {
        "status": final_status,
        "stop_reason": stop_reason,
        "total_rounds": len(round_results),
        "consumed_budget": aggregate.to_dict(),
        "remaining_budget": remaining.to_dict(),
        "limits": {
            "max_rounds": limits.max_rounds,
            "max_files_changed": limits.max_files_changed,
            "max_commits": limits.max_commits,
            "max_runtime_minutes": limits.max_runtime_minutes,
            "validation_mode": limits.validation_mode,
            "stop_on_validation_failure": limits.stop_on_validation_failure,
            "require_clean_worktree": limits.require_clean_worktree,
            "scope_expansion_policy": limits.scope_expansion_policy,
            "allowed_scopes": limits.allowed_scopes,
        },
        "rounds": round_results,
    }
    return result


def ensure_repo_root(repo_root: Path) -> None:
    if not repo_root.exists():
        raise LoopError(f"Repository root does not exist: {repo_root}")
    if not (repo_root / ".git").exists():
        raise LoopError(f"Repository root does not look like a git worktree: {repo_root}")


def ensure_task_input(task_file: str | None, task_text: str | None) -> None:
    if bool(task_file) == bool(task_text):
        raise LoopError("Provide exactly one of --task-file or --task-text.")


def ensure_clean_worktree(repo_root: Path) -> None:
    result = subprocess.run(
        ["git", "status", "--short"],
        cwd=repo_root,
        capture_output=True,
        text=True,
        check=True,
    )
    if result.stdout.strip():
        raise LoopError("Refusing to start with a dirty worktree while --require-clean-worktree is enabled.")


def load_task_text(task_file: str | None, task_text: str | None) -> str:
    if task_file:
        return Path(task_file).read_text(encoding="utf-8").strip()
    assert task_text is not None
    return task_text.strip()


def build_round_prompt(
    *,
    round_index: int,
    task_class: str,
    recovery_source: str,
    owned_scope: str,
    allowed_scopes: list[str],
    validation_mode: str,
    remaining_budget: BudgetUsage,
    task_prompt: str,
) -> str:
    allowed_scope_text = "\n".join(f"- {scope}" for scope in allowed_scopes)
    return textwrap.dedent(
        f"""
        Use $graft-multi-agent-task for this bounded Graft slice.

        Governance source: root AGENTS.md
        Task class: {task_class}
        Recovery source: {recovery_source}
        Owned scope: {owned_scope}
        Loop round: {round_index}

        Allowed scopes:
        {allowed_scope_text}

        Remaining execution budget:
        - rounds: {remaining_budget.rounds}
        - files_changed: {remaining_budget.files_changed}
        - commits: {remaining_budget.commits}
        - runtime_minutes: {remaining_budget.runtime_minutes}
        - validation_mode: {validation_mode}

        Execute the slice, then close out through graft-task-closeout.

        Human-readable closeout rules:
        - Keep the closeout concise and preferably in Chinese.
        - If another future round is required, include exactly one line starting with:
          {NEXT_PROMPT_PREFIX} 
        - If no further round is needed, omit that line.

        Machine-readable closeout rules:
        - End with one fenced ```json block.
        - The JSON object must include:
          closeout_status, continue, next_prompt, stop_reason, validation, commit,
          consumed_budget, remaining_budget, scope_expanded, risk_level
        - Use English field names.
        - If continue is true, next_prompt must be non-empty.
        - If continue is false, next_prompt must be null.
        - consumed_budget must describe only this round.
        - remaining_budget must describe the budget after this round.
        - risk_level must be one of low, medium, high.

        Current task prompt:
        {task_prompt}
        """
    ).strip() + "\n"


def run_codex_exec(
    *,
    codex_bin: str,
    repo_root: Path,
    prompt_text: str,
    model: str | None,
    dangerously_bypass: bool,
    output_dir: Path | None,
    round_index: int,
) -> str:
    if output_dir is None:
        with tempfile.TemporaryDirectory(prefix="graft-multi-agent-loop-") as temp_dir:
            round_dir = Path(temp_dir)
            return _run_codex_exec_once(
                codex_bin=codex_bin,
                repo_root=repo_root,
                prompt_text=prompt_text,
                model=model,
                dangerously_bypass=dangerously_bypass,
                round_dir=round_dir,
                round_index=round_index,
            )

    round_dir = output_dir / f"round-{round_index:02d}"
    round_dir.mkdir(parents=True, exist_ok=True)
    return _run_codex_exec_once(
        codex_bin=codex_bin,
        repo_root=repo_root,
        prompt_text=prompt_text,
        model=model,
        dangerously_bypass=dangerously_bypass,
        round_dir=round_dir,
        round_index=round_index,
    )


def _run_codex_exec_once(
    *,
    codex_bin: str,
    repo_root: Path,
    prompt_text: str,
    model: str | None,
    dangerously_bypass: bool,
    round_dir: Path,
    round_index: int,
) -> str:
    prompt_path = round_dir / "prompt.txt"
    message_path = round_dir / "last_message.txt"
    prompt_path.write_text(prompt_text, encoding="utf-8")

    cmd = [codex_bin, "exec", "--ephemeral", "--cd", str(repo_root), "-o", str(message_path)]
    if model:
        cmd.extend(["--model", model])
    if dangerously_bypass:
        cmd.append("--dangerously-bypass-approvals-and-sandbox")
    cmd.extend(["-"])

    result = subprocess.run(
        cmd,
        input=prompt_text,
        cwd=repo_root,
        capture_output=True,
        text=True,
        check=False,
    )

    if result.returncode != 0:
        stderr = result.stderr.strip() or result.stdout.strip()
        raise LoopError(f"codex exec failed on round {round_index}: {stderr}")
    if not message_path.exists():
        raise LoopError(f"codex exec did not write the last message file on round {round_index}.")
    message = message_path.read_text(encoding="utf-8").strip()
    if not message:
        raise LoopError(f"codex exec returned an empty last message on round {round_index}.")
    return message


def parse_closeout_message(
    *,
    message: str,
    default_runtime_minutes: int,
    aggregate: BudgetUsage,
    limits: BudgetLimits,
) -> CloseoutResult:
    json_block = extract_json_block(message)
    keyword_prompt = extract_keyword_prompt(message)
    if json_block is not None:
        try:
            data = json.loads(json_block)
        except json.JSONDecodeError as exc:
            raise LoopError(f"Closeout JSON block is malformed: {exc}") from exc
        if not isinstance(data, dict):
            raise LoopError("Closeout JSON block must be an object.")
        return closeout_from_json(
            data=data,
            raw_message=message,
            keyword_prompt=keyword_prompt,
            default_runtime_minutes=default_runtime_minutes,
            aggregate=aggregate,
            limits=limits,
        )

    continue_loop = keyword_prompt is not None
    consumed = BudgetUsage(rounds=1, runtime_minutes=default_runtime_minutes)
    new_aggregate = BudgetUsage(**aggregate.to_dict())
    new_aggregate.add(consumed)
    return CloseoutResult(
        closeout_status="handoff_only" if continue_loop else "completed_no_handoff",
        continue_loop=continue_loop,
        next_prompt=keyword_prompt,
        stop_reason=None if continue_loop else "no_next_prompt",
        validation=ValidationResult(status="not_run", commands=[], note="JSON closeout block missing; fell back to keyword parsing."),
        commit=CommitResult(created=False, sha=None, title=None),
        consumed_budget=consumed,
        remaining_budget=compute_remaining_budget(limits, new_aggregate),
        scope_expanded=False,
        risk_level="medium" if continue_loop else "low",
        raw_message=message,
        parse_mode="keyword_fallback",
    )


def closeout_from_json(
    *,
    data: dict[str, Any],
    raw_message: str,
    keyword_prompt: str | None,
    default_runtime_minutes: int,
    aggregate: BudgetUsage,
    limits: BudgetLimits,
) -> CloseoutResult:
    continue_loop = bool(data.get("continue"))
    next_prompt = data.get("next_prompt")
    if next_prompt is not None:
        next_prompt = str(next_prompt).strip() or None
    if continue_loop and not next_prompt:
        raise LoopError("Closeout JSON set continue=true but next_prompt is empty.")
    if not continue_loop:
        next_prompt = None

    if keyword_prompt is not None and next_prompt is not None and keyword_prompt != next_prompt:
        raise LoopError("Closeout JSON next_prompt does not match the keyword startup prompt line.")

    consumed = BudgetUsage.from_mapping(data.get("consumed_budget"))
    if consumed.rounds == 0:
        consumed.rounds = 1
    if consumed.runtime_minutes == 0:
        consumed.runtime_minutes = default_runtime_minutes
    remaining = BudgetUsage.from_mapping(data.get("remaining_budget"))
    if remaining.rounds == 0 and remaining.files_changed == 0 and remaining.commits == 0 and remaining.runtime_minutes == 0:
        prospective = BudgetUsage(**aggregate.to_dict())
        prospective.add(consumed)
        remaining = compute_remaining_budget(limits, prospective)

    return CloseoutResult(
        closeout_status=str(data.get("closeout_status") or "handoff_only"),
        continue_loop=continue_loop,
        next_prompt=next_prompt,
        stop_reason=str(data["stop_reason"]) if data.get("stop_reason") is not None else None,
        validation=ValidationResult.from_mapping(data.get("validation")),
        commit=CommitResult.from_mapping(data.get("commit")),
        consumed_budget=consumed,
        remaining_budget=remaining,
        scope_expanded=bool(data.get("scope_expanded", False)),
        risk_level=str(data.get("risk_level") or "medium"),
        raw_message=raw_message,
        parse_mode="json",
    )


def extract_json_block(message: str) -> str | None:
    matches = list(JSON_BLOCK_PATTERN.finditer(message))
    if not matches:
        return None
    return matches[-1].group(1)


def extract_keyword_prompt(message: str) -> str | None:
    for line in message.splitlines():
        if line.startswith(NEXT_PROMPT_PREFIX):
            prompt = line[len(NEXT_PROMPT_PREFIX) :].strip()
            return prompt or None
    return None


def compute_remaining_budget(limits: BudgetLimits, aggregate: BudgetUsage) -> BudgetUsage:
    return BudgetUsage(
        rounds=max(limits.max_rounds - aggregate.rounds, 0),
        files_changed=max(limits.max_files_changed - aggregate.files_changed, 0),
        commits=max(limits.max_commits - aggregate.commits, 0),
        runtime_minutes=max(limits.max_runtime_minutes - aggregate.runtime_minutes, 0),
    )


def evaluate_stop_condition(
    *,
    closeout: CloseoutResult,
    aggregate: BudgetUsage,
    limits: BudgetLimits,
    seen_prompts: set[str],
) -> str | None:
    if closeout.scope_expanded and limits.scope_expansion_policy == "forbid":
        return "scope_expanded"
    if closeout.risk_level.lower() == "high":
        return "risk_level_high"
    if (
        limits.stop_on_validation_failure
        and closeout.validation.status.lower() == "failed"
    ):
        return "validation_failed"
    if closeout.continue_loop and closeout.next_prompt in seen_prompts:
        return "repeated_next_prompt"
    if aggregate.files_changed >= limits.max_files_changed and limits.max_files_changed > 0:
        return "max_files_changed_exhausted"
    if aggregate.commits >= limits.max_commits and limits.max_commits > 0:
        return "max_commits_exhausted"
    if aggregate.runtime_minutes >= limits.max_runtime_minutes:
        return "max_runtime_minutes_exhausted"
    if not closeout.continue_loop:
        return closeout.stop_reason or "closeout_requested_stop"
    if not closeout.next_prompt:
        return "missing_next_prompt"
    return None


def _coerce_non_negative_int(value: Any, default: int) -> int:
    if value is None:
        return default
    try:
        result = int(value)
    except (TypeError, ValueError):
        return default
    return max(result, 0)


def _require_positive_int(value: int, name: str) -> int:
    if value <= 0:
        raise LoopError(f"{name} must be positive.")
    return value


def _require_non_negative_int(value: int, name: str) -> int:
    if value < 0:
        raise LoopError(f"{name} must be non-negative.")
    return value


if __name__ == "__main__":
    sys.exit(main())
