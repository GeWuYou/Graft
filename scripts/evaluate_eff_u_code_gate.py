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
    """
    从指定路径读取并解析 JSON 对象。
    
    Parameters:
    	path (pathlib.Path): JSON 文件路径。
    
    Returns:
    	dict[str, Any]: 解析得到的对象根值。
    
    Raises:
    	GateConfigError: 当文件缺失、JSON 无效或根值不是对象时。
    """
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
    """
    获取指定键对应的对象值。
    
    Parameters:
    	container (dict[str, Any]): 待检查的容器。
    	key (str): 要读取的键名。
    	context (str): 用于错误信息的上下文名称。
    
    Returns:
    	dict[str, Any]: 指定键对应的字典。
    
    Raises:
    	GateConfigError: 当键不存在或对应值不是字典时抛出。
    """
    value = container.get(key)
    if not isinstance(value, dict):
        raise GateConfigError(f"{context}.{key} must be an object")
    return value


def require_string(container: dict[str, Any], key: str, *, context: str) -> str:
    """
    获取并验证容器中的非空字符串值。
    
    参数:
    	container (dict[str, Any]): 待检查的映射。
    	key (str): 要读取的键名。
    	context (str): 用于错误信息的配置上下文名称。
    
    返回:
    	str: 对应键的字符串值。
    
    异常:
    	GateConfigError: 当键不存在、值不是字符串或为空白字符串时抛出。
    """
    value = container.get(key)
    if not isinstance(value, str) or not value.strip():
        raise GateConfigError(f"{context}.{key} must be a non-empty string")
    return value


def require_string_list(container: dict[str, Any], key: str, *, context: str) -> list[str]:
    """
    验证并返回指定键对应的字符串数组。
    
    Parameters:
    	container (dict[str, Any]): 待检查的映射。
    	key (str): 要读取的键名。
    	context (str): 错误信息中的上下文名称。
    
    Returns:
    	list[str]: 指定键对应的非空字符串列表。
    
    Raises:
    	GateConfigError: 当键不存在、值不是列表或列表中包含空字符串时抛出。
    """
    value = container.get(key)
    if not isinstance(value, list) or not all(isinstance(item, str) and item.strip() for item in value):
        raise GateConfigError(f"{context}.{key} must be a string array")
    return value


def optional_string_list(container: dict[str, Any], key: str, *, context: str) -> list[str]:
    """
    获取可选字符串数组配置。
    
    Parameters:
    	container (dict[str, Any]): 配置容器。
    	key (str): 要读取的字段名。
    	context (str): 用于错误信息的上下文路径。
    
    Returns:
    	list[str]: 字段缺失或为 `None` 时返回空列表；否则返回已验证的字符串数组。
    """
    value = container.get(key)
    if value is None:
        return []
    if not isinstance(value, list) or not all(isinstance(item, str) and item.strip() for item in value):
        raise GateConfigError(f"{context}.{key} must be a string array when provided")
    return value


def optional_number(container: dict[str, Any], key: str, *, context: str) -> float | None:
    """
    获取可选数值字段，并在存在时将其转换为浮点数。
    
    参数：
    	container (dict[str, Any]): 待读取的容器。
    	key (str): 字段名。
    	context (str): 用于错误信息的上下文名称。
    
    返回：
    	float | None: 字段存在时返回其数值的浮点表示；字段缺失或为 None 时返回 None。
    """
    value = container.get(key)
    if value is None:
        return None
    if not isinstance(value, (int, float)):
        raise GateConfigError(f"{context}.{key} must be a number when provided")
    return float(value)


def run_git(args: list[str]) -> str:
    """
    在仓库根目录执行 Git 命令并返回标准输出。
    
    Parameters:
    	args (list[str]): 传递给 `git` 的参数列表。
    
    Returns:
    	str: 去除首尾空白后的标准输出。
    
    Raises:
    	RuntimeError: 当 Git 命令执行失败时抛出。
    """
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
    """
    获取与给定 diff 过滤条件匹配的暂存或已修改文件列表。
    
    参数:
    	diff_filter (str): 传递给 `git diff --diff-filter` 的筛选条件。
    
    返回:
    	list[str]: 按行分割得到的文件路径列表；若没有匹配文件则返回空列表。
    """
    staged = run_git(["diff", "--cached", "--name-only", f"--diff-filter={diff_filter}"])
    if staged:
        return [line for line in staged.splitlines() if line]

    changed = run_git(["diff", "HEAD", "--name-only", f"--diff-filter={diff_filter}"])
    if changed:
        return [line for line in changed.splitlines() if line]

    return []


