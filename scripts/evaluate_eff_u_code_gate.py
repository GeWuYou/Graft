#!/usr/bin/env python3
"""Evaluate Graft Quality Policy from eff-u-code JSON reports."""

from __future__ import annotations

import argparse
import collections
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


@dataclass(frozen=True)
class ChangedBaselineResolution:
    revision: str | None
    compare_revision: str
    base_ref: str
    normalized_base_ref: str
    fetch_target: str
    source: str


SEVERITY_ORDER = ("critical", "high", "medium", "low")
DEFAULT_DISPLAY_ORDER = ("complexity", "duplication", "file_size", "structure", "error_handling")


def severity_rank(level: str) -> int:
    """
    获取严重程度的排序权重。
    
    Parameters:
        level (str): 严重程度名称。
    
    Returns:
        int: 严重程度在排序中的索引；未知等级返回最低优先级的索引。
    """
    try:
        return SEVERITY_ORDER.index(level)
    except ValueError:
        return len(SEVERITY_ORDER)


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
    读取可选数值字段，并在提供时转换为浮点数。
    
    参数：
    	container (dict[str, Any]): 待读取的容器。
    	key (str): 字段名。
    	context (str): 用于错误信息的上下文名称。
    
    返回：
    	float | None: 字段存在且为数值时返回其浮点值；字段缺失或为 None 时返回 None。
    
    异常：
    	GateConfigError: 当字段存在但不是数值时抛出。
    """
    value = container.get(key)
    if value is None:
        return None
    if not isinstance(value, (int, float)):
        raise GateConfigError(f"{context}.{key} must be a number when provided")
    return float(value)


def optional_bool(container: dict[str, Any], key: str, *, context: str) -> bool | None:
    """
    获取可选布尔配置项。
    
    Parameters:
        container (dict[str, Any]): 配置容器。
        key (str): 字段名。
        context (str): 错误信息上下文。
    
    Returns:
        bool | None: 提供时返回对应布尔值；字段缺失或为 None 时返回 None。
    """
    value = container.get(key)
    if value is None:
        return None
    if not isinstance(value, bool):
        raise GateConfigError(f"{context}.{key} must be a boolean when provided")
    return value


def run_git(args: list[str]) -> str:
    """
    在仓库根目录执行 Git 命令并返回输出。
    
    Parameters:
    	args (list[str]): 传递给 `git` 的参数。
    
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


def resolve_changed_mode_files(
    diff_filter: str,
    base_ref_override: str | None = None,
) -> tuple[list[str], ChangedBaselineResolution]:
    """
    解析 changed 模式下的候选文件，并返回与之对应的基线解析结果。

    Parameters:
        diff_filter (str): Git diff 的文件状态过滤条件。
        base_ref_override (str | None): 覆盖用于解析基准分支的名称。

    Returns:
        tuple[list[str], ChangedBaselineResolution]: 变更文件列表与实际采用的基线结果。
    """
    base_ref = (
        base_ref_override
        or os.environ.get("GRAFT_LINT_BASE_REF", "")
        or os.environ.get("GITHUB_BASE_REF", "")
    ).strip()
    normalized_base_ref = normalize_remote_base_ref(base_ref) if base_ref else ""
    fetch_target = fetch_target_from_base_ref(base_ref) if base_ref else ""

    local_changed = staged_or_changed_files(diff_filter)
    if local_changed:
        return local_changed, ChangedBaselineResolution(
            revision=None,
            compare_revision="HEAD",
            base_ref=base_ref,
            normalized_base_ref=normalized_base_ref,
            fetch_target=fetch_target,
            source="local-changes",
        )

    explicit_base_sha = os.environ.get("GRAFT_QUALITY_BASE_SHA", "").strip()
    if explicit_base_sha:
        candidate = ChangedBaselineResolution(
            revision=explicit_base_sha,
            compare_revision="HEAD",
            base_ref=base_ref,
            normalized_base_ref=normalized_base_ref,
            fetch_target=fetch_target,
            source="explicit-sha",
        )
        changed = run_git(["diff", "--name-only", f"--diff-filter={diff_filter}", f"{candidate.revision}...{candidate.compare_revision}"])
        changed_files = [line for line in changed.splitlines() if line]
        if changed_files:
            return changed_files, candidate

    if fetch_target:
        subprocess.run(
            ["git", "fetch", "--no-tags", "--prune", "origin", fetch_target],
            cwd=REPO_ROOT,
            check=False,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )

    if normalized_base_ref:
        try:
            merge_base = run_git(["merge-base", "HEAD", normalized_base_ref])
        except RuntimeError:
            merge_base = ""
        if merge_base:
            candidate = ChangedBaselineResolution(
                revision=merge_base,
                compare_revision="HEAD",
                base_ref=base_ref,
                normalized_base_ref=normalized_base_ref,
                fetch_target=fetch_target,
                source="merge-base",
            )
            changed = run_git(["diff", "--name-only", f"--diff-filter={diff_filter}", f"{candidate.revision}...{candidate.compare_revision}"])
            changed_files = [line for line in changed.splitlines() if line]
            if changed_files:
                return changed_files, candidate

    head_sha = os.environ.get("GITHUB_SHA", "").strip()
    if head_sha:
        try:
            previous = run_git(["rev-parse", f"{head_sha}^"])
        except RuntimeError:
            previous = ""
        if previous:
            candidate = ChangedBaselineResolution(
                revision=previous,
                compare_revision=head_sha,
                base_ref=base_ref,
                normalized_base_ref=normalized_base_ref,
                fetch_target=fetch_target,
                source="previous-head",
            )
            changed = run_git(["diff", "--name-only", f"--diff-filter={diff_filter}", f"{candidate.revision}...{candidate.compare_revision}"])
            changed_files = [line for line in changed.splitlines() if line]
            if changed_files:
                return changed_files, candidate

    tracked = run_git(["ls-files"])
    return [line for line in tracked.splitlines() if line], ChangedBaselineResolution(
        revision=None,
        compare_revision="HEAD",
        base_ref=base_ref,
        normalized_base_ref=normalized_base_ref,
        fetch_target=fetch_target,
        source="tracked",
    )


