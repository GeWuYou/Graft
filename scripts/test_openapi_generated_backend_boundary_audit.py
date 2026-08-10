#!/usr/bin/env python3
"""Unit tests for backend DTO boundary audit severity and verdict policy."""

from __future__ import annotations

import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import openapi_generated_backend_boundary_audit as audit


class BackendBoundaryAuditVerdictTests(unittest.TestCase):
    def test_script_uses_its_own_repository_root_outside_repository(self) -> None:
        repo_root = Path(__file__).resolve().parent.parent
        with tempfile.TemporaryDirectory() as working_directory:
            completed = subprocess.run(
                [sys.executable, str(repo_root / "scripts" / "openapi_generated_backend_boundary_audit.py")],
                cwd=working_directory,
                check=False,
                capture_output=True,
                text=True,
            )

        self.assertEqual(completed.returncode, 0, completed.stdout + completed.stderr)

    def test_allowed_findings_only_pass(self) -> None:
        result = audit.build_result()
        result.generated_mapper_allowed.append("mapper allowed")
        result.httpx_runtime_allowed.append("httpx allowed")

        self.assertEqual(audit.verdict(result), "PASS_ALLOWED_BOUNDARY_MODELS")

    def test_warnings_only_pass_with_warnings(self) -> None:
        result = audit.build_result()
        result.generated_boundary_preserved = False
        result.warnings.append("missing marker")

        self.assertEqual(audit.verdict(result), "PASS_WITH_WARNINGS")

    def test_stale_manual_request_dto_fails(self) -> None:
        result = audit.build_result()
        result.stale_manual_api_request_dto.append("manual request dto")

        self.assertEqual(audit.verdict(result), "FAILED_VIOLATIONS")

    def test_stale_manual_response_dto_fails(self) -> None:
        result = audit.build_result()
        result.stale_manual_api_response_dto.append("manual response dto")

        self.assertEqual(audit.verdict(result), "FAILED_VIOLATIONS")

    def test_generated_runtime_takeover_fails(self) -> None:
        result = audit.build_result()
        result.generated_runtime_used = True
        result.violations.append("generated runtime takeover")

        self.assertEqual(audit.verdict(result), "FAILED_VIOLATIONS")


if __name__ == "__main__":
    unittest.main()
