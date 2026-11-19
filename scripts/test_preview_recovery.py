"""Regression checks for the read-only preview recovery dry run."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent))

import preview
import preview_recovery


def scan_args(**overrides):
    values = {"state_dir": None, "identity": None,
              "claim_ownership": set(), "claim_task_id": {}}
    values.update(overrides)
    return argparse.Namespace(**values)


class RecoveryFixture(unittest.TestCase):
    """Builds a preview state tree with one healthy, one worktree-lost
    registration, one orphan directory and one archived cleanup record."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name) / "graft-preview"
        self.state = patch.dict(os.environ, {"GRAFT_PREVIEW_STATE_DIR": str(self.root)})
        self.state.start()
        self.addCleanup(self.state.stop)
        self.checkout = preview.checkout_root()

        document = {"schema_version": 1, "instances": {
            "healthy-task-111111111111": {
                "identity": "healthy-task-111111111111", "task_id": "healthy-task",
                "state": "running", "web_port": 38010,
                "compose_project": "graft-preview-healthy-task-111111111111",
                "checkout": str(self.checkout)},
            "lost-task-222222222222": {
                "identity": "lost-task-222222222222", "task_id": "lost-task",
                "state": "running", "web_port": 38011,
                "compose_project": "graft-preview-lost-task-222222222222",
                "checkout": "/nonexistent/worktree"},
        }}
        preview.write_registry(self.root, document)
        for identity in document["instances"]:
            directory = preview.instance_dir(self.root, identity)
            directory.mkdir(parents=True, exist_ok=True)
            (directory / "compose.yml").write_text("services: {}\n", encoding="utf-8")
            (directory / ".env").write_text("GRAFT_IMAGE_TAG=x\n", encoding="utf-8")
        (self.root / "instances" / "orphan-333333333333").mkdir(parents=True)
        self.entry = document["instances"]["lost-task-222222222222"]
        preview.append_archive(self.root, self.entry)

    def snapshot(self):
        return preview_recovery.tree_snapshot(self.root)


class ReadOnlyBehaviorTests(RecoveryFixture):
    def test_scan_never_mutates_the_state_tree(self):
        before = self.snapshot()
        report = preview_recovery.scan(scan_args())
        self.assertEqual(self.snapshot(), before)
        self.assertEqual(report["mode"], "dry-run-read-only")

    def test_main_reports_anomalies_and_exits_nonzero(self):
        import io
        from contextlib import redirect_stdout
        before = self.snapshot()
        buffer = io.StringIO()
        with redirect_stdout(buffer):
            with self.assertRaises(SystemExit) as context:
                preview_recovery.main([])
        self.assertEqual(context.exception.code, 1)
        report = json.loads(buffer.getvalue())
        self.assertEqual(report["verdict"], "attention")
        self.assertEqual(self.snapshot(), before)

    def test_clean_state_scans_with_exit_zero(self):
        document = preview.read_registry(self.root)
        document["instances"] = {"healthy-task-111111111111":
                                 document["instances"]["healthy-task-111111111111"]}
        preview.write_registry(self.root, document)
        (self.root / "instances" / "orphan-333333333333").rmdir()
        import shutil
        shutil.rmtree(self.root / "instances" / "lost-task-222222222222")
        import io
        from contextlib import redirect_stdout
        buffer = io.StringIO()
        with redirect_stdout(buffer):
            preview_recovery.main([])
        report = json.loads(buffer.getvalue())
        self.assertEqual(report["verdict"], "clean")
        self.assertEqual(report["anomalies"], [])