def ci_changed_files(diff_filter: str, base_ref_override: str | None = None) -> list[str]:
    """
    获取用于 CI 的候选变更文件列表。
    
    Parameters:
    	diff_filter (str): Git diff 的文件状态过滤条件。
    	base_ref_override (str | None): 覆盖用于解析基准分支的名称。
    
    Returns:
    	list[str]: 按优先级解析得到的变更文件路径列表；若未找到变更，则返回已跟踪文件列表。
    """
    files, _ = resolve_changed_mode_files(diff_filter, base_ref_override)
    return files


def normalize_remote_base_ref(base_ref: str) -> str:
    """
    统一将各种 base ref 形式规范化为可用于 merge-base 的远端引用。

    Parameters:
        base_ref (str): 用户参数、环境变量或 workflow 提供的基准引用。

    Returns:
        str: 规范化后的远端 ref。
    """
    ref = base_ref.strip()
    if not ref:
        return ref
    if ref.startswith("refs/remotes/"):
        return ref
    if ref.startswith("refs/heads/"):
        return f"refs/remotes/origin/{ref.removeprefix('refs/heads/')}"
    if ref.startswith("origin/"):
        return f"refs/remotes/{ref}"
    if ref.startswith("refs/"):
        return ref
    return f"refs/remotes/origin/{ref}"


def fetch_target_from_base_ref(base_ref: str) -> str:
    """
    从规范化或未规范化的 base ref 推导 `git fetch origin` 目标。

    Parameters:
        base_ref (str): 基准引用。

    Returns:
        str: 适合传给 `git fetch origin <target>` 的目标字符串。
    """
    normalized = normalize_remote_base_ref(base_ref)
    if normalized.startswith("refs/remotes/origin/"):
        return normalized.removeprefix("refs/remotes/origin/")
    if normalized.startswith("refs/remotes/"):
        return normalized.removeprefix("refs/remotes/")
    if normalized.startswith("refs/heads/"):
        return normalized.removeprefix("refs/heads/")
    return normalized


def matches_any(path: str, patterns: list[str]) -> bool:
    """
    判断路径是否匹配任一模式。
    
    Returns:
    	如果路径匹配给定模式列表中的任一模式，则为 `True`，否则为 `False`。
    """
    for pattern in patterns:
        variants = {pattern}
        pending = [pattern]
        while pending:
            current = pending.pop()
            marker = "/**/"
            if marker not in current:
                continue
            collapsed = current.replace(marker, "/", 1)
            if collapsed not in variants:
                variants.add(collapsed)
                pending.append(collapsed)
        if any(fnmatch.fnmatch(path, variant) for variant in variants):
            return True
    return False


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
    计算文件的 Curated Score。
    
    按规则在当前文件中的最佳得分结合配置权重计算加权平均值；仅用于展示，不参与门禁判定。
    
    参数:
    	file_evaluations (list[RuleEvaluation]): 文件的规则评估结果。
    	gate_config (dict[str, Any]): 门禁配置，其中必须包含 `curatedScore.weights`。
    
    返回:
    	float | None: Curated Score；当没有可参与计算的正权重或对应分数时为 `None`。
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


def score_gate_config(gate_config: dict[str, Any]) -> dict[str, Any]:
    """
    获取评分门禁配置。
    
    Parameters:
        gate_config (dict[str, Any]): 门禁配置。
    
    Returns:
        dict[str, Any]: 评分门禁配置；缺失时返回空对象。
    """
    value = gate_config.get("scoreGate")
    if value is None:
        return {}
    if not isinstance(value, dict):
        raise GateConfigError("gate_config.scoreGate must be an object")
    return value


