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

    def test_web_module_permission_contract_is_canonical_definition_context(self) -> None:
        """Module-owned permission contracts may define canonical permission literals."""
        self.assertTrue(
            MODULE.is_definition_context(
                "web/src/modules/user/contract/permissions.ts",
                "  READ: 'user.read',",
                "user.read",
            )
        )
        self.assertFalse(
            MODULE.is_definition_context(
                "web/src/modules/rbac/contract/permission-copy.ts",
                "  [USER_PERMISSION_CODE.READ]: {",
                "user.read",
            )
        )

    def test_web_module_api_path_contract_is_canonical_definition_context(self) -> None:
        """Module-owned API path contracts may define canonical API path literals."""
        self.assertTrue(
            MODULE.is_definition_context(
                "web/src/modules/auth/contract/paths.ts",
                "  REFRESH: '/api/auth/refresh',",
                "/api/auth/refresh",
            )
        )
        self.assertFalse(
            MODULE.is_definition_context(
                "web/src/modules/auth/api/auth.ts",
                "    url: '/api/auth/refresh',",
                "/api/auth/refresh",
            )
        )


class TestFixtureExemptionTests(unittest.TestCase):
    """Keep test-only fixtures quiet without weakening runtime checks."""

    def test_scan_file_ignores_go_test_fixture_literals(self) -> None:
        text = "\n".join(
            [
                'engine.Use(RequirePermission(localizer, authService, authorizer, "user.read"))',
                'engine.GET("/api/users/:id", func(inner *gin.Context) {',
                'request := newBearerRequest("/api/users/1", "token-1")',
                'request.Header.Set("Accept-Language", "en-US")',
                'if payload.MessageKey != "auth.forbidden" || payload.Code != "AUTH_FORBIDDEN" {',
                'if err := bus.Subscribe("audit.record", func(_ context.Context, _ Event) error {',
            ]
        )

        findings = MODULE.scan_file("server/internal/httpx/authz_test.go", text)

        self.assertEqual(findings, [])

    def test_scan_file_ignores_typescript_test_fixture_literals(self) -> None:
        text = "\n".join(
            [
                "permissions: ['user.read'],",
                "code: 'USER_NOT_FOUND',",
                "messageKey: 'user.not_found',",
                "path: '/users',",
                "expect(messageMocks.error).not.toHaveBeenCalledWith('user.userList.statusUpdateFailed')",
            ]
        )

        findings = MODULE.scan_file("web/src/modules/user/pages/index.test.ts", text)

        self.assertEqual(findings, [])

    def test_scan_file_keeps_runtime_permission_findings(self) -> None:
        findings = MODULE.scan_file(
            "server/internal/httpx/authz.go",
            'engine.Use(RequirePermission(localizer, authService, authorizer, "user.read"))',
        )

        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0].rule, "permission-code-literal")
        self.assertEqual(findings[0].severity, "P0")


if __name__ == "__main__":
    unittest.main()
