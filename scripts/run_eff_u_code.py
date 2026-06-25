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
    merged: dict[str, Any] = dict(base)
    for key, value in override.items():
        current = merged.get(key)
        if isinstance(current, dict) and isinstance(value, dict):
            merged[key] = deep_merge(current, value)
        else:
            merged[key] = value
    return merged


def load_config() -> dict[str, Any]:
    config = load_json(EXAMPLE_CONFIG)
    if LOCAL_CONFIG.is_file():
        config = deep_merge(config, load_json(LOCAL_CONFIG))
    return config


def require_string(container: dict[str, Any], key: str, *, context: str) -> str:
    value = container.get(key)
    if not isinstance(value, str) or not value.strip():
        raise ConfigError(f"{context}.{key} must be a non-empty string")
    return value


def require_int(container: dict[str, Any], key: str, *, context: str) -> int:
    value = container.get(key)
    if not isinstance(value, int) or value <= 0:
        raise ConfigError(f"{context}.{key} must be a positive integer")
    return value


def require_string_list(container: dict[str, Any], key: str, *, context: str) -> list[str]:
    value = container.get(key)
    if not isinstance(value, list) or not all(isinstance(item, str) and item.strip() for item in value):
        raise ConfigError(f"{context}.{key} must be a string array")
    return value


def build_scope_config(config: dict[str, Any], scope: str, overrides: argparse.Namespace) -> dict[str, Any]:
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
    if requested_scope == "all":
        return list(SCOPES)
    if requested_scope not in SCOPES:
        raise ConfigError(f"unsupported scope: {requested_scope}")
    return [requested_scope]


def build_command(scope_config: dict[str, Any], *, output_file: Path | None, verbose: bool) -> list[str]:
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
    local_tool = LOCAL_BIN_DIR / LOCAL_TOOL_NAME
    if local_tool.is_file():
        return local_tool
    raise ConfigError(
        "missing project-local eff-u-code install: run `bun install` at the repository root "
        "and avoid using the global eff-u-code package"
    )


def ensure_tree_sitter_wasms_layout() -> None:
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
    if LOCAL_CONFIG.exists():
        raise ConfigError(f"local config already exists: {LOCAL_CONFIG}")
    LOCAL_CONFIG.write_text(EXAMPLE_CONFIG.read_text(encoding="utf-8"), encoding="utf-8")
    print(f"Wrote {LOCAL_CONFIG}")


def parse_args() -> argparse.Namespace:
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
    if output_format == "console":
        return "txt"
    return output_format


def main() -> int:
    args = parse_args()

    try:
        tool = resolve_local_tool()
        ensure_tree_sitter_wasms_layout()
        if args.init_config:
            init_local_config()

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