def score_profile_config(gate_config: dict[str, Any], profile: str) -> dict[str, Any]:
    """
    读取指定评分 profile 的配置对象。
    
    Parameters:
        gate_config (dict[str, Any]): 门禁配置。
        profile (str): profile 名称。
    
    Returns:
        dict[str, Any]: 指定 profile 的配置对象；未配置时返回空字典。
    """
    config = score_gate_config(gate_config)
    profiles = config.get("profiles")
    if profiles is None:
        return {}
    if not isinstance(profiles, dict):
        raise GateConfigError("gate_config.scoreGate.profiles must be an object")
    value = profiles.get(profile)
    if value is None:
        return {}
    if not isinstance(value, dict):
        raise GateConfigError(f"gate_config.scoreGate.profiles.{profile} must be an object")
    return value


def score_gate_enabled(gate_config: dict[str, Any], profile: str, scan_mode: str) -> bool:
    """
    判断评分门禁在当前 profile 和扫描模式下是否启用。
    
    Parameters:
        gate_config (dict[str, Any]): 门禁配置。
        profile (str): 评分门禁 profile。
        scan_mode (str): 当前扫描模式。
    
    Returns:
        bool: `true` 如果该 profile 已启用且允许当前扫描模式，`false` 其他情况。
    """
    if profile == "legacy":
        return False
    profile_config = score_profile_config(gate_config, profile)
    if not profile_config:
        return False
    enabled = optional_bool(profile_config, "enabled", context=f"gate_config.scoreGate.profiles.{profile}")
    if enabled is False:
        return False
    enabled_modes = optional_string_list(profile_config, "enabledScanModes", context=f"gate_config.scoreGate.profiles.{profile}")
    if enabled_modes and scan_mode not in enabled_modes:
        return False
    return True


def score_gate_threshold(gate_config: dict[str, Any], profile: str) -> float:
    """
    获取指定评分 profile 的门禁阈值。
    
    Parameters:
        gate_config (dict[str, Any]): 门禁配置。
        profile (str): 评分 profile 名称。
    
    Returns:
        float: 该 profile 的阈值。
    """
    profile_config = score_profile_config(gate_config, profile)
    threshold = optional_number(profile_config, "threshold", context=f"gate_config.scoreGate.profiles.{profile}")
    if threshold is None:
        raise GateConfigError(f"gate_config.scoreGate.profiles.{profile}.threshold must be a number")
    return threshold


def score_gate_top_contributors(gate_config: dict[str, Any], profile: str) -> int:
    """
    获取评分门禁配置中的 Top Contributors 数量。
    
    Parameters:
        gate_config (dict[str, Any]): 门禁配置。
        profile (str): 评分配置档位名称。
    
    Returns:
        int: Top Contributors 的显示数量；未配置或无效时为 10。
    """
    profile_config = score_profile_config(gate_config, profile)
    top = positive_int(profile_config.get("topContributors"))
    return top or 10


def score_gate_detail_limit(gate_config: dict[str, Any], profile: str) -> int:
    """
    获取详情展开的文件数量上限。

    Parameters:
        gate_config (dict[str, Any]): 门禁配置。
        profile (str): 评分 profile。

    Returns:
        int: 正整数，默认 10。
    """
    profile_config = score_profile_config(gate_config, profile)
    top = positive_int(profile_config.get("detailLimit"))
    return top or 10


def score_gate_gain_steps(gate_config: dict[str, Any], profile: str) -> list[int]:
    """
    获取评分门禁的预计收益展示步长列表。
    
    Parameters:
        gate_config (dict[str, Any]): 门禁配置。
        profile (str): 评分 profile。
    
    Returns:
        list[int]: 升序去重后的正整数列表；未配置时返回 [3, 5, 10]。
    """
    profile_config = score_profile_config(gate_config, profile)
    raw = profile_config.get("potentialGainSteps")
    if raw is None:
        return [3, 5, 10]
    if not isinstance(raw, list):
        raise GateConfigError(f"gate_config.scoreGate.profiles.{profile}.potentialGainSteps must be an array when provided")
    steps: set[int] = set()
    for item in raw:
        if not isinstance(item, int) or item <= 0:
            raise GateConfigError(f"gate_config.scoreGate.profiles.{profile}.potentialGainSteps must contain positive integers")
        steps.add(item)
    return sorted(steps) or [3, 5, 10]


def score_gate_rule_order(gate_config: dict[str, Any], profile: str) -> tuple[str, ...]:
    """
    确定输出中规则分类的顺序。
    
    Parameters:
        gate_config (dict[str, Any]): 门禁配置。
        profile (str): 评分配置档位。
    
    Returns:
        tuple[str, ...]: 规则分类名称的输出顺序；未配置时返回默认顺序。
    """
    profile_config = score_profile_config(gate_config, profile)
    categories = optional_string_list(profile_config, "categoryOrder", context=f"gate_config.scoreGate.profiles.{profile}")
    if not categories:
        return DEFAULT_DISPLAY_ORDER
    seen: list[str] = []
    for item in categories:
        if item not in seen:
            seen.append(item)
    for item in DEFAULT_DISPLAY_ORDER:
        if item not in seen:
            seen.append(item)
    return tuple(seen)


