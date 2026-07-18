"""Focused regression tests for the bounded error logging governance scanner."""

from __future__ import annotations

import importlib.util
from pathlib import Path
import sys
import unittest


SCRIPT_PATH = Path(__file__).with_name("check_error_logging_governance.py")
MODULE_SPEC = importlib.util.spec_from_file_location("check_error_logging_governance", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"Unable to load module from {SCRIPT_PATH}.")
MODULE = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = MODULE
MODULE_SPEC.loader.exec_module(MODULE)


class ErrorLoggingGovernanceTests(unittest.TestCase):
    """Keep this guard limited to direct, unambiguous logging regressions."""

    def test_rejects_each_governed_regression(self) -> None:
        source = """\
engine.Use(gin.Recovery())
return fmt.Errorf("load: %v", err)
return errors.New(err.Error())
accessLogger.Error("http access")
WriteError(gin.H{"data": err.Error()})
"""
        findings = MODULE.scan_source(Path("server/internal/httpx/example.go"), source)
        self.assertEqual(len(findings), 5)
        self.assertEqual({finding.line for finding in findings}, {1, 2, 3, 4, 5})

    def test_allows_wrapped_internal_error_and_info_access_log(self) -> None:
        source = """\
return fmt.Errorf("load: %w", err)
accessLogger.Info("http access")
message := err.Error()
"""
        self.assertEqual(MODULE.scan_source(Path("server/internal/httpx/example.go"), source), [])

    def test_excludes_tests_and_generated_files(self) -> None:
        self.assertFalse(MODULE.is_production_go_path(Path("server/internal/httpx/server_test.go")))
        self.assertFalse(MODULE.is_production_go_path(Path("server/internal/contract/generated/error.go")))


if __name__ == "__main__":
    unittest.main()