def ci_changed_files(diff_filter: str, base_ref_override: str | None = None) -> list[str]:
    """
    获取用于 CI 的候选变更文件列表。
    
    Parameters:
    	diff_filter (str): Git diff 的文件状态过滤条件。
    	base_ref_override (str | None): 覆盖用于解析基准分支的名称。
    
    Returns:
    	list[str]: 按优先级解析得到的变更文件路径列表；若未找到变更，则返回已跟踪文件列表。
    """
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
    """
    判断路径是否匹配任一模式。
    
    Returns:
    	如果路径匹配给定模式列表中的任一模式，则为 `True`，否则为 `False`。
    """
    return any(fnmatch.fnmatch(path, pattern) for pattern in patterns)


def load_file_text(repo_path: str) -> str:
    """
    读取仓库内指定路径的文件文本。
    
    Parameters:
    	repo_path (str): 相对于仓库根目录的文件路径。
    
    Returns:
    	str: 文件内容；如果文件不存在则返回空字符串。
    """
    try:
        return (REPO_ROOT / repo_path).read_text(encoding="utf-8")
    except FileNotFoundError:
        return ""


def is_reactive_tracking_noise(repo_path: str, metric_name: str, details: str) -> bool:
    """
    判断给定指标是否属于反应式跟踪噪声。
    
    参数:
        repo_path (str): 仓库内文件的相对路径。
        metric_name (str): 指标名称。
        details (str): 指标详情文本。
    
    返回:
        bool: 如果该指标命中反应式跟踪噪声条件，则为 `true`，否则为 `false`。
    """
    if metric_name != "error_handling":
        return False
    if "错误被忽略" not in details and "ignored" not in details.lower():
        return False
    source = load_file_text(repo_path)
    if not source:
        return False
    return "void version.value" in source or "void entry.changeTick" in source


def noise_reason(repo_path: str, rule_name: str, metric_name: str, rule_config: dict[str, Any], details: str) -> str | None:
    """
    判断规则是否应按噪声原因抑制。
    
    Returns:
        str | None: 命中抑制时返回对应的噪声原因，否则返回 `None`。
    """
    patterns = optional_string_list(rule_config, "noiseExcludes", context=f"gateRules.{rule_name}")
    if not matches_any(repo_path, patterns):
        return None
    if is_reactive_tracking_noise(repo_path, metric_name, details):
        return "reactive-tracking read pattern"
    if rule_name == "duplication":
        return "declarative duplication surface"
    return "bounded policy noise"


def target_config(scope: str, gate_config: dict[str, Any]) -> dict[str, Any]:
    """
    获取指定 scope 的目标配置。
    
    Parameters:
    	scope (str): 目标范围名称。
    	gate_config (dict[str, Any]): 门禁配置对象。
    
    Returns:
    	dict[str, Any]: 对应 scope 的目标配置字典。
    """
    targets = require_dict(gate_config, "targets", context="gate_config")
    return require_dict(targets, scope, context="gate_config.targets")


def scope_root(scope: str, gate_config: dict[str, Any]) -> str:
    """
    获取指定 scope 的根目录。
    
    Parameters:
    	scope (str): scope 名称。
    	gate_config (dict[str, Any]): 门禁配置对象。
    
    Returns:
    	str: 该 scope 的根目录路径。
    """
    return require_string(target_config(scope, gate_config), "root", context=f"gate_config.targets.{scope}")