def estimate_rule_impact(rule_name: str, current_score: float, gate_config: dict[str, Any]) -> float:
    """
    估算单条规则对项目评分的影响值。
    
    Parameters:
        rule_name (str): 规则名称。
        current_score (float): 当前得分。
        gate_config (dict[str, Any]): 门禁配置。
    
    Returns:
        float: 该规则对应的非负影响值。
    """
    curated = require_dict(gate_config, "curatedScore", context="gate_config")
    weights = require_dict(curated, "weights", context="gate_config.curatedScore")
    weight_value = weights.get(rule_name, 0.0)
    if not isinstance(weight_value, (int, float)):
        raise GateConfigError(f"gate_config.curatedScore.weights.{rule_name} must be a number")
    weight = float(weight_value)
    if weight <= 0:
        return 0.0
    normalized_gap = max(0.0, 100.0 - current_score)
    return (normalized_gap / 100.0) * weight * 100.0


def issue_severity(rule_name: str, current_score: float, threshold: float | None) -> str:
    """
    根据规则类别和偏离程度估算问题严重程度。
    
    Parameters:
        rule_name (str): 规则名称。
        current_score (float): 当前得分。
        threshold (float | None): 规则阈值；为 `None` 时按满分基准计算偏离程度。
    
    Returns:
        str: `critical`、`high`、`medium` 或 `low`。
    """
    if rule_name == "error_handling" and current_score < 40:
        return "critical"
    if threshold is None:
        gap = max(0.0, 100.0 - current_score)
    else:
        gap = max(0.0, threshold - current_score)
    if rule_name in {"complexity", "error_handling"} and gap >= 15:
        return "critical"
    if gap >= 10:
        return "high"
    if gap >= 5:
        return "medium"
    return "low"


def format_score(value: float | None) -> str:
    """
    将得分格式化为终端显示文本。
    
    Parameters:
        value (float | None): 需要格式化的得分。
    
    Returns:
        str: `value` 为 `None` 时返回 `n/a`，否则返回保留一位小数的字符串。
    """
    if value is None:
        return "n/a"
    return f"{value:.1f}"


def truncate_detail(text: str, limit: int = 120) -> str:
    """
    压缩文本中连续的空白字符，并在超长时截断。
    
    Parameters:
        text (str): 原始文本。
        limit (int): 最大长度；默认 120。
    
    Returns:
        str: 去除多余空白后的文本，若超过长度限制则被截断并在末尾加省略号。
    """
    compact = " ".join(text.split())
    if len(compact) <= limit:
        return compact
    return compact[: limit - 1].rstrip() + "…"


def extract_subject(details: str) -> str | None:
    """
    提取指标详情中的首个有效主体文本。
    
    Parameters:
        details (str): 指标详情文本。
    
    Returns:
        str | None: 提取并截断后的主体文本；如果无法提取则返回 `None`。
    """
    for line in details.splitlines():
        stripped = line.strip(" -•\t")
        if not stripped:
            continue
        return truncate_detail(stripped, limit=80)
    compact = truncate_detail(details, limit=80)
    return compact or None


def build_file_diagnostics(
    repo_path: str,
    evaluations: list[RuleEvaluation],
    gate_config: dict[str, Any],
    category_order: tuple[str, ...],
) -> dict[str, Any]:
    """
    为单个文件生成评分诊断摘要。
    
    Parameters:
        repo_path (str): 仓库相对路径。
        evaluations (list[RuleEvaluation]): 该文件的规则评估结果。
        gate_config (dict[str, Any]): 门禁配置，用于计算分类影响度。
        category_order (tuple[str, ...]): 分类汇总的输出顺序。
    
    Returns:
        dict[str, Any]: 包含文件路径、总影响度、问题数、最高严重度、分类影响度和严重度汇总的诊断摘要。
    """
    file_impact = 0.0
    issue_count = 0
    highest_severity = "low"
    category_impacts: dict[str, float] = collections.defaultdict(float)
    severity_groups: dict[str, list[dict[str, Any]]] = {level: [] for level in SEVERITY_ORDER}

    for evaluation in evaluations:
        impact = estimate_rule_impact(evaluation.rule, evaluation.current_score, gate_config)
        severity = issue_severity(evaluation.rule, evaluation.current_score, evaluation.threshold)
        status = evaluation.status
        contributes = status not in {"display-only", "advisory", "suppressed-noise"}
        if contributes and impact > 0:
            category_impacts[evaluation.rule] += impact
            file_impact += impact
            issue_count += 1
            if severity_rank(severity) < severity_rank(highest_severity):
                highest_severity = severity
            severity_groups[severity].append(
                {
                    "rule": evaluation.rule,
                    "metric": evaluation.metric,
                    "subject": extract_subject(evaluation.details),
                    "impact": impact,
                    "currentScore": evaluation.current_score,
                    "threshold": evaluation.threshold,
                    "details": truncate_detail(evaluation.details),
                    "status": status,
                }
            )

    ordered_categories = [
        {"rule": rule_name, "impact": category_impacts.get(rule_name, 0.0)}
        for rule_name in category_order
        if category_impacts.get(rule_name, 0.0) > 0
    ]
    severity_summary = {
        level: {
            "count": len(severity_groups[level]),
            "issues": sorted(severity_groups[level], key=lambda item: (-item["impact"], item["rule"], item["metric"])),
        }
        for level in SEVERITY_ORDER
    }
    return {
        "path": repo_path,
        "impact": file_impact,
        "issueCount": issue_count,
        "highestSeverity": highest_severity if issue_count else "low",
        "categoryImpacts": ordered_categories,
        "severitySummary": severity_summary,
    }