class OwnershipProofTests(RecoveryFixture):
    def test_worktree_lost_instance_has_no_suggestion_without_claims(self):
        report = preview_recovery.scan(scan_args(identity="lost-task-222222222222"))
        instance = report["instances"][0]
        self.assertIn("worktree_lost", instance["findings"])
        self.assertFalse(instance["ownership"]["proven"])
        self.assertIsNone(instance["destroy_suggestion"])
        self.assertIn("ownership unproven", instance["ownership"]["reason"])

    def test_task_claim_on_repository_bound_registry_emits_exact_command(self):
        args = scan_args(identity="lost-task-222222222222")
        args.claim_ownership = {"lost-task-222222222222"}
        args.claim_task_id = {"lost-task-222222222222": "lost-task"}
        with patch.object(preview_recovery, "resolve_state_dir",
                          return_value=(self.root, True)):
            report = preview_recovery.scan(args)
        instance = report["instances"][0]
        self.assertTrue(instance["ownership"]["proven"])
        self.assertIsNotNone(instance["destroy_suggestion"])
        command = instance["destroy_suggestion"]["command"]
        self.assertIn("--confirm-instance lost-task-222222222222", command)
        self.assertIn("destroy --task \"lost-task\"", command)

    def test_wrong_task_or_identity_claim_stays_unproven(self):
        args = scan_args(identity="lost-task-222222222222")
        args.claim_ownership = {"lost-task-222222222222"}
        args.claim_task_id = {"lost-task-222222222222": "other-task"}
        with patch.object(preview_recovery, "resolve_state_dir",
                          return_value=(self.root, True)):
            report = preview_recovery.scan(args)
        self.assertFalse(report["instances"][0]["ownership"]["proven"])

        args = scan_args(identity="lost-task-222222222222")
        args.claim_ownership = {"different-identity"}
        args.claim_task_id = {"different-identity": "lost-task"}
        with patch.object(preview_recovery, "resolve_state_dir",
                          return_value=(self.root, True)):
            report = preview_recovery.scan(args)
        self.assertFalse(report["instances"][0]["ownership"]["proven"])
        self.assertIsNone(report["instances"][0]["destroy_suggestion"])

    def test_overridden_state_dir_fails_task_proof_closed(self):
        args = scan_args(identity="lost-task-222222222222",
                         state_dir=str(self.root))
        args.claim_ownership = {"lost-task-222222222222"}
        args.claim_task_id = {"lost-task-222222222222": "lost-task"}
        report = preview_recovery.scan(args)
        instance = report["instances"][0]
        self.assertFalse(report["repository_bound_registry"])
        self.assertFalse(instance["ownership"]["proven"])
        self.assertIsNone(instance["destroy_suggestion"])

    def test_registered_checkout_path_proves_ownership(self):
        document = preview.read_registry(self.root)
        entry = document["instances"]["healthy-task-111111111111"]
        entry["state"] = "stopped"
        (preview.instance_dir(self.root, entry["identity"]) / ".env").unlink()
        preview.write_registry(self.root, document)
        report = preview_recovery.scan(scan_args(identity="healthy-task-111111111111"))
        instance = report["instances"][0]
        self.assertTrue(instance["ownership"]["proven"])
        self.assertIn("instance_files_missing", instance["findings"])
        self.assertIsNotNone(instance["destroy_suggestion"])
        self.assertIn("--confirm-instance healthy-task-111111111111",
                      instance["destroy_suggestion"]["command"])

    def test_healthy_running_instance_never_gets_a_destroy_suggestion(self):
        report = preview_recovery.scan(scan_args(identity="healthy-task-111111111111"))
        instance = report["instances"][0]
        self.assertEqual(instance["findings"], [])
        self.assertIsNone(instance["destroy_suggestion"])


class AnomalyTests(RecoveryFixture):
    def test_orphan_directory_is_reported(self):
        report = preview_recovery.scan(scan_args())
        self.assertIn(f"{self.root}/instances/orphan-333333333333: "
                      "unregistered instance directory (no registry entry)",
                      report["anomalies"])

    def test_unknown_identity_is_rejected(self):
        with self.assertRaises(preview.PreviewError):
            preview_recovery.scan(scan_args(identity="missing-identity"))


class ArchiveEvidenceTests(RecoveryFixture):
    def archive_files(self):
        return [path for path in (self.root / "archive").glob("*.json")
                if path.name != "manifest.json"]

    def test_intact_archive_evidence_verifies(self):
        report = preview_recovery.scan(scan_args())
        self.assertEqual(len(report["archive_evidence"]), 1)
        finding = report["archive_evidence"][0]
        self.assertEqual(finding["status"], "ok")
        self.assertTrue(finding["sha256_ok"])
        self.assertTrue(finding["evidence_ok"])

    def test_tampered_archive_bytes_fail_the_manifest_digest(self):
        target = self.archive_files()[0]
        original = json.loads(target.read_text(encoding="utf-8"))
        original["destroyed_at"] = "2000-01-01T00:00:00Z"
        target.write_text(json.dumps(original, indent=2) + "\n", encoding="utf-8")
        report = preview_recovery.scan(scan_args())
        self.assertEqual(report["archive_evidence"][0]["status"], "tampered")
        self.assertFalse(report["archive_evidence"][0]["sha256_ok"])
        self.assertIn("archive/" + target.name + ": tampered", report["anomalies"])

    def test_tampered_registry_entry_fails_the_inner_evidence_digest(self):
        target = self.archive_files()[0]
        record = json.loads(target.read_text(encoding="utf-8"))
        record["registry_entry"]["task_id"] = "rewritten-history"
        target.write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
        report = preview_recovery.scan(scan_args())
        self.assertEqual(report["archive_evidence"][0]["status"], "tampered")
        self.assertFalse(report["archive_evidence"][0]["evidence_ok"])

    def test_unlisted_archive_file_is_flagged(self):
        manifest_path = self.root / "archive" / "manifest.json"
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        removed_name = next(iter(manifest["entries"]))
        manifest["entries"].pop(removed_name)
        manifest_path.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
        report = preview_recovery.scan(scan_args())
        finding = next(item for item in report["archive_evidence"]
                       if item["file"] == removed_name)
        self.assertEqual(finding["status"], "unlisted")
        self.assertIn(f"archive/{removed_name}: unlisted", report["anomalies"])

    def test_manifest_entry_without_a_file_is_reported_missing(self):
        manifest_path = self.root / "archive" / "manifest.json"
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        manifest["entries"]["ghost-identity-999999999999.json"] = {
            "identity": "ghost-identity-999999999999", "destroyed_at": "2026-01-01T00:00:00Z",
            "sha256": hashlib.sha256(b"missing").hexdigest()}
        manifest_path.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
        report = preview_recovery.scan(scan_args())
        finding = next(item for item in report["archive_evidence"]
                       if item["file"] == "ghost-identity-999999999999.json")
        self.assertEqual(finding["status"], "missing")
        self.assertFalse(finding["sha256_ok"])
        self.assertIn("archive/ghost-identity-999999999999.json: missing", report["anomalies"])


if __name__ == "__main__":
    unittest.main()
