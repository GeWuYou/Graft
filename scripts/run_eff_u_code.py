#!/usr/bin/env python3
"""Run eff-u-code as a developer-local optional check for Graft."""

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import sys
from pathlib import Path
from typing import Any


ROOT_DIR = Path(__file__).resolve().parents[1]
EXAMPLE_CONFIG = ROOT_DIR / "scripts" / "eff-u-code.example.json"
LOCAL_CONFIG = ROOT_DIR / ".eff-u-code.local.json"
SCOPES = ("server", "web")
LOCAL_BIN_DIR = ROOT_DIR / "node_modules" / ".bin"
LOCAL_NODE_MODULES_DIR = ROOT_DIR / "node_modules"
EFF_U_CODE_DIR = LOCAL_NODE_MODULES_DIR / "eff-u-code"
TREE_SITTER_WASMS_DIR = LOCAL_NODE_MODULES_DIR / "tree-sitter-wasms"
EFF_U_CODE_WASMS_DIR = EFF_U_CODE_DIR / "node_modules" / "tree-sitter-wasms"
LOCAL_TOOL_NAME = "fuck-u-code.cmd" if sys.platform == "win32" else "fuck-u-code"


class ConfigError(RuntimeError):
    """Raised when the optional local configuration is invalid."""


def load_json(path: Path) -> dict[str, Any]:
    """
    读取 UTF-8 编码的 JSON 文件并返回其对象根值。
    
    Returns:
    	value (dict[str, Any]): 解析得到的 JSON 对象。
    
    Raises:
    	ConfigError: 当配置文件缺失、JSON 无法解析或根值不是对象时。
    """
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise ConfigError(f"missing config file: {path}") from exc
    except json.JSONDecodeError as exc:
        raise ConfigError(f"invalid JSON in {path}: {exc}") from exc
    if not isinstance(value, dict):
        raise ConfigError(f"config root must be an object: {path}")
    return value


def deep_merge(base: dict[str, Any], override: dict[str, Any]) -> dict[str, Any]:
    """
    递归合并两个字典并返回结果。
    
    Parameters:
    	base (dict[str, Any]): 基础字典。
    	override (dict[str, Any]): 覆盖字典。
    
    Returns:
    	dict[str, Any]: 合并后的字典。
    """
    merged: dict[str, Any] = dict(base)
    for key, value in override.items():
        current = merged.get(key)
        if isinstance(current, dict) and isinstance(value, dict):
            merged[key] = deep_merge(current, value)
        else:
            merged[key] = value
    return merged


def load_config() -> dict[str, Any]:
    """
    加载示例配置，并在存在本地配置时将其覆盖合并进去。
    
    Returns:
    	config (dict[str, Any]): 合并后的配置字典。
    """
    config = load_json(EXAMPLE_CONFIG)
    if LOCAL_CONFIG.is_file():
        config = deep_merge(config, load_json(LOCAL_CONFIG))
    return config


def require_string(container: dict[str, Any], key: str, *, context: str) -> str:
    """
    获取容器中指定键的非空字符串值。
    
    Parameters:
    	context (str): 用于错误信息的配置上下文名称。
    
    Returns:
    	str: 指定键对应的非空字符串值。
    """
    value = container.get(key)
    if not isinstance(value, str) or not value.strip():
        raise ConfigError(f"{context}.{key} must be a non-empty string")
    return value


def require_int(container: dict[str, Any], key: str, *, context: str) -> int:
    """
    要求容器中的指定项为正整数。
    
    参数:
        container (dict[str, Any]): 待检查的映射。
        key (str): 要读取的键名。
        context (str): 用于错误信息的上下文名称。
    
    返回:
        int: 指定键对应的正整数值。
    """
    value = container.get(key)
    if not isinstance(value, int) or value <= 0:
        raise ConfigError(f"{context}.{key} must be a positive integer")
    return value


def require_string_list(container: dict[str, Any], key: str, *, context: str) -> list[str]:
    """
    获取指定配置项中的字符串数组。
    
    Parameters:
    	key (str): 要读取的键名。
    	context (str): 用于错误信息的配置上下文。
    
    Returns:
    	list[str]: 由非空字符串组成的列表。
    """
    value = container.get(key)
    if not isinstance(value, list) or not all(isinstance(item, str) and item.strip() for item in value):
        raise ConfigError(f"{context}.{key} must be a string array")
    return value


