#!/usr/bin/env python3
# Copyright (c) 2025-2026 GeWuYou
# SPDX-License-Identifier: Apache-2.0

"""Validate curated shared asset registry structure."""

from __future__ import annotations

import argparse
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml


REPO_ROOT = Path(__file__).resolve().parents[1]
REGISTRY_DIR = REPO_ROOT / ".ai" / "registries"
REGISTRY_FILES = (
    REGISTRY_DIR / "web-shared-assets.yaml",
    REGISTRY_DIR / "server-shared-assets.yaml",
    REGISTRY_DIR / "cross-boundary-assets.yaml",
)

REQUIRED_FIELDS = {
    "name",
    "type",
    "path",
    "owner",
    "purpose",
    "use_when",
    "do_not_use_when",
}

ALLOWED_TYPES = {
    "frontend-component",
    "frontend-component-suite",
    "frontend-composable",
    "frontend-utility",
    "frontend-page-pattern",
    "frontend-contract",
    "backend-moduleapi",
    "backend-runtime-helper",
    "backend-registry",
    "backend-test-helper",
    "backend-validation-helper",
    "cross-boundary-contract",
    "governance-skill",
    "governance-validator",
}

FORBIDDEN_PARTS = {
    ".venv",
    "node_modules",
    "dist",
    "tmp",
    "coverage",
    "build",
    "__pycache__",
}

FORBIDDEN_PREFIXES = (
    ".ai/venv/",
    ".ai/ms-playwright/",
    ".ai/artifacts/",
    ".ai/headroom/",
    ".codegraph/",
)

GENERATED_MARKERS = (
    "/generated/",
    "/zz_generated.",
    ".bundle.json",
    ".gen.go",
)
GENERATED_PARTS = {"generated"}


@dataclass(frozen=True)
class Finding:
    path: Path
    message: str

    def format(self) -> str:
        try:
            display_path = self.path.relative_to(REPO_ROOT)
        except ValueError:
            display_path = self.path
        return f"{display_path}: {self.message}"


def repo_relative(path_value: str) -> Path:
    if not path_value or path_value.startswith("/"):
        raise ValueError("path must be repository-relative")
    if ".." in Path(path_value).parts:
        raise ValueError("path must not contain '..'")
    return REPO_ROOT / path_value


def normalized_path(path_value: str) -> str:
    return path_value.replace("\\", "/").strip()


def is_forbidden_output_path(path_value: str) -> bool:
    path_value = normalized_path(path_value)
    if any(path_value.startswith(prefix) for prefix in FORBIDDEN_PREFIXES):
        return True
    return any(part in FORBIDDEN_PARTS for part in Path(path_value).parts)


def is_generated_only_authority(path_value: str) -> bool:
    path_value = normalized_path(path_value)
    if any(part in GENERATED_PARTS for part in Path(path_value).parts):
        return True
    return any(marker in path_value for marker in GENERATED_MARKERS)


def validate_string_list(asset: dict[str, Any], field: str, registry_path: Path) -> list[Finding]:
    value = asset.get(field)
    if not isinstance(value, list) or not value:
        return [Finding(registry_path, f"asset {asset.get('name', '<unknown>')!r} field {field!r} must be a non-empty list")]
    findings: list[Finding] = []
    for item in value:
        if not isinstance(item, str) or not item.strip():
            findings.append(Finding(registry_path, f"asset {asset.get('name', '<unknown>')!r} field {field!r} contains a non-string or empty item"))
    return findings


def validate_path_field(registry_path: Path, asset: dict[str, Any], field: str, *, allow_missing: bool = False) -> list[Finding]:
    value = asset.get(field)
    if not isinstance(value, str) or not value.strip():
        return [Finding(registry_path, f"asset {asset.get('name', '<unknown>')!r} field {field!r} must be a non-empty string")]

    path_value = normalized_path(value)
    findings: list[Finding] = []
    try:
        absolute = repo_relative(path_value)
    except ValueError as exc:
        return [Finding(registry_path, f"asset {asset.get('name', '<unknown>')!r} field {field!r}: {exc}")]

    if is_forbidden_output_path(path_value):
        findings.append(Finding(registry_path, f"asset {asset.get('name', '<unknown>')!r} field {field!r} points at forbidden generated/local output path {path_value!r}"))
    if is_generated_only_authority(path_value):
        findings.append(Finding(registry_path, f"asset {asset.get('name', '<unknown>')!r} field {field!r} points at generated-only artifact {path_value!r}"))
    if not allow_missing and not absolute.exists():
        findings.append(Finding(registry_path, f"asset {asset.get('name', '<unknown>')!r} field {field!r} path does not exist: {path_value}"))
    return findings