def scope_patterns(scope: str, gate_config: dict[str, Any]) -> tuple[list[str], list[str]]:
    """
    获取指定作用域的包含与排除模式。
    
    Parameters:
    	scope (str): 作用域名称。
    	gate_config (dict[str, Any]): 门禁配置。
    
    Returns:
    	tuple[list[str], list[str]]: 依次返回包含模式和排除模式。
    """
    target = target_config(scope, gate_config)
    include = require_string_list(target, "include", context=f"gate_config.targets.{scope}")
    exclude = require_string_list(target, "exclude", context=f"gate_config.targets.{scope}")
    return include, exclude


def path_matches_scope(path: str, scope: str, gate_config: dict[str, Any]) -> bool:
    """
    判断路径是否属于指定的作用域。
    
    Parameters:
    	path (str): 待检查的仓库相对路径。
    	scope (str): 作用域名称。
    	gate_config (dict[str, Any]): 门禁配置。
    
    Returns:
    	bool: `true` 表示路径未命中排除规则且命中包含规则，`false`  otherwise.
    """
    include, exclude = scope_patterns(scope, gate_config)
    if matches_any(path, exclude):
        return False
    return matches_any(path, include)


def classify_scope(path: str, gate_config: dict[str, Any]) -> str | None:
    """
    按网格配置中的目标规则为路径确定所属的 scope。
    
    Returns:
    	scope (str | None): 首个匹配该路径的 scope；如果没有任何 scope 匹配，则返回 None。
    """
    targets = require_dict(gate_config, "targets", context="gate_config")
    for scope, target in targets.items():
        if not isinstance(target, dict):
            raise GateConfigError(f"gate_config.targets.{scope} must be an object")
        if path_matches_scope(path, scope, gate_config):
            return scope
    return None


def relative_path_for_scope(path: str, scope: str, gate_config: dict[str, Any]) -> str:
    """
    将仓库路径转换为相对于指定作用域根目录的路径。
    
    Parameters:
    	path (str): 仓库相对路径。
    	scope (str): 作用域名称。
    	gate_config (dict[str, Any]): 门禁配置。
    
    Returns:
    	str: 相对于作用域根目录的路径；如果输入不在该根目录下，则返回原始路径。
    """
    root_path = pathlib.PurePosixPath(scope_root(scope, gate_config))
    full_path = pathlib.PurePosixPath(path)
    try:
        return full_path.relative_to(root_path).as_posix()
    except ValueError:
        return path


def repository_path_for_scope(relative_path: str, scope: str, gate_config: dict[str, Any]) -> str:
    """
    将范围内的相对路径转换为仓库相对路径。
    
    Parameters:
    	relative_path (str): 范围内的相对路径。
    	scope (str): 目标范围名称。
    	gate_config (dict[str, Any]): 门禁配置。
    
    Returns:
    	str: 组合后的仓库相对 POSIX 路径。
    """
    return (pathlib.PurePosixPath(scope_root(scope, gate_config)) / pathlib.PurePosixPath(relative_path)).as_posix()


def list_scope_files_on_disk(scope: str, gate_config: dict[str, Any]) -> list[str]:
    """
    列出磁盘上属于指定作用域的文件。
    
    返回：
        list[str]: 仓库相对路径列表，按字典序排序；如果作用域根目录不存在，则返回空列表。
    """
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
    """
    将仓库相对匹配模式转换为 scope 相对模式。
    
    Parameters:
    	pattern (str): 仓库相对的匹配模式。
    	scope (str): 目标作用域名称。
    	gate_config (dict[str, Any]): 门禁配置。
    
    Returns:
    	str: 去除作用域根路径前缀后的模式；若未匹配前缀则返回原模式。
    """
    root = scope_root(scope, gate_config)
    prefix = f"{root}/"
    if pattern.startswith(prefix):
        return pattern[len(prefix):]
    return pattern


def scope_relative_excludes(scope: str, gate_config: dict[str, Any]) -> list[str]:
    """
    将作用域级排除模式转换为相对该作用域根目录的模式列表。
    
    Parameters:
    	scope (str): 作用域名称。
    	gate_config (dict[str, Any]): 门禁配置。
    
    Returns:
    	list[str]: 相对作用域根目录的排除模式列表。
    """
    _, exclude = scope_patterns(scope, gate_config)
    return [to_scope_relative_pattern(pattern, scope, gate_config) for pattern in exclude]