def build_scope_config(config: dict[str, Any], scope: str, overrides: argparse.Namespace) -> dict[str, Any]:
    """
    构建指定 scope 的运行配置，并应用命令行覆盖项。
    
    Parameters:
    	config (dict[str, Any]): 读取到的配置对象。
    	scope (str): 目标分析范围名称。
    	overrides (argparse.Namespace): 命令行参数覆盖值。
    
    Returns:
    	dict[str, Any]: 包含 `path`、`locale`、`format`、`top` 和 `exclude` 的配置字典。
    """
    defaults = config.get("defaults")
    targets = config.get("targets")
    if not isinstance(defaults, dict):
        raise ConfigError("defaults must be an object")
    if not isinstance(targets, dict):
        raise ConfigError("targets must be an object")
    target = targets.get(scope)
    if not isinstance(target, dict):
        raise ConfigError(f"targets.{scope} must be an object")

    merged = deep_merge(defaults, target)
    path = require_string(merged, "path", context=f"targets.{scope}")
    locale = overrides.locale or require_string(merged, "locale", context=f"targets.{scope}")
    output_format = overrides.format or require_string(merged, "format", context=f"targets.{scope}")
    top = overrides.top or require_int(merged, "top", context=f"targets.{scope}")
    exclude = require_string_list(merged, "exclude", context=f"targets.{scope}")
    return {
        "path": path,
        "locale": locale,
        "format": output_format,
        "top": top,
        "exclude": exclude,
    }


def resolve_scopes(requested_scope: str) -> list[str]:
    """
    解析要执行的分析范围。
    
    Parameters:
    	requested_scope (str): 请求的范围名称。
    
    Returns:
    	list[str]: 需要执行的范围列表。
    """
    if requested_scope == "all":
        return list(SCOPES)
    if requested_scope not in SCOPES:
        raise ConfigError(f"unsupported scope: {requested_scope}")
    return [requested_scope]


def build_command(scope_config: dict[str, Any], *, output_file: Path | None, verbose: bool) -> list[str]:
    """
    构建 eff-u-code 的分析命令参数。
    
    Parameters:
    	scope_config (dict[str, Any]): 作用域配置，包含路径、语言、输出格式、数量限制和排除模式。
    	output_file (Path | None): 输出文件路径；为 `None` 时不添加输出参数。
    	verbose (bool): 是否启用详细输出。
    
    Returns:
    	list[str]: 生成的命令参数列表。
    """
    command = [
        "analyze",
        scope_config["path"],
        "--locale",
        scope_config["locale"],
        "--format",
        scope_config["format"],
        "--top",
        str(scope_config["top"]),
    ]
    if verbose:
        command.append("--verbose")
    for pattern in scope_config["exclude"]:
        command.extend(["--exclude", pattern])
    if output_file is not None:
        command.extend(["--output", str(output_file)])
    return command


def resolve_local_tool() -> Path:
    """
    定位项目本地的 eff-u-code 可执行文件。
    
    Returns:
    	local_tool (Path): 项目本地 eff-u-code 可执行文件路径。
    
    Raises:
    	ConfigError: 当未找到项目本地安装时抛出。
    """
    local_tool = LOCAL_BIN_DIR / LOCAL_TOOL_NAME
    if local_tool.is_file():
        return local_tool
    raise ConfigError(
        "missing project-local eff-u-code install: run `bun install` at the repository root "
        "and avoid using the global eff-u-code package"
    )


def ensure_tree_sitter_wasms_layout() -> None:
    """
    确保 eff-u-code 所需的 tree-sitter-wasms 目录布局可用。
    
    如果项目本地未安装 tree-sitter-wasms，则抛出配置错误；否则会在 eff-u-code 的本地依赖目录下建立指向该目录的符号链接，并清理冲突的旧目录或链接。
    """
    if not TREE_SITTER_WASMS_DIR.is_dir():
        raise ConfigError(
            "missing project-local tree-sitter-wasms install: run `bun install` at the repository root"
        )

    expected_out_dir = EFF_U_CODE_WASMS_DIR / "out"
    if expected_out_dir.is_dir():
        return

    EFF_U_CODE_WASMS_DIR.parent.mkdir(parents=True, exist_ok=True)
    if EFF_U_CODE_WASMS_DIR.exists():
        if EFF_U_CODE_WASMS_DIR.is_symlink() and EFF_U_CODE_WASMS_DIR.resolve() == TREE_SITTER_WASMS_DIR.resolve():
            return
        if EFF_U_CODE_WASMS_DIR.is_dir():
            shutil.rmtree(EFF_U_CODE_WASMS_DIR)
        else:
            EFF_U_CODE_WASMS_DIR.unlink()

    EFF_U_CODE_WASMS_DIR.symlink_to(TREE_SITTER_WASMS_DIR, target_is_directory=True)


