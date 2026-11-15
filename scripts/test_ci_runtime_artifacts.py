from pathlib import Path
import json
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

    def write(self, path, content):
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content)
        return target

    def test_server_artifact_reuses_binary_and_verifies_revision_and_bytes(self):
        self.write("server/graft", "frozen binary bytes")
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
        self.write("server/graft", "binary")
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
