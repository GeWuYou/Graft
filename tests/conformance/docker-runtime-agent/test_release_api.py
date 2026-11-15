"""Bounded checks for synthetic release metadata and API acceptance decisions."""
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import Mock, patch


SPEC = importlib.util.spec_from_file_location("graft_fixture_release_api", Path(__file__).with_name("release_api.py"))
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class ReleaseAPIFixtureTests(unittest.TestCase):
    def task_response(self, status="success"):
        return {"id": 7, "type": "platform.update.compose", "owner_type": "platform_update",
                "owner_id": "update-test", "status": status}

    def operation_responses(self, operation, task):
        return [(200, {"data": [operation]}), (200, {"data": {"items": [task], "total": 1}}),
                (200, {"data": task})]

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
        operation = {"operation_id": "update-test", "phase": "SUCCESS", "state_source": "terminal_history"}
        nested.api.side_effect = self.operation_responses(operation, self.task_response())
        history, task = MODULE.wait_operation(nested, "update-test", "SUCCESS")
        self.assertEqual(history, operation)
        self.assertEqual(task["status"], "success")
        nested.api.side_effect = self.operation_responses({**operation, "phase": "FAILED"}, self.task_response("failed"))
        with self.assertRaisesRegex(RuntimeError, "unexpected terminal"):
            MODULE.wait_operation(nested, "update-test", "SUCCESS")

    def test_waiting_retries_only_reads_across_server_replacement(self):
        nested = Mock()
        nested.api.side_effect = [OSError("server restarting"),
            *self.operation_responses({"operation_id": "update-test", "phase": "SUCCESS"}, self.task_response())]
        with patch.object(MODULE.time, "sleep"):
            MODULE.wait_operation(nested, "update-test", "SUCCESS")
        self.assertTrue(all(call.args[0] == "GET" for call in nested.api.call_args_list))

    def test_missing_history_reads_the_current_operation_projection_after_task_failure(self):
        nested = Mock()
        task = self.task_response("failed")
        operation = {"operation_id": "update-test", "phase": "FAILED", "state_source": "task_recovery",
                     "error": "invalid_execution_intent"}
        nested.api.side_effect = [(200, {"data": []}), (200, {"data": {"items": [task], "total": 1}}),
                                  (200, {"data": task}), (200, {"data": operation})]
        history, result = MODULE.wait_operation(nested, "update-test", "FAILED")
        self.assertEqual(history, operation)
        self.assertEqual(result["id"], 7)
        self.assertTrue(nested.api.call_args_list[1].args[1].endswith("owner_type=platform_update&owner_id=update-test"))

    def test_task_owner_query_rejects_ambiguous_or_mismatched_details(self):
        nested = Mock()
        task = self.task_response()
        nested.api.return_value = (200, {"data": {"items": [task, task]}})
        with self.assertRaisesRegex(RuntimeError, "ambiguous Task owner"):
            MODULE.owned_update_task(nested, "update-test")
        nested.api.side_effect = [(200, {"data": {"items": [task]}}),
                                  (200, {"data": {**task, "owner_id": "different-operation"}})]
        with self.assertRaisesRegex(RuntimeError, "public owner binding"):
            MODULE.owned_update_task(nested, "update-test")

    def test_backup_detail_is_bound_to_task_and_receipt_integrity(self):
        receipt = {"ConfigSnapshotSHA256": "a" * 64, "ConfigSnapshotBytes": 12,
                   "DatabaseDumpSHA256": "b" * 64, "DatabaseDumpBytes": 34}
        backup = {"id": 8, "task_id": 7, "status": "AVAILABLE",
                  "config_snapshot": {"sha256": "a" * 64, "size_bytes": 12},
                  "database_dump": {"sha256": "b" * 64, "size_bytes": 34}}
        for size in (34, 35):
            nested = Mock()
            detail = {**backup, "database_dump": {"sha256": "b" * 64, "size_bytes": size}}
            nested.api.side_effect = [(200, {"data": {"items": [{"id": 8}]}}), (200, {"data": detail})]
            if size == 34:
                self.assertEqual(MODULE.backup_for_task(nested, 7, receipt), backup)
            else:
                with self.assertRaisesRegex(RuntimeError, "integrity metadata"):
                    MODULE.backup_for_task(nested, 7, receipt)

    def test_update_submission_is_sent_once_and_rejection_is_reported(self):
        nested = Mock()
        nested.api.return_value = (409, {"code": "common.task_resource_busy"})
        with self.assertRaisesRegex(RuntimeError, "HTTP 409"):
            MODULE.submit_update(nested)
        self.assertEqual(nested.api.call_count, 1)


if __name__ == "__main__":
    unittest.main()