def init_local_config() -> None:
    """
    创建本地配置文件。
    
    如果本地配置已存在，则引发错误；否则将示例配置复制为本地配置并写入磁盘。
    """
    if LOCAL_CONFIG.exists():
        raise ConfigError(f"local config already exists: {LOCAL_CONFIG}")
    LOCAL_CONFIG.write_text(EXAMPLE_CONFIG.read_text(encoding="utf-8"), encoding="utf-8")
    print(f"Wrote {LOCAL_CONFIG}")


def parse_args() -> argparse.Namespace:
    """
    解析 eff-u-code 本地检查脚本的命令行参数。
    
    Returns:
    	Namespace: 解析后的命令行参数。
    """
    parser = argparse.ArgumentParser(
        description="Run eff-u-code as an optional local Graft quality check.",
    )
    parser.add_argument(
        "scope",
        nargs="?",
        default="all",
        choices=("server", "web", "all"),
        help="check scope: server, web, or all",
    )
    parser.add_argument(
        "--config",
        help="use an alternate eff-u-code JSON config file instead of the tracked/local merge",
    )
    parser.add_argument(
        "--format",
        choices=("console", "markdown", "json", "html"),
        help="override report format",
    )
    parser.add_argument("--locale", choices=("en", "zh", "ru"), help="override output locale")
    parser.add_argument("--top", type=int, help="override top-N worst files")
    parser.add_argument("--output-dir", help="write per-scope reports into a directory")
    parser.add_argument("--verbose", action="store_true", help="enable eff-u-code verbose mode")
    parser.add_argument("--dry-run", action="store_true", help="print commands without executing them")
    parser.add_argument(
        "--init-config",
        action="store_true",
        help="copy scripts/eff-u-code.example.json to .eff-u-code.local.json",
    )
    return parser.parse_args()


def suffix_for_format(output_format: str) -> str:
    """
    将输出格式映射为文件后缀。
    
    Parameters:
    	output_format (str): 输出格式名称。
    
    Returns:
    	str: 对应的文件后缀；`console` 对应 `txt`，其余值保持不变。
    """
    if output_format == "console":
        return "txt"
    return output_format


def main() -> int:
    """
    运行本地 `eff-u-code` 检查并返回退出码。
    
    Returns:
    	exit_code (int): 成功时为 `0`；配置或本地环境错误时为 `2`；任一 scope 执行失败时为该进程的返回码。
    """
    args = parse_args()

    try:
        if args.init_config:
            init_local_config()
            return 0

        tool = resolve_local_tool()
        if not args.dry_run:
            ensure_tree_sitter_wasms_layout()

        if args.config:
            config = load_json(Path(args.config).resolve())
        else:
            config = load_config()
        scopes = resolve_scopes(args.scope)
        output_dir = Path(args.output_dir).resolve() if args.output_dir else None
        if output_dir is not None and not args.dry_run:
            output_dir.mkdir(parents=True, exist_ok=True)

        for scope in scopes:
            scope_config = build_scope_config(config, scope, args)
            output_file = None
            if output_dir is not None:
                output_file = output_dir / f"eff-u-code-{scope}.{suffix_for_format(scope_config['format'])}"

            command = [str(tool), *build_command(scope_config, output_file=output_file, verbose=args.verbose)]
            print(f"[eff-u-code:{scope}] {' '.join(command)}")
            if args.dry_run:
                continue
            completed = subprocess.run(command, cwd=ROOT_DIR)
            if completed.returncode != 0:
                return completed.returncode

        return 0
    except ConfigError as exc:
        print(f"run_eff_u_code.py: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