def average(values: list[float]) -> float | None:
    """
    计算数值列表的平均值。
    
    Parameters:
        values (list[float]): 要计算平均值的数值列表。
    
    Returns:
        float | None: 平均值；列表为空时返回 `None`。
    """
    if not values:
        return None
    return sum(values) / len(values)


def print_section_divider(char: str, width: int = 46) -> None:
    """
    打印由指定字符重复组成的分割线。
    
    Parameters:
        char (str): 用于生成分割线的字符。
        width (int): 分割线长度。
    """
    print(char * width)


def display_scope_name(scope: str) -> str:
    """
    将内部 scope 名称转换为展示名称。

    Parameters:
        scope (str): 内部 scope 名称。

    Returns:
        str: 终端展示名称。
    """
    return "Server" if scope == "server" else "Web"


def display_rule_name(rule_name: str) -> str:
    """
    将规则名称转换为友好的展示名称。

    Parameters:
        rule_name (str): 规则名称。

    Returns:
        str: 展示名称。
    """
    mapping = {
        "complexity": "Complexity",
        "duplication": "Duplication",
        "file_size": "File Size",
        "structure": "Structure",
        "error_handling": "Error Handling",
        "documentation": "Documentation",
        "naming": "Naming",
        "parameter_count": "Parameter Count",
    }
    return mapping.get(rule_name, rule_name.replace("_", " ").title())


def display_severity(level: str) -> str:
    """
    将严重程度转换为展示名称。
    
    Parameters:
        level (str): 严重程度标识。
    
    Returns:
        str: 大写后的展示名称。
    """
    return level.upper()


def strip_scope_root(path: str, root: str) -> str:
    """
    将路径中的 scope 根目录前缀去掉。
    
    Parameters:
        path (str): 仓库相对路径。
        root (str): scope 根目录。
    
    Returns:
        str: 去除前缀后的路径。
    """
    prefix = f"{root}/"
    if path.startswith(prefix):
        return path[len(prefix):]
    return path


def summarize_category_impacts(file_results: list[dict[str, Any]], category_order: tuple[str, ...]) -> list[dict[str, Any]]:
    """
    汇总文件结果中的类别影响总和。
    
    Parameters:
        file_results (list[dict[str, Any]]): 文件结果列表。
        category_order (tuple[str, ...]): 类别输出顺序。
    
    Returns:
        list[dict[str, Any]]: 按指定顺序返回累计影响大于 0 的类别项，每项包含 `rule` 和 `impact`。
    """
    totals: dict[str, float] = collections.defaultdict(float)
    for file_result in file_results:
        diagnostics = file_result.get("scoreDiagnostics") or {}
        for entry in diagnostics.get("categoryImpacts", []):
            rule = entry.get("rule")
            impact = entry.get("impact")
            if isinstance(rule, str) and isinstance(impact, (int, float)):
                totals[rule] += float(impact)
    return [
        {"rule": rule, "impact": totals[rule]}
        for rule in category_order
        if totals.get(rule, 0.0) > 0
    ]


def summarize_severity(file_results: list[dict[str, Any]]) -> dict[str, Any]:
    """
    汇总文件结果中的严重程度分布。
    
    Parameters:
        file_results (list[dict[str, Any]]): 文件评估结果列表。
    
    Returns:
        dict[str, Any]: 包含各严重程度计数与对应高亮文件摘要的统计结果。
    """
    counts = {level: 0 for level in SEVERITY_ORDER}
    highlights: dict[str, list[dict[str, Any]]] = {level: [] for level in SEVERITY_ORDER}
    for file_result in file_results:
        diagnostics = file_result.get("scoreDiagnostics") or {}
        summary = diagnostics.get("severitySummary") or {}
        for level in SEVERITY_ORDER:
            level_data = summary.get(level) or {}
            count = level_data.get("count", 0)
            if isinstance(count, int):
                counts[level] += count
            issues = level_data.get("issues") or []
            if issues:
                highlights[level].append(
                    {
                        "path": file_result["path"],
                        "impact": diagnostics.get("impact", 0.0),
                        "issues": issues[:2],
                    }
                )
    for level in SEVERITY_ORDER:
        highlights[level].sort(key=lambda item: (-float(item["impact"]), item["path"]))
    return {"counts": counts, "highlights": highlights}


