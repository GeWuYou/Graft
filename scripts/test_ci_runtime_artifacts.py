from pathlib import Path
import json
from copy import deepcopy
import tempfile
import unittest
from unittest.mock import patch

from scripts.ci_runtime_artifacts import prepare, verify, verify_acceptance


class RuntimeArtifactTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.revision = patch("scripts.ci_runtime_artifacts.source_revision", return_value="a" * 40)
        self.revision.start()
        self.addCleanup(self.revision.stop)
        self.identity = patch("scripts.ci_runtime_artifacts.binary_identity", return_value={
            "version": "0.0.0-beta.1", "git_commit": "a" * 40, "git_tree_state": "clean"})
        self.identity_mock = self.identity.start()
        self.addCleanup(self.identity.stop)

    def write(self, path, content):
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content)
        return target

    def test_server_artifact_reuses_binary_and_verifies_revision_and_bytes(self):
        self.write(".tmp/compiled/server/graft-server-linux-amd64", "frozen binary bytes")
        artifact = self.root / "artifact"
        prepare(self.root, artifact, "server")
        self.assertEqual((artifact / "graft-server-linux-amd64").read_text(), "frozen binary bytes")
        verify(self.root, artifact, "server")
        (artifact / "graft-server-linux-amd64").write_text("tampered")
        with self.assertRaisesRegex(ValueError, "hashes"):
            verify(self.root, artifact, "server")

    def test_web_artifact_preserves_hidden_build_assets(self):
        self.write("web/dist/index.html", "<html>built</html>")
        self.write("web/dist/.vite/manifest.json", "{}")
        artifact = self.root / "artifact"
        prepare(self.root, artifact, "web")
        verify(self.root, artifact, "web")
        metadata = json.loads((artifact / "provenance.json").read_text())
        self.assertIn("dist/.vite/manifest.json", metadata["sha256"])

    def test_artifact_rejects_different_source_and_unrecorded_files(self):
        self.write(".tmp/compiled/server/graft-server-linux-amd64", "binary")
        artifact = self.root / "artifact"
        prepare(self.root, artifact, "server")
        with patch("scripts.ci_runtime_artifacts.source_revision", return_value="b" * 40):
            with self.assertRaisesRegex(ValueError, "source"):
                verify(self.root, artifact, "server")
        (artifact / "extra").write_text("unexpected")
        with self.assertRaisesRegex(ValueError, "hashes"):
            verify(self.root, artifact, "server")

    def test_missing_build_or_symlink_is_rejected(self):
        with self.assertRaisesRegex(ValueError, "missing"):
            prepare(self.root, self.root / "missing", "web")
        self.write("web/dist/index.html", "index")
        (self.root / "web/dist/external").symlink_to(self.root / "missing")
        with self.assertRaisesRegex(ValueError, "symlinks"):
            prepare(self.root, self.root / "linked", "web")

    def test_runtime_acceptance_rejects_missing_unverified_or_wrong_source(self):
        self.write("scripts/validate_docker_isolated.py",
                   "def require_local_acceptance(results):\n"
                   "    statuses = {item['scenario']: item['status'] for item in results}\n"
                   "    if statuses.get('required') != 'passed':\n"
                   "        raise RuntimeError('required scenario incomplete')\n")
        evidence = {"source_revision": "a" * 40, "execution_mode": "local",
                    "results": [{"scenario": "required", "status": "passed"}]}
        self.write("fixture/api-evidence.json", json.dumps(evidence))
        self.write("fixture/official/api-evidence.json", json.dumps(evidence))
        self.write("fixture/source-assets.json", json.dumps({"source_revision": "a" * 40}))
        self.write("fixture/official/installation-evidence.json", json.dumps({
            "source_revision": "a" * 40, "execution_mode": "local", "official_compose": True}))
        verify_acceptance(self.root, self.root / "fixture")
        for results in [[], [{"scenario": "required", "status": "unverified"}]]:
            self.write("fixture/api-evidence.json", json.dumps({**evidence, "results": results}))
            with self.assertRaisesRegex(RuntimeError, "incomplete"):
                verify_acceptance(self.root, self.root / "fixture")
        self.write("fixture/api-evidence.json", json.dumps({**evidence, "source_revision": "b" * 40}))
        with self.assertRaisesRegex(ValueError, "source"):
            verify_acceptance(self.root, self.root / "fixture")

    def test_target_backend_identity_is_version_specific(self):
        self.write(".tmp/compiled/target-server/graft-server-linux-amd64", "target")
        artifact = self.root / "target"
        with self.assertRaisesRegex(ValueError, "identity"):
            prepare(self.root, artifact, "target-server")
        self.identity_mock.return_value = {"version": "0.0.0-beta.2", "git_commit": "a" * 40, "git_tree_state": "clean"}
        prepare(self.root, self.root / "correct-target", "target-server")
        verify(self.root, self.root / "correct-target", "target-server")

    def test_runtime_helpers_match_existing_build_stage_paths(self):
        for kind, binary in [("runner", "graft-compose-runner"), ("release-source", "graft-release-fixture"), ("build-registry", "graft-fixture-htpasswd")]:
            self.write(f".tmp/compiled/{kind}/{binary}", "compiled")
            artifact = self.root / ("artifact-" + kind)
            prepare(self.root, artifact, kind)
            verify(self.root, artifact, kind)
            self.assertTrue((artifact / "out" / binary).is_file())

    def test_update_acceptance_requires_distinct_binary_provenance_and_backup_receipt(self):
        from scripts.ci_runtime_artifacts import verify_update_acceptance
        evidence = {
            "source_revision": "a" * 40, "synthetic_release_source": True,
            "source_version": "0.0.0-beta.1", "target_version": "0.0.0-beta.2",
            "source_binary_sha256": "1" * 64, "target_binary_sha256": "2" * 64,
            "task_status": "success", "task_id": 7, "server_container_replaced": True,
            "operation": {"operation_id": "update-test", "status": "SUCCESS"},
            "failed_backup_operation": {"status": "FAILED", "failure_code": "backup_failed"},
            "receipt": {"execution_mode": "local", "succeeded": True, "operation_id": "update-test",
                        "protocol_version": 2, "migration_started": True,
                        "launch_stage_id": 9, "launch_attempt": 1,
                        "backup_completion": {"OperationID": "update-test", "TaskID": 7,
                                              "ConfigSnapshotBytes": 10, "DatabaseDumpBytes": 20,
                                              "ConfigSnapshotSHA256": "3" * 64, "DatabaseDumpSHA256": "4" * 64}},
        }
        self.write(".tmp/runtime-server/provenance.json", json.dumps({"source_revision": "a" * 40, "kind": "server", "sha256": {"graft-server-linux-amd64": "1" * 64}}))
        self.write(".tmp/runtime-target-server/provenance.json", json.dumps({"source_revision": "a" * 40, "kind": "target-server", "sha256": {"graft-server-linux-amd64": "2" * 64}}))
        path = self.write("fixture/release-api/update-api-evidence.json", json.dumps(evidence))
        with patch("scripts.ci_runtime_artifacts.verify_acceptance"), patch("scripts.ci_runtime_artifacts.verify_build_acceptance"):
            verify_update_acceptance(self.root, self.root / "fixture")
            for field, value in [("task_status", "needs_attention"), ("target_binary_sha256", "1" * 64),
                                 ("server_container_replaced", False), ("source_version", "dev")]:
                broken = {**evidence, field: value}
                path.write_text(json.dumps(broken))
                with self.assertRaisesRegex(ValueError, "incomplete"):
                    verify_update_acceptance(self.root, self.root / "fixture")
            broken = deepcopy(evidence)
            broken["receipt"]["backup_completion"]["TaskID"] = 999
            path.write_text(json.dumps(broken))
            with self.assertRaisesRegex(ValueError, "incomplete"):
                verify_update_acceptance(self.root, self.root / "fixture")

    def test_build_proof_binds_authentication_publication_engine_and_secret_checks(self):
        from scripts.ci_runtime_artifacts import verify_build_acceptance
        image = "sha256:" + "1" * 64
        digest = "sha256:" + "2" * 64
        evidence = {
            "source_revision": "a" * 40, "task_id": 7, "task_status": "success",
            "registry_anonymous_status": 401, "registry_authenticated_status": 200,
            "registry_verified": True, "credential_absent_from_task_and_logs": True,
            "artifact": {"artifact_id": "artifact-fixture", "digest": digest, "image_id": image, "tag": "probe"},
            "engine_image_id": image, "remote_manifest_digest": digest,
            "publications": [{"publication_id": "publication-fixture", "artifact_id": "artifact-fixture",
                              "digest": digest, "destination": {"kind": "oci_registry", "reference": "probe"}}],
        }
        path = self.write("fixture/build-api/build-api-evidence.json", json.dumps(evidence))
        verify_build_acceptance(self.root, self.root / "fixture")
        for field, value in [
            ("registry_anonymous_status", 200), ("registry_authenticated_status", 401),
            ("credential_absent_from_task_and_logs", False), ("publications", []),
            ("engine_image_id", "sha256:" + "3" * 64), ("remote_manifest_digest", "sha256:" + "3" * 64),
            ("source_revision", "b" * 40), ("task_status", "needs_attention"),
        ]:
            with self.subTest(field=field):
                path.write_text(json.dumps({**evidence, field: value}))
                with self.assertRaisesRegex(ValueError, "incomplete"):
                    verify_build_acceptance(self.root, self.root / "fixture")
        broken = deepcopy(evidence)
        broken["publications"][0]["artifact_id"] = "different-artifact"
        path.write_text(json.dumps(broken))
        with self.assertRaisesRegex(ValueError, "incomplete"):
            verify_build_acceptance(self.root, self.root / "fixture")

    def test_update_proof_requires_prior_build_acceptance(self):
        from scripts.ci_runtime_artifacts import verify_update_acceptance
        with patch("scripts.ci_runtime_artifacts.verify_acceptance"), patch(
            "scripts.ci_runtime_artifacts.verify_build_acceptance", side_effect=ValueError("Build proof incomplete")
        ) as build:
            with self.assertRaisesRegex(ValueError, "Build proof incomplete"):
                verify_update_acceptance(self.root, self.root / "fixture")
        build.assert_called_once_with(self.root, self.root / "fixture")