def validate_asset(registry_path: Path, asset: Any, seen_names: set[str]) -> list[Finding]:
    if not isinstance(asset, dict):
        return [Finding(registry_path, "each asset entry must be a mapping")]

    findings: list[Finding] = []
    missing = sorted(field for field in REQUIRED_FIELDS if field not in asset)
    if missing:
        findings.append(Finding(registry_path, f"asset {asset.get('name', '<unknown>')!r} missing required fields: {', '.join(missing)}"))

    name = asset.get("name")
    if not isinstance(name, str) or not name.strip():
        findings.append(Finding(registry_path, "asset name must be a non-empty string"))
    elif name in seen_names:
        findings.append(Finding(registry_path, f"duplicate asset name {name!r}"))
    elif name.strip() != name:
        findings.append(Finding(registry_path, f"asset name {name!r} must not have surrounding whitespace"))
    else:
        seen_names.add(name)

    asset_type = asset.get("type")
    if not isinstance(asset_type, str) or asset_type not in ALLOWED_TYPES:
        findings.append(Finding(registry_path, f"asset {name!r} has invalid type {asset_type!r}"))

    for field in ("owner", "purpose"):
        value = asset.get(field)
        if not isinstance(value, str) or not value.strip():
            findings.append(Finding(registry_path, f"asset {name!r} field {field!r} must be a non-empty string"))

    findings.extend(validate_path_field(registry_path, asset, "path"))
    findings.extend(validate_string_list(asset, "use_when", registry_path))
    findings.extend(validate_string_list(asset, "do_not_use_when", registry_path))

    for optional_list in ("related_assets", "validation_commands"):
        if optional_list in asset:
            findings.extend(validate_string_list(asset, optional_list, registry_path))

    if "examples" in asset:
        examples = asset.get("examples")
        if not isinstance(examples, list):
            findings.append(Finding(registry_path, f"asset {name!r} field 'examples' must be a list"))
        else:
            for example in examples:
                if not isinstance(example, str) or not example.strip():
                    findings.append(Finding(registry_path, f"asset {name!r} has an invalid example path"))
                    continue
                example_asset = {"name": name, "path": example}
                findings.extend(validate_path_field(registry_path, example_asset, "path"))

    return findings


def validate_registry(path: Path, seen_names: set[str]) -> list[Finding]:
    if not path.is_file():
        return [Finding(path, "registry file is missing")]

    try:
        data = yaml.safe_load(path.read_text(encoding="utf-8"))
    except yaml.YAMLError as exc:
        return [Finding(path, f"invalid YAML: {exc}")]

    if not isinstance(data, dict):
        return [Finding(path, "registry root must be a mapping")]

    findings: list[Finding] = []
    if data.get("schema_version") != 1:
        findings.append(Finding(path, "schema_version must be 1"))
    updated_at = data.get("updated_at")
    if not isinstance(updated_at, str) or not updated_at.strip():
        findings.append(Finding(path, "updated_at must be a non-empty string"))
    assets = data.get("assets")
    if not isinstance(assets, list) or not assets:
        findings.append(Finding(path, "assets must be a non-empty list"))
        return findings

    for asset in assets:
        findings.extend(validate_asset(path, asset, seen_names))
    return findings


def validate_registries() -> list[Finding]:
    findings: list[Finding] = []
    seen_names: set[str] = set()
    for registry_file in REGISTRY_FILES:
        findings.extend(validate_registry(registry_file, seen_names))
    return findings


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.parse_args(argv)
    findings = validate_registries()
    if findings:
        for finding in findings:
            print(finding.format(), file=sys.stderr)
        return 1
    print("Shared asset registry validation passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