def top_contributors(file_results: list[dict[str, Any]], limit: int) -> list[dict[str, Any]]:
    """
    按影响度从高到低选出文件贡献者。
    
    Parameters:
        file_results (list[dict[str, Any]]): 文件结果列表。
        limit (int): 返回的最大条目数。
    
    Returns:
        list[dict[str, Any]]: 按影响度降序排列的文件结果列表，仅包含影响度大于 0 的条目。
    """
    ranked = sorted(
        file_results,
        key=lambda item: (-float((item.get("scoreDiagnostics") or {}).get("impact", 0.0)), item["path"]),
    )
    return [item for item in ranked if float((item.get("scoreDiagnostics") or {}).get("impact", 0.0)) > 0][:limit]


def potential_score_gain(scope_score: float | None, contributors: list[dict[str, Any]], file_count: int, steps: list[int]) -> list[dict[str, Any]]:
    """
    估算修复 Top N 文件后的潜在得分提升。

    Parameters:
        scope_score (float | None): 当前 scope 分数。
        contributors (list[dict[str, Any]]): 排名前列文件。
        file_count (int): 文件总数。
        steps (list[int]): 展示的 TopN 档位。

    Returns:
        list[dict[str, Any]]: 每个档位的预计提升。
    """
    if scope_score is None or file_count <= 0:
        return []
    contributions = [float((item.get("scoreDiagnostics") or {}).get("impact", 0.0)) for item in contributors]
    results: list[dict[str, Any]] = []
    running = 0.0
    contribution_prefix: list[float] = []
    for impact in contributions:
        running += impact
        contribution_prefix.append(running)
    for step in steps:
        if step <= 0 or step > len(contribution_prefix):
            continue
        gain = contribution_prefix[step - 1] / file_count
        projected = min(100.0, scope_score + gain)
        results.append({"topN": step, "gain": gain, "projectedScore": projected})
    return results


def render_score_gate_summary(
    scope: str,
    scope_result: dict[str, Any],
    threshold: float,
    top_limit: int,
    detail_limit: int,
    gain_steps: list[int],
) -> None:
    """
    打印评分门禁的分层摘要。
    
    Parameters:
        scope (str): scope 名称。
        scope_result (dict[str, Any]): scope 的评估结果。
        threshold (float): 评分门禁阈值。
        top_limit (int): Top Contributors 的显示数量上限。
        detail_limit (int): 详细展开的文件数量上限。
        gain_steps (list[int]): 计算潜在得分收益的档位。
    """
    file_results = scope_result["files"]
    scope_score = scope_result.get("scopeQualityScore")
    contributors = top_contributors(file_results, top_limit)
    category_summary = scope_result.get("categorySummary", [])
    severity_summary = scope_result.get("severitySummary", {"counts": {}, "highlights": {}})
    gains = potential_score_gain(scope_score, contributors, len(file_results), gain_steps)

    print_section_divider("━")
    print("Project Score Gate")
    print()
    print(display_scope_name(scope))
    print_section_divider("─")
    status_text = "PASS" if scope_result.get("scoreGateStatus") == "pass" else "FAIL"
    status_icon = "✅" if status_text == "PASS" else "❌"
    print(f"Score      : {format_score(scope_score)} / 100    {status_icon} {status_text} (threshold:{threshold:.0f})")
    print(
        f"Coverage   : {scope_result.get('coverageCount', 0)} / {scope_result.get('candidateCount', 0)}"
        + ("    ❌ FAIL" if scope_result.get("coverageStatus") == "fail" else "")
    )
    print(f"Top Impact : {len(contributors)} files")
    print()

    print("Top Contributors")
    print_section_divider("─")
    if not contributors:
        print("No score-impacting files reported.")
    else:
        for index, file_result in enumerate(contributors, start=1):
            impact = float((file_result.get("scoreDiagnostics") or {}).get("impact", 0.0))
            display_path = strip_scope_root(file_result["path"], str(scope_result.get("root", "")))
            print(f"{index:>2}. {display_path:<45} -{impact:.2f}")
    print()

    print("Total Impact")
    print_section_divider("─")
    if category_summary:
        for item in category_summary:
            print(f"{display_rule_name(item['rule']):<15} -{item['impact']:.1f}")
    else:
        print("No score-impacting rule categories.")
    print()

    print("Severity")
    print_section_divider("─")
    counts = severity_summary.get("counts", {})
    for level in SEVERITY_ORDER:
        count = counts.get(level, 0)
        if count:
            print(f"{display_severity(level):<9} {count}")
    if not any(counts.get(level, 0) for level in SEVERITY_ORDER):
        print("No score-impacting issues.")
    print()

    if gains:
        print("Potential Score Gain")
        print_section_divider("─")
        for item in gains:
            print(f"Fix Top {item['topN']:<2} +{item['gain']:.1f} -> {item['projectedScore']:.1f}")
        print()

    if contributors:
        advice_target = min(5, len(contributors))
        advice_projected = next((item["projectedScore"] for item in gains if item["topN"] == advice_target), None)
        print("Advice")
        print_section_divider("─")
        print(f"Fix the top {advice_target} contributors first.")
        if advice_projected is not None:
            print(f"Estimated score after fixing them: {advice_projected:.1f}")
        print()

        for index, file_result in enumerate(contributors[:detail_limit], start=1):
            diagnostics = file_result.get("scoreDiagnostics") or {}
            print_section_divider("━")
            print(f"Top Contributor #{index}")
            print()
            print(file_result["path"])
            print(f"Impact: -{float(diagnostics.get('impact', 0.0)):.2f}")
            print(f"Severity: {display_severity(str(diagnostics.get('highestSeverity', 'low')))}")
            print()
            print("Problems")
            print()
            for level in SEVERITY_ORDER:
                level_data = (diagnostics.get("severitySummary") or {}).get(level) or {}
                issues = level_data.get("issues") or []
                if not issues:
                    continue
                print(display_severity(level))
                for issue in issues:
                    print(f"  • {display_rule_name(issue['rule'])}")
                    if issue.get("subject"):
                        print(f"      {issue['subject']}")
                    print(f"      Score Impact: -{float(issue['impact']):.2f}")
                print()


