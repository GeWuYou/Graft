#!/usr/bin/env python3
"""Regression tests for the local eff-u-code wrapper."""

from __future__ import annotations

import importlib.util
import os
from pathlib import Path
import sys
import unittest
from unittest import mock


SCRIPT_PATH = Path(__file__).with_name("run_eff_u_code.py")
MODULE_SPEC = importlib.util.spec_from_file_location("run_eff_u_code", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"Unable to load module from {SCRIPT_PATH}.")

MODULE = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = MODULE
MODULE_SPEC.loader.exec_module(MODULE)


class CleanNodeDebugEnvironmentTests(unittest.TestCase):
    def test_removes_vscode_inspector_options(self) -> None:
        env = {"VSCODE_INSPECTOR_OPTIONS": '{"foo":"bar"}', "PATH": "/bin"}
        cleaned = MODULE.clean_node_debug_environment(env)
        self.assertNotIn("VSCODE_INSPECTOR_OPTIONS", cleaned)
        self.assertEqual(cleaned["PATH"], "/bin")

    def test_removes_inspect_flags_from_node_options(self) -> None:
        env = {"NODE_OPTIONS": "--max-old-space-size=4096 --inspect=0 --inspect-publish-uid=http"}
        cleaned = MODULE.clean_node_debug_environment(env)
        self.assertEqual(cleaned["NODE_OPTIONS"], "--max-old-space-size=4096")

    def test_removes_vscode_bootloader_require_forms(self) -> None:
        env = {
            "NODE_OPTIONS": (
                "--max-old-space-size=4096 "
                "--require /tmp/js-debug/bootloader.js "
                "--require=/tmp/other/BOOTLOADER.js "
                "--trace-warnings"
            )
        }
        cleaned = MODULE.clean_node_debug_environment(env)
        self.assertEqual(cleaned["NODE_OPTIONS"], "--max-old-space-size=4096 --trace-warnings")

    def test_removes_quoted_bootloader_path_with_spaces(self) -> None:
        env = {
            "NODE_OPTIONS": '--require "/tmp/js debug/bootloader.js" --max-old-space-size=4096'
        }
        cleaned = MODULE.clean_node_debug_environment(env)
        self.assertEqual(cleaned["NODE_OPTIONS"], "--max-old-space-size=4096")

    def test_drops_node_options_when_only_debug_tokens_exist(self) -> None:
        env = {"NODE_OPTIONS": "--inspect-brk=0 --require /tmp/js-debug/bootloader.js"}
        cleaned = MODULE.clean_node_debug_environment(env)
        self.assertNotIn("NODE_OPTIONS", cleaned)


class MainEnvironmentPropagationTests(unittest.TestCase):
    @mock.patch.object(MODULE.subprocess, "run")
    @mock.patch.object(MODULE, "load_config")
    @mock.patch.object(MODULE, "ensure_tree_sitter_wasms_layout")
    @mock.patch.object(MODULE, "resolve_local_tool")
    @mock.patch.object(MODULE, "parse_args")
    def test_main_runs_child_with_sanitized_environment(
        self,
        parse_args: mock.Mock,
        resolve_local_tool: mock.Mock,
        ensure_tree_sitter_wasms_layout: mock.Mock,
        load_config: mock.Mock,
        subprocess_run: mock.Mock,
    ) -> None:
        parse_args.return_value = mock.Mock(
            init_config=False,
            dry_run=False,
            config=None,
            scope="web",
            output_dir=None,
            locale=None,
            format=None,
            top=None,
            verbose=False,
        )
        resolve_local_tool.return_value = Path("/tmp/fuck-u-code")
        load_config.return_value = {
            "defaults": {"locale": "zh", "format": "console", "top": 20},
            "targets": {"web": {"path": "web/src", "exclude": ["**/*.d.ts"]}},
        }
        subprocess_run.return_value = mock.Mock(returncode=0)

        original_env = os.environ.copy()
        try:
            os.environ["VSCODE_INSPECTOR_OPTIONS"] = '{"inspectorIpc":"/tmp/node"}'
            os.environ["NODE_OPTIONS"] = "--inspect=0 --max-old-space-size=4096"
            exit_code = MODULE.main()
        finally:
            os.environ.clear()
            os.environ.update(original_env)

        self.assertEqual(exit_code, 0)
        ensure_tree_sitter_wasms_layout.assert_called_once_with()
        subprocess_run.assert_called_once()
        _, kwargs = subprocess_run.call_args
        self.assertEqual(kwargs["cwd"], MODULE.ROOT_DIR)
        self.assertEqual(kwargs["env"]["NODE_OPTIONS"], "--max-old-space-size=4096")
        self.assertNotIn("VSCODE_INSPECTOR_OPTIONS", kwargs["env"])


if __name__ == "__main__":
    unittest.main()
