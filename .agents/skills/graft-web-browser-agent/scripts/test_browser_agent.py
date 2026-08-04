from __future__ import annotations

import importlib.util
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import call, patch


SCRIPT_PATH = Path(__file__).with_name("browser_agent.py")
SPEC = importlib.util.spec_from_file_location("browser_agent", SCRIPT_PATH)
assert SPEC and SPEC.loader
browser_agent = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(browser_agent)


class ParseCredentialsTest(unittest.TestCase):
    def test_defaults_to_dev_profile(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            credentials_file = Path(directory) / "credentials.yaml"
            credentials_file.write_text(
                """dev:
  username: dev-user
  password: dev-password
test:
  username: test-user
  password: test-password
""",
                encoding="utf-8",
            )

            credentials, profile = browser_agent.parse_credentials(credentials_file)

        self.assertEqual(profile, "dev")
        self.assertEqual(credentials, {"username": "dev-user", "password": "dev-password"})

    def test_selects_requested_profile(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            credentials_file = Path(directory) / "credentials.yaml"
            credentials_file.write_text(
                """dev:
  username: dev-user
  password: dev-password
test:
  username: test-user
  password: test-password
""",
                encoding="utf-8",
            )

            credentials, profile = browser_agent.parse_credentials(credentials_file, "test")

        self.assertEqual(profile, "test")
        self.assertEqual(credentials, {"username": "test-user", "password": "test-password"})

    def test_requested_profile_overrides_top_level_credentials(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            credentials_file = Path(directory) / "credentials.yaml"
            credentials_file.write_text(
                """username: flat-user
password: flat-password
dev:
  username: dev-user
  password: dev-password
test:
  username: test-user
  password: test-password
""",
                encoding="utf-8",
            )

            credentials, profile = browser_agent.parse_credentials(credentials_file, "test")

        self.assertEqual(profile, "test")
        self.assertEqual(credentials, {"username": "test-user", "password": "test-password"})

    def test_rejects_unknown_profile_without_disclosing_values(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            credentials_file = Path(directory) / "credentials.yaml"
            credentials_file.write_text(
                """dev:
  username: dev-user
  password: dev-password
""",
                encoding="utf-8",
            )

            with self.assertRaisesRegex(ValueError, "Credential profile not found: missing"):
                browser_agent.parse_credentials(credentials_file, "missing")


class RuntimeIdentityTest(unittest.TestCase):
    def test_records_primary_checkout_branch_and_full_head(self) -> None:
        results = [
            subprocess.CompletedProcess([], 0, "/repo\n"),
            subprocess.CompletedProcess([], 0, "feature/test\n"),
            subprocess.CompletedProcess([], 0, "abcdef123456\n"),
        ]

        with patch.object(browser_agent.subprocess, "run", side_effect=results) as run:
            identity = browser_agent.checkout_identity()

        self.assertEqual(
            identity,
            {"repository_root": "/repo", "branch": "feature/test", "head": "abcdef123456"},
        )
        self.assertEqual(
            run.call_args_list,
            [
                call(["git", "-C", str(browser_agent.ROOT_DIR), "rev-parse", "--show-toplevel"], check=True, capture_output=True, text=True),
                call(["git", "-C", str(browser_agent.ROOT_DIR), "branch", "--show-current"], check=True, capture_output=True, text=True),
                call(["git", "-C", str(browser_agent.ROOT_DIR), "rev-parse", "HEAD"], check=True, capture_output=True, text=True),
            ],
        )

    def test_prefers_explicit_non_secret_runtime_label(self) -> None:
        checkout = {"repository_root": "/repo", "branch": "feature/test", "head": "abc123"}

        self.assertEqual(
            browser_agent.runtime_identity("primary-web feature/test abc123", checkout),
            {"value": "primary-web feature/test abc123", "source": "explicit"},
        )

    def test_requires_explicit_runtime_label(self) -> None:
        checkout = {"repository_root": "/repo", "branch": "feature/test", "head": "abc123"}

        with self.assertRaisesRegex(ValueError, "must be a non-secret label"):
            browser_agent.runtime_identity(None, checkout)

    def test_rejects_runtime_label_for_a_different_checkout(self) -> None:
        checkout = {"repository_root": "/repo", "branch": "feature/test", "head": "abc123"}

        with self.assertRaisesRegex(ValueError, "current branch and full HEAD"):
            browser_agent.runtime_identity("primary-web feature/other abc123", checkout)

    def test_rejects_url_like_runtime_label(self) -> None:
        checkout = {"repository_root": "/repo", "branch": "feature/test", "head": "abc123"}

        with self.assertRaisesRegex(ValueError, "non-secret label"):
            browser_agent.runtime_identity("unsafe:label", checkout)


class BrowserPreflightTest(unittest.TestCase):
    def test_rejects_non_primary_checkout(self) -> None:
        checkout = {"repository_root": "/repo/worktree", "branch": "feature/test", "head": "abc123"}

        with (
            patch.object(browser_agent, "checkout_identity", return_value=checkout),
            patch.object(browser_agent, "primary_checkout_root", return_value=Path("/repo/primary")),
        ):
            with self.assertRaisesRegex(ValueError, "developer-owned primary checkout"):
                browser_agent.browser_preflight("primary-web feature/test abc123")

    def test_invalid_preflight_does_not_import_playwright(self) -> None:
        with (
            patch.object(browser_agent, "browser_preflight", side_effect=ValueError("invalid runtime label")),
            patch.object(browser_agent, "load_sync_playwright") as load_playwright,
            patch.object(sys, "argv", ["browser_agent.py", "--url", "http://localhost:5173"]),
        ):
            self.assertEqual(browser_agent.main(), 2)

        load_playwright.assert_not_called()


if __name__ == "__main__":
    unittest.main()