def parse_args() -> argparse.Namespace:
    """
    解析并返回命令行参数。
    
    Returns:
        argparse.Namespace: 包含配置文件路径、eff-u-code 配置路径、扫描模式、基准分支、输出路径、报告目录、作用域限制和门禁配置档位的参数。
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
    parser.add_argument(
        "--gate-profile",
        choices=("legacy", "score-changed", "score-project"),
        default="legacy",
        help="legacy: preserve existing rule-by-rule gate semantics; score-* use repository-owned score thresholds",
    )
    return parser.parse_args()


def main() -> int:
    """
    执行 Graft Quality Policy 门禁评估并输出结果。
    
    根据配置和扫描模式汇总 eff-u-code 报告，计算规则失败、覆盖缺失与可选的评分门禁结果，并在需要时输出 JSON 与终端诊断信息。
    
    Returns:
        int: 退出码；成功为 0，存在评估失败为 1，配置或运行错误为 2。
    """
    args = parse_args()
    temp_ctx: tempfile.TemporaryDirectory[str] | None = None

    try:
        gate_config = load_json(pathlib.Path(args.config))
        score_enabled = score_gate_enabled(gate_config, args.gate_profile, args.scan_mode)
        score_threshold = score_gate_threshold(gate_config, args.gate_profile) if score_enabled else None
        contributor_limit = score_gate_top_contributors(gate_config, args.gate_profile) if score_enabled else 0
        detail_limit = score_gate_detail_limit(gate_config, args.gate_profile) if score_enabled else 0
        gain_steps = score_gate_gain_steps(gate_config, args.gate_profile) if score_enabled else []
        category_order = score_gate_rule_order(gate_config, args.gate_profile) if score_enabled else DEFAULT_DISPLAY_ORDER
        requested_scopes = args.scopes or ["server", "web"]
        changed_files: list[str] = []
        changed_baseline = ChangedBaselineResolution(
            revision=None,
            compare_revision="HEAD",
            base_ref="",
            normalized_base_ref="",
            fetch_target="",
            source="unused",
        )
        scoped_candidates: dict[str, list[str]] = {}

        if args.scan_mode == "changed":
            diff_config = require_dict(gate_config, "changedFiles", context="gate_config")
            diff_filter = require_string(diff_config, "diffFilter", context="gate_config.changedFiles")
            changed_files, changed_baseline = resolve_changed_mode_files(diff_filter, args.base_ref)

            scoped_changed: dict[str, list[str]] = {}
            for path in changed_files:
                scope = classify_scope(path, gate_config)
                if scope is None:
                    continue
                scoped_changed.setdefault(scope, []).append(path)
            scoped_candidates = {
                scope: [path for path in scoped_changed.get(scope, []) if path_matches_scope(path, scope, gate_config)]
                for scope in requested_scopes
            }
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

        for scope in selected_scopes:
            head_path = run_eff_u_code(scope, output_dir=report_dir, eff_config_override=eff_override_path)
            head_reports[scope] = load_report(head_path)

        if args.scan_mode == "changed" and changed_baseline.revision:
            snapshot_root = export_git_snapshot(changed_baseline.revision, report_dir / "baseline")
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
                    base_ref=changed_baseline.normalized_base_ref or changed_baseline.base_ref or None,
                )
                base_reports[scope] = load_report(base_path)

        gate_rules = require_dict(gate_config, "gateRules", context="gate_config")
        scope_results: dict[str, Any] = {}
        overall_rule_failures = 0
        overall_unreported_failures = 0
        scope_scores: list[float] = []
        overall_score_failures = 0

        for scope in selected_scopes:
            head_index = build_file_index(head_reports[scope])
            base_index = build_file_index(base_reports.get(scope, {"files": []}))
            file_results = []
            unreported_files: list[str] = []
            root_path = scope_root(scope, gate_config)
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
                diagnostics = build_file_diagnostics(candidate_path, evaluations, gate_config, category_order)
                file_results.append(
                    {
                        "path": candidate_path,
                        "scope": scope,
                        "isNewFile": baseline_file is None,
                        "curatedScore": file_curated,
                        "gateStatus": "fail" if failures else "pass",
                        "scoreDiagnostics": diagnostics,
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
            score_values = [float(file_result["curatedScore"]) for file_result in file_results if file_result["curatedScore"] is not None]
            scope_quality_score = average(score_values)
            if scope_quality_score is not None:
                scope_scores.append(scope_quality_score)
            category_summary = summarize_category_impacts(file_results, category_order)
            severity_summary = summarize_severity(file_results)
            top_files = top_contributors(file_results, contributor_limit or len(file_results))
            score_gate_status = "pass"
            coverage_count = len(file_results)
            candidate_count = len(scoped_candidates[scope])
            score_gate_reason = None
            if score_enabled:
                coverage_blocks_score_gate = args.scan_mode == "changed"
                if unreported_files and coverage_blocks_score_gate:
                    score_gate_status = "fail"
                    score_gate_reason = "coverage-failure"
                elif scope_quality_score is None:
                    score_gate_status = "fail"
                    score_gate_reason = "missing-score"
                elif score_threshold is not None and scope_quality_score < score_threshold:
                    score_gate_status = "fail"
                    score_gate_reason = "below-threshold"
                if score_gate_status == "fail":
                    overall_score_failures += 1
            scope_results[scope] = {
                "changedFiles": scoped_candidates[scope] if args.scan_mode == "changed" else [],
                "candidateFiles": scoped_candidates[scope],
                "unreportedFiles": unreported_files,
                "coverageStatus": "fail" if unreported_files and args.scan_mode == "changed" else "pass",
                "coverageCount": coverage_count,
                "candidateCount": candidate_count,
                "root": root_path,
                "scopeQualityScore": scope_quality_score,
                "scoreGateStatus": score_gate_status,
                "scoreGateReason": score_gate_reason,
                "topContributors": [
                    {
                        "path": item["path"],
                        "displayPath": strip_scope_root(item["path"], root_path),
                        "impact": float((item.get("scoreDiagnostics") or {}).get("impact", 0.0)),
                        "highestSeverity": (item.get("scoreDiagnostics") or {}).get("highestSeverity", "low"),
                    }
                    for item in top_files
                ],
                "categorySummary": category_summary,
                "severitySummary": severity_summary,
                "potentialScoreGain": potential_score_gain(scope_quality_score, top_files, len(file_results), gain_steps) if score_enabled else [],
                "files": file_results,
            }

        overall_failures = overall_rule_failures + overall_unreported_failures
        overall_quality_score = average(scope_scores)
        effective_failures = overall_score_failures if score_enabled else overall_failures
        result = {
            "status": "fail" if effective_failures else "pass",
            "policy": "Graft Quality Policy",
            "tool": "eff-u-code",
            "toolRole": "raw-json-source",
            "gateMode": "repository-evaluator",
            "gateProfile": args.gate_profile,
            "scanMode": args.scan_mode,
            "changedFiles": changed_files,
            "changedBaseline": {
                "revision": changed_baseline.revision,
                "compareRevision": changed_baseline.compare_revision,
                "baseRef": changed_baseline.base_ref,
                "normalizedBaseRef": changed_baseline.normalized_base_ref,
                "source": changed_baseline.source,
            }
            if args.scan_mode == "changed"
            else None,
            "overallQualityScore": overall_quality_score,
            "scoreThreshold": score_threshold,
            "scopes": scope_results,
            "summary": {
                "filesEvaluated": sum(len(scope_results[scope]["files"]) for scope in scope_results),
                "failures": effective_failures,
                "rawFailures": overall_failures,
                "ruleFailures": overall_rule_failures,
                "coverageFailures": overall_unreported_failures,
                "unreportedFiles": sum(len(scope_results[scope]["unreportedFiles"]) for scope in scope_results),
                "scoreGateFailures": overall_score_failures,
                "overallQualityScore": overall_quality_score,
                "projectScoreThreshold": score_threshold,
            },
        }

        if args.output_json:
            output_path = pathlib.Path(args.output_json)
            output_path.parent.mkdir(parents=True, exist_ok=True)
            output_path.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

        if score_enabled and score_threshold is not None:
            for scope in selected_scopes:
                render_score_gate_summary(
                    scope,
                    scope_results[scope],
                    score_threshold,
                    contributor_limit,
                    detail_limit,
                    gain_steps,
                )
        else:
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
                    for rule_name in DEFAULT_DISPLAY_ORDER:
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

        return 1 if effective_failures else 0
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
