"""Regression tests for the bounded log category governance scanner."""

from __future__ import annotations

import importlib.util
from pathlib import Path
import sys
import unittest


SCRIPT_PATH = Path(__file__).with_name("check_log_category_governance.py")
MODULE_SPEC = importlib.util.spec_from_file_location("check_log_category_governance", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"Unable to load module from {SCRIPT_PATH}.")
MODULE = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = MODULE
MODULE_SPEC.loader.exec_module(MODULE)


class ProductionPathTests(unittest.TestCase):
    """Keep test, generated, and vendor Go outside the bounded production scan."""

    def test_only_handwritten_production_server_go_is_scanned(self) -> None:
        self.assertTrue(MODULE.is_production_go_path(Path("server/modules/user/service.go")))
        self.assertFalse(MODULE.is_production_go_path(Path("server/modules/user/service_test.go")))
        self.assertFalse(MODULE.is_production_go_path(Path("server/internal/contract/generated/value.go")))
        self.assertFalse(MODULE.is_production_go_path(Path("server/vendor/example/value.go")))


class CategoryCallTests(unittest.TestCase):
    """Cover literal rejection without restricting typed logger constants."""

    def test_literal_category_is_rejected(self) -> None:
        findings = MODULE.scan_source(
            Path("server/modules/user/service.go"),
            'categoryLog := logger.Category(base, "runtime.cache")\ncategoryLog.Warn("failure")\n',
        )
        self.assertEqual([(finding.path, finding.line) for finding in findings], [(Path("server/modules/user/service.go"), 1)])

    def test_with_category_and_qualified_conversion_literals_are_rejected(self) -> None:
        findings = MODULE.scan_source(
            Path("server/modules/user/service.go"),
            'first := logger.WithCategory(base, "runtime.cache")\nsecond := logger.Category(base, logger.LogCategory("runtime.metrics"))\n',
        )
        self.assertEqual(
            [(finding.path, finding.line) for finding in findings],
            [(Path("server/modules/user/service.go"), 1), (Path("server/modules/user/service.go"), 2)],
        )

    def test_typed_category_is_allowed_with_multiline_base_expression(self) -> None:
        findings = MODULE.scan_source(
            Path("server/modules/user/service.go"),
            "categoryLog := logger.Category(\n\tbase.With(zap.String(\"module\", \"user\")),\n\tlogger.CategoryRuntimeCache,\n)\n",
        )
        self.assertEqual(findings, [])

    def test_literal_in_comment_or_string_is_not_a_call(self) -> None:
        findings = MODULE.scan_source(
            Path("server/modules/user/service.go"),
            '// logger.Category(base, "runtime.cache")\nmessage := "logger.Category(base, \\"runtime.cache\\")"\n',
        )
        self.assertEqual(findings, [])


if __name__ == "__main__":
    unittest.main()
