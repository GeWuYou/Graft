from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path


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


if __name__ == "__main__":
    unittest.main()
