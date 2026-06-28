#!/usr/bin/env python3
"""Evaluate Graft Quality Policy from eff-u-code JSON reports."""

from __future__ import annotations

import argparse
import fnmatch
import json
import os
import pathlib
import subprocess
import sys
import tempfile
from dataclasses import dataclass
from typing import Any
import tarfile


REPO_ROOT = pathlib.Path(__file__).resolve().parents[1]
DEFAULT_GATE_CONFIG = REPO_ROOT / "scripts" / "eff-u-code.gate.json"
DEFAULT_EFF_U_CODE_CONFIG = REPO_ROOT / "scripts" / "eff-u-code.example.json"

class GateConfigError(RuntimeError):
    """Raised when the gate config is invalid."""


@dataclass(frozen=True)
class RuleEvaluation:
    rule: str
    metric: str
    status: str
    threshold: float | None
    regression: float | None
    current_score: float
    baseline_score: float | None
    details: str
    noise_reason: str | None = None


def load_json(path: pathlib.Path) -> dict[str, Any]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise GateConfigError(f"missing config file: {path}") from exc
    except json.JSONDecodeError as exc:
        raise GateConfigError(f"invalid JSON in {path}: {exc}") from exc
    if not isinstance(value, dict):
        raise GateConfigError(f"config root must be an object: {path}")
    return value


def require_dict(container: dict[str, Any], key: str, *, context: str) -> dict[str, Any]:
    value = container.get(key)
    if not isinstance(value, dict):
        raise GateConfigError(f"{context}.{key} must be an object")
    return value


def require_string(container: dict[str, Any], key: str, *, context: str) -> str:
    value = container.get(key)
    if not isinstance(value, str) or not value.strip():
        raise GateConfigError(f"{context}.{key} must be a non-empty string")
    return value


def require_string_list(container: dict[str, Any], key: str, *, context: str) -> list[str]:
    value = container.get(key)
    if not isinstance(value, list) or not all(isinstance(item, str) and item.strip() for item in value):
        raise GateConfigError(f"{context}.{key} must be a string array")
    return value


def optional_string_list(container: dict[str, Any], key: str, *, context: str) -> list[str]:
    value = container.get(key)
    if value is None:
        return []
    if not isinstance(value, list) or not all(isinstance(item, str) and item.strip() for item in value):
        raise GateConfigError(f"{context}.{key} must be a string array when provided")
    return value


def optional_number(container: dict[str, Any], key: str, *, context: str) -> float | None:
    value = container.get(key)
    if value is None:
        return None
    if not isinstance(value, (int, float)):
        raise GateConfigError(f"{context}.{key} must be a number when provided")
    return float(value)


