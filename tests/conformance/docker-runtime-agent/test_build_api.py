"""Focused checks for the immutable Build context and public upload contract."""
import importlib.util
import io
import json
from pathlib import Path
import tarfile
import unittest
from unittest.mock import Mock, patch


SPEC = importlib.util.spec_from_file_location("graft_fixture_build_api", Path(__file__).with_name("build_api.py"))
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class BuildAPIFixtureTests(unittest.TestCase):
    def test_context_contains_a_real_scratch_build_without_external_base_dependencies(self):
        with tarfile.open(fileobj=io.BytesIO(MODULE.context_archive()), mode="r:gz") as archive:
            self.assertEqual(set(archive.getnames()), {"Dockerfile", "probe"})
            self.assertEqual(archive.extractfile("Dockerfile").read(), b"FROM scratch\nCOPY probe /probe\n")
            self.assertIn(b"graft-build-publication-verified", archive.extractfile("probe").read())

    def test_snapshot_upload_uses_the_declared_archive_field_and_authenticated_entrypoint(self):
        nested = Mock()
        nested.environment = {"GRAFT_CONFORMANCE_BACKEND_PORT": "28081"}
        nested.token = "synthetic-session"
        response = Mock(status=201)
        response.read.return_value = json.dumps({"data": {"snapshot_id": "snapshot-fixture"}}).encode()
        response.__enter__ = Mock(return_value=response)
        response.__exit__ = Mock(return_value=False)
        with patch.object(MODULE.request, "urlopen", return_value=response) as transport:
            self.assertEqual(MODULE.upload_snapshot(nested), "snapshot-fixture")
        call = transport.call_args.args[0]
        self.assertTrue(call.full_url.endswith("/api/build/input-snapshots"))
        self.assertEqual(call.get_header("Authorization"), "Bearer synthetic-session")
        self.assertIn(b'name="archive"', call.data)
        self.assertNotIn(b"synthetic-session", call.data)


if __name__ == "__main__":
    unittest.main()
