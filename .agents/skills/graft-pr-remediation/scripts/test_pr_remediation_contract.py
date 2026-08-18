#!/usr/bin/env python3
"""Protect the high-risk graft-pr-remediation authorization and closeout contract."""

from pathlib import Path
import unittest


SKILL_PATH = Path(__file__).resolve().parents[1] / "SKILL.md"


class PRRemediationContractTests(unittest.TestCase):
    """Require the remediation workflow to keep its write and validation gates explicit."""

    def test_skill_contract_keeps_authorization_validation_and_write_guards(self) -> None:
        """The named skill contract must fail closed across every high-risk stage."""
        code_quote = chr(96)
        text = SKILL_PATH.read_text(encoding="utf-8")
        required_terms = (
            "equivalent explicit request naming every stage",
            "python3 scripts/run_skill_tests.py",
            ".agents/skills/graft-pr-remediation/scripts/test_pr_remediation_contract.py",
            "A failed or",
            f"unavailable command makes the run {code_quote}blocked{code_quote}",
            f"authoritative {code_quote}github-graphql{code_quote} inventory",
            "immediately before resolution",
            "every relevant fixing commit (or one commit range)",
            "--ledger-expected-revision <sha256-or-absent>",
            "baseline_revision",
            "exact validated body and exactly one target entry",
        )

        for term in required_terms:
            with self.subTest(term=term):
                self.assertIn(term, text)


if __name__ == "__main__":
    unittest.main()
