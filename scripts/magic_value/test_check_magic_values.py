#!/usr/bin/env python3
"""Regression tests for the contract governance scanner."""

from __future__ import annotations

import importlib.util
from pathlib import Path
import sys
import unittest


SCRIPT_PATH = Path(__file__).with_name("check_magic_values.py")
MODULE_SPEC = importlib.util.spec_from_file_location("check_magic_values", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"Unable to load module from {SCRIPT_PATH}.")

MODULE = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = MODULE
MODULE_SPEC.loader.exec_module(MODULE)


class DefinitionContextTests(unittest.TestCase):
    """Cover narrow definition-context exemptions for contract scanning."""

    def test_runtime_i18n_file_only_whitelists_message_contract_references(self) -> None:
        """Runtime files should not act as blanket contract owners."""
        self.assertTrue(
            MODULE.is_definition_context(
                "server/internal/i18n/service.go",
                'messagecontract.AuthForbidden.String(): "Forbidden",',
                "auth.forbidden",
            )
        )
        self.assertFalse(
            MODULE.is_definition_context(
                "server/internal/i18n/service.go",
                'const accidental = "auth.forbidden"',
                "auth.forbidden",
            )
        )

    def test_test_file_api_path_constant_is_treated_as_local_definition(self) -> None:
        """Local test constants may define repeated fixture paths without widening runtime ownership."""
        self.assertTrue(
            MODULE.is_definition_context(
                "web/src/utils/request.test.ts",
                "const USERS_API_PATH = '/api/users';",
                "/api/users",
            )
        )
        self.assertFalse(
            MODULE.is_definition_context(
                "web/src/utils/request.ts",
                "const USERS_API_PATH = '/api/users';",
                "/api/users",
            )
        )


if __name__ == "__main__":
    unittest.main()