def positive_int(value: Any) -> int:
    """
    返回一个大于 0 的整数值。
    
    Parameters:
    	value (Any): 待转换的值。
    
    Returns:
    	int: 当值本身是大于 0 的整数时返回该值；否则返回 0。
    """
    if isinstance(value, bool):
        return 0
    if isinstance(value, int) and value > 0:
        return value
    return 0


def build_eff_u_code_overrides(
    base_eff_config: dict[str, Any],
    gate_config: dict[str, Any],
    scoped_candidates: dict[str, list[str]],
    scopes: list[str],
) -> dict[str, Any]:
    """
    构建适用于 eff-u-code 的覆盖配置。
    
    参数:
    	base_eff_config (dict[str, Any]): 基础 eff-u-code 配置。
    	gate_config (dict[str, Any]): 门禁配置。
    	scoped_candidates (dict[str, list[str]]): 各 scope 的候选文件列表。
    	scopes (list[str]): 需要生成覆盖项的 scope 列表。
    
    返回:
    	dict[str, Any]: 包含更新后的 defaults 和 targets 的覆盖配置。
    """
    defaults = dict(require_dict(base_eff_config, "defaults", context="eff_u_code_config"))
    base_targets = require_dict(base_eff_config, "targets", context="eff_u_code_config")
    overrides = {"defaults": defaults, "targets": {}}
    defaults_top = positive_int(defaults.get("top"))

    for scope in scopes:
        base_target = base_targets.get(scope)
        if not isinstance(base_target, dict):
            raise GateConfigError(f"eff_u_code_config.targets.{scope} must be an object")
        candidate_top = max(1, len(scoped_candidates.get(scope, [])))
        merged_target = dict(base_target)
        merged_target["path"] = scope_root(scope, gate_config)
        merged_target["exclude"] = scope_relative_excludes(scope, gate_config)
        merged_target["top"] = max(defaults_top, positive_int(base_target.get("top")), candidate_top)
        overrides["targets"][scope] = merged_target
    return overrides


def run_eff_u_code(scope: str, *, output_dir: pathlib.Path, eff_config_override: pathlib.Path | None, base_ref: str | None = None) -> pathlib.Path:
    """
    运行 eff-u-code 并返回生成的报告路径。
    
    Parameters:
    	scope (str): 要评估的 scope 名称。
    	output_dir (pathlib.Path): 报告输出目录。
    	eff_config_override (pathlib.Path | None): 传递给 eff-u-code 的配置覆盖文件路径；为 `None` 时不传该参数。
    	base_ref (str | None): 用于设置 `GRAFT_EFF_U_CODE_BASE_REF` 的基准引用。
    
    Returns:
    	pathlib.Path: 对应 scope 的 JSON 报告路径。
    
    Raises:
    	RuntimeError: 当 eff-u-code 进程以非零退出码结束时。
    """
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
    """
    将指定 Git 修订版本导出为快照目录。
    
    Parameters:
    	revision (str): 要导出的 Git 修订版本。
    	destination (pathlib.Path): 快照输出目录。
    
    Returns:
    	pathlib.Path: 解压后的快照目录路径。
    """
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


def build_snapshot_eff_config(
    base_eff_config: dict[str, Any],
    gate_config: dict[str, Any],
    snapshot_root: pathlib.Path,
    scoped_candidates: dict[str, list[str]],
    scopes: list[str],
) -> dict[str, Any]:
    """
    生成用于基线快照的 eff-u-code 配置覆盖。
    
    Parameters:
    	base_eff_config (dict[str, Any]): eff-u-code 基础配置。
    	gate_config (dict[str, Any]): 门禁配置。
    	snapshot_root (pathlib.Path): 快照导出目录。
    	scoped_candidates (dict[str, list[str]]): 各 scope 的候选文件列表。
    	scopes (list[str]): 需要生成配置的 scope 列表。
    
    Returns:
    	dict[str, Any]: 可用于基线扫描的 eff-u-code 配置覆盖。
    """
    config = build_eff_u_code_overrides(base_eff_config, gate_config, scoped_candidates, scopes)
    for scope in scopes:
        root = require_string(target_config(scope, gate_config), "root", context=f"gate_config.targets.{scope}")
        config["targets"][scope]["path"] = str((snapshot_root / root).resolve())
    return config


