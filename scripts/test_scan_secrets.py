import hashlib
from copy import deepcopy
import json
import unittest

from scripts.scan_secrets import SYNTHETIC_FINDINGS, evaluate_scan, match_synthetic_finding

# Each known synthetic finding's raw credential is composed at runtime so this
# file never contains a scannable literal; the composition is pinned to the
# registered raw_sha256 in the tests below.
SYNTHETIC_TEST_RAWS = {
    "scripts/test_runtime_failure_diagnostics.py": (
        "postgres" + "://" + "db:" + "unknown-uri-password" + "@postgres:5432"
    ),
    "scripts/test_browser_acceptance.py": (
        "postgres://" + "graft" + ":" + "hunter2" + "@db:5432"
    ),
}


class SecretScanPolicyTests(unittest.TestCase):
    def fixture(self, entry):
        raw = SYNTHETIC_TEST_RAWS[entry["file"]]
        self.assertEqual(hashlib.sha256(raw.encode()).hexdigest(), entry["raw_sha256"])
        return {"DetectorName": entry["detector"], "Verified": False, "Raw": raw,
                "SourceMetadata": {"Data": {"Git": {key: entry[key] for key in ("file", "commit", "line")}}}}

    def test_every_registered_entry_is_exactly_recognized(self):
        self.assertGreaterEqual(len(SYNTHETIC_FINDINGS), 2)
        self.assertEqual(len({entry["raw_sha256"] for entry in SYNTHETIC_FINDINGS}), len(SYNTHETIC_FINDINGS))
        for entry in SYNTHETIC_FINDINGS:
            with self.subTest(entry=entry["file"]):
                self.assertEqual(evaluate_scan(json.dumps(self.fixture(entry)), 183), (0, 1))
                dns_result = {**self.fixture(entry), "VerificationError": "lookup postgres: no such host"}
                self.assertEqual(evaluate_scan(json.dumps(dns_result), 183), (0, 1))
                self.assertIsNotNone(match_synthetic_finding(self.fixture(entry)))

    def test_both_entries_are_accepted_together(self):
        findings = [self.fixture(entry) for entry in SYNTHETIC_FINDINGS]
        output = "\n".join(json.dumps(item) for item in findings)
        self.assertEqual(evaluate_scan(output, 183), (0, len(SYNTHETIC_FINDINGS)))

    def test_near_miss_variations_are_not_recognized(self):
        for entry in SYNTHETIC_FINDINGS:
            with self.subTest(entry=entry["file"]):
                # Same location fields but a different credential never matches.
                near_miss_raw = SYNTHETIC_TEST_RAWS[entry["file"]] + "x"
                finding = {"DetectorName": entry["detector"], "Verified": False, "Raw": near_miss_raw,
                           "SourceMetadata": {"Data": {"Git": {key: entry[key] for key in ("file", "commit", "line")}}}}
                self.assertEqual(evaluate_scan(json.dumps(finding), 183), (1, 0))
                self.assertIsNone(match_synthetic_finding(finding))
                # Same credential but shifted location fields never matches.
                for field, value in [("file", "server/secret.go"), ("commit", "b" * 40),
                                     ("line", entry["line"] + 1), ("line", str(entry["line"]))]:
                    finding = deepcopy(self.fixture(entry))
                    finding["SourceMetadata"]["Data"]["Git"][field] = value
                    self.assertEqual(evaluate_scan(json.dumps(finding), 183), (1, 0))
                    self.assertIsNone(match_synthetic_finding(finding))

    def test_only_exact_historical_synthetic_finding_is_recognized(self):
        entry = SYNTHETIC_FINDINGS[0]
        for field, value in [("Verified", True),
                             ("DetectorName", "OtherDetector"), ("Raw", "different database credential")]:
            with self.subTest(field=field):
                finding = {**self.fixture(entry), field: value}
                self.assertEqual(evaluate_scan(json.dumps(finding), 183), (1, 0))
                finding["VerificationError"] = "lookup failed"
                self.assertEqual(evaluate_scan(json.dumps(finding), 183), (1, 0))

    def test_real_or_unclassified_findings_keep_the_gate_failed(self):
        unrecognized = {"DetectorName": "Postgres", "Verified": False, "Raw": "another password"}
        all_known = "\n".join(json.dumps(self.fixture(entry)) for entry in SYNTHETIC_FINDINGS)
        output = "\n".join([all_known, json.dumps(unrecognized)])
        self.assertEqual(evaluate_scan(output, 183), (1, len(SYNTHETIC_FINDINGS)))
        self.assertEqual(evaluate_scan("{}", 183), (1, 0))
        self.assertEqual(evaluate_scan(json.dumps({"SourceMetadata": None}), 183), (1, 0))

    def test_empty_malformed_and_inconsistent_output_fail_closed(self):
        self.assertEqual(evaluate_scan("", 0), (0, 0))
        for output, code in [("", 183), (json.dumps(self.fixture(SYNTHETIC_FINDINGS[0])), 0), ("not JSON", 183),
                             ("[]", 183), ("", 1), ("", 137), ("", 125)]:
            with self.subTest(output=output, code=code):
                with self.assertRaises(ValueError):
                    evaluate_scan(output, code)


if __name__ == "__main__":
    unittest.main()
