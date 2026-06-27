#!/usr/bin/env python3
"""Run one server `go test -json` stream and emit a single CTRF report file."""

from __future__ import annotations

import argparse
import os
import platform
from dataclasses import dataclass
from pathlib import Path
import subprocess
import sys


REPO_ROOT = Path(__file__).resolve().parents[2]
SERVER_ROOT = REPO_ROOT / "server"
DEFAULT_OUTPUT = Path(".tmp/test-results/server-go-test-ctrf.json")
DEFAULT_APP_NAME = "graft-server"
DEFAULT_REPORTER_VERSION = "v0.1.0"
REPORTER_PACKAGE = "github.com/ctrf-io/go-ctrf-json-reporter/cmd/go-ctrf-json-reporter"


@dataclass(frozen=True)
class ReporterMetadata:
    app_name: str
    app_version: str
    os_platform: str
    os_release: str
    os_version: str
    build_name: str
    build_number: str


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Run a single authoritative `go test -json` pass for `server` and "
            "convert that stream into CTRF JSON for GitHub test reporting."
        )
    )
    parser.add_argument(
        "--output",
        default=str(DEFAULT_OUTPUT),
        help="CTRF output path relative to the repository root unless absolute.",
    )
    parser.add_argument(
        "--reporter-version",
        default=DEFAULT_REPORTER_VERSION,
        help="Pinned go-ctrf-json-reporter version to execute with `go run`.",
    )
    parser.add_argument(
        "--test-target",
        action="append",
        dest="test_targets",
        default=[],
        help="`go test` package target to validate; repeatable, defaults to ./....",
    )
    parser.add_argument(
        "--go-test-arg",
        action="append",
        dest="go_test_args",
        default=[],
        help="Additional argument forwarded to `go test` before package targets; repeatable.",
    )
    parser.add_argument("--app-name", default=DEFAULT_APP_NAME, help="CTRF environment appName.")
    parser.add_argument("--app-version", default="", help="CTRF environment appVersion.")
    parser.add_argument("--build-name", default="", help="CTRF environment buildName.")
    parser.add_argument("--build-number", default="", help="CTRF environment buildNumber.")
    parser.add_argument("--os-platform", default="", help="CTRF environment osPlatform.")
    parser.add_argument("--os-release", default="", help="CTRF environment osRelease.")
    parser.add_argument("--os-version", default="", help="CTRF environment osVersion.")
    return parser.parse_args(argv)


def resolve_output_path(raw_output: str, repo_root: Path = REPO_ROOT) -> Path:
    output_path = Path(raw_output)
    if output_path.is_absolute():
        return output_path
    return repo_root / output_path


def resolve_reporter_metadata(
    args: argparse.Namespace,
    env: dict[str, str] | None = None,
) -> ReporterMetadata:
    env = os.environ if env is None else env
    build_name = args.build_name or env.get("GITHUB_HEAD_REF") or env.get("GITHUB_REF_NAME", "")
    build_number = args.build_number or env.get("GITHUB_RUN_NUMBER") or env.get("GITHUB_RUN_ID", "")
    app_version = args.app_version or env.get("GITHUB_SHA", "")

    os_platform = args.os_platform or platform.system()
    os_release = args.os_release or platform.release()
    os_version = args.os_version or platform.version()

    return ReporterMetadata(
        app_name=args.app_name,
        app_version=app_version,
        os_platform=os_platform,
        os_release=os_release,
        os_version=os_version,
        build_name=build_name,
        build_number=build_number,
    )


def build_go_test_command(test_targets: list[str], go_test_args: list[str]) -> list[str]:
    targets = test_targets or ["./..."]
    return ["go", "test", "-json", *go_test_args, *targets]


def build_reporter_command(
    output_path: Path,
    reporter_version: str,
    metadata: ReporterMetadata,
) -> list[str]:
    command = [
        "go",
        "run",
        f"{REPORTER_PACKAGE}@{reporter_version}",
        "-output",
        str(output_path),
        "-verbose",
    ]

    metadata_flags = (
        ("-appName", metadata.app_name),
        ("-appVersion", metadata.app_version),
        ("-osPlatform", metadata.os_platform),
        ("-osRelease", metadata.os_release),
        ("-osVersion", metadata.os_version),
        ("-buildName", metadata.build_name),
        ("-buildNumber", metadata.build_number),
    )
    for flag, value in metadata_flags:
        if value:
            command.extend([flag, value])

    return command


def run_pipeline(
    go_test_command: list[str],
    reporter_command: list[str],
    *,
    server_root: Path = SERVER_ROOT,
    repo_root: Path = REPO_ROOT,
    popen: type[subprocess.Popen[bytes]] | callable = subprocess.Popen,
) -> tuple[int, int]:
    go_test = popen(
        go_test_command,
        cwd=server_root,
        stdout=subprocess.PIPE,
    )
    if go_test.stdout is None:
        raise RuntimeError("go test process did not expose stdout for CTRF piping")

    reporter = popen(
        reporter_command,
        cwd=repo_root,
        stdin=go_test.stdout,
    )
    go_test.stdout.close()

    reporter_exit = reporter.wait()
    go_test_exit = go_test.wait()
    return go_test_exit, reporter_exit


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    output_path = resolve_output_path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)

    metadata = resolve_reporter_metadata(args)
    go_test_command = build_go_test_command(args.test_targets, args.go_test_args)
    reporter_command = build_reporter_command(output_path, args.reporter_version, metadata)

    go_test_exit, reporter_exit = run_pipeline(go_test_command, reporter_command)

    if reporter_exit != 0:
        return reporter_exit
    if go_test_exit != 0:
        return go_test_exit
    if not output_path.is_file():
        print(f"expected CTRF report at {output_path}, but no file was written", file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
