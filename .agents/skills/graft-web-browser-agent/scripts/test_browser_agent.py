from __future__ import annotations

import importlib.util
import json
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


class TargetConfigTest(unittest.TestCase):
    def config(self) -> dict[str, object]:
        return {
            "schema_version": 1,
            "defaults": {"environment": "local", "instance": "primary", "service": "web"},
            "environments": {
                "local": {
                    "instances": {
                        "primary": {
                            "services": {
                                "web": {
                                    "base_url": "http://127.0.0.1:3002",
                                    "credentials": {"username": "dev-user", "password": "dev-password"},
                                },
                                "api": {"base_url": "http://127.0.0.1:8080"},
                            }
                        }
                    }
                }
            },
        }

    def test_init_creates_template_only_when_missing(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "private" / "targets.yaml"
            browser_agent.initialize_target_config(path)

            document = browser_agent.load_target_config(path)

        self.assertEqual(document["schema_version"], 1)
        self.assertEqual(document["defaults"]["service"], "web")

    def test_init_refuses_overwrite(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "targets.yaml"
            path.write_text("user-owned\n", encoding="utf-8")

            with self.assertRaisesRegex(FileExistsError, "refusing to overwrite"):
                browser_agent.initialize_target_config(path)

            self.assertEqual(path.read_text(encoding="utf-8"), "user-owned\n")

    def test_first_main_invocation_creates_config_and_stops(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "private" / "targets.yaml"
            with (
                patch.object(browser_agent, "TARGET_CONFIG_FILE", path),
                patch.object(browser_agent, "browser_preflight") as browser_preflight,
                patch.object(sys, "argv", ["browser_agent.py"]),
            ):
                self.assertEqual(browser_agent.main(), 2)

            self.assertTrue(path.is_file())
            browser_preflight.assert_not_called()

    def test_explicit_init_refuses_existing_config(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "targets.yaml"
            path.write_text("user-owned\n", encoding="utf-8")
            with (
                patch.object(browser_agent, "TARGET_CONFIG_FILE", path),
                patch.object(sys, "argv", ["browser_agent.py", "--init-config"]),
            ):
                self.assertEqual(browser_agent.main(), 2)

            self.assertEqual(path.read_text(encoding="utf-8"), "user-owned\n")

    def test_resolves_defaults_and_one_run_override(self) -> None:
        target = browser_agent.resolve_target(
            self.config(), None, None, None, "http://localhost:4173", True
        )

        self.assertEqual(target["environment"], "local")
        self.assertEqual(target["instance"], "primary")
        self.assertEqual(target["service"], "web")
        self.assertEqual(target["url"], "http://localhost:4173")
        self.assertEqual(target["url_source"], "override")
        self.assertEqual(target["credentials"], {"username": "dev-user", "password": "dev-password"})

    def test_selects_explicit_service(self) -> None:
        target = browser_agent.resolve_target(self.config(), "local", "primary", "api", None, False)

        self.assertEqual(target["url"], "http://127.0.0.1:8080")
        self.assertIsNone(target["credentials"])

    def test_rejects_missing_target(self) -> None:
        with self.assertRaisesRegex(ValueError, "Unknown service: missing"):
            browser_agent.resolve_target(self.config(), None, None, "missing", None, False)

    def test_public_metadata_redacts_credentials(self) -> None:
        config = self.config()
        config["environments"]["local"]["instances"]["primary"]["services"]["web"]["base_url"] = (
            "http://private-host.internal:3002/audit?private=value"
        )
        target = browser_agent.resolve_target(config, None, None, None, None, True)

        public_summary = {
            "target": browser_agent.public_target_metadata(target),
            "navigation": browser_agent.public_navigation_metadata(target["url"], target["url"] + "#done"),
            "title": "Welcome dev-user at http://private-host.internal:3002/audit?private=value",
        }
        redacted_summary = browser_agent.redact_sensitive_values(
            public_summary,
            [target["url"], target["credentials"]["username"], target["credentials"]["password"]],
        )
        encoded = json.dumps(redacted_summary)

        self.assertNotIn("dev-user", encoded)
        self.assertNotIn("dev-password", encoded)
        self.assertNotIn("credentials", encoded)
        self.assertNotIn("private-host.internal", encoded)
        self.assertNotIn("private=value", encoded)
        self.assertNotIn(target["url"], encoded)
        self.assertEqual(redacted_summary["navigation"]["requested_path"], "/audit")


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
        with tempfile.TemporaryDirectory() as directory:
            target_config = Path(directory) / "targets.yaml"
            target_config.write_text(browser_agent.TARGET_CONFIG_TEMPLATE, encoding="utf-8")
            with (
                patch.object(browser_agent, "TARGET_CONFIG_FILE", target_config),
                patch.object(browser_agent, "browser_preflight", side_effect=ValueError("invalid runtime label")),
                patch.object(browser_agent, "load_sync_playwright") as load_playwright,
                patch.object(sys, "argv", ["browser_agent.py"]),
            ):
                self.assertEqual(browser_agent.main(), 2)

        load_playwright.assert_not_called()


if __name__ == "__main__":
    unittest.main()
