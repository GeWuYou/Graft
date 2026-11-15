from copy import deepcopy
import json
import unittest

from scripts.scan_secrets import SYNTHETIC_FINDING, evaluate_scan


class SecretScanPolicyTests(unittest.TestCase):
    def fixture(self):
        raw = "postgres" + "://" + "db:" + "unknown-uri-password" + "@postgres:5432"
        return {"DetectorName": "Postgres", "Verified": False, "Raw": raw,
                "SourceMetadata": {"Data": {"Git": {key: SYNTHETIC_FINDING[key] for key in ("file", "commit", "line")}}}}

    def test_only_exact_historical_synthetic_finding_is_recognized(self):
        self.assertEqual(evaluate_scan(json.dumps(self.fixture()), 183), (0, 1))
        dns_result = {**self.fixture(), "VerificationError": "lookup postgres: no such host"}
        self.assertEqual(evaluate_scan(json.dumps(dns_result), 183), (0, 1))
        for field, value in [("Verified", True),
                             ("DetectorName", "OtherDetector"), ("Raw", "different database credential")]:
            with self.subTest(field=field):
                finding = {**self.fixture(), field: value}
                self.assertEqual(evaluate_scan(json.dumps(finding), 183), (1, 0))
                finding["VerificationError"] = "lookup failed"
                self.assertEqual(evaluate_scan(json.dumps(finding), 183), (1, 0))
        for field, value in [("file", "server/secret.go"), ("commit", "b" * 40), ("line", 46), ("line", "45")]:
            with self.subTest(field=field):
                finding = deepcopy(self.fixture())
                finding["SourceMetadata"]["Data"]["Git"][field] = value
                self.assertEqual(evaluate_scan(json.dumps(finding), 183), (1, 0))
                finding["VerificationError"] = "lookup failed"
                self.assertEqual(evaluate_scan(json.dumps(finding), 183), (1, 0))

    def test_real_or_unclassified_findings_keep_the_gate_failed(self):
        unrecognized = {"DetectorName": "Postgres", "Verified": False, "Raw": "another password"}
        output = "\n".join(json.dumps(item) for item in [self.fixture(), unrecognized])
        self.assertEqual(evaluate_scan(output, 183), (1, 1))
        self.assertEqual(evaluate_scan("{}", 183), (1, 0))
        self.assertEqual(evaluate_scan(json.dumps({"SourceMetadata": None}), 183), (1, 0))

    def test_empty_malformed_and_inconsistent_output_fail_closed(self):
        self.assertEqual(evaluate_scan("", 0), (0, 0))
        for output, code in [("", 183), (json.dumps(self.fixture()), 0), ("not JSON", 183), ("[]", 183),
                             ("", 1), ("", 137), ("", 125)]:
            with self.subTest(output=output, code=code):
                with self.assertRaises(ValueError):
                    evaluate_scan(output, code)