def load_report(path: pathlib.Path) -> dict[str, Any]:
    """
    加载并解析 eff-u-code 报告。
    
    Returns:
    	report (dict[str, Any]): 解析后的 JSON 对象。
    
    Raises:
    	RuntimeError: 当报告文件不存在时抛出。
    """
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise RuntimeError(f"missing eff-u-code report: {path}") from exc


def build_file_index(report: dict[str, Any]) -> dict[str, dict[str, Any]]:
    """
    按文件路径构建报告索引。
    
    Parameters:
    	report (dict[str, Any]): eff-u-code 报告对象。
    
    Returns:
    	dict[str, dict[str, Any]]: 以文件路径为键的文件报告索引。
    
    Raises:
    	GateConfigError: 当报告中缺少 files 数组时抛出。
    """
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
    """
    按指标名称索引文件报告中的指标。
    
    Parameters:
    	file_report (dict[str, Any]): 单个文件的报告对象。
    
    Returns:
    	dict[str, dict[str, Any]]: 以非空指标名称为键、对应指标对象为值的字典；当指标列表缺失或格式不正确时返回空字典。
    """
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
    """
    评估单个文件的单条规则与指标结果。
    
    Parameters:
    	repo_path (str): 仓库内文件路径。
    	rule_name (str): 规则名称。
    	rule_config (dict[str, Any]): 规则配置。
    	metric_name (str): 指标名称。
    	current_metric (dict[str, Any]): 当前评估文件的指标数据。
    	baseline_metric (dict[str, Any] | None): 基线文件对应的指标数据。
    	is_new_file (bool): 是否为新增文件。
    	scan_mode (str): 扫描模式。
    
    Returns:
    	RuleEvaluation: 该规则与指标的评估结果。
    """
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
    """
    计算文件的展示用 Curated Score。
    
    从每个规则在当前文件中的最佳得分按权重求加权平均；当没有可用权重或可用分数时返回空值。
    
    Returns:
        float | None: Curated Score；当没有可参与计算的规则权重或分数时为 ``None``。
    """
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
    """
    解析命令行参数。
    
    返回：
    	argparse.Namespace: 包含配置路径、扫描模式、基准分支、输出路径、报告目录和作用域限制等参数。
    """
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
    """
    执行 Graft Quality Policy 门禁评估并输出结果。
    
    Returns:
        int: 退出码；无失败时为 0，存在评估失败时为 1，配置或运行错误时为 2。
    """
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
        merged_eff_config = build_eff_u_code_overrides(base_eff_config, gate_config, scoped_candidates, selected_scopes)

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
                snapshot_eff_config = build_snapshot_eff_config(
                    base_eff_config,
                    gate_config,
                    snapshot_root,
                    scoped_candidates,
                    selected_scopes,
                )
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
        overall_rule_failures = 0
        overall_unreported_failures = 0

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
                overall_rule_failures += len(failures)
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
            overall_unreported_failures += len(unreported_files)
            scope_results[scope] = {
                "changedFiles": scoped_candidates[scope] if args.scan_mode == "changed" else [],
                "candidateFiles": scoped_candidates[scope],
                "unreportedFiles": unreported_files,
                "coverageStatus": "fail" if unreported_files else "pass",
                "files": file_results,
            }

        overall_failures = overall_rule_failures + overall_unreported_failures
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
                "ruleFailures": overall_rule_failures,
                "coverageFailures": overall_unreported_failures,
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
                if not scope_file_results and not scope_results[scope]["unreportedFiles"]:
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
            if scope_results[scope]["unreportedFiles"]:
                print(f"  unreported by eff-u-code: {len(scope_results[scope]['unreportedFiles'])}")
                for path in scope_results[scope]["unreportedFiles"]:
                    print(f"  UNREPORTED {path}")

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
