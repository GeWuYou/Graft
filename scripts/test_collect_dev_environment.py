#!/usr/bin/env python3
"""Regression tests for the development environment collector."""

from __future__ import annotations

from pathlib import Path
import subprocess
import tempfile
import unittest


SCRIPT_PATH = Path(__file__).with_name("collect-dev-environment.sh")


def call_collector_function(function_name: str, *arguments: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [
            "bash",
            "-c",
            'source "$1"; shift; "$@"',
            "bash",
            str(SCRIPT_PATH),
            function_name,
            *arguments,
        ],
        check=False,
        capture_output=True,
        text=True,
    )


class EnvironmentCollectorTests(unittest.TestCase):
    def test_missing_os_release_returns_unknown(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_dir:
            missing_path = Path(temporary_dir) / "missing-os-release"
            completed = call_collector_function("read_os_release", "PRETTY_NAME", str(missing_path))

        self.assertEqual(completed.returncode, 0, completed.stderr)
        self.assertEqual(completed.stdout, "unknown\n")

    def test_missing_headroom_has_no_partial_mcp_command(self) -> None:
        completed = call_collector_function("headroom_mcp_command", "")

        self.assertEqual(completed.returncode, 0, completed.stderr)
        self.assertEqual(completed.stdout, "unavailable")


if __name__ == "__main__":
    unittest.main()