def run_git(args: list[str]) -> str:
    completed = subprocess.run(
        ["git", *args],
        cwd=REPO_ROOT,
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if completed.returncode != 0:
        raise RuntimeError(completed.stderr.strip() or f"git {' '.join(args)} failed")
    return completed.stdout.strip()


def staged_or_changed_files(diff_filter: str) -> list[str]:
    staged = run_git(["diff", "--cached", "--name-only", f"--diff-filter={diff_filter}"])
    if staged:
        return [line for line in staged.splitlines() if line]

    changed = run_git(["diff", "HEAD", "--name-only", f"--diff-filter={diff_filter}"])
    if changed:
        return [line for line in changed.splitlines() if line]

    return []


def ci_changed_files(diff_filter: str, base_ref_override: str | None = None) -> list[str]:
    local_changed = staged_or_changed_files(diff_filter)
    if local_changed:
        return local_changed

    base_ref = (base_ref_override or os.environ.get("GITHUB_BASE_REF", "")).strip()
    explicit_base_sha = os.environ.get("GRAFT_QUALITY_BASE_SHA", "").strip()
    if explicit_base_sha:
        changed = run_git(["diff", "--name-only", f"--diff-filter={diff_filter}", f"{explicit_base_sha}...HEAD"])
        if changed:
            return [line for line in changed.splitlines() if line]

    if base_ref:
        try:
            merge_base = run_git(["merge-base", "HEAD", f"origin/{base_ref}"])
        except RuntimeError:
            merge_base = ""
        if merge_base:
            changed = run_git(["diff", "--name-only", f"--diff-filter={diff_filter}", f"{merge_base}...HEAD"])
            if changed:
                return [line for line in changed.splitlines() if line]

    head_sha = os.environ.get("GITHUB_SHA", "").strip()
    if head_sha:
        try:
            previous = run_git(["rev-parse", f"{head_sha}^"])
        except RuntimeError:
            previous = ""
        if previous:
            changed = run_git(["diff", "--name-only", f"--diff-filter={diff_filter}", f"{previous}...{head_sha}"])
            if changed:
                return [line for line in changed.splitlines() if line]

    tracked = run_git(["ls-files"])
    return [line for line in tracked.splitlines() if line]


def matches_any(path: str, patterns: list[str]) -> bool:
    return any(fnmatch.fnmatch(path, pattern) for pattern in patterns)


def load_file_text(repo_path: str) -> str:
    try:
        return (REPO_ROOT / repo_path).read_text(encoding="utf-8")
    except FileNotFoundError:
        return ""


def is_reactive_tracking_noise(repo_path: str, metric_name: str, details: str) -> bool:
    if metric_name != "error_handling":
        return False
    if "错误被忽略" not in details and "ignored" not in details.lower():
        return False
    source = load_file_text(repo_path)
    if not source:
        return False
    return "void version.value" in source or "void entry.changeTick" in source


def noise_reason(repo_path: str, rule_name: str, metric_name: str, rule_config: dict[str, Any], details: str) -> str | None:
    patterns = optional_string_list(rule_config, "noiseExcludes", context=f"gateRules.{rule_name}")
    if not matches_any(repo_path, patterns):
        return None
    if is_reactive_tracking_noise(repo_path, metric_name, details):
        return "reactive-tracking read pattern"
    if rule_name == "duplication":
        return "declarative duplication surface"
    return "bounded policy noise"


def target_config(scope: str, gate_config: dict[str, Any]) -> dict[str, Any]:
    targets = require_dict(gate_config, "targets", context="gate_config")
    return require_dict(targets, scope, context="gate_config.targets")


def scope_root(scope: str, gate_config: dict[str, Any]) -> str:
    return require_string(target_config(scope, gate_config), "root", context=f"gate_config.targets.{scope}")


def scope_patterns(scope: str, gate_config: dict[str, Any]) -> tuple[list[str], list[str]]:
    target = target_config(scope, gate_config)
    include = require_string_list(target, "include", context=f"gate_config.targets.{scope}")
    exclude = require_string_list(target, "exclude", context=f"gate_config.targets.{scope}")
    return include, exclude


def path_matches_scope(path: str, scope: str, gate_config: dict[str, Any]) -> bool:
    include, exclude = scope_patterns(scope, gate_config)
    if matches_any(path, exclude):
        return False
    return matches_any(path, include)


def classify_scope(path: str, gate_config: dict[str, Any]) -> str | None:
    targets = require_dict(gate_config, "targets", context="gate_config")
    for scope, target in targets.items():
        if not isinstance(target, dict):
            raise GateConfigError(f"gate_config.targets.{scope} must be an object")
        if path_matches_scope(path, scope, gate_config):
            return scope
    return None


def relative_path_for_scope(path: str, scope: str, gate_config: dict[str, Any]) -> str:
    root_path = pathlib.PurePosixPath(scope_root(scope, gate_config))
    full_path = pathlib.PurePosixPath(path)
    try:
        return full_path.relative_to(root_path).as_posix()
    except ValueError:
        return path


def repository_path_for_scope(relative_path: str, scope: str, gate_config: dict[str, Any]) -> str:
    return (pathlib.PurePosixPath(scope_root(scope, gate_config)) / pathlib.PurePosixPath(relative_path)).as_posix()


def list_scope_files_on_disk(scope: str, gate_config: dict[str, Any]) -> list[str]:
    root_dir = REPO_ROOT / scope_root(scope, gate_config)
    if not root_dir.is_dir():
        return []

    files: list[str] = []
    for path in root_dir.rglob("*"):
        if not path.is_file():
            continue
        repo_path = path.relative_to(REPO_ROOT).as_posix()
        if path_matches_scope(repo_path, scope, gate_config):
            files.append(repo_path)
    return sorted(files)


def to_scope_relative_pattern(pattern: str, scope: str, gate_config: dict[str, Any]) -> str:
    root = scope_root(scope, gate_config)
    prefix = f"{root}/"
    if pattern.startswith(prefix):
        return pattern[len(prefix):]
    return pattern


def scope_relative_excludes(scope: str, gate_config: dict[str, Any]) -> list[str]:
    _, exclude = scope_patterns(scope, gate_config)
    return [to_scope_relative_pattern(pattern, scope, gate_config) for pattern in exclude]


def build_eff_u_code_overrides(gate_config: dict[str, Any]) -> dict[str, Any]:
    overrides = {"defaults": {}, "targets": {}}
    targets = require_dict(gate_config, "targets", context="gate_config")
    for scope, target in targets.items():
        target_dict = require_dict(targets, scope, context="gate_config.targets")
        overrides["targets"][scope] = {
            "path": scope_root(scope, gate_config),
            "exclude": scope_relative_excludes(scope, gate_config),
        }
    return overrides


def run_eff_u_code(scope: str, *, output_dir: pathlib.Path, eff_config_override: pathlib.Path | None, base_ref: str | None = None) -> pathlib.Path:
    command = [
        sys.executable,
        str(REPO_ROOT / "scripts" / "run_eff_u_code.py"),
        scope,
        "--format",
        "json",
        "--output-dir",
        str(output_dir),
    ]
    if eff_config_override is not None:
        command.extend(["--config", str(eff_config_override)])
    env = os.environ.copy()
    if base_ref:
        env["GRAFT_EFF_U_CODE_BASE_REF"] = base_ref
    completed = subprocess.run(command, cwd=REPO_ROOT, env=env, check=False)
    if completed.returncode != 0:
        raise RuntimeError(f"eff-u-code run failed for scope {scope} with exit code {completed.returncode}")
    return output_dir / f"eff-u-code-{scope}.json"


def export_git_snapshot(revision: str, destination: pathlib.Path) -> pathlib.Path:
    destination.mkdir(parents=True, exist_ok=True)
    archive = subprocess.run(
        ["git", "archive", "--format=tar", revision],
        cwd=REPO_ROOT,
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if archive.returncode != 0:
        stderr = archive.stderr.decode("utf-8", errors="replace").strip()
        raise RuntimeError(stderr or f"git archive failed for revision {revision}")
    tar_path = destination / "snapshot.tar"
    tar_path.write_bytes(archive.stdout)
    extract_dir = destination / "snapshot"
    extract_dir.mkdir(parents=True, exist_ok=True)
    with tarfile.open(tar_path) as bundle:
        bundle.extractall(extract_dir)
    tar_path.unlink(missing_ok=True)
    return extract_dir


def build_snapshot_eff_config(gate_config: dict[str, Any], snapshot_root: pathlib.Path) -> dict[str, Any]:
    config = {"defaults": {}, "targets": {}}
    targets = require_dict(gate_config, "targets", context="gate_config")
    for scope, raw_target in targets.items():
        target = require_dict(targets, scope, context="gate_config.targets")
        root = require_string(target, "root", context=f"gate_config.targets.{scope}")
        config["targets"][scope] = {
            "path": str((snapshot_root / root).resolve()),
            "exclude": scope_relative_excludes(scope, gate_config),
        }
    return config


def load_report(path: pathlib.Path) -> dict[str, Any]:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise RuntimeError(f"missing eff-u-code report: {path}") from exc


def build_file_index(report: dict[str, Any]) -> dict[str, dict[str, Any]]:
    files = report.get("files")
    if not isinstance(files, list):
        raise GateConfigError("eff-u-code report must contain a files array")
    index: dict[str, dict[str, Any]] = {}
    for file_report in files:
        if not isinstance(file_report, dict):
            continue
        path = file_report.get("path")
        if isinstance(path, str) and path:
            index[path] = file_report
    return index


def metric_scores(file_report: dict[str, Any]) -> dict[str, dict[str, Any]]:
    metrics = file_report.get("metrics")
    if not isinstance(metrics, list):
        return {}
    result: dict[str, dict[str, Any]] = {}
    for metric in metrics:
        if not isinstance(metric, dict):
            continue
        name = metric.get("name")
        if isinstance(name, str) and name:
            result[name] = metric
    return result


def evaluate_rule(
    repo_path: str,
    rule_name: str,
    rule_config: dict[str, Any],
    metric_name: str,
    current_metric: dict[str, Any],
    baseline_metric: dict[str, Any] | None,
    *,
    is_new_file: bool,
    scan_mode: str = "changed",
) -> RuleEvaluation:
    current_score = float(current_metric.get("normalizedScore", 0))
    details = str(current_metric.get("details", ""))
    mode = rule_config.get("mode")
    threshold = optional_number(rule_config, "threshold", context=f"gateRules.{rule_name}")
    regression = optional_number(rule_config, "regression", context=f"gateRules.{rule_name}")
    new_file_threshold = optional_number(rule_config, "newFileThreshold", context=f"gateRules.{rule_name}")
    baseline_score = None if baseline_metric is None else float(baseline_metric.get("normalizedScore", 0))
    suppressed = noise_reason(repo_path, rule_name, metric_name, rule_config, details)

    if suppressed is not None:
        return RuleEvaluation(
            rule_name,
            metric_name,
            "suppressed-noise",
            threshold,
            regression,
            current_score,
            baseline_score,
            details,
            suppressed,
        )

    if mode == "display-only":
        return RuleEvaluation(rule_name, metric_name, "display-only", threshold, regression, current_score, baseline_score, details)
    if mode == "advisory":
        return RuleEvaluation(rule_name, metric_name, "advisory", threshold, regression, current_score, baseline_score, details)

    if scan_mode == "project":
        gate_threshold = threshold if threshold is not None else new_file_threshold
        if gate_threshold is None:
            raise GateConfigError(f"gateRules.{rule_name} must define threshold or newFileThreshold")
        if current_score < gate_threshold:
            return RuleEvaluation(rule_name, metric_name, "fail", gate_threshold, regression, current_score, baseline_score, details)
        return RuleEvaluation(rule_name, metric_name, "pass", gate_threshold, regression, current_score, baseline_score, details)

    if is_new_file:
        gate_threshold = new_file_threshold if new_file_threshold is not None else threshold
        if gate_threshold is None:
            raise GateConfigError(f"gateRules.{rule_name} must define threshold or newFileThreshold")
        if current_score < gate_threshold:
            return RuleEvaluation(rule_name, metric_name, "fail", gate_threshold, regression, current_score, baseline_score, details)
        return RuleEvaluation(rule_name, metric_name, "pass", gate_threshold, regression, current_score, baseline_score, details)

    if baseline_score is None:
        return RuleEvaluation(rule_name, metric_name, "pass", threshold, regression, current_score, baseline_score, details)

    if threshold is None or regression is None:
        raise GateConfigError(f"gateRules.{rule_name} must define threshold and regression")

    if current_score < threshold and (baseline_score - current_score) >= regression:
        return RuleEvaluation(rule_name, metric_name, "fail", threshold, regression, current_score, baseline_score, details)

    if current_score < threshold:
        return RuleEvaluation(rule_name, metric_name, "legacy-warning", threshold, regression, current_score, baseline_score, details)

    if current_score < baseline_score:
        return RuleEvaluation(rule_name, metric_name, "regressed-but-above-threshold", threshold, regression, current_score, baseline_score, details)

    return RuleEvaluation(rule_name, metric_name, "pass", threshold, regression, current_score, baseline_score, details)


def curated_score(file_evaluations: list[RuleEvaluation], gate_config: dict[str, Any]) -> float | None:
    curated = require_dict(gate_config, "curatedScore", context="gate_config")
    participates_in_gate = curated.get("participatesInGate")
    if participates_in_gate not in (False, None):
        raise GateConfigError("gate_config.curatedScore.participatesInGate must remain false; Curated Score is display-only")
    weights = require_dict(curated, "weights", context="gate_config.curatedScore")
    weighted_sum = 0.0
    total_weight = 0.0
    best_by_rule: dict[str, float] = {}
    for evaluation in file_evaluations:
        best_by_rule[evaluation.rule] = min(best_by_rule.get(evaluation.rule, 100.0), evaluation.current_score)
    for rule_name, weight_value in weights.items():
        if not isinstance(weight_value, (int, float)):
            raise GateConfigError(f"gate_config.curatedScore.weights.{rule_name} must be a number")
        weight = float(weight_value)
        if weight <= 0:
            continue
        score = best_by_rule.get(rule_name)
        if score is None:
            continue
        weighted_sum += score * weight
        total_weight += weight
    if total_weight == 0:
        return None
    return weighted_sum / total_weight


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Evaluate Graft Quality Policy from eff-u-code reports.")
    parser.add_argument("--config", default=str(DEFAULT_GATE_CONFIG), help="path to gate config JSON")
    parser.add_argument(
        "--eff-u-code-config",
        default=str(DEFAULT_EFF_U_CODE_CONFIG),
        help="base eff-u-code config JSON used by the local wrapper",
    )
    parser.add_argument(
        "--scan-mode",
        choices=("changed", "project"),
        default="changed",
        help="changed: PR/incremental gate semantics; project: full in-scope local governance scan",
    )
    parser.add_argument("--base-ref", help="explicit base branch name for changed-file and baseline resolution")
    parser.add_argument("--output-json", help="write evaluation report JSON to this path")
    parser.add_argument("--report-dir", help="reuse or write raw eff-u-code JSON reports into this directory")
    parser.add_argument("--scopes", nargs="*", choices=("server", "web"), help="restrict evaluation to scopes")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    temp_ctx: tempfile.TemporaryDirectory[str] | None = None

    try:
        gate_config = load_json(pathlib.Path(args.config))
        requested_scopes = args.scopes or ["server", "web"]
        changed_files: list[str] = []
        scoped_candidates: dict[str, list[str]] = {}

        if args.scan_mode == "changed":
            diff_config = require_dict(gate_config, "changedFiles", context="gate_config")
            diff_filter = require_string(diff_config, "diffFilter", context="gate_config.changedFiles")
            changed_files = ci_changed_files(diff_filter, args.base_ref)

            scoped_changed: dict[str, list[str]] = {}
            for path in changed_files:
                scope = classify_scope(path, gate_config)
                if scope is None:
                    continue
                scoped_changed.setdefault(scope, []).append(path)
            scoped_candidates = {scope: scoped_changed.get(scope, []) for scope in requested_scopes}
        else:
            scoped_candidates = {scope: list_scope_files_on_disk(scope, gate_config) for scope in requested_scopes}

        selected_scopes = [scope for scope, paths in scoped_candidates.items() if paths]
        if not selected_scopes:
            if args.scan_mode == "changed":
                reason = "no changed files matched Graft Quality Policy targets"
                print("Graft Quality Policy: skipped (no matched changed files)")
            else:
                reason = "no in-scope files matched Graft Quality Policy targets"
                print("Graft Quality Policy: skipped (no matched in-scope files)")
            result = {
                "status": "skipped",
                "reason": reason,
                "scanMode": args.scan_mode,
                "changedFiles": changed_files,
                "scopes": {},
            }
            if args.output_json:
                output_path = pathlib.Path(args.output_json)
                output_path.parent.mkdir(parents=True, exist_ok=True)
                output_path.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
            return 0

        base_eff_config = load_json(pathlib.Path(args.eff_u_code_config))
        override = build_eff_u_code_overrides(gate_config)
        merged_eff_config = dict(base_eff_config)
        merged_defaults = base_eff_config.get("defaults")
        if not isinstance(merged_defaults, dict):
            merged_defaults = {}
        else:
            merged_defaults = dict(merged_defaults)
        if args.scan_mode == "project":
            project_top = max((len(paths) for paths in scoped_candidates.values() if paths), default=1)
            existing_top = merged_defaults.get("top")
            existing_top_value = existing_top if isinstance(existing_top, int) and existing_top > 0 else 0
            merged_defaults["top"] = max(existing_top_value, project_top)
        merged_eff_config["defaults"] = merged_defaults
        merged_eff_config["targets"] = override["targets"]

        if args.report_dir:
            report_dir = pathlib.Path(args.report_dir).resolve()
            report_dir.mkdir(parents=True, exist_ok=True)
        else:
            temp_ctx = tempfile.TemporaryDirectory(prefix="graft-eff-u-code-gate-")
            report_dir = pathlib.Path(temp_ctx.name)

        eff_override_path = report_dir / "eff-u-code-gate.override.json"
        eff_override_path.write_text(json.dumps(merged_eff_config, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

        head_reports: dict[str, dict[str, Any]] = {}
        base_reports: dict[str, dict[str, Any]] = {}
        base_ref = args.base_ref or os.environ.get("GITHUB_BASE_REF", "").strip() or "main"

        for scope in selected_scopes:
            head_path = run_eff_u_code(scope, output_dir=report_dir, eff_config_override=eff_override_path)
            head_reports[scope] = load_report(head_path)

        if args.scan_mode == "changed" and base_ref:
            subprocess.run(
                ["git", "fetch", "--no-tags", "--prune", "origin", base_ref],
                cwd=REPO_ROOT,
                check=False,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
            )

            current_head = run_git(["rev-parse", "HEAD"])
            try:
                merge_base = run_git(["merge-base", current_head, f"origin/{base_ref}"])
            except RuntimeError:
                merge_base = ""
            if merge_base:
                snapshot_root = export_git_snapshot(merge_base, report_dir / "baseline")
                snapshot_eff_config = build_snapshot_eff_config(gate_config, snapshot_root)
                snapshot_eff_config_path = report_dir / "eff-u-code-gate.baseline.override.json"
                snapshot_eff_config_path.write_text(
                    json.dumps(snapshot_eff_config, ensure_ascii=False, indent=2) + "\n",
                    encoding="utf-8",
                )
                for scope in selected_scopes:
                    base_path = run_eff_u_code(
                        scope,
                        output_dir=report_dir / "baseline-reports",
                        eff_config_override=snapshot_eff_config_path,
                        base_ref=base_ref,
                    )
                    base_reports[scope] = load_report(base_path)

        gate_rules = require_dict(gate_config, "gateRules", context="gate_config")
        scope_results: dict[str, Any] = {}
        overall_failures = 0

        for scope in selected_scopes:
            head_index = build_file_index(head_reports[scope])
            base_index = build_file_index(base_reports.get(scope, {"files": []}))
            file_results = []
            unreported_files: list[str] = []
            for candidate_path in scoped_candidates[scope]:
                relative = relative_path_for_scope(candidate_path, scope, gate_config)
                current_file = head_index.get(relative)
                if current_file is None:
                    unreported_files.append(candidate_path)
                    continue
                baseline_file = base_index.get(relative) if args.scan_mode == "changed" else None
                current_metrics = metric_scores(current_file)
                baseline_metrics = metric_scores(baseline_file) if baseline_file is not None else {}
                evaluations: list[RuleEvaluation] = []
                for rule_name, raw_rule_config in gate_rules.items():
                    if not isinstance(raw_rule_config, dict):
                        raise GateConfigError(f"gateRules.{rule_name} must be an object")
                    metrics = require_string_list(raw_rule_config, "metrics", context=f"gateRules.{rule_name}")
                    for metric_name in metrics:
                        current_metric = current_metrics.get(metric_name)
                        if current_metric is None:
                            continue
                        baseline_metric = baseline_metrics.get(metric_name)
                        evaluations.append(
                            evaluate_rule(
                                candidate_path,
                                rule_name,
                                raw_rule_config,
                                metric_name,
                                current_metric,
                                baseline_metric,
                                is_new_file=baseline_file is None,
                                scan_mode=args.scan_mode,
                            )
                        )
                file_curated = curated_score(evaluations, gate_config)
                failures = [evaluation for evaluation in evaluations if evaluation.status == "fail"]
                advisories = [evaluation for evaluation in evaluations if evaluation.status not in {"pass", "display-only"}]
                overall_failures += len(failures)
                file_results.append(
                    {
                        "path": candidate_path,
                        "scope": scope,
                        "isNewFile": baseline_file is None,
                        "curatedScore": file_curated,
                        "gateStatus": "fail" if failures else "pass",
                        "rules": [
                            {
                                "rule": evaluation.rule,
                                "metric": evaluation.metric,
                                "status": evaluation.status,
                                "threshold": evaluation.threshold,
                                "regression": evaluation.regression,
                                "currentScore": evaluation.current_score,
                                "baselineScore": evaluation.baseline_score,
                                "details": evaluation.details,
                                "noiseReason": evaluation.noise_reason,
                            }
                            for evaluation in evaluations
                        ],
                        "advisories": [
                            {
                                "rule": evaluation.rule,
                                "metric": evaluation.metric,
                                "status": evaluation.status,
                                "currentScore": evaluation.current_score,
                                "baselineScore": evaluation.baseline_score,
                                "details": evaluation.details,
                                "noiseReason": evaluation.noise_reason,
                            }
                            for evaluation in advisories
                            if evaluation.status != "fail"
                        ],
                    }
                )
            scope_results[scope] = {
                "changedFiles": scoped_candidates[scope] if args.scan_mode == "changed" else [],
                "candidateFiles": scoped_candidates[scope],
                "unreportedFiles": unreported_files,
                "files": file_results,
            }

        result = {
            "status": "fail" if overall_failures else "pass",
            "policy": "Graft Quality Policy",
            "tool": "eff-u-code",
            "toolRole": "raw-json-source",
            "gateMode": "repository-evaluator",
            "scanMode": args.scan_mode,
            "changedFiles": changed_files,
            "scopes": scope_results,
            "summary": {
                "filesEvaluated": sum(len(scope_results[scope]["files"]) for scope in scope_results),
                "failures": overall_failures,
                "unreportedFiles": sum(len(scope_results[scope]["unreportedFiles"]) for scope in scope_results),
            },
        }

        if args.output_json:
            output_path = pathlib.Path(args.output_json)
            output_path.parent.mkdir(parents=True, exist_ok=True)
            output_path.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

        print(f"Graft Quality Policy ({args.scan_mode})")
        for scope in selected_scopes:
            print(f"[{scope}]")
            scope_file_results = scope_results[scope]["files"]
            if args.scan_mode == "project":
                scope_file_results = [file_result for file_result in scope_file_results if file_result["gateStatus"] == "fail"]
                print(
                    "  summary: "
                    f"evaluated={len(scope_results[scope]['files'])} "
                    f"failing={len(scope_file_results)} "
                    f"unreported={len(scope_results[scope]['unreportedFiles'])}"
                )
                if not scope_file_results:
                    continue
            for file_result in scope_file_results:
                print(file_result["path"])
                failing_rules = [rule for rule in file_result["rules"] if rule["status"] == "fail"]
                display_order = ("complexity", "duplication", "file_size", "structure", "error_handling")
                for rule_name in display_order:
                    matching = [rule for rule in file_result["rules"] if rule["rule"] == rule_name]
                    if not matching:
                        continue
                    status = "✅"
                    if any(rule["status"] == "fail" for rule in matching):
                        status = "❌"
                    elif any(rule["status"] == "suppressed-noise" for rule in matching):
                        status = "🫥"
                    elif any(rule["status"] in {"legacy-warning", "regressed-but-above-threshold"} for rule in matching):
                        status = "⚠️"
                    print(f"  {rule_name}: {status}")
                if file_result["curatedScore"] is not None:
                    print(f"  Curated Score (display-only): {file_result['curatedScore']:.1f}")
                suppressed_rules = [rule for rule in file_result["rules"] if rule["status"] == "suppressed-noise"]
                for suppressed in suppressed_rules:
                    print(
                        f"  SUPPRESSED {suppressed['rule']}::{suppressed['metric']} "
                        f"reason={suppressed['noiseReason']} current={suppressed['currentScore']:.1f} {suppressed['details']}"
                    )
                for failing in failing_rules:
                    baseline = "n/a" if failing["baselineScore"] is None else f"{failing['baselineScore']:.1f}"
                    threshold = "n/a" if failing["threshold"] is None else f"{failing['threshold']:.1f}"
                    print(
                        f"  FAIL {failing['rule']}::{failing['metric']} "
                        f"current={failing['currentScore']:.1f} baseline={baseline} threshold={threshold} {failing['details']}"
                    )
            if args.scan_mode != "project" and scope_results[scope]["unreportedFiles"]:
                print(f"  unreported by eff-u-code: {len(scope_results[scope]['unreportedFiles'])}")

        return 1 if overall_failures else 0
    except GateConfigError as exc:
        print(f"evaluate_eff_u_code_gate.py: {exc}", file=sys.stderr)
        return 2
    except RuntimeError as exc:
        print(f"evaluate_eff_u_code_gate.py: {exc}", file=sys.stderr)
        return 2
    finally:
        if temp_ctx is not None:
            temp_ctx.cleanup()


if __name__ == "__main__":
    raise SystemExit(main())
