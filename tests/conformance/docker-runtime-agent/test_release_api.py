"""Bounded checks for synthetic release metadata and API acceptance decisions."""
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import Mock, patch


SPEC = importlib.util.spec_from_file_location("graft_fixture_release_api", Path(__file__).with_name("release_api.py"))
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class ReleaseAPIFixtureTests(unittest.TestCase):
    def test_successful_update_requires_actual_migration_invocation_and_records_same_source_scope(self):
        for value in (None, False, 0, 1, "true"):
            with self.subTest(value=value), self.assertRaisesRegex(RuntimeError, "migration invocation"):
                MODULE.migration_invocation_scope({"migration_started": value}, "a" * 40)
        proof = MODULE.migration_invocation_scope({"migration_started": True}, "a" * 40)
        self.assertTrue(proof["bootstrap_invoked"])
        self.assertEqual(proof["source_revision"], proof["target_revision"])
        self.assertIn("no new pending DDL", proof["scope"])

    def test_release_metadata_keeps_real_image_digests_and_production_urls(self):
        images = {role: {"image": "ghcr.io/gewuyou/graft-" + role, "digest": "sha256:" + "a" * 64,
            "reference": "ghcr.io/gewuyou/graft-" + role + "@sha256:" + "a" * 64}
            for role in ("server", "web", "runner")}
        hashes = {"server.tar.gz": "b" * 64, "web.tar.gz": "c" * 64}
        manifest, releases = MODULE.release_metadata(images, hashes)
        self.assertEqual(manifest["images"]["server"], images["server"])
        self.assertEqual(manifest["runners"]["compose"], images["runner"])
        self.assertEqual(manifest["artifacts"]["sha256"], hashes)
        self.assertEqual(releases[0]["tag_name"], "v0.0.0-beta.2")
        self.assertTrue(releases[0]["prerelease"])
        self.assertTrue(all(asset["browser_download_url"].startswith("https://github.com/GeWuYou/Graft/releases/download/") for asset in releases[0]["assets"]))

    def test_wait_operation_requires_matching_terminal_operation_and_task(self):
        nested = Mock()
        operation = {"operation_id": "update-test", "status": "SUCCESS", "task_id": 7}
        nested.api.side_effect = [(200, {"data": [operation]}), (200, {"data": {"id": 7, "status": "success"}})]
        history, task = MODULE.wait_operation(nested, "update-test", "SUCCESS")
        self.assertEqual(history, operation)
        self.assertEqual(task["status"], "success")
        nested.api.side_effect = [(200, {"data": [{**operation, "status": "FAILED"}]}),
                                  (200, {"data": {"id": 7, "status": "failed"}})]
        with self.assertRaisesRegex(RuntimeError, "unexpected terminal"):
            MODULE.wait_operation(nested, "update-test", "SUCCESS")

    def test_waiting_retries_only_reads_across_server_replacement(self):
        nested = Mock()
        nested.api.side_effect = [OSError("server restarting"),
            (200, {"data": [{"operation_id": "update-test", "status": "SUCCESS", "task_id": 7}]}),
            (200, {"data": {"id": 7, "status": "success"}})]
        with patch.object(MODULE.time, "sleep"):
            MODULE.wait_operation(nested, "update-test", "SUCCESS")
        self.assertTrue(all(call.args[0] == "GET" for call in nested.api.call_args_list))

    def test_update_submission_is_sent_once_and_rejection_is_reported(self):
        nested = Mock()
        nested.api.return_value = (409, {"code": "common.task_resource_busy"})
        with self.assertRaisesRegex(RuntimeError, "HTTP 409"):
            MODULE.submit_update(nested)
        self.assertEqual(nested.api.call_count, 1)


if __name__ == "__main__":
    unittest.main()
