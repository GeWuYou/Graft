#!/usr/bin/env python3
"""Regression tests for the server CTRF report helper."""

from __future__ import annotations

import io
import importlib.util
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock


SCRIPT_PATH = Path(__file__).parent / "test-reporting" / "run_server_ctrf_report.py"
MODULE_SPEC = importlib.util.spec_from_file_location("run_server_ctrf_report", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"Unable to load module from {SCRIPT_PATH}.")

MODULE = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = MODULE
MODULE_SPEC.loader.exec_module(MODULE)


class CommandBuilderTests(unittest.TestCase):
    def test_build_go_test_command_defaults_to_all_packages(self) -> None:
        command = MODULE.build_go_test_command([], [])

        self.assertEqual(command, ["go", "test", "-json", "./..."])

    def test_build_reporter_command_uses_pinned_upstream_binary_path(self) -> None:
        metadata = MODULE.ReporterMetadata(
            app_name="graft-server",
            app_version="deadbeef",
            os_platform="Linux",
            os_release="6.8.0",
            os_version="Ubuntu",
            build_name="feat/test",
            build_number="42",
        )

        command = MODULE.build_reporter_command(
            Path("/tmp/report.json"),
            "v0.1.0",
            metadata,
        )

        self.assertEqual(command[:3], ["go", "run", f"{MODULE.REPORTER_PACKAGE}@v0.1.0"])
        self.assertIn("-output", command)
        self.assertEqual(command[command.index("-output") + 1], "/tmp/report.json")
        self.assertIn("-verbose", command)
        self.assertIn("-appName", command)
        self.assertIn("-buildNumber", command)


class PipelineTests(unittest.TestCase):
    def test_run_pipeline_wires_go_test_stdout_into_reporter(self) -> None:
        class FakePipe:
            def __init__(self) -> None:
                self.closed = False

            def close(self) -> None:
                self.closed = True

        class FakeProcess:
            def __init__(self, returncode: int, stdout=None) -> None:
                self.returncode = returncode
                self.stdout = stdout
                self.wait_calls = 0

            def wait(self) -> int:
                self.wait_calls += 1
                return self.returncode

        go_pipe = FakePipe()
        go_process = FakeProcess(0, stdout=go_pipe)
        reporter_process = FakeProcess(0)
        calls: list[tuple[list[str], dict[str, object]]] = []

        def fake_popen(command, **kwargs):
            calls.append((command, kwargs))
            if len(calls) == 1:
                return go_process
            return reporter_process

        go_test_exit, reporter_exit = MODULE.run_pipeline(
            ["go", "test", "-json", "./internal/buildinfo"],
            ["go", "run", "example.com/reporter@v1.2.3"],
            server_root=Path("/repo/server"),
            repo_root=Path("/repo"),
            popen=fake_popen,
        )

        self.assertEqual((go_test_exit, reporter_exit), (0, 0))
        self.assertEqual(calls[0][1]["cwd"], Path("/repo/server"))
        self.assertIs(calls[0][1]["stdout"], MODULE.subprocess.PIPE)
        self.assertEqual(calls[1][1]["cwd"], Path("/repo"))
        self.assertIs(calls[1][1]["stdin"], go_pipe)
        self.assertTrue(go_pipe.closed)
        self.assertEqual(go_process.wait_calls, 1)
        self.assertEqual(reporter_process.wait_calls, 1)


class MainTests(unittest.TestCase):
    def test_main_requires_report_file_after_success(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            output_path = Path(temp_dir) / "server-ctrf.json"

            with mock.patch.object(MODULE, "run_pipeline", return_value=(0, 0)) as run_pipeline, mock.patch(
                "sys.stderr",
                new_callable=io.StringIO,
            ) as stderr:
                exit_code = MODULE.main(
                    [
                        "--output",
                        str(output_path),
                    ]
                )

            run_pipeline.assert_called_once()
            self.assertNotEqual(exit_code, 0)
            self.assertIn("expected CTRF report at", stderr.getvalue())


if __name__ == "__main__":
    unittest.main()
