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

    def test_command_requires_successful_execution_probe(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_dir:
            fake_rg = Path(temporary_dir) / "rg"
            fake_rg.write_text("#!/usr/bin/env bash\nexit 126\n", encoding="utf-8")
            fake_rg.chmod(0o755)
            completed = subprocess.run(
                [
                    "bash",
                    "-c",
                    'source "$1"; PATH="$2:/usr/bin:/bin"; command_installed rg; printf "\\n"; command_path rg',
                    "bash",
                    str(SCRIPT_PATH),
                    temporary_dir,
                ],
                check=False,
                capture_output=True,
                text=True,
            )

        self.assertEqual(completed.returncode, 0, completed.stderr)
        self.assertEqual(completed.stdout, "false\n")


if __name__ == "__main__":
    unittest.main()
